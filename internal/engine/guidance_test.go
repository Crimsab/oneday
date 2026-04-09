package engine

import (
	"testing"
)

func TestUpsertPlayerGuidanceMergesDuplicateDirectives(t *testing.T) {
	existing := []PlayerGuidance{
		{
			ID:            "guide-1",
			Kind:          "boss_fight",
			Title:         "Boss fight memorabile",
			Detail:        "Serve un boss di capitolo veramente pesante.",
			Scope:         "chapter",
			Priority:      "high",
			Status:        "active",
			RequestedTurn: 5,
		},
	}

	merged := upsertPlayerGuidance(existing, []PlayerGuidance{
		{
			Kind:     "boss_fight",
			Title:    "Boss fight memorabile",
			Detail:   "Meglio se ha piu fasi e un reward forte.",
			Scope:    "chapter",
			Priority: "high",
			Status:   "active",
		},
	}, 9)

	if len(merged) != 1 {
		t.Fatalf("merged guidance len = %d, want 1", len(merged))
	}
	if merged[0].RequestedTurn != 5 {
		t.Fatalf("RequestedTurn = %d, want to preserve original 5", merged[0].RequestedTurn)
	}
	if merged[0].UpdatedTurn != 9 {
		t.Fatalf("UpdatedTurn = %d, want 9", merged[0].UpdatedTurn)
	}
	if merged[0].Detail != "Meglio se ha piu fasi e un reward forte." {
		t.Fatalf("Detail = %q, want merged latest detail", merged[0].Detail)
	}
}

func TestApplyStateChangesGuideUpdateProgressesGuidance(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	storePlayerGuidance(world, []PlayerGuidance{
		{
			ID:            "guide-1",
			Kind:          "boss_fight",
			Title:         "Boss fight memorabile",
			Detail:        "Serve un boss di capitolo veramente pesante.",
			Scope:         "chapter",
			Priority:      "high",
			Status:        "active",
			RequestedTurn: 4,
		},
	})

	applied, err := ApplyStateChanges(map[string]interface{}{
		"guide_update": map[string]interface{}{
			"title":    "Boss fight memorabile",
			"status":   "seeded",
			"progress": "Un cacciatore corazzato inizia a inseguirti.",
		},
	}, char, world, nil, "test-story-id", 7)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("applied len = %d, want 1", len(applied))
	}

	guidance := loadPlayerGuidance(world)
	if len(guidance) != 1 {
		t.Fatalf("guidance len = %d, want 1", len(guidance))
	}
	if guidance[0].Status != "seeded" {
		t.Fatalf("Status = %q, want seeded", guidance[0].Status)
	}
	if guidance[0].Progress != "Un cacciatore corazzato inizia a inseguirti." {
		t.Fatalf("Progress = %q", guidance[0].Progress)
	}
	if guidance[0].UpdatedTurn != 7 {
		t.Fatalf("UpdatedTurn = %d, want 7", guidance[0].UpdatedTurn)
	}
}
