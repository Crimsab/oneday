package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/providers"
	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/buildinfo"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/setup"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui"
)

func main() {
	configureGatewayRequestContext()
	if wantsHelp(os.Args[1:]) {
		printUsage(os.Stdout)
		return
	}
	if wantsVersion(os.Args[1:]) {
		fmt.Println(buildinfo.Text("oneday"))
		return
	}
	loc := cliLocalizer()
	if wantsSetup(os.Args[1:]) {
		if err := runSetup(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, loc.T("cli.setup_failed", err))
			os.Exit(1)
		}
		return
	}
	if wantsDoctor(os.Args[1:]) {
		if err := runDoctor(os.Args[1:]); err != nil {
			if errors.Is(err, errDoctorRequiredFailure) {
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, loc.T("cli.doctor_failed", err))
			os.Exit(1)
		}
		return
	}
	if wantsConfigShowSafe(os.Args[1:]) {
		if err := runConfigShowSafe(); err != nil {
			fmt.Fprintln(os.Stderr, loc.T("cli.config_show_failed", err))
			os.Exit(1)
		}
		return
	}
	if wantsConfigLocale(os.Args[1:]) {
		if err := runConfigLocale(os.Args[1:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, loc.T("cli.config_locale_failed", err))
			os.Exit(1)
		}
		return
	}
	if wantsRAGBenchmark(os.Args[1:]) {
		if err := runRAGBenchmark(); err != nil {
			fmt.Fprintln(os.Stderr, loc.T("cli.rag_benchmark_failed", err))
			os.Exit(1)
		}
		return
	}
	if wantsRAGReindex(os.Args[1:]) {
		if err := runRAGReindex(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, loc.T("cli.rag_reindex_failed", err))
			os.Exit(1)
		}
		return
	}
	if wantsStoryPacksList(os.Args[1:]) {
		if err := runStoryPacksList(); err != nil {
			fmt.Fprintln(os.Stderr, loc.T("cli.story_packs_failed", err))
			os.Exit(1)
		}
		return
	}
	if wantsExport(os.Args[1:]) {
		if err := runExport(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, loc.T("cli.export_failed", err))
			os.Exit(1)
		}
		return
	}
	if wantsGatewayCommandDescriptors(os.Args[1:]) {
		if err := runGatewayCommandDescriptors(os.Stdout, os.Args[2:]...); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway command descriptors failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayModelSettings(os.Args[1:]) {
		if err := runGatewayModelSettings(resolveConfigPath(), os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway model settings failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayModelSettingsUpdate(os.Args[1:]) {
		if err := runGatewayModelSettingsUpdate(resolveConfigPath(), os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway model settings update failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Fprintln(os.Stderr, loc.T("cli.env_warn", err))
	}
	// Load config
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, loc.T("cli.config_load_failed", err))
		os.Exit(1)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "oneday.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, loc.T("cli.database_open_failed", err))
		os.Exit(1)
	}
	defer db.Close()

	if wantsGatewaySchemaPreflight(os.Args[1:]) {
		if err := runGatewaySchemaPreflight(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway schema preflight failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayTimeline(os.Args[1:]) {
		if err := runGatewayTimeline(db, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway timeline failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayMiniGameStart(os.Args[1:]) || wantsGatewayMiniGameGet(os.Args[1:]) || wantsGatewayMiniGameInput(os.Args[1:]) {
		operation := "get"
		if wantsGatewayMiniGameStart(os.Args[1:]) {
			operation = "start"
		} else if wantsGatewayMiniGameInput(os.Args[1:]) {
			operation = "input"
		}
		if err := runGatewayMiniGame(db, operation, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway minigame failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayAudio(os.Args[1:]) {
		if err := runGatewayAudio(gatewayContext(), cfg, db, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway audio failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Create AI router
	router, err := aifactory.NewRouterFromConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, loc.T("cli.router_failed", err))
		os.Exit(1)
	}
	router.SetTelemetryRecorder(storage.NewAITelemetryRecorder(db))

	if wantsGatewayTurn(os.Args[1:]) {
		if err := runGatewayTurn(gatewayContext(), cfg, db, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway turn failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayCraft(os.Args[1:]) {
		if err := runGatewayCraft(gatewayContext(), cfg, db, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway craft failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayStoryCreate(os.Args[1:]) {
		if err := runGatewayStoryCreate(gatewayContext(), cfg, db, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway story create failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayStoryWizard(os.Args[1:]) {
		if err := runGatewayStoryWizard(gatewayContext(), cfg, db, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway story wizard failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayStoryEnhance(os.Args[1:]) {
		if err := runGatewayStoryEnhance(gatewayContext(), cfg, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway story enhance failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayTranslate(os.Args[1:]) {
		if err := runGatewayTranslate(gatewayContext(), cfg, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway translation failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayMeta(os.Args[1:]) {
		if err := runGatewayMeta(gatewayContext(), cfg, db, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway meta failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewaySave(os.Args[1:]) {
		if err := runGatewaySave(gatewayContext(), cfg, db, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway save failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayLoad(os.Args[1:]) {
		if err := runGatewayLoad(gatewayContext(), cfg, db, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway load failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if wantsGatewayDeleteSave(os.Args[1:]) {
		if err := runGatewayDeleteSave(gatewayContext(), cfg, db, router, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Gateway delete save failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Start TUI
	locale := appi18n.Resolve(cfg.Interface.Locale, nil)
	app := tui.New(cfg, db, router, appi18n.New(locale), resolveConfigPath())
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, loc.T("cli.tui_failed", err))
		os.Exit(1)
	}
}

func gatewayRequestID() string {
	raw := strings.TrimSpace(os.Getenv("ONEDAY_REQUEST_ID"))
	if raw == "" {
		return ""
	}
	var safe strings.Builder
	for _, char := range raw {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' {
			safe.WriteRune(char)
			if safe.Len() >= 128 {
				break
			}
		}
	}
	return safe.String()
}

func configureGatewayRequestContext() {
	if requestID := gatewayRequestID(); requestID != "" {
		log.SetPrefix("request_id=" + requestID + " ")
	}
}

func gatewayContext() context.Context {
	ctx := context.Background()
	requestID := gatewayRequestID()
	if requestID == "" {
		return ctx
	}
	return ai.WithTelemetry(ctx, ai.TelemetryMetadata{
		TraceID:      requestID,
		SafeMetadata: map[string]string{"request_id": requestID},
	})
}

func resolveDotEnvPath() string {
	const envName = ".env"
	if configPath := explicitConfigPath(); configPath != "" {
		return filepath.Join(filepath.Dir(configPath), envName)
	}

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

	if configPath := explicitConfigPath(); configPath != "" {
		return configPath
	}

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

func explicitConfigPath() string {
	return strings.TrimSpace(os.Getenv("ONEDAY_CONFIG"))
}

// configDisplaySource is safe to include in diagnostics. Configuration paths
// can reveal local usernames or deployment layouts, so doctor reports only
// how the configuration was selected.
func configDisplaySource() string {
	if explicitConfigPath() != "" {
		return "ONEDAY_CONFIG"
	}
	return "default config search"
}

func cliLocalizer() appi18n.Localizer {
	cfg, _ := config.Load(resolveConfigPath())
	return appi18n.New(appi18n.Resolve(cfg.Interface.Locale, nil))
}

func configLocalizer(cfg config.Config) appi18n.Localizer {
	return appi18n.New(appi18n.Resolve(cfg.Interface.Locale, nil))
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

func wantsNoInput(args []string) bool {
	for _, arg := range args {
		if arg == "--no-input" {
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

func wantsHelp(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "help", "--help", "-h":
			return true
		}
	}
	return false
}

func printUsage(w io.Writer) {
	loc := cliLocalizer()
	for _, key := range []string{
		"cli.help.title", "", "cli.help.usage", "", "cli.help.commands",
		"cli.help.play", "cli.help.setup", "cli.help.doctor", "cli.help.config_show",
		"cli.help.config_locale", "cli.help.rag_benchmark", "cli.help.rag_reindex",
		"cli.help.story_packs", "cli.help.export", "cli.help.version", "cli.help.help",
		"", "cli.help.docs",
	} {
		if key == "" {
			fmt.Fprintln(w)
			continue
		}
		fmt.Fprintln(w, loc.T(key))
	}
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

func wantsConfigLocale(args []string) bool {
	return len(args) >= 2 && args[0] == "config" && args[1] == "locale"
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

func wantsExport(args []string) bool {
	return len(args) >= 1 && args[0] == "export"
}

func wantsGatewayCommandDescriptors(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-command-descriptors"
}

func wantsGatewaySchemaPreflight(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-schema-preflight"
}

func wantsGatewayModelSettings(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-model-settings"
}

func wantsGatewayModelSettingsUpdate(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-model-settings-update"
}

func wantsGatewayTurn(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-turn"
}

func wantsGatewayCraft(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-craft"
}

func wantsGatewayStoryCreate(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-story-create"
}

func wantsGatewayStoryWizard(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-story-wizard"
}

func wantsGatewayStoryEnhance(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-story-enhance"
}

func wantsGatewayTranslate(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-translate"
}

func wantsGatewayMiniGameStart(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-minigame-start"
}

func wantsGatewayMiniGameGet(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-minigame-get"
}

func wantsGatewayMiniGameInput(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-minigame-input"
}

func wantsGatewayAudio(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-audio"
}

func wantsGatewayMeta(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-meta"
}

func wantsGatewaySave(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-save"
}

func wantsGatewayLoad(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-load"
}

func wantsGatewayDeleteSave(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-delete-save"
}

func wantsGatewayTimeline(args []string) bool {
	return len(args) >= 1 && args[0] == "gateway-timeline"
}

func runSetup(args []string) error {
	reader := bufio.NewReader(os.Stdin)
	configPath := resolveConfigPath()
	current, err := config.LoadForEdit(configPath)
	if err != nil {
		return err
	}
	loc := appi18n.New(appi18n.Resolve(current.Interface.Locale, nil))

	fmt.Println(loc.T("cli.setup_title"))
	fmt.Printf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	reportCommand("go", "version")
	reportCommand("codex", "--version")
	reportCommand("codex", "login", "status")
	reportCommand("claude", "--version")

	force := wantsSetupForce(args)
	if _, err := os.Stat(configPath); err == nil && !force {
		fmt.Println(loc.SetupPresentation("config_exists", "config.yaml already exists; leaving it in place."))
		fmt.Println(loc.SetupPresentation("config_reconfigure", "Run `oneday setup --reconfigure` or `oneday setup --force` to open the setup wizard again."))
		return nil
	}
	if wantsNoInput(args) {
		if err := current.Validate(); err != nil {
			return fmt.Errorf("--no-input needs a complete existing configuration: %w; run `oneday setup --reconfigure` interactively", err)
		}
		if len(current.EnabledProviders()) == 0 {
			return errors.New("--no-input needs an enabled narrative provider; run `oneday setup --reconfigure` interactively")
		}
		fmt.Println("Setup unchanged: existing configuration is complete.")
		return nil
	}

	fmt.Println()
	fmt.Println(loc.T("cli.choose_language"))
	fmt.Println(loc.T("cli.language_choice"))
	defaultChoice := "1"
	if loc.Locale() == appi18n.Italian {
		defaultChoice = "2"
	}
	fmt.Print(loc.T("cli.selection", defaultChoice))
	languageChoice, _ := reader.ReadString('\n')
	switch strings.TrimSpace(languageChoice) {
	case "", defaultChoice:
	case "1", "en":
		loc = appi18n.New(appi18n.English)
	case "2", "it":
		loc = appi18n.New(appi18n.Italian)
	default:
		return fmt.Errorf(loc.T("cli.locale_invalid"), strings.TrimSpace(languageChoice))
	}

	fmt.Println()
	fmt.Println(loc.T("cli.choose_provider"))
	fmt.Println(loc.SetupPresentation("provider_codex", "  1) Codex OAuth (uses local `codex login`)"))
	fmt.Println(loc.SetupPresentation("provider_litellm", "  2) LiteLLM / homelab proxy"))
	fmt.Println(loc.SetupPresentation("provider_openrouter", "  3) OpenRouter"))
	fmt.Println(loc.SetupPresentation("provider_codex_rag", "  4) Codex OAuth + local RAG embeddings"))
	fmt.Print(loc.SetupPresentation("selection", "Selection [1]: "))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	// Reconfiguration changes only the setup-owned provider choices; retaining
	// the loaded config preserves storage, media, and other user settings.
	cfg := current
	pendingEnv := newPendingEnvUpdates()
	cfg.Interface.Locale = string(loc.Locale())
	cfg, err = setupConfigForChoice(cfg, choice, loc)
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		if err := configureCodex(reader, &cfg); err != nil {
			return err
		}
		fmt.Println(loc.SetupPresentation("codex_login", "If Codex is not logged in yet, run: codex login"))
		fmt.Println(loc.SetupPresentation("rag_disabled", "RAG: disabled, reason: no embedding-capable provider configured"))
	case "2":
		pendingEnv.Ensure("ONEDAY_LITELLM_API_KEY")
		if err := configureLiteLLM(reader, &cfg); err != nil {
			return err
		}
		if err := configureRAGChoice(reader, &cfg, loc); err != nil {
			return err
		}
	case "3":
		pendingEnv.Ensure("ONEDAY_OPENROUTER_API_KEY")
		if err := configureOpenRouter(reader, &cfg); err != nil {
			return err
		}
		if err := configureRAGChoice(reader, &cfg, loc); err != nil {
			return err
		}
	case "4":
		if err := configureCodex(reader, &cfg); err != nil {
			return err
		}
		if err := configureLocalRAG(reader, &cfg); err != nil {
			return err
		}
	}
	if err := configureImageChoice(reader, &cfg, loc, pendingEnv); err != nil {
		return err
	}
	if err := configureTTSChoice(reader, &cfg, loc, pendingEnv); err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := configureStoryPackChoice(reader, loc); err != nil {
		return err
	}
	data, err := config.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := writePendingEnv(resolveDotEnvPath(), pendingEnv); err != nil {
		return err
	}
	if err := setup.WriteFileAtomic(configPath, data); err != nil {
		return err
	}
	fmt.Println(loc.T("cli.wrote_config"))
	return nil
}

func configureImageChoice(reader *bufio.Reader, cfg *config.Config, loc appi18n.Localizer, pending ...*pendingEnvUpdates) error {
	updates := firstPendingEnvUpdate(pending)
	image := &cfg.AI.ImageGeneration
	fmt.Println()
	fmt.Println(loc.T("cli.image_choice"))
	fmt.Println(loc.T("cli.image_text_only"))
	fmt.Println(loc.T("cli.image_hosted"))
	fmt.Println(loc.T("cli.image_local"))
	fmt.Println(loc.T("cli.image_bridge"))
	fmt.Print(loc.T("cli.media_selection"))
	choice, _ := reader.ReadString('\n')
	switch strings.TrimSpace(choice) {
	case "":
		return nil
	case "1", "text", "text-only":
		disableImageGeneration(image)
		return nil
	case "2", "hosted":
		return configureHostedImage(reader, image, loc, updates)
	case "3", "local", "openai-compatible":
		return configureLocalImage(reader, image, loc, updates)
	case "4", "bridge", "imagegen-bridge":
		return configureImagegenBridge(reader, image, loc, updates)
	default:
		return fmt.Errorf(loc.T("cli.media_selection_invalid"), strings.TrimSpace(choice))
	}
}

func disableImageGeneration(image *config.ImageGenerationConfig) {
	previous := strings.ToLower(strings.TrimSpace(image.Provider))
	image.AutoGenerate = false
	image.Provider = "text-only"
	image.MapIconProvider = "text-only"
	image.Model = ""
	image.MapIconModel = ""
	image.BaseURL = ""
	image.APIKey = ""
	if image.Providers != nil {
		delete(image.Providers, imageProviderConfigKey(previous))
	}
	if previous == "codex-oauth" || previous == "imagegen-bridge" || previous == "imagegen_bridge" || previous == "bridge-native" {
		image.ImagegenBridgeToken = ""
	}
}

func imageProviderConfigKey(provider string) string {
	switch provider {
	case "litellm":
		return config.ImageProviderOpenAICompatible
	case "imagegen-bridge", "imagegen_bridge", "bridge-native":
		return config.ImageProviderCodexOAuth
	default:
		return provider
	}
}

func configureHostedImage(reader *bufio.Reader, image *config.ImageGenerationConfig, loc appi18n.Localizer, updates *pendingEnvUpdates) error {
	previousEndpoint := imageProviderBaseURL(*image, config.ImageProviderOpenAI)
	endpoint, err := promptRequiredValue(reader, loc.T("cli.image_endpoint"), previousEndpoint, loc)
	if err != nil {
		return err
	}
	model, err := promptRequiredValue(reader, loc.T("cli.image_model"), imageModel(*image, "gpt-image-2"), loc)
	if err != nil {
		return err
	}
	previousKey := imageProviderSecret(*image, config.ImageProviderOpenAI)
	key, entered, err := promptSecret(reader, loc.T("cli.image_api_key"), credentialForOrigin(previousKey, previousEndpoint, endpoint))
	if err != nil {
		return err
	}
	if key == "" {
		if requiresCredentialReentry(previousKey, previousEndpoint, endpoint) {
			return errors.New(loc.T("cli.credential_reentry_required"))
		}
		return errors.New(loc.T("cli.secret_required"))
	}
	if key, err = envBackedSecret(updates, "ONEDAY_IMAGEGEN_OPENAI_API_KEY", key, entered); err != nil {
		return err
	}
	configureDirectImage(image, config.ImageProviderOpenAI, endpoint, model, key, "bearer", "")
	return nil
}

func configureLocalImage(reader *bufio.Reader, image *config.ImageGenerationConfig, loc appi18n.Localizer, updates *pendingEnvUpdates) error {
	previousEndpoint := imageProviderBaseURL(*image, config.ImageProviderOpenAICompatible)
	endpoint, err := promptRequiredValue(reader, loc.T("cli.image_endpoint"), previousEndpoint, loc)
	if err != nil {
		return err
	}
	model, err := promptRequiredValue(reader, loc.T("cli.image_model"), imageModel(*image, ""), loc)
	if err != nil {
		return err
	}
	fmt.Print(loc.T("cli.image_auth_mode"))
	authMode, _ := reader.ReadString('\n')
	authMode = strings.ToLower(strings.TrimSpace(authMode))
	if authMode == "" {
		authMode = "bearer"
	}
	if authMode != "bearer" && authMode != "none" {
		return fmt.Errorf(loc.T("cli.auth_mode_invalid"), authMode)
	}
	key := ""
	if authMode == "bearer" {
		previousKey := imageProviderSecret(*image, config.ImageProviderOpenAICompatible)
		var entered bool
		key, entered, err = promptSecret(reader, loc.T("cli.image_api_key"), credentialForOrigin(previousKey, previousEndpoint, endpoint))
		if err != nil {
			return err
		}
		if key == "" {
			if requiresCredentialReentry(previousKey, previousEndpoint, endpoint) {
				return errors.New(loc.T("cli.credential_reentry_required"))
			}
			return errors.New(loc.T("cli.secret_required"))
		}
		if key, err = envBackedSecret(updates, "ONEDAY_IMAGEGEN_OPENAI_COMPATIBLE_API_KEY", key, entered); err != nil {
			return err
		}
	}
	probe, err := promptRequiredValue(reader, loc.T("cli.image_capability_probe"), "", loc)
	if err != nil {
		return err
	}
	configureDirectImage(image, config.ImageProviderOpenAICompatible, endpoint, model, key, authMode, probe)
	return nil
}

func configureDirectImage(image *config.ImageGenerationConfig, provider, endpoint, model, key, authMode, probe string) {
	if image.Providers == nil {
		image.Providers = make(map[string]config.ImageProviderConfig)
	}
	image.Providers[provider] = config.ImageProviderConfig{BaseURL: endpoint, APIKey: key, AuthMode: authMode, CapabilityProbeURL: probe, Models: []string{model}}
	image.Provider = provider
	image.MapIconProvider = provider
	image.BaseURL = ""
	image.APIKey = ""
	image.Model = model
	image.MapIconModel = model
	image.AutoGenerate = true
}

func configureImagegenBridge(reader *bufio.Reader, image *config.ImageGenerationConfig, loc appi18n.Localizer, updates *pendingEnvUpdates) error {
	previousEndpoint := image.ImagegenBridgeURL
	endpoint, err := promptRequiredValue(reader, loc.T("cli.image_bridge_endpoint"), previousEndpoint, loc)
	if err != nil {
		return err
	}
	model, err := promptRequiredValue(reader, loc.T("cli.image_model"), imageModel(*image, "gpt-image-2"), loc)
	if err != nil {
		return err
	}
	previousToken := image.ImagegenBridgeToken
	token, entered, err := promptSecret(reader, loc.T("cli.image_bridge_token"), credentialForOrigin(previousToken, previousEndpoint, endpoint))
	if err != nil {
		return err
	}
	if token == "" && requiresCredentialReentry(previousToken, previousEndpoint, endpoint) {
		return errors.New(loc.T("cli.credential_reentry_required"))
	}
	if token, err = envBackedSecret(updates, "ONEDAY_IMAGEGEN_BRIDGE_TOKEN", token, entered); err != nil {
		return err
	}
	image.Provider = config.ImageProviderCodexOAuth
	image.MapIconProvider = config.ImageProviderCodexOAuth
	image.ImagegenBridgeURL = endpoint
	if token != "" {
		image.ImagegenBridgeToken = token
	}
	image.Model = model
	image.MapIconModel = model
	image.AutoGenerate = true
	return nil
}

func imageProviderBaseURL(image config.ImageGenerationConfig, provider string) string {
	if configured, ok := image.Providers[provider]; ok && strings.TrimSpace(configured.BaseURL) != "" {
		return configured.BaseURL
	}
	if strings.TrimSpace(image.BaseURL) != "" {
		return image.BaseURL
	}
	if provider == config.ImageProviderOpenAI {
		return "https://api.openai.com/v1"
	}
	return ""
}

func imageModel(image config.ImageGenerationConfig, fallback string) string {
	if strings.TrimSpace(image.Model) != "" {
		return image.Model
	}
	return fallback
}

func imageProviderSecret(image config.ImageGenerationConfig, provider string) string {
	if configured, ok := image.Providers[provider]; ok {
		return configured.APIKey
	}
	if image.Provider == provider {
		return image.APIKey
	}
	return ""
}

func credentialForOrigin(credential, previousEndpoint, endpoint string) string {
	if sameURLOrigin(previousEndpoint, endpoint) {
		return credential
	}
	return ""
}

func requiresCredentialReentry(credential, previousEndpoint, endpoint string) bool {
	return strings.TrimSpace(credential) != "" && !sameURLOrigin(previousEndpoint, endpoint)
}

func sameURLOrigin(left, right string) bool {
	leftOrigin, ok := normalizedURLOrigin(left)
	if !ok {
		return false
	}
	rightOrigin, ok := normalizedURLOrigin(right)
	return ok && leftOrigin == rightOrigin
}

func normalizedURLOrigin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	return scheme + "://" + host + ":" + port, true
}

func configureTTSChoice(reader *bufio.Reader, cfg *config.Config, loc appi18n.Localizer, pending ...*pendingEnvUpdates) error {
	updates := firstPendingEnvUpdate(pending)
	tts := &cfg.AI.TTS
	fmt.Println()
	fmt.Println(loc.T("cli.tts_choice"))
	fmt.Println(loc.T("cli.tts_disabled"))
	fmt.Println(loc.T("cli.tts_cloud"))
	fmt.Println(loc.T("cli.tts_local"))
	fmt.Print(loc.T("cli.media_selection"))
	choice, _ := reader.ReadString('\n')
	switch strings.TrimSpace(choice) {
	case "":
		return nil
	case "1", "disabled", "off":
		disableTTS(tts)
		return nil
	case "2", "cloud":
		return configureCloudTTS(reader, tts, loc, updates)
	case "3", "local":
		return configureLocalTTS(reader, tts, loc)
	default:
		return fmt.Errorf(loc.T("cli.media_selection_invalid"), strings.TrimSpace(choice))
	}
}

func disableTTS(tts *config.TTSConfig) {
	tts.Cloud = config.TTSEndpoint{}
	tts.Local = config.TTSEndpoint{}
}

func configureCloudTTS(reader *bufio.Reader, tts *config.TTSConfig, loc appi18n.Localizer, updates *pendingEnvUpdates) error {
	previousEndpoint := tts.Cloud.BaseURL
	endpoint, err := promptRequiredValue(reader, loc.T("cli.tts_endpoint"), previousEndpoint, loc)
	if err != nil {
		return err
	}
	model, err := promptRequiredValue(reader, loc.T("cli.tts_model"), tts.Cloud.Model, loc)
	if err != nil {
		return err
	}
	voice, err := promptRequiredValue(reader, loc.T("cli.tts_voice"), tts.Cloud.Voice, loc)
	if err != nil {
		return err
	}
	previousKey := tts.Cloud.APIKey
	key, entered, err := promptSecret(reader, loc.T("cli.tts_api_key"), credentialForOrigin(previousKey, previousEndpoint, endpoint))
	if err != nil {
		return err
	}
	if key == "" {
		if requiresCredentialReentry(previousKey, previousEndpoint, endpoint) {
			return errors.New(loc.T("cli.credential_reentry_required"))
		}
		return errors.New(loc.T("cli.secret_required"))
	}
	if key, err = envBackedSecret(updates, "ONEDAY_TTS_API_KEY", key, entered); err != nil {
		return err
	}
	tts.Cloud = config.TTSEndpoint{Enabled: true, BaseURL: endpoint, APIKey: key, Model: model, Voice: voice, Version: tts.Cloud.Version}
	tts.Local.Enabled = false
	return nil
}

func configureLocalTTS(reader *bufio.Reader, tts *config.TTSConfig, loc appi18n.Localizer) error {
	endpoint, err := promptRequiredValue(reader, loc.T("cli.tts_endpoint"), tts.Local.BaseURL, loc)
	if err != nil {
		return err
	}
	model, err := promptRequiredValue(reader, loc.T("cli.tts_model"), tts.Local.Model, loc)
	if err != nil {
		return err
	}
	voice, err := promptRequiredValue(reader, loc.T("cli.tts_voice"), tts.Local.Voice, loc)
	if err != nil {
		return err
	}
	tts.Local = config.TTSEndpoint{Enabled: true, BaseURL: endpoint, Model: model, Voice: voice, Version: tts.Local.Version}
	tts.Cloud.Enabled = false
	return nil
}

func promptRequiredValue(reader *bufio.Reader, label, current string, loc appi18n.Localizer) (string, error) {
	value, err := promptValue(reader, label, current)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf(loc.T("cli.value_required"), strings.ToLower(label))
	}
	return value, nil
}

func promptValue(reader *bufio.Reader, label, current string) (string, error) {
	if strings.TrimSpace(current) == "" {
		fmt.Printf("%s: ", label)
	} else {
		fmt.Printf("%s [%s]: ", label, current)
	}
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = strings.TrimSpace(current)
	}
	return value, nil
}

func promptSecret(reader *bufio.Reader, label, current string) (string, bool, error) {
	fmt.Printf("%s: ", label)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		value = []byte(strings.TrimSpace(string(value)))
		if len(value) == 0 {
			return strings.TrimSpace(current), false, err
		}
		return string(value), true, err
	}
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", false, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(current), false, nil
	}
	return value, true, nil
}

// pendingEnvUpdates holds secret input until setup has completed every prompt
// and validation check. Config receives references immediately, never raw
// values, but the adjacent .env is not touched until the final write boundary.
type pendingEnvUpdates struct {
	secrets map[string]string
	ensure  map[string]struct{}
}

func newPendingEnvUpdates() *pendingEnvUpdates {
	return &pendingEnvUpdates{secrets: make(map[string]string), ensure: make(map[string]struct{})}
}

func firstPendingEnvUpdate(updates []*pendingEnvUpdates) *pendingEnvUpdates {
	if len(updates) == 0 {
		return nil
	}
	return updates[0]
}

func (updates *pendingEnvUpdates) Ensure(name string) {
	if updates == nil || strings.TrimSpace(name) == "" {
		return
	}
	updates.ensure[name] = struct{}{}
}

func (updates *pendingEnvUpdates) SetSecret(name, value string) error {
	if updates == nil {
		return nil
	}
	if strings.TrimSpace(name) == "" || strings.ContainsAny(value, "\r\n") {
		return errors.New("credential cannot be written to .env")
	}
	updates.secrets[name] = value
	return nil
}

func envBackedSecret(updates *pendingEnvUpdates, name, value string, entered bool) (string, error) {
	if !entered || updates == nil {
		return value, nil
	}
	if err := updates.SetSecret(name, value); err != nil {
		return "", err
	}
	return "${" + name + "}", nil
}

func (updates *pendingEnvUpdates) Empty() bool {
	return updates == nil || (len(updates.secrets) == 0 && len(updates.ensure) == 0)
}

func writePendingEnv(path string, updates *pendingEnvUpdates) error {
	if updates.Empty() {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading .env for setup: %w", err)
	}
	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		return errors.New("writing .env for setup: path is a directory")
	}
	content := mergeDotEnvUpdates(data, updates)
	if err := setup.WriteFileAtomic(path, content); err != nil {
		return fmt.Errorf("writing .env for setup: %w", err)
	}
	return nil
}

func mergeDotEnvUpdates(data []byte, updates *pendingEnvUpdates) []byte {
	if updates.Empty() {
		return data
	}
	seen := make(map[string]bool)
	lines := []string{}
	if len(data) > 0 {
		lines = strings.Split(string(data), "\n")
	}
	for index, line := range lines {
		key, ok := dotEnvKey(line)
		if !ok {
			continue
		}
		if value, replace := updates.secrets[key]; replace {
			lines[index] = key + "=" + value
			seen[key] = true
			continue
		}
		if _, ensure := updates.ensure[key]; ensure {
			seen[key] = true
		}
	}
	keys := make([]string, 0, len(updates.secrets)+len(updates.ensure))
	for key := range updates.secrets {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	for key := range updates.ensure {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		value, isSecret := updates.secrets[key]
		if !isSecret {
			value = ""
		}
		lines = append(lines, key+"="+value)
	}
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return []byte(content)
}

func dotEnvKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	key, _, ok := strings.Cut(trimmed, "=")
	key = strings.TrimSpace(key)
	return key, ok && key != ""
}

func runConfigLocale(args []string, out io.Writer) error {
	current, _ := config.Load(resolveConfigPath())
	loc := appi18n.New(appi18n.Resolve(current.Interface.Locale, nil))
	if len(args) != 3 {
		_, _ = fmt.Fprintln(out, loc.T("cli.locale_usage"))
		return errors.New(loc.T("cli.locale_required"))
	}
	value := strings.ToLower(strings.TrimSpace(args[2]))
	if value != "en" && value != "it" && value != "auto" {
		return fmt.Errorf(loc.T("cli.locale_invalid"), value)
	}
	if err := config.UpdateInterfaceLocale(resolveConfigPath(), value); err != nil {
		return err
	}
	loc = appi18n.New(appi18n.Resolve(value, nil))
	label := value
	if value == "auto" {
		label = string(appi18n.Resolve("", nil)) + " (" + loc.T("cli.auto") + ")"
	}
	_, err := fmt.Fprintln(out, loc.T("cli.locale_saved", label))
	return err
}

func configureCodex(reader *bufio.Reader, cfg *config.Config) error {
	loc := configLocalizer(*cfg)
	model, err := promptRequiredModel(reader, loc.T("cli.codex_model"), cfg.AI.Codex.Model, loc)
	if err != nil {
		return err
	}
	cfg.AI.Codex.Model = model
	ensureGenerationModels(&cfg.AI.Generation, model)
	fmt.Print(loc.T("cli.reasoning_prompt"))
	reasoning, _ := reader.ReadString('\n')
	if reasoning = strings.TrimSpace(reasoning); reasoning != "" {
		cfg.AI.Codex.Reasoning = reasoning
	}
	return nil
}

func configureLiteLLM(reader *bufio.Reader, cfg *config.Config) error {
	loc := configLocalizer(*cfg)
	model, err := promptRequiredModel(reader, loc.T("cli.litellm_model"), cfg.AI.LiteLLM.DefaultModel, loc)
	if err != nil {
		return err
	}
	cfg.AI.LiteLLM.DefaultModel = model
	ensureGenerationModels(&cfg.AI.Generation, model)
	return nil
}

func configureOpenRouter(reader *bufio.Reader, cfg *config.Config) error {
	loc := configLocalizer(*cfg)
	model, err := promptRequiredModel(reader, loc.T("cli.openrouter_model"), cfg.AI.OpenRouter.DefaultModel, loc)
	if err != nil {
		return err
	}
	cfg.AI.OpenRouter.DefaultModel = model
	ensureGenerationModels(&cfg.AI.Generation, model)
	return nil
}

func promptRequiredModel(reader *bufio.Reader, label, current string, localizers ...appi18n.Localizer) (string, error) {
	loc := appi18n.New(appi18n.English)
	if len(localizers) > 0 {
		loc = localizers[0]
	}
	current = strings.TrimSpace(current)
	if current == "" {
		fmt.Printf("%s: ", label)
	} else {
		fmt.Printf("%s [%s]: ", label, current)
	}
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		value = current
	}
	if value == "" {
		return "", fmt.Errorf(loc.T("cli.model_required"), strings.ToLower(label))
	}
	return value, nil
}

func ensureGenerationModels(generation *config.GenerationConfig, model string) {
	model = strings.TrimSpace(model)
	if model == "" {
		return
	}
	if strings.TrimSpace(generation.UtilityModel) == "" {
		generation.UtilityModel = model
	}
	if strings.TrimSpace(generation.RepairModel) == "" {
		generation.RepairModel = model
	}
	if len(generation.RepairFallbackModels) == 0 {
		generation.RepairFallbackModels = []string{model}
	}
}

func configureRAGChoice(reader *bufio.Reader, cfg *config.Config, localizers ...appi18n.Localizer) error {
	loc := appi18n.New(appi18n.English)
	if len(localizers) > 0 {
		loc = localizers[0]
	}
	fmt.Println()
	fmt.Println(loc.SetupPresentation("rag_title", "Choose RAG embeddings:"))
	fmt.Println(loc.SetupPresentation("rag_remote", "  1) Remote provider from current AI config"))
	fmt.Println(loc.SetupPresentation("rag_ollama", "  2) Local Ollama embeddings"))
	fmt.Println(loc.SetupPresentation("rag_custom", "  3) Custom local embedding endpoint"))
	fmt.Println(loc.SetupPresentation("rag_off", "  4) Disable RAG"))
	fmt.Print(loc.SetupPresentation("selection", "Selection [1]: "))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}
	switch choice {
	case "1":
		return configureRemoteRAG(reader, cfg)
	case "2":
		return configureLocalRAG(reader, cfg)
	case "3":
		return configureCustomLocalRAG(reader, cfg)
	case "4":
		cfg.RAG.Enabled = false
		return nil
	default:
		return fmt.Errorf(loc.T("cli.rag_selection_invalid"), choice)
	}
}

func configureRemoteRAG(reader *bufio.Reader, cfg *config.Config) error {
	loc := configLocalizer(*cfg)
	model, err := promptRequiredModel(reader, loc.T("cli.embedding_model"), cfg.AI.Embedding.Model, loc)
	if err != nil {
		return err
	}
	fmt.Print(loc.T("cli.embedding_dimensions", 1536))
	dimText, _ := reader.ReadString('\n')
	dimensions := parsePositiveInt(strings.TrimSpace(dimText), 1536)

	cfg.RAG.Enabled = true
	cfg.RAG.Dimensions = dimensions
	cfg.AI.Embedding.Provider = "auto"
	cfg.AI.Embedding.Model = model
	return nil
}

func configureLocalRAG(reader *bufio.Reader, cfg *config.Config) error {
	fmt.Println()
	loc := configLocalizer(*cfg)
	model, err := promptRequiredModel(reader, loc.T("cli.ollama_model"), cfg.AI.Embedding.Local.Model, loc)
	if err != nil {
		return err
	}
	fmt.Print(loc.T("cli.embedding_dimensions", 1024))
	dimText, _ := reader.ReadString('\n')
	dimensions := parsePositiveInt(strings.TrimSpace(dimText), 1024)

	cfg.RAG.Enabled = true
	cfg.RAG.Dimensions = dimensions
	cfg.AI.Embedding.Provider = "local"
	cfg.AI.Embedding.Local.Enabled = true
	cfg.AI.Embedding.Local.Type = "ollama"
	cfg.AI.Embedding.Local.BaseURL = "http://127.0.0.1:11434"
	cfg.AI.Embedding.Local.Model = model
	cfg.AI.Embedding.Local.Dimensions = dimensions

	fmt.Println(loc.T("cli.ollama_use", model, cfg.AI.Embedding.Local.BaseURL))
	if _, err := exec.LookPath("ollama"); err != nil {
		fmt.Println(loc.T("cli.ollama_missing"))
		return nil
	}
	fmt.Print(loc.T("cli.ollama_pull", model, model))
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "" || answer == "y" || answer == "yes" || answer == "s" || answer == "si" || answer == "sì" {
		if err := runInteractiveCommand("ollama", "pull", model); err != nil {
			fmt.Println(loc.T("cli.ollama_pull_failed", err))
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := smokeLocalEmbedding(ctx, cfg.AI.Embedding.Local, dimensions); err != nil {
		fmt.Println(loc.T("cli.embedding_warn", err))
		fmt.Println(loc.T("cli.ollama_continue"))
	} else {
		fmt.Println(loc.T("cli.embedding_ok", model, dimensions))
	}
	return nil
}

func configureCustomLocalRAG(reader *bufio.Reader, cfg *config.Config) error {
	loc := configLocalizer(*cfg)
	fmt.Print(loc.T("cli.custom_endpoint"))
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000/embed"
	}
	model, err := promptRequiredModel(reader, loc.T("cli.embedding_model_name"), cfg.AI.Embedding.Local.Model, loc)
	if err != nil {
		return err
	}
	fmt.Print(loc.T("cli.embedding_dimensions", 1024))
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
		fmt.Println(loc.T("cli.embedding_warn", err))
		fmt.Println(loc.T("cli.custom_continue"))
	} else {
		fmt.Println(loc.T("cli.embedding_ok", model, dimensions))
	}
	return nil
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
		return fmt.Errorf(cliLocalizer().T("cli.embedding_type_unknown"), local.Type)
	}
	resp, err := emb.Embed(ctx, ai.EmbeddingRequest{Input: "oneday local rag smoke", Model: local.Model})
	if err != nil {
		return err
	}
	if len(resp.Embedding) != dimensions {
		return fmt.Errorf(cliLocalizer().T("cli.embedding_dimension_mismatch"), resp.Model, len(resp.Embedding), dimensions)
	}
	return nil
}

func setupConfigForChoice(cfg config.Config, choice string, localizers ...appi18n.Localizer) (config.Config, error) {
	loc := appi18n.New(appi18n.English)
	if len(localizers) > 0 {
		loc = localizers[0]
	}
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
		cfg.AI.Embedding.Local.Dimensions = 1024
		cfg.RAG.Dimensions = 1024
	default:
		return config.Config{}, fmt.Errorf(loc.T("cli.setup_selection_invalid"), choice)
	}
	return cfg, nil
}

var errDoctorRequiredFailure = errors.New("one or more required readiness probes failed")

func runDoctor(args []string) error {
	return runDoctorTo(args, os.Stdout, setup.DefaultDependencies())
}

func runDoctorTo(args []string, out io.Writer, deps setup.Dependencies) error {
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil && !wantsJSON(args) {
		fmt.Fprintln(out, cliLocalizer().T("cli.env_warn", "unable to load dotenv"))
	}
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return errors.New("configuration could not be loaded; run `oneday setup --reconfigure`")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	report := setup.Run(ctx, cfg, configDisplaySource(), deps)
	if wantsJSON(args) {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
	} else {
		loc := configLocalizer(cfg)
		fmt.Fprintln(out, loc.T("cli.doctor_title"))
		fmt.Fprintf(out, "OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Fprintln(out, loc.T("cli.config_ok", configDisplaySource()))
		for _, probe := range report.Probes {
			fmt.Fprintf(out, "%s: %s [%s] %s\n", probe.Name, probe.Status, probe.Code, loc.DoctorProbeSummary(probe.Code, probe.Summary))
			if action := loc.DoctorProbeAction(probe.Code); action != "" {
				fmt.Fprintln(out, action)
			}
		}
	}
	if report.RequiredFailure() {
		return errDoctorRequiredFailure
	}
	return nil
}

func runConfigShowSafe() error {
	loc := cliLocalizer()
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Println(loc.T("cli.env_warn", err))
	}
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return err
	}
	fmt.Println(loc.T("cli.config_safe_title"))
	fmt.Printf("config_path: %s\n", resolveConfigPath())
	fmt.Printf("data_dir: %s\n", cfg.DataDir)
	interfaceLocale := cfg.Interface.Locale
	if interfaceLocale == "" {
		interfaceLocale = "auto"
	}
	fmt.Printf("interface_locale: %s\n", interfaceLocale)
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
	loc := cliLocalizer()
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Println(loc.T("cli.env_warn", err))
	}
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return err
	}
	fmt.Println(loc.T("cli.rag_benchmark_title"))
	if !cfg.RAG.Enabled {
		fmt.Println(loc.T("cli.rag_disabled"))
		fmt.Println(loc.T("cli.rag_reconfigure"))
		return nil
	}
	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		fmt.Println(loc.T("cli.rag_unavailable", reason))
		fmt.Println(ragBenchmarkAdvice(cfg, reason))
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	start := time.Now()
	emb := embeddingProviderForSpec(spec, 30*time.Second)
	resp, err := emb.Embed(ctx, ai.EmbeddingRequest{Input: "oneday rag benchmark local retrieval smoke", Model: spec.Model})
	if err != nil {
		fmt.Printf("benchmark: FAIL: %v\n", err)
		fmt.Println(ragBenchmarkAdvice(cfg, err.Error()))
		return nil
	}
	latency := time.Since(start)
	status := "OK"
	if spec.Dimensions > 0 && len(resp.Embedding) != spec.Dimensions {
		status = "DIMENSION_MISMATCH"
	}
	fmt.Printf("benchmark: %s provider=%s kind=%s model=%s dimensions=%d expected=%d latency=%s\n", status, spec.Name, spec.Kind, resp.Model, len(resp.Embedding), spec.Dimensions, latency.Round(time.Millisecond))
	if status == "DIMENSION_MISMATCH" {
		fmt.Println(loc.T("cli.rag_dimensions_fix"))
	} else {
		fmt.Println(loc.T("cli.rag_ready"))
	}
	return nil
}

func ragBenchmarkAdvice(cfg config.Config, detail string) string {
	loc := cliLocalizer()
	switch {
	case cfg.AI.Embedding.Provider == "local" && cfg.AI.Embedding.Local.Type == "ollama":
		return loc.T("cli.rag_advice_ollama", cfg.AI.Embedding.Local.Model)
	case cfg.AI.Embedding.Provider == "local":
		return loc.T("cli.rag_advice_local")
	case strings.Contains(detail, "api_key"):
		return loc.T("cli.rag_advice_key")
	default:
		return loc.T("cli.rag_advice_default")
	}
}

func runRAGReindex(args []string) error {
	loc := cliLocalizer()
	if err := config.LoadDotEnv(resolveDotEnvPath()); err != nil {
		fmt.Println(loc.T("cli.env_warn", err))
	}
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return err
	}
	storyID := argValue(args, "--story")
	all := hasArg(args, "--all")
	if storyID == "" && !all {
		fmt.Println(loc.T("cli.rag_reindex_usage"))
		fmt.Println(loc.T("cli.rag_reindex_help"))
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
		fmt.Println(loc.T("cli.rag_reindex_all", removed))
		return nil
	}
	removed, err := store.DeleteByStory(ctx, storyID)
	if err != nil {
		return err
	}
	fmt.Println(loc.T("cli.rag_reindex_story", removed, storyID))
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
	return listStoryPacks([]string{"plugins/story-packs", "plugins/examples"}, os.Stdout)
}

func listStoryPacks(searchPaths []string, w io.Writer) error {
	loc := cliLocalizer()
	packs, err := discoverStoryPacks(searchPaths)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, loc.T("cli.story_packs_title"))
	if len(packs) == 0 {
		fmt.Fprintln(w, loc.T("cli.story_packs_none"))
		return nil
	}
	invalid := 0
	for _, pack := range packs {
		if err := validateStoryPack(pack); err != nil {
			fmt.Fprintf(w, "- %s (%s: %v)\n", pack, loc.T("cli.story_pack_invalid"), err)
			invalid++
			continue
		}
		fmt.Fprintf(w, "- %s\n", pack)
	}
	if invalid > 0 {
		return errors.New(loc.Plural("cli.story_packs_invalid_count", invalid))
	}
	return nil
}

func configureStoryPackChoice(reader *bufio.Reader, localizers ...appi18n.Localizer) error {
	loc := cliLocalizer()
	if len(localizers) > 0 {
		loc = localizers[0]
	}
	packs, err := discoverStoryPacks([]string{"plugins/story-packs", "plugins/examples"})
	if err != nil || len(packs) == 0 {
		return err
	}
	fmt.Println()
	fmt.Println(loc.T("cli.story_pack_optional"))
	fmt.Println("  0) " + loc.T("cli.none"))
	for i, pack := range packs {
		status := ""
		if err := validateStoryPack(pack); err != nil {
			status = " (" + loc.T("cli.story_pack_invalid") + ")"
		}
		fmt.Printf("  %d) %s%s\n", i+1, pack, status)
	}
	fmt.Print(loc.T("cli.story_pack_selection"))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" || choice == "0" {
		return nil
	}
	idx := parsePositiveInt(choice, 0)
	if idx <= 0 || idx > len(packs) {
		return fmt.Errorf(loc.T("cli.story_pack_unknown"), choice)
	}
	if err := validateStoryPack(packs[idx-1]); err != nil {
		return err
	}
	fmt.Println(loc.T("cli.story_pack_selected", packs[idx-1]))
	return nil
}

func runExport(args []string) error {
	outDir := "dist/oneday-export"
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
	manifest := "OneDay safe export\n\nRun:\n  ./oneday setup --reconfigure\n  ./oneday doctor\n\nAlways excluded: config.yaml, .env, oneday_data, databases, generated binaries, local secrets.\n"
	if err := os.WriteFile(filepath.Join(outDir, "SAFE-SETUP.txt"), []byte(manifest), 0644); err != nil {
		return err
	}
	fmt.Println(cliLocalizer().T("cli.export_written", outDir))
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

func validateStoryPack(path string) error {
	_, err := engine.LoadStoryPack(path)
	return err
}

func reportConfigConsistency(cfg config.Config) {
	loc := cliLocalizer()
	warnings := providerConsistencyWarnings(cfg)
	for _, warning := range warnings {
		fmt.Println(loc.T("cli.config_warning", localizedProviderWarning(cfg, warning, loc)))
	}
}

func localizedProviderWarning(cfg config.Config, warning string, loc appi18n.Localizer) string {
	switch warning {
	case "litellm is enabled but ONEDAY_LITELLM_API_KEY/api_key is empty":
		return loc.T("cli.warning_litellm_missing_key")
	case "ONEDAY_LITELLM_API_KEY is set but litellm is disabled":
		return loc.T("cli.warning_litellm_disabled")
	case "openrouter is enabled but ONEDAY_OPENROUTER_API_KEY/api_key is empty":
		return loc.T("cli.warning_openrouter_missing_key")
	case "ONEDAY_OPENROUTER_API_KEY is set but openrouter is disabled":
		return loc.T("cli.warning_openrouter_disabled")
	case "rag is enabled with local embedding provider but local embeddings are disabled":
		return loc.T("cli.warning_local_embeddings_disabled")
	case fmt.Sprintf("rag.dimensions=%d differs from local embedding dimensions=%d", cfg.RAG.Dimensions, cfg.AI.Embedding.Local.Dimensions):
		return loc.T("cli.warning_embedding_dimensions", cfg.RAG.Dimensions, cfg.AI.Embedding.Local.Dimensions)
	default:
		return warning
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

// ensureEnvFile is retained for callers that need only the public narrative
// provider placeholders. Setup itself defers this write until its final
// persistence boundary through pendingEnvUpdates.
func ensureEnvFile() error {
	updates := newPendingEnvUpdates()
	if os.Getenv("ONEDAY_LITELLM_API_KEY") == "" {
		updates.Ensure("ONEDAY_LITELLM_API_KEY")
	}
	if os.Getenv("ONEDAY_OPENROUTER_API_KEY") == "" {
		updates.Ensure("ONEDAY_OPENROUTER_API_KEY")
	}
	return writePendingEnv(resolveDotEnvPath(), updates)
}
