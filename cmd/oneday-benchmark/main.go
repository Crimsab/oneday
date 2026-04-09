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
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
)

type benchmarkCase struct {
	ID          string
	Title       string
	Category    string
	Weight      float64
	Temperature float64
	MaxTokens   int
	Messages    []ai.Message
	Schema      *ai.ResponseFormat
	Eval        func(string) caseEvaluation
}

type benchmarkMode string

const (
	modePrompt     benchmarkMode = "prompt"
	modeJSONObject benchmarkMode = "json_object"
	modeJSONSchema benchmarkMode = "json_schema"
	modeAll        benchmarkMode = "all"
)

type caseEvaluation struct {
	Score        float64            `json:"score"`
	Breakdown    map[string]float64 `json:"breakdown"`
	Notes        []string           `json:"notes"`
	RawJSON      string             `json:"raw_json,omitempty"`
	JSONValid    bool               `json:"json_valid"`
	JSONOnly     bool               `json:"json_only"`
	Parsed       bool               `json:"parsed"`
	WordCount    int                `json:"word_count,omitempty"`
	LooksItalian bool               `json:"looks_italian"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type apiRequest struct {
	Model          string             `json:"model"`
	Messages       []ai.Message       `json:"messages"`
	Temperature    float64            `json:"temperature,omitempty"`
	MaxTokens      int                `json:"max_tokens,omitempty"`
	ResponseFormat *ai.ResponseFormat `json:"response_format,omitempty"`
}

type apiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage usage  `json:"usage"`
}

type modelCatalogResponse struct {
	Data []modelCatalogEntry `json:"data"`
}

type modelCatalogEntry struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	ContextLength       int               `json:"context_length"`
	Pricing             map[string]string `json:"pricing"`
	SupportedParameters []string          `json:"supported_parameters"`
	TopProvider         struct {
		ContextLength       int  `json:"context_length"`
		MaxCompletionTokens int  `json:"max_completion_tokens"`
		IsModerated         bool `json:"is_moderated"`
	} `json:"top_provider"`
}

type modelMetadata struct {
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

type caseResult struct {
	CaseID           string         `json:"case_id"`
	CaseTitle        string         `json:"case_title"`
	Category         string         `json:"category"`
	Weight           float64        `json:"weight"`
	ModelRequested   string         `json:"model_requested"`
	ModelResolved    string         `json:"model_resolved,omitempty"`
	DurationMS       int64          `json:"duration_ms"`
	DurationSeconds  float64        `json:"duration_seconds"`
	CompletionTokens int            `json:"completion_tokens"`
	PromptTokens     int            `json:"prompt_tokens"`
	TotalTokens      int            `json:"total_tokens"`
	TokensPerSecond  float64        `json:"tokens_per_second"`
	CharsPerSecond   float64        `json:"chars_per_second"`
	CostUSD          float64        `json:"cost_usd"`
	OutputChars      int            `json:"output_chars"`
	OutputWords      int            `json:"output_words"`
	Compatibility    caseEvaluation `json:"compatibility"`
	RawResponse      string         `json:"raw_response,omitempty"`
	Error            string         `json:"error,omitempty"`
}

type modelSummary struct {
	Model               string        `json:"model"`
	Metadata            modelMetadata `json:"metadata"`
	WeightedCompatScore float64       `json:"weighted_compat_score"`
	SuccessRate         float64       `json:"success_rate"`
	AverageLatencyMS    float64       `json:"average_latency_ms"`
	AverageLatencySec   float64       `json:"average_latency_seconds"`
	AverageTokensPerSec float64       `json:"average_tokens_per_second"`
	AverageCharsPerSec  float64       `json:"average_chars_per_second"`
	AverageCostUSD      float64       `json:"average_cost_usd"`
	TotalCostUSD        float64       `json:"total_cost_usd"`
	Results             []caseResult  `json:"results"`
}

type benchmarkReport struct {
	GeneratedAt string         `json:"generated_at"`
	BaseURL     string         `json:"base_url"`
	Mode        benchmarkMode  `json:"mode"`
	Models      []string       `json:"models"`
	Summaries   []modelSummary `json:"summaries"`
}

type client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func main() {
	defaultModels := strings.Join([]string{
		"x-ai/grok-4.1-fast",
		"qwen/qwen3.5-flash-02-23",
		"google/gemini-2.5-flash-lite",
		"google/gemini-3.1-flash-lite-preview",
	}, ",")

	baseURLFlag := flag.String("base-url", firstNonEmpty(
		os.Getenv("ONEDAY_BENCH_BASE_URL"),
		os.Getenv("OPENAI_BASE_URL"),
		"https://openrouter.ai/api/v1",
	), "OpenAI-compatible base URL ending in /v1")
	apiKeyFlag := flag.String("api-key", firstNonEmpty(
		os.Getenv("ONEDAY_BENCH_API_KEY"),
		os.Getenv("OPENROUTER_API_KEY"),
		os.Getenv("OPENAI_API_KEY"),
	), "API key for the benchmark endpoint")
	modelsFlag := flag.String("models", firstNonEmpty(os.Getenv("ONEDAY_BENCH_MODELS"), defaultModels), "Comma-separated model list")
	outputDirFlag := flag.String("output-dir", firstNonEmpty(os.Getenv("ONEDAY_BENCH_OUTPUT_DIR"), "docs/benchmarks/runs"), "Directory for JSON and Markdown reports")
	timeoutFlag := flag.Duration("timeout", 90*time.Second, "Per-request timeout")
	modeFlag := flag.String("mode", string(modePrompt), "Benchmark mode: prompt, json_object, json_schema, all")
	versionFlag := flag.Bool("version", false, "Print build information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(buildinfo.Text("oneday-benchmark"))
		return
	}

	if strings.TrimSpace(*apiKeyFlag) == "" {
		fmt.Fprintln(os.Stderr, "missing API key: pass -api-key or export ONEDAY_BENCH_API_KEY / OPENROUTER_API_KEY")
		os.Exit(1)
	}

	models := splitCSV(*modelsFlag)
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "missing models list")
		os.Exit(1)
	}

	cli := &client{
		baseURL: strings.TrimRight(*baseURLFlag, "/"),
		apiKey:  strings.TrimSpace(*apiKeyFlag),
		client:  &http.Client{Timeout: *timeoutFlag},
	}

	cases := buildBenchmarkCases()
	modelMeta := cli.fetchModelMetadata(models)

	mode := benchmarkMode(strings.TrimSpace(*modeFlag))
	var modes []benchmarkMode
	switch mode {
	case modePrompt, modeJSONObject, modeJSONSchema:
		modes = []benchmarkMode{mode}
	case modeAll:
		modes = []benchmarkMode{modePrompt, modeJSONObject, modeJSONSchema}
	default:
		fmt.Fprintf(os.Stderr, "unsupported mode %q (expected prompt, json_object, json_schema, all)\n", *modeFlag)
		os.Exit(1)
	}

	outputDir := *outputDirFlag
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "creating output dir: %v\n", err)
		os.Exit(1)
	}

	for _, benchMode := range modes {
		report := benchmarkReport{
			GeneratedAt: time.Now().Format(time.RFC3339),
			BaseURL:     cli.baseURL,
			Mode:        benchMode,
			Models:      models,
		}

		for _, model := range models {
			fmt.Fprintf(os.Stdout, "benchmarking %s (%s)\n", model, benchMode)
			summary := runModelBenchmarks(cli, model, modelMeta[model], cases, benchMode)
			report.Summaries = append(report.Summaries, summary)
		}

		sort.Slice(report.Summaries, func(i, j int) bool {
			return report.Summaries[i].WeightedCompatScore > report.Summaries[j].WeightedCompatScore
		})

		stamp := time.Now().Format("2006-01-02-150405")
		baseName := fmt.Sprintf("%s-oneday-benchmark-%s", stamp, benchMode)
		jsonPath := filepath.Join(outputDir, baseName+".json")
		mdPath := filepath.Join(outputDir, baseName+".md")

		if err := writeJSONReport(jsonPath, report); err != nil {
			fmt.Fprintf(os.Stderr, "writing json report: %v\n", err)
			os.Exit(1)
		}
		if err := writeMarkdownReport(mdPath, filepath.Base(jsonPath), report); err != nil {
			fmt.Fprintf(os.Stderr, "writing markdown report: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stdout, "json report: %s\n", jsonPath)
		fmt.Fprintf(os.Stdout, "markdown report: %s\n", mdPath)
	}
}

func runModelBenchmarks(cli *client, model string, meta modelMetadata, cases []benchmarkCase, mode benchmarkMode) modelSummary {
	var results []caseResult
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
		result := caseResult{
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
		result.CostUSD = estimateCostUSD(meta, usage)
		result.OutputChars = len(raw)
		result.OutputWords = wordCount(raw)
		if duration > 0 {
			seconds := duration.Seconds()
			result.CharsPerSecond = float64(result.OutputChars) / seconds
			if usage.CompletionTokens > 0 {
				result.TokensPerSecond = float64(usage.CompletionTokens) / seconds
			}
		}

		result.Compatibility = bc.Eval(raw)
		results = append(results, result)

		totalWeight += bc.Weight
		weightedScore += result.Compatibility.Score * bc.Weight
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

	summary := modelSummary{
		Model:    model,
		Metadata: meta,
		Results:  results,
	}
	if totalWeight > 0 {
		summary.WeightedCompatScore = weightedScore / totalWeight
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

func buildBenchmarkCases() []benchmarkCase {
	return []benchmarkCase{
		buildStoryCreationCase(),
		buildNarrativeIntroCase(),
		buildDialogueMetadataCase(),
		buildChallengeCase(),
		buildChapterSummaryCase(),
	}
}

func buildStoryCreationCase() benchmarkCase {
	messages := []ai.Message{
		{Role: ai.RoleSystem, Content: prompts.StoryCreationSystem},
		{Role: ai.RoleUser, Content: "Vorrei un fantasy oscuro ispirato a una Venezia decadente, con politica, culti del mare e un tono malinconico ma avventuroso. Voglio la storia in italiano, con una prosa elegante e inquieta, e come direttiva extra tieni i dialoghi asciutti e il pericolo sempre vicino."},
		{Role: ai.RoleAssistant, Content: "Perfetto. Ti propongo una citta-laguna chiamata Vespera, costruita su isole nere e canali di sale. La magia nasce dal suono delle campane sommerse e ogni casata nobile cerca di controllarne gli echi. Il tono e cupo, elegante e pieno di intrighi. La storia restera in italiano con una voce elegante e inquieta, dialoghi asciutti e tensione costante. Se vuoi, possiamo fissare regole del mondo, fazioni e statistiche."},
		{Role: ai.RoleUser, Content: "Si. Voglio che ci siano corporazioni mercantili spietate, una religione delle campane sommerse, guardie di porto corrotte e quartieri allagati pieni di reliquie."},
		{Role: ai.RoleAssistant, Content: "Ottimo. Regole del mondo: la magia ha sempre un costo, il sale protegge dai sussurri del mare, le campane sommerse alterano memoria e volonta, i giuramenti pubblici hanno valore legale e spirituale, il debito non pagato puo essere venduto come servitu. Fazioni: Casata Valcerra, Coro del Sale, Guardie di Marea. Culture: nobili di canale, scavatori di relitti, pellegrini delle campane. Pericoli: nebbie senzienti, reliquie infette, inquisitori del suono, allagamenti improvvisi. Statistiche: vigore, agilita, ingegno, presenza, volonta, occulto, piu reputazione e debito. Confermi il pacchetto finale?"},
		{Role: ai.RoleUser, Content: "Sì, confermo tutto. Genera adesso il JSON finale definitivo."},
	}

	return benchmarkCase{
		ID:          "story-creation-final",
		Title:       "Story creation final JSON",
		Category:    "story-creation",
		Weight:      0.15,
		Temperature: 0.4,
		MaxTokens:   1400,
		Messages:    messages,
		Schema:      ai.StoryDefinitionResponseFormat(),
		Eval:        evaluateStoryDefinition,
	}
}

func buildNarrativeIntroCase() benchmarkCase {
	story := &storage.Story{
		Name:             "Le Campane di Vespera",
		Description:      "Un fantasy oscuro di intrighi, culti sommersi e reliquie cantanti.",
		Genre:            "fantasy oscuro",
		Tone:             "malinconico, elegante, teso",
		Language:         "italiano",
		WritingStyle:     "fantasy urbano malinconico con prosa elegante, sensoriale e controllata",
		PromptDirectives: "Dialoghi asciutti, pericolo costante, evita prolissita.",
		SettingJSON:      `{"world_name":"Vespera","era":"Età delle Maree Spezzate","geography":"Una città lagunare di ponti bassi, banchine nere e isole collegate da canali salmastri","magic_system":"Il suono delle campane sommerse piega memoria, volontà e fortuna, ma ogni uso lascia un prezzo","technology_level":"Rinascimento decadente con alchimia del sale","society":"Casate nobili, corporazioni mercantili, scavatori di relitti e ordini religiosi in lotta","rules":["La magia del suono ha sempre un prezzo","Il sale puro protegge dai sussurri","I debiti possono diventare servitù legale","Le reliquie emerse attirano culti e predoni"],"factions":["Casata Valcerra — nobili mercanti che controllano le banchine orientali","Coro del Sale — ordine religioso che custodisce campane e reliquie","Guardie di Marea — pattuglie portuali corrotte ma indispensabili"],"cultures":["Nobili di canale","Scavatori di relitti","Pellegrini delle campane"],"dangers":["Nebbie senzienti","Reliquie infette","Allagamenti improvvisi","Predoni dei canali"]}`,
		StatsSchemaJSON:  `{"vitals":[{"key":"hp","label":"Salute","starting":10},{"key":"stress","label":"Stress","starting":3}],"attributes":[{"key":"vig","label":"Vigore","starting":3},{"key":"agi","label":"Agilità","starting":3},{"key":"ing","label":"Ingegno","starting":3},{"key":"pre","label":"Presenza","starting":3},{"key":"vol","label":"Volontà","starting":3},{"key":"occ","label":"Occulto","starting":3}],"secondary":[{"key":"rep","label":"Reputazione","starting":0},{"key":"debt","label":"Debito","starting":2}],"currency":{"name":"Corone di sale","starting":8},"has_combat":true}`,
	}
	character := &storage.Character{
		Name:       "Nerea",
		Background: "Ex corriere delle banchine, cresciuta tra debiti di famiglia e mappe rubate.",
		StatsJSON:  `{"vitals":{"hp":{"current":10,"max":10},"stress":{"current":3,"max":3}},"attributes":{"vig":3,"agi":3,"ing":4,"pre":2,"vol":3,"occ":1},"secondary":{"rep":0,"debt":2},"currency":8,"traits":[],"skills":{"Canali":{"level":1,"xp":15}},"titles":[],"deaths":0}`,
	}
	world := &storage.WorldState{
		CurrentLocation:      "Banchina delle Nebbie",
		KnownLocationsJSON:   `["Banchina delle Nebbie"]`,
		GlobalEventsJSON:     `[]`,
		FactionStandingsJSON: `{"Casata Valcerra":0,"Coro del Sale":0,"Guardie di Marea":-5}`,
		CurrentChapter:       1,
		CurrentTurn:          0,
	}

	return benchmarkCase{
		ID:          "narrative-intro",
		Title:       "Narrative first turn",
		Category:    "narrative",
		Weight:      0.25,
		Temperature: 0.8,
		MaxTokens:   1200,
		Messages:    engine.BuildContext(story, character, world, nil, nil, nil, "", prompts.FirstTurnUser, nil),
		Schema:      ai.NarrativeResponseFormat(),
		Eval:        evaluateNarrativeBasic,
	}
}

func buildDialogueMetadataCase() benchmarkCase {
	story := &storage.Story{
		Name:             "Le Campane di Vespera",
		Description:      "Fantasy oscuro veneziano pieno di debiti, reliquie e culti del suono.",
		Genre:            "fantasy oscuro",
		Tone:             "teso, urbano, misterioso",
		Language:         "italiano",
		WritingStyle:     "dark fantasy urbano con dialoghi tesi e immagini acquatiche precise",
		PromptDirectives: "Tieni il ritmo serrato, fai emergere il pericolo attraverso suoni e dettagli d'acqua.",
		SettingJSON:      `{"world_name":"Vespera","era":"Età delle Maree Spezzate","geography":"Città di moli, cortili allagati e campanili semi-affondati","magic_system":"Le campane sommerse risvegliano ricordi e giuramenti","technology_level":"Rinascimento decadente","society":"Mercanti, contrabbandieri, pellegrini e guardie corrotte","rules":["Il sale sigilla i sussurri","La memoria può essere alterata dal suono","I debiti sono pubblici e spietati"],"factions":["Dock Wardens","Salt Choir","House Valcerra"],"cultures":["Scavatori","Mercanti","Pellegrini"],"dangers":["Patrols","Flood surges","Relic fever"]}`,
		StatsSchemaJSON:  `{"vitals":[{"key":"hp","label":"Salute","starting":10},{"key":"stress","label":"Stress","starting":4}],"attributes":[{"key":"agi","label":"Agilità","starting":3},{"key":"ing","label":"Ingegno","starting":4},{"key":"pre","label":"Presenza","starting":3},{"key":"vol","label":"Volontà","starting":3}],"secondary":[{"key":"rep","label":"Reputazione","starting":1}],"currency":{"name":"Corone di sale","starting":5},"has_combat":true}`,
	}
	character := &storage.Character{
		Name:       "Nerea",
		Background: "Ex corriere che ha appena rubato una reliquia di scarso valore ma enorme rischio.",
		StatsJSON:  `{"vitals":{"hp":{"current":9,"max":10},"stress":{"current":4,"max":4}},"attributes":{"agi":3,"ing":4,"pre":3,"vol":3},"secondary":{"rep":1},"currency":5,"traits":["Cauta"],"skills":{"Furtività":{"level":1,"xp":20},"Canali":{"level":2,"xp":10}},"titles":[],"deaths":0}`,
	}
	world := &storage.WorldState{
		CurrentLocation:      "Old Harbor",
		KnownLocationsJSON:   `["Old Harbor","Mercato della Marea","Passi del Campanile"]`,
		GlobalEventsJSON:     `["Le campane sommerse hanno risuonato all'alba","Le pattuglie del porto stanno fermando chiunque porti reliquie"]`,
		FactionStandingsJSON: `{"Dock Wardens":-10,"Salt Choir":5,"House Valcerra":0}`,
		CurrentChapter:       1,
		CurrentTurn:          7,
	}
	npcs := []storage.NPC{
		{
			Name:               "Lyanna Voss",
			Role:               "contrabbandiera",
			Appearance:         "Capelli scuri raccolti male, mantello cerato, occhi stanchi che registrano tutto",
			PersonalityJSON:    `{"traits":["cauta","percettiva"],"speech_style":"basso, asciutto, tagliente","quirks":["tocca due volte il tavolo prima di mentire"],"values":["sopravvivenza","debiti ripagati"],"fears":["campane sommerse","trappole delle guardie"]}`,
			PrivateThoughts:    `["Nerea è utile ma si avvicina troppo ai guai grossi"]`,
			NotesOnProtagonist: `["Ha sangue freddo sotto pressione","Tiene troppo ai debiti di famiglia"]`,
			Desires:            `[{"desire":"Lasciare Vespera con abbastanza denaro per sparire","priority":"high","known_to_player":false}]`,
			Disposition:        18,
			IsAlive:            true,
			FirstAppearedTurn:  4,
			LastSeenTurn:       7,
			CanHelp:            true,
		},
	}
	recent := []storage.ChatMessage{
		{
			Role:    "assistant",
			Content: "```json\n{\"narrative\":\"La pioggia fina martella i pali del molo. Lyanna Voss ti aspetta nell'ombra di una tettoia, una mano sul coltello e l'altra su un sacchetto di sale. Le pattuglie stanno chiudendo il porto un varco alla volta.\",\"choices\":[{\"id\":1,\"text\":\"Mostrale la reliquia avvolta nel panno\"},{\"id\":2,\"text\":\"Chiedile cosa ha sentito sotto le campane\"},{\"id\":3,\"text\":\"Dirigiti subito verso un'uscita secondaria\"}],\"mood\":\"tense\",\"location\":\"Old Harbor\"}\n```",
		},
	}
	messages := engine.BuildContext(
		story,
		character,
		world,
		npcs,
		recent,
		nil,
		"",
		"I keep my voice low and ask Lyanna what she really heard beneath the bells, and whether the patrols are looking for me or for the relic. Rispondi in italiano.",
		[]storage.Achievement{{Name: "Debito sul Molo", Category: "story"}},
	)

	return benchmarkCase{
		ID:          "dialogue-metadata",
		Title:       "Dialogue scene with renderer metadata",
		Category:    "narrative-structured",
		Weight:      0.25,
		Temperature: 0.7,
		MaxTokens:   1300,
		Messages:    messages,
		Schema:      ai.NarrativeResponseFormat(),
		Eval:        evaluateNarrativeStructured,
	}
}

func buildChallengeCase() benchmarkCase {
	story := &storage.Story{
		Name:             "Le Campane di Vespera",
		Description:      "Un fantasy urbano in cui ogni fuga costa sangue, sale o memoria.",
		Genre:            "fantasy oscuro",
		Tone:             "teso, cinematico, disperato",
		Language:         "italiano",
		WritingStyle:     "cinematico, nervoso, fisico, con frasi compatte",
		PromptDirectives: "Mantieni alta la pressione della scena e fai sentire il rischio in ogni scelta.",
		SettingJSON:      `{"world_name":"Vespera","era":"Età delle Maree Spezzate","geography":"Ponti stretti, canali gonfi e torri piegate","magic_system":"Le reliquie del suono alterano memoria e fortuna","technology_level":"Rinascimento decadente","society":"Ordini religiosi, mercanti e contrabbandieri si contendono i canali","rules":["La magia richiede sempre un prezzo","Le reliquie attirano pattuglie e culti","Le fughe lasciano tracce"],"factions":["Guardie di Marea","Coro del Sale"],"cultures":["Scavatori","Battellieri"],"dangers":["Ponti cedevoli","Lanternieri armati","Piene improvvise"]}`,
		StatsSchemaJSON:  `{"vitals":[{"key":"hp","label":"Salute","starting":10},{"key":"stress","label":"Stress","starting":5}],"attributes":[{"key":"agi","label":"Agilità","starting":4},{"key":"vol","label":"Volontà","starting":3},{"key":"vig","label":"Vigore","starting":3}],"secondary":[{"key":"rep","label":"Reputazione","starting":0}],"currency":{"name":"Corone di sale","starting":2},"has_combat":true}`,
	}
	character := &storage.Character{
		Name:       "Nerea",
		Background: "Corriere dei canali che conosce i ponti meglio delle preghiere.",
		StatsJSON:  `{"vitals":{"hp":{"current":8,"max":10},"stress":{"current":5,"max":5}},"attributes":{"agi":4,"vol":3,"vig":3},"secondary":{"rep":0},"currency":2,"traits":["Tenace"],"skills":{"Canali":{"level":2,"xp":35},"Corsa":{"level":1,"xp":10}},"titles":[],"deaths":0}`,
	}
	world := &storage.WorldState{
		CurrentLocation:      "Stormglass Causeway",
		KnownLocationsJSON:   `["Stormglass Causeway","Old Harbor"]`,
		GlobalEventsJSON:     `["Le pattuglie hanno chiuso le uscite a nord","Una piena sta spingendo detriti contro i piloni"]`,
		FactionStandingsJSON: `{"Guardie di Marea":-15}`,
		CurrentChapter:       2,
		CurrentTurn:          13,
	}
	recent := []storage.ChatMessage{
		{
			Role:    "assistant",
			Content: "```json\n{\"narrative\":\"Il ponte di vetro tempestato scricchiola sotto il peso dell'acqua e dei detriti. Due lanternieri sbucano dalla sponda opposta, le aste di ferro già alzate. Il solo passaggio è una fune laterale che vibra sopra il canale in piena.\",\"choices\":[{\"id\":1,\"text\":\"Scatta verso la fune e attraversa di corsa\"},{\"id\":2,\"text\":\"Lancia il sacchetto di sale per distrarre i lanternieri\"},{\"id\":3,\"text\":\"Cerca riparo dietro il pilone spezzato\"}],\"mood\":\"desperate\",\"location\":\"Stormglass Causeway\"}\n```",
		},
	}
	messages := engine.BuildContext(
		story,
		character,
		world,
		nil,
		recent,
		nil,
		"Nel capitolo precedente Nerea è sfuggita al sequestro di una reliquia, ma ha attirato il sospetto delle Guardie di Marea.",
		"I sprint for the side rope and try to keep my balance while the bridge collapses and the lantern chains lash overhead. Rispondi in italiano.",
		nil,
	)

	return benchmarkCase{
		ID:          "challenge-scene",
		Title:       "Challenge-producing action scene",
		Category:    "narrative-challenge",
		Weight:      0.20,
		Temperature: 0.7,
		MaxTokens:   1200,
		Messages:    messages,
		Schema:      ai.NarrativeResponseFormat(),
		Eval:        evaluateChallengeNarrative,
	}
}

func buildChapterSummaryCase() benchmarkCase {
	transcript := strings.TrimSpace(`
USER: Begin the story.
ASSISTANT: Nerea arriva alla Banchina delle Nebbie con una mappa rubata cucita nel cappotto e il nome di suo fratello ancora segnato in un registro di debiti.
USER: Parlo con il battelliere e gli chiedo un passaggio senza farmi notare.
ASSISTANT: Il battelliere accetta metà pagamento, ma nota il sigillo spezzato sul tuo polso e capisce che stai scappando da qualcuno.
USER: Seguo la pista verso il Mercato della Marea.
ASSISTANT: Nel mercato trovi Lyanna Voss, che ti avverte che le Guardie di Marea stanno cercando una reliquia campanaria scomparsa all'alba.
USER: Le mostro la reliquia e le chiedo se sa aprirla.
ASSISTANT: Lyanna riconosce un sigillo del Coro del Sale e ti porta in un magazzino allagato dove un vecchio altare di sale vibra a ogni rintocco sommerso.
USER: Tocco la reliquia con la punta del coltello.
ASSISTANT: La reliquia si apre appena e ti mostra un frammento di memoria: tuo fratello inginocchiato davanti a una campana nera, con le Guardie che bussano alla porta.
USER: Scappiamo prima che ci trovino.
ASSISTANT: Durante la fuga un ponte cede, perdi il sacchetto di sale, ma Lyanna ti salva da una piena improvvisa e decide di aiutarti a raggiungere l'Old Harbor.
USER: Chiudo il capitolo con la promessa di scoprire perché mio fratello era legato al Coro del Sale.
ASSISTANT: La notte si chiude con le campane sommerse che tornano a suonare e un nuovo debito inciso sul tuo nome.
`)

	return benchmarkCase{
		ID:          "chapter-summary",
		Title:       "Chapter title + summary",
		Category:    "summary",
		Weight:      0.15,
		Temperature: 0.3,
		MaxTokens:   900,
		Messages: []ai.Message{
			{
				Role: ai.RoleSystem,
				Content: prompts.ChapterSummarySystem(
					"italiano",
					"fantasy urbano malinconico con prosa elegante e concreta",
					"Privilegia chiarezza, ritmo e continuita emotiva.",
				),
			},
			{Role: ai.RoleUser, Content: prompts.ChapterSummaryUser(transcript)},
		},
		Schema: ai.ChapterSummaryResponseFormat(),
		Eval:   evaluateChapterSummary,
	}
}

func evaluateStoryDefinition(raw string) caseEvaluation {
	points := map[string]float64{
		"json_block":          10,
		"json_only":           5,
		"parseable":           15,
		"required_fields":     10,
		"authoring_fields":    10,
		"setting_fields":      15,
		"setting_counts":      15,
		"stats_schema_shape":  10,
		"stats_schema_counts": 10,
	}
	eval := baseJSONEvaluation(raw)
	score := 0.0
	if eval.JSONValid {
		score += points["json_block"]
	}
	if eval.JSONOnly {
		score += points["json_only"]
	}
	if !eval.JSONValid {
		eval.Score = score
		eval.Breakdown = points
		return eval
	}

	var def engine.StoryDefinition
	if err := json.Unmarshal([]byte(eval.RawJSON), &def); err != nil {
		eval.Notes = append(eval.Notes, "JSON valido ma non parseabile come StoryDefinition: "+err.Error())
		eval.Score = score
		eval.Breakdown = points
		return eval
	}
	eval.Parsed = true
	score += points["parseable"]

	if def.Name != "" && def.Description != "" && def.Genre != "" && def.Tone != "" {
		score += points["required_fields"]
	} else {
		eval.Notes = append(eval.Notes, "Mancano campi top-level richiesti")
	}

	if strings.TrimSpace(def.Language) != "" && strings.TrimSpace(def.WritingStyle) != "" {
		score += points["authoring_fields"]
	} else {
		eval.Notes = append(eval.Notes, "Mancano language e/o writing_style nella story definition")
	}

	settingFilled := 0
	for _, v := range []string{
		def.Setting.WorldName,
		def.Setting.Era,
		def.Setting.Geography,
		def.Setting.MagicSystem,
		def.Setting.TechnologyLevel,
		def.Setting.Society,
	} {
		if strings.TrimSpace(v) != "" {
			settingFilled++
		}
	}
	if settingFilled == 6 {
		score += points["setting_fields"]
	} else {
		eval.Notes = append(eval.Notes, fmt.Sprintf("Setting incompleto: %d/6 campi principali", settingFilled))
	}

	if inRange(len(def.Setting.Rules), 4, 6) &&
		inRange(len(def.Setting.Factions), 2, 4) &&
		inRange(len(def.Setting.Cultures), 2, 3) &&
		inRange(len(def.Setting.Dangers), 3, 5) {
		score += points["setting_counts"]
	} else {
		eval.Notes = append(eval.Notes, fmt.Sprintf("Liste fuori target: rules=%d factions=%d cultures=%d dangers=%d",
			len(def.Setting.Rules), len(def.Setting.Factions), len(def.Setting.Cultures), len(def.Setting.Dangers)))
	}

	if len(def.StatsSchema.Vitals) > 0 && len(def.StatsSchema.Attributes) > 0 && def.StatsSchema.Currency != nil {
		score += points["stats_schema_shape"]
	} else {
		eval.Notes = append(eval.Notes, "Stats schema incompleto")
	}

	if len(def.StatsSchema.Vitals) >= 2 &&
		inRange(len(def.StatsSchema.Attributes), 6, 10) &&
		len(def.StatsSchema.Secondary) >= 1 {
		score += points["stats_schema_counts"]
	} else {
		eval.Notes = append(eval.Notes, fmt.Sprintf("Conteggi stats fuori target: vitals=%d attributes=%d secondary=%d",
			len(def.StatsSchema.Vitals), len(def.StatsSchema.Attributes), len(def.StatsSchema.Secondary)))
	}

	eval.Score = score
	eval.Breakdown = points
	return eval
}

func evaluateNarrativeBasic(raw string) caseEvaluation {
	points := map[string]float64{
		"json_block":     10,
		"json_only":      5,
		"parseable":      15,
		"narrative":      20,
		"choices":        20,
		"mood":           10,
		"location":       10,
		"italian_output": 10,
	}
	eval := baseJSONEvaluation(raw)
	score := 0.0
	if eval.JSONValid {
		score += points["json_block"]
	}
	if eval.JSONOnly {
		score += points["json_only"]
	}
	if !eval.JSONValid {
		eval.Score = score
		eval.Breakdown = points
		return eval
	}

	var nr engine.NarrativeResponse
	if err := json.Unmarshal([]byte(eval.RawJSON), &nr); err != nil {
		eval.Notes = append(eval.Notes, "JSON valido ma non parseabile come NarrativeResponse: "+err.Error())
		eval.Score = score
		eval.Breakdown = points
		return eval
	}
	eval.Parsed = true
	score += points["parseable"]

	if strings.TrimSpace(nr.Narrative) != "" {
		score += points["narrative"]
	}
	if validChoices(nr.Choices, 2, 4) {
		score += points["choices"]
	} else {
		eval.Notes = append(eval.Notes, fmt.Sprintf("Choices fuori contratto: %d", len(nr.Choices)))
	}
	if strings.TrimSpace(nr.Mood) != "" {
		score += points["mood"]
	}
	if strings.TrimSpace(nr.Location) != "" {
		score += points["location"]
	}
	if looksItalian(raw) {
		eval.LooksItalian = true
		score += points["italian_output"]
	} else {
		eval.Notes = append(eval.Notes, "Output non chiaramente in italiano")
	}

	eval.Score = score
	eval.Breakdown = points
	return eval
}

func evaluateNarrativeStructured(raw string) caseEvaluation {
	points := map[string]float64{
		"json_block":       10,
		"json_only":        5,
		"parseable":        15,
		"narrative":        10,
		"choices":          15,
		"dialogue_blocks":  15,
		"entities":         10,
		"event_callouts":   10,
		"semantic_choices": 5,
		"mood_location":    5,
	}
	eval := baseJSONEvaluation(raw)
	score := 0.0
	if eval.JSONValid {
		score += points["json_block"]
	}
	if eval.JSONOnly {
		score += points["json_only"]
	}
	if !eval.JSONValid {
		eval.Score = score
		eval.Breakdown = points
		return eval
	}

	var nr engine.NarrativeResponse
	if err := json.Unmarshal([]byte(eval.RawJSON), &nr); err != nil {
		eval.Notes = append(eval.Notes, "JSON valido ma non parseabile come NarrativeResponse: "+err.Error())
		eval.Score = score
		eval.Breakdown = points
		return eval
	}
	eval.Parsed = true
	score += points["parseable"]

	if strings.TrimSpace(nr.Narrative) != "" {
		score += points["narrative"]
	}
	if validChoices(nr.Choices, 2, 4) {
		score += points["choices"]
	}
	if countDialogueBlocks(nr.DialogueBlocks) >= 2 {
		score += points["dialogue_blocks"]
	} else {
		eval.Notes = append(eval.Notes, "dialogue_blocks insufficienti o assenti")
	}
	if mentionsEntity(nr.EntitiesMentioned, "Lyanna") || mentionsEntity(nr.EntitiesMentioned, "Old Harbor") {
		score += points["entities"]
	} else {
		eval.Notes = append(eval.Notes, "entities_mentioned non include Lyanna/Old Harbor")
	}
	if len(nr.EventCallouts) >= 1 {
		score += points["event_callouts"]
	} else {
		eval.Notes = append(eval.Notes, "event_callouts assenti")
	}
	if semanticChoiceCount(nr.Choices) >= 2 {
		score += points["semantic_choices"]
	} else {
		eval.Notes = append(eval.Notes, "semantic metadata scarso sulle choices")
	}
	if strings.TrimSpace(nr.Mood) != "" && strings.TrimSpace(nr.Location) != "" {
		score += points["mood_location"]
	}

	eval.Score = score
	eval.Breakdown = points
	return eval
}

func evaluateChallengeNarrative(raw string) caseEvaluation {
	points := map[string]float64{
		"json_block":      10,
		"json_only":       5,
		"parseable":       15,
		"narrative":       10,
		"choices":         15,
		"challenges":      20,
		"challenge_shape": 15,
		"mood_location":   10,
	}
	eval := baseJSONEvaluation(raw)
	score := 0.0
	if eval.JSONValid {
		score += points["json_block"]
	}
	if eval.JSONOnly {
		score += points["json_only"]
	}
	if !eval.JSONValid {
		eval.Score = score
		eval.Breakdown = points
		return eval
	}

	var nr engine.NarrativeResponse
	if err := json.Unmarshal([]byte(eval.RawJSON), &nr); err != nil {
		eval.Notes = append(eval.Notes, "JSON valido ma non parseabile come NarrativeResponse: "+err.Error())
		eval.Score = score
		eval.Breakdown = points
		return eval
	}
	eval.Parsed = true
	score += points["parseable"]

	if strings.TrimSpace(nr.Narrative) != "" {
		score += points["narrative"]
	}
	if validChoices(nr.Choices, 2, 4) {
		score += points["choices"]
	}
	if len(nr.Challenges) >= 1 {
		score += points["challenges"]
	} else {
		eval.Notes = append(eval.Notes, "challenges assente")
	}
	if validChallengeShape(nr.Challenges) {
		score += points["challenge_shape"]
	} else {
		eval.Notes = append(eval.Notes, "challenge presente ma con shape debole o incompleta")
	}
	if strings.TrimSpace(nr.Mood) != "" && strings.TrimSpace(nr.Location) != "" {
		score += points["mood_location"]
	}

	eval.Score = score
	eval.Breakdown = points
	return eval
}

func evaluateChapterSummary(raw string) caseEvaluation {
	points := map[string]float64{
		"json_block":         15,
		"json_only":          5,
		"parseable":          15,
		"title_present":      10,
		"title_word_count":   10,
		"summary_word_count": 20,
		"summary_shape":      10,
		"italian_output":     5,
		"coverage":           10,
	}
	eval := baseJSONEvaluation(raw)
	score := 0.0
	if eval.JSONValid {
		score += points["json_block"]
	}
	if eval.JSONOnly {
		score += points["json_only"]
	}
	if !eval.JSONValid {
		eval.Score = score
		eval.Breakdown = points
		return eval
	}

	var summary struct {
		Title   string `json:"title"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(eval.RawJSON), &summary); err != nil {
		eval.Notes = append(eval.Notes, "JSON valido ma non parseabile come chapter summary: "+err.Error())
		eval.Score = score
		eval.Breakdown = points
		return eval
	}
	eval.Parsed = true
	score += points["parseable"]

	if strings.TrimSpace(summary.Title) != "" {
		score += points["title_present"]
	}
	titleWords := wordCount(summary.Title)
	if inRange(titleWords, 3, 6) {
		score += points["title_word_count"]
	} else {
		eval.Notes = append(eval.Notes, fmt.Sprintf("Titolo fuori target: %d parole", titleWords))
	}

	summaryWords := wordCount(summary.Summary)
	eval.WordCount = summaryWords
	if inRange(summaryWords, 200, 400) {
		score += points["summary_word_count"]
	} else {
		eval.Notes = append(eval.Notes, fmt.Sprintf("Summary fuori target: %d parole", summaryWords))
	}
	if strings.Count(summary.Summary, ".") >= 3 || strings.Count(summary.Summary, "\n") >= 1 {
		score += points["summary_shape"]
	}
	if looksItalian(summary.Summary) {
		eval.LooksItalian = true
		score += points["italian_output"]
	}
	if coverageCount(strings.ToLower(summary.Summary), []string{"nerea", "lyanna", "reliquia", "guardie di marea", "old harbor", "coro del sale"}) >= 4 {
		score += points["coverage"]
	} else {
		eval.Notes = append(eval.Notes, "Copertura limitata di eventi/entità attese")
	}

	eval.Score = score
	eval.Breakdown = points
	return eval
}

func baseJSONEvaluation(raw string) caseEvaluation {
	eval := caseEvaluation{Breakdown: map[string]float64{}}
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

func (c *client) complete(ctx context.Context, model string, bc benchmarkCase, mode benchmarkMode, meta modelMetadata) (string, string, usage, time.Duration, error) {
	body := apiRequest{
		Model:       model,
		Messages:    bc.Messages,
		Temperature: bc.Temperature,
		MaxTokens:   bc.MaxTokens,
	}
	if responseFormat := responseFormatForMode(mode, bc, meta); responseFormat != nil {
		body.ResponseFormat = responseFormat
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", "", usage{}, 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", usage{}, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://homelab.local/oneday-benchmark")
	req.Header.Set("X-Title", "OneDay Benchmark")

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", usage{}, 0, err
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", usage{}, duration, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", usage{}, duration, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed apiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", "", usage{}, duration, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", "", usage{}, duration, fmt.Errorf("no choices returned")
	}

	return parsed.Choices[0].Message.Content, parsed.Model, parsed.Usage, duration, nil
}

func responseFormatForMode(mode benchmarkMode, bc benchmarkCase, meta modelMetadata) *ai.ResponseFormat {
	switch mode {
	case modePrompt:
		return nil
	case modeJSONObject:
		if meta.SupportsResponseFormat {
			return ai.NewJSONObjectResponseFormat()
		}
	case modeJSONSchema:
		if meta.SupportsStructured && bc.Schema != nil {
			return bc.Schema
		}
	}
	return nil
}

func (c *client) fetchModelMetadata(models []string) map[string]modelMetadata {
	out := map[string]modelMetadata{}
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

	var payload modelCatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return out
	}

	index := map[string]modelCatalogEntry{}
	for _, entry := range payload.Data {
		index[entry.ID] = entry
	}

	for _, model := range models {
		entry, ok := index[model]
		if !ok {
			out[model] = modelMetadata{ID: model}
			continue
		}
		out[model] = modelMetadata{
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

func estimateCostUSD(meta modelMetadata, usage usage) float64 {
	return meta.PromptCostPerTokenUSD*float64(usage.PromptTokens) +
		meta.OutputCostPerTokenUSD*float64(usage.CompletionTokens)
}

func semanticChoiceCount(choices []engine.Choice) int {
	count := 0
	for _, c := range choices {
		if c.Intent != "" || c.Risk != "" || c.Scope != "" || c.Certainty != "" || len(c.RelatedStats) > 0 {
			count++
		}
	}
	return count
}

func validChoices(choices []engine.Choice, min, max int) bool {
	if !inRange(len(choices), min, max) {
		return false
	}
	seen := map[int]bool{}
	for _, c := range choices {
		if c.ID == 0 || strings.TrimSpace(c.Text) == "" || seen[c.ID] {
			return false
		}
		seen[c.ID] = true
	}
	return true
}

func countDialogueBlocks(blocks []engine.DialogueBlock) int {
	valid := 0
	for _, b := range blocks {
		if strings.TrimSpace(b.Speaker) != "" && strings.TrimSpace(b.Text) != "" {
			valid++
		}
	}
	return valid
}

func mentionsEntity(entities []engine.EntityMention, needle string) bool {
	needle = strings.ToLower(needle)
	for _, e := range entities {
		if strings.Contains(strings.ToLower(e.Name), needle) {
			return true
		}
	}
	return false
}

func validChallengeShape(challenges []*engine.ChallengeSpec) bool {
	if len(challenges) == 0 {
		return false
	}
	c := challenges[0]
	if c == nil || c.Type == "" {
		return false
	}
	switch c.Type {
	case engine.ChallengeStatCheck:
		return c.Stat != "" && c.Difficulty > 0
	case engine.ChallengeDiceRoll:
		return c.Difficulty > 0
	case engine.ChallengeItemCheck:
		return c.Item != ""
	case engine.ChallengeSkillCheck:
		return c.Skill != ""
	case engine.ChallengeRelCheck:
		return c.NPCName != "" && c.Disposition != 0
	case engine.ChallengeMiniGame:
		return c.MiniGame != ""
	default:
		return false
	}
}

func coverageCount(text string, needles []string) int {
	found := 0
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			found++
		}
	}
	return found
}

func looksItalian(text string) bool {
	lower := " " + strings.ToLower(text) + " "
	hits := 0
	for _, token := range []string{" il ", " la ", " che ", " non ", " con ", " una ", " del ", " della ", " nel ", " per ", " mentre ", " guarda ", " campane "} {
		if strings.Contains(lower, token) {
			hits++
		}
	}
	return hits >= 3
}

func writeJSONReport(path string, report benchmarkReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeMarkdownReport(path, jsonName string, report benchmarkReport) error {
	var b strings.Builder
	b.WriteString("# OneDay Model Benchmark\n\n")
	b.WriteString(fmt.Sprintf("- Generated: `%s`\n", report.GeneratedAt))
	b.WriteString(fmt.Sprintf("- Base URL: `%s`\n", report.BaseURL))
	b.WriteString(fmt.Sprintf("- Mode: `%s`\n", report.Mode))
	b.WriteString("- Scoring: `compatibility only` from the command output. Narrative quality is reviewed separately by hand.\n")
	b.WriteString(fmt.Sprintf("- Raw JSON artifact: `%s`\n\n", jsonName))

	b.WriteString("## Leaderboard\n\n")
	b.WriteString("| Model | Compat Score | Success Rate | Avg Seconds | Avg Cost | Context | Max Out | Avg Tok/s |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, summary := range report.Summaries {
		b.WriteString(fmt.Sprintf("| `%s` | %.1f | %.0f%% | %.3f | $%.6f | %d | %d | %.1f |\n",
			summary.Model,
			summary.WeightedCompatScore,
			summary.SuccessRate,
			summary.AverageLatencySec,
			summary.AverageCostUSD,
			summary.Metadata.ContextLength,
			summary.Metadata.MaxCompletionTokens,
			summary.AverageTokensPerSec,
		))
	}
	b.WriteString("\n")

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
			b.WriteString(fmt.Sprintf("- Compat score: `%.1f/100`\n", result.Compatibility.Score))
			b.WriteString(fmt.Sprintf("- Duration: `%.3f s`\n", result.DurationSeconds))
			b.WriteString(fmt.Sprintf("- Completion tokens: `%d`\n", result.CompletionTokens))
			b.WriteString(fmt.Sprintf("- Prompt tokens: `%d`\n", result.PromptTokens))
			b.WriteString(fmt.Sprintf("- Estimated cost: `$%.6f`\n", result.CostUSD))
			b.WriteString(fmt.Sprintf("- Throughput: `%.1f tok/s`, `%.1f char/s`\n", result.TokensPerSecond, result.CharsPerSecond))
			if len(result.Compatibility.Notes) > 0 {
				b.WriteString("- Notes:\n")
				for _, note := range result.Compatibility.Notes {
					b.WriteString(fmt.Sprintf("  - %s\n", note))
				}
			}
			b.WriteString("\n```text\n")
			b.WriteString(truncate(result.RawResponse, 1800))
			b.WriteString("\n```\n\n")
		}
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
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

func inRange(n, min, max int) bool {
	return n >= min && n <= max
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
