package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
)

type seededRepairProvider struct {
	invalid    string
	callCount  int
	repairPath *ai.Router
}

func (p *seededRepairProvider) Name() string { return "seeded-repair-provider" }

func (p *seededRepairProvider) Complete(ctx context.Context, req ai.Request) (ai.Response, error) {
	if p.callCount == 0 {
		p.callCount++
		return ai.Response{
			Content:   p.invalid,
			Model:     "seed-invalid",
			Provider:  "seeded-repair-provider",
			LatencyMs: 1,
		}, nil
	}
	p.callCount++
	return p.repairPath.Complete(ctx, req)
}

type repairCase struct {
	Name    string `json:"name"`
	Invalid string `json:"-"`
}

type caseResult struct {
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	DurationMS    int64   `json:"duration_ms"`
	DurationSec   float64 `json:"duration_seconds"`
	ResolvedModel string  `json:"resolved_model,omitempty"`
	StoryName     string  `json:"story_name,omitempty"`
	WorldName     string  `json:"world_name,omitempty"`
	Error         string  `json:"error,omitempty"`
}

type modelSummary struct {
	Model            string       `json:"model"`
	SuccessCount     int          `json:"success_count"`
	CaseCount        int          `json:"case_count"`
	SuccessRate      float64      `json:"success_rate"`
	AverageLatencyMS float64      `json:"average_latency_ms"`
	AverageLatencyS  float64      `json:"average_latency_seconds"`
	Results          []caseResult `json:"results"`
}

type report struct {
	GeneratedAt string         `json:"generated_at"`
	Brief       string         `json:"brief"`
	Models      []string       `json:"models"`
	Cases       []repairCase   `json:"cases"`
	Summaries   []modelSummary `json:"summaries"`
}

func main() {
	defaultModels := "grok-4.1-fast,main-fast,gemini-2.5-flash-lite,gemini-3.1-flash-lite-preview,qwen3.6-plus"
	defaultBrief := "Mondo steampunk in tono serio e tenebroso, città industriale decadente, culto del vapore, dialoghi tesi, prosa elegante ma non prolissa. Lingua italiana. Voglio una campagna lunga, politica, investigativa e pericolosa."

	configPath := flag.String("config", "config.yaml", "Path to config.yaml")
	modelsFlag := flag.String("models", defaultModels, "Comma-separated repair model aliases")
	outputDir := flag.String("output-dir", "docs/benchmarks/runs", "Directory for benchmark reports")
	brief := flag.String("brief", defaultBrief, "Story brief used for schema-repair runs")
	timeout := flag.Duration("timeout", 120*time.Second, "Per-case timeout")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if secs := int(timeout.Seconds()); secs > 0 {
		cfg.AI.Generation.TimeoutSeconds = secs
	}

	realRouter, err := aifactory.NewRouterFromConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "router: %v\n", err)
		os.Exit(1)
	}

	cases := []repairCase{
		{
			Name: "missing_authoring_fields",
			Invalid: `{
  "name": "Le Ciminiere di Nerofumo",
  "description": "Una capitale di ottone e fuliggine sospesa tra culto e repressione.",
  "setting": {
    "world_name": "Nerofumo",
    "era": "Secolo delle Caldaie",
    "geography": "Canali tossici e quartieri-fabbrica",
    "magic_system": "Liturgie del vapore",
    "technology_level": "Macchine a pressione e automi rituali",
    "society": "Gilde industriali, clero del vapore e polizia segreta",
    "rules": ["Il vapore consacrato alimenta la città", "Ogni inquisizione lascia un marchio"],
    "factions": ["Conclave delle Caldaie", "Ispettorato Fuliggine"],
    "cultures": ["Operai dei canali", "Nobiltà delle turbine"],
    "dangers": ["Blackout rituali", "Sparizioni nelle condotte", "Sommosse di automi"]
  },
  "stats_schema": {
    "vitals": [{"key":"hp","label":"Salute","starting":10}],
    "attributes": [{"key":"wit","label":"Acume","starting":3}],
    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
    "currency": {"name":"Corone di Carbone","starting":8},
    "has_combat": true
  }
}`,
		},
		{
			Name: "missing_world_rules_and_stats",
			Invalid: `{
  "name": "Le Ciminiere di Nerofumo",
  "description": "Una capitale di ottone e fuliggine sospesa tra culto e repressione.",
  "genre": "steampunk investigativo",
  "tone": "serio e tenebroso",
  "language": "italiano",
  "writing_style": "prosa elegante ma non prolissa",
  "prompt_directives": "keep dialogue sharp",
  "setting": {
    "era": "Secolo delle Caldaie",
    "geography": "Canali tossici e quartieri-fabbrica",
    "magic_system": "Liturgie del vapore",
    "technology_level": "Macchine a pressione e automi rituali",
    "society": "Gilde industriali, clero del vapore e polizia segreta",
    "factions": ["Conclave delle Caldaie", "Ispettorato Fuliggine"],
    "cultures": ["Operai dei canali", "Nobiltà delle turbine"],
    "dangers": ["Blackout rituali", "Sparizioni nelle condotte", "Sommosse di automi"]
  },
  "stats_schema": {
    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
    "currency": {"name":"Corone di Carbone","starting":8},
    "has_combat": true
  }
}`,
		},
		{
			Name: "wrong_shapes",
			Invalid: `{
  "name": "Le Ciminiere di Nerofumo",
  "description": "Una capitale di ottone e fuliggine sospesa tra culto e repressione.",
  "genre": "steampunk investigativo",
  "tone": "serio e tenebroso",
  "language": "italiano",
  "writing_style": "prosa elegante ma non prolissa",
  "prompt_directives": "keep dialogue sharp",
  "setting": {
    "world_name": "Nerofumo",
    "era": "Secolo delle Caldaie",
    "geography": "Canali tossici e quartieri-fabbrica",
    "magic_system": "Liturgie del vapore",
    "technology_level": "Macchine a pressione e automi rituali",
    "society": "Gilde industriali, clero del vapore e polizia segreta",
    "rules": {"r1":"Il vapore consacrato alimenta la città"},
    "factions": {"Conclave delle Caldaie":{"description":"oligarchi del vapore"}},
    "cultures": {"Operai dei canali":"resistenti e superstiziosi"},
    "dangers": {"Blackout rituali":"aprono varchi e panico"}
  },
  "stats_schema": {
    "vitals": {"hp":{"label":"Salute","starting":10}},
    "attributes": {"wit":{"label":"Acume","starting":3}},
    "secondary": {"rep":{"label":"Reputazione","starting":0}},
    "currency": [{"name":"Corone di Carbone","starting":8}],
    "has_combat": true
  }
}`,
		},
		{
			Name: "not_json",
			Invalid: `I think the setting should be darker and more political, with steam priests and rail inspectors.`,
		},
	}

	models := splitCSV(*modelsFlag)
	rep := report{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Brief:       *brief,
		Models:      models,
		Cases:       cases,
	}

	for _, model := range models {
		fmt.Printf("== REPAIR MODEL: %s ==\n", model)
		rep.Summaries = append(rep.Summaries, runModel(&cfg, realRouter, model, *brief, cases, *timeout))
		fmt.Println()
	}

	sort.Slice(rep.Summaries, func(i, j int) bool {
		if rep.Summaries[i].SuccessRate == rep.Summaries[j].SuccessRate {
			return rep.Summaries[i].AverageLatencyMS < rep.Summaries[j].AverageLatencyMS
		}
		return rep.Summaries[i].SuccessRate > rep.Summaries[j].SuccessRate
	})

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir output dir: %v\n", err)
		os.Exit(1)
	}
	stamp := time.Now().Format("2006-01-02-150405")
	base := filepath.Join(*outputDir, stamp+"-oneday-schema-reliability")
	if err := writeJSON(base+".json", rep); err != nil {
		fmt.Fprintf(os.Stderr, "write json: %v\n", err)
		os.Exit(1)
	}
	if err := writeMarkdown(base+".md", rep); err != nil {
		fmt.Fprintf(os.Stderr, "write markdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("json report: %s\n", base+".json")
	fmt.Printf("markdown report: %s\n", base+".md")
}

func runModel(cfg *config.Config, realRouter *ai.Router, model, brief string, cases []repairCase, timeout time.Duration) modelSummary {
	summary := modelSummary{
		Model:     model,
		CaseCount: len(cases),
	}

	var totalLatency int64
	var successCount int
	for _, tc := range cases {
		result := runCase(cfg, realRouter, model, brief, tc, timeout)
		summary.Results = append(summary.Results, result)
		if result.Success {
			successCount++
			totalLatency += result.DurationMS
			fmt.Printf("- %s: PASS in %.3fs via %s | story=%q world=%q\n", tc.Name, result.DurationSec, result.ResolvedModel, result.StoryName, result.WorldName)
		} else {
			fmt.Printf("- %s: FAIL in %.3fs (%s)\n", tc.Name, result.DurationSec, result.Error)
		}
	}

	summary.SuccessCount = successCount
	if summary.CaseCount > 0 {
		summary.SuccessRate = float64(successCount) / float64(summary.CaseCount) * 100
	}
	if successCount > 0 {
		summary.AverageLatencyMS = float64(totalLatency) / float64(successCount)
		summary.AverageLatencyS = summary.AverageLatencyMS / 1000
	}
	return summary
}

func runCase(cfg *config.Config, realRouter *ai.Router, model, brief string, tc repairCase, timeout time.Duration) caseResult {
	tempDir, err := os.MkdirTemp("", "oneday-schema-bench-*")
	if err != nil {
		return caseResult{Name: tc.Name, Error: err.Error()}
	}
	defer os.RemoveAll(tempDir)

	db, err := storage.Open(filepath.Join(tempDir, "bench.db"))
	if err != nil {
		return caseResult{Name: tc.Name, Error: err.Error()}
	}
	defer db.Close()

	provider := &seededRepairProvider{
		invalid:    tc.Invalid,
		repairPath: realRouter,
	}
	router, err := ai.NewRouter([]ai.Provider{provider})
	if err != nil {
		return caseResult{Name: tc.Name, Error: err.Error()}
	}

	genCfg := cfg.AI.Generation
	genCfg.RepairModel = model
	creator := engine.NewStoryCreator(router, db, genCfg)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	_, err = creator.SendMessage(ctx, brief)
	elapsed := time.Since(start)

	result := caseResult{
		Name:        tc.Name,
		DurationMS:  elapsed.Milliseconds(),
		DurationSec: elapsed.Seconds(),
	}
	if err != nil {
		result.Error = err.Error()
		return result
	}

	def := creator.Definition()
	result.Success = true
	result.ResolvedModel = creator.LastModel()
	if def != nil {
		result.StoryName = def.Name
		result.WorldName = def.Setting.WorldName
	}
	return result
}

func writeJSON(path string, rep report) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeMarkdown(path string, rep report) error {
	var sb strings.Builder
	sb.WriteString("# OneDay Schema Reliability Benchmark\n\n")
	sb.WriteString(fmt.Sprintf("- Generated at: `%s`\n", rep.GeneratedAt))
	sb.WriteString(fmt.Sprintf("- Brief: `%s`\n", rep.Brief))
	sb.WriteString(fmt.Sprintf("- Cases: `%d`\n\n", len(rep.Cases)))

	sb.WriteString("## Ranking\n\n")
	sb.WriteString("| Model | Success Rate | Avg Latency |\n")
	sb.WriteString("| --- | ---: | ---: |\n")
	for _, summary := range rep.Summaries {
		sb.WriteString(fmt.Sprintf("| `%s` | `%.1f%%` | `%.3fs` |\n", summary.Model, summary.SuccessRate, summary.AverageLatencyS))
	}
	sb.WriteString("\n")

	for _, summary := range rep.Summaries {
		sb.WriteString(fmt.Sprintf("## %s\n\n", summary.Model))
		sb.WriteString(fmt.Sprintf("- Success rate: `%.1f%%`\n", summary.SuccessRate))
		sb.WriteString(fmt.Sprintf("- Average latency: `%.3fs`\n\n", summary.AverageLatencyS))
		sb.WriteString("| Case | Result | Time | Resolved Model | Notes |\n")
		sb.WriteString("| --- | --- | ---: | --- | --- |\n")
		for _, result := range summary.Results {
			outcome := "FAIL"
			if result.Success {
				outcome = "PASS"
			}
			notes := result.Error
			if notes == "" {
				notes = fmt.Sprintf("story=%q world=%q", result.StoryName, result.WorldName)
			}
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | `%.3fs` | `%s` | %s |\n",
				result.Name, outcome, result.DurationSec, emptyDash(result.ResolvedModel), escapeTable(notes)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Cases\n\n")
	for _, tc := range rep.Cases {
		sb.WriteString(fmt.Sprintf("- `%s`\n", tc.Name))
	}
	sb.WriteString("\n")

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}
