package engine

import (
	"encoding/json"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestParseNarrativeFromAINormalizesSocialDuelOffer(t *testing.T) {
	input := `{
		"narrative": "Lyanna lets the silence do half the work.",
		"choices": [{"id":1,"text":"Answer carefully"}],
		"social_duel": {
			"mode": " offer ",
			"npc_name": " Lyanna ",
			"objective": " Convince her to reveal the harbor ledger ",
			"stakes": " If you fail, the watch arrives ",
			"pressure": " Everyone in the room is listening ",
			"opening": "\"One lie and this ends badly.\"",
			"leverage": [
				{"label":" Harbor Ledger ","detail":"Proof of the bribe trail","kind":"Evidence"},
				{"label":"Harbor Ledger","detail":"duplicate and should collapse","kind":"evidence"}
			],
			"suggested_actions": ["appeal", " expose ", "unknown", "appeal"]
		}
	}`

	nr, err := parseNarrativeFromAI(input)
	if err != nil {
		t.Fatalf("parseNarrativeFromAI: %v", err)
	}
	normalizeNarrativeResponse(nr)

	if nr.SocialDuel == nil {
		t.Fatal("SocialDuel = nil, want normalized cue")
	}
	if nr.SocialDuel.Mode != SocialDuelCueOffer {
		t.Fatalf("mode = %q, want %q", nr.SocialDuel.Mode, SocialDuelCueOffer)
	}
	if nr.SocialDuel.NPCName != "Lyanna" {
		t.Fatalf("npc_name = %q, want Lyanna", nr.SocialDuel.NPCName)
	}
	if got := len(nr.SocialDuel.Leverage); got != 1 {
		t.Fatalf("leverage count = %d, want 1", got)
	}
	if nr.SocialDuel.Leverage[0].Key != "harbor-ledger" {
		t.Fatalf("leverage key = %q, want harbor-ledger", nr.SocialDuel.Leverage[0].Key)
	}
	if got := len(nr.SocialDuel.SuggestedActions); got != 2 {
		t.Fatalf("suggested action count = %d, want 2", got)
	}
	if nr.SocialDuel.SuggestedActions[0] != SocialActionAppeal || nr.SocialDuel.SuggestedActions[1] != SocialActionExpose {
		t.Fatalf("suggested actions = %#v, want [appeal expose]", nr.SocialDuel.SuggestedActions)
	}
}

func TestNormalizeNarrativeResponseDowngradesPartialSocialDuelOffer(t *testing.T) {
	nr := &NarrativeResponse{
		Narrative: "The magistrate waits for your answer.",
		Choices:   []Choice{{ID: 1, Text: "Speak"}},
		SocialDuel: &SocialDuelCue{
			Mode:            SocialDuelCueOffer,
			Pressure:        "The courtroom gallery is hostile.",
			ExchangeSummary: "You only get one chance to seize the room.",
			SuggestedActions: []SocialAction{
				SocialActionPressure,
				"invalid",
			},
		},
	}

	normalizeNarrativeResponse(nr)

	if nr.SocialDuel == nil {
		t.Fatal("SocialDuel = nil, want continue fallback")
	}
	if nr.SocialDuel.Mode != SocialDuelCueContinue {
		t.Fatalf("mode = %q, want %q", nr.SocialDuel.Mode, SocialDuelCueContinue)
	}
	if nr.SocialDuel.ExchangeSummary == "" {
		t.Fatal("exchange_summary empty after fallback")
	}
	if got := len(nr.SocialDuel.SuggestedActions); got != 1 {
		t.Fatalf("suggested action count = %d, want 1", got)
	}
	if nr.SocialDuel.SuggestedActions[0] != SocialActionPressure {
		t.Fatalf("suggested action = %q, want pressure", nr.SocialDuel.SuggestedActions[0])
	}
}

func TestResumeNarrativeFromStoredMessageRestoresSocialDuelCue(t *testing.T) {
	output := &ChatOutput{
		Narrative: "Lyanna studies you in silence.",
		ChoicesData: []Choice{
			{ID: 1, Text: "Tell the truth"},
		},
		Location: "Old Harbor",
		SocialDuel: &SocialDuelCue{
			Mode:             SocialDuelCueContinue,
			NPCName:          "Lyanna",
			Objective:        "Keep her talking",
			ExchangeSummary:  "The room is quiet enough to hear every breath.",
			SuggestedActions: []SocialAction{SocialActionAppeal, SocialActionExpose},
		},
	}

	meta := persistedAssistantMeta{
		Mood:     "tense",
		Location: "Old Harbor",
		Output:   output,
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	msg := storage.ChatMessage{
		Content:      output.Narrative,
		MetadataJSON: string(raw),
	}

	nr := resumeNarrativeFromStoredMessage(msg, "Fallback")
	if nr == nil {
		t.Fatal("resumeNarrativeFromStoredMessage = nil")
	}
	if nr.SocialDuel == nil {
		t.Fatal("SocialDuel = nil, want restored cue")
	}
	if nr.SocialDuel.Mode != SocialDuelCueContinue {
		t.Fatalf("mode = %q, want %q", nr.SocialDuel.Mode, SocialDuelCueContinue)
	}
	if nr.SocialDuel.NPCName != "Lyanna" {
		t.Fatalf("npc_name = %q, want Lyanna", nr.SocialDuel.NPCName)
	}
	if got := len(nr.SocialDuel.SuggestedActions); got != 2 {
		t.Fatalf("suggested action count = %d, want 2", got)
	}
}
