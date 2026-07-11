package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/buildinfo"
)

type asciiBenchmarkCase struct {
	ID          string
	Title       string
	Category    string
	Weight      float64
	Temperature float64
	MaxTokens   int
	Kind        string
	Subject     string
	Detail      string
	Placement   string
	Mood        string
	Location    string
	SceneType   string
	Narrative   string
	Eval        func(string) asciiCaseEvaluation
}

type asciiBenchmarkMode string

const (
	asciiModePrompt     asciiBenchmarkMode = "prompt"
	asciiModeJSONObject asciiBenchmarkMode = "json_object"
	asciiModeJSONSchema asciiBenchmarkMode = "json_schema"
	asciiModeAll        asciiBenchmarkMode = "all"
)

type asciiCaseEvaluation struct {
	Score      float64            `json:"score"`
	Breakdown  map[string]float64 `json:"breakdown"`
	Notes      []string           `json:"notes"`
	RawJSON    string             `json:"raw_json,omitempty"`
	JSONValid  bool               `json:"json_valid"`
	JSONOnly   bool               `json:"json_only"`
	Parsed     bool               `json:"parsed"`
	LineCount  int                `json:"line_count,omitempty"`
	MaxWidth   int                `json:"max_width,omitempty"`
	ASCIIOnly  bool               `json:"ascii_only"`
	HasDrawing bool               `json:"has_drawing"`
}

type asciiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type asciiAPIRequest struct {
	Model          string             `json:"model"`
	Messages       []ai.Message       `json:"messages"`
	Temperature    float64            `json:"temperature,omitempty"`
	MaxTokens      int                `json:"max_tokens,omitempty"`
	ResponseFormat *ai.ResponseFormat `json:"response_format,omitempty"`
}

type asciiAPIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string     `json:"model"`
	Usage asciiUsage `json:"usage"`
}

type asciiModelCatalogResponse struct {
	Data []asciiModelCatalogEntry `json:"data"`
}

type asciiModelCatalogEntry struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	ContextLength       int               `json:"context_length"`
	Pricing             map[string]string `json:"pricing"`
	SupportedParameters []string          `json:"supported_parameters"`
	TopProvider         struct {
		MaxCompletionTokens int `json:"max_completion_tokens"`
	} `json:"top_provider"`
}

type asciiModelMetadata struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	ContextLength          int      `json:"context_length"`
	MaxCompletionTokens    int      `json:"max_completion_tokens"`
	PromptCostPerTokenUSD  float64  `json:"prompt_cost_per_token_usd"`
	OutputCostPerTokenUSD  float64  `json:"output_cost_per_token_usd"`
	SupportsResponseFormat bool     `json:"supports_response_format"`
	SupportsStructured     bool     `json:"supports_structured_outputs"`
	SupportedParameters    []string `json:"supported_parameters,omitempty"`
}

type asciiCaseResult struct {
	CaseID           string              `json:"case_id"`
	CaseTitle        string              `json:"case_title"`
	Category         string              `json:"category"`
	Weight           float64             `json:"weight"`
	ModelRequested   string              `json:"model_requested"`
	ModelResolved    string              `json:"model_resolved,omitempty"`
	DurationMS       int64               `json:"duration_ms"`
	DurationSeconds  float64             `json:"duration_seconds"`
	CompletionTokens int                 `json:"completion_tokens"`
	PromptTokens     int                 `json:"prompt_tokens"`
	TotalTokens      int                 `json:"total_tokens"`
	TokensPerSecond  float64             `json:"tokens_per_second"`
	CharsPerSecond   float64             `json:"chars_per_second"`
	CostUSD          float64             `json:"cost_usd"`
	OutputChars      int                 `json:"output_chars"`
	OutputWords      int                 `json:"output_words"`
	Evaluation       asciiCaseEvaluation `json:"evaluation"`
	RawResponse      string              `json:"raw_response,omitempty"`
	Error            string              `json:"error,omitempty"`
}

type asciiModelSummary struct {
	Model               string             `json:"model"`
	Metadata            asciiModelMetadata `json:"metadata"`
	WeightedScore       float64            `json:"weighted_score"`
	SuccessRate         float64            `json:"success_rate"`
	AverageLatencyMS    float64            `json:"average_latency_ms"`
	AverageLatencySec   float64            `json:"average_latency_seconds"`
	AverageTokensPerSec float64            `json:"average_tokens_per_second"`
	AverageCharsPerSec  float64            `json:"average_chars_per_second"`
	AverageCostUSD      float64            `json:"average_cost_usd"`
	TotalCostUSD        float64            `json:"total_cost_usd"`
	Results             []asciiCaseResult  `json:"results"`
}

type asciiBenchmarkReport struct {
	GeneratedAt string              `json:"generated_at"`
	BaseURL     string              `json:"base_url"`
	Mode        asciiBenchmarkMode  `json:"mode"`
	Models      []string            `json:"models"`
	Summaries   []asciiModelSummary `json:"summaries"`
}

type asciiClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func main() {
	baseURLFlag := flag.String("base-url", firstNonEmpty(
		os.Getenv("ONEDAY_ASCII_BENCH_BASE_URL"),
		os.Getenv("OPENAI_BASE_URL"),
		"https://openrouter.ai/api/v1",
	), "OpenAI-compatible base URL ending in /v1")
	apiKeyFlag := flag.String("api-key", firstNonEmpty(
		os.Getenv("ONEDAY_ASCII_BENCH_API_KEY"),
		os.Getenv("OPENROUTER_API_KEY"),
		os.Getenv("OPENAI_API_KEY"),
	), "API key for the benchmark endpoint")
	modelsFlag := flag.String("models", os.Getenv("ONEDAY_ASCII_BENCH_MODELS"), "Comma-separated model list; required unless ONEDAY_ASCII_BENCH_MODELS is set")
	outputDirFlag := flag.String("output-dir", firstNonEmpty(os.Getenv("ONEDAY_ASCII_BENCH_OUTPUT_DIR"), "docs/benchmarks/runs"), "Directory for JSON and Markdown reports")
	timeoutFlag := flag.Duration("timeout", 90*time.Second, "Per-request timeout")
	modeFlag := flag.String("mode", string(asciiModeJSONSchema), "Benchmark mode: prompt, json_object, json_schema, all")
	versionFlag := flag.Bool("version", false, "Print build information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(buildinfo.Text("oneday-ascii-benchmark"))
		return
	}

	if strings.TrimSpace(*apiKeyFlag) == "" {
		fmt.Fprintln(os.Stderr, "missing API key: pass -api-key or export ONEDAY_ASCII_BENCH_API_KEY / OPENROUTER_API_KEY")
		os.Exit(1)
	}

	models := splitCSV(*modelsFlag)
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "missing models list")
		os.Exit(1)
	}

	cli := &asciiClient{
		baseURL: strings.TrimRight(*baseURLFlag, "/"),
		apiKey:  strings.TrimSpace(*apiKeyFlag),
		client:  &http.Client{Timeout: *timeoutFlag},
	}

	cases := buildASCIIBenchmarkCases()
	modelMeta := cli.fetchModelMetadata(models)

	mode := asciiBenchmarkMode(strings.TrimSpace(*modeFlag))
	var modes []asciiBenchmarkMode
	switch mode {
	case asciiModePrompt, asciiModeJSONObject, asciiModeJSONSchema:
		modes = []asciiBenchmarkMode{mode}
	case asciiModeAll:
		modes = []asciiBenchmarkMode{asciiModePrompt, asciiModeJSONObject, asciiModeJSONSchema}
	default:
		fmt.Fprintf(os.Stderr, "unsupported mode %q (expected prompt, json_object, json_schema, all)\n", *modeFlag)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outputDirFlag, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "creating output dir: %v\n", err)
		os.Exit(1)
	}

	for _, benchMode := range modes {
		report := asciiBenchmarkReport{
			GeneratedAt: time.Now().Format(time.RFC3339),
			BaseURL:     cli.baseURL,
			Mode:        benchMode,
			Models:      models,
		}

		for _, model := range models {
			fmt.Fprintf(os.Stdout, "benchmarking %s (%s)\n", model, benchMode)
			summary := runASCIIBenchmarks(cli, model, modelMeta[model], cases, benchMode)
			report.Summaries = append(report.Summaries, summary)
		}

		sort.Slice(report.Summaries, func(i, j int) bool {
			return report.Summaries[i].WeightedScore > report.Summaries[j].WeightedScore
		})

		stamp := time.Now().Format("2006-01-02-150405")
		baseName := fmt.Sprintf("%s-oneday-ascii-benchmark-%s", stamp, benchMode)
		jsonPath := filepath.Join(*outputDirFlag, baseName+".json")
		mdPath := filepath.Join(*outputDirFlag, baseName+".md")

		if err := writeASCIIJSONReport(jsonPath, report); err != nil {
			fmt.Fprintf(os.Stderr, "writing json report: %v\n", err)
			os.Exit(1)
		}
		if err := writeASCIIMarkdownReport(mdPath, filepath.Base(jsonPath), report); err != nil {
			fmt.Fprintf(os.Stderr, "writing markdown report: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stdout, "json report: %s\n", jsonPath)
		fmt.Fprintf(os.Stdout, "markdown report: %s\n", mdPath)
	}
}

func runASCIIBenchmarks(cli *asciiClient, model string, meta asciiModelMetadata, cases []asciiBenchmarkCase, mode asciiBenchmarkMode) asciiModelSummary {
	var results []asciiCaseResult
	var weightedScore float64
	var totalWeight float64
	var successCount int
	var latencyTotal float64
	var tpsTotal float64
	var cpsTotal float64
	var totalCost float64
	var perfCount int

	for _, bc := range cases {
		fmt.Fprintf(os.Stdout, "  - %s\n", bc.ID)
		result := asciiCaseResult{
			CaseID:         bc.ID,
			CaseTitle:      bc.Title,
			Category:       bc.Category,
			Weight:         bc.Weight,
			ModelRequested: model,
		}

		ctx, cancel := context.WithTimeout(context.Background(), cli.client.Timeout)
		raw, resolvedModel, usage, duration, err := cli.complete(ctx, model, bc, mode, meta)
		cancel()

		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			totalWeight += bc.Weight
			continue
		}

		result.ModelResolved = resolvedModel
		result.RawResponse = raw
		result.DurationMS = duration.Milliseconds()
		result.DurationSeconds = duration.Seconds()
		result.CompletionTokens = usage.CompletionTokens
		result.PromptTokens = usage.PromptTokens
		result.TotalTokens = usage.TotalTokens
		result.CostUSD = estimateASCIICostUSD(meta, usage)
		result.OutputChars = len(raw)
		result.OutputWords = wordCount(raw)
		if duration > 0 {
			seconds := duration.Seconds()
			result.CharsPerSecond = float64(result.OutputChars) / seconds
			if usage.CompletionTokens > 0 {
				result.TokensPerSecond = float64(usage.CompletionTokens) / seconds
			}
		}

		result.Evaluation = bc.Eval(raw)
		results = append(results, result)

		totalWeight += bc.Weight
		weightedScore += result.Evaluation.Score * bc.Weight
		successCount++
		latencyTotal += float64(result.DurationMS)
		totalCost += result.CostUSD
		if result.TokensPerSecond > 0 {
			tpsTotal += result.TokensPerSecond
		}
		if result.CharsPerSecond > 0 {
			cpsTotal += result.CharsPerSecond
		}
		perfCount++
	}

	summary := asciiModelSummary{
		Model:    model,
		Metadata: meta,
		Results:  results,
	}
	if totalWeight > 0 {
		summary.WeightedScore = weightedScore / totalWeight
	}
	if len(cases) > 0 {
		summary.SuccessRate = float64(successCount) / float64(len(cases)) * 100
	}
	if perfCount > 0 {
		summary.AverageLatencyMS = latencyTotal / float64(perfCount)
		summary.AverageLatencySec = summary.AverageLatencyMS / 1000
		summary.AverageTokensPerSec = tpsTotal / float64(perfCount)
		summary.AverageCharsPerSec = cpsTotal / float64(perfCount)
		summary.AverageCostUSD = totalCost / float64(perfCount)
	}
	summary.TotalCostUSD = totalCost
	return summary
}

func buildASCIIBenchmarkCases() []asciiBenchmarkCase {
	return []asciiBenchmarkCase{
		buildASCIILocationCase(),
		buildASCIISignageCase(),
		buildASCIITerminalCase(),
		buildASCIIRitualCase(),
		buildASCIIMapCase(),
		buildASCIIArtifactCase(),
	}
}

func buildASCIILocationCase() asciiBenchmarkCase {
	return asciiBenchmarkCase{
		ID:          "location-reveal",
		Title:       "Major location reveal",
		Category:    "location",
		Weight:      0.20,
		Temperature: 0.4,
		MaxTokens:   400,
		Kind:        "location",
		Subject:     "Podara Neon skyline over the Tallone d'Oro district",
		Detail:      "floating arcologies, devotional foot shrines, layered neon arches, elevated walkways",
		Placement:   "scene_header",
		Mood:        "decadent, electric, cultish",
		Location:    "Podara Neon",
		SceneType:   "chapter_opener",
		Narrative:   "Arrivi a Podara Neon per la prima volta: un canyon urbano di arcologie e santuari podali, con ponti sospesi e vetrate al neon che tremano nella foschia industriale.",
		Eval:        evaluateASCIIArt("location", 4),
	}
}

func buildASCIISignageCase() asciiBenchmarkCase {
	return asciiBenchmarkCase{
		ID:          "neon-signage",
		Title:       "Neon signage",
		Category:    "signage",
		Weight:      0.15,
		Temperature: 0.3,
		MaxTokens:   260,
		Kind:        "signage",
		Subject:     "Tallone d'Oro club entrance sign",
		Detail:      "flickering devotional slogan, gaudy neon elegance, terminal-friendly",
		Placement:   "scene_header",
		Mood:        "flashy, decadent",
		Location:    "Ingresso Tallone d'Oro",
		SceneType:   "arrival",
		Narrative:   "Davanti a te vibra l'insegna del Tallone d'Oro, il locale più chiassoso del distretto, pieno di lustrini, vapori dolciastri e culto ostentato.",
		Eval:        evaluateASCIIArt("signage", 3),
	}
}

func buildASCIITerminalCase() asciiBenchmarkCase {
	return asciiBenchmarkCase{
		ID:          "terminal-screen",
		Title:       "Terminal screen",
		Category:    "terminal",
		Weight:      0.15,
		Temperature: 0.2,
		MaxTokens:   260,
		Kind:        "terminal",
		Subject:     "cult access terminal warning display",
		Detail:      "access denied, devotional industrial UI, compact",
		Placement:   "inline",
		Mood:        "clinical, tense",
		Location:    "Archivio del Coro",
		SceneType:   "investigation",
		Narrative:   "Il terminale del Coro del Sale lampeggia davanti a te: è vecchio, sporco di condensa e protetto da un linguaggio liturgico trasformato in interfaccia.",
		Eval:        evaluateASCIIArt("terminal", 3),
	}
}

func buildASCIIRitualCase() asciiBenchmarkCase {
	return asciiBenchmarkCase{
		ID:          "ritual-circle",
		Title:       "Ritual circle / diagram",
		Category:    "ritual",
		Weight:      0.20,
		Temperature: 0.4,
		MaxTokens:   320,
		Kind:        "ritual",
		Subject:     "devotional ritual circle for bell-memory resonance",
		Detail:      "salt ring, bell sigils, concentric geometry, ominous but readable",
		Placement:   "scene_header",
		Mood:        "occult, ceremonial",
		Location:    "Altare sommerso",
		SceneType:   "ritual",
		Narrative:   "L'altare sommerso si illumina appena. Intorno alla campana nera emerge un diagramma di sale e rame, disposto come un rituale di memoria e colpa.",
		Eval:        evaluateASCIIArt("ritual", 4),
	}
}

func buildASCIIMapCase() asciiBenchmarkCase {
	return asciiBenchmarkCase{
		ID:          "map-fragment",
		Title:       "Map fragment",
		Category:    "map",
		Weight:      0.15,
		Temperature: 0.2,
		MaxTokens:   300,
		Kind:        "map",
		Subject:     "smuggler map fragment of harbor tunnels and floodgates",
		Detail:      "simple district paths, junctions, water channels, a marked safe route",
		Placement:   "inline",
		Mood:        "practical, clandestine",
		Location:    "Old Harbor",
		SceneType:   "investigation",
		Narrative:   "Sul retro del registro trovi una mappa grezza dei tunnel sotto l'Old Harbor, con valvole, chiuse e un passaggio segnato a carbone.",
		Eval:        evaluateASCIIArt("map", 4),
	}
}

func buildASCIIArtifactCase() asciiBenchmarkCase {
	return asciiBenchmarkCase{
		ID:          "artifact-reveal",
		Title:       "Iconic artifact reveal",
		Category:    "artifact",
		Weight:      0.15,
		Temperature: 0.35,
		MaxTokens:   280,
		Kind:        "artifact",
		Subject:     "neuro-podale reliquary implant in a velvet case",
		Detail:      "small iconic object, devotional luxury, compact silhouette",
		Placement:   "inline",
		Mood:        "luxe, unsettling",
		Location:    "Camera privata della Dee",
		SceneType:   "reveal",
		Narrative:   "La custodia si apre e mostra un impianto podale rituale, lucido e quasi sacro, progettato per sembrare una reliquia da indossare sul corpo.",
		Eval:        evaluateASCIIArt("artifact", 3),
	}
}

func evaluateASCIIArt(kind string, minLines int) func(string) asciiCaseEvaluation {
	return func(raw string) asciiCaseEvaluation {
		points := map[string]float64{
			"json_payload": 10,
			"json_only":    5,
			"parseable":    15,
			"non_empty":    20,
			"bounded":      20,
			"ascii_only":   10,
			"drawing":      10,
			"case_fit":     10,
		}

		eval := baseASCIIJSONEvaluation(raw)
		score := 0.0
		if eval.JSONValid {
			score += points["json_payload"]
		}
		if eval.JSONOnly {
			score += points["json_only"]
		}
		if !eval.JSONValid {
			eval.Score = score
			eval.Breakdown = points
			return eval
		}

		payload, err := ai.ParseASCIIArtJSON(raw)
		if err != nil || payload == nil {
			if err != nil {
				eval.Notes = append(eval.Notes, "ASCII payload non parseabile: "+err.Error())
			} else {
				eval.Notes = append(eval.Notes, "ASCII payload assente")
			}
			eval.Score = score
			eval.Breakdown = points
			return eval
		}
		eval.Parsed = true
		score += points["parseable"]

		art := strings.TrimSpace(payload.ASCIIArt)
		if art != "" {
			score += points["non_empty"]
		} else {
			eval.Notes = append(eval.Notes, "ascii_art vuoto")
		}

		lineCount, maxWidth := asciiDimensions(art)
		eval.LineCount = lineCount
		eval.MaxWidth = maxWidth
		if lineCount >= minLines && lineCount <= 12 && maxWidth <= 72 {
			score += points["bounded"]
		} else {
			eval.Notes = append(eval.Notes, fmt.Sprintf("Fuori bounds utili: lines=%d width=%d", lineCount, maxWidth))
		}

		if isASCIIOnly(art) {
			eval.ASCIIOnly = true
			score += points["ascii_only"]
		} else {
			eval.Notes = append(eval.Notes, "Contiene caratteri non ASCII")
		}

		if hasDrawingSignal(art) {
			eval.HasDrawing = true
			score += points["drawing"]
		} else {
			eval.Notes = append(eval.Notes, "Art poco visiva: troppo testo o pochi segni grafici")
		}

		if fitsASCIICase(kind, art) {
			score += points["case_fit"]
		} else {
			eval.Notes = append(eval.Notes, "La forma non sembra ideale per il tipo di scena richiesto")
		}

		eval.Score = score
		eval.Breakdown = points
		return eval
	}
}

func baseASCIIJSONEvaluation(raw string) asciiCaseEvaluation {
	eval := asciiCaseEvaluation{Breakdown: map[string]float64{}}
	jsonText, err := ai.ExtractJSONPayload(raw)
	if err != nil {
		eval.Notes = append(eval.Notes, "Payload JSON non parseabile: "+err.Error())
		return eval
	}
	if jsonText == "" {
		eval.Notes = append(eval.Notes, "Nessun payload JSON parseabile trovato")
		return eval
	}
	eval.JSONValid = true
	eval.RawJSON = jsonText
	trimmed := strings.TrimSpace(raw)
	if trimmed == strings.TrimSpace("```json\n"+jsonText+"\n```") || trimmed == strings.TrimSpace(jsonText) {
		eval.JSONOnly = true
	}
	return eval
}

func asciiDimensions(art string) (int, int) {
	if strings.TrimSpace(art) == "" {
		return 0, 0
	}
	lines := strings.Split(art, "\n")
	maxWidth := 0
	for _, line := range lines {
		if width := len([]rune(line)); width > maxWidth {
			maxWidth = width
		}
	}
	return len(lines), maxWidth
}

func isASCIIOnly(text string) bool {
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if r > 127 {
			return false
		}
	}
	return true
}

func hasDrawingSignal(art string) bool {
	drawing := 0
	for _, r := range art {
		if strings.ContainsRune(`/\|-_=+*#<>[](){}.:;'"~^`, r) {
			drawing++
		}
	}
	return drawing >= 10
}

func fitsASCIICase(kind, art string) bool {
	lines, width := asciiDimensions(art)
	lower := strings.ToLower(art)
	switch kind {
	case "signage":
		return lines >= 3 && width <= 72 && (strings.Contains(lower, "tallone") || strings.Contains(lower, "oro") || hasDrawingSignal(art))
	case "terminal":
		return lines >= 3 && (strings.Contains(art, "|") || strings.Contains(art, "+") || strings.Contains(art, "["))
	case "map":
		return lines >= 4 && (strings.Contains(art, "+") || strings.Contains(art, "|") || strings.Contains(art, "-") || strings.Contains(art, "/"))
	case "ritual":
		return lines >= 4 && (strings.Contains(art, "*") || strings.Contains(art, "o") || strings.Contains(art, "("))
	case "artifact":
		return lines >= 3 && width <= 72 && hasDrawingSignal(art)
	case "location":
		fallthrough
	default:
		return lines >= 4 && hasDrawingSignal(art)
	}
}

func (c *asciiClient) complete(ctx context.Context, model string, bc asciiBenchmarkCase, mode asciiBenchmarkMode, meta asciiModelMetadata) (string, string, asciiUsage, time.Duration, error) {
	body := asciiAPIRequest{
		Model: model,
		Messages: []ai.Message{
			{Role: ai.RoleSystem, Content: prompts.ASCIIArtSystem()},
			{Role: ai.RoleUser, Content: prompts.ASCIIArtUser(
				"OneDay ASCII Benchmark",
				bc.Location,
				bc.SceneType,
				bc.Mood,
				bc.Narrative,
				bc.Kind,
				bc.Subject,
				bc.Detail,
				bc.Placement,
			)},
		},
		Temperature: bc.Temperature,
		MaxTokens:   bc.MaxTokens,
	}
	if responseFormat := asciiResponseFormatForMode(mode, meta); responseFormat != nil {
		body.ResponseFormat = responseFormat
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", "", asciiUsage{}, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", asciiUsage{}, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://example.com/oneday-ascii-benchmark")
	req.Header.Set("X-Title", "OneDay ASCII Benchmark")

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", asciiUsage{}, 0, err
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", asciiUsage{}, duration, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", asciiUsage{}, duration, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed asciiAPIResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", "", asciiUsage{}, duration, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", "", asciiUsage{}, duration, fmt.Errorf("no choices returned")
	}

	return parsed.Choices[0].Message.Content, parsed.Model, parsed.Usage, duration, nil
}

func asciiResponseFormatForMode(mode asciiBenchmarkMode, meta asciiModelMetadata) *ai.ResponseFormat {
	switch mode {
	case asciiModePrompt:
		return nil
	case asciiModeJSONObject:
		if meta.SupportsResponseFormat {
			return ai.NewJSONObjectResponseFormat()
		}
	case asciiModeJSONSchema:
		if meta.SupportsStructured {
			return ai.ASCIIArtResponseFormat()
		}
	}
	return nil
}

func (c *asciiClient) fetchModelMetadata(models []string) map[string]asciiModelMetadata {
	out := map[string]asciiModelMetadata{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return out
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out
	}

	var payload asciiModelCatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return out
	}

	index := map[string]asciiModelCatalogEntry{}
	for _, entry := range payload.Data {
		index[entry.ID] = entry
	}

	for _, model := range models {
		entry, ok := index[model]
		if !ok {
			out[model] = asciiModelMetadata{ID: model}
			continue
		}
		out[model] = asciiModelMetadata{
			ID:                     entry.ID,
			Name:                   entry.Name,
			ContextLength:          entry.ContextLength,
			MaxCompletionTokens:    entry.TopProvider.MaxCompletionTokens,
			PromptCostPerTokenUSD:  parsePrice(entry.Pricing["prompt"]),
			OutputCostPerTokenUSD:  parsePrice(entry.Pricing["completion"]),
			SupportsResponseFormat: contains(entry.SupportedParameters, "response_format"),
			SupportsStructured:     contains(entry.SupportedParameters, "structured_outputs"),
			SupportedParameters:    entry.SupportedParameters,
		}
	}
	return out
}

func estimateASCIICostUSD(meta asciiModelMetadata, usage asciiUsage) float64 {
	return meta.PromptCostPerTokenUSD*float64(usage.PromptTokens) +
		meta.OutputCostPerTokenUSD*float64(usage.CompletionTokens)
}

func writeASCIIJSONReport(path string, report asciiBenchmarkReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeASCIIMarkdownReport(path, jsonName string, report asciiBenchmarkReport) error {
	var b strings.Builder
	b.WriteString("# OneDay ASCII Model Benchmark\n\n")
	b.WriteString(fmt.Sprintf("- Generated: `%s`\n", report.GeneratedAt))
	b.WriteString(fmt.Sprintf("- Base URL: `%s`\n", report.BaseURL))
	b.WriteString(fmt.Sprintf("- Mode: `%s`\n", report.Mode))
	b.WriteString(fmt.Sprintf("- Raw JSON artifact: `%s`\n", jsonName))
	b.WriteString("- Scoring: automated ASCII runtime suitability; visual taste still needs human review.\n\n")

	b.WriteString("## Overall Leaderboard\n\n")
	b.WriteString("| Model | Score | Success | Avg Seconds | Avg Cost | Context | Max Out | Avg Tok/s |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, summary := range report.Summaries {
		b.WriteString(fmt.Sprintf("| `%s` | %.1f | %.0f%% | %.3f | $%.6f | %d | %d | %.1f |\n",
			summary.Model,
			summary.WeightedScore,
			summary.SuccessRate,
			summary.AverageLatencySec,
			summary.AverageCostUSD,
			summary.Metadata.ContextLength,
			summary.Metadata.MaxCompletionTokens,
			summary.AverageTokensPerSec,
		))
	}
	b.WriteString("\n")

	appendRankingTable := func(title string, value func(asciiModelSummary) float64) {
		sorted := append([]asciiModelSummary(nil), report.Summaries...)
		sort.Slice(sorted, func(i, j int) bool {
			return value(sorted[i]) > value(sorted[j])
		})
		b.WriteString("## " + title + "\n\n")
		b.WriteString("| Model | Metric | Score | Avg Seconds | Avg Cost |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: |\n")
		for _, summary := range sorted {
			b.WriteString(fmt.Sprintf("| `%s` | %.3f | %.1f | %.3f | $%.6f |\n",
				summary.Model,
				value(summary),
				summary.WeightedScore,
				summary.AverageLatencySec,
				summary.AverageCostUSD,
			))
		}
		b.WriteString("\n")
	}

	appendRankingTable("Best Quality / Cost", qualityPerCost)
	appendRankingTable("Best Quality / Latency", qualityPerLatency)
	appendRankingTable("Best Practical Runtime Fit", runtimeFitScore)

	for _, summary := range report.Summaries {
		b.WriteString(fmt.Sprintf("## %s\n\n", summary.Model))
		b.WriteString(fmt.Sprintf("- Context: `%d`\n", summary.Metadata.ContextLength))
		b.WriteString(fmt.Sprintf("- Max completion: `%d`\n", summary.Metadata.MaxCompletionTokens))
		b.WriteString(fmt.Sprintf("- Prompt cost/token: `$%.9f`\n", summary.Metadata.PromptCostPerTokenUSD))
		b.WriteString(fmt.Sprintf("- Completion cost/token: `$%.9f`\n", summary.Metadata.OutputCostPerTokenUSD))
		b.WriteString(fmt.Sprintf("- Avg latency: `%.3f s`\n", summary.AverageLatencySec))
		b.WriteString(fmt.Sprintf("- Avg cost: `$%.6f`\n", summary.AverageCostUSD))
		b.WriteString(fmt.Sprintf("- Total benchmark cost: `$%.6f`\n\n", summary.TotalCostUSD))
		for _, result := range summary.Results {
			b.WriteString(fmt.Sprintf("### %s\n\n", result.CaseTitle))
			if result.Error != "" {
				b.WriteString(fmt.Sprintf("- Error: `%s`\n\n", result.Error))
				continue
			}
			b.WriteString(fmt.Sprintf("- Score: `%.1f/100`\n", result.Evaluation.Score))
			b.WriteString(fmt.Sprintf("- Duration: `%.3f s`\n", result.DurationSeconds))
			b.WriteString(fmt.Sprintf("- Output shape: `%d lines`, `max width %d`\n", result.Evaluation.LineCount, result.Evaluation.MaxWidth))
			b.WriteString(fmt.Sprintf("- Completion tokens: `%d`\n", result.CompletionTokens))
			b.WriteString(fmt.Sprintf("- Prompt tokens: `%d`\n", result.PromptTokens))
			b.WriteString(fmt.Sprintf("- Estimated cost: `$%.6f`\n", result.CostUSD))
			b.WriteString(fmt.Sprintf("- Throughput: `%.1f tok/s`, `%.1f char/s`\n", result.TokensPerSecond, result.CharsPerSecond))
			if len(result.Evaluation.Notes) > 0 {
				b.WriteString("- Notes:\n")
				for _, note := range result.Evaluation.Notes {
					b.WriteString(fmt.Sprintf("  - %s\n", note))
				}
			}
			b.WriteString("\n```text\n")
			b.WriteString(truncate(result.RawResponse, 1400))
			b.WriteString("\n```\n\n")
		}
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func qualityPerCost(summary asciiModelSummary) float64 {
	if summary.AverageCostUSD <= 0 {
		return summary.WeightedScore * 1000000
	}
	return summary.WeightedScore / summary.AverageCostUSD
}

func qualityPerLatency(summary asciiModelSummary) float64 {
	if summary.AverageLatencySec <= 0 {
		return 0
	}
	return summary.WeightedScore / summary.AverageLatencySec
}

func runtimeFitScore(summary asciiModelSummary) float64 {
	if summary.AverageLatencySec <= 0 {
		return 0
	}
	return summary.WeightedScore * (summary.SuccessRate / 100.0) / summary.AverageLatencySec
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func splitCSV(text string) []string {
	var out []string
	for _, part := range strings.Split(text, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func wordCount(text string) int {
	return len(strings.Fields(text))
}

func truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return text[:max] + "\n...[truncated]..."
}

func contains(values []string, needle string) bool {
	for _, v := range values {
		if v == needle {
			return true
		}
	}
	return false
}

func parsePrice(raw string) float64 {
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return v
}
