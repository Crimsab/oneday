package engine

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type MiniGameCandidate struct {
	Definition    MiniGameDefinition `json:"definition"`
	NarrativeTags []string           `json:"narrative_tags,omitempty"`
	Reflex        bool               `json:"reflex,omitempty"`
	CooldownTurns int                `json:"cooldown_turns,omitempty"`
}

type MiniGameUsage struct {
	Kind MiniGameType `json:"kind"`
	Turn int          `json:"turn"`
}

type MiniGameSelectionContext struct {
	NarrativeTags  []string        `json:"narrative_tags,omitempty"`
	CurrentTurn    int             `json:"current_turn"`
	Difficulty     int             `json:"difficulty,omitempty"`
	TimingFreeOnly bool            `json:"timing_free_only,omitempty"`
	PreferredKinds []MiniGameType  `json:"preferred_kinds,omitempty"`
	ExcludedKinds  []MiniGameType  `json:"excluded_kinds,omitempty"`
	Recent         []MiniGameUsage `json:"recent,omitempty"`
}

type MiniGameSelection struct {
	Definition MiniGameDefinition `json:"definition"`
	Score      int                `json:"score"`
	Reasons    []string           `json:"reasons"`
}

func DefaultMiniGameCandidates() []MiniGameCandidate {
	return []MiniGameCandidate{
		{DefaultMiniGameDefinition(MiniGameDeduction), []string{"mystery", "investigation", "evidence", "identity"}, false, 4},
		{DefaultMiniGameDefinition(MiniGameNegotiation), []string{"social", "diplomacy", "trade", "conflict"}, false, 3},
		{DefaultMiniGameDefinition(MiniGamePattern), []string{"puzzle", "ritual", "technology", "discovery"}, false, 3},
		{DefaultMiniGameDefinition(MiniGameBidding), []string{"trade", "auction", "resource", "social"}, false, 4},
		{DefaultMiniGameDefinition(MiniGameCourtroom), []string{"courtroom", "trial", "evidence", "social"}, false, 5},
		{DefaultMiniGameDefinition(MiniGameComedy), []string{"comedy", "social", "performance", "zero-combat"}, false, 3},
		{DefaultMiniGameDefinition(MiniGameQuickTime), []string{"chase", "trap", "reflex"}, true, 6},
	}
}

func SelectMiniGame(candidates []MiniGameCandidate, context MiniGameSelectionContext) (*MiniGameSelection, error) {
	if len(candidates) == 0 {
		return nil, errors.New("no minigame candidates are configured")
	}
	if context.Difficulty <= 0 {
		context.Difficulty = 50
	}
	tags := normalizedSet(context.NarrativeTags)
	preferred := miniGameKindSet(context.PreferredKinds)
	excluded := miniGameKindSet(context.ExcludedKinds)
	type scored struct {
		candidate MiniGameCandidate
		score     int
		reasons   []string
	}
	values := make([]scored, 0, len(candidates))
	for _, candidate := range candidates {
		kind := candidate.Definition.Kind
		if excluded[kind] || context.TimingFreeOnly && candidate.Reflex {
			continue
		}
		coolingDown := false
		repetitionPenalty := 0
		for index, usage := range context.Recent {
			if usage.Kind != kind {
				continue
			}
			if context.CurrentTurn-usage.Turn < candidate.CooldownTurns {
				coolingDown = true
				break
			}
			repetitionPenalty += maxSelectorInt(4, 18-index*3)
		}
		if coolingDown {
			continue
		}
		score := 100 - absInt(candidate.Definition.Difficulty-context.Difficulty)
		reasons := []string{fmt.Sprintf("difficulty fit %d", candidate.Definition.Difficulty)}
		matches := 0
		for tag := range normalizedSet(candidate.NarrativeTags) {
			if tags[tag] {
				matches++
			}
		}
		if matches > 0 {
			score += matches * 25
			reasons = append(reasons, fmt.Sprintf("%d narrative tag matches", matches))
		}
		if preferred[kind] {
			score += 30
			reasons = append(reasons, "player preference")
		}
		if repetitionPenalty > 0 {
			score -= repetitionPenalty
			reasons = append(reasons, "recent repetition penalty")
		}
		if !candidate.Reflex {
			reasons = append(reasons, "timing-free")
		}
		values = append(values, scored{candidate, score, reasons})
	}
	if len(values) == 0 {
		if len(excluded) > 0 {
			fallbackContext := context
			fallbackContext.ExcludedKinds = nil
			selection, fallbackErr := SelectMiniGame(candidates, fallbackContext)
			if fallbackErr == nil {
				selection.Reasons = append(selection.Reasons, "disabled-family safety fallback")
				selection.Definition.Rules["selection_reason"] = strings.Join(selection.Reasons, "; ")
				return selection, nil
			}
		}
		return nil, errors.New("no minigame satisfies accessibility and cooldown policy")
	}
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].score == values[j].score {
			return values[i].candidate.Definition.Kind < values[j].candidate.Definition.Kind
		}
		return values[i].score > values[j].score
	})
	selected := values[0]
	definition := selected.candidate.Definition
	if definition.Rules == nil {
		definition.Rules = map[string]string{}
	}
	definition.Rules["selection_reason"] = strings.Join(selected.reasons, "; ")
	return &MiniGameSelection{Definition: definition, Score: selected.score, Reasons: selected.reasons}, nil
}

func normalizedSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		key := normalizeRiddleAnswer(value)
		if key != "" {
			result[key] = true
		}
	}
	return result
}

func miniGameKindSet(values []MiniGameType) map[MiniGameType]bool {
	result := map[MiniGameType]bool{}
	for _, value := range values {
		result[value] = true
	}
	return result
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func maxSelectorInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
