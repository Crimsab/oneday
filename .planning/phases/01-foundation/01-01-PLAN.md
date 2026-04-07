---
phase: 1
plan: 1.1
title: "Configuration system with YAML parsing and provider priority chain"
wave: 1
depends_on: []
files_modified:
  - internal/config/config.go
  - internal/config/config_test.go
  - go.mod
  - go.sum
requirements_addressed: [CONF-01, CONF-02]
autonomous: true
---

# Plan 1.1: Configuration System

## Objective

Implement the global configuration system that loads `config.yaml`, provides typed access to all settings (AI providers, endpoints, keys, game settings), and exposes the AI provider priority chain. This is the foundation every other package depends on.

## must_haves

- Config struct fully represents config.example.yaml structure
- YAML file loads and unmarshals without error
- Provider priority chain is a slice of strings, configurable
- Missing config file returns a sensible default config
- Validation catches invalid provider names in the priority chain

## Tasks

### Task 1: Add gopkg.in/yaml.v3 dependency

<read_first>
- go.mod
</read_first>

<action>
Run `go get gopkg.in/yaml.v3` to add the YAML parsing dependency.
</action>

<acceptance_criteria>
- `grep "gopkg.in/yaml.v3" go.mod` returns a match
</acceptance_criteria>

### Task 2: Create config struct and loader

<read_first>
- config.example.yaml
- docs/design.md (Section: Project Structure, lines 22-42)
</read_first>

<action>
Create `internal/config/config.go` with the following:

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for OneDay.
type Config struct {
	DataDir string     `yaml:"data_dir"`
	AI      AIConfig   `yaml:"ai"`
	RAG     RAGConfig  `yaml:"rag"`
	Game    GameConfig `yaml:"game"`
}

// AIConfig holds all AI provider settings.
type AIConfig struct {
	ProviderPriority []string          `yaml:"provider_priority"`
	ClaudeCode       ClaudeCodeConfig  `yaml:"claude_code"`
	LiteLLM          LiteLLMConfig     `yaml:"litellm"`
	OpenRouter       OpenRouterConfig  `yaml:"openrouter"`
	Embedding        EmbeddingConfig   `yaml:"embedding"`
	Generation       GenerationConfig  `yaml:"generation"`
}

// ClaudeCodeConfig for the Claude Code CLI provider.
type ClaudeCodeConfig struct {
	Enabled bool   `yaml:"enabled"`
	Binary  string `yaml:"binary"`
}

// LiteLLMConfig for the LiteLLM proxy provider.
type LiteLLMConfig struct {
	Enabled      bool   `yaml:"enabled"`
	BaseURL      string `yaml:"base_url"`
	APIKey       string `yaml:"api_key"`
	DefaultModel string `yaml:"default_model"`
}

// OpenRouterConfig for the OpenRouter provider.
type OpenRouterConfig struct {
	Enabled      bool   `yaml:"enabled"`
	BaseURL      string `yaml:"base_url"`
	APIKey       string `yaml:"api_key"`
	DefaultModel string `yaml:"default_model"`
}

// EmbeddingConfig for RAG embedding model.
type EmbeddingConfig struct {
	Model string `yaml:"model"`
}

// GenerationConfig for AI text generation.
type GenerationConfig struct {
	Temperature    float64 `yaml:"temperature"`
	MaxTokens      int     `yaml:"max_tokens"`
	TimeoutSeconds int     `yaml:"timeout_seconds"`
}

// RAGConfig for retrieval-augmented generation.
type RAGConfig struct {
	SummarizeEvery int `yaml:"summarize_every"`
	TopK           int `yaml:"top_k"`
	Dimensions     int `yaml:"dimensions"`
}

// GameConfig for gameplay settings.
type GameConfig struct {
	AutosaveEvery    int  `yaml:"autosave_every"`
	TypewriterEffect bool `yaml:"typewriter_effect"`
	TypewriterSpeed  int  `yaml:"typewriter_speed"`
}

// validProviders is the set of recognized provider names.
var validProviders = map[string]bool{
	"claude-code": true,
	"litellm":     true,
	"openrouter":  true,
}

// Default returns a Config with sensible defaults matching config.example.yaml.
func Default() Config {
	return Config{
		DataDir: "./oneday_data",
		AI: AIConfig{
			ProviderPriority: []string{"claude-code", "litellm", "openrouter"},
			ClaudeCode: ClaudeCodeConfig{
				Enabled: true,
				Binary:  "claude",
			},
			LiteLLM: LiteLLMConfig{
				Enabled:      true,
				BaseURL:      "http://ai-proxy:4000/v1",
				DefaultModel: "claude-sonnet-4-6",
			},
			OpenRouter: OpenRouterConfig{
				Enabled:      false,
				BaseURL:      "https://openrouter.ai/api/v1",
				DefaultModel: "anthropic/claude-sonnet-4-6",
			},
			Embedding: EmbeddingConfig{
				Model: "text-embedding-3-small",
			},
			Generation: GenerationConfig{
				Temperature:    0.8,
				MaxTokens:      2048,
				TimeoutSeconds: 60,
			},
		},
		RAG: RAGConfig{
			SummarizeEvery: 10,
			TopK:           5,
			Dimensions:     1536,
		},
		Game: GameConfig{
			AutosaveEvery:    5,
			TypewriterEffect: true,
			TypewriterSpeed:  80,
		},
	}
}

// Load reads a YAML config file and returns a Config.
// If the file does not exist, returns Default().
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// Validate checks that the config is internally consistent.
func (c *Config) Validate() error {
	if len(c.AI.ProviderPriority) == 0 {
		return fmt.Errorf("ai.provider_priority must have at least one provider")
	}
	for _, p := range c.AI.ProviderPriority {
		if !validProviders[p] {
			return fmt.Errorf("unknown provider in priority chain: %q (valid: claude-code, litellm, openrouter)", p)
		}
	}
	if c.AI.Generation.MaxTokens <= 0 {
		return fmt.Errorf("ai.generation.max_tokens must be positive")
	}
	if c.AI.Generation.TimeoutSeconds <= 0 {
		return fmt.Errorf("ai.generation.timeout_seconds must be positive")
	}
	return nil
}

// EnabledProviders returns the provider priority chain filtered to only enabled providers.
func (c *Config) EnabledProviders() []string {
	enabled := make([]string, 0, len(c.AI.ProviderPriority))
	for _, name := range c.AI.ProviderPriority {
		switch name {
		case "claude-code":
			if c.AI.ClaudeCode.Enabled {
				enabled = append(enabled, name)
			}
		case "litellm":
			if c.AI.LiteLLM.Enabled {
				enabled = append(enabled, name)
			}
		case "openrouter":
			if c.AI.OpenRouter.Enabled {
				enabled = append(enabled, name)
			}
		}
	}
	return enabled
}
```
</action>

<acceptance_criteria>
- `grep "type Config struct" internal/config/config.go` matches
- `grep "type AIConfig struct" internal/config/config.go` matches
- `grep "func Load" internal/config/config.go` matches
- `grep "func Default" internal/config/config.go` matches
- `grep "func.*Validate" internal/config/config.go` matches
- `grep "func.*EnabledProviders" internal/config/config.go` matches
- `grep "ProviderPriority" internal/config/config.go` matches
</acceptance_criteria>

### Task 3: Create config tests

<read_first>
- internal/config/config.go (just created)
- config.example.yaml
</read_first>

<action>
Create `internal/config/config_test.go` with table-driven tests:

```go
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
	if cfg.AI.ProviderPriority[0] != "claude-code" {
		t.Errorf("ProviderPriority[0] = %q, want %q", cfg.AI.ProviderPriority[0], "claude-code")
	}
	if cfg.AI.Generation.Temperature != 0.8 {
		t.Errorf("Temperature = %f, want 0.8", cfg.AI.Generation.Temperature)
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

func TestEnabledProviders(t *testing.T) {
	cfg := Default()
	// Default: claude-code enabled, litellm enabled, openrouter disabled
	enabled := cfg.EnabledProviders()

	if len(enabled) != 2 {
		t.Fatalf("EnabledProviders length = %d, want 2", len(enabled))
	}
	if enabled[0] != "claude-code" {
		t.Errorf("EnabledProviders[0] = %q, want claude-code", enabled[0])
	}
	if enabled[1] != "litellm" {
		t.Errorf("EnabledProviders[1] = %q, want litellm", enabled[1])
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
```
</action>

<acceptance_criteria>
- `grep "func TestDefault" internal/config/config_test.go` matches
- `grep "func TestLoadMissingFile" internal/config/config_test.go` matches
- `grep "func TestValidateInvalidProvider" internal/config/config_test.go` matches
- `grep "func TestEnabledProviders" internal/config/config_test.go` matches
- `go test ./internal/config/ -v` passes all tests
</acceptance_criteria>
