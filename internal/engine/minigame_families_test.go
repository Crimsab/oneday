package engine

import (
	"testing"

	"github.com/crimsab/oneday/internal/game/contracts"
)

func TestGenreNeutralMiniGameFamilies(t *testing.T) {
	tests := []struct {
		name       string
		definition MiniGameDefinition
		input      MiniGameInput
		degree     contracts.OutcomeDegree
	}{
		{"deduction", MiniGameDefinition{ID: "d", Kind: MiniGameDeduction, Difficulty: 50, Answers: []string{"the curator"}, Rules: map[string]string{"required_evidence": "2"}}, MiniGameInput{Action: "submit", Value: "curator", Values: []string{"broken seal", "ink trace"}}, contracts.OutcomeFullSuccess},
		{"negotiation", MiniGameDefinition{ID: "n", Kind: MiniGameNegotiation, Difficulty: 25, Options: []string{"cooperative", "assertive"}}, MiniGameInput{Action: "submit", Value: "cooperative", Values: []string{"shared goal", "deadline"}}, contracts.OutcomeFullSuccess},
		{"pattern", MiniGameDefinition{ID: "p", Kind: MiniGamePattern, Difficulty: 50, Answers: []string{"8"}}, MiniGameInput{Action: "submit", Value: "8"}, contracts.OutcomeFullSuccess},
		{"bidding", MiniGameDefinition{ID: "b", Kind: MiniGameBidding, Difficulty: 50, Rules: map[string]string{"reserve": "40", "market_value": "60", "budget": "90"}}, MiniGameInput{Action: "submit", Value: "55"}, contracts.OutcomeFullSuccess},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host := NewMiniGameHost()
			instance := NewMiniGameInstance("mini-"+test.name, "story", "branch", 1, 7, test.definition)
			if err := host.Start(&instance); err != nil {
				t.Fatal(err)
			}
			if err := host.Apply(&instance, test.input); err != nil {
				t.Fatal(err)
			}
			if instance.Runtime.Result == nil || instance.Runtime.Result.Outcome.Degree != test.degree {
				t.Fatalf("degree = %+v, want %s", instance.Runtime.Result, test.degree)
			}
			replay, err := host.Replay(instance)
			if err != nil || replay.Runtime.Result.Outcome.Degree != test.degree {
				t.Fatalf("replay = %+v err=%v", replay, err)
			}
		})
	}
}

func TestBiddingOverBudgetIsCatastrophe(t *testing.T) {
	host := NewMiniGameHost()
	instance := NewMiniGameInstance("mini-bid", "story", "branch", 1, 9, MiniGameDefinition{ID: "b", Kind: MiniGameBidding, Difficulty: 50, Rules: map[string]string{"reserve": "40", "market_value": "60", "budget": "90"}})
	if err := host.Start(&instance); err != nil {
		t.Fatal(err)
	}
	if err := host.Apply(&instance, MiniGameInput{Action: "submit", Value: "100"}); err != nil {
		t.Fatal(err)
	}
	if got := instance.Runtime.Result.Outcome.Degree; got != contracts.OutcomeCatastrophe {
		t.Fatalf("degree = %s", got)
	}
}
