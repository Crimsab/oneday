package engine

import "testing"

func TestStateChangesToEventCalloutsBuildsTrustedSummaries(t *testing.T) {
	callouts := StateChangesToEventCallouts([]StateChange{
		{Description: "Gained trait: Resolute"},
		{Description: "New NPC encountered: Lyanna (Scout)"},
		{Field: "location", New: "Silver Vale"},
		{Field: "inventory", New: map[string]interface{}{"name": "Moon Key"}},
	})

	if len(callouts) != 4 {
		t.Fatalf("expected 4 callouts, got %d", len(callouts))
	}
	if callouts[0].Kind != "trait" || callouts[0].Title != "Resolute" {
		t.Fatalf("unexpected trait callout: %+v", callouts[0])
	}
	if callouts[1].Kind != "npc" || callouts[1].Detail != "New NPC encountered" {
		t.Fatalf("unexpected npc callout: %+v", callouts[1])
	}
	if callouts[2].Kind != "location" || callouts[2].Title != "Silver Vale" {
		t.Fatalf("unexpected location callout: %+v", callouts[2])
	}
	if callouts[3].Kind != "item" || callouts[3].Title != "Moon Key" {
		t.Fatalf("unexpected item callout: %+v", callouts[3])
	}
}
