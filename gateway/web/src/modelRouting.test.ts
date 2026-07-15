import { describe, expect, it } from "vitest";
import {
  draftFromModelSettings,
  hasModelRoutingChanges,
  modelRoutingIssues,
  promoteProvider,
  splitModelList,
  updateFromDraft,
} from "./modelRouting";
import type { ImageProviderCatalogEntry, ModelSettings } from "./types";

describe("model routing helpers", () => {
  it("builds editable drafts from backend settings", () => {
    const draft = draftFromModelSettings(settings);

    expect(draft.providerPriority).toEqual([
      "codex",
      "litellm",
      "openrouter",
      "claude-code",
    ]);
    expect(draft.providers.codex).toMatchObject({
      enabled: true,
      model: "test-codex-model",
      reasoning: "off",
    });
    expect(draft.repairFallbackModels).toBe("test-repair-fallback");
    expect(draft.imageGeneration).toMatchObject({
      provider: "openclaw-bridge",
      model: "test-image-model",
      mapIconModel: "openai/gpt-image-1",
      openClawBridgeUrl: "http://openclaw-imagegen:8099/generate",
    });
  });

  it("promotes providers while preserving a complete priority chain", () => {
    expect(
      promoteProvider(
        ["codex", "litellm"],
        ["codex", "litellm", "openrouter"],
        "openrouter",
      ),
    ).toEqual(["openrouter", "codex", "litellm"]);
  });

  it("deduplicates comma and newline fallback model lists", () => {
    expect(splitModelList("test-model-a, test-model-b\ntest-model-a")).toEqual([
      "test-model-a",
      "test-model-b",
    ]);
  });

  it("creates a shared config update payload", () => {
    const draft = draftFromModelSettings(settings);
    draft.providerPriority = promoteProvider(
      draft.providerPriority,
      settings.providers.map((provider) => provider.id),
      "litellm",
    );
    draft.providers.litellm.enabled = true;
    draft.providers.litellm.model = "test-litellm-model-updated";
    draft.utilityModel = "test-utility-model";
    draft.repairFallbackModels = "test-fallback-one, test-fallback-two";
    draft.imageGeneration.autoGenerate = true;
    draft.imageGeneration.locationSize = "1792x1024";

    expect(updateFromDraft(settings, draft)).toMatchObject({
      base_revision: "revision-1",
      provider_priority: ["litellm", "codex", "openrouter", "claude-code"],
      utility_model: "test-utility-model",
      repair_fallback_models: ["test-fallback-one", "test-fallback-two"],
      image_model: "test-image-model",
      image_generation: expect.objectContaining({
        provider: "openclaw-bridge",
        model: "test-image-model",
        map_icon_model: "openai/gpt-image-1",
        location_size: "1792x1024",
        auto_generate: true,
      }),
      providers: expect.arrayContaining([
        expect.objectContaining({
          id: "litellm",
          enabled: true,
          model: "test-litellm-model-updated",
        }),
      ]),
    });
  });

  it("detects dirty and invalid drafts before save", () => {
    const draft = draftFromModelSettings(settings);
    expect(hasModelRoutingChanges(settings, draft)).toBe(false);
    draft.providers.codex.enabled = false;
    draft.providers.litellm.enabled = false;
    draft.providers.openrouter.enabled = false;
    draft.providers["claude-code"].enabled = false;
    expect(hasModelRoutingChanges(settings, draft)).toBe(true);
    expect(modelRoutingIssues(settings, draft)).toContain(
      "Enable at least one provider.",
    );
  });

  it("treats Codex OAuth as imagegen-bridge transport and validates its URL", () => {
    const bridgeSettings: ModelSettings = {
      ...settings,
      image_generation: {
        ...settings.image_generation,
        provider: "codex-oauth",
        map_icon_provider: "codex-oauth",
        model: "gpt-image-2",
        map_icon_model: "gpt-image-2",
        imagegen_bridge_url: "http://imagegen-bridge:8787",
      },
    };
    const draft = draftFromModelSettings(bridgeSettings);
    draft.imageGeneration.autoGenerate = true;
    expect(modelRoutingIssues(bridgeSettings, draft)).toEqual([]);

    draft.imageGeneration.imagegenBridgeUrl = "";
    expect(modelRoutingIssues(bridgeSettings, draft)).toContain(
      "imagegen-bridge needs its native API URL.",
    );
  });

  it("accepts a fresh Codex bridge URL without an optional bearer token", () => {
    const bridgeSettings = withImageProviders([
      imageProvider("codex-oauth", { configured: false, base_url: "" }),
    ]);
    const draft = draftFromModelSettings(bridgeSettings);
    draft.imageGeneration.autoGenerate = true;
    draft.imageGeneration.provider = "codex-oauth";
    draft.imageGeneration.mapIconProvider = "codex-oauth";
    draft.imageGeneration.imagegenBridgeUrl = "http://localhost:8787";

    expect(
      modelRoutingIssues(bridgeSettings, draft, {
        providerConfigs: {},
        bridgeToken: "",
        clearBridgeToken: false,
      }),
    ).toEqual([]);
  });

  it("accepts a fresh Gemini setup with a pending write-only key", () => {
    const providerSettings = withImageProviders([
      imageProvider("gemini", { base_url: "https://generativelanguage.googleapis.com" }),
    ]);
    const draft = draftFromModelSettings(providerSettings);
    draft.imageGeneration.autoGenerate = true;
    draft.imageGeneration.provider = "gemini";
    draft.imageGeneration.mapIconProvider = "gemini";
    draft.imageGeneration.model = "imagen-4.0-generate-001";
    draft.imageGeneration.mapIconModel = "imagen-4.0-generate-001";

    expect(
      modelRoutingIssues(providerSettings, draft, {
        providerConfigs: {
          gemini: { baseUrl: "", apiKey: "pending-secret", clearApiKey: false },
        },
        bridgeToken: "",
        clearBridgeToken: false,
      }),
    ).toEqual([]);
  });

  it("accepts a fresh compatible endpoint and a separately configured map provider", () => {
    const providerSettings = withImageProviders([
      imageProvider("openai-compatible", { base_url: "" }),
      imageProvider("gemini", { base_url: "https://generativelanguage.googleapis.com" }),
    ]);
    const draft = draftFromModelSettings(providerSettings);
    draft.imageGeneration.autoGenerate = true;
    draft.imageGeneration.provider = "openai-compatible";
    draft.imageGeneration.mapIconProvider = "gemini";
    draft.imageGeneration.model = "custom-scene-model";
    draft.imageGeneration.mapIconModel = "imagen-map-model";

    expect(
      modelRoutingIssues(providerSettings, draft, {
        providerConfigs: {
          "openai-compatible": {
            baseUrl: "http://lite.homelab.local/v1",
            apiKey: "pending-compatible-secret",
            clearApiKey: false,
          },
          gemini: {
            baseUrl: "",
            apiKey: "pending-gemini-secret",
            clearApiKey: false,
          },
        },
        bridgeToken: "",
        clearBridgeToken: false,
      }),
    ).toEqual([]);
  });
});

function imageProvider(
  id: string,
  overrides: Partial<ImageProviderCatalogEntry> = {},
): ImageProviderCatalogEntry {
  return {
    id,
    display_name: id,
    auth_type: "api-key",
    default: id === "codex-oauth",
    configured: false,
    api_key_configured: false,
    status: "",
    base_url: "https://example.test",
    models: [],
    model_validation: "provider-defined",
    capabilities: {
      generate: true,
      edit: false,
      sizes: [],
      aspect_ratios: [],
      qualities: [],
      output_formats: ["png"],
      supports_transparency: false,
    },
    ...overrides,
  };
}

function withImageProviders(
  imageProviders: ImageProviderCatalogEntry[],
): ModelSettings {
  return {
    ...settings,
    image_providers: imageProviders,
    image_generation: {
      ...settings.image_generation,
      provider: imageProviders[0]?.id ?? "codex-oauth",
      map_icon_provider: imageProviders[0]?.id ?? "codex-oauth",
      imagegen_bridge_url: "",
      imagegen_bridge_token_configured: false,
      api_key_configured: false,
      available: false,
      status: "",
    },
  };
}

const settings: ModelSettings = {
  config_path: "/opt/oneday/config.yaml",
  config_revision: "revision-1",
  provider_priority: ["codex", "litellm"],
  providers: [
    {
      id: "codex",
      label: "Codex OAuth",
      enabled: true,
      model: "test-codex-model",
      reasoning: "off",
      supports_model: true,
      supports_reasoning: true,
    },
    {
      id: "litellm",
      label: "LiteLLM",
      enabled: false,
      model: "test-litellm-model",
      supports_model: true,
      supports_reasoning: false,
    },
    {
      id: "openrouter",
      label: "OpenRouter",
      enabled: false,
      model: "test-openrouter-model",
      supports_model: true,
      supports_reasoning: false,
    },
    {
      id: "claude-code",
      label: "Claude Code",
      enabled: false,
      supports_model: false,
      supports_reasoning: false,
    },
  ],
  narrative_models: ["test-codex-model", "test-openrouter-model"],
  utility_models: ["test-utility-model"],
  repair_models: ["test-repair-model", "test-repair-fallback"],
  image_models: ["test-image-model", "openai/gpt-image-1"],
  ascii_models: ["test-ascii-model"],
  embedding_providers: ["auto", "litellm", "openrouter", "local"],
  image_providers: [],
  image_generation: {
    provider: "openclaw-bridge",
    map_icon_provider: "openclaw-bridge",
    base_url: "",
    api_key_configured: false,
    model: "test-image-model",
    map_icon_model: "openai/gpt-image-1",
    openclaw_bridge_url: "http://openclaw-imagegen:8099/generate",
    imagegen_bridge_url: "http://imagegen-bridge:8787",
    imagegen_bridge_token_configured: true,
    imagegen_bridge_provider: "codex-app-server",
    imagegen_bridge_map_icon_provider: "codex-app-server",
    imagegen_bridge_fallbacks: [],
    imagegen_bridge_fallback_policy: "on_unavailable",
    imagegen_bridge_compatibility: "normalize",
    default_size: "1024x1024",
    location_size: "1536x1024",
    character_size: "1024x1024",
    default_resolution: "",
    location_resolution: "",
    character_resolution: "",
    default_aspect_ratio: "",
    location_aspect_ratio: "",
    character_aspect_ratio: "",
    quality: "",
    output_format: "png",
    background: "",
    timeout_seconds: 180,
    auto_generate: false,
    append_negative_prompt: true,
    available: true,
    status: "configured through OpenClaw bridge",
  },
  active: {
    provider: "codex",
    narrative_model: "test-codex-model",
    utility_model: "test-utility-model",
    repair_model: "test-repair-model",
    repair_fallback_models: ["test-repair-fallback"],
    image_model: "test-image-model",
    ascii_model: "test-ascii-model",
    embedding_provider: "auto",
    embedding_model: "test-embedding-model",
    codex_reasoning: "off",
  },
  tts_status: "disabled",
};
