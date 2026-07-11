package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/crimsab/oneday/internal/game/contracts"
)

func DefaultOutcomePolicy(genre, difficultyProfile string) contracts.OutcomePolicy {
	budget, criticalBand := 2, 5
	switch strings.ToLower(strings.TrimSpace(difficultyProfile)) {
	case "generous", "easy", "story":
		budget = 1
	case "harsh", "hard", "brutal":
		budget, criticalBand = 3, 7
	}
	return contracts.OutcomePolicy{ID: "oneday.v1." + fallbackOutcomeString(strings.ToLower(difficultyProfile), "balanced"), Genre: genre, DifficultyProfile: fallbackOutcomeString(difficultyProfile, "balanced"), ConsequenceBudget: budget, CriticalBand: criticalBand, Fairness: "fail_forward"}
}

func fallbackOutcomeString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func NewOrdinaryActionChallenge(storyID, branchID string, turn int, intent string, policy contracts.OutcomePolicy) contracts.ChallengeInstance {
	if policy.ID == "" {
		policy = DefaultOutcomePolicy("", "balanced")
	}
	difficulty := 50
	lower := strings.ToLower(intent)
	if strings.Contains(lower, "carefully") || strings.Contains(lower, "cautious") || strings.Contains(lower, "con attenzione") {
		difficulty = 45
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("oneday.challenge.v1\x00%s\x00%s\x00%d\x00%s", storyID, branchID, turn, intent)))
	seed := int64(binary.BigEndian.Uint64(digest[:8]) & uint64(contracts.MaxPortableChallengeSeed))
	return contracts.ChallengeInstance{ProtocolVersion: contracts.ChallengeProtocolVersion, ID: fmt.Sprintf("challenge-%x", digest[:12]), StoryID: storyID, BranchID: branchID, Turn: turn, Definition: contracts.ChallengeDefinition{ID: "ordinary-action", Kind: "action", Description: intent, Difficulty: difficulty}, Seed: seed, Policy: policy}
}

func ResolveChallengeInstance(instance contracts.ChallengeInstance, input contracts.ChallengeInput) (*contracts.ChallengeResolution, error) {
	if err := instance.Validate(); err != nil {
		return nil, err
	}
	roll := rand.New(rand.NewSource(instance.Seed)).Intn(100) + 1
	modifierTotal := 0
	for _, modifier := range input.Modifiers {
		modifierTotal += modifier.Value
	}
	total := roll + modifierTotal
	margin := total - instance.Definition.Difficulty
	band := instance.Policy.CriticalBand
	if band <= 0 {
		band = 5
	}
	degree := contracts.OutcomeHardFailure
	switch {
	case roll > 100-band:
		degree = contracts.OutcomeCriticalSuccess
	case roll <= band:
		degree = contracts.OutcomeCatastrophe
	case margin >= 0:
		degree = contracts.OutcomeFullSuccess
	case margin >= -10:
		degree = contracts.OutcomeSuccessWithCost
	case margin >= -25:
		degree = contracts.OutcomeFailureWithProgress
	}
	outcome := contracts.OutcomeEnvelope{Version: contracts.ChallengeProtocolVersion, Degree: degree, Difficulty: instance.Definition.Difficulty, Seed: instance.Seed, Roll: roll, Modifiers: input.Modifiers, Total: total, Margin: margin}
	applyDefaultOutcomeBudget(&outcome, instance.Policy)
	return &contracts.ChallengeResolution{ProtocolVersion: contracts.ChallengeProtocolVersion, InstanceID: instance.ID, Input: input, Outcome: outcome}, nil
}

func applyDefaultOutcomeBudget(outcome *contracts.OutcomeEnvelope, policy contracts.OutcomePolicy) {
	budget := policy.ConsequenceBudget
	if budget <= 0 {
		budget = 2
	}
	switch outcome.Degree {
	case contracts.OutcomeCriticalSuccess:
		outcome.RevealedFacts = []string{"The action reveals an additional useful opportunity."}
	case contracts.OutcomeSuccessWithCost:
		outcome.Costs = []contracts.OutcomeEffect{{Kind: "pressure", Amount: 1, Detail: "Success creates a complication."}}
		outcome.Consequences = []string{"The intent succeeds, but the cost must remain true in canon."}
		outcome.FollowUpPressure = 1
	case contracts.OutcomeFailureWithProgress:
		outcome.StateDeltas = []contracts.OutcomeEffect{{Kind: "progress", Amount: 1, Detail: "Failure still changes the situation."}}
		outcome.Consequences = []string{"The intent does not succeed, but useful progress or information remains."}
		outcome.FollowUpPressure = 1
	case contracts.OutcomeHardFailure:
		outcome.Consequences = []string{"The intent fails and creates a concrete setback."}
		outcome.FollowUpPressure = budget
	case contracts.OutcomeCatastrophe:
		outcome.Consequences = []string{"The intent fails and the situation escalates sharply."}
		outcome.FollowUpPressure = budget + 1
	}
}

func OutcomeFromLegacy(passed bool, difficulty int) contracts.OutcomeEnvelope {
	degree, margin := contracts.OutcomeFailureWithProgress, -1
	if passed {
		degree, margin = contracts.OutcomeFullSuccess, 0
	}
	return contracts.OutcomeEnvelope{Version: contracts.ChallengeProtocolVersion, Degree: degree, Difficulty: difficulty, Margin: margin}
}

func EnsureLegacyChallengeOutcome(result *ChallengeResult) *ChallengeResult {
	if result == nil || result.Outcome != nil {
		return result
	}
	outcome := OutcomeFromLegacy(result.Passed, result.Difficulty)
	if result.Roll != 0 || len(result.RollLog) > 0 {
		outcome.Roll, outcome.Total = result.Roll, result.Total
		outcome.Margin = result.Total - result.Difficulty
	}
	if len(result.RollLog) > 0 {
		outcome.Seed = result.RollLog[0].Seed & contracts.MaxPortableChallengeSeed
	}
	result.Outcome = &outcome
	return result
}

func EnforceOutcomeNarrative(narrative *NarrativeResponse, outcome *contracts.OutcomeEnvelope) {
	if narrative == nil || outcome == nil {
		return
	}
	narrative.ResolvedOutcome = outcome
	if outcome.Succeeded() {
		return
	}
	for _, key := range []string{"inventory_add", "trait_add", "title_add", "skill_learn", "skill_xp", "attributes", "secondary", "currency", "achievement_earned"} {
		delete(narrative.StateChanges, key)
	}
}

func OutcomePromptContract(instance contracts.ChallengeInstance, resolution contracts.ChallengeResolution) string {
	payload, _ := json.Marshal(struct {
		Instance contracts.ChallengeInstance `json:"instance"`
		Outcome  contracts.OutcomeEnvelope   `json:"outcome"`
	}{instance, resolution.Outcome})
	return "## AUTHORITATIVE ENGINE OUTCOME (resolved before narration)\n" + string(payload) + "\nNarrate exactly this degree. Player intent is an attempt, not an established fact. Do not upgrade failure, erase costs/pressure, or add state rewards unsupported by the outcome."
}
