package config

import (
	"fmt"
	"os"
	"strings"

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
	ProviderPriority []string         `yaml:"provider_priority"`
	ClaudeCode       ClaudeCodeConfig `yaml:"claude_code"`
	LiteLLM          LiteLLMConfig    `yaml:"litellm"`
	OpenRouter       OpenRouterConfig `yaml:"openrouter"`
	Embedding        EmbeddingConfig  `yaml:"embedding"`
	ASCIIArt         ASCIIArtConfig   `yaml:"ascii_art"`
	Generation       GenerationConfig `yaml:"generation"`
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
	Model    string `yaml:"model"`
	Provider string `yaml:"provider"`
}

// ASCIIArtConfig controls optional ambient ASCII-art generation.
type ASCIIArtConfig struct {
	Enabled        bool    `yaml:"enabled"`
	Model          string  `yaml:"model"`
	Temperature    float64 `yaml:"temperature"`
	MaxTokens      int     `yaml:"max_tokens"`
	TimeoutSeconds int     `yaml:"timeout_seconds"`
}

// GenerationConfig for AI text generation.
type GenerationConfig struct {
	Temperature          float64  `yaml:"temperature"`
	MaxTokens            int      `yaml:"max_tokens"`
	TimeoutSeconds       int      `yaml:"timeout_seconds"`
	RepairModel          string   `yaml:"repair_model"`
	RepairFallbackModels []string `yaml:"repair_fallback_models"`
}

// RAGConfig for retrieval-augmented generation.
type RAGConfig struct {
	Enabled        bool `yaml:"enabled"`
	SummarizeEvery int  `yaml:"summarize_every"`
	TopK           int  `yaml:"top_k"`
	Dimensions     int  `yaml:"dimensions"`
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
			ProviderPriority: []string{"litellm", "openrouter", "claude-code"},
			ClaudeCode: ClaudeCodeConfig{
				Enabled: false,
				Binary:  "claude",
			},
			LiteLLM: LiteLLMConfig{
				Enabled:      true,
				BaseURL:      "http://lite.homelab.local/v1",
				DefaultModel: "grok-4.1-fast",
			},
			OpenRouter: OpenRouterConfig{
				Enabled:      false,
				BaseURL:      "https://openrouter.ai/api/v1",
				DefaultModel: "google/gemini-2.5-flash-lite",
			},
			Embedding: EmbeddingConfig{
				Model:    "text-embedding-3-small",
				Provider: "auto",
			},
			ASCIIArt: ASCIIArtConfig{
				Enabled:        true,
				Model:          "ascii-ambient",
				Temperature:    0.4,
				MaxTokens:      400,
				TimeoutSeconds: 25,
			},
			Generation: GenerationConfig{
				Temperature:          0.8,
				MaxTokens:            2048,
				TimeoutSeconds:       60,
				RepairModel:          "gemini-3.1-flash-lite-preview",
				RepairFallbackModels: []string{"grok-4.1-fast"},
			},
		},
		RAG: RAGConfig{
			Enabled:        true,
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
	if c.AI.ASCIIArt.Enabled {
		if c.AI.ASCIIArt.MaxTokens <= 0 {
			return fmt.Errorf("ai.ascii_art.max_tokens must be positive when ascii art is enabled")
		}
		if c.AI.ASCIIArt.TimeoutSeconds <= 0 {
			return fmt.Errorf("ai.ascii_art.timeout_seconds must be positive when ascii art is enabled")
		}
	}
	switch c.AI.Embedding.Provider {
	case "", "auto", "litellm", "openrouter":
	default:
		return fmt.Errorf("ai.embedding.provider must be one of: auto, litellm, openrouter")
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

// RepairModelCandidates returns the ordered list of models to try for repair
// passes, with duplicates and blanks removed.
func (g GenerationConfig) RepairModelCandidates() []string {
	seen := map[string]bool{}
	var out []string

	appendModel := func(model string) {
		model = strings.TrimSpace(model)
		if model == "" || seen[model] {
			return
		}
		seen[model] = true
		out = append(out, model)
	}

	appendModel(g.RepairModel)
	for _, model := range g.RepairFallbackModels {
		appendModel(model)
	}

	return out
}
