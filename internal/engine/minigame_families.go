package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/crimsab/oneday/internal/game/contracts"
)

type genreNeutralMiniGameState struct {
	RequiredEvidence int `json:"required_evidence,omitempty"`
	MarketValue      int `json:"market_value,omitempty"`
	Reserve          int `json:"reserve,omitempty"`
	Budget           int `json:"budget,omitempty"`
}

type genreNeutralMiniGameReducer struct{}

func DefaultMiniGameDefinition(kind MiniGameType) MiniGameDefinition {
	switch kind {
	case MiniGameDeduction:
		return MiniGameDefinition{ID: "deduction-generic", Kind: kind, Prompt: "Choose the conclusion that best fits the evidence, then name the evidence you relied on.", Difficulty: 50, Options: []string{"system fault", "outside interference", "coincidence"}, Answers: []string{"system fault"}, Rules: map[string]string{"required_evidence": "2"}}
	case MiniGameNegotiation:
		return MiniGameDefinition{ID: "negotiation-generic", Kind: kind, Prompt: "Choose an approach and commit the leverage you are willing to reveal.", Difficulty: 50, Options: []string{"cooperative", "assertive", "deceptive"}}
	case MiniGamePattern:
		return MiniGameDefinition{ID: "pattern-generic", Kind: kind, Prompt: "Complete the pattern: 2, 4, 6, ?", Difficulty: 50, Options: []string{"8", "9", "10"}, Sequence: []string{"2", "4", "6"}, Answers: []string{"8"}}
	case MiniGameBidding:
		return MiniGameDefinition{ID: "bidding-generic", Kind: kind, Prompt: "Make an integer offer without exceeding your budget.", Difficulty: 50, Rules: map[string]string{"reserve": "40", "market_value": "60", "budget": "90"}}
	default:
		return MiniGameDefinition{ID: string(kind), Kind: kind, Difficulty: 50}
	}
}

func (genreNeutralMiniGameReducer) Initialize(definition MiniGameDefinition, _ int64) (json.RawMessage, error) {
	state := genreNeutralMiniGameState{}
	switch definition.Kind {
	case MiniGameDeduction, MiniGamePattern:
		if len(definition.Answers) == 0 {
			return nil, fmt.Errorf("%s minigame requires an explicit answer", definition.Kind)
		}
		state.RequiredEvidence = ruleInt(definition.Rules, "required_evidence", 2)
	case MiniGameNegotiation:
		if len(definition.Options) == 0 {
			definition.Options = []string{"cooperative", "assertive", "deceptive"}
		}
	case MiniGameBidding:
		state.MarketValue = ruleInt(definition.Rules, "market_value", 60)
		state.Reserve = ruleInt(definition.Rules, "reserve", state.MarketValue*3/4)
		state.Budget = ruleInt(definition.Rules, "budget", state.MarketValue*3/2)
		if state.Reserve <= 0 || state.MarketValue < state.Reserve || state.Budget < state.MarketValue {
			return nil, errors.New("bidding rules require 0 < reserve <= market_value <= budget")
		}
	default:
		return nil, fmt.Errorf("genre-neutral reducer cannot initialize %q", definition.Kind)
	}
	return json.Marshal(state)
}

func (genreNeutralMiniGameReducer) Reduce(definition MiniGameDefinition, seed int64, payload json.RawMessage, input MiniGameInput) (json.RawMessage, *ChallengeResult, error) {
	if input.Action != "submit" {
		return payload, nil, fmt.Errorf("minigame action %q is not supported", input.Action)
	}
	var state genreNeutralMiniGameState
	if err := json.Unmarshal(payload, &state); err != nil {
		return payload, nil, fmt.Errorf("decoding %s state: %w", definition.Kind, err)
	}
	var result *ChallengeResult
	switch definition.Kind {
	case MiniGameDeduction:
		correct := matchesAcceptedAnswer(input.Value, definition.Answers)
		evidence := uniqueNonEmpty(input.Values)
		degree := contracts.OutcomeHardFailure
		switch {
		case correct && len(evidence) >= state.RequiredEvidence:
			degree = contracts.OutcomeFullSuccess
		case correct:
			degree = contracts.OutcomeSuccessWithCost
		case len(evidence) >= state.RequiredEvidence:
			degree = contracts.OutcomeFailureWithProgress
		}
		result = miniGameResult(degree, definition.Difficulty, len(evidence), state.RequiredEvidence,
			fmt.Sprintf("Deduction: %d evidence links, conclusion %q", len(evidence), input.Value))
	case MiniGamePattern:
		degree := contracts.OutcomeHardFailure
		if matchesAcceptedAnswer(input.Value, definition.Answers) {
			degree = contracts.OutcomeFullSuccess
		} else if strings.TrimSpace(input.Value) != "" {
			degree = contracts.OutcomeFailureWithProgress
		}
		result = miniGameResult(degree, definition.Difficulty, 0, 0, fmt.Sprintf("Pattern answer: %q", input.Value))
	case MiniGameNegotiation:
		approach := normalizeRiddleAnswer(input.Value)
		if approach == "" {
			return payload, nil, errors.New("negotiation approach is required")
		}
		roll := NewRNGService(seed).Roll("minigame.negotiation", 21).Raw - 11
		total := 50 + len(uniqueNonEmpty(input.Values))*10 + roll - definition.Difficulty
		switch approach {
		case "cooperative":
			total += 5
		case "deceptive":
			total -= 5
		}
		degree := degreeForScore(total)
		result = miniGameResult(degree, definition.Difficulty, total, 0,
			fmt.Sprintf("Negotiation: %s approach with %d leverage", approach, len(uniqueNonEmpty(input.Values))))
		result.Roll = roll
		result.Outcome.Roll = roll
		result.Outcome.Seed = seed
	case MiniGameBidding:
		bid, err := strconv.Atoi(strings.TrimSpace(input.Value))
		if err != nil || bid <= 0 {
			return payload, nil, errors.New("bid must be a positive integer")
		}
		degree := contracts.OutcomeHardFailure
		switch {
		case bid > state.Budget:
			degree = contracts.OutcomeCatastrophe
		case bid > state.MarketValue:
			degree = contracts.OutcomeSuccessWithCost
		case bid >= state.Reserve:
			degree = contracts.OutcomeFullSuccess
		default:
			degree = contracts.OutcomeFailureWithProgress
		}
		result = miniGameResult(degree, definition.Difficulty, bid, state.Reserve,
			fmt.Sprintf("Bidding: offer %d (reserve %d, value %d, budget %d)", bid, state.Reserve, state.MarketValue, state.Budget))
	default:
		return payload, nil, fmt.Errorf("genre-neutral reducer cannot resolve %q", definition.Kind)
	}
	return payload, result, nil
}

func miniGameResult(degree contracts.OutcomeDegree, difficulty, total, target int, detail string) *ChallengeResult {
	outcome := contracts.OutcomeEnvelope{
		Version: contracts.ChallengeProtocolVersion, Degree: degree, Difficulty: difficulty,
		Total: total, Margin: total - target,
	}
	applyDefaultOutcomeBudget(&outcome, DefaultOutcomePolicy("", "balanced"))
	return &ChallengeResult{Passed: outcome.Succeeded(), Total: total, Difficulty: difficulty, Detail: detail + " → " + strings.ToUpper(strings.ReplaceAll(string(degree), "_", " ")), Outcome: &outcome}
}

func degreeForScore(score int) contracts.OutcomeDegree {
	switch {
	case score >= 20:
		return contracts.OutcomeFullSuccess
	case score >= 5:
		return contracts.OutcomeSuccessWithCost
	case score >= -15:
		return contracts.OutcomeFailureWithProgress
	default:
		return contracts.OutcomeHardFailure
	}
}

func matchesAcceptedAnswer(value string, answers []string) bool {
	normalized := normalizeRiddleAnswer(value)
	if normalized == "" {
		return false
	}
	for _, answer := range answers {
		if normalized == normalizeRiddleAnswer(answer) {
			return true
		}
	}
	return false
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		key := normalizeRiddleAnswer(value)
		if key != "" && !seen[key] {
			seen[key] = true
			result = append(result, value)
		}
	}
	return result
}

func ruleInt(rules map[string]string, key string, fallback int) int {
	if rules == nil {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(rules[key]))
	if err != nil {
		return fallback
	}
	return value
}
