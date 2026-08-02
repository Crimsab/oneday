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
	Operations           []ImageOperationCapability `json:"operations"`
	Sizes                []string                   `json:"sizes"`
	AspectRatios         []string                   `json:"aspect_ratios"`
	Qualities            []string                   `json:"qualities"`
	OutputFormats        []string                   `json:"output_formats"`
	SupportsTransparency bool                       `json:"supports_transparency"`
}

type ImageOperationCapability struct {
	Operation      string                 `json:"operation"`
	Supported      bool                   `json:"supported"`
	Availability   string                 `json:"availability"`
	EndpointID     string                 `json:"endpoint_id"`
	Models         []string               `json:"models"`
	ModelVersion   string                 `json:"model_version,omitempty"`
	CredentialMode string                 `json:"credential_mode"`
	APIVersion     string                 `json:"api_version,omitempty"`
	SchemaRevision string                 `json:"schema_revision"`
	SourceImages   *ImageSourceCapability `json:"source_images,omitempty"`
	Mask           *ImageMaskCapability   `json:"mask,omitempty"`
	Controls       ImageControlCapability `json:"controls"`
	Provenance     CapabilityProvenance   `json:"provenance"`
}

type ImageSourceCapability struct {
	Min   int      `json:"min"`
	Max   int      `json:"max"`
	Roles []string `json:"roles"`
}

type ImageMaskCapability struct {
	Required          bool     `json:"required"`
	Kind              string   `json:"kind"`
	AcceptedFormats   []string `json:"accepted_formats"`
	SoftValues        string   `json:"soft_values"`
	ProviderSemantics string   `json:"provider_semantics"`
	Adherence         string   `json:"adherence"`
}

type ImageControlCapability struct {
	NegativePrompt bool     `json:"negative_prompt"`
	Strength       bool     `json:"strength"`
	Seed           bool     `json:"seed"`
	QualityValues  []string `json:"quality_values"`
	OutputFormats  []string `json:"output_formats"`
}

type CapabilityProvenance struct {
	Kind           string `json:"kind"`
	VerifiedAt     string `json:"verified_at"`
	SchemaRevision string `json:"schema_revision"`
}

const imageCapabilityVerifiedAt = "2026-07-15"

func operationCapability(operation, endpoint, credential string, models []string, source *ImageSourceCapability, mask *ImageMaskCapability, supported bool, availability, provenance string) ImageOperationCapability {
	return ImageOperationCapability{
		Operation: operation, Supported: supported, Availability: availability,
		EndpointID: endpoint, Models: append([]string{}, models...), CredentialMode: credential,
		SchemaRevision: "oneday-image-operations-v1", SourceImages: source, Mask: mask,
		Controls:   ImageControlCapability{QualityValues: []string{}, OutputFormats: []string{"png", "jpeg"}},
		Provenance: CapabilityProvenance{Kind: provenance, VerifiedAt: imageCapabilityVerifiedAt, SchemaRevision: "oneday-image-operations-v1"},
	}
}

func sourceOne() *ImageSourceCapability {
	return &ImageSourceCapability{Min: 1, Max: 1, Roles: []string{"source"}}
}

func rasterMask(adherence, semantics string) *ImageMaskCapability {
	return &ImageMaskCapability{Required: true, Kind: "raster", AcceptedFormats: []string{"image/png"}, SoftValues: "supported", ProviderSemantics: semantics, Adherence: adherence}
}

func generationCapability(endpoint, credential string, models []string) ImageOperationCapability {
	return operationCapability("generate", endpoint, credential, models, nil, nil, true, "available", "static_verified")
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
		capabilities: ImageProviderCapabilities{Operations: []ImageOperationCapability{
			generationCapability("/v1/images", "codex_oauth", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}),
			operationCapability("edit", "/v1/images/edits", "codex_oauth", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, sourceOne(), nil, true, "requires_configuration", "runtime_probe"),
			operationCapability("inpaint", "/v1/images/edits", "codex_oauth", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, sourceOne(), rasterMask("best_effort", "model_specific"), false, "unavailable", "provider_schema"),
		}, Sizes: []string{"1024x1024", "1024x1536", "1536x1024"}, Qualities: []string{"auto", "low", "medium", "high"}, OutputFormats: []string{"png", "jpeg", "webp"}, SupportsTransparency: true},
	},
	{
		id: ImageProviderOpenAI, displayName: "OpenAI Platform", authType: "api_key",
		models: []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, modelValidation: "allowlist",
		capabilities: ImageProviderCapabilities{Operations: []ImageOperationCapability{
			generationCapability("/images/generations", "api_key", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}),
			operationCapability("edit", "/images/edits", "api_key", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, sourceOne(), nil, true, "available", "static_verified"),
			operationCapability("inpaint", "/images/edits", "api_key", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, sourceOne(), rasterMask("best_effort", "transparent_is_edit"), true, "available", "static_verified"),
		}, Sizes: []string{"1024x1024", "1024x1536", "1536x1024"}, Qualities: []string{"auto", "low", "medium", "high"}, OutputFormats: []string{"png", "jpeg", "webp"}, SupportsTransparency: true},
	},
	{
		id: ImageProviderOpenAICompatible, displayName: "OpenAI-compatible", authType: "api_key",
		modelValidation: "configured", capabilities: ImageProviderCapabilities{Operations: []ImageOperationCapability{generationCapability("/images/generations", "api_key", nil)}, Sizes: []string{"provider-defined"}, AspectRatios: []string{"provider-defined"}, Qualities: []string{"provider-defined"}, OutputFormats: []string{"png", "jpeg", "webp"}, SupportsTransparency: true},
	},
	{
		id: ImageProviderGemini, displayName: "Google Gemini", authType: "api_key",
		models: []string{"gemini-3.1-flash-image", "gemini-3.1-flash-lite-image", "gemini-3-pro-image"}, modelValidation: "allowlist_or_gemini_image_model",
		capabilities: ImageProviderCapabilities{Operations: []ImageOperationCapability{
			generationCapability("v1beta/interactions", "api_key", []string{"gemini-3.1-flash-image", "gemini-3.1-flash-lite-image", "gemini-3-pro-image"}),
			operationCapability("edit", "v1beta/interactions", "api_key", []string{"gemini-3.1-flash-image", "gemini-3.1-flash-lite-image", "gemini-3-pro-image"}, sourceOne(), nil, true, "available", "static_verified"),
			operationCapability("inpaint", "v1beta/interactions", "api_key", []string{"gemini-3.1-flash-image", "gemini-3.1-flash-lite-image", "gemini-3-pro-image"}, sourceOne(), rasterMask("best_effort", "model_specific"), false, "unavailable", "static_verified"),
		}, AspectRatios: []string{"1:1", "3:2", "2:3", "4:3", "3:4", "16:9", "9:16"}, OutputFormats: []string{"png", "jpeg"}},
	},
	{
		id: ImageProviderFal, displayName: "fal.ai", authType: "api_key",
		models: []string{"fal-ai/flux/schnell", "fal-ai/nano-banana-2"}, modelValidation: "vendor_slug",
		capabilities: ImageProviderCapabilities{Operations: []ImageOperationCapability{generationCapability("queue/{model}", "api_key", nil)}, AspectRatios: []string{"model-defined"}, OutputFormats: []string{"png", "jpeg", "webp"}},
	},
	{
		id: ImageProviderReplicate, displayName: "Replicate", authType: "api_token",
		models: []string{"black-forest-labs/flux-schnell"}, modelValidation: "owner_model_slug",
		capabilities: ImageProviderCapabilities{Operations: []ImageOperationCapability{generationCapability("models/{owner}/{model}/predictions", "api_token", nil)}, AspectRatios: []string{"model-defined"}, OutputFormats: []string{"png", "jpeg", "webp"}},
	},
	{
		id: ImageProviderStability, displayName: "Stability AI", authType: "api_key",
		models: []string{"stable-image-core"}, modelValidation: "allowlist",
		capabilities: ImageProviderCapabilities{Operations: []ImageOperationCapability{
			generationCapability("/stable-image/generate/core", "api_key", []string{"stable-image-core"}),
			stabilityInpaintCapability(),
		}, AspectRatios: []string{"1:1", "16:9", "9:16", "3:2", "2:3", "4:5", "5:4", "21:9", "9:21"}, OutputFormats: []string{"png", "jpeg", "webp"}},
	},
	{
		id: ImageProviderAzureOpenAI, displayName: "Azure OpenAI Images", authType: "api_key",
		modelValidation: "deployment_name",
		capabilities: ImageProviderCapabilities{Operations: []ImageOperationCapability{
			generationCapability("/openai/v1/images/generations", "api_key", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}),
			operationCapability("edit", "/images/edits", "api_key", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, sourceOne(), nil, true, "requires_configuration", "static_verified"),
			operationCapability("inpaint", "/images/edits", "api_key", []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"}, sourceOne(), rasterMask("best_effort", "transparent_is_edit"), true, "requires_configuration", "static_verified"),
		}, Sizes: []string{"1024x1024", "1024x1536", "1536x1024"}, Qualities: []string{"auto", "low", "medium", "high"}, OutputFormats: []string{"png", "jpeg"}, SupportsTransparency: true},
	},
}

func stabilityInpaintCapability() ImageOperationCapability {
	capability := operationCapability("inpaint", "/stable-image/edit/inpaint", "api_key", []string{"stable-image-core"}, sourceOne(), rasterMask("region_constrained", "white_is_edit"), true, "available", "static_verified")
	capability.Controls.NegativePrompt = true
	return capability
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
		bridgeURL, _ := imagegenBridgeRuntimeCredentials(cfg)
		if bridgeURL == "" {
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
	if provider == ImageProviderOpenAICompatible {
		if strings.TrimSpace(direct.CapabilityProbeURL) == "" {
			return false, "missing explicit image capability probe URL"
		}
		if direct.AuthMode != "" && direct.AuthMode != "bearer" && direct.AuthMode != "none" {
			return false, "auth mode must be bearer or none"
		}
		if direct.AuthMode != "none" && strings.TrimSpace(direct.APIKey) == "" {
			return false, "missing API key for bearer authentication"
		}
		return true, "configured; image capability is probed before dispatch"
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
		// A saved custom ID must remain selectable even when it is not part of a
		// provider's static catalog. This is especially important for Azure
		// deployment names and explicitly configured compatible endpoints.
		models = uniqueNonEmpty(append(models, direct.Models...)...)
		models = uniqueNonEmpty(append(models, selectedImageProviderModels(cfg, definition.id)...)...)
		entries = append(entries, ImageProviderCatalogEntry{
			ID: definition.id, DisplayName: definition.displayName, AuthType: definition.authType,
			Default: definition.id == ImageProviderCodexOAuth, Configured: configured,
			APIKeyConfigured: strings.TrimSpace(direct.APIKey) != "", Status: status,
			BaseURL: direct.BaseURL, APIVersion: direct.APIVersion,
			Models: models, ModelValidation: definition.modelValidation,
			Capabilities: normalizedImageProviderCapabilities(definition.capabilities),
		})
	}
	return entries
}

func selectedImageProviderModels(cfg ImageGenerationConfig, provider string) []string {
	provider = canonicalImageProviderID(provider)
	models := []string{}
	if canonicalImageProviderID(cfg.Provider) == provider {
		models = append(models, cfg.Model)
	}
	if canonicalImageProviderID(cfg.MapIconProvider) == provider {
		models = append(models, cfg.MapIconModel)
	}
	return cleanStringSlice(models)
}

func normalizedImageProviderCapabilities(capabilities ImageProviderCapabilities) ImageProviderCapabilities {
	// The gateway contract requires arrays, not null. Copy through a non-nil
	// slice so providers without a size, aspect-ratio, or quality allowlist
	// still serialize as [] and can be decoded by every generated client.
	capabilities.Sizes = append([]string{}, capabilities.Sizes...)
	capabilities.AspectRatios = append([]string{}, capabilities.AspectRatios...)
	capabilities.Qualities = append([]string{}, capabilities.Qualities...)
	capabilities.OutputFormats = append([]string{}, capabilities.OutputFormats...)
	capabilities.Operations = append([]ImageOperationCapability{}, capabilities.Operations...)
	for index := range capabilities.Operations {
		operation := &capabilities.Operations[index]
		operation.Models = append([]string{}, operation.Models...)
		operation.Controls.QualityValues = append([]string{}, operation.Controls.QualityValues...)
		operation.Controls.OutputFormats = append([]string{}, operation.Controls.OutputFormats...)
		if operation.SourceImages != nil {
			copy := *operation.SourceImages
			copy.Roles = append([]string{}, copy.Roles...)
			operation.SourceImages = &copy
		}
		if operation.Mask != nil {
			copy := *operation.Mask
			copy.AcceptedFormats = append([]string{}, copy.AcceptedFormats...)
			operation.Mask = &copy
		}
	}
	return capabilities
}
