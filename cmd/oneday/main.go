package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/providers"
	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/buildinfo"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui"
)

func main() {
	if wantsVersion(os.Args[1:]) {
		fmt.Println(buildinfo.Text("oneday"))
		return
	}
	if wantsSetup(os.Args[1:]) {
		if err := runSetup(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsDoctor(os.Args[1:]) {
		if err := runDoctor(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Doctor failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsConfigShowSafe(os.Args[1:]) {
		if err := runConfigShowSafe(); err != nil {
			fmt.Fprintf(os.Stderr, "Config show failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsRAGBenchmark(os.Args[1:]) {
		if err := runRAGBenchmark(); err != nil {
			fmt.Fprintf(os.Stderr, "RAG benchmark failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsRAGReindex(os.Args[1:]) {
		if err := runRAGReindex(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "RAG reindex failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsStoryPacksList(os.Args[1:]) {
		if err := runStoryPacksList(); err != nil {
			fmt.Fprintf(os.Stderr, "Story packs failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsExportFriend(os.Args[1:]) {
		if err := runExportFriend(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Friend export failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load .env: %v\n", err)
	}
	// Load config
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "oneday.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create AI router
	router, err := aifactory.NewRouterFromConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating AI router: %v\n", err)
		os.Exit(1)
	}

	// Start TUI
	app := tui.New(cfg, db, router)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func resolveDotEnvPath() string {
	const envName = ".env"

	if _, err := os.Stat(envName); err == nil {
		return envName
	}

	exePath, err := os.Executable()
	if err != nil {
		return envName
	}

	exeEnv := filepath.Join(filepath.Dir(exePath), envName)
	if _, err := os.Stat(exeEnv); err == nil {
		return exeEnv
	}

	return envName
}

func resolveConfigPath() string {
	const configName = "config.yaml"

	if _, err := os.Stat(configName); err == nil {
		return configName
	}

	exePath, err := os.Executable()
	if err != nil {
		return configName
	}

	exeConfig := filepath.Join(filepath.Dir(exePath), configName)
	if _, err := os.Stat(exeConfig); err == nil {
		return exeConfig
	}

	return configName
}

func wantsSetup(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "setup", "--setup":
			return true
		}
	}
	return false
}

func wantsSetupForce(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--force", "--reconfigure":
			return true
		}
	}
	return false
}

func wantsVersion(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--version", "version":
			return true
		}
	}
	return false
}

func wantsDoctor(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "doctor", "--doctor":
			return true
		}
	}
	return false
}

func wantsJSON(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
	}
	return false
}

func wantsConfigShowSafe(args []string) bool {
	return len(args) >= 3 && args[0] == "config" && args[1] == "show" && args[2] == "--safe"
}

func wantsRAGBenchmark(args []string) bool {
	return len(args) >= 2 && args[0] == "rag" && args[1] == "benchmark"
}

func wantsRAGReindex(args []string) bool {
	return len(args) >= 2 && args[0] == "rag" && args[1] == "reindex"
}

func wantsStoryPacksList(args []string) bool {
	return len(args) >= 2 && (args[0] == "story-packs" || args[0] == "storypacks") && args[1] == "list"
}

func wantsExportFriend(args []string) bool {
	return len(args) >= 2 && args[0] == "export" && args[1] == "--friend"
}

type localEmbeddingModel struct {
	ID          string
	Label       string
	Description string
	Dimensions  int
}

var localEmbeddingModels = []localEmbeddingModel{
	{ID: "bge-m3", Label: "bge-m3", Description: "Recommended default: strong multilingual/Italian support, reliable quality, moderate size", Dimensions: 1024},
	{ID: "nomic-embed-text", Label: "nomic-embed-text", Description: "Fast/lightweight: good for English/general notes, smaller download, lower memory", Dimensions: 768},
	{ID: "mxbai-embed-large", Label: "mxbai-embed-large", Description: "English retrieval quality: stronger for English docs, less ideal for multilingual text", Dimensions: 1024},
	{ID: "qwen3-embedding", Label: "qwen3-embedding", Description: "Quality/heavier: best when available locally, more resource-hungry; dimensions should be smoke-tested", Dimensions: 1024},
}

func runSetup(args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("OneDay first-time setup")
	fmt.Printf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	reportCommand("go", "version")
	reportCommand("codex", "--version")
	reportCommand("codex", "login", "status")
	reportCommand("claude", "--version")

	force := wantsSetupForce(args)
	if _, err := os.Stat("config.yaml"); err == nil && !force {
		fmt.Println("config.yaml already exists; leaving it in place.")
		fmt.Println("Run `oneday setup --reconfigure` or `oneday setup --force` to open the setup wizard again.")
		return nil
	}

	fmt.Println()
	fmt.Println("Choose AI provider:")
	fmt.Println("  1) Codex OAuth (experimental, uses local `codex login`)")
	fmt.Println("  2) LiteLLM / homelab proxy")
	fmt.Println("  3) OpenRouter")
	fmt.Println("  4) Codex OAuth + local RAG embeddings")
	fmt.Print("Selection [1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	cfg := config.Default()
	cfg, err := setupConfigForChoice(cfg, choice)
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		configureCodex(reader, &cfg)
		fmt.Println("If Codex is not logged in yet, run: codex login")
		fmt.Println("RAG: disabled, reason: no embedding-capable provider configured")
	case "2":
		ensureEnvFile()
		if err := configureRAGChoice(reader, &cfg); err != nil {
			return err
		}
	case "3":
		ensureEnvFile()
		if err := configureRAGChoice(reader, &cfg); err != nil {
			return err
		}
	case "4":
		configureCodex(reader, &cfg)
		if err := configureLocalRAG(reader, &cfg); err != nil {
			return err
		}
	}

	if err := cfg.Validate(); err != nil {
		return err
	}
	data, err := config.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile("config.yaml", data, 0600); err != nil {
		return err
	}
	fmt.Println("Wrote config.yaml")
	return nil
}

func configureCodex(reader *bufio.Reader, cfg *config.Config) {
	fmt.Print("Codex model [gpt-5.5]: ")
	model, _ := reader.ReadString('\n')
	if model = strings.TrimSpace(model); model != "" {
		cfg.AI.Codex.Model = model
	}
	fmt.Print("Reasoning off/none/minimal/low/medium/high/xhigh [off]: ")
	reasoning, _ := reader.ReadString('\n')
	if reasoning = strings.TrimSpace(reasoning); reasoning != "" {
		cfg.AI.Codex.Reasoning = reasoning
	}
}

func configureRAGChoice(reader *bufio.Reader, cfg *config.Config) error {
	fmt.Println()
	fmt.Println("Choose RAG embeddings:")
	fmt.Println("  1) Remote provider from current AI config")
	fmt.Println("  2) Local Ollama embeddings")
	fmt.Println("  3) Custom local embedding endpoint")
	fmt.Println("  4) Disable RAG")
	fmt.Print("Selection [1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}
	switch choice {
	case "1":
		cfg.RAG.Enabled = true
		cfg.AI.Embedding.Provider = "auto"
		return nil
	case "2":
		return configureLocalRAG(reader, cfg)
	case "3":
		return configureCustomLocalRAG(reader, cfg)
	case "4":
		cfg.RAG.Enabled = false
		return nil
	default:
		return fmt.Errorf("unknown RAG selection %q", choice)
	}
}

func configureLocalRAG(reader *bufio.Reader, cfg *config.Config) error {
	fmt.Println()
	fmt.Println("Local RAG model choices:")
	for i, model := range localEmbeddingModels {
		fmt.Printf("  %d) %s — %s (%d dimensions)\n", i+1, model.Label, model.Description, model.Dimensions)
	}
	fmt.Print("Selection [1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}
	model, err := localEmbeddingModelByChoice(choice)
	if err != nil {
		return err
	}

	cfg.RAG.Enabled = true
	cfg.RAG.Dimensions = model.Dimensions
	cfg.AI.Embedding.Provider = "local"
	cfg.AI.Embedding.Local.Enabled = true
	cfg.AI.Embedding.Local.Type = "ollama"
	cfg.AI.Embedding.Local.BaseURL = "http://127.0.0.1:11434"
	cfg.AI.Embedding.Local.Model = model.ID
	cfg.AI.Embedding.Local.Dimensions = model.Dimensions

	fmt.Printf("Use Ollama model %s at %s\n", model.ID, cfg.AI.Embedding.Local.BaseURL)
	if _, err := exec.LookPath("ollama"); err != nil {
		fmt.Println("Ollama CLI not found. Install from https://docs.ollama.com/linux or use custom local endpoint.")
		return nil
	}
	fmt.Printf("Pull %s now with `ollama pull %s`? [Y/n]: ", model.ID, model.ID)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "" || answer == "y" || answer == "yes" {
		if err := runInteractiveCommand("ollama", "pull", model.ID); err != nil {
			fmt.Printf("Ollama pull failed: %v\n", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := smokeLocalEmbedding(ctx, cfg.AI.Embedding.Local, model.Dimensions); err != nil {
		fmt.Printf("Embedding smoke: WARN: %v\n", err)
		fmt.Println("Config will still be written; run `oneday doctor` after starting Ollama.")
	} else {
		fmt.Printf("Embedding smoke: OK (%s, %d dimensions)\n", model.ID, model.Dimensions)
	}
	return nil
}

func configureCustomLocalRAG(reader *bufio.Reader, cfg *config.Config) error {
	fmt.Print("Custom embedding endpoint URL [http://127.0.0.1:8000/embed]: ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000/embed"
	}
	fmt.Print("Embedding model name [local-embedding]: ")
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model == "" {
		model = "local-embedding"
	}
	fmt.Print("Embedding dimensions [1024]: ")
	dimText, _ := reader.ReadString('\n')
	dimensions := parsePositiveInt(strings.TrimSpace(dimText), 1024)

	cfg.RAG.Enabled = true
	cfg.RAG.Dimensions = dimensions
	cfg.AI.Embedding.Provider = "local"
	cfg.AI.Embedding.Local.Enabled = true
	cfg.AI.Embedding.Local.Type = "custom"
	cfg.AI.Embedding.Local.BaseURL = baseURL
	cfg.AI.Embedding.Local.Model = model
	cfg.AI.Embedding.Local.Dimensions = dimensions

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := smokeLocalEmbedding(ctx, cfg.AI.Embedding.Local, dimensions); err != nil {
		fmt.Printf("Embedding smoke: WARN: %v\n", err)
		fmt.Println("Config will still be written; run `oneday doctor` after starting your local embedding server.")
	} else {
		fmt.Printf("Embedding smoke: OK (%s, %d dimensions)\n", model, dimensions)
	}
	return nil
}

func localEmbeddingModelByChoice(choice string) (localEmbeddingModel, error) {
	for i, model := range localEmbeddingModels {
		if choice == fmt.Sprintf("%d", i+1) || choice == model.ID {
			return model, nil
		}
	}
	return localEmbeddingModel{}, fmt.Errorf("unknown local embedding model choice %q", choice)
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(value, "%d", &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}

func runInteractiveCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func smokeLocalEmbedding(ctx context.Context, local config.LocalEmbeddingConfig, dimensions int) error {
	var emb interface {
		Embed(context.Context, ai.EmbeddingRequest) (ai.EmbeddingResponse, error)
	}
	switch local.Type {
	case "ollama":
		emb = providers.NewOllamaEmbedding(providers.OllamaEmbeddingConfig{
			BaseURL: local.BaseURL,
			Model:   local.Model,
			Timeout: 20 * time.Second,
		})
	case "custom":
		emb = providers.NewLocalHTTPEmbedding(local.BaseURL, local.Model, 20*time.Second)
	default:
		return fmt.Errorf("unknown local embedding type %q", local.Type)
	}
	resp, err := emb.Embed(ctx, ai.EmbeddingRequest{Input: "oneday local rag smoke", Model: local.Model})
	if err != nil {
		return err
	}
	if len(resp.Embedding) != dimensions {
		return fmt.Errorf("model %s returned %d dimensions, expected %d", resp.Model, len(resp.Embedding), dimensions)
	}
	return nil
}

func setupConfigForChoice(cfg config.Config, choice string) (config.Config, error) {
	switch choice {
	case "1":
		cfg.AI.ProviderPriority = []string{"codex", "litellm", "openrouter", "claude-code"}
		cfg.AI.Codex.Enabled = true
		cfg.AI.LiteLLM.Enabled = false
		cfg.AI.OpenRouter.Enabled = false
		cfg.RAG.Enabled = false
	case "2":
		cfg.AI.ProviderPriority = []string{"litellm", "openrouter", "codex", "claude-code"}
		cfg.AI.LiteLLM.Enabled = true
		cfg.AI.OpenRouter.Enabled = false
		cfg.AI.Codex.Enabled = false
		cfg.AI.LiteLLM.APIKey = "${ONEDAY_LITELLM_API_KEY}"
	case "3":
		cfg.AI.ProviderPriority = []string{"openrouter", "litellm", "codex", "claude-code"}
		cfg.AI.OpenRouter.Enabled = true
		cfg.AI.LiteLLM.Enabled = false
		cfg.AI.Codex.Enabled = false
		cfg.AI.OpenRouter.APIKey = "${ONEDAY_OPENROUTER_API_KEY}"
	case "4":
		cfg.AI.ProviderPriority = []string{"codex", "litellm", "openrouter", "claude-code"}
		cfg.AI.Codex.Enabled = true
		cfg.AI.LiteLLM.Enabled = false
		cfg.AI.OpenRouter.Enabled = false
		cfg.RAG.Enabled = true
		cfg.AI.Embedding.Provider = "local"
		cfg.AI.Embedding.Local.Enabled = true
		cfg.AI.Embedding.Local.Type = "ollama"
		cfg.AI.Embedding.Local.BaseURL = "http://127.0.0.1:11434"
		cfg.AI.Embedding.Local.Model = "bge-m3"
		cfg.AI.Embedding.Local.Dimensions = 1024
		cfg.RAG.Dimensions = 1024
	default:
		return config.Config{}, fmt.Errorf("unknown selection %q", choice)
	}
	return cfg, nil
}

type doctorReport struct {
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
	ConfigPath      string   `json:"config_path"`
	EnabledProvider []string `json:"enabled_providers"`
	CodexLogin      string   `json:"codex_login"`
	RAGEnabled      bool     `json:"rag_enabled"`
	EmbeddingKind   string   `json:"embedding_kind"`
	EmbeddingModel  string   `json:"embedding_model"`
	EmbeddingDims   int      `json:"embedding_dimensions"`
	Warnings        []string `json:"warnings"`
}

func runDoctor(args []string) error {
	if wantsJSON(args) {
		return runDoctorJSON()
	}
	fmt.Println("OneDay doctor")
	fmt.Printf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	reportCommand("go", "version")
	reportCommand("codex", "--version")
	reportCommand("codex", "login", "status")
	reportCommand("claude", "--version")

	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Printf("ENV: WARN: could not load .env: %v\n", err)
	} else {
		fmt.Println("ENV: OK")
	}

	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return err
	}
	fmt.Printf("Config: OK (%s)\n", resolveConfigPath())
	fmt.Printf("Models: codex=%s utility=%s embedding=%s\n", cfg.AI.Codex.Model, cfg.AI.Generation.UtilityModel, cfg.AI.Embedding.Model)
	reportConfigConsistency(cfg)

	codexStatus := commandStatus("codex", "login", "status")
	if codexStatus == "" {
		fmt.Println("Codex login: SKIP: codex CLI not found")
	} else if strings.Contains(strings.ToLower(codexStatus), "not") && strings.Contains(strings.ToLower(codexStatus), "login") {
		fmt.Printf("Codex login: FAIL: %s\n", codexStatus)
	} else {
		fmt.Printf("Codex login: OK: %s\n", codexStatus)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	router, err := aifactory.NewRouterFromConfig(cfg)
	if err != nil {
		fmt.Printf("Provider smoke: FAIL: %v\n", err)
	} else {
		resp, err := router.Complete(ctx, ai.Request{
			Messages:  []ai.Message{{Role: ai.RoleUser, Content: "reply with OK"}},
			MaxTokens: 8,
		})
		if err != nil {
			fmt.Printf("Provider smoke: FAIL: %v\n", err)
		} else {
			fmt.Printf("Provider smoke: OK: %s (%s)\n", firstLine(resp.Content), resp.Provider)
		}
	}

	if !cfg.RAG.Enabled {
		fmt.Println("RAG: disabled, reason: config rag.enabled=false")
		fmt.Println("Embedding smoke: SKIP: RAG disabled")
		return nil
	}

	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		fmt.Printf("RAG: disabled, reason: %s\n", reason)
		fmt.Println("Embedding smoke: SKIP: no embedding-capable provider configured")
		return nil
	}
	fmt.Printf("RAG: enabled, embedding provider: %s, model: %s\n", spec.Name, cfg.AI.Embedding.Model)
	if spec.Kind == "ollama" || spec.Kind == "custom" {
		fmt.Printf("Local RAG: enabled, type: %s, url: %s, model: %s, dimensions: %d\n", spec.Kind, spec.BaseURL, spec.Model, spec.Dimensions)
	}
	emb := embeddingProviderForSpec(spec, 20*time.Second)
	embResp, err := emb.Embed(ctx, ai.EmbeddingRequest{
		Input: "oneday doctor embedding smoke",
		Model: spec.Model,
	})
	if err != nil {
		fmt.Printf("Embedding smoke: FAIL: %v\n", err)
		return nil
	}
	fmt.Printf("Embedding smoke: OK: %d dimensions (%s)\n", len(embResp.Embedding), embResp.Model)
	return nil
}

func runDoctorJSON() error {
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		// JSON mode reports config/env issues through warnings where possible.
	}
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return err
	}
	report := doctorReport{
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		ConfigPath:      resolveConfigPath(),
		EnabledProvider: cfg.EnabledProviders(),
		CodexLogin:      commandStatus("codex", "login", "status"),
		RAGEnabled:      cfg.RAG.Enabled,
		EmbeddingModel:  cfg.AI.Embedding.Model,
		EmbeddingDims:   cfg.RAG.Dimensions,
		Warnings:        providerConsistencyWarnings(cfg),
	}
	if spec, reason := aifactory.SelectEmbeddingProvider(cfg); reason == "" {
		report.EmbeddingKind = spec.Kind
		report.EmbeddingModel = spec.Model
		report.EmbeddingDims = spec.Dimensions
	} else if cfg.RAG.Enabled {
		report.Warnings = append(report.Warnings, "rag unavailable: "+reason)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func runConfigShowSafe() error {
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Printf("ENV: WARN: could not load .env: %v\n", err)
	}
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return err
	}
	fmt.Println("OneDay config (safe)")
	fmt.Printf("config_path: %s\n", resolveConfigPath())
	fmt.Printf("data_dir: %s\n", cfg.DataDir)
	fmt.Printf("providers: %s\n", strings.Join(cfg.EnabledProviders(), ", "))
	fmt.Printf("codex: enabled=%v model=%s reasoning=%s\n", cfg.AI.Codex.Enabled, cfg.AI.Codex.Model, cfg.AI.Codex.Reasoning)
	fmt.Printf("litellm: enabled=%v base_url=%s api_key=%s model=%s\n", cfg.AI.LiteLLM.Enabled, cfg.AI.LiteLLM.BaseURL, redactSecret(cfg.AI.LiteLLM.APIKey), cfg.AI.LiteLLM.DefaultModel)
	fmt.Printf("openrouter: enabled=%v base_url=%s api_key=%s model=%s\n", cfg.AI.OpenRouter.Enabled, cfg.AI.OpenRouter.BaseURL, redactSecret(cfg.AI.OpenRouter.APIKey), cfg.AI.OpenRouter.DefaultModel)
	fmt.Printf("generation: utility=%s repair=%s fallbacks=%s\n", cfg.AI.Generation.UtilityModel, cfg.AI.Generation.RepairModel, strings.Join(cfg.AI.Generation.RepairFallbackModels, ","))
	fmt.Printf("rag: enabled=%v top_k=%d summarize_every=%d dimensions=%d\n", cfg.RAG.Enabled, cfg.RAG.TopK, cfg.RAG.SummarizeEvery, cfg.RAG.Dimensions)
	fmt.Printf("embedding: provider=%s model=%s\n", cfg.AI.Embedding.Provider, cfg.AI.Embedding.Model)
	fmt.Printf("embedding.local: enabled=%v type=%s base_url=%s model=%s dimensions=%d\n", cfg.AI.Embedding.Local.Enabled, cfg.AI.Embedding.Local.Type, cfg.AI.Embedding.Local.BaseURL, cfg.AI.Embedding.Local.Model, cfg.AI.Embedding.Local.Dimensions)
	return nil
}

func runRAGBenchmark() error {
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Printf("ENV: WARN: could not load .env: %v\n", err)
	}
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return err
	}
	fmt.Println("OneDay RAG benchmark")
	if !cfg.RAG.Enabled {
		fmt.Println("RAG: disabled")
		return nil
	}
	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		fmt.Printf("RAG: unavailable: %s\n", reason)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	start := time.Now()
	emb := embeddingProviderForSpec(spec, 30*time.Second)
	resp, err := emb.Embed(ctx, ai.EmbeddingRequest{Input: "oneday rag benchmark local retrieval smoke", Model: spec.Model})
	if err != nil {
		fmt.Printf("benchmark: FAIL: %v\n", err)
		return nil
	}
	latency := time.Since(start)
	status := "OK"
	if spec.Dimensions > 0 && len(resp.Embedding) != spec.Dimensions {
		status = "DIMENSION_MISMATCH"
	}
	fmt.Printf("benchmark: %s provider=%s kind=%s model=%s dimensions=%d expected=%d latency=%s\n", status, spec.Name, spec.Kind, resp.Model, len(resp.Embedding), spec.Dimensions, latency.Round(time.Millisecond))
	return nil
}

func runRAGReindex(args []string) error {
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Printf("ENV: WARN: could not load .env: %v\n", err)
	}
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return err
	}
	storyID := argValue(args, "--story")
	all := hasArg(args, "--all")
	if storyID == "" && !all {
		fmt.Println("Usage: oneday rag reindex --story <story-id> [--all]")
		fmt.Println("This clears stale RAG chunks so they are regenerated during play.")
		return nil
	}
	db, err := storage.Open(filepath.Join(cfg.DataDir, "oneday.db"))
	if err != nil {
		return err
	}
	defer db.Close()
	store := rag.NewVectorStore(db.Conn())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if all {
		removed, err := clearAllRAGChunks(ctx, db.Conn())
		if err != nil {
			return err
		}
		fmt.Printf("RAG reindex: cleared %d chunks across all stories\n", removed)
		return nil
	}
	removed, err := store.DeleteByStory(ctx, storyID)
	if err != nil {
		return err
	}
	fmt.Printf("RAG reindex: cleared %d chunks for story %s\n", removed, storyID)
	return nil
}

func clearAllRAGChunks(ctx context.Context, db *sql.DB) (int64, error) {
	result, err := db.ExecContext(ctx, `DELETE FROM rag_chunks`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func hasArg(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}

func argValue(args []string, key string) string {
	for i, arg := range args {
		if arg == key && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func runStoryPacksList() error {
	packs, err := discoverStoryPacks([]string{"plugins/story-packs", "plugins/examples"})
	if err != nil {
		return err
	}
	fmt.Println("OneDay story packs")
	if len(packs) == 0 {
		fmt.Println("No story packs found.")
		return nil
	}
	for _, pack := range packs {
		fmt.Printf("- %s\n", pack)
	}
	return nil
}

func runExportFriend(args []string) error {
	outDir := "dist/oneday-friend"
	for i, arg := range args {
		if arg == "--out" && i+1 < len(args) {
			outDir = args[i+1]
		}
	}
	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	files := []string{"README.md", "config.example.yaml", ".env.example"}
	for _, file := range files {
		if err := copyFileIfExists(file, filepath.Join(outDir, file)); err != nil {
			return err
		}
	}
	if err := copyDirIfExists("plugins/examples", filepath.Join(outDir, "plugins", "examples")); err != nil {
		return err
	}
	manifest := "OneDay friend-safe export\n\nRun:\n  ./oneday setup --reconfigure\n  ./oneday doctor\n\nExcluded: config.yaml, .env, oneday_data, databases, generated binaries, local secrets.\n"
	if err := os.WriteFile(filepath.Join(outDir, "FRIEND-SETUP.txt"), []byte(manifest), 0644); err != nil {
		return err
	}
	fmt.Printf("Friend-safe export written to %s\n", outDir)
	return nil
}

func copyFileIfExists(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func copyDirIfExists(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDirIfExists(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFileIfExists(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func discoverStoryPacks(roots []string) ([]string, error) {
	var packs []string
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				packs = append(packs, filepath.Join(root, entry.Name()))
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".json") {
				packs = append(packs, filepath.Join(root, name))
			}
		}
	}
	sort.Strings(packs)
	return packs, nil
}

func reportConfigConsistency(cfg config.Config) {
	warnings := providerConsistencyWarnings(cfg)
	for _, warning := range warnings {
		fmt.Printf("Config warning: %s\n", warning)
	}
}

func providerConsistencyWarnings(cfg config.Config) []string {
	var warnings []string
	if cfg.AI.LiteLLM.Enabled && strings.TrimSpace(cfg.AI.LiteLLM.APIKey) == "" {
		warnings = append(warnings, "litellm is enabled but ONEDAY_LITELLM_API_KEY/api_key is empty")
	}
	if !cfg.AI.LiteLLM.Enabled && strings.TrimSpace(os.Getenv("ONEDAY_LITELLM_API_KEY")) != "" {
		warnings = append(warnings, "ONEDAY_LITELLM_API_KEY is set but litellm is disabled")
	}
	if cfg.AI.OpenRouter.Enabled && strings.TrimSpace(cfg.AI.OpenRouter.APIKey) == "" {
		warnings = append(warnings, "openrouter is enabled but ONEDAY_OPENROUTER_API_KEY/api_key is empty")
	}
	if !cfg.AI.OpenRouter.Enabled && strings.TrimSpace(os.Getenv("ONEDAY_OPENROUTER_API_KEY")) != "" {
		warnings = append(warnings, "ONEDAY_OPENROUTER_API_KEY is set but openrouter is disabled")
	}
	if cfg.RAG.Enabled && cfg.AI.Embedding.Provider == "local" && !cfg.AI.Embedding.Local.Enabled {
		warnings = append(warnings, "rag is enabled with local embedding provider but local embeddings are disabled")
	}
	if cfg.AI.Embedding.Provider == "local" && cfg.RAG.Dimensions != cfg.AI.Embedding.Local.Dimensions {
		warnings = append(warnings, fmt.Sprintf("rag.dimensions=%d differs from local embedding dimensions=%d", cfg.RAG.Dimensions, cfg.AI.Embedding.Local.Dimensions))
	}
	return warnings
}

func redactSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "<empty>"
	}
	if strings.HasPrefix(value, "${") {
		return value
	}
	return "<redacted>"
}

func embeddingProviderForSpec(spec aifactory.EmbeddingProviderSpec, timeout time.Duration) interface {
	Embed(context.Context, ai.EmbeddingRequest) (ai.EmbeddingResponse, error)
} {
	switch spec.Kind {
	case "ollama":
		return providers.NewOllamaEmbedding(providers.OllamaEmbeddingConfig{
			BaseURL: spec.BaseURL,
			Model:   spec.Model,
			Timeout: timeout,
		})
	case "custom":
		return providers.NewLocalHTTPEmbedding(spec.BaseURL, spec.Model, timeout)
	default:
		return providers.NewOpenAICompat(providers.OpenAICompatConfig{
			Name:         spec.Name,
			BaseURL:      spec.BaseURL,
			APIKey:       spec.APIKey,
			DefaultModel: spec.Model,
			Timeout:      timeout,
		})
	}
}

func reportCommand(name string, args ...string) {
	path, err := exec.LookPath(name)
	if err != nil {
		fmt.Printf("%s: not found\n", name)
		return
	}
	cmd := exec.Command(path, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s: found at %s\n", name, path)
		return
	}
	fmt.Printf("%s: %s\n", name, firstLine(strings.TrimSpace(string(out))))
}

func commandStatus(name string, args ...string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	cmd := exec.Command(path, args...)
	out, err := cmd.CombinedOutput()
	line := firstLine(strings.TrimSpace(string(out)))
	if err != nil && line == "" {
		return fmt.Sprintf("found at %s", path)
	}
	if line == "" {
		return fmt.Sprintf("found at %s", path)
	}
	return line
}

func firstLine(value string) string {
	line, _, _ := strings.Cut(value, "\n")
	return line
}

func ensureEnvFile() {
	if _, err := os.Stat(".env"); err == nil {
		return
	}
	var lines []string
	if os.Getenv("ONEDAY_LITELLM_API_KEY") == "" {
		lines = append(lines, "ONEDAY_LITELLM_API_KEY=")
	}
	if os.Getenv("ONEDAY_OPENROUTER_API_KEY") == "" {
		lines = append(lines, "ONEDAY_OPENROUTER_API_KEY=")
	}
	if len(lines) == 0 {
		return
	}
	content := strings.Join(lines, "\n") + "\n"
	_ = os.WriteFile(".env", []byte(content), 0600)
}
