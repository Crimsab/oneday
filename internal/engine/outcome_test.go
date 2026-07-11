package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/crimsab/oneday/internal/game/contracts"
)

func TestResolveChallengeInstanceIsDeterministic(t *testing.T) {
	instance := contracts.ChallengeInstance{
		ProtocolVersion: contracts.ChallengeProtocolVersion,
		ID:              "challenge-replay",
		Definition: contracts.ChallengeDefinition{
			ID: "locked-door", Kind: "action", Difficulty: 55,
		},
		Seed:   90210,
		Policy: contracts.OutcomePolicy{ID: "balanced", CriticalBand: 5, ConsequenceBudget: 2, Fairness: "fail_forward"},
	}
	input := contracts.ChallengeInput{ActorID: "hero", Intent: "pick the lock", Modifiers: []contracts.ChallengeModifier{{Source: "lockpicks", Value: 8}}}
	first, err := ResolveChallengeInstance(instance, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolveChallengeInstance(instance, input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay differs:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if first.Outcome.Seed != instance.Seed || first.Outcome.Roll < 1 || first.Outcome.Roll > 100 {
		t.Fatalf("invalid deterministic outcome: %+v", first.Outcome)
	}
}

func TestOutcomeDegreesCoverSixAuthoritativeStates(t *testing.T) {
	want := []contracts.OutcomeDegree{
		contracts.OutcomeCriticalSuccess, contracts.OutcomeFullSuccess,
		contracts.OutcomeSuccessWithCost, contracts.OutcomeFailureWithProgress,
		contracts.OutcomeHardFailure, contracts.OutcomeCatastrophe,
	}
	for _, degree := range want {
		if !contracts.ValidOutcomeDegree(degree) {
			t.Fatalf("degree %q is not valid", degree)
		}
	}
}

func TestLegacyBooleanMappingIsStable(t *testing.T) {
	if got := OutcomeFromLegacy(true, 50); got.Degree != contracts.OutcomeFullSuccess || !got.Succeeded() {
		t.Fatalf("legacy pass mapped to %+v", got)
	}
	if got := OutcomeFromLegacy(false, 50); got.Degree != contracts.OutcomeFailureWithProgress || got.Succeeded() {
		t.Fatalf("legacy fail mapped to %+v", got)
	}
}

func TestEnforceOutcomeRemovesUnsupportedRewardsOnFailure(t *testing.T) {
	narrative := &NarrativeResponse{StateChanges: map[string]any{
		"inventory_add": []any{"Crown"}, "title_add": "Conqueror", "fail_forward": map[string]any{"title": "Alarm"},
	}}
	outcome := OutcomeFromLegacy(false, 60)
	EnforceOutcomeNarrative(narrative, &outcome)
	if _, ok := narrative.StateChanges["inventory_add"]; ok {
		t.Fatal("failure retained unsupported inventory reward")
	}
	if _, ok := narrative.StateChanges["title_add"]; ok {
		t.Fatal("failure retained unsupported title")
	}
	if _, ok := narrative.StateChanges["fail_forward"]; !ok {
		t.Fatal("failure consequence was removed")
	}
	if narrative.ResolvedOutcome == nil || narrative.ResolvedOutcome.Degree != contracts.OutcomeFailureWithProgress {
		t.Fatalf("resolved outcome not attached: %+v", narrative.ResolvedOutcome)
	}
}

func TestPortableChallengeContractJSON(t *testing.T) {
	instance := NewOrdinaryActionChallenge("story-1", "branch-1", 7, "Sneak past the guard", contracts.OutcomePolicy{ID: "balanced"})
	raw, err := json.Marshal(instance)
	if err != nil {
		t.Fatal(err)
	}
	var decoded contracts.ChallengeInstance
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("decoded contract invalid: %v", err)
	}
}
