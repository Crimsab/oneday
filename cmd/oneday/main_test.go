package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/game/contracts"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/setup"
	"github.com/crimsab/oneday/internal/storage"
)

func TestGatewayRequestIDIsSanitizedAndAddedToTelemetry(t *testing.T) {
	t.Setenv("ONEDAY_REQUEST_ID", " request-123\nunsafe:value ")
	if got := gatewayRequestID(); got != "request-123unsafevalue" {
		t.Fatalf("gatewayRequestID = %q", got)
	}
	metadata := ai.TelemetryFromContext(gatewayContext())
	if metadata.TraceID != "request-123unsafevalue" || metadata.SafeMetadata["request_id"] != metadata.TraceID {
		t.Fatalf("gateway telemetry metadata = %#v", metadata)
	}
}

func TestWantsVersion(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{args: nil, want: false},
		{args: []string{"--version"}, want: true},
		{args: []string{"version"}, want: true},
		{args: []string{"play"}, want: false},
	}

	for _, tc := range tests {
		if got := wantsVersion(tc.args); got != tc.want {
			t.Fatalf("wantsVersion(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

func TestHelpDoesNotRequireConfiguration(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"setup", "--help"}} {
		if !wantsHelp(args) {
			t.Fatalf("wantsHelp(%v) = false", args)
		}
	}
	if wantsHelp([]string{"play"}) {
		t.Fatal("wantsHelp(play) = true")
	}

	var output bytes.Buffer
	printUsage(&output)
	for _, expected := range []string{"Start the terminal client", "story-packs list", "docs/first-story.md"} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Fatalf("help output missing %q:\n%s", expected, output.String())
		}
	}
}

func TestWantsOperatorCommands(t *testing.T) {
	if !wantsConfigShowSafe([]string{"config", "show", "--safe"}) {
		t.Fatal("expected config show --safe")
	}
	if !wantsRAGBenchmark([]string{"rag", "benchmark"}) {
		t.Fatal("expected rag benchmark")
	}
	if !wantsRAGReindex([]string{"rag", "reindex"}) {
		t.Fatal("expected rag reindex")
	}
	if !wantsStoryPacksList([]string{"story-packs", "list"}) {
		t.Fatal("expected story-packs list")
	}
	if !wantsExport([]string{"export"}) {
		t.Fatal("expected export")
	}
	if !wantsGatewayModelSettings([]string{"gateway-model-settings"}) {
		t.Fatal("expected gateway-model-settings")
	}
	if !wantsGatewayModelSettingsUpdate([]string{"gateway-model-settings-update"}) {
		t.Fatal("expected gateway-model-settings-update")
	}
	if !wantsGatewayStoryCreate([]string{"gateway-story-create"}) {
		t.Fatal("expected gateway-story-create")
	}
	if !wantsGatewayStoryWizard([]string{"gateway-story-wizard"}) {
		t.Fatal("expected gateway-story-wizard")
	}
	if !wantsGatewayStoryEnhance([]string{"gateway-story-enhance"}) {
		t.Fatal("expected gateway-story-enhance")
	}
	if !wantsGatewayTimeline([]string{"gateway-timeline"}) {
		t.Fatal("expected gateway-timeline")
	}
	if !wantsGatewayCraft([]string{"gateway-craft"}) {
		t.Fatal("expected gateway-craft")
	}
}

func TestConfigLocaleCommandPersistsWithoutAffectingStoryOrTTSDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("ONEDAY_CONFIG", path)
	var out bytes.Buffer
	if err := runConfigLocale([]string{"config", "locale", "it"}, &out); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Interface.Locale != "it" {
		t.Fatalf("locale=%q", cfg.Interface.Locale)
	}
	if cfg.AI.TTS.Cloud.Languages != nil || cfg.AI.TTS.Local.Languages != nil {
		t.Fatal("interface locale changed TTS languages")
	}
	story := storage.Story{Language: "en-US"}
	if story.Language != "en-US" {
		t.Fatal("interface locale changed story language")
	}
}

func TestItalianSetupTranscriptStrings(t *testing.T) {
	loc := appi18n.New(appi18n.Italian)
	transcript := strings.Join([]string{
		loc.T("cli.setup_title"), loc.T("cli.choose_language"), loc.T("cli.choose_provider"),
		loc.SetupPresentation("provider_codex", ""), loc.SetupPresentation("rag_title", ""),
		loc.SetupPresentation("rag_off", ""),
	}, "\n")
	for _, want := range []string{"Prima configurazione", "lingua dell'interfaccia", "provider IA", "usa `codex login`", "embedding RAG", "Disabilita RAG"} {
		if !strings.Contains(transcript, want) {
			t.Errorf("setup transcript missing %q:\n%s", want, transcript)
		}
	}
}

func TestPublicCLIResidualStringsUseCatalog(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, literal := range []string{
		`fmt.Println("No story packs found.")`,
		`fmt.Println("Usage: oneday rag reindex`,
		`fmt.Println("OneDay doctor")`,
		`fmt.Print("Reasoning off/none/minimal`,
		`fmt.Println("OneDay config (safe)")`,
	} {
		if strings.Contains(text, literal) {
			t.Errorf("public CLI presentation bypasses catalog: %s", literal)
		}
	}
}

func TestGatewayTurnEventPhase(t *testing.T) {
	if got := gatewayTurnEventPhase(contracts.TurnEvent{ID: "turn-key:live:1", Type: contracts.EventTurnStarted}); got != "live" {
		t.Fatalf("live phase = %q", got)
	}
	if got := gatewayTurnEventPhase(contracts.TurnEvent{ID: "turn-key:2", Type: contracts.EventNarrativeDelta}); got != "live" {
		t.Fatalf("delta phase = %q", got)
	}
	if got := gatewayTurnEventPhase(contracts.TurnEvent{ID: "turn-key:3", Type: contracts.EventTurnCommitted}); got != "final" {
		t.Fatalf("final phase = %q", got)
	}
}

func TestProviderConsistencyWarnings(t *testing.T) {
	cfg := config.Default()
	cfg.AI.LiteLLM.Enabled = true
	cfg.AI.LiteLLM.APIKey = ""
	warnings := providerConsistencyWarnings(cfg)
	if len(warnings) == 0 {
		t.Fatal("expected missing LiteLLM key warning")
	}
}

func TestDiscoverStoryPacks(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "plugins", "examples")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pack.yaml"), []byte("id: pack\n"), 0644); err != nil {
		t.Fatal(err)
	}
	packs, err := discoverStoryPacks([]string{root})
	if err != nil {
		t.Fatalf("discoverStoryPacks: %v", err)
	}
	if len(packs) != 1 || filepath.Base(packs[0]) != "pack.yaml" {
		t.Fatalf("packs = %#v", packs)
	}
}

func TestValidateStoryPack(t *testing.T) {
	dir := t.TempDir()
	pack := filepath.Join(dir, "pack.yaml")
	valid := "id: pack\nname: Pack\ndescription: Demo\nstats_schema:\n  attributes: [{key: insight, label: Insight, starting: 2}]\nchallenge_pools:\n  clues:\n    definitions: [{id: clue, kind: deduction, difficulty: 40, answers: [mara]}]\n"
	if err := os.WriteFile(pack, []byte(valid), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateStoryPack(pack); err != nil {
		t.Fatalf("validateStoryPack: %v", err)
	}
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("id: bad\nname: Bad\ndescription: Bad\nchallenge_pools:\n  clues:\n    definitions: [{id: clue, kind: deduction, difficulty: 40}]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateStoryPack(bad); err == nil {
		t.Fatal("expected invalid story pack")
	}
}

func TestListStoryPacksFailsWhenAnyPackIsInvalid(t *testing.T) {
	dir := t.TempDir()
	valid := "id: valid\nname: Valid\ndescription: Valid pack\n"
	if err := os.WriteFile(filepath.Join(dir, "valid.yaml"), []byte(valid), 0644); err != nil {
		t.Fatal(err)
	}
	invalid := "id: invalid\nname: Invalid\ndescription: Invalid pack\nchallenge_pools:\n  clues:\n    definitions: [{id: clue, kind: deduction, difficulty: 40}]\n"
	if err := os.WriteFile(filepath.Join(dir, "invalid.yaml"), []byte(invalid), 0644); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	err := listStoryPacks([]string{dir}, &output)
	if err == nil || !strings.Contains(err.Error(), "1 invalid story pack") {
		t.Fatalf("listStoryPacks error = %v, output:\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "valid.yaml") || !strings.Contains(output.String(), "invalid:") {
		t.Fatalf("unexpected list output:\n%s", output.String())
	}
}

func TestSetupConfigForChoice(t *testing.T) {
	tests := []struct {
		name           string
		choice         string
		wantProvider   string
		wantCodex      bool
		wantLiteLLM    bool
		wantOpenRouter bool
		wantRAG        bool
		wantKey        string
	}{
		{
			name:         "codex",
			choice:       "1",
			wantProvider: "codex",
			wantCodex:    true,
			wantRAG:      false,
		},
		{
			name:         "litellm",
			choice:       "2",
			wantProvider: "litellm",
			wantLiteLLM:  true,
			wantRAG:      false,
			wantKey:      "${ONEDAY_LITELLM_API_KEY}",
		},
		{
			name:           "openrouter",
			choice:         "3",
			wantProvider:   "openrouter",
			wantOpenRouter: true,
			wantRAG:        false,
			wantKey:        "${ONEDAY_OPENROUTER_API_KEY}",
		},
		{
			name:         "codex local rag",
			choice:       "4",
			wantProvider: "codex",
			wantCodex:    true,
			wantRAG:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := setupConfigForChoice(config.Default(), tc.choice)
			if err != nil {
				t.Fatalf("setupConfigForChoice: %v", err)
			}
			if cfg.AI.Codex.Enabled {
				cfg.AI.Codex.Model = "test-codex-model"
				cfg.AI.Generation.UtilityModel = "test-codex-model"
			}
			if cfg.AI.LiteLLM.Enabled {
				cfg.AI.LiteLLM.DefaultModel = "test-litellm-model"
				cfg.AI.Generation.UtilityModel = "test-litellm-model"
			}
			if cfg.AI.OpenRouter.Enabled {
				cfg.AI.OpenRouter.DefaultModel = "test-openrouter-model"
				cfg.AI.Generation.UtilityModel = "test-openrouter-model"
			}
			if cfg.RAG.Enabled && cfg.AI.Embedding.Provider == "local" {
				cfg.AI.Embedding.Local.Model = "test-local-embedding-model"
			} else if cfg.RAG.Enabled {
				cfg.AI.Embedding.Model = "test-embedding-model"
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if cfg.AI.ProviderPriority[0] != tc.wantProvider {
				t.Fatalf("first provider = %q, want %q", cfg.AI.ProviderPriority[0], tc.wantProvider)
			}
			if cfg.AI.Codex.Enabled != tc.wantCodex {
				t.Fatalf("Codex enabled = %v, want %v", cfg.AI.Codex.Enabled, tc.wantCodex)
			}
			if cfg.AI.LiteLLM.Enabled != tc.wantLiteLLM {
				t.Fatalf("LiteLLM enabled = %v, want %v", cfg.AI.LiteLLM.Enabled, tc.wantLiteLLM)
			}
			if cfg.AI.OpenRouter.Enabled != tc.wantOpenRouter {
				t.Fatalf("OpenRouter enabled = %v, want %v", cfg.AI.OpenRouter.Enabled, tc.wantOpenRouter)
			}
			if cfg.RAG.Enabled != tc.wantRAG {
				t.Fatalf("RAG enabled = %v, want %v", cfg.RAG.Enabled, tc.wantRAG)
			}
			if tc.wantKey != "" && cfg.AI.LiteLLM.APIKey != tc.wantKey && cfg.AI.OpenRouter.APIKey != tc.wantKey {
				t.Fatalf("expected placeholder key %q in setup config", tc.wantKey)
			}
			if tc.choice == "4" {
				if cfg.AI.Embedding.Provider != "local" || cfg.AI.Embedding.Local.Type != "ollama" {
					t.Fatalf("local RAG config not selected: %#v", cfg.AI.Embedding)
				}
			}
		})
	}
}

func TestSetupConfigForChoicePreservesExistingSettings(t *testing.T) {
	cfg := config.Default()
	cfg.DataDir = "./kept-data"
	cfg.AI.ImageGeneration.AutoGenerate = true
	cfg.AI.ImageGeneration.Model = "kept-image-model"
	cfg.AI.ImageGeneration.MapIconModel = "kept-map-model"
	cfg.AI.TTS.Local.Enabled = true
	cfg.AI.TTS.Local.BaseURL = "http://tts.example.test"
	cfg.Game.RewardBudget = "generous"

	next, err := setupConfigForChoice(cfg, "1")
	if err != nil {
		t.Fatal(err)
	}
	if next.DataDir != cfg.DataDir || next.AI.ImageGeneration.Model != cfg.AI.ImageGeneration.Model || next.AI.TTS.Local.BaseURL != cfg.AI.TTS.Local.BaseURL || next.Game.RewardBudget != cfg.Game.RewardBudget {
		t.Fatalf("setup choice clobbered existing settings: %#v", next)
	}
}

func TestSetupReconfigureUsesExplicitConfigPathAndPreservesSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.yaml")
	cfg := config.Default()
	cfg.DataDir = "./kept-data"
	cfg.Game.RewardBudget = "generous"
	data, err := config.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ONEDAY_CONFIG", path)

	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := output.WriteString("\n1\ntest-codex-model\n\n0\n"); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	os.Stdin = input
	defer func() {
		os.Stdin = oldStdin
		_ = input.Close()
	}()

	if err := runSetup([]string{"setup", "--reconfigure"}); err != nil {
		t.Fatal(err)
	}
	saved, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.DataDir != cfg.DataDir || saved.Game.RewardBudget != cfg.Game.RewardBudget || saved.AI.Codex.Model != "test-codex-model" {
		t.Fatalf("unexpected setup result: %#v", saved)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
	if _, err := os.Stat(path + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("setup must not create a config backup: %v", err)
	}
}

func TestWantsSetupForce(t *testing.T) {
	if !wantsSetupForce([]string{"setup", "--reconfigure"}) {
		t.Fatal("expected --reconfigure to force setup")
	}
	if !wantsSetupForce([]string{"setup", "--force"}) {
		t.Fatal("expected --force to force setup")
	}
	if wantsSetupForce([]string{"setup"}) {
		t.Fatal("plain setup should not force")
	}
}

func TestWantsJSON(t *testing.T) {
	if !wantsJSON([]string{"doctor", "--json"}) {
		t.Fatal("expected --json")
	}
	if wantsJSON([]string{"doctor"}) {
		t.Fatal("plain doctor should not request json")
	}
}

func TestDoctorJSONAndTextShareReadinessProbesAndRequiredExit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.yaml")
	cfg := config.Default()
	cfg.AI.Codex.Enabled = true
	cfg.AI.Codex.Model = "test"
	cfg.AI.Generation.UtilityModel = "test"
	cfg.DataDir = t.TempDir()
	data, err := config.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ONEDAY_CONFIG", path)
	deps := setup.Dependencies{Narrative: func(context.Context, config.Config) error { return errors.New("secret narrative failure") }}
	var jsonOut bytes.Buffer
	err = runDoctorTo([]string{"doctor", "--json"}, &jsonOut, deps)
	if !errors.Is(err, errDoctorRequiredFailure) {
		t.Fatalf("doctor error = %v", err)
	}
	if strings.Contains(jsonOut.String(), "secret") || !strings.Contains(jsonOut.String(), "NARRATIVE_UNAVAILABLE") {
		t.Fatalf("unexpected JSON: %s", jsonOut.String())
	}
	var textOut bytes.Buffer
	err = runDoctorTo([]string{"doctor"}, &textOut, deps)
	if !errors.Is(err, errDoctorRequiredFailure) {
		t.Fatalf("doctor text error = %v", err)
	}
	if strings.Contains(jsonOut.String(), path) || strings.Contains(textOut.String(), path) {
		t.Fatalf("doctor output leaked private config path; json=%s text=%s", jsonOut.String(), textOut.String())
	}
	if !strings.Contains(jsonOut.String(), `"config_source": "ONEDAY_CONFIG"`) || !strings.Contains(textOut.String(), "NARRATIVE_UNAVAILABLE") || !strings.Contains(textOut.String(), "ONEDAY_CONFIG") {
		t.Fatalf("unexpected text: %s", textOut.String())
	}
}

func TestDoctorDoesNotExposePrivateConfigPathWhenConfigurationIsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not: [valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ONEDAY_CONFIG", path)
	var output bytes.Buffer
	err := runDoctorTo([]string{"doctor"}, &output, setup.Dependencies{})
	if err == nil {
		t.Fatal("expected invalid configuration error")
	}
	if strings.Contains(output.String(), path) || strings.Contains(err.Error(), path) {
		t.Fatalf("doctor exposed private path; output=%q error=%q", output.String(), err)
	}
}

func TestDoctorUsesExplicitDatabasePathWithoutExposingIt(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.AI.Codex.Enabled = true
	cfg.AI.Codex.Model = "test"
	cfg.AI.Generation.UtilityModel = "test"
	cfg.DataDir = filepath.Join(root, "configured-data")
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.yaml")
	data, err := config.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "private", "gateway-state.sqlite")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(databasePath, []byte("sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ONEDAY_CONFIG", configPath)
	t.Setenv("ONEDAY_DB_PATH", databasePath)

	deps := setup.Dependencies{
		Narrative: func(context.Context, config.Config) error { return nil },
		Embedding: func(context.Context, aifactory.EmbeddingProviderSpec) (int, error) { return 0, nil },
		HTTPGet:   func(context.Context, string) error { return nil },
		Stat:      os.Stat,
	}
	var output bytes.Buffer
	if err := runDoctorTo([]string{"doctor", "--json"}, &output, deps); err != nil {
		t.Fatalf("doctor error = %v", err)
	}
	if strings.Contains(output.String(), databasePath) || strings.Contains(output.String(), cfg.DataDir) {
		t.Fatalf("doctor leaked private path: %s", output.String())
	}
	var report setup.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	for _, probe := range report.Probes {
		if probe.Name == "storage" && probe.Code != "STORAGE_READY" {
			t.Fatalf("storage probe = %#v", probe)
		}
		if probe.Name == "backup" && probe.Code != "BACKUP_READY" {
			t.Fatalf("backup probe = %#v", probe)
		}
	}
}

func TestSetupWithExplicitConfigWritesAdjacentDotEnvFromDifferentWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	workingDir := filepath.Join(root, "elsewhere")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workingDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	data, err := config.Marshal(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workingDir)
	t.Setenv("ONEDAY_CONFIG", configPath)
	t.Setenv("ONEDAY_LITELLM_API_KEY", "")
	t.Setenv("ONEDAY_OPENROUTER_API_KEY", "")

	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := output.WriteString("\n2\ntest-litellm-model\n4\n0\n"); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	os.Stdin = input
	defer func() {
		os.Stdin = oldStdin
		_ = input.Close()
	}()

	if err := runSetup([]string{"setup", "--reconfigure"}); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(configDir, ".env")
	if got := resolveDotEnvPath(); got != envPath {
		t.Fatalf("dotenv path = %q, want %q", got, envPath)
	}
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf("expected adjacent dotenv: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workingDir, ".env")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected dotenv in working directory: %v", err)
	}
	const dotenvKey = "ONEDAY_SETUP_DOTENV_TEST"
	previous, wasSet := os.LookupEnv(dotenvKey)
	if err := os.Unsetenv(dotenvKey); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if wasSet {
			_ = os.Setenv(dotenvKey, previous)
			return
		}
		_ = os.Unsetenv(dotenvKey)
	}()
	if err := os.WriteFile(envPath, []byte(dotenvKey+"=loaded-from-config-directory\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(dotenvKey); got != "loaded-from-config-directory" {
		t.Fatalf("dotenv value = %q", got)
	}
}

func TestGatewayModelSettingsCommands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := config.Default()
	cfg.AI.ProviderPriority = []string{"codex", "litellm", "openrouter", "claude-code"}
	cfg.AI.Codex.Enabled = true
	cfg.AI.Codex.Model = "test-codex-model"
	data, err := config.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := runGatewayModelSettings(path, &out); err != nil {
		t.Fatalf("runGatewayModelSettings: %v", err)
	}
	var resp gatewayModelSettingsResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if resp.Settings == nil || resp.Settings.Active.Provider != "codex" {
		t.Fatalf("settings response = %#v", resp)
	}
	if resp.Settings.ConfigRevision == "" {
		t.Fatalf("settings response missing config revision = %#v", resp)
	}

	priority := []string{"litellm", "codex", "openrouter", "claude-code"}
	litellmEnabled := true
	litellmModel := "test-litellm-model-updated"
	update := config.ModelRoutingUpdate{
		BaseRevision:     resp.Settings.ConfigRevision,
		ProviderPriority: &priority,
		Providers: []config.ModelProviderUpdate{
			{ID: "litellm", Enabled: &litellmEnabled, Model: &litellmModel},
		},
	}
	input, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := runGatewayModelSettingsUpdate(path, bytes.NewReader(input), &out); err != nil {
		t.Fatalf("runGatewayModelSettingsUpdate: %v", err)
	}
	resp = gatewayModelSettingsResponse{}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if resp.Settings == nil || resp.Settings.Active.Provider != "litellm" || resp.Settings.Active.NarrativeModel != litellmModel {
		t.Fatalf("updated settings response = %#v", resp)
	}
	saved, err := config.LoadForEdit(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.AI.LiteLLM.DefaultModel != litellmModel {
		t.Fatalf("saved LiteLLM model = %q", saved.AI.LiteLLM.DefaultModel)
	}

	out.Reset()
	update.BaseRevision = "stale"
	input, err = json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	if err := runGatewayModelSettingsUpdate(path, bytes.NewReader(input), &out); err == nil {
		t.Fatal("expected stale update to return an error")
	}
	resp = gatewayModelSettingsResponse{}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode stale update: %v", err)
	}
	if resp.ErrorCode != config.ModelRoutingErrorStale {
		t.Fatalf("stale ErrorCode = %q, want %q", resp.ErrorCode, config.ModelRoutingErrorStale)
	}
}

func TestEnsureEnvFileCreatesPrivateTemplate(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ONEDAY_LITELLM_API_KEY", "")
	t.Setenv("ONEDAY_OPENROUTER_API_KEY", "")
	if err := ensureEnvFile(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(".env")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ONEDAY_LITELLM_API_KEY=\nONEDAY_OPENROUTER_API_KEY=\n" {
		t.Fatalf("env template=%q", data)
	}
	info, err := os.Stat(".env")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("env permissions=%#o want=0600", info.Mode().Perm())
	}
}

func TestEnsureEnvFileReportsInvalidPath(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir(".env", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ensureEnvFile(); err == nil {
		t.Fatal("ensureEnvFile accepted a directory")
	}
}
