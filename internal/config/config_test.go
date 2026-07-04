package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.ConfigVersion != 2 {
		t.Errorf("ConfigVersion = %d, want 2", cfg.ConfigVersion)
	}

	if cfg.DataDir != "./oneday_data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "./oneday_data")
	}
	if len(cfg.AI.ProviderPriority) != 4 {
		t.Errorf("ProviderPriority length = %d, want 4", len(cfg.AI.ProviderPriority))
	}
	if cfg.AI.ProviderPriority[0] != "litellm" {
		t.Errorf("ProviderPriority[0] = %q, want %q", cfg.AI.ProviderPriority[0], "litellm")
	}
	if cfg.AI.LiteLLM.BaseURL != "http://llm.example.com/v1" {
		t.Errorf("LiteLLM.BaseURL = %q, want %q", cfg.AI.LiteLLM.BaseURL, "http://llm.example.com/v1")
	}
	if cfg.AI.LiteLLM.Enabled {
		t.Error("LiteLLM.Enabled = true, want false by default")
	}
	if cfg.AI.LiteLLM.DefaultModel != "" {
		t.Errorf("LiteLLM.DefaultModel = %q, want empty default", cfg.AI.LiteLLM.DefaultModel)
	}
	if cfg.AI.OpenRouter.DefaultModel != "" {
		t.Errorf("OpenRouter.DefaultModel = %q, want empty default", cfg.AI.OpenRouter.DefaultModel)
	}
	if cfg.AI.OpenRouter.Enabled {
		t.Error("OpenRouter.Enabled = true, want false by default")
	}
	if cfg.AI.Codex.Enabled {
		t.Error("Codex.Enabled = true, want false by default")
	}
	if cfg.AI.Generation.Temperature != 0.8 {
		t.Errorf("Temperature = %f, want 0.8", cfg.AI.Generation.Temperature)
	}
	if cfg.AI.Generation.UtilityModel != "" {
		t.Errorf("UtilityModel = %q, want empty default", cfg.AI.Generation.UtilityModel)
	}
	if cfg.AI.ASCIIArt.Enabled {
		t.Error("ASCIIArt.Enabled = true, want false by default")
	}
	if cfg.AI.ASCIIArt.Model != "" {
		t.Errorf("ASCIIArt.Model = %q, want empty default", cfg.AI.ASCIIArt.Model)
	}
	if cfg.AI.ImageGeneration.Model != "" {
		t.Errorf("ImageGeneration.Model = %q, want empty default", cfg.AI.ImageGeneration.Model)
	}
	if cfg.AI.Embedding.Provider != "auto" {
		t.Errorf("Embedding.Provider = %q, want auto", cfg.AI.Embedding.Provider)
	}
	if cfg.AI.Embedding.Model != "" {
		t.Errorf("Embedding.Model = %q, want empty default", cfg.AI.Embedding.Model)
	}
	if cfg.RAG.Enabled {
		t.Error("RAG.Enabled = true, want false by default")
	}
	if cfg.Game.VisiblePrivateThoughts {
		t.Error("VisiblePrivateThoughts = true, want false in player mode defaults")
	}
	if cfg.Game.RewardBudget != "balanced" {
		t.Errorf("RewardBudget = %q, want balanced", cfg.Game.RewardBudget)
	}
}

func TestMigrateFillsLocalEmbeddingDefaults(t *testing.T) {
	cfg := Default()
	cfg.ConfigVersion = 1
	cfg.AI.Embedding.Local.Type = ""
	cfg.AI.Embedding.Local.BaseURL = ""
	cfg.AI.Embedding.Local.Model = ""
	cfg.AI.Embedding.Local.Dimensions = 0
	cfg.AI.Generation.UtilityModel = ""
	cfg.Game.RewardBudget = ""

	cfg.Migrate()

	if cfg.ConfigVersion != 2 {
		t.Fatalf("ConfigVersion = %d, want 2", cfg.ConfigVersion)
	}
	if cfg.AI.Embedding.Local.Type != "ollama" || cfg.AI.Embedding.Local.Model != "" || cfg.AI.Embedding.Local.Dimensions != 1024 {
		t.Fatalf("local embedding defaults not migrated: %#v", cfg.AI.Embedding.Local)
	}
	if cfg.AI.Generation.UtilityModel != "" {
		t.Fatalf("UtilityModel = %q", cfg.AI.Generation.UtilityModel)
	}
	if cfg.Game.RewardBudget != "balanced" {
		t.Fatalf("RewardBudget = %q, want balanced", cfg.Game.RewardBudget)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	// Should return defaults
	if cfg.DataDir != "./oneday_data" {
		t.Errorf("DataDir = %q, want default", cfg.DataDir)
	}
}

func TestLoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yaml := `
data_dir: "/tmp/test_data"
ai:
  provider_priority:
    - litellm
    - openrouter
  litellm:
    enabled: true
    base_url: "http://localhost:4000/v1"
    default_model: "test-model"
  generation:
    temperature: 0.5
    max_tokens: 1024
    timeout_seconds: 30
game:
  autosave_every: 10
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "/tmp/test_data" {
		t.Errorf("DataDir = %q, want /tmp/test_data", cfg.DataDir)
	}
	if len(cfg.AI.ProviderPriority) != 2 {
		t.Errorf("ProviderPriority length = %d, want 2", len(cfg.AI.ProviderPriority))
	}
	if cfg.AI.Generation.Temperature != 0.5 {
		t.Errorf("Temperature = %f, want 0.5", cfg.AI.Generation.Temperature)
	}
	if cfg.Game.AutosaveEvery != 10 {
		t.Errorf("AutosaveEvery = %d, want 10", cfg.Game.AutosaveEvery)
	}
}

func TestLoadExpandsEnvironmentVariables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	t.Setenv("ONEDAY_TEST_API_KEY", "test-secret-key")

	yaml := `
ai:
  provider_priority:
    - litellm
  litellm:
    enabled: true
    api_key: "${ONEDAY_TEST_API_KEY}"
    default_model: test-litellm-model
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AI.LiteLLM.APIKey != "test-secret-key" {
		t.Errorf("LiteLLM.APIKey = %q, want expanded env value", cfg.AI.LiteLLM.APIKey)
	}
}

func TestLoadForEditDoesNotExpandEnvironmentVariables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	t.Setenv("ONEDAY_TEST_API_KEY", "test-secret-key")

	yaml := `
ai:
  provider_priority:
    - litellm
  litellm:
    enabled: true
    api_key: "${ONEDAY_TEST_API_KEY}"
    default_model: test-litellm-model
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadForEdit(path)
	if err != nil {
		t.Fatalf("LoadForEdit: %v", err)
	}
	if cfg.AI.LiteLLM.APIKey != "${ONEDAY_TEST_API_KEY}" {
		t.Errorf("LiteLLM.APIKey = %q, want placeholder preserved", cfg.AI.LiteLLM.APIKey)
	}
}

func TestLoadDotEnvDoesNotOverwriteExistingEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	t.Setenv("ONEDAY_EXISTING", "from-shell")

	content := `
# comment
ONEDAY_EXISTING=from-file
ONEDAY_NEW="from dotenv"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("ONEDAY_EXISTING"); got != "from-shell" {
		t.Errorf("ONEDAY_EXISTING = %q, want shell value", got)
	}
	if got := os.Getenv("ONEDAY_NEW"); got != "from dotenv" {
		t.Errorf("ONEDAY_NEW = %q, want dotenv value", got)
	}
}

func TestValidateInvalidProvider(t *testing.T) {
	cfg := Default()
	cfg.AI.ProviderPriority = []string{"claude-code", "invalid-provider"}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid provider")
	}
}

func TestValidateInvalidCodexReasoning(t *testing.T) {
	cfg := Default()
	cfg.AI.Codex.Reasoning = "turbo"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid codex reasoning")
	}
}

func TestValidateEmptyUtilityModel(t *testing.T) {
	cfg := Default()
	cfg.AI.LiteLLM.Enabled = true
	cfg.AI.LiteLLM.DefaultModel = "test-litellm-model"
	cfg.AI.Generation.UtilityModel = " "

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty utility model")
	}
}

func TestValidateInvalidRewardBudget(t *testing.T) {
	cfg := Default()
	cfg.Game.RewardBudget = "shower-of-gold"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid reward budget")
	}
}

func TestRepairModelCandidatesUsesUtilityOnlyWhenRepairMissing(t *testing.T) {
	withRepair := GenerationConfig{
		UtilityModel:         "test-utility-model",
		RepairModel:          "repair-primary",
		RepairFallbackModels: []string{"repair-fallback"},
	}
	candidates := withRepair.RepairModelCandidates()
	if len(candidates) != 2 || candidates[0] != "repair-primary" || candidates[1] != "repair-fallback" {
		t.Fatalf("RepairModelCandidates with repair = %#v", candidates)
	}

	withoutRepair := GenerationConfig{
		UtilityModel:         "test-utility-model",
		RepairFallbackModels: []string{"repair-fallback"},
	}
	candidates = withoutRepair.RepairModelCandidates()
	if len(candidates) != 2 || candidates[0] != "test-utility-model" || candidates[1] != "repair-fallback" {
		t.Fatalf("RepairModelCandidates without repair = %#v", candidates)
	}
}

func TestBuildModelRoutingSettings(t *testing.T) {
	cfg := Default()
	cfg.AI.ProviderPriority = []string{"codex", "litellm", "openrouter", "claude-code"}
	cfg.AI.Codex.Enabled = true
	cfg.AI.Codex.Model = "test-narrative-model"
	cfg.AI.Generation.UtilityModel = "test-narrative-model"
	cfg.AI.Generation.RepairFallbackModels = []string{"test-narrative-model", "test-narrative-model"}

	settings := BuildModelRoutingSettings("/tmp/config.yaml", cfg, "revision-1")

	if settings.ConfigRevision != "revision-1" {
		t.Fatalf("ConfigRevision = %q", settings.ConfigRevision)
	}
	if settings.Active.Provider != "codex" {
		t.Fatalf("Active.Provider = %q, want codex", settings.Active.Provider)
	}
	if settings.Active.NarrativeModel != "test-narrative-model" {
		t.Fatalf("Active.NarrativeModel = %q", settings.Active.NarrativeModel)
	}
	if len(settings.Providers) != 4 {
		t.Fatalf("Providers length = %d, want 4", len(settings.Providers))
	}
	if got := settings.RepairModels; len(got) != 1 || got[0] != "test-narrative-model" {
		t.Fatalf("RepairModels = %#v", got)
	}
}

func TestUpdateModelRoutingSettingsRequiresFreshRevision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data, err := Marshal(Default())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = UpdateModelRoutingSettings(path, ModelRoutingUpdate{})
	var missingRevisionErr ModelRoutingError
	if !errors.As(err, &missingRevisionErr) {
		t.Fatalf("expected missing revision ModelRoutingError, got %T %v", err, err)
	}
	if missingRevisionErr.Code != ModelRoutingErrorValidation {
		t.Fatalf("missing revision code = %q, want %q", missingRevisionErr.Code, ModelRoutingErrorValidation)
	}

	_, err = UpdateModelRoutingSettings(path, ModelRoutingUpdate{BaseRevision: "stale"})
	var routingErr ModelRoutingError
	if !errors.As(err, &routingErr) {
		t.Fatalf("expected ModelRoutingError, got %T %v", err, err)
	}
	if routingErr.Code != ModelRoutingErrorStale {
		t.Fatalf("routing error code = %q, want %q", routingErr.Code, ModelRoutingErrorStale)
	}
}

func TestReadModelRoutingSettingsRejectsDuplicateYamlKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	raw := []byte(`config_version: 2
ai:
  provider_priority: [codex, litellm, openrouter, claude-code]
ai:
  provider_priority: [litellm, codex, openrouter, claude-code]
`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadModelRoutingSettings(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate YAML key") {
		t.Fatalf("ReadModelRoutingSettings error = %v, want duplicate YAML key", err)
	}
}

func TestPatchModelRoutingYAMLRejectsWrongTypeEditablePath(t *testing.T) {
	raw := []byte(`config_version: 2
ai:
  provider_priority: [codex, litellm, openrouter, claude-code]
  generation: wrong-type
  litellm:
    enabled: true
    default_model: test-litellm-model
  codex:
    enabled: true
    model: test-codex-model
    reasoning: off
`)
	_, err := patchModelRoutingYAML(raw, Default())
	if err == nil || !strings.Contains(err.Error(), "ai.generation must be a mapping") {
		t.Fatalf("patchModelRoutingYAML error = %v, want wrong-type path error", err)
	}
}

func TestUpdateModelRoutingSettingsPatchesModelFieldsWithoutClobberingYaml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	raw := []byte(`# keep this top-level comment
config_version: 2
unknown_top: keep-me
ai:
  provider_priority:
    - codex
    - litellm
    - openrouter
    - claude-code
  litellm:
    enabled: true
    base_url: ${LITELLM_BASE_URL}
    default_model: test-litellm-model
  codex:
    enabled: true
    model: test-codex-model
    reasoning: off
`)
	if err := os.WriteFile(path, raw, 0640); err != nil {
		t.Fatal(err)
	}
	settings, err := ReadModelRoutingSettings(path)
	if err != nil {
		t.Fatal(err)
	}

	nextModel := "test-litellm-model-updated"
	next, err := UpdateModelRoutingSettings(path, ModelRoutingUpdate{
		BaseRevision: settings.ConfigRevision,
		Providers: []ModelProviderUpdate{
			{ID: "litellm", Model: &nextModel},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if next.ConfigRevision == "" || next.ConfigRevision == settings.ConfigRevision {
		t.Fatalf("revision was not updated: before=%q after=%q", settings.ConfigRevision, next.ConfigRevision)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, needle := range []string{"# keep this top-level comment", "unknown_top: keep-me", "${LITELLM_BASE_URL}", "default_model: test-litellm-model-updated"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("updated config missing %q:\n%s", needle, text)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0640 {
		t.Fatalf("mode = %v, want 0640", got)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
}

func TestApplyModelRoutingUpdate(t *testing.T) {
	cfg := Default()
	cfg.AI.LiteLLM.Enabled = true
	cfg.AI.LiteLLM.DefaultModel = "test-litellm-model"
	priority := []string{"openrouter", "litellm", "codex", "claude-code"}
	openRouterEnabled := true
	openRouterModel := "test-openrouter-model"
	codexEnabled := false
	utility := "test-utility-model"
	repair := "test-repair-model"
	fallbacks := []string{"test-fallback-one", "test-fallback-two"}
	image := "test-image-model"
	ascii := "test-ascii-model"

	err := ApplyModelRoutingUpdate(&cfg, ModelRoutingUpdate{
		ProviderPriority: &priority,
		Providers: []ModelProviderUpdate{
			{ID: "openrouter", Enabled: &openRouterEnabled, Model: &openRouterModel},
			{ID: "codex", Enabled: &codexEnabled},
		},
		UtilityModel:         &utility,
		RepairModel:          &repair,
		RepairFallbackModels: &fallbacks,
		ImageModel:           &image,
		ASCIIModel:           &ascii,
	})
	if err != nil {
		t.Fatalf("ApplyModelRoutingUpdate: %v", err)
	}
	if got := cfg.EnabledProviders(); len(got) != 2 || got[0] != "openrouter" || got[1] != "litellm" {
		t.Fatalf("EnabledProviders = %#v", got)
	}
	if cfg.AI.OpenRouter.DefaultModel != openRouterModel {
		t.Fatalf("OpenRouter.DefaultModel = %q", cfg.AI.OpenRouter.DefaultModel)
	}
	if cfg.AI.Generation.UtilityModel != utility || cfg.AI.Generation.RepairModel != repair {
		t.Fatalf("generation models not updated: %#v", cfg.AI.Generation)
	}
	if cfg.AI.ImageGeneration.Model != image {
		t.Fatalf("ImageGeneration.Model = %q", cfg.AI.ImageGeneration.Model)
	}
	if cfg.AI.ASCIIArt.Model != ascii {
		t.Fatalf("ASCIIArt.Model = %q", cfg.AI.ASCIIArt.Model)
	}
}

func TestApplyModelRoutingUpdateRejectsNoEnabledProviders(t *testing.T) {
	cfg := Default()
	disabled := false
	err := ApplyModelRoutingUpdate(&cfg, ModelRoutingUpdate{
		Providers: []ModelProviderUpdate{
			{ID: "litellm", Enabled: &disabled},
			{ID: "openrouter", Enabled: &disabled},
			{ID: "codex", Enabled: &disabled},
			{ID: "claude-code", Enabled: &disabled},
		},
	})
	if err == nil {
		t.Fatal("expected error when all providers are disabled")
	}
}

func TestValidateEmptyPriority(t *testing.T) {
	cfg := Default()
	cfg.AI.ProviderPriority = []string{}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty priority chain")
	}
}

func TestValidateInvalidEmbeddingProvider(t *testing.T) {
	cfg := Default()
	cfg.AI.Embedding.Provider = "claude-code"

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid embedding provider")
	}
}

func TestEnabledProviders(t *testing.T) {
	cfg := Default()
	enabled := cfg.EnabledProviders()

	if len(enabled) != 0 {
		t.Fatalf("EnabledProviders length = %d, want 0", len(enabled))
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
