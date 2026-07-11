package views

import (
	"testing"

	"github.com/crimsab/oneday/internal/engine"
)

func TestApplyNarrativeResponseQueuesSocialDuelPrelude(t *testing.T) {
	model := newTalkTestNarrativeModel(t)
	model.width = 100
	model.height = 40

	model.applyNarrativeResponse(&engine.NarrativeResponse{
		Narrative: "Lyanna blocks the gangplank and waits.",
		Choices: []engine.Choice{
			{ID: 1, Text: "Try ordinary banter"},
		},
		SocialDuel: &engine.SocialDuelCue{
			Mode:      engine.SocialDuelCueOffer,
			NPCName:   "Lyanna",
			Objective: "Talk your way onto the ship",
			Stakes:    "If you fail, the ship leaves without you",
		},
	}, true)

	if model.pendingSocialDuel == nil {
		t.Fatal("pendingSocialDuel = nil, want queued duel prelude")
	}
	if model.pendingSocialDuel.Objective != "Talk your way onto the ship" {
		t.Fatalf("pendingSocialDuel objective = %q", model.pendingSocialDuel.Objective)
	}
	if model.choices.HasChoices() {
		t.Fatal("choices still visible during queued social duel")
	}
}

func TestApplyNarrativeResponseFallsBackToActiveSocialDuelCue(t *testing.T) {
	model := newTalkTestNarrativeModel(t)
	model.width = 100
	model.height = 40
	model.activeSocialDuel = &engine.SocialDuelState{
		NPCName:          "Lyanna",
		Objective:        "Keep Lyanna talking",
		Stakes:           "If she walks away, the clue is gone",
		Status:           engine.SocialDuelActive,
		Round:            2,
		LastExchangeNote: "She is close to losing patience.",
	}
	model.activeSocialDuelCue = &engine.SocialDuelCue{
		Mode:      engine.SocialDuelCueContinue,
		NPCName:   "Lyanna",
		Objective: "Keep Lyanna talking",
		Stakes:    "If she walks away, the clue is gone",
	}

	model.applyNarrativeResponse(&engine.NarrativeResponse{
		Narrative: "Lyanna drums her fingers on the railing.",
		Choices: []engine.Choice{
			{ID: 1, Text: "Back off"},
		},
	}, true)

	if model.pendingSocialDuel == nil {
		t.Fatal("pendingSocialDuel = nil, want fallback duel continuation")
	}
	if model.pendingSocialDuel.ExchangeSummary != "She is close to losing patience." {
		t.Fatalf("exchange_summary = %q, want last exchange note", model.pendingSocialDuel.ExchangeSummary)
	}
	if model.choices.HasChoices() {
		t.Fatal("choices still visible while duel should continue")
	}
}

func TestBeginPendingSocialDuelStartsViewAndState(t *testing.T) {
	model := newTalkTestNarrativeModel(t)
	model.width = 100
	model.height = 40
	model.pendingSocialDuel = &engine.SocialDuelCue{
		Mode:      engine.SocialDuelCueOffer,
		NPCName:   "Lyanna",
		Objective: "Get through the checkpoint",
		Stakes:    "If you fail, the gate closes for the night",
		Leverage: []engine.SocialDuelLeverage{
			{Key: "writ", Label: "Stamped harbor writ", Kind: "evidence"},
		},
	}

	model.beginPendingSocialDuel()

	if !model.inSocialDuel || model.socialDuelView == nil {
		t.Fatal("social duel view not started")
	}
	if model.activeSocialDuel == nil {
		t.Fatal("activeSocialDuel = nil, want seeded duel state")
	}
	if model.activeSocialDuel.Objective != "Get through the checkpoint" {
		t.Fatalf("active objective = %q", model.activeSocialDuel.Objective)
	}
	if len(model.activeSocialDuel.PlayerLeverage) != 1 {
		t.Fatalf("player leverage = %+v, want seeded cue leverage", model.activeSocialDuel.PlayerLeverage)
	}
}
