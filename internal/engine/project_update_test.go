package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestApplyStateChangesProjectUpdateAdvancesWithCostAndPressure(t *testing.T) {
	char := newTestChar()
	char.StatsJSON = `{"currency":10}`
	world := newTestWorld()
	world.FrontsJSON = `[
		{
			"id":"front-known",
			"faction":"Bell Choir",
			"title":"The Silent Bell Choir is seeding sleepers across the district",
			"public_title":"Whispers Around the Bell Tower",
			"visibility":"known",
			"segments":4,
			"progress":1
		}
	]`

	applied, err := ApplyStateChanges(map[string]interface{}{
		"project_update": map[string]interface{}{
			"action":          "advance",
			"title":           "Train with Lyanna",
			"kind":            "training",
			"segments":        4,
			"amount":          1,
			"summary":         "You finally start putting the footwork together.",
			"currency_cost":   2,
			"front_id":        "front-known",
			"front_advance":   1,
			"pressure_region": "Bell Quarter",
			"pressure_kind":   "suspicion",
			"pressure_change": 15,
		},
	}, char, world, nil, "test-story", 6)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected project-related state changes")
	}

	stats := parseLooseJSONMap(char.StatsJSON)
	if testToInt(stats["currency"]) != 8 {
		t.Fatalf("currency = %v, want 8 after cost", stats["currency"])
	}

	board := loadProjectBoard(world)
	if len(board.Projects) != 1 {
		t.Fatalf("projects = %+v, want 1 project", board.Projects)
	}
	project := board.Projects[0]
	if project.Progress != 1 || project.Segments != 4 {
		t.Fatalf("project progress = %d/%d, want 1/4", project.Progress, project.Segments)
	}

	fronts := loadFronts(world)
	if len(fronts) != 1 || fronts[0].Progress != 2 {
		t.Fatalf("fronts = %+v, want progress advanced to 2", fronts)
	}
	if len(fronts[0].Pressures) != 1 || fronts[0].Pressures[0].Level != 15 {
		t.Fatalf("front pressure = %+v, want Bell Quarter suspicion 15", fronts[0].Pressures)
	}
}

func TestApplyStateChangesProjectUpdateSetbackCreatesFailForward(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	storeProjectBoard(world, ProjectBoard{
		Projects: []ProjectClock{
			{
				ID:       "project-lyanna",
				Title:    "Train with Lyanna",
				Kind:     "training",
				Status:   "active",
				Progress: 2,
				Segments: 4,
			},
		},
	})

	applied, err := ApplyStateChanges(map[string]interface{}{
		"project_update": map[string]interface{}{
			"action":              "setback",
			"id":                  "project-lyanna",
			"amount":              1,
			"fail_forward_title":  "Lyanna calls out your sloppy guard",
			"fail_forward_detail": "The embarrassment stings, but the lesson gets sharper.",
		},
	}, char, world, nil, "test-story", 7)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied changes for setback")
	}

	board := loadProjectBoard(world)
	if board.Projects[0].Progress != 1 {
		t.Fatalf("progress = %d, want setback to reduce progress to 1", board.Projects[0].Progress)
	}
	reactions := visibleWorldReactions(loadWorldReactions(world))
	if len(reactions) != 1 || !strings.Contains(reactions[0].Title, "Lyanna calls out your sloppy guard") {
		t.Fatalf("reactions = %+v, want fail-forward reaction", reactions)
	}
}

func TestApplyStateChangesProjectUpdateCompleteAppliesDurableRewards(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:        "story-project-complete",
		Name:      "Project Completion Story",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID

	npc := &storage.NPC{
		ID:                 "npc-lyanna-project",
		StoryID:            story.ID,
		Name:               "Lyanna",
		Role:               "duelist",
		RelationshipJSON:   `{"trust":2,"respect":1}`,
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

	world := newTestWorld()
	world.StoryID = story.ID
	world.UpdatedAt = now

	applied, err := ApplyStateChanges(map[string]interface{}{
		"project_update": map[string]interface{}{
			"action":   "complete",
			"title":    "Train with Lyanna",
			"kind":     "relationship",
			"segments": 4,
			"summary":  "Your drills finally stop looking clumsy.",
			"outcome":  "Lyanna begins treating you like a real student.",
			"owner":    "Lyanna",
			"links": []interface{}{
				map[string]interface{}{"kind": "npc", "label": "Lyanna"},
			},
			"rewards": []interface{}{
				map[string]interface{}{"kind": "skill", "label": "Blade Forms"},
				map[string]interface{}{"kind": "trait", "label": "Patient Footwork"},
				map[string]interface{}{"kind": "title", "label": "Lyanna's Proven Student"},
				map[string]interface{}{"kind": "item", "label": "Weighted Practice Blade", "detail": "A balanced sparring weapon Lyanna lets you keep."},
				map[string]interface{}{"kind": "relationship", "label": "Lyanna", "detail": "She stops treating you like dead weight."},
			},
		},
	}, char, world, db, story.ID, 10)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied changes for completed project")
	}

	stats := parseStats(t, char)
	skills := toSkillsMap(stats["skills"])
	if _, ok := skills["Blade Forms"]; !ok {
		t.Fatalf("skills = %+v, want Blade Forms unlocked", skills)
	}
	if !strings.Contains(char.TraitsJSON, "Patient Footwork") {
		t.Fatalf("traits_json = %s, want Patient Footwork", char.TraitsJSON)
	}
	if !strings.Contains(char.StatsJSON, "Lyanna's Proven Student") {
		t.Fatalf("stats_json = %s, want title reward persisted", char.StatsJSON)
	}
	if !strings.Contains(char.InventoryJSON, "Weighted Practice Blade") {
		t.Fatalf("inventory_json = %s, want item reward persisted", char.InventoryJSON)
	}

	board := loadProjectBoard(world)
	if len(board.Projects) != 1 || !strings.EqualFold(board.Projects[0].Status, "completed") {
		t.Fatalf("project board = %+v, want one completed project", board.Projects)
	}
	if board.Projects[0].Outcome != "Lyanna begins treating you like a real student." {
		t.Fatalf("project outcome = %q, want durable outcome persisted", board.Projects[0].Outcome)
	}

	updatedNPC, err := db.GetNPCByName(story.ID, "Lyanna")
	if err != nil || updatedNPC == nil {
		t.Fatalf("GetNPCByName: %v, npc = %+v", err, updatedNPC)
	}
	axes := loadRelationshipAxes(updatedNPC)
	if axes.Trust <= 2 || axes.Respect <= 1 {
		t.Fatalf("relationship axes = %+v, want project completion boost", axes)
	}
	if updatedNPC.Disposition <= 0 {
		t.Fatalf("disposition = %d, want positive project completion bump", updatedNPC.Disposition)
	}
	if !strings.Contains(updatedNPC.NotesOnProtagonist, "dead weight") {
		t.Fatalf("notes_on_protagonist = %s, want durable project note", updatedNPC.NotesOnProtagonist)
	}

	reactions := visibleWorldReactions(loadWorldReactions(world))
	foundCompletionEcho := false
	for _, reaction := range reactions {
		if strings.Contains(reaction.Title, "Project completed: Train with Lyanna") {
			foundCompletionEcho = true
			break
		}
	}
	if !foundCompletionEcho {
		t.Fatalf("reactions = %+v, want completion echo in world reactions", reactions)
	}
}
