package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildStoryCodexShowsVisibleFrontsWithoutLeakingHiddenState(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:          "story-codex-fronts",
		Name:        "Codex Front Story",
		SettingJSON: `{"factions":["Bell Choir","Ash Court"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	world := &storage.WorldState{
		ID:              "world-codex-fronts",
		StoryID:         story.ID,
		CurrentLocation: "Bell Quarter",
		KnownLocationsJSON: `[
			{"name":"Bell Quarter","description":"A district of bells and narrow alleys.","discovered_turn":1},
			{"name":"Harbor Ward","region":"Bell Quarter","description":"Warehouses and watchfires.","discovered_turn":2}
		]`,
		CurrentChapter: 1,
		CurrentTurn:    5,
		UpdatedAt:      now,
	}
	storeFronts(world, []Front{
		{
			ID:           "front-rumor",
			Faction:      "Bell Choir",
			Title:        "The Silent Bell Choir is seeding sleepers across the district",
			PublicTitle:  "Whispers Around the Bell Tower",
			Stakes:       "Sleeper-priests will take the guard towers.",
			PublicStakes: "Something ugly is taking hold around the tower.",
			Visibility:   "rumored",
			Segments:     6,
			Progress:     3,
			Pressures: []FrontPressure{
				{Region: "Bell Quarter", Kind: "suspicion", Level: 35, UpdatedTurn: 5},
			},
		},
		{
			ID:         "front-hidden",
			Faction:    "Ash Court",
			Title:      "The Ash Court is buying judges in secret",
			Stakes:     "They will own the courts by moonrise.",
			Visibility: "hidden",
			Segments:   4,
			Progress:   1,
		},
	})
	syncKnownFrontContinuity(world, world.CurrentTurn)
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	index, err := BuildStoryCodex(db, story, char, world)
	if err != nil {
		t.Fatalf("BuildStoryCodex: %v", err)
	}

	frontIDs := index.CategoryEntries["fronts"]
	if len(frontIDs) != 1 {
		t.Fatalf("front category entries = %v, want only the visible front", frontIDs)
	}

	entry, ok := index.Entry(codexFrontEntryID("front-rumor"))
	if !ok {
		t.Fatalf("missing visible front entry in codex")
	}
	if entry.Title != "Whispers Around the Bell Tower" {
		t.Fatalf("front entry title = %q, want public title", entry.Title)
	}
	if strings.Contains(entry.Subtitle, "Bell Choir") {
		t.Fatalf("rumored front leaked faction in subtitle: %+v", entry)
	}

	joinedSections := strings.Join(flattenCodexSections(entry.Sections), "\n")
	if strings.Contains(joinedSections, "Bell Choir") {
		t.Fatalf("rumored front leaked faction in sections:\n%s", joinedSections)
	}
	if strings.Contains(joinedSections, "Sleeper-priests will take the guard towers.") {
		t.Fatalf("rumored front leaked hidden stakes:\n%s", joinedSections)
	}
	if !strings.Contains(joinedSections, "Something ugly is taking hold around the tower.") {
		t.Fatalf("rumored front should retain public stakes:\n%s", joinedSections)
	}

	locationEntry, ok := index.Entry(codexLocationEntryID("Bell Quarter"))
	if !ok {
		t.Fatalf("missing location entry for Bell Quarter")
	}
	locationSections := strings.Join(flattenCodexSections(locationEntry.Sections), "\n")
	if !strings.Contains(locationSections, "Whispers Around the Bell Tower - Bell Quarter [suspicion 35 rising]") {
		t.Fatalf("location entry missing front pressure trace:\n%s", locationSections)
	}

	protagonist, ok := index.Entry(ProtagonistCodexEntryID())
	if !ok {
		t.Fatalf("missing protagonist codex entry")
	}
	if !strings.Contains(strings.Join(flattenCodexSections(protagonist.Sections), "\n"), "Whispers Around the Bell Tower 3/6") {
		t.Fatalf("protagonist entry missing known front summary")
	}
}

func TestBuildStoryCodexSurfacesNemesisTrailWithoutLeakingPrivateVow(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:          "story-codex-nemesis",
		Name:        "Codex Nemesis Story",
		SettingJSON: `{"factions":["Bell Choir"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	npc := &storage.NPC{
		ID:                 "npc-lyanna",
		StoryID:            story.ID,
		Name:               "Lyanna",
		Role:               "broker",
		Appearance:         "An immaculate broker with a split lip and cold eyes.",
		RelationshipJSON:   `{"fear":18,"respect":12}`,
		NemesisJSON:        `{"status":"active","rivalry_score":8,"escalation_tier":3,"threat_posture":"political","vow":"Lyanna swears this rivalry is not finished in the senate cellars.","last_outcome":"Escaped through Bell Quarter after the tribunal turned ugly.","last_seen_turn":8,"visible_scars":["Split lip from the Bell Quarter tribunal"],"event_history":[{"kind":"humiliation","turn":5,"detail":"Publicly outmaneuvered at Bell Quarter."},{"kind":"political_fallout","turn":8,"detail":"Bell Choir fixers started whispering through the ward."}]}`,
		PrivateThoughts:    `[]`,
		NotesOnProtagonist: `[]`,
		IsAlive:            true,
		FirstAppearedTurn:  2,
		LastSeenTurn:       8,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := db.CreateNPC(npc); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	world := &storage.WorldState{
		ID:              "world-codex-nemesis",
		StoryID:         story.ID,
		CurrentLocation: "Bell Quarter",
		KnownLocationsJSON: `[
			{"name":"Bell Quarter","description":"A district of bells and narrow alleys.","discovered_turn":1},
			{"name":"Harbor Ward","region":"Bell Quarter","description":"Warehouses and watchfires.","discovered_turn":2}
		]`,
		CurrentChapter: 2,
		CurrentTurn:    9,
		UpdatedAt:      now,
	}
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
				{Region: "Bell Quarter", Kind: "suspicion", Level: 50, UpdatedTurn: 9},
			},
		},
	})
	storeWorldReactions(world, []WorldReaction{
		{
			ID:          "reaction-lyanna",
			Kind:        "rumor",
			Title:       "Lyanna's name is back on every tongue",
			Detail:      "Lyanna is leaning on Bell Quarter notaries again.",
			SourceTurn:  8,
			CreatedTurn: 9,
		},
	})
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	index, err := BuildStoryCodex(db, story, char, world)
	if err != nil {
		t.Fatalf("BuildStoryCodex: %v", err)
	}

	entry, ok := index.Entry(codexNPCEntryID("Lyanna"))
	if !ok {
		t.Fatalf("missing Lyanna codex entry")
	}
	joined := strings.Join(flattenCodexSections(entry.Sections), "\n")
	if !strings.Contains(joined, "Status: Active nemesis") {
		t.Fatalf("nemesis entry missing status:\n%s", joined)
	}
	if !strings.Contains(joined, "Suspected agenda: Pressure through allies, rumor, or institutions seems likely.") {
		t.Fatalf("nemesis entry missing player-safe agenda:\n%s", joined)
	}
	if !strings.Contains(joined, "Visible scars: Split lip from the Bell Quarter tribunal") {
		t.Fatalf("nemesis entry missing scars:\n%s", joined)
	}
	if !strings.Contains(joined, "Turn 8: Political Fallout — Bell Choir fixers started whispering through the ward.") {
		t.Fatalf("nemesis entry missing escalation trace:\n%s", joined)
	}
	if strings.Contains(joined, "swears this rivalry is not finished") {
		t.Fatalf("nemesis entry leaked private vow:\n%s", joined)
	}
	if !hasCodexLink(entry.Related, codexFrontEntryID("front-bell")) {
		t.Fatalf("nemesis entry missing front link: %+v", entry.Related)
	}
	if !hasCodexLink(entry.Related, codexLocationEntryID("Bell Quarter")) {
		t.Fatalf("nemesis entry missing location link: %+v", entry.Related)
	}
	if !hasCodexLink(entry.Related, codexThreadEntryID("reaction", "reaction-lyanna")) {
		t.Fatalf("nemesis entry missing reaction link: %+v", entry.Related)
	}

	frontEntry, ok := index.Entry(codexFrontEntryID("front-bell"))
	if !ok {
		t.Fatalf("missing front entry")
	}
	if !hasCodexLink(frontEntry.Related, codexNPCEntryID("Lyanna")) {
		t.Fatalf("front entry missing nemesis back-link: %+v", frontEntry.Related)
	}

	locationEntry, ok := index.Entry(codexLocationEntryID("Bell Quarter"))
	if !ok {
		t.Fatalf("missing location entry")
	}
	if !hasCodexLink(locationEntry.Related, codexNPCEntryID("Lyanna")) {
		t.Fatalf("location entry missing nemesis back-link: %+v", locationEntry.Related)
	}

	protagonist, ok := index.Entry(ProtagonistCodexEntryID())
	if !ok {
		t.Fatalf("missing protagonist entry")
	}
	if !strings.Contains(strings.Join(flattenCodexSections(protagonist.Sections), "\n"), "Lyanna — Active nemesis · Tier 3") {
		t.Fatalf("protagonist entry missing nemesis overview")
	}
}

func TestBuildStoryCodexSurfacesInvestigationCasesWithoutHiddenTruthLeaks(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:          "story-codex-investigation",
		Name:        "Codex Investigation Story",
		SettingJSON: `{"factions":["Bell Choir"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	npc := &storage.NPC{
		ID:                 "npc-lyanna-investigation",
		StoryID:            story.ID,
		Name:               "Lyanna",
		Role:               "broker",
		RelationshipJSON:   `{}`,
		NotesOnProtagonist: `[]`,
		PrivateThoughts:    `[]`,
		NemesisJSON:        `{}`,
		IsAlive:            true,
		FirstAppearedTurn:  2,
		LastSeenTurn:       8,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := db.CreateNPC(npc); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	world := &storage.WorldState{
		ID:              "world-codex-investigation",
		StoryID:         story.ID,
		CurrentLocation: "Bell Quarter",
		KnownLocationsJSON: `[
			{"name":"Bell Quarter","description":"A district of bells and narrow alleys.","discovered_turn":1}
		]`,
		CurrentChapter: 2,
		CurrentTurn:    9,
		UpdatedAt:      now,
	}
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
		},
	})
	storeInvestigationBoard(world, InvestigationBoard{
		Cases: []InvestigationCase{
			{
				ID:          "case-sold-out",
				Title:       "Who sold you out?",
				Summary:     "Someone warned the checkpoint ahead of time.",
				UpdatedTurn: 9,
				Links: []InvestigationLink{
					{Kind: "front", RefID: "front-bell", Label: "Whispers Around the Bell Tower"},
				},
				Suspects: []InvestigationSuspect{
					{Name: "Lyanna", Links: []InvestigationLink{{Kind: "npc", Label: "Lyanna"}}},
				},
				Clues: []InvestigationClue{
					{Label: "Ledger ash", Detail: "The burned ledger page still smelled of sealing wax."},
				},
				Contradictions: []InvestigationContradiction{
					{Label: "Two alibis overlap", Detail: "The quartermaster and captain both claim to have sealed the log."},
				},
				Theories: []InvestigationTheory{
					{Statement: "The guard captain was bribed", Confidence: "likely", Status: "likely"},
				},
				HiddenTruths: []InvestigationHiddenTruth{
					{Label: "Bell Choir silver changed hands", Detail: "This should stay hidden from the codex."},
				},
			},
		},
	})
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	index, err := BuildStoryCodex(db, story, char, world)
	if err != nil {
		t.Fatalf("BuildStoryCodex: %v", err)
	}

	caseEntry, ok := index.Entry(codexInvestigationEntryID("case-sold-out"))
	if !ok {
		t.Fatalf("missing investigation entry")
	}
	caseSections := strings.Join(flattenCodexSections(caseEntry.Sections), "\n")
	if !strings.Contains(caseSections, "Ledger ash") {
		t.Fatalf("investigation entry missing clue:\n%s", caseSections)
	}
	if !strings.Contains(caseSections, "Two alibis overlap") {
		t.Fatalf("investigation entry missing contradiction:\n%s", caseSections)
	}
	if strings.Contains(caseSections, "Bell Choir silver changed hands") {
		t.Fatalf("investigation entry leaked hidden truth:\n%s", caseSections)
	}
	if !hasCodexLink(caseEntry.Related, codexNPCEntryID("Lyanna")) {
		t.Fatalf("investigation entry missing npc link: %+v", caseEntry.Related)
	}
	if !hasCodexLink(caseEntry.Related, codexFrontEntryID("front-bell")) {
		t.Fatalf("investigation entry missing front link: %+v", caseEntry.Related)
	}

	npcEntry, ok := index.Entry(codexNPCEntryID("Lyanna"))
	if !ok {
		t.Fatalf("missing npc entry")
	}
	if !hasCodexLink(npcEntry.Related, codexInvestigationEntryID("case-sold-out")) {
		t.Fatalf("npc entry missing investigation backlink: %+v", npcEntry.Related)
	}

	frontEntry, ok := index.Entry(codexFrontEntryID("front-bell"))
	if !ok {
		t.Fatalf("missing front entry")
	}
	if !hasCodexLink(frontEntry.Related, codexInvestigationEntryID("case-sold-out")) {
		t.Fatalf("front entry missing investigation backlink: %+v", frontEntry.Related)
	}

	protagonist, ok := index.Entry(ProtagonistCodexEntryID())
	if !ok {
		t.Fatalf("missing protagonist entry")
	}
	if !strings.Contains(strings.Join(flattenCodexSections(protagonist.Sections), "\n"), "Who sold you out? · clues 1 · contradictions 1 · theories 1") {
		t.Fatalf("protagonist entry missing open investigation overview")
	}
}

func TestBuildStoryCodexSurfacesProjectsAndBacklinks(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:          "story-codex-projects",
		Name:        "Codex Projects Story",
		SettingJSON: `{"factions":["Bell Choir"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	npc := &storage.NPC{
		ID:                 "npc-lyanna-project-codex",
		StoryID:            story.ID,
		Name:               "Lyanna",
		Role:               "duelist",
		RelationshipJSON:   `{}`,
		PrivateThoughts:    `[]`,
		NotesOnProtagonist: `[]`,
		NemesisJSON:        `{}`,
		IsAlive:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := db.CreateNPC(npc); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	world := &storage.WorldState{
		ID:              "world-codex-projects",
		StoryID:         story.ID,
		CurrentLocation: "Bell Quarter",
		KnownLocationsJSON: `[
			{"name":"Bell Quarter","description":"A district of bells and narrow alleys.","discovered_turn":1}
		]`,
		CurrentChapter: 2,
		CurrentTurn:    9,
		UpdatedAt:      now,
	}
	storeFronts(world, []Front{
		{
			ID:           "front-bell-projects",
			Faction:      "Bell Choir",
			Title:        "The Silent Bell Choir is seeding sleepers across the district",
			PublicTitle:  "Whispers Around the Bell Tower",
			PublicStakes: "Something ugly is taking hold around the tower.",
			Visibility:   "known",
			Segments:     6,
			Progress:     4,
		},
	})
	storeProjectBoard(world, ProjectBoard{
		Projects: []ProjectClock{
			{
				ID:       "project-training",
				Title:    "Train with Lyanna",
				Kind:     "training",
				Status:   "active",
				Progress: 2,
				Segments: 4,
				Owner:    "Lyanna",
				Location: "Bell Quarter",
				Summary:  "Your sparring finally starts to look deliberate.",
				Stakes:   "Earn Lyanna's respect before the Bell Choir moves.",
				Rewards: []ProjectReward{
					{Kind: "skill", Label: "Blade Forms"},
				},
				Links: []ProjectLink{
					{Kind: "npc", Label: "Lyanna"},
					{Kind: "front", RefID: "front-bell-projects", Label: "Whispers Around the Bell Tower"},
				},
			},
			{
				ID:            "project-safehouse",
				Title:         "Restore the Lantern Loft",
				Kind:          "base",
				Status:        "completed",
				Progress:      4,
				Segments:      4,
				Location:      "Bell Quarter",
				Outcome:       "You now have a safe place to disappear for a night.",
				CompletedTurn: 8,
				Rewards: []ProjectReward{
					{Kind: "title", Label: "Lantern Loft Keeper"},
				},
				Links: []ProjectLink{
					{Kind: "place", Label: "Bell Quarter"},
				},
			},
		},
	})
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	index, err := BuildStoryCodex(db, story, char, world)
	if err != nil {
		t.Fatalf("BuildStoryCodex: %v", err)
	}

	projectIDs := index.CategoryEntries["projects"]
	if len(projectIDs) != 2 {
		t.Fatalf("project category entries = %v, want 2 project entries", projectIDs)
	}

	activeEntry, ok := index.Entry(codexProjectEntryID("project-training"))
	if !ok {
		t.Fatalf("missing active project entry")
	}
	activeSections := strings.Join(flattenCodexSections(activeEntry.Sections), "\n")
	if !strings.Contains(activeSections, "Progress: 2/4") {
		t.Fatalf("active project missing progress section:\n%s", activeSections)
	}
	if !strings.Contains(activeSections, "Skill: Blade Forms") {
		t.Fatalf("active project missing reward summary:\n%s", activeSections)
	}
	if !hasCodexLink(activeEntry.Related, codexNPCEntryID("Lyanna")) {
		t.Fatalf("active project missing npc link: %+v", activeEntry.Related)
	}
	if !hasCodexLink(activeEntry.Related, codexFrontEntryID("front-bell-projects")) {
		t.Fatalf("active project missing front link: %+v", activeEntry.Related)
	}

	completedEntry, ok := index.Entry(codexProjectEntryID("project-safehouse"))
	if !ok {
		t.Fatalf("missing completed project entry")
	}
	completedSections := strings.Join(flattenCodexSections(completedEntry.Sections), "\n")
	if !strings.Contains(completedSections, "Outcome") || !strings.Contains(completedSections, "safe place to disappear") {
		t.Fatalf("completed project missing durable outcome:\n%s", completedSections)
	}

	protagonist, ok := index.Entry(ProtagonistCodexEntryID())
	if !ok {
		t.Fatalf("missing protagonist entry")
	}
	protagonistSections := strings.Join(flattenCodexSections(protagonist.Sections), "\n")
	if !strings.Contains(protagonistSections, "Train with Lyanna 2/4 · training") {
		t.Fatalf("protagonist entry missing active project summary:\n%s", protagonistSections)
	}
	if !strings.Contains(protagonistSections, "Restore the Lantern Loft 4/4 · base · You now have a safe place to disappear for a night.") {
		t.Fatalf("protagonist entry missing completed project summary:\n%s", protagonistSections)
	}

	npcEntry, ok := index.Entry(codexNPCEntryID("Lyanna"))
	if !ok {
		t.Fatalf("missing npc entry")
	}
	if !hasCodexLink(npcEntry.Related, codexProjectEntryID("project-training")) {
		t.Fatalf("npc entry missing project backlink: %+v", npcEntry.Related)
	}

	frontEntry, ok := index.Entry(codexFrontEntryID("front-bell-projects"))
	if !ok {
		t.Fatalf("missing front entry")
	}
	if !hasCodexLink(frontEntry.Related, codexProjectEntryID("project-training")) {
		t.Fatalf("front entry missing project backlink: %+v", frontEntry.Related)
	}

	locationEntry, ok := index.Entry(codexLocationEntryID("Bell Quarter"))
	if !ok {
		t.Fatalf("missing location entry")
	}
	if !hasCodexLink(locationEntry.Related, codexProjectEntryID("project-safehouse")) {
		t.Fatalf("location entry missing project backlink: %+v", locationEntry.Related)
	}
}

func flattenCodexSections(sections []CodexSection) []string {
	lines := make([]string, 0, len(sections)*2)
	for _, section := range sections {
		lines = append(lines, section.Title)
		lines = append(lines, section.Lines...)
	}
	return lines
}

func hasCodexLink(links []CodexLink, entryID string) bool {
	for _, link := range links {
		if link.EntryID == entryID {
			return true
		}
	}
	return false
}
