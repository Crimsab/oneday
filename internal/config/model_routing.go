package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	EmbeddingProvider    string   `json:"embedding_provider"`
	EmbeddingModel       string   `json:"embedding_model"`
	CodexReasoning       string   `json:"codex_reasoning"`
}

type ModelRoutingSettings struct {
	ConfigPath         string                 `json:"config_path"`
	ProviderPriority   []string               `json:"provider_priority"`
	Providers          []ModelProviderSetting `json:"providers"`
	NarrativeModels    []string               `json:"narrative_models"`
	UtilityModels      []string               `json:"utility_models"`
	RepairModels       []string               `json:"repair_models"`
	ImageModels        []string               `json:"image_models"`
	EmbeddingProviders []string               `json:"embedding_providers"`
	Active             ModelRoutingActive     `json:"active"`
	TTSStatus          string                 `json:"tts_status"`
}

type ModelProviderUpdate struct {
	ID        string  `json:"id"`
	Enabled   *bool   `json:"enabled,omitempty"`
	Model     *string `json:"model,omitempty"`
	Reasoning *string `json:"reasoning,omitempty"`
}

type ModelRoutingUpdate struct {
	ProviderPriority     *[]string             `json:"provider_priority,omitempty"`
	Providers            []ModelProviderUpdate `json:"providers,omitempty"`
	UtilityModel         *string               `json:"utility_model,omitempty"`
	RepairModel          *string               `json:"repair_model,omitempty"`
	RepairFallbackModels *[]string             `json:"repair_fallback_models,omitempty"`
	ImageModel           *string               `json:"image_model,omitempty"`
	EmbeddingProvider    *string               `json:"embedding_provider,omitempty"`
	EmbeddingModel       *string               `json:"embedding_model,omitempty"`
}

func LoadForEdit(path string) (Config, error) {
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
	cfg.Migrate()
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}

func SaveForEdit(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing temp config %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replacing config %s: %w", path, err)
	}
	return nil
}

func BuildModelRoutingSettings(path string, cfg Config) ModelRoutingSettings {
	activeProvider := ""
	if enabled := cfg.EnabledProviders(); len(enabled) > 0 {
		activeProvider = enabled[0]
	}
	activeNarrative := providerModel(cfg, activeProvider)
	repairModels := cfg.AI.Generation.RepairModelCandidates()

	return ModelRoutingSettings{
		ConfigPath:       path,
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
		ImageModels:        uniqueNonEmpty(cfg.AI.ASCIIArt.Model, activeNarrative),
		EmbeddingProviders: []string{"auto", "litellm", "openrouter", "local"},
		Active: ModelRoutingActive{
			Provider:             activeProvider,
			NarrativeModel:       activeNarrative,
			UtilityModel:         cfg.AI.Generation.UtilityModel,
			RepairModel:          firstNonEmpty(cfg.AI.Generation.RepairModel, firstString(repairModels)),
			RepairFallbackModels: append([]string(nil), cfg.AI.Generation.RepairFallbackModels...),
			ImageModel:           cfg.AI.ASCIIArt.Model,
			EmbeddingProvider:    firstNonEmpty(cfg.AI.Embedding.Provider, "auto"),
			EmbeddingModel:       cfg.AI.Embedding.Model,
			CodexReasoning:       firstNonEmpty(cfg.AI.Codex.Reasoning, "off"),
		},
		TTSStatus: "planned",
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
		cfg.AI.ASCIIArt.Model = cleanString(*update.ImageModel)
	}
	if update.EmbeddingProvider != nil {
		cfg.AI.Embedding.Provider = cleanString(*update.EmbeddingProvider)
	}
	if update.EmbeddingModel != nil {
		cfg.AI.Embedding.Model = cleanString(*update.EmbeddingModel)
	}
	if len(cfg.EnabledProviders()) == 0 {
		return fmt.Errorf("at least one provider must be enabled")
	}
	return cfg.Validate()
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
