package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ModelRoutingErrorValidation = "validation_failed"
	ModelRoutingErrorStale      = "stale_config"
	ModelRoutingErrorWrite      = "write_failed"
	ModelRoutingErrorLocked     = "config_locked"
)

var providerOrder = []string{"codex", "litellm", "openrouter", "claude-code"}

type ModelRoutingError struct {
	Code string
	Err  error
}

func (e ModelRoutingError) Error() string {
	if e.Err == nil {
		return e.Code
	}
	return e.Err.Error()
}

func (e ModelRoutingError) Unwrap() error {
	return e.Err
}

type ModelProviderSetting struct {
	ID                string `json:"id"`
	Label             string `json:"label"`
	Enabled           bool   `json:"enabled"`
	Model             string `json:"model,omitempty"`
	Reasoning         string `json:"reasoning,omitempty"`
	SupportsModel     bool   `json:"supports_model"`
	SupportsReasoning bool   `json:"supports_reasoning"`
}

type ModelRoutingActive struct {
	Provider             string   `json:"provider"`
	NarrativeModel       string   `json:"narrative_model"`
	UtilityModel         string   `json:"utility_model"`
	RepairModel          string   `json:"repair_model"`
	RepairFallbackModels []string `json:"repair_fallback_models"`
	ImageModel           string   `json:"image_model"`
	ASCIIModel           string   `json:"ascii_model"`
	EmbeddingProvider    string   `json:"embedding_provider"`
	EmbeddingModel       string   `json:"embedding_model"`
	CodexReasoning       string   `json:"codex_reasoning"`
}

type ModelRoutingSettings struct {
	ConfigPath         string                 `json:"config_path"`
	ConfigRevision     string                 `json:"config_revision"`
	ProviderPriority   []string               `json:"provider_priority"`
	Providers          []ModelProviderSetting `json:"providers"`
	NarrativeModels    []string               `json:"narrative_models"`
	UtilityModels      []string               `json:"utility_models"`
	RepairModels       []string               `json:"repair_models"`
	ImageModels        []string               `json:"image_models"`
	ASCIIModels        []string               `json:"ascii_models"`
	EmbeddingProviders []string               `json:"embedding_providers"`
	ImageGeneration    ImageGenerationSetting `json:"image_generation"`
	Active             ModelRoutingActive     `json:"active"`
	TTSStatus          string                 `json:"tts_status"`
}

type ImageGenerationSetting struct {
	Provider                      string   `json:"provider"`
	BaseURL                       string   `json:"base_url"`
	APIKeyConfigured              bool     `json:"api_key_configured"`
	Model                         string   `json:"model"`
	MapIconModel                  string   `json:"map_icon_model"`
	OpenClawBridgeURL             string   `json:"openclaw_bridge_url"`
	ImagegenBridgeURL             string   `json:"imagegen_bridge_url"`
	ImagegenBridgeTokenConfigured bool     `json:"imagegen_bridge_token_configured"`
	ImagegenBridgeProvider        string   `json:"imagegen_bridge_provider"`
	ImagegenBridgeMapIconProvider string   `json:"imagegen_bridge_map_icon_provider"`
	ImagegenBridgeFallbacks       []string `json:"imagegen_bridge_fallbacks"`
	ImagegenBridgeFallbackPolicy  string   `json:"imagegen_bridge_fallback_policy"`
	ImagegenBridgeCompatibility   string   `json:"imagegen_bridge_compatibility"`
	DefaultSize                   string   `json:"default_size"`
	LocationSize                  string   `json:"location_size"`
	CharacterSize                 string   `json:"character_size"`
	DefaultResolution             string   `json:"default_resolution"`
	LocationResolution            string   `json:"location_resolution"`
	CharacterResolution           string   `json:"character_resolution"`
	DefaultAspectRatio            string   `json:"default_aspect_ratio"`
	LocationAspectRatio           string   `json:"location_aspect_ratio"`
	CharacterAspectRatio          string   `json:"character_aspect_ratio"`
	Quality                       string   `json:"quality"`
	OutputFormat                  string   `json:"output_format"`
	Background                    string   `json:"background"`
	TimeoutSeconds                int      `json:"timeout_seconds"`
	AutoGenerate                  bool     `json:"auto_generate"`
	AppendNegativePrompt          bool     `json:"append_negative_prompt"`
	Available                     bool     `json:"available"`
	Status                        string   `json:"status"`
}

type ModelProviderUpdate struct {
	ID        string  `json:"id"`
	Enabled   *bool   `json:"enabled,omitempty"`
	Model     *string `json:"model,omitempty"`
	Reasoning *string `json:"reasoning,omitempty"`
}

type ModelRoutingUpdate struct {
	BaseRevision         string                 `json:"base_revision,omitempty"`
	ProviderPriority     *[]string              `json:"provider_priority,omitempty"`
	Providers            []ModelProviderUpdate  `json:"providers,omitempty"`
	UtilityModel         *string                `json:"utility_model,omitempty"`
	RepairModel          *string                `json:"repair_model,omitempty"`
	RepairFallbackModels *[]string              `json:"repair_fallback_models,omitempty"`
	ImageModel           *string                `json:"image_model,omitempty"`
	ImageGeneration      *ImageGenerationUpdate `json:"image_generation,omitempty"`
	ASCIIModel           *string                `json:"ascii_model,omitempty"`
	EmbeddingProvider    *string                `json:"embedding_provider,omitempty"`
	EmbeddingModel       *string                `json:"embedding_model,omitempty"`
}

type ImageGenerationUpdate struct {
	Provider                      *string   `json:"provider,omitempty"`
	BaseURL                       *string   `json:"base_url,omitempty"`
	Model                         *string   `json:"model,omitempty"`
	MapIconModel                  *string   `json:"map_icon_model,omitempty"`
	OpenClawBridgeURL             *string   `json:"openclaw_bridge_url,omitempty"`
	ImagegenBridgeURL             *string   `json:"imagegen_bridge_url,omitempty"`
	ImagegenBridgeProvider        *string   `json:"imagegen_bridge_provider,omitempty"`
	ImagegenBridgeMapIconProvider *string   `json:"imagegen_bridge_map_icon_provider,omitempty"`
	ImagegenBridgeFallbacks       *[]string `json:"imagegen_bridge_fallbacks,omitempty"`
	ImagegenBridgeFallbackPolicy  *string   `json:"imagegen_bridge_fallback_policy,omitempty"`
	ImagegenBridgeCompatibility   *string   `json:"imagegen_bridge_compatibility,omitempty"`
	DefaultSize                   *string   `json:"default_size,omitempty"`
	LocationSize                  *string   `json:"location_size,omitempty"`
	CharacterSize                 *string   `json:"character_size,omitempty"`
	DefaultResolution             *string   `json:"default_resolution,omitempty"`
	LocationResolution            *string   `json:"location_resolution,omitempty"`
	CharacterResolution           *string   `json:"character_resolution,omitempty"`
	DefaultAspectRatio            *string   `json:"default_aspect_ratio,omitempty"`
	LocationAspectRatio           *string   `json:"location_aspect_ratio,omitempty"`
	CharacterAspectRatio          *string   `json:"character_aspect_ratio,omitempty"`
	Quality                       *string   `json:"quality,omitempty"`
	OutputFormat                  *string   `json:"output_format,omitempty"`
	Background                    *string   `json:"background,omitempty"`
	TimeoutSeconds                *int      `json:"timeout_seconds,omitempty"`
	AutoGenerate                  *bool     `json:"auto_generate,omitempty"`
	AppendNegativePrompt          *bool     `json:"append_negative_prompt,omitempty"`
}

func ReadModelRoutingSettings(path string) (ModelRoutingSettings, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return ModelRoutingSettings{}, fmt.Errorf("reading config %s: %w", path, err)
		}
		raw = nil
	}
	cfg, err := configFromEditBytes(path, raw)
	if err != nil {
		return ModelRoutingSettings{}, err
	}
	return BuildModelRoutingSettings(path, cfg, ConfigRevision(raw)), nil
}

func UpdateModelRoutingSettings(path string, update ModelRoutingUpdate) (ModelRoutingSettings, error) {
	var settings ModelRoutingSettings
	err := withConfigLock(path, func() error {
		raw, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return ModelRoutingError{Code: ModelRoutingErrorWrite, Err: fmt.Errorf("reading config %s: %w", path, err)}
			}
			raw = nil
		}
		revision := ConfigRevision(raw)
		if strings.TrimSpace(update.BaseRevision) == "" {
			return ModelRoutingError{Code: ModelRoutingErrorValidation, Err: fmt.Errorf("base_revision is required; reload model settings before saving")}
		}
		if update.BaseRevision != revision {
			return ModelRoutingError{Code: ModelRoutingErrorStale, Err: fmt.Errorf("config changed on disk; reload before saving")}
		}

		cfg, err := configFromEditBytes(path, raw)
		if err != nil {
			return err
		}
		if err := ApplyModelRoutingUpdate(&cfg, update); err != nil {
			return ModelRoutingError{Code: ModelRoutingErrorValidation, Err: err}
		}

		nextRaw, err := patchModelRoutingYAML(raw, cfg)
		if err != nil {
			return ModelRoutingError{Code: ModelRoutingErrorWrite, Err: err}
		}
		if _, err := configFromEditBytes(path, nextRaw); err != nil {
			return err
		}
		if err := writeConfigAtomic(path, raw, nextRaw); err != nil {
			return ModelRoutingError{Code: ModelRoutingErrorWrite, Err: err}
		}
		nextRevision := ConfigRevision(nextRaw)
		settings = BuildModelRoutingSettings(path, cfg, nextRevision)
		return nil
	})
	return settings, err
}

func ConfigRevision(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func LoadForEdit(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("reading config %s: %w", path, err)
	}
	return configFromEditBytes(path, raw)
}

func BuildModelRoutingSettings(path string, cfg Config, revision string) ModelRoutingSettings {
	activeProvider := ""
	if enabled := cfg.EnabledProviders(); len(enabled) > 0 {
		activeProvider = enabled[0]
	}
	activeNarrative := providerModel(cfg, activeProvider)
	repairModels := cfg.AI.Generation.RepairModelCandidates()

	return ModelRoutingSettings{
		ConfigPath:       path,
		ConfigRevision:   revision,
		ProviderPriority: append([]string(nil), cfg.AI.ProviderPriority...),
		Providers: []ModelProviderSetting{
			{
				ID:                "codex",
				Label:             "Codex OAuth",
				Enabled:           cfg.AI.Codex.Enabled,
				Model:             cfg.AI.Codex.Model,
				Reasoning:         cfg.AI.Codex.Reasoning,
				SupportsModel:     true,
				SupportsReasoning: true,
			},
			{
				ID:            "litellm",
				Label:         "LiteLLM",
				Enabled:       cfg.AI.LiteLLM.Enabled,
				Model:         cfg.AI.LiteLLM.DefaultModel,
				SupportsModel: true,
			},
			{
				ID:            "openrouter",
				Label:         "OpenRouter",
				Enabled:       cfg.AI.OpenRouter.Enabled,
				Model:         cfg.AI.OpenRouter.DefaultModel,
				SupportsModel: true,
			},
			{
				ID:            "claude-code",
				Label:         "Claude Code",
				Enabled:       cfg.AI.ClaudeCode.Enabled,
				SupportsModel: false,
			},
		},
		NarrativeModels:    uniqueNonEmpty(providerModel(cfg, "codex"), providerModel(cfg, "litellm"), providerModel(cfg, "openrouter"), activeNarrative),
		UtilityModels:      uniqueNonEmpty(cfg.AI.Generation.UtilityModel, activeNarrative, providerModel(cfg, "litellm"), providerModel(cfg, "openrouter")),
		RepairModels:       uniqueNonEmpty(append([]string{cfg.AI.Generation.RepairModel, cfg.AI.Generation.UtilityModel}, cfg.AI.Generation.RepairFallbackModels...)...),
		ImageModels:        uniqueNonEmpty(cfg.AI.ImageGeneration.Model, cfg.AI.ImageGeneration.MapIconModel),
		ASCIIModels:        uniqueNonEmpty(cfg.AI.ASCIIArt.Model, activeNarrative),
		EmbeddingProviders: []string{"auto", "litellm", "openrouter", "local"},
		ImageGeneration:    buildImageGenerationSetting(cfg.AI.ImageGeneration),
		Active: ModelRoutingActive{
			Provider:             activeProvider,
			NarrativeModel:       activeNarrative,
			UtilityModel:         cfg.AI.Generation.UtilityModel,
			RepairModel:          firstNonEmpty(cfg.AI.Generation.RepairModel, firstString(repairModels)),
			RepairFallbackModels: append([]string(nil), cfg.AI.Generation.RepairFallbackModels...),
			ImageModel:           cfg.AI.ImageGeneration.Model,
			ASCIIModel:           cfg.AI.ASCIIArt.Model,
			EmbeddingProvider:    firstNonEmpty(cfg.AI.Embedding.Provider, "auto"),
			EmbeddingModel:       cfg.AI.Embedding.Model,
			CodexReasoning:       firstNonEmpty(cfg.AI.Codex.Reasoning, "off"),
		},
		TTSStatus: "planned",
	}
}

func buildImageGenerationSetting(cfg ImageGenerationConfig) ImageGenerationSetting {
	available, status := imageGenerationAvailability(cfg)
	// The Rust gateway contract expects an array even when no fallbacks are configured.
	fallbacks := append([]string{}, cfg.ImagegenBridgeFallbacks...)
	return ImageGenerationSetting{
		Provider:                      cfg.Provider,
		BaseURL:                       cfg.BaseURL,
		APIKeyConfigured:              strings.TrimSpace(cfg.APIKey) != "",
		Model:                         cfg.Model,
		MapIconModel:                  cfg.MapIconModel,
		OpenClawBridgeURL:             cfg.OpenClawBridgeURL,
		ImagegenBridgeURL:             cfg.ImagegenBridgeURL,
		ImagegenBridgeTokenConfigured: strings.TrimSpace(cfg.ImagegenBridgeToken) != "",
		ImagegenBridgeProvider:        cfg.ImagegenBridgeProvider,
		ImagegenBridgeMapIconProvider: cfg.ImagegenBridgeMapIconProvider,
		ImagegenBridgeFallbacks:       fallbacks,
		ImagegenBridgeFallbackPolicy:  cfg.ImagegenBridgeFallbackPolicy,
		ImagegenBridgeCompatibility:   cfg.ImagegenBridgeCompatibility,
		DefaultSize:                   cfg.DefaultSize,
		LocationSize:                  cfg.LocationSize,
		CharacterSize:                 cfg.CharacterSize,
		DefaultResolution:             cfg.DefaultResolution,
		LocationResolution:            cfg.LocationResolution,
		CharacterResolution:           cfg.CharacterResolution,
		DefaultAspectRatio:            cfg.DefaultAspectRatio,
		LocationAspectRatio:           cfg.LocationAspectRatio,
		CharacterAspectRatio:          cfg.CharacterAspectRatio,
		Quality:                       cfg.Quality,
		OutputFormat:                  cfg.OutputFormat,
		Background:                    cfg.Background,
		TimeoutSeconds:                cfg.TimeoutSeconds,
		AutoGenerate:                  cfg.AutoGenerate,
		AppendNegativePrompt:          cfg.AppendNegativePrompt,
		Available:                     available,
		Status:                        status,
	}
}

func imageGenerationAvailability(cfg ImageGenerationConfig) (bool, string) {
	if strings.TrimSpace(cfg.Provider) == "" {
		return false, "missing provider"
	}
	if !isImagegenBridgeProvider(cfg.Provider) && strings.TrimSpace(cfg.Model) == "" {
		return false, "missing model"
	}
	if isImagegenBridgeProvider(cfg.Provider) {
		if strings.TrimSpace(cfg.ImagegenBridgeURL) == "" {
			return false, "missing imagegen-bridge URL"
		}
		return true, "configured through imagegen-bridge native API"
	}
	if isOpenClawImageProvider(cfg.Provider) {
		if strings.TrimSpace(cfg.OpenClawBridgeURL) == "" {
			return false, "missing OpenClaw bridge URL"
		}
		return true, "configured through OpenClaw bridge"
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return false, "missing base URL"
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return false, "missing API key"
	}
	return true, "configured through OpenAI-compatible image endpoint"
}

func isOpenClawImageProvider(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openclaw", "openclaw-bridge", "codex-oauth":
		return true
	default:
		return false
	}
}

func ApplyModelRoutingUpdate(cfg *Config, update ModelRoutingUpdate) error {
	if update.ProviderPriority != nil {
		priority, err := cleanProviderPriority(*update.ProviderPriority)
		if err != nil {
			return err
		}
		cfg.AI.ProviderPriority = priority
	}
	for _, provider := range update.Providers {
		id := strings.TrimSpace(provider.ID)
		switch id {
		case "codex":
			if provider.Enabled != nil {
				cfg.AI.Codex.Enabled = *provider.Enabled
			}
			if provider.Model != nil {
				cfg.AI.Codex.Model = cleanString(*provider.Model)
			}
			if provider.Reasoning != nil {
				cfg.AI.Codex.Reasoning = cleanString(*provider.Reasoning)
			}
		case "litellm":
			if provider.Enabled != nil {
				cfg.AI.LiteLLM.Enabled = *provider.Enabled
			}
			if provider.Model != nil {
				cfg.AI.LiteLLM.DefaultModel = cleanString(*provider.Model)
			}
		case "openrouter":
			if provider.Enabled != nil {
				cfg.AI.OpenRouter.Enabled = *provider.Enabled
			}
			if provider.Model != nil {
				cfg.AI.OpenRouter.DefaultModel = cleanString(*provider.Model)
			}
		case "claude-code":
			if provider.Enabled != nil {
				cfg.AI.ClaudeCode.Enabled = *provider.Enabled
			}
		default:
			return fmt.Errorf("unknown provider %q", provider.ID)
		}
	}
	if update.UtilityModel != nil {
		cfg.AI.Generation.UtilityModel = cleanString(*update.UtilityModel)
	}
	if update.RepairModel != nil {
		cfg.AI.Generation.RepairModel = cleanString(*update.RepairModel)
	}
	if update.RepairFallbackModels != nil {
		cfg.AI.Generation.RepairFallbackModels = cleanStringSlice(*update.RepairFallbackModels)
	}
	if update.ImageModel != nil {
		cfg.AI.ImageGeneration.Model = cleanString(*update.ImageModel)
	}
	if update.ImageGeneration != nil {
		applyImageGenerationUpdate(&cfg.AI.ImageGeneration, *update.ImageGeneration)
	}
	if update.ASCIIModel != nil {
		cfg.AI.ASCIIArt.Model = cleanString(*update.ASCIIModel)
	}
	if update.EmbeddingProvider != nil {
		cfg.AI.Embedding.Provider = cleanString(*update.EmbeddingProvider)
	}
	if update.EmbeddingModel != nil {
		cfg.AI.Embedding.Model = cleanString(*update.EmbeddingModel)
	}
	if cfg.AI.Embedding.Provider == "local" && !cfg.AI.Embedding.Local.Enabled {
		return fmt.Errorf("ai.embedding.local.enabled must be true when ai.embedding.provider is local")
	}
	if len(cfg.EnabledProviders()) == 0 {
		return fmt.Errorf("at least one provider must be enabled")
	}
	return cfg.Validate()
}

func applyImageGenerationUpdate(cfg *ImageGenerationConfig, update ImageGenerationUpdate) {
	if update.Provider != nil {
		cfg.Provider = cleanString(*update.Provider)
	}
	if update.BaseURL != nil {
		cfg.BaseURL = cleanString(*update.BaseURL)
	}
	if update.Model != nil {
		cfg.Model = cleanString(*update.Model)
	}
	if update.MapIconModel != nil {
		cfg.MapIconModel = cleanString(*update.MapIconModel)
	}
	if update.OpenClawBridgeURL != nil {
		cfg.OpenClawBridgeURL = cleanString(*update.OpenClawBridgeURL)
	}
	if update.ImagegenBridgeURL != nil {
		cfg.ImagegenBridgeURL = cleanString(*update.ImagegenBridgeURL)
	}
	if update.ImagegenBridgeProvider != nil {
		cfg.ImagegenBridgeProvider = cleanString(*update.ImagegenBridgeProvider)
	}
	if update.ImagegenBridgeMapIconProvider != nil {
		cfg.ImagegenBridgeMapIconProvider = cleanString(*update.ImagegenBridgeMapIconProvider)
	}
	if update.ImagegenBridgeFallbacks != nil {
		cfg.ImagegenBridgeFallbacks = cleanStringSlice(*update.ImagegenBridgeFallbacks)
	}
	if update.ImagegenBridgeFallbackPolicy != nil {
		cfg.ImagegenBridgeFallbackPolicy = cleanString(*update.ImagegenBridgeFallbackPolicy)
	}
	if update.ImagegenBridgeCompatibility != nil {
		cfg.ImagegenBridgeCompatibility = cleanString(*update.ImagegenBridgeCompatibility)
	}
	if update.DefaultSize != nil {
		cfg.DefaultSize = cleanString(*update.DefaultSize)
	}
	if update.LocationSize != nil {
		cfg.LocationSize = cleanString(*update.LocationSize)
	}
	if update.CharacterSize != nil {
		cfg.CharacterSize = cleanString(*update.CharacterSize)
	}
	if update.DefaultResolution != nil {
		cfg.DefaultResolution = cleanString(*update.DefaultResolution)
	}
	if update.LocationResolution != nil {
		cfg.LocationResolution = cleanString(*update.LocationResolution)
	}
	if update.CharacterResolution != nil {
		cfg.CharacterResolution = cleanString(*update.CharacterResolution)
	}
	if update.DefaultAspectRatio != nil {
		cfg.DefaultAspectRatio = cleanString(*update.DefaultAspectRatio)
	}
	if update.LocationAspectRatio != nil {
		cfg.LocationAspectRatio = cleanString(*update.LocationAspectRatio)
	}
	if update.CharacterAspectRatio != nil {
		cfg.CharacterAspectRatio = cleanString(*update.CharacterAspectRatio)
	}
	if update.Quality != nil {
		cfg.Quality = cleanString(*update.Quality)
	}
	if update.OutputFormat != nil {
		cfg.OutputFormat = cleanString(*update.OutputFormat)
	}
	if update.Background != nil {
		cfg.Background = cleanString(*update.Background)
	}
	if update.TimeoutSeconds != nil {
		cfg.TimeoutSeconds = *update.TimeoutSeconds
	}
	if update.AutoGenerate != nil {
		cfg.AutoGenerate = *update.AutoGenerate
	}
	if update.AppendNegativePrompt != nil {
		cfg.AppendNegativePrompt = *update.AppendNegativePrompt
	}
}

func configFromEditBytes(path string, raw []byte) (Config, error) {
	cfg := Default()
	if len(raw) > 0 {
		var doc yaml.Node
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return cfg, fmt.Errorf("parsing config %s: %w", path, err)
		}
		if err := rejectDuplicateMappingKeys(&doc, ""); err != nil {
			return cfg, fmt.Errorf("parsing config %s: %w", path, err)
		}
		if err := doc.Decode(&cfg); err != nil {
			return cfg, fmt.Errorf("parsing config %s: %w", path, err)
		}
	}
	cfg.Migrate()
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}

func patchModelRoutingYAML(raw []byte, cfg Config) ([]byte, error) {
	var doc yaml.Node
	if len(raw) == 0 {
		raw = []byte("config_version: 2\n")
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML document: %w", err)
	}
	if err := rejectDuplicateMappingKeys(&doc, ""); err != nil {
		return nil, err
	}
	root, err := documentRoot(&doc)
	if err != nil {
		return nil, err
	}
	for _, apply := range []func() error{
		func() error { return setStringSlice(root, cfg.AI.ProviderPriority, "ai", "provider_priority") },
		func() error { return setBool(root, cfg.AI.Codex.Enabled, "ai", "codex", "enabled") },
		func() error { return setString(root, cfg.AI.Codex.Model, "ai", "codex", "model") },
		func() error { return setString(root, cfg.AI.Codex.Reasoning, "ai", "codex", "reasoning") },
		func() error { return setBool(root, cfg.AI.LiteLLM.Enabled, "ai", "litellm", "enabled") },
		func() error { return setString(root, cfg.AI.LiteLLM.DefaultModel, "ai", "litellm", "default_model") },
		func() error { return setBool(root, cfg.AI.OpenRouter.Enabled, "ai", "openrouter", "enabled") },
		func() error {
			return setString(root, cfg.AI.OpenRouter.DefaultModel, "ai", "openrouter", "default_model")
		},
		func() error { return setBool(root, cfg.AI.ClaudeCode.Enabled, "ai", "claude_code", "enabled") },
		func() error {
			return setString(root, cfg.AI.Generation.UtilityModel, "ai", "generation", "utility_model")
		},
		func() error {
			return setString(root, cfg.AI.Generation.RepairModel, "ai", "generation", "repair_model")
		},
		func() error {
			return setStringSlice(root, cfg.AI.Generation.RepairFallbackModels, "ai", "generation", "repair_fallback_models")
		},
		func() error { return setString(root, cfg.AI.ASCIIArt.Model, "ai", "ascii_art", "model") },
		func() error {
			return setString(root, cfg.AI.ImageGeneration.Provider, "ai", "image_generation", "provider")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.BaseURL, "ai", "image_generation", "base_url")
		},
		func() error { return setString(root, cfg.AI.ImageGeneration.Model, "ai", "image_generation", "model") },
		func() error {
			return setString(root, cfg.AI.ImageGeneration.MapIconModel, "ai", "image_generation", "map_icon_model")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.OpenClawBridgeURL, "ai", "image_generation", "openclaw_bridge_url")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.ImagegenBridgeURL, "ai", "image_generation", "imagegen_bridge_url")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.ImagegenBridgeProvider, "ai", "image_generation", "imagegen_bridge_provider")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.ImagegenBridgeMapIconProvider, "ai", "image_generation", "imagegen_bridge_map_icon_provider")
		},
		func() error {
			return setStringSlice(root, cfg.AI.ImageGeneration.ImagegenBridgeFallbacks, "ai", "image_generation", "imagegen_bridge_fallbacks")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.ImagegenBridgeFallbackPolicy, "ai", "image_generation", "imagegen_bridge_fallback_policy")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.ImagegenBridgeCompatibility, "ai", "image_generation", "imagegen_bridge_compatibility")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.DefaultSize, "ai", "image_generation", "default_size")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.LocationSize, "ai", "image_generation", "location_size")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.CharacterSize, "ai", "image_generation", "character_size")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.DefaultResolution, "ai", "image_generation", "default_resolution")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.LocationResolution, "ai", "image_generation", "location_resolution")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.CharacterResolution, "ai", "image_generation", "character_resolution")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.DefaultAspectRatio, "ai", "image_generation", "default_aspect_ratio")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.LocationAspectRatio, "ai", "image_generation", "location_aspect_ratio")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.CharacterAspectRatio, "ai", "image_generation", "character_aspect_ratio")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.Quality, "ai", "image_generation", "quality")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.OutputFormat, "ai", "image_generation", "output_format")
		},
		func() error {
			return setString(root, cfg.AI.ImageGeneration.Background, "ai", "image_generation", "background")
		},
		func() error {
			return setInt(root, cfg.AI.ImageGeneration.TimeoutSeconds, "ai", "image_generation", "timeout_seconds")
		},
		func() error {
			return setBool(root, cfg.AI.ImageGeneration.AutoGenerate, "ai", "image_generation", "auto_generate")
		},
		func() error {
			return setBool(root, cfg.AI.ImageGeneration.AppendNegativePrompt, "ai", "image_generation", "append_negative_prompt")
		},
		func() error { return setString(root, cfg.AI.Embedding.Provider, "ai", "embedding", "provider") },
		func() error { return setString(root, cfg.AI.Embedding.Model, "ai", "embedding", "model") },
	} {
		if err := apply(); err != nil {
			return nil, err
		}
	}
	var out strings.Builder
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		_ = encoder.Close()
		return nil, fmt.Errorf("encoding YAML document: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing YAML encoder: %w", err)
	}
	return []byte(out.String()), nil
}

func documentRoot(doc *yaml.Node) (*yaml.Node, error) {
	if doc.Kind != yaml.DocumentNode {
		doc.Kind = yaml.DocumentNode
	}
	if len(doc.Content) == 0 || doc.Content[0] == nil {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("YAML document root must be a mapping")
	}
	return doc.Content[0], nil
}

func setString(root *yaml.Node, value string, path ...string) error {
	node, err := ensurePath(root, path)
	if err != nil {
		return err
	}
	node.Kind = yaml.ScalarNode
	node.Tag = "!!str"
	node.Value = value
	return nil
}

func setBool(root *yaml.Node, value bool, path ...string) error {
	node, err := ensurePath(root, path)
	if err != nil {
		return err
	}
	node.Kind = yaml.ScalarNode
	node.Tag = "!!bool"
	if value {
		node.Value = "true"
	} else {
		node.Value = "false"
	}
	return nil
}

func setInt(root *yaml.Node, value int, path ...string) error {
	node, err := ensurePath(root, path)
	if err != nil {
		return err
	}
	node.Kind = yaml.ScalarNode
	node.Tag = "!!int"
	node.Value = strconv.Itoa(value)
	return nil
}

func setStringSlice(root *yaml.Node, values []string, path ...string) error {
	node, err := ensurePath(root, path)
	if err != nil {
		return err
	}
	node.Kind = yaml.SequenceNode
	node.Tag = "!!seq"
	node.Content = node.Content[:0]
	for _, value := range values {
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
	}
	return nil
}

func ensurePath(root *yaml.Node, path []string) (*yaml.Node, error) {
	current := root
	for index, key := range path {
		if current.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("YAML path %s must be a mapping", yamlPath(path[:index]))
		}
		value := mappingValue(current, key)
		if value == nil {
			value = &yaml.Node{Kind: yaml.MappingNode}
			current.Content = append(current.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
		}
		if index == len(path)-1 {
			return value, nil
		}
		if value.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("YAML path %s must be a mapping, got kind %d", yamlPath(path[:index+1]), value.Kind)
		}
		current = value
	}
	return current, nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func rejectDuplicateMappingKeys(node *yaml.Node, path string) error {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if err := rejectDuplicateMappingKeys(child, path); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		seen := map[string]bool{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			keyPath := joinYAMLPath(path, key)
			if seen[key] {
				return fmt.Errorf("duplicate YAML key at %s", keyPath)
			}
			seen[key] = true
			if err := rejectDuplicateMappingKeys(node.Content[i+1], keyPath); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for index, child := range node.Content {
			if err := rejectDuplicateMappingKeys(child, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func yamlPath(parts []string) string {
	if len(parts) == 0 {
		return "<root>"
	}
	return strings.Join(parts, ".")
}

func joinYAMLPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func writeConfigAtomic(path string, oldRaw, nextRaw []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	mode := os.FileMode(0600)
	uid, gid := -1, -1
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
		if ownerUID, ownerGID, ok := fileOwnerIDs(info); ok {
			uid = ownerUID
			gid = ownerGID
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config %s: %w", path, err)
	}
	if len(oldRaw) > 0 {
		if err := writeBackup(path+".bak", oldRaw, mode, uid, gid); err != nil {
			return fmt.Errorf("writing config backup: %w", err)
		}
	}
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp config: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if uid >= 0 && os.Geteuid() == 0 {
		if err := tmp.Chown(uid, gid); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("chown temp config: %w", err)
		}
	}
	if _, err := tmp.Write(nextRaw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("syncing temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replacing config: %w", err)
	}
	cleanup = false
	return fsyncDir(dir)
}

func writeBackup(path string, data []byte, mode os.FileMode, uid, gid int) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if uid >= 0 && os.Geteuid() == 0 {
		if err := os.Chown(tmp, uid, gid); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return os.Rename(tmp, path)
}

func withConfigLock(path string, fn func() error) error {
	lockPath := path + ".lock"
	deadline := time.Now().Add(5 * time.Second)
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = fmt.Fprintf(file, "%d %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
			_ = file.Close()
			defer os.Remove(lockPath)
			return fn()
		}
		if !os.IsExist(err) {
			return ModelRoutingError{Code: ModelRoutingErrorLocked, Err: fmt.Errorf("creating config lock: %w", err)}
		}
		_ = recoverStaleConfigLock(lockPath)
		if time.Now().After(deadline) {
			return ModelRoutingError{Code: ModelRoutingErrorLocked, Err: fmt.Errorf("config lock is held: %s", lockPath)}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func recoverStaleConfigLock(lockPath string) error {
	info, err := os.Stat(lockPath)
	if err != nil {
		return err
	}
	if time.Since(info.ModTime()) < 2*time.Minute {
		return nil
	}
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return os.Remove(lockPath)
	}
	pid, err := strconv.Atoi(fields[0])
	if err != nil || pid <= 0 {
		return os.Remove(lockPath)
	}
	if processExists(pid) {
		return nil
	}
	return os.Remove(lockPath)
}

func cleanProviderPriority(values []string) ([]string, error) {
	out := cleanStringSlice(values)
	if len(out) == 0 {
		return nil, fmt.Errorf("ai.provider_priority must have at least one provider")
	}
	for _, provider := range out {
		if !validProviders[provider] {
			return nil, fmt.Errorf("unknown provider in priority chain: %q", provider)
		}
	}
	for _, provider := range providerOrder {
		if !containsString(out, provider) {
			out = append(out, provider)
		}
	}
	return out, nil
}

func providerModel(cfg Config, provider string) string {
	switch provider {
	case "codex":
		return cfg.AI.Codex.Model
	case "litellm":
		return cfg.AI.LiteLLM.DefaultModel
	case "openrouter":
		return cfg.AI.OpenRouter.DefaultModel
	default:
		return ""
	}
}

func cleanString(value string) string {
	return strings.TrimSpace(value)
}

func cleanStringSlice(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		clean := cleanString(value)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	return out
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func uniqueNonEmpty(values ...string) []string {
	return cleanStringSlice(values)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if clean := cleanString(value); clean != "" {
			return clean
		}
	}
	return ""
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
