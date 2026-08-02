package config

import "testing"

func TestImageProviderCatalogKeepsSelectedCustomModelIDs(t *testing.T) {
	cfg := Default().AI.ImageGeneration
	cfg.Provider = ImageProviderReplicate
	cfg.Model = "owner/custom-scene-model"
	cfg.MapIconProvider = ImageProviderAzureOpenAI
	cfg.MapIconModel = "production-image-deployment"
	cfg.Providers = map[string]ImageProviderConfig{
		ImageProviderOpenAICompatible: {
			Models: []string{"configured-compatible-image"},
		},
	}

	catalog := buildImageProviderCatalog(cfg)
	if !containsCatalogModel(catalog, ImageProviderReplicate, "owner/custom-scene-model") {
		t.Fatalf("Replicate catalog lost selected custom model: %#v", catalog)
	}
	if !containsCatalogModel(catalog, ImageProviderAzureOpenAI, "production-image-deployment") {
		t.Fatalf("Azure catalog lost selected deployment name: %#v", catalog)
	}
	if !containsCatalogModel(catalog, ImageProviderOpenAICompatible, "configured-compatible-image") {
		t.Fatalf("compatible catalog lost configured model: %#v", catalog)
	}
}

func TestImageProviderCatalogDoesNotLeakCustomModelsAcrossProviders(t *testing.T) {
	cfg := Default().AI.ImageGeneration
	cfg.Provider = ImageProviderGemini
	cfg.Model = "gemini-custom-image"
	cfg.MapIconProvider = ImageProviderGemini
	cfg.MapIconModel = "gemini-map-icon"

	catalog := buildImageProviderCatalog(cfg)
	if containsCatalogModel(catalog, ImageProviderOpenAI, "gemini-custom-image") {
		t.Fatalf("OpenAI catalog received a Gemini-only custom model: %#v", catalog)
	}
	if !containsCatalogModel(catalog, ImageProviderGemini, "gemini-custom-image") || !containsCatalogModel(catalog, ImageProviderGemini, "gemini-map-icon") {
		t.Fatalf("Gemini catalog lost selected models: %#v", catalog)
	}
}

func TestImageProviderCatalogUsesManagedBridgeRuntimeEnvironment(t *testing.T) {
	t.Setenv("ONEDAY_IMAGEGEN_BRIDGE_URL", "http://127.0.0.1:49152")
	t.Setenv("ONEDAY_IMAGEGEN_BRIDGE_TOKEN", "per-launch-secret")
	cfg := Default().AI.ImageGeneration
	cfg.ImagegenBridgeURL = ""
	cfg.ImagegenBridgeToken = ""

	settings := buildImageGenerationSetting(cfg)
	if !settings.Available || settings.ImagegenBridgeURL != "http://127.0.0.1:49152" {
		t.Fatalf("managed bridge runtime was not reflected: %#v", settings)
	}
	if !settings.ImagegenBridgeTokenConfigured {
		t.Fatal("managed bridge token was not reported as configured")
	}
	catalog := buildImageProviderCatalog(cfg)
	for _, provider := range catalog {
		if provider.ID == ImageProviderCodexOAuth && !provider.Configured {
			t.Fatalf("Codex OAuth provider should be configured: %#v", provider)
		}
	}
}

func TestImageProviderCatalogHonorsExplicitEmptyRuntimeBridgeOverride(t *testing.T) {
	t.Setenv("ONEDAY_IMAGEGEN_BRIDGE_URL", "")
	t.Setenv("ONEDAY_IMAGEGEN_BRIDGE_TOKEN", "")
	cfg := Default().AI.ImageGeneration
	cfg.ImagegenBridgeURL = "http://127.0.0.1:8787"
	cfg.ImagegenBridgeToken = "saved-secret"

	settings := buildImageGenerationSetting(cfg)
	if settings.Available || settings.ImagegenBridgeURL != "" || settings.ImagegenBridgeTokenConfigured {
		t.Fatalf("explicit disabled runtime bridge was ignored: %#v", settings)
	}
}

func containsCatalogModel(catalog []ImageProviderCatalogEntry, provider, model string) bool {
	for _, entry := range catalog {
		if entry.ID != provider {
			continue
		}
		return containsString(entry.Models, model)
	}
	return false
}
