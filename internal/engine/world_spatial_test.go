package engine

import (
	"fmt"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestTravelPolicyFromSettingJSON(t *testing.T) {
	if got := TravelPolicyFromSettingJSON(`{"world_rules":{"travel_mode":"graph"}}`); got != "graph" {
		t.Fatalf("travel policy = %q, want graph", got)
	}
	for _, raw := range []string{"", `{}`, `{"world_rules":{"travel_mode":"narrative"}}`, `{invalid`} {
		if got := TravelPolicyFromSettingJSON(raw); got != "narrative" {
			t.Fatalf("travel policy for %q = %q, want narrative", raw, got)
		}
	}
}

func TestSpatialNarratorContextCapsLargeWorldsAndStatesTravelPolicy(t *testing.T) {
	world := &storage.PlayerWorldProjection{CurrentLocation: "Place 00", CurrentLocationID: "loc-00"}
	for i := range 80 {
		world.Locations = append(world.Locations, storage.CanonicalLocation{ID: fmt.Sprintf("loc-%02d", i), Name: fmt.Sprintf("Place %02d", i)})
	}
	for i := range 80 {
		world.Edges = append(world.Edges, storage.LocationEdge{FromLocationID: fmt.Sprintf("loc-%02d", i), ToLocationID: fmt.Sprintf("loc-%02d", (i+1)%80), TravelMode: "walk"})
	}

	context := FormatSpatialNarratorContext(world, "graph")
	if !strings.Contains(context, "Travel policy: graph") || !strings.Contains(context, "32 additional locations omitted") || !strings.Contains(context, "16 additional routes omitted") {
		t.Fatalf("large spatial context did not expose policy and truncation:\n%s", context)
	}
	if strings.Contains(context, "Place 79 [") {
		t.Fatal("spatial context included a location beyond its bounded window")
	}
}
