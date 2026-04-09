package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.DataDir != "./oneday_data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "./oneday_data")
	}
	if len(cfg.AI.ProviderPriority) != 3 {
		t.Errorf("ProviderPriority length = %d, want 3", len(cfg.AI.ProviderPriority))
	}
	if cfg.AI.ProviderPriority[0] != "litellm" {
		t.Errorf("ProviderPriority[0] = %q, want %q", cfg.AI.ProviderPriority[0], "litellm")
	}
	if cfg.AI.LiteLLM.BaseURL != "http://llm.example.com/v1" {
		t.Errorf("LiteLLM.BaseURL = %q, want %q", cfg.AI.LiteLLM.BaseURL, "http://llm.example.com/v1")
	}
	if cfg.AI.LiteLLM.DefaultModel != "grok-4.1-fast" {
		t.Errorf("LiteLLM.DefaultModel = %q, want %q", cfg.AI.LiteLLM.DefaultModel, "grok-4.1-fast")
	}
	if cfg.AI.OpenRouter.DefaultModel != "google/gemini-2.5-flash-lite" {
		t.Errorf("OpenRouter.DefaultModel = %q, want %q", cfg.AI.OpenRouter.DefaultModel, "google/gemini-2.5-flash-lite")
	}
	if cfg.AI.OpenRouter.Enabled {
		t.Error("OpenRouter.Enabled = true, want false by default")
	}
	if cfg.AI.Generation.Temperature != 0.8 {
		t.Errorf("Temperature = %f, want 0.8", cfg.AI.Generation.Temperature)
	}
	if !cfg.AI.ASCIIArt.Enabled {
		t.Error("ASCIIArt.Enabled = false, want true by default")
	}
	if cfg.AI.ASCIIArt.Model != "ascii-ambient" {
		t.Errorf("ASCIIArt.Model = %q, want %q", cfg.AI.ASCIIArt.Model, "ascii-ambient")
	}
	if cfg.AI.Embedding.Provider != "auto" {
		t.Errorf("Embedding.Provider = %q, want auto", cfg.AI.Embedding.Provider)
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
    default_model: "gpt-4"
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

func TestValidateInvalidProvider(t *testing.T) {
	cfg := Default()
	cfg.AI.ProviderPriority = []string{"claude-code", "invalid-provider"}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for invalid provider")
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
	// Default: litellm primary, openrouter disabled, claude-code disabled
	enabled := cfg.EnabledProviders()

	if len(enabled) != 1 {
		t.Fatalf("EnabledProviders length = %d, want 1", len(enabled))
	}
	if enabled[0] != "litellm" {
		t.Errorf("EnabledProviders[0] = %q, want litellm", enabled[0])
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
