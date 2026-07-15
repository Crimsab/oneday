package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for OneDay.
type Config struct {
	ConfigVersion int        `yaml:"config_version"`
	DataDir       string     `yaml:"data_dir"`
	AI            AIConfig   `yaml:"ai"`
	RAG           RAGConfig  `yaml:"rag"`
	Game          GameConfig `yaml:"game"`
}

// AIConfig holds all AI provider settings.
type AIConfig struct {
	ProviderPriority []string              `yaml:"provider_priority"`
	Codex            CodexConfig           `yaml:"codex"`
	ClaudeCode       ClaudeCodeConfig      `yaml:"claude_code"`
	LiteLLM          LiteLLMConfig         `yaml:"litellm"`
	OpenRouter       OpenRouterConfig      `yaml:"openrouter"`
	Embedding        EmbeddingConfig       `yaml:"embedding"`
	ASCIIArt         ASCIIArtConfig        `yaml:"ascii_art"`
	ImageGeneration  ImageGenerationConfig `yaml:"image_generation"`
	TTS              TTSConfig             `yaml:"tts"`
	Generation       GenerationConfig      `yaml:"generation"`
}

type TTSConfig struct {
	OutputDir      string      `yaml:"output_dir"`
	TimeoutSeconds int         `yaml:"timeout_seconds"`
	ProviderOrder  []string    `yaml:"provider_order"`
	Cloud          TTSEndpoint `yaml:"cloud"`
	Local          TTSEndpoint `yaml:"local"`
}

type TTSEndpoint struct {
	Enabled   bool     `yaml:"enabled"`
	BaseURL   string   `yaml:"base_url"`
	APIKey    string   `yaml:"api_key"`
	Model     string   `yaml:"model"`
	Voice     string   `yaml:"voice"`
	Version   string   `yaml:"version"`
	Languages []string `yaml:"languages"`
}

// CodexConfig for the OpenAI Codex CLI provider.
type CodexConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Binary    string `yaml:"binary"`
	Model     string `yaml:"model"`
	Reasoning string `yaml:"reasoning"`
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
	Model    string               `yaml:"model"`
	Provider string               `yaml:"provider"`
	Local    LocalEmbeddingConfig `yaml:"local"`
}

// LocalEmbeddingConfig configures Ollama or a compatible local HTTP embedding backend.
// It is disabled by default and selected only when explicitly enabled.
type LocalEmbeddingConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Type       string `yaml:"type"`
	BaseURL    string `yaml:"base_url"`
	Model      string `yaml:"model"`
	Dimensions int    `yaml:"dimensions"`
}

// ASCIIArtConfig controls optional ambient ASCII-art generation.
type ASCIIArtConfig struct {
	Enabled        bool    `yaml:"enabled"`
	Model          string  `yaml:"model"`
	Temperature    float64 `yaml:"temperature"`
	MaxTokens      int     `yaml:"max_tokens"`
	TimeoutSeconds int     `yaml:"timeout_seconds"`
}

// ImageGenerationConfig controls non-blocking visual asset generation.
type ImageGenerationConfig struct {
	Provider                      string                         `yaml:"provider"`
	MapIconProvider               string                         `yaml:"map_icon_provider"`
	BaseURL                       string                         `yaml:"base_url"`
	APIKey                        string                         `yaml:"api_key"`
	Model                         string                         `yaml:"model"`
	MapIconModel                  string                         `yaml:"map_icon_model"`
	OpenClawBridgeURL             string                         `yaml:"openclaw_bridge_url"`
	ImagegenBridgeURL             string                         `yaml:"imagegen_bridge_url"`
	ImagegenBridgeToken           string                         `yaml:"imagegen_bridge_token"`
	ImagegenBridgeProvider        string                         `yaml:"imagegen_bridge_provider"`
	ImagegenBridgeMapIconProvider string                         `yaml:"imagegen_bridge_map_icon_provider"`
	ImagegenBridgeFallbacks       []string                       `yaml:"imagegen_bridge_fallbacks"`
	ImagegenBridgeFallbackPolicy  string                         `yaml:"imagegen_bridge_fallback_policy"`
	ImagegenBridgeCompatibility   string                         `yaml:"imagegen_bridge_compatibility"`
	DefaultSize                   string                         `yaml:"default_size"`
	LocationSize                  string                         `yaml:"location_size"`
	CharacterSize                 string                         `yaml:"character_size"`
	DefaultResolution             string                         `yaml:"default_resolution"`
	LocationResolution            string                         `yaml:"location_resolution"`
	CharacterResolution           string                         `yaml:"character_resolution"`
	DefaultAspectRatio            string                         `yaml:"default_aspect_ratio"`
	LocationAspectRatio           string                         `yaml:"location_aspect_ratio"`
	CharacterAspectRatio          string                         `yaml:"character_aspect_ratio"`
	Quality                       string                         `yaml:"quality"`
	OutputFormat                  string                         `yaml:"output_format"`
	Background                    string                         `yaml:"background"`
	TimeoutSeconds                int                            `yaml:"timeout_seconds"`
	AutoGenerate                  bool                           `yaml:"auto_generate"`
	AppendNegativePrompt          bool                           `yaml:"append_negative_prompt"`
	Providers                     map[string]ImageProviderConfig `yaml:"providers,omitempty"`
}

// ImageProviderConfig holds server-side credentials and endpoint overrides for
// one direct image adapter. Secrets are never copied into gateway responses.
type ImageProviderConfig struct {
	BaseURL    string   `yaml:"base_url"`
	APIKey     string   `yaml:"api_key"`
	APIVersion string   `yaml:"api_version,omitempty"`
	Models     []string `yaml:"models,omitempty"`
}

// GenerationConfig for AI text generation.
type GenerationConfig struct {
	Temperature          float64  `yaml:"temperature"`
	MaxTokens            int      `yaml:"max_tokens"`
	TimeoutSeconds       int      `yaml:"timeout_seconds"`
	UtilityModel         string   `yaml:"utility_model"`
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
	AutosaveEvery          int    `yaml:"autosave_every"`
	TypewriterEffect       bool   `yaml:"typewriter_effect"`
	TypewriterSpeed        int    `yaml:"typewriter_speed"`
	VisiblePrivateThoughts bool   `yaml:"visible_private_thoughts"`
	RewardBudget           string `yaml:"reward_budget"`
}

// validProviders is the set of recognized provider names.
var validProviders = map[string]bool{
	"codex":       true,
	"claude-code": true,
	"litellm":     true,
	"openrouter":  true,
}

// Default returns a Config with structural defaults. Model identifiers are
// intentionally empty here so user config, setup, or the browser settings panel
// is the only source of provider-specific model choices.
func Default() Config {
	return Config{
		ConfigVersion: 2,
		DataDir:       "./oneday_data",
		AI: AIConfig{
			ProviderPriority: []string{"litellm", "openrouter", "codex", "claude-code"},
			Codex: CodexConfig{
				Enabled:   false,
				Binary:    "codex",
				Reasoning: "off",
			},
			ClaudeCode: ClaudeCodeConfig{
				Enabled: false,
				Binary:  "claude",
			},
			LiteLLM: LiteLLMConfig{
				Enabled: false,
				BaseURL: "http://127.0.0.1:4000/v1",
			},
			OpenRouter: OpenRouterConfig{
				Enabled: false,
				BaseURL: "https://openrouter.ai/api/v1",
			},
			Embedding: EmbeddingConfig{
				Provider: "auto",
				Local: LocalEmbeddingConfig{
					Enabled:    false,
					Type:       "ollama",
					BaseURL:    "http://127.0.0.1:11434",
					Dimensions: 1024,
				},
			},
			ASCIIArt: ASCIIArtConfig{
				Enabled:        false,
				Temperature:    0.4,
				MaxTokens:      400,
				TimeoutSeconds: 25,
			},
			ImageGeneration: ImageGenerationConfig{
				Provider:                      "codex-oauth",
				MapIconProvider:               "codex-oauth",
				MapIconModel:                  "gpt-image-2",
				OpenClawBridgeURL:             "http://127.0.0.1:8099/generate",
				ImagegenBridgeURL:             "http://127.0.0.1:8787",
				ImagegenBridgeProvider:        "codex-responses",
				ImagegenBridgeMapIconProvider: "codex-responses",
				ImagegenBridgeFallbacks:       []string{"codex-app-server:gpt-image-2"},
				ImagegenBridgeFallbackPolicy:  "on_error",
				ImagegenBridgeCompatibility:   "normalize",
				DefaultSize:                   "1024x1024",
				LocationSize:                  "1536x1024",
				CharacterSize:                 "1024x1024",
				OutputFormat:                  "png",
				TimeoutSeconds:                360,
				AutoGenerate:                  false,
				AppendNegativePrompt:          true,
			},
			TTS: TTSConfig{
				OutputDir:      "./oneday_data/audio",
				TimeoutSeconds: 45,
				ProviderOrder:  []string{"local", "cloud"},
				Cloud:          TTSEndpoint{BaseURL: "http://ai-proxy:4000/v1", Model: "gpt-4o-mini-tts", Voice: "alloy", Version: "1"},
				Local:          TTSEndpoint{BaseURL: "http://piper-tts:5000", Model: "piper", Voice: "", Version: "1"},
			},
			Generation: GenerationConfig{
				Temperature:    0.8,
				MaxTokens:      2048,
				TimeoutSeconds: 60,
			},
		},
		RAG: RAGConfig{
			Enabled:        false,
			SummarizeEvery: 10,
			TopK:           5,
			Dimensions:     1536,
		},
		Game: GameConfig{
			AutosaveEvery:          5,
			TypewriterEffect:       true,
			TypewriterSpeed:        80,
			VisiblePrivateThoughts: false,
			RewardBudget:           "balanced",
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

	expanded := os.ExpandEnv(string(data))
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}
	cfg.Migrate()

	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// Migrate normalizes older config files onto current safe defaults without
// overwriting explicit user choices.
func (c *Config) Migrate() {
	if c.ConfigVersion < 2 {
		if c.AI.Embedding.Local.Type == "" {
			c.AI.Embedding.Local.Type = "ollama"
		}
		if c.AI.Embedding.Local.BaseURL == "" {
			c.AI.Embedding.Local.BaseURL = "http://127.0.0.1:11434"
		}
		if c.AI.Embedding.Local.Dimensions == 0 {
			c.AI.Embedding.Local.Dimensions = 1024
		}
	}
	if strings.TrimSpace(c.Game.RewardBudget) == "" {
		c.Game.RewardBudget = "balanced"
	}
	if strings.TrimSpace(c.AI.Generation.UtilityModel) == "" {
		c.AI.Generation.UtilityModel = c.firstEnabledProviderModel()
	}
	// imagegen-bridge has always been OneDay's Codex OAuth transport. Normalize
	// old transport-oriented names to the user-facing provider ID without
	// changing any bridge-specific settings.
	switch strings.ToLower(strings.TrimSpace(c.AI.ImageGeneration.Provider)) {
	case "", "imagegen-bridge", "imagegen_bridge", "bridge-native":
		c.AI.ImageGeneration.Provider = "codex-oauth"
	}
	if strings.TrimSpace(c.AI.ImageGeneration.MapIconProvider) == "" {
		c.AI.ImageGeneration.MapIconProvider = c.AI.ImageGeneration.Provider
	} else {
		switch strings.ToLower(strings.TrimSpace(c.AI.ImageGeneration.MapIconProvider)) {
		case "imagegen-bridge", "imagegen_bridge", "bridge-native":
			c.AI.ImageGeneration.MapIconProvider = "codex-oauth"
		}
	}
	c.ConfigVersion = 2
}

// Marshal serializes config for local setup files.
func Marshal(cfg Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// Validate checks that the config is internally consistent.
func (c *Config) Validate() error {
	if len(c.AI.ProviderPriority) == 0 {
		return fmt.Errorf("ai.provider_priority must have at least one provider")
	}
	for _, p := range c.AI.ProviderPriority {
		if !validProviders[p] {
			return fmt.Errorf("unknown provider in priority chain: %q (valid: codex, claude-code, litellm, openrouter)", p)
		}
	}
	switch c.AI.Codex.Reasoning {
	case "", "off", "none", "minimal", "low", "medium", "high", "xhigh":
	default:
		return fmt.Errorf("ai.codex.reasoning must be one of: off, none, minimal, low, medium, high, xhigh")
	}
	if c.AI.Generation.MaxTokens <= 0 {
		return fmt.Errorf("ai.generation.max_tokens must be positive")
	}
	if c.AI.Generation.TimeoutSeconds <= 0 {
		return fmt.Errorf("ai.generation.timeout_seconds must be positive")
	}
	if c.AI.TTS.TimeoutSeconds <= 0 {
		return fmt.Errorf("ai.tts.timeout_seconds must be positive")
	}
	for _, provider := range c.AI.TTS.ProviderOrder {
		if provider != "cloud" && provider != "local" {
			return fmt.Errorf("ai.tts.provider_order contains unknown provider %q", provider)
		}
	}
	if c.AI.Codex.Enabled && strings.TrimSpace(c.AI.Codex.Model) == "" {
		return fmt.Errorf("ai.codex.model must not be empty when Codex is enabled")
	}
	if c.AI.LiteLLM.Enabled && strings.TrimSpace(c.AI.LiteLLM.DefaultModel) == "" {
		return fmt.Errorf("ai.litellm.default_model must not be empty when LiteLLM is enabled")
	}
	if c.AI.OpenRouter.Enabled && strings.TrimSpace(c.AI.OpenRouter.DefaultModel) == "" {
		return fmt.Errorf("ai.openrouter.default_model must not be empty when OpenRouter is enabled")
	}
	if len(c.EnabledProviders()) > 0 && strings.TrimSpace(c.AI.Generation.UtilityModel) == "" {
		return fmt.Errorf("ai.generation.utility_model must not be empty")
	}
	if c.AI.ASCIIArt.Enabled {
		if strings.TrimSpace(c.AI.ASCIIArt.Model) == "" {
			return fmt.Errorf("ai.ascii_art.model must not be empty when ascii art is enabled")
		}
		if c.AI.ASCIIArt.MaxTokens <= 0 {
			return fmt.Errorf("ai.ascii_art.max_tokens must be positive when ascii art is enabled")
		}
		if c.AI.ASCIIArt.TimeoutSeconds <= 0 {
			return fmt.Errorf("ai.ascii_art.timeout_seconds must be positive when ascii art is enabled")
		}
	}
	if c.AI.ImageGeneration.AutoGenerate {
		if strings.TrimSpace(c.AI.ImageGeneration.Provider) == "" {
			return fmt.Errorf("ai.image_generation.provider must not be empty when auto_generate is enabled")
		}
		if strings.TrimSpace(c.AI.ImageGeneration.Model) == "" {
			return fmt.Errorf("ai.image_generation.model must not be empty when auto_generate is enabled")
		}
		if strings.TrimSpace(c.AI.ImageGeneration.MapIconModel) == "" {
			return fmt.Errorf("ai.image_generation.map_icon_model must not be empty when auto_generate is enabled")
		}
		if !isSupportedImageProvider(c.AI.ImageGeneration.Provider) {
			return fmt.Errorf("ai.image_generation.provider %q is not supported", c.AI.ImageGeneration.Provider)
		}
		if !isSupportedImageProvider(c.AI.ImageGeneration.MapIconProvider) {
			return fmt.Errorf("ai.image_generation.map_icon_provider %q is not supported", c.AI.ImageGeneration.MapIconProvider)
		}
		if isImagegenBridgeProvider(c.AI.ImageGeneration.Provider) && strings.TrimSpace(c.AI.ImageGeneration.ImagegenBridgeURL) == "" {
			return fmt.Errorf("ai.image_generation.imagegen_bridge_url must not be empty when imagegen-bridge auto-generation is enabled")
		}
		if !isImagegenBridgeProvider(c.AI.ImageGeneration.Provider) && !isOpenClawImageProvider(c.AI.ImageGeneration.Provider) {
			configured, status := imageProviderConfigured(c.AI.ImageGeneration, c.AI.ImageGeneration.Provider)
			if !configured {
				return fmt.Errorf("ai.image_generation provider %q is not configured: %s", c.AI.ImageGeneration.Provider, status)
			}
		}
		if c.AI.ImageGeneration.TimeoutSeconds <= 0 {
			return fmt.Errorf("ai.image_generation.timeout_seconds must be positive when auto_generate is enabled")
		}
	}
	if policy := strings.TrimSpace(c.AI.ImageGeneration.ImagegenBridgeFallbackPolicy); policy != "" && policy != "on_unavailable" && policy != "on_error" {
		return fmt.Errorf("ai.image_generation.imagegen_bridge_fallback_policy must be on_unavailable or on_error")
	}
	if compatibility := strings.TrimSpace(c.AI.ImageGeneration.ImagegenBridgeCompatibility); compatibility != "" && compatibility != "strict" && compatibility != "normalize" && compatibility != "best_effort" {
		return fmt.Errorf("ai.image_generation.imagegen_bridge_compatibility must be strict, normalize, or best_effort")
	}
	for _, route := range c.AI.ImageGeneration.ImagegenBridgeFallbacks {
		provider, _, _ := strings.Cut(strings.TrimSpace(route), ":")
		if provider != "codex-responses" && provider != "codex-app-server" {
			return fmt.Errorf("ai.image_generation.imagegen_bridge_fallbacks may contain only codex-responses or codex-app-server routes")
		}
	}
	switch c.AI.Embedding.Provider {
	case "", "auto", "litellm", "openrouter", "local":
	default:
		return fmt.Errorf("ai.embedding.provider must be one of: auto, litellm, openrouter, local")
	}
	if c.AI.Embedding.Provider == "local" {
		if !c.AI.Embedding.Local.Enabled {
			return fmt.Errorf("ai.embedding.local.enabled must be true when ai.embedding.provider is local")
		}
		switch c.AI.Embedding.Local.Type {
		case "ollama", "custom":
		default:
			return fmt.Errorf("ai.embedding.local.type must be one of: ollama, custom")
		}
		if strings.TrimSpace(c.AI.Embedding.Local.BaseURL) == "" {
			return fmt.Errorf("ai.embedding.local.base_url must not be empty when ai.embedding.provider is local")
		}
		if strings.TrimSpace(c.AI.Embedding.Local.Model) == "" {
			return fmt.Errorf("ai.embedding.local.model must not be empty when ai.embedding.provider is local")
		}
		if c.AI.Embedding.Local.Dimensions <= 0 {
			return fmt.Errorf("ai.embedding.local.dimensions must be positive when ai.embedding.provider is local")
		}
	}
	if c.RAG.Enabled && c.AI.Embedding.Provider != "local" && strings.TrimSpace(c.AI.Embedding.Model) == "" {
		return fmt.Errorf("ai.embedding.model must not be empty when RAG uses remote embeddings")
	}
	switch strings.ToLower(strings.TrimSpace(c.Game.RewardBudget)) {
	case "generous", "balanced", "harsh":
		c.Game.RewardBudget = strings.ToLower(strings.TrimSpace(c.Game.RewardBudget))
	default:
		return fmt.Errorf("game.reward_budget must be one of: generous, balanced, harsh")
	}
	return nil
}

func isImagegenBridgeProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex-oauth", "imagegen-bridge", "imagegen_bridge", "bridge-native":
		return true
	default:
		return false
	}
}

func isSupportedImageProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex-oauth", "imagegen-bridge", "imagegen_bridge", "bridge-native",
		"openai", "openai-compatible", "litellm", "gemini", "fal", "replicate",
		"stability", "azure-openai", "openclaw", "openclaw-bridge":
		return true
	default:
		return false
	}
}

// EnabledProviders returns the provider priority chain filtered to only enabled providers.
func (c *Config) EnabledProviders() []string {
	enabled := make([]string, 0, len(c.AI.ProviderPriority))
	for _, name := range c.AI.ProviderPriority {
		switch name {
		case "codex":
			if c.AI.Codex.Enabled {
				enabled = append(enabled, name)
			}
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

func (c *Config) firstEnabledProviderModel() string {
	for _, name := range c.EnabledProviders() {
		switch name {
		case "codex":
			if model := strings.TrimSpace(c.AI.Codex.Model); model != "" {
				return model
			}
		case "litellm":
			if model := strings.TrimSpace(c.AI.LiteLLM.DefaultModel); model != "" {
				return model
			}
		case "openrouter":
			if model := strings.TrimSpace(c.AI.OpenRouter.DefaultModel); model != "" {
				return model
			}
		}
	}
	return ""
}

// RepairModelCandidates returns the ordered list of models to try for repair
// passes, with duplicates and blanks removed. UtilityModel is a fallback only
// when no explicit repair model is configured, preserving old config behavior.
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

	if strings.TrimSpace(g.RepairModel) == "" {
		appendModel(g.UtilityModel)
	} else {
		appendModel(g.RepairModel)
	}
	for _, model := range g.RepairFallbackModels {
		appendModel(model)
	}

	return out
}
