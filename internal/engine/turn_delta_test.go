package engine

import "testing"

func TestBuildTurnDeltaClassifiesActiveSystemKinds(t *testing.T) {
	t.Parallel()

	delta := buildTurnDelta([]StateChange{
		{Field: "front.whispers", Description: "Front advances: Whispers Around the Bell Tower"},
		{Field: "front_pressure.bell-quarter.suspicion", Description: "Pressure rises in Bell Quarter"},
		{Field: "project.train-with-lyanna", Description: "Project advanced: Train with Lyanna"},
		{Field: "investigation.case-harbor.clue", Description: "Clue added: Missing seal"},
	})
	if delta == nil || len(delta.Items) != 4 {
		t.Fatalf("delta = %+v, want 4 items", delta)
	}
	if delta.Items[0].Kind != "front" {
		t.Fatalf("delta.Items[0].Kind = %q, want front", delta.Items[0].Kind)
	}
	if delta.Items[1].Kind != "front" {
		t.Fatalf("delta.Items[1].Kind = %q, want front", delta.Items[1].Kind)
	}
	if delta.Items[2].Kind != "project" {
		t.Fatalf("delta.Items[2].Kind = %q, want project", delta.Items[2].Kind)
	}
	if delta.Items[3].Kind != "investigation" {
		t.Fatalf("delta.Items[3].Kind = %q, want investigation", delta.Items[3].Kind)
	}
}
