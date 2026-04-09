package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestApplyStateChangesPersistsRelationshipsHooksAndReactions(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:        "story-social",
		Name:      "Social Story",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID

	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-lyanna",
		StoryID:           story.ID,
		Name:              "Lyanna",
		Role:              "scout",
		PersonalityJSON:   `{}`,
		RelationshipJSON:  `{}`,
		Disposition:       5,
		IsAlive:           true,
		FirstAppearedTurn: 1,
		LastSeenTurn:      1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	changes := map[string]interface{}{
		"npc_relationship": map[string]interface{}{
			"name":    "Lyanna",
			"trust":   map[string]interface{}{"change": 12},
			"fear":    map[string]interface{}{"change": -4},
			"respect": map[string]interface{}{"value": 9},
		},
		"hook_add": map[string]interface{}{
			"kind":   "mystery",
			"title":  "Who sold you out?",
			"detail": "Someone warned the guard before you arrived.",
		},
		"world_reaction_add": map[string]interface{}{
			"kind":   "rumor",
			"title":  "The market is whispering",
			"detail": "Your name is spreading through the stalls.",
		},
		"fail_forward": map[string]interface{}{
			"title":  "The guard is now suspicious",
			"detail": "The setback adds heat instead of ending the thread.",
		},
	}

	applied, err := ApplyStateChanges(changes, char, world, db, story.ID, 2)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied state changes, got none")
	}

	npc, err := db.GetNPCByName(story.ID, "Lyanna")
	if err != nil || npc == nil {
		t.Fatalf("GetNPCByName: %v", err)
	}
	axes := loadRelationshipAxes(npc)
	if axes.Trust != 12 || axes.Fear != -4 || axes.Respect != 9 {
		t.Fatalf("relationship axes = %+v, want trust=12 fear=-4 respect=9", axes)
	}

	hooks := activeStoryHooks(loadStoryHooks(world))
	if len(hooks) != 1 || hooks[0].Title != "Who sold you out?" {
		t.Fatalf("hooks = %+v, want single mystery hook", hooks)
	}

	reactions := visibleWorldReactions(loadWorldReactions(world))
	if len(reactions) != 2 {
		t.Fatalf("reactions = %+v, want 2 active reactions", reactions)
	}

	delta := buildTurnDelta(applied)
	if delta == nil || len(delta.Items) == 0 {
		t.Fatal("expected turn delta items from applied changes")
	}
}

func TestApplyNarratorStateChangesPersistsCanonicalFronts(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:          "story-fronts",
		Name:        "Front Story",
		SettingJSON: `{}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	world := &storage.WorldState{
		ID:              "world-fronts",
		StoryID:         story.ID,
		CurrentLocation: "Bell Quarter",
		CurrentChapter:  1,
		CurrentTurn:     6,
		UpdatedAt:       now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	if err := ApplyNarratorStateChanges(context.Background(), map[string]interface{}{
		"front_add": []interface{}{
			map[string]interface{}{
				"id":            "front-bell-choir",
				"faction":       "Bell Choir",
				"title":         "The Silent Bell Choir is seeding sleepers across the district",
				"public_title":  "Whispers Around the Bell Tower",
				"stakes":        "If the choir succeeds, the district guard will answer to sleeper-priests.",
				"public_stakes": "The district keeps growing more tense around the bell tower.",
				"visibility":    "hidden",
				"segments":      6,
				"progress":      1,
			},
		},
	}, db, story, world, nil); err != nil {
		t.Fatalf("ApplyNarratorStateChanges front_add: %v", err)
	}
	if hooks := activeStoryHooks(loadStoryHooks(world)); len(hooks) != 0 {
		t.Fatalf("hidden front should not create visible hooks yet: %+v", hooks)
	}
	if reactions := visibleWorldReactions(loadWorldReactions(world)); len(reactions) != 0 {
		t.Fatalf("hidden front should not create visible reactions yet: %+v", reactions)
	}

	world.CurrentTurn = 7
	if err := ApplyNarratorStateChanges(context.Background(), map[string]interface{}{
		"front_reveal": []interface{}{
			map[string]interface{}{
				"id":            "front-bell-choir",
				"visibility":    "known",
				"public_title":  "Whispers Around the Bell Tower",
				"public_stakes": "The district keeps growing more tense around the bell tower.",
			},
		},
	}, db, story, world, nil); err != nil {
		t.Fatalf("ApplyNarratorStateChanges front_reveal: %v", err)
	}

	world.CurrentTurn = 8
	if err := ApplyNarratorStateChanges(context.Background(), map[string]interface{}{
		"front_pressure": []interface{}{
			map[string]interface{}{
				"id":     "front-bell-choir",
				"region": "Bell Quarter",
				"kind":   "suspicion",
				"value":  35,
				"detail": "Street sermons are drawing nervous crowds.",
			},
		},
	}, db, story, world, nil); err != nil {
		t.Fatalf("ApplyNarratorStateChanges front_pressure: %v", err)
	}

	world.CurrentTurn = 9
	if err := ApplyNarratorStateChanges(context.Background(), map[string]interface{}{
		"front_advance": []interface{}{
			map[string]interface{}{
				"id":     "front-bell-choir",
				"amount": 2,
			},
		},
	}, db, story, world, nil); err != nil {
		t.Fatalf("ApplyNarratorStateChanges front_advance: %v", err)
	}

	storedWorld, err := db.GetWorldState(story.ID)
	if err != nil {
		t.Fatalf("GetWorldState: %v", err)
	}

	fronts := loadFronts(storedWorld)
	if len(fronts) != 1 {
		t.Fatalf("fronts = %+v, want 1 front", fronts)
	}
	front := fronts[0]
	if front.Visibility != "known" {
		t.Fatalf("front visibility = %q, want known", front.Visibility)
	}
	if front.Progress != 3 || front.Segments != 6 {
		t.Fatalf("front progress = %d/%d, want 3/6", front.Progress, front.Segments)
	}
	if front.LastAdvancedTurn != 9 {
		t.Fatalf("front last advanced turn = %d, want 9", front.LastAdvancedTurn)
	}
	if len(front.Pressures) != 1 {
		t.Fatalf("front pressures = %+v, want 1 pressure", front.Pressures)
	}
	if front.Pressures[0].Region != "Bell Quarter" || front.Pressures[0].Level != 35 {
		t.Fatalf("front pressure = %+v, want Bell Quarter @ 35", front.Pressures[0])
	}
	hooks := activeStoryHooks(loadStoryHooks(storedWorld))
	if len(hooks) != 1 || hooks[0].Title != "Whispers Around the Bell Tower" {
		t.Fatalf("front hooks = %+v, want visible front hook", hooks)
	}
	reactions := visibleWorldReactions(loadWorldReactions(storedWorld))
	if len(reactions) != 1 || !strings.Contains(reactions[0].Title, "Bell Quarter grows watchful") {
		t.Fatalf("front reactions = %+v, want derived front pressure reaction", reactions)
	}
}

func TestFormatStoryTrackerViewShowsKnownFrontsOnly(t *testing.T) {
	world := &storage.WorldState{
		StoryHooksJSON:     `[]`,
		WorldReactionsJSON: `[]`,
		FrontsJSON: `[
			{
				"id":"front-known",
				"faction":"Bell Choir",
				"title":"The Silent Bell Choir is seeding sleepers across the district",
				"public_title":"Whispers Around the Bell Tower",
				"stakes":"Sleeper-priests will take the guard towers.",
				"public_stakes":"Something ugly is taking hold around the tower.",
				"visibility":"known",
				"segments":4,
				"progress":2,
				"pressures":[{"region":"Bell Quarter","kind":"suspicion","level":35,"detail":"Street sermons are drawing nervous crowds."}]
			},
			{
				"id":"front-hidden",
				"faction":"Ash Court",
				"title":"The Ash Court is buying judges in secret",
				"stakes":"They will own the city courts by moonrise.",
				"visibility":"hidden",
				"segments":4,
				"progress":1
			}
		]`,
	}

	view := FormatStoryTrackerView(world)
	if !strings.Contains(view, "## Open Fronts") {
		t.Fatalf("tracker view missing fronts section:\n%s", view)
	}
	if !strings.Contains(view, "Whispers Around the Bell Tower") {
		t.Fatalf("tracker view missing known front title:\n%s", view)
	}
	if !strings.Contains(view, "Bell Quarter [suspicion 35 rising]") {
		t.Fatalf("tracker view missing known front pressure:\n%s", view)
	}
	if strings.Contains(view, "Ash Court is buying judges in secret") {
		t.Fatalf("tracker view leaked hidden front title:\n%s", view)
	}
	if strings.Contains(view, "They will own the city courts by moonrise") {
		t.Fatalf("tracker view leaked hidden front stakes:\n%s", view)
	}
}

func TestApplyStateChangesFailForwardAdvancesReferencedFront(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	world.FrontsJSON = `[
		{
			"id":"front-known",
			"faction":"Bell Choir",
			"title":"The Silent Bell Choir is seeding sleepers across the district",
			"public_title":"Whispers Around the Bell Tower",
			"public_stakes":"Something ugly is taking hold around the tower.",
			"visibility":"known",
			"segments":4,
			"progress":1
		}
	]`

	applied, err := ApplyStateChanges(map[string]interface{}{
		"fail_forward": map[string]interface{}{
			"title":           "The checkpoint captain doubles inspections",
			"detail":          "You slip away, but the district is now on edge.",
			"front_id":        "front-known",
			"front_advance":   1,
			"pressure_region": "Bell Quarter",
			"pressure_kind":   "suspicion",
			"pressure_change": 20,
		},
	}, char, world, nil, "test-story-id", 4)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}

	foundFrontAdvance := false
	for _, change := range applied {
		if strings.Contains(change.Description, "Front advances: Whispers Around the Bell Tower") {
			foundFrontAdvance = true
			break
		}
	}
	if !foundFrontAdvance {
		t.Fatalf("applied changes = %+v, want front advancement entry", applied)
	}

	fronts := loadFronts(world)
	if len(fronts) != 1 || fronts[0].Progress != 2 {
		t.Fatalf("fronts = %+v, want progress advanced to 2", fronts)
	}
	if len(fronts[0].Pressures) != 1 || fronts[0].Pressures[0].Level != 20 {
		t.Fatalf("front pressure = %+v, want Bell Quarter suspicion 20", fronts[0].Pressures)
	}

	hooks := activeStoryHooks(loadStoryHooks(world))
	if len(hooks) != 1 || hooks[0].Title != "Whispers Around the Bell Tower" {
		t.Fatalf("hooks = %+v, want synced front hook", hooks)
	}

	reactions := visibleWorldReactions(loadWorldReactions(world))
	if len(reactions) != 2 {
		t.Fatalf("reactions = %+v, want fail-forward reaction plus derived front pressure reaction", reactions)
	}
}

func TestApplyStateChangesNemesisResolutionCaptureLeavesContainmentFallout(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:          "story-nemesis-capture",
		Name:        "Nemesis Capture Story",
		SettingJSON: `{"factions":["Bell Choir"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID
	world.CurrentLocation = "Bell Quarter"
	world.CurrentTurn = 9
	storeFronts(world, []Front{
		{
			ID:           "front-bell",
			Faction:      "Bell Choir",
			Title:        "The Silent Bell Choir is seeding sleepers across the district",
			PublicTitle:  "Whispers Around the Bell Tower",
			PublicStakes: "Something ugly is taking hold around the tower.",
			Visibility:   "known",
			Segments:     6,
			Progress:     4,
			Pressures: []FrontPressure{
				{Region: "Bell Quarter", Kind: "suspicion", Level: 40, UpdatedTurn: 9},
			},
		},
	})

	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-lyanna",
		StoryID:           story.ID,
		Name:              "Lyanna",
		Role:              "broker",
		PersonalityJSON:   `{}`,
		RelationshipJSON:  `{"fear":5,"respect":2}`,
		NemesisJSON:       `{"status":"active","rivalry_score":8,"escalation_tier":3,"threat_posture":"vengeful","last_outcome":"Escaped the tribunal."}`,
		Disposition:       -30,
		IsAlive:           true,
		FirstAppearedTurn: 2,
		LastSeenTurn:      8,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	applied, err := ApplyStateChanges(map[string]interface{}{
		"nemesis_resolution": map[string]interface{}{
			"name":     "Lyanna",
			"outcome":  "capture",
			"detail":   "The harbor watch drags Lyanna into an iron cell.",
			"front_id": "front-bell",
		},
	}, char, world, db, story.ID, 10)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) < 3 {
		t.Fatalf("applied changes = %+v, want nemesis + world fallout entries", applied)
	}

	npc, err := db.GetNPCByName(story.ID, "Lyanna")
	if err != nil || npc == nil {
		t.Fatalf("GetNPCByName: %v", err)
	}
	profile := loadNemesisProfile(npc)
	if profile == nil {
		t.Fatal("expected persisted nemesis profile after capture")
	}
	if profile.Status != NemesisStatusResolved {
		t.Fatalf("profile status = %q, want resolved", profile.Status)
	}
	if profile.ThreatPosture != "contained" {
		t.Fatalf("profile threat_posture = %q, want contained", profile.ThreatPosture)
	}
	if !npc.IsAlive {
		t.Fatal("captured nemesis should remain alive")
	}

	hooks := activeStoryHooks(loadStoryHooks(world))
	if len(hooks) != 1 || !strings.Contains(hooks[0].Title, "Holding Lyanna") {
		t.Fatalf("hooks = %+v, want holding hook", hooks)
	}
	reactions := visibleWorldReactions(loadWorldReactions(world))
	if len(reactions) != 1 || reactions[0].Title != "Lyanna is in chains" {
		t.Fatalf("reactions = %+v, want capture reaction", reactions)
	}
	fronts := loadFronts(world)
	if len(fronts) != 1 || fronts[0].Status != "stalled" {
		t.Fatalf("fronts = %+v, want stalled front", fronts)
	}
}

func TestApplyStateChangesNemesisResolutionSuccessionKeepsArcMoving(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:          "story-nemesis-succession",
		Name:        "Nemesis Succession Story",
		SettingJSON: `{"factions":["Bell Choir"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID
	world.CurrentLocation = "Bell Quarter"
	world.CurrentTurn = 11
	storeFronts(world, []Front{
		{
			ID:           "front-bell",
			Faction:      "Bell Choir",
			Title:        "The Silent Bell Choir is seeding sleepers across the district",
			PublicTitle:  "Whispers Around the Bell Tower",
			PublicStakes: "Something ugly is taking hold around the tower.",
			Visibility:   "known",
			Segments:     6,
			Progress:     3,
			Pressures: []FrontPressure{
				{Region: "Bell Quarter", Kind: "control", Level: 55, UpdatedTurn: 11},
			},
		},
	})

	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-lyanna-succession",
		StoryID:           story.ID,
		Name:              "Lyanna",
		Role:              "broker",
		PersonalityJSON:   `{}`,
		RelationshipJSON:  `{}`,
		NemesisJSON:       `{"status":"active","rivalry_score":9,"escalation_tier":4,"threat_posture":"political","last_outcome":"Bell Choir fixers keep moving around Lyanna."}`,
		Disposition:       -40,
		IsAlive:           true,
		FirstAppearedTurn: 2,
		LastSeenTurn:      10,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	applied, err := ApplyStateChanges(map[string]interface{}{
		"nemesis_resolution": map[string]interface{}{
			"name":      "Lyanna",
			"outcome":   "succession",
			"detail":    "Sable Jack picks up Lyanna's debts and grudges.",
			"successor": "Sable Jack",
			"front_id":  "front-bell",
		},
	}, char, world, db, story.ID, 12)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) < 3 {
		t.Fatalf("applied changes = %+v, want nemesis succession fallout", applied)
	}

	npc, err := db.GetNPCByName(story.ID, "Lyanna")
	if err != nil || npc == nil {
		t.Fatalf("GetNPCByName: %v", err)
	}
	profile := loadNemesisProfile(npc)
	if profile == nil || profile.Status != NemesisStatusResolved {
		t.Fatalf("profile = %+v, want resolved after succession", profile)
	}
	if profile.ThreatPosture != "succession" {
		t.Fatalf("threat_posture = %q, want succession", profile.ThreatPosture)
	}

	hooks := activeStoryHooks(loadStoryHooks(world))
	if len(hooks) != 1 || hooks[0].Title != "Sable Jack" {
		t.Fatalf("hooks = %+v, want successor hook", hooks)
	}
	reactions := visibleWorldReactions(loadWorldReactions(world))
	if len(reactions) != 1 || reactions[0].Title != "Sable Jack" {
		t.Fatalf("reactions = %+v, want successor reaction", reactions)
	}
	fronts := loadFronts(world)
	if len(fronts) != 1 {
		t.Fatalf("fronts = %+v, want 1 front", fronts)
	}
	if fronts[0].Status != "active" || fronts[0].Progress != 4 {
		t.Fatalf("front after succession = %+v, want active with progress advanced", fronts[0])
	}
}
