package engine

import (
	"strings"
	"testing"
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
