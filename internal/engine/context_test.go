package engine

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildStateSummaryIncludesKnownFrontPressureAndHidesHiddenFronts(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	world.FrontsJSON = `[
		{
			"id":"front-known",
			"faction":"Bell Choir",
			"title":"The Silent Bell Choir is seeding sleepers across the district",
			"public_title":"Whispers Around the Bell Tower",
			"stakes":"Sleeper-priests will take the guard towers.",
			"visibility":"known",
			"segments":6,
			"progress":3,
			"pressures":[{"region":"Bell Quarter","kind":"suspicion","level":35}]
		},
		{
			"id":"front-hidden",
			"faction":"Ash Court",
			"title":"The Ash Court is buying judges in secret",
			"visibility":"hidden",
			"segments":4,
			"progress":1,
			"pressures":[{"region":"High Court","kind":"control","level":80}]
		}
	]`

	summary := buildStateSummary(char, world, nil, "")
	if !strings.Contains(summary, "Active Fronts: Whispers Around the Bell Tower {Bell Choir} 3/6") {
		t.Fatalf("summary missing known front headline:\n%s", summary)
	}
	if !strings.Contains(summary, "Bell Quarter [suspicion 35 rising]") {
		t.Fatalf("summary missing known front pressure:\n%s", summary)
	}
	if !strings.Contains(summary, "guards ask sharper questions") {
		t.Fatalf("summary missing derived systemic fallout:\n%s", summary)
	}
	if strings.Contains(summary, "Ash Court is buying judges in secret") {
		t.Fatalf("summary leaked hidden front title:\n%s", summary)
	}
	if strings.Contains(summary, "High Court [control 80 critical]") {
		t.Fatalf("summary leaked hidden front pressure:\n%s", summary)
	}
}

func TestBuildStateSummaryIncludesActiveNemesisRoster(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	npcs := []storage.NPC{
		{
			Name:        "Lyanna",
			Role:        "broker",
			NemesisJSON: `{"status":"active","escalation_tier":3,"threat_posture":"vengeful","rivalry_score":8}`,
		},
		{
			Name:        "Brother Alden",
			Role:        "healer",
			NemesisJSON: `{}`,
		},
	}

	summary := buildStateSummary(char, world, npcs, "")
	if !strings.Contains(summary, "Known NPCs: 2 (Lyanna, Brother Alden)") {
		t.Fatalf("summary missing npc roster:\n%s", summary)
	}
	if !strings.Contains(summary, "Active Nemeses: Lyanna(tier 3)") {
		t.Fatalf("summary missing active nemesis line:\n%s", summary)
	}
}
