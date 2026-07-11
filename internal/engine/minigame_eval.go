package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/game/contracts"
)

const MiniGameEvalSchemaVersion = 1

var requiredMiniGameEvalConcerns = []string{
	"generosity_bias", "anti_loop", "false_identity", "metamorphosis",
	"zero_combat", "social_comedy", "deterministic_replay", "cross_surface_parity",
}

type MiniGameEvalCorpus struct {
	Version               int                `json:"version"`
	PromptProfileRevision string             `json:"prompt_profile_revision"`
	SchemaVersion         int                `json:"schema_version"`
	Cases                 []MiniGameEvalCase `json:"cases"`
}

type MiniGameEvalCase struct {
	ID             string                  `json:"id"`
	Seed           int64                   `json:"seed"`
	Definition     MiniGameDefinition      `json:"definition"`
	Input          MiniGameInput           `json:"input"`
	ExpectedDegree contracts.OutcomeDegree `json:"expected_degree"`
	Concerns       []string                `json:"concerns"`
}

type MiniGameEvalReport struct {
	Version               int            `json:"version"`
	PromptProfileRevision string         `json:"prompt_profile_revision"`
	SchemaVersion         int            `json:"schema_version"`
	TotalCases            int            `json:"total_cases"`
	PassedCases           int            `json:"passed_cases"`
	SuccessfulOutcomes    int            `json:"successful_outcomes"`
	SuccessRate           float64        `json:"success_rate"`
	ConcernCoverage       map[string]int `json:"concern_coverage"`
	Failures              []string       `json:"failures,omitempty"`
}

func LoadMiniGameEvalCorpus(path string) (*MiniGameEvalCorpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var corpus MiniGameEvalCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		return nil, fmt.Errorf("decoding minigame eval corpus: %w", err)
	}
	if err := corpus.Validate(); err != nil {
		return nil, err
	}
	return &corpus, nil
}

func (corpus MiniGameEvalCorpus) Validate() error {
	if corpus.Version < 1 || corpus.SchemaVersion != MiniGameEvalSchemaVersion {
		return fmt.Errorf("unsupported minigame eval version/schema %d/%d", corpus.Version, corpus.SchemaVersion)
	}
	if strings.TrimSpace(corpus.PromptProfileRevision) == "" || len(corpus.Cases) == 0 {
		return errors.New("minigame eval corpus needs a prompt profile revision and cases")
	}
	seen := map[string]bool{}
	coverage := map[string]int{}
	for _, evalCase := range corpus.Cases {
		if strings.TrimSpace(evalCase.ID) == "" || seen[evalCase.ID] {
			return fmt.Errorf("minigame eval case id %q is empty or duplicated", evalCase.ID)
		}
		seen[evalCase.ID] = true
		if err := validatePackMiniGameDefinition(evalCase.Definition); err != nil {
			return fmt.Errorf("minigame eval case %q: %w", evalCase.ID, err)
		}
		if !contracts.ValidOutcomeDegree(evalCase.ExpectedDegree) {
			return fmt.Errorf("minigame eval case %q has invalid expected degree %q", evalCase.ID, evalCase.ExpectedDegree)
		}
		for _, concern := range evalCase.Concerns {
			coverage[concern]++
		}
	}
	for _, concern := range requiredMiniGameEvalConcerns {
		if coverage[concern] == 0 {
			return fmt.Errorf("minigame eval corpus does not cover %q", concern)
		}
	}
	return nil
}

func RunMiniGameEvalCorpus(corpus MiniGameEvalCorpus) MiniGameEvalReport {
	report := MiniGameEvalReport{
		Version: corpus.Version, PromptProfileRevision: corpus.PromptProfileRevision,
		SchemaVersion: corpus.SchemaVersion, TotalCases: len(corpus.Cases),
		ConcernCoverage: map[string]int{},
	}
	host := NewMiniGameHost()
	for _, evalCase := range corpus.Cases {
		for _, concern := range evalCase.Concerns {
			report.ConcernCoverage[concern]++
		}
		instance := NewMiniGameInstance("eval-"+evalCase.ID, "eval-story", "eval-branch", 1, evalCase.Seed, evalCase.Definition)
		if err := host.Autoplay(&instance, evalCase.Input); err != nil {
			report.Failures = append(report.Failures, fmt.Sprintf("%s: %v", evalCase.ID, err))
			continue
		}
		if instance.Runtime.Result == nil || instance.Runtime.Result.Outcome == nil {
			report.Failures = append(report.Failures, evalCase.ID+": no authoritative outcome")
			continue
		}
		outcome := instance.Runtime.Result.Outcome
		if outcome.Degree != evalCase.ExpectedDegree {
			report.Failures = append(report.Failures, fmt.Sprintf("%s: degree %s, want %s", evalCase.ID, outcome.Degree, evalCase.ExpectedDegree))
			continue
		}
		replay, err := host.Replay(instance)
		if err != nil || replay.Runtime.Result == nil || !reflect.DeepEqual(instance.Runtime.Result, replay.Runtime.Result) {
			report.Failures = append(report.Failures, evalCase.ID+": deterministic replay mismatch")
			continue
		}
		report.PassedCases++
		if outcome.Succeeded() {
			report.SuccessfulOutcomes++
		}
	}
	if report.TotalCases > 0 {
		report.SuccessRate = float64(report.SuccessfulOutcomes) / float64(report.TotalCases)
	}
	if report.SuccessRate < 0.30 || report.SuccessRate > 0.70 {
		report.Failures = append(report.Failures, fmt.Sprintf("success rate %.2f is outside fairness band 0.30..0.70", report.SuccessRate))
	}
	sort.Strings(report.Failures)
	return report
}

func (report MiniGameEvalReport) Passed() bool {
	return report.TotalCases > 0 && report.PassedCases == report.TotalCases && len(report.Failures) == 0
}
