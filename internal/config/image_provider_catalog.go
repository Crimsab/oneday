package config

import "strings"

const (
	ImageProviderCodexOAuth       = "codex-oauth"
	ImageProviderOpenAI           = "openai"
	ImageProviderOpenAICompatible = "openai-compatible"
	ImageProviderGemini           = "gemini"
	ImageProviderFal              = "fal"
	ImageProviderReplicate        = "replicate"
	ImageProviderStability        = "stability"
	ImageProviderAzureOpenAI      = "azure-openai"
)

// ImageProviderCapabilities is intentionally data-only so Settings can render
// provider controls without duplicating backend routing knowledge.
type ImageProviderCapabilities struct {
	Generate             bool     `json:"generate"`
	Edit                 bool     `json:"edit"`
	Sizes                []string `json:"sizes"`
	AspectRatios         []string `json:"aspect_ratios"`
	Qualities            []string `json:"qualities"`
	OutputFormats        []string `json:"output_formats"`
	SupportsTransparency bool     `json:"supports_transparency"`
}

type ImageProviderCatalogEntry struct {
	ID               string                    `json:"id"`
	DisplayName      string                    `json:"display_name"`
	AuthType         string                    `json:"auth_type"`
	Default          bool                      `json:"default"`
	Configured       bool                      `json:"configured"`
	APIKeyConfigured bool                      `json:"api_key_configured"`
	Status           string                    `json:"status"`
	BaseURL          string                    `json:"base_url"`
	APIVersion       string                    `json:"api_version,omitempty"`
	Models           []string                  `json:"models"`
	ModelValidation  string                    `json:"model_validation"`
	Capabilities     ImageProviderCapabilities `json:"capabilities"`
}

type imageProviderDefinition struct {
	id              string
	displayName     string
	authType        string
	models          []string
	modelValidation string
	capabilities    ImageProviderCapabilities
}

var imageProviderDefinitions = []imageProviderDefinition{
	{
		id: ImageProviderCodexOAuth, displayName: "Codex OAuth", authType: "codex_oauth",
		models: []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, modelValidation: "allowlist",
		capabilities: ImageProviderCapabilities{Generate: true, Sizes: []string{"1024x1024", "1024x1536", "1536x1024"}, Qualities: []string{"auto", "low", "medium", "high"}, OutputFormats: []string{"png", "jpeg", "webp"}, SupportsTransparency: true},
	},
	{
		id: ImageProviderOpenAI, displayName: "OpenAI Platform", authType: "api_key",
		models: []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, modelValidation: "allowlist",
		capabilities: ImageProviderCapabilities{Generate: true, Sizes: []string{"1024x1024", "1024x1536", "1536x1024"}, Qualities: []string{"auto", "low", "medium", "high"}, OutputFormats: []string{"png", "jpeg", "webp"}, SupportsTransparency: true},
	},
	{
		id: ImageProviderOpenAICompatible, displayName: "OpenAI-compatible / LiteLLM", authType: "api_key",
		modelValidation: "configured", capabilities: ImageProviderCapabilities{Generate: true, Sizes: []string{"provider-defined"}, AspectRatios: []string{"provider-defined"}, Qualities: []string{"provider-defined"}, OutputFormats: []string{"png", "jpeg", "webp"}, SupportsTransparency: true},
	},
	{
		id: ImageProviderGemini, displayName: "Google Gemini", authType: "api_key",
		models: []string{"gemini-3.1-flash-image", "gemini-3-pro-image"}, modelValidation: "allowlist_or_gemini_image_model",
		capabilities: ImageProviderCapabilities{Generate: true, AspectRatios: []string{"1:1", "3:2", "2:3", "4:3", "3:4", "16:9", "9:16"}, OutputFormats: []string{"png", "jpeg"}},
	},
	{
		id: ImageProviderFal, displayName: "fal.ai", authType: "api_key",
		models: []string{"fal-ai/flux/schnell", "fal-ai/nano-banana-2"}, modelValidation: "vendor_slug",
		capabilities: ImageProviderCapabilities{Generate: true, AspectRatios: []string{"model-defined"}, OutputFormats: []string{"png", "jpeg", "webp"}},
	},
	{
		id: ImageProviderReplicate, displayName: "Replicate", authType: "api_token",
		models: []string{"black-forest-labs/flux-schnell"}, modelValidation: "owner_model_slug",
		capabilities: ImageProviderCapabilities{Generate: true, AspectRatios: []string{"model-defined"}, OutputFormats: []string{"png", "jpeg", "webp"}},
	},
	{
		id: ImageProviderStability, displayName: "Stability AI", authType: "api_key",
		models: []string{"stable-image-core"}, modelValidation: "allowlist",
		capabilities: ImageProviderCapabilities{Generate: true, AspectRatios: []string{"1:1", "16:9", "9:16", "3:2", "2:3", "4:5", "5:4", "21:9", "9:21"}, OutputFormats: []string{"png", "jpeg", "webp"}},
	},
	{
		id: ImageProviderAzureOpenAI, displayName: "Azure OpenAI Images", authType: "api_key",
		models: []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, modelValidation: "deployment_name",
		capabilities: ImageProviderCapabilities{Generate: true, Sizes: []string{"1024x1024", "1024x1536", "1536x1024"}, Qualities: []string{"auto", "low", "medium", "high"}, OutputFormats: []string{"png", "jpeg"}, SupportsTransparency: true},
	},
}

func canonicalImageProviderID(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "imagegen-bridge", "imagegen_bridge", "bridge-native":
		return ImageProviderCodexOAuth
	case "litellm":
		return ImageProviderOpenAICompatible
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func imageProviderConfig(cfg ImageGenerationConfig, provider string) ImageProviderConfig {
	provider = canonicalImageProviderID(provider)
	if direct, ok := cfg.Providers[provider]; ok {
		direct.Models = cleanStringSlice(direct.Models)
		if strings.TrimSpace(direct.BaseURL) == "" {
			direct.BaseURL = defaultImageProviderBaseURL(provider)
		}
		return direct
	}
	// Backward compatibility: the old flat endpoint/key apply only to the
	// currently selected direct provider. They are never exposed by the API.
	if canonicalImageProviderID(cfg.Provider) == provider {
		baseURL := cfg.BaseURL
		if strings.TrimSpace(baseURL) == "" {
			baseURL = defaultImageProviderBaseURL(provider)
		}
		return ImageProviderConfig{BaseURL: baseURL, APIKey: cfg.APIKey}
	}
	return ImageProviderConfig{BaseURL: defaultImageProviderBaseURL(provider)}
}

func defaultImageProviderBaseURL(provider string) string {
	switch provider {
	case ImageProviderOpenAI:
		return "https://api.openai.com/v1"
	case ImageProviderGemini:
		return "https://generativelanguage.googleapis.com/v1beta"
	case ImageProviderFal:
		return "https://queue.fal.run"
	case ImageProviderReplicate:
		return "https://api.replicate.com/v1"
	case ImageProviderStability:
		return "https://api.stability.ai/v2beta"
	default:
		return ""
	}
}

func imageProviderConfigured(cfg ImageGenerationConfig, provider string) (bool, string) {
	provider = canonicalImageProviderID(provider)
	if provider == ImageProviderCodexOAuth {
		if strings.TrimSpace(cfg.ImagegenBridgeURL) == "" {
			return false, "missing imagegen-bridge URL"
		}
		return true, "Codex OAuth via imagegen-bridge"
	}
	direct := imageProviderConfig(cfg, provider)
	if provider == ImageProviderOpenAI {
		if strings.TrimSpace(direct.APIKey) == "" {
			return false, "missing OpenAI Platform API key"
		}
		return true, "configured"
	}
	if strings.TrimSpace(direct.BaseURL) == "" {
		return false, "missing base URL"
	}
	if strings.TrimSpace(direct.APIKey) == "" {
		return false, "missing API key"
	}
	return true, "configured"
}

func buildImageProviderCatalog(cfg ImageGenerationConfig) []ImageProviderCatalogEntry {
	entries := make([]ImageProviderCatalogEntry, 0, len(imageProviderDefinitions))
	for _, definition := range imageProviderDefinitions {
		configured, status := imageProviderConfigured(cfg, definition.id)
		direct := imageProviderConfig(cfg, definition.id)
		models := append([]string(nil), definition.models...)
		models = uniqueNonEmpty(append(models, imageProviderConfig(cfg, definition.id).Models...)...)
		entries = append(entries, ImageProviderCatalogEntry{
			ID: definition.id, DisplayName: definition.displayName, AuthType: definition.authType,
			Default: definition.id == ImageProviderCodexOAuth, Configured: configured,
			APIKeyConfigured: strings.TrimSpace(direct.APIKey) != "", Status: status,
			BaseURL: direct.BaseURL, APIVersion: direct.APIVersion,
			Models: models, ModelValidation: definition.modelValidation, Capabilities: definition.capabilities,
		})
	}
	return entries
}
