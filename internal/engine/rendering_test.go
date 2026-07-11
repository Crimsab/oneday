package engine

import (
	"strings"
	"testing"
)

func TestStateChangesToEventCalloutsBuildsTrustedSummaries(t *testing.T) {
	callouts := StateChangesToEventCallouts([]StateChange{
		{Description: "Gained trait: Resolute"},
		{Description: "New NPC encountered: Lyanna (Scout)"},
		{Description: "Timeline advanced: Age 8 — First stable magical habit"},
		{Field: "location", New: "Silver Vale"},
		{Field: "inventory", New: map[string]interface{}{"name": "Moon Key"}},
	})

	if len(callouts) != 5 {
		t.Fatalf("expected 5 callouts, got %d", len(callouts))
	}
	if callouts[0].Kind != "trait" || callouts[0].Title != "Resolute" {
		t.Fatalf("unexpected trait callout: %+v", callouts[0])
	}
	if callouts[1].Kind != "npc" || callouts[1].Detail != "New NPC encountered" {
		t.Fatalf("unexpected npc callout: %+v", callouts[1])
	}
	if callouts[2].Kind != "timeline" || !strings.Contains(callouts[2].Title, "Age 8") {
		t.Fatalf("unexpected timeline callout: %+v", callouts[2])
	}
	if callouts[3].Kind != "location" || callouts[3].Title != "Silver Vale" {
		t.Fatalf("unexpected location callout: %+v", callouts[3])
	}
	if callouts[4].Kind != "item" || callouts[4].Title != "Moon Key" {
		t.Fatalf("unexpected item callout: %+v", callouts[4])
	}
}
