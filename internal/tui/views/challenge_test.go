package views

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/engine"
)

func TestChallengePreludeUsesDescriptionAndFlowHint(t *testing.T) {
	t.Parallel()

	model := newTalkTestNarrativeModel(t)
	model.SetSize(100, 30)
	model.pendingChallenge = &engine.ChallengeSpec{
		Type:        engine.ChallengeDiceRoll,
		Difficulty:  60,
		Description: "Pick the rusted lock before the guard returns.",
	}

	view := model.challengePreludeView()
	if !strings.Contains(view, "Pick the rusted lock before the guard returns.") {
		t.Fatalf("challenge prelude missing description: %q", view)
	}
	if !strings.Contains(view, "continue the scene automatically") {
		t.Fatalf("challenge prelude missing continuation hint: %q", view)
	}
}

func TestApplyNarrativeResponseSuppressesChoicesWhenChallengesPending(t *testing.T) {
	t.Parallel()

	model := newTalkTestNarrativeModel(t)
	model.SetSize(100, 30)

	model.applyNarrativeResponse(&engine.NarrativeResponse{
		Narrative: "The lock clicks under your fingers, but one final pin resists.",
		Choices: []engine.Choice{
			{ID: 1, Text: "Force it"},
			{ID: 2, Text: "Back away"},
		},
		Challenges: []*engine.ChallengeSpec{
			{
				Type:        engine.ChallengeDiceRoll,
				Difficulty:  60,
				Description: "Pick the rusted lock before the guard returns.",
			},
		},
	}, true)

	if model.choices.HasChoices() {
		t.Fatal("choices should stay hidden while a challenge is pending")
	}
	if model.pendingChallenge == nil {
		t.Fatal("pendingChallenge = nil, want queued challenge prelude")
	}
}
