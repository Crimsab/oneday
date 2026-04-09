package engine

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestRecordNemesisEventIgnoresLowImpactOrdinaryRival(t *testing.T) {
	npc := &storage.NPC{Name: "Lyanna", NemesisJSON: "{}"}

	profile := RecordNemesisEvent(npc, NemesisEvent{
		Kind:   "social_duel",
		Turn:   8,
		Impact: 1,
	})
	if profile != nil {
		t.Fatalf("profile = %+v, want nil for low-impact ordinary rival", profile)
	}
	if npc.NemesisJSON != "{}" {
		t.Fatalf("nemesis_json = %s, want {}", npc.NemesisJSON)
	}
}

func TestRecordNemesisEventPromotesAfterStackedHumiliation(t *testing.T) {
	npc := &storage.NPC{Name: "Lyanna", NemesisJSON: "{}"}

	profile := RecordNemesisEvent(npc, NemesisEvent{
		Kind:    "humiliation",
		Turn:    10,
		Impact:  3,
		Detail:  "public defeat at Old Harbor",
		Pattern: "appeal",
		Scar:    "Publicly bent at Old Harbor",
	})
	if profile == nil {
		t.Fatal("profile = nil after qualifying humiliation")
	}
	if profile.Status != NemesisStatusRival {
		t.Fatalf("status = %q, want rival after first event", profile.Status)
	}

	profile = RecordNemesisEvent(npc, NemesisEvent{
		Kind:    "political_fallout",
		Turn:    14,
		Impact:  2,
		Detail:  "the harbor syndicate lost face",
		Pattern: "pressure",
	})
	if profile == nil {
		t.Fatal("profile = nil after second event")
	}
	if profile.Status != NemesisStatusActive {
		t.Fatalf("status = %q, want active nemesis", profile.Status)
	}
	if profile.EscalationTier < 2 {
		t.Fatalf("escalation_tier = %d, want >= 2", profile.EscalationTier)
	}
	if len(profile.RememberedPatterns) != 2 {
		t.Fatalf("remembered_patterns = %+v, want both patterns", profile.RememberedPatterns)
	}
	if len(profile.VisibleScars) != 1 || !strings.Contains(profile.VisibleScars[0], "Old Harbor") {
		t.Fatalf("visible_scars = %+v, want recorded humiliation scar", profile.VisibleScars)
	}
	if strings.TrimSpace(profile.Vow) == "" {
		t.Fatal("vow empty after promotion")
	}
	if !strings.Contains(npc.NemesisJSON, `"status":"active"`) {
		t.Fatalf("nemesis_json = %s, want active persistence", npc.NemesisJSON)
	}
}
