package engine

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestRelevantNPCsPrefersEligibleActiveNemesis(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "nemesis-reentry.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now()
	story := &storage.Story{ID: "story-nemesis-roster", Name: "Nemesis Roster", CreatedAt: now, UpdatedAt: now}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	nemesis := &storage.NPC{
		ID:                 "npc-nemesis",
		StoryID:            story.ID,
		Name:               "Lyanna",
		Role:               "broker",
		RelationshipJSON:   `{}`,
		NemesisJSON:        `{"status":"active","rivalry_score":7,"escalation_tier":3,"threat_posture":"vengeful","vow":"Lyanna swears this rivalry is not finished."}`,
		PrivateThoughts:    `[]`,
		NotesOnProtagonist: `[]`,
		IsAlive:            true,
		FirstAppearedTurn:  2,
		LastSeenTurn:       3,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	recent := &storage.NPC{
		ID:                 "npc-recent",
		StoryID:            story.ID,
		Name:               "Brother Alden",
		Role:               "healer",
		RelationshipJSON:   `{}`,
		NemesisJSON:        `{}`,
		PrivateThoughts:    `[]`,
		NotesOnProtagonist: `[]`,
		IsAlive:            true,
		FirstAppearedTurn:  4,
		LastSeenTurn:       10,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := db.CreateNPC(nemesis); err != nil {
		t.Fatalf("CreateNPC nemesis: %v", err)
	}
	if err := db.CreateNPC(recent); err != nil {
		t.Fatalf("CreateNPC recent: %v", err)
	}

	roster, err := RelevantNPCs(db, story.ID, 10, 3, 2)
	if err != nil {
		t.Fatalf("RelevantNPCs: %v", err)
	}
	if len(roster) != 2 {
		t.Fatalf("len(roster) = %d, want 2", len(roster))
	}
	if roster[0].Name != "Lyanna" {
		t.Fatalf("roster[0] = %q, want active nemesis Lyanna", roster[0].Name)
	}
}

func TestFormatNPCForContextIncludesNemesisSignals(t *testing.T) {
	npc := &storage.NPC{
		Name:               "Lyanna",
		Role:               "broker",
		RelationshipJSON:   `{"trust":5,"respect":3}`,
		NemesisJSON:        `{"status":"active","rivalry_score":6,"escalation_tier":2,"threat_posture":"political","vow":"Lyanna swears this rivalry is not finished.","visible_scars":["Publicly bent over the ledger"],"remembered_patterns":["appeal","pressure"]}`,
		PrivateThoughts:    `[]`,
		NotesOnProtagonist: `[]`,
	}

	text := FormatNPCForContext(npc)
	if !strings.Contains(text, "Nemesis status: active") {
		t.Fatalf("context missing nemesis status:\n%s", text)
	}
	if !strings.Contains(text, "Remembered player patterns: appeal, pressure") {
		t.Fatalf("context missing remembered patterns:\n%s", text)
	}
}
