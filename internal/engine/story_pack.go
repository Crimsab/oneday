package engine

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/game/contracts"
	"gopkg.in/yaml.v3"
)

type StoryPack struct {
	ID                 string                                `json:"id" yaml:"id"`
	Name               string                                `json:"name" yaml:"name"`
	Description        string                                `json:"description" yaml:"description"`
	Genre              string                                `json:"genre,omitempty" yaml:"genre,omitempty"`
	StatsSchema        *StatsSchema                          `json:"stats_schema,omitempty" yaml:"stats_schema,omitempty"`
	DifficultyProfiles map[string]StoryPackDifficultyProfile `json:"difficulty_profiles,omitempty" yaml:"difficulty_profiles,omitempty"`
	ChallengePools     map[string]StoryPackChallengePool     `json:"challenge_pools,omitempty" yaml:"challenge_pools,omitempty"`
	OutcomePolicies    map[string]contracts.OutcomePolicy    `json:"outcome_policies,omitempty" yaml:"outcome_policies,omitempty"`
}

type StoryPackDifficultyProfile struct {
	TargetDifficulty  int  `json:"target_difficulty" yaml:"target_difficulty"`
	TimingFreeOnly    bool `json:"timing_free_only,omitempty" yaml:"timing_free_only,omitempty"`
	ConsequenceBudget int  `json:"consequence_budget,omitempty" yaml:"consequence_budget,omitempty"`
}

type StoryPackChallengePool struct {
	NarrativeTags []string             `json:"narrative_tags,omitempty" yaml:"narrative_tags,omitempty"`
	CooldownTurns int                  `json:"cooldown_turns,omitempty" yaml:"cooldown_turns,omitempty"`
	Definitions   []MiniGameDefinition `json:"definitions" yaml:"definitions"`
}

func LoadStoryPack(path string) (*StoryPack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pack StoryPack
	if err := yaml.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("decoding story pack: %w", err)
	}
	if err := pack.Validate(); err != nil {
		return nil, err
	}
	return &pack, nil
}

func (pack StoryPack) Validate() error {
	if strings.TrimSpace(pack.ID) == "" || strings.TrimSpace(pack.Name) == "" || strings.TrimSpace(pack.Description) == "" {
		return errors.New("story pack id, name, and description are required")
	}
	if pack.StatsSchema != nil {
		if err := validateStoryPackStats(*pack.StatsSchema); err != nil {
			return err
		}
	}
	for name, profile := range pack.DifficultyProfiles {
		if strings.TrimSpace(name) == "" || profile.TargetDifficulty < 1 || profile.TargetDifficulty > 100 {
			return fmt.Errorf("difficulty profile %q target must be within 1..100", name)
		}
		if profile.ConsequenceBudget < 0 || profile.ConsequenceBudget > 10 {
			return fmt.Errorf("difficulty profile %q consequence budget must be within 0..10", name)
		}
	}
	definitionIDs := map[string]bool{}
	for name, pool := range pack.ChallengePools {
		if strings.TrimSpace(name) == "" || len(pool.Definitions) == 0 {
			return fmt.Errorf("challenge pool %q needs at least one definition", name)
		}
		for _, definition := range pool.Definitions {
			if err := validatePackMiniGameDefinition(definition); err != nil {
				return fmt.Errorf("challenge pool %q: %w", name, err)
			}
			if definitionIDs[definition.ID] {
				return fmt.Errorf("duplicate minigame definition id %q", definition.ID)
			}
			definitionIDs[definition.ID] = true
		}
	}
	for name, policy := range pack.OutcomePolicies {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(policy.ID) == "" {
			return fmt.Errorf("outcome policy %q needs an id", name)
		}
		if policy.ConsequenceBudget < 0 || policy.ConsequenceBudget > 10 || policy.CriticalBand < 0 || policy.CriticalBand > 20 {
			return fmt.Errorf("outcome policy %q has invalid budget or critical band", name)
		}
	}
	return nil
}

func (pack StoryPack) Candidates(poolName string) ([]MiniGameCandidate, error) {
	pool, ok := pack.ChallengePools[poolName]
	if !ok {
		return nil, fmt.Errorf("story pack challenge pool %q not found", poolName)
	}
	candidates := make([]MiniGameCandidate, 0, len(pool.Definitions))
	for _, definition := range pool.Definitions {
		candidates = append(candidates, MiniGameCandidate{
			Definition: definition, NarrativeTags: append([]string(nil), pool.NarrativeTags...),
			Reflex: definition.Kind == MiniGameQuickTime, CooldownTurns: pool.CooldownTurns,
		})
	}
	return candidates, nil
}

func (pack StoryPack) Select(poolName, profileName string, context MiniGameSelectionContext) (*MiniGameSelection, error) {
	candidates, err := pack.Candidates(poolName)
	if err != nil {
		return nil, err
	}
	if profile, ok := pack.DifficultyProfiles[profileName]; ok {
		context.Difficulty = profile.TargetDifficulty
		context.TimingFreeOnly = context.TimingFreeOnly || profile.TimingFreeOnly
	}
	return SelectMiniGame(candidates, context)
}

func validateStoryPackStats(schema StatsSchema) error {
	seen := map[string]bool{}
	for _, group := range [][]StatDef{schema.Vitals, schema.Attributes, schema.Secondary} {
		for _, stat := range group {
			key := strings.TrimSpace(stat.Key)
			if key == "" || strings.TrimSpace(stat.Label) == "" {
				return errors.New("story pack stats require key and label")
			}
			if seen[key] {
				return fmt.Errorf("duplicate story pack stat key %q", key)
			}
			seen[key] = true
		}
	}
	return nil
}

func validatePackMiniGameDefinition(definition MiniGameDefinition) error {
	if strings.TrimSpace(definition.ID) == "" || !knownMiniGameKind(definition.Kind) {
		return fmt.Errorf("minigame definition needs a unique id and registered kind, got %q/%q", definition.ID, definition.Kind)
	}
	if definition.Difficulty < 1 || definition.Difficulty > 100 {
		return fmt.Errorf("minigame %q difficulty must be within 1..100", definition.ID)
	}
	if (definition.Kind == MiniGameRiddle || definition.Kind == MiniGameDeduction || definition.Kind == MiniGamePattern) && len(definition.Answers) == 0 {
		return fmt.Errorf("minigame %q requires explicit accepted answers", definition.ID)
	}
	return nil
}

func knownMiniGameKind(kind MiniGameType) bool {
	kinds := []MiniGameType{MiniGameRPS, MiniGameMemory, MiniGameQuickTime, MiniGameRiddle, MiniGameDeduction, MiniGameNegotiation, MiniGamePattern, MiniGameBidding, MiniGameCourtroom, MiniGameComedy}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	index := sort.Search(len(kinds), func(i int) bool { return kinds[i] >= kind })
	return index < len(kinds) && kinds[index] == kind
}
