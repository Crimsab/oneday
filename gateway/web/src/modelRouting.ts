import type {
  ImageGenerationUpdate,
  ModelProviderSetting,
  ModelSettings,
  ModelSettingsUpdate,
} from "./types";
import i18n from "./i18n";

export const DEFAULT_CODEX_MODEL = "gpt-5.6-luna";
export const DEFAULT_CODEX_REASONING = "low";
export const DEFAULT_IMAGE_SIZE = "1024x1024";
export const DEFAULT_LOCATION_IMAGE_SIZE = "1536x1024";
export const DEFAULT_CHARACTER_IMAGE_SIZE = "1024x1024";
export const DEFAULT_IMAGE_FORMAT = "png";

export type ModelSetupSection = "connections" | "models" | "images" | "diagnostics";

type PendingProviderConfiguration = {
  providerConfigs: Record<string, { baseUrl: string; apiKey: string; clearApiKey: boolean }>;
  bridgeToken: string;
  clearBridgeToken: boolean;
};

export interface ModelProviderDraft {
  enabled: boolean;
  model: string;
  reasoning: string;
  baseUrl: string;
  apiKey: string;
  clearApiKey: boolean;
}

export interface ModelRoutingDraft {
  providerPriority: string[];
  providers: Record<string, ModelProviderDraft>;
  utilityModel: string;
  repairModel: string;
  repairFallbackModels: string;
  imageGeneration: ImageGenerationDraft;
  asciiModel: string;
  embeddingProvider: string;
  embeddingModel: string;
}

export interface ImageGenerationDraft {
  provider: string;
  mapIconProvider: string;
  baseUrl: string;
  model: string;
  mapIconModel: string;
  openClawBridgeUrl: string;
  imagegenBridgeUrl: string;
  imagegenBridgeProvider: string;
  imagegenBridgeMapIconProvider: string;
  imagegenBridgeFallbacks: string;
  imagegenBridgeFallbackPolicy: string;
  imagegenBridgeCompatibility: string;
  defaultSize: string;
  locationSize: string;
  characterSize: string;
  defaultResolution: string;
  locationResolution: string;
  characterResolution: string;
  defaultAspectRatio: string;
  locationAspectRatio: string;
  characterAspectRatio: string;
  quality: string;
  outputFormat: string;
  background: string;
  timeoutSeconds: number;
  autoGenerate: boolean;
  appendNegativePrompt: boolean;
}

export function draftFromModelSettings(
  settings: ModelSettings,
): ModelRoutingDraft {
  const priority = completePriority(
    settings.provider_priority,
    settings.providers.map((provider) => provider.id),
  );
  const providers = Object.fromEntries(
    settings.providers.map((provider) => {
      const missingCodexModel = provider.id === "codex" && provider.enabled && !provider.model?.trim();
      return [
        provider.id,
        {
          enabled: provider.enabled,
          model: missingCodexModel ? DEFAULT_CODEX_MODEL : provider.model ?? "",
          reasoning: missingCodexModel && (!provider.reasoning || provider.reasoning === "off")
            ? DEFAULT_CODEX_REASONING
            : provider.reasoning ?? "",
          baseUrl: provider.base_url ?? "",
          apiKey: "",
          clearApiKey: false,
        },
      ];
    }),
  );
  const activeProvider = priority.find((id) => providers[id]?.enabled);
  const activeModel = activeProvider ? providers[activeProvider]?.model.trim() : "";
  const narrativeModel = activeModel || settings.active.narrative_model.trim();
  return {
    providerPriority: priority,
    providers,
    utilityModel: settings.active.utility_model || narrativeModel,
    repairModel: settings.active.repair_model,
    repairFallbackModels: settings.active.repair_fallback_models.join(", "),
    imageGeneration: imageGenerationDraft(settings),
    asciiModel: settings.active.ascii_model.trim() || narrativeModel,
    embeddingProvider: settings.active.embedding_provider || "auto",
    embeddingModel: settings.active.embedding_model,
  };
}

export function updateFromDraft(
  settings: ModelSettings,
  draft: ModelRoutingDraft,
): ModelSettingsUpdate {
  return {
    base_revision: settings.config_revision,
    provider_priority: completePriority(
      draft.providerPriority,
      settings.providers.map((provider) => provider.id),
    ),
    providers: settings.providers.map((provider) =>
      providerUpdate(provider, draft.providers[provider.id]),
    ),
    utility_model: draft.utilityModel.trim(),
    repair_model: draft.repairModel.trim(),
    repair_fallback_models: splitModelList(draft.repairFallbackModels),
    image_model: draft.imageGeneration.model.trim(),
    image_generation: imageGenerationUpdate(draft.imageGeneration),
    ascii_model: draft.asciiModel.trim(),
    embedding_provider: draft.embeddingProvider.trim(),
    embedding_model: draft.embeddingModel.trim(),
  };
}

export function hasModelRoutingChanges(
  settings: ModelSettings,
  draft: ModelRoutingDraft,
): boolean {
  return (
    JSON.stringify(draft) !== JSON.stringify(draftFromModelSettings(settings))
  );
}

export function modelRoutingIssues(
  settings: ModelSettings,
  draft: ModelRoutingDraft,
  pending?: PendingProviderConfiguration,
): string[] {
  const groups = modelRoutingIssueGroups(settings, draft, pending);
  return ["connections", "models", "images"].flatMap(
    (section) => groups[section as ModelSetupSection],
  );
}

export function modelRoutingIssueGroups(
  settings: ModelSettings,
  draft: ModelRoutingDraft,
  pending?: PendingProviderConfiguration,
): Record<ModelSetupSection, string[]> {
  const enabledProviders = settings.providers.filter(
    (provider) => draft.providers[provider.id]?.enabled,
  );
  const groups: Record<ModelSetupSection, string[]> = {
    connections: [],
    models: [],
    images: [],
    diagnostics: [],
  };
  if (enabledProviders.length === 0) {
    groups.connections.push(i18n.t("model_issues:providerRequired"));
  }
  for (const provider of enabledProviders) {
    const value = draft.providers[provider.id];
    if (provider.supports_model && !value?.model.trim()) {
      groups.connections.push(i18n.t("model_issues:modelName", { provider: provider.label }));
    }
    if (provider.supports_base_url && !value?.baseUrl.trim()) {
      groups.connections.push(i18n.t("model_issues:providerBaseUrl", { provider: provider.label }));
    }
    if (
      provider.supports_api_key &&
      (value?.clearApiKey || (!provider.api_key_configured && !value?.apiKey.trim()))
    ) {
      groups.connections.push(i18n.t("model_issues:providerApiKey", { provider: provider.label }));
    }
  }
  if (!draft.utilityModel.trim()) {
    groups.models.push(i18n.t("model_issues:utility"));
  }
  if (!draft.embeddingProvider.trim()) {
    groups.images.push(i18n.t("model_issues:embedding"));
  }
  if (draft.imageGeneration.autoGenerate) {
    if (!draft.imageGeneration.provider.trim()) {
      groups.images.push(i18n.t("model_issues:imageProvider"));
    }
    if (!draft.imageGeneration.model.trim()) {
      groups.images.push(i18n.t("model_issues:imageModel"));
    }
    if (!draft.imageGeneration.mapIconModel.trim()) {
      groups.images.push(i18n.t("model_issues:mapIcon"));
    }
    if (draft.imageGeneration.timeoutSeconds <= 0) {
      groups.images.push(i18n.t("model_issues:timeout"));
    }
    groups.images.push(
      ...selectedImageProviderIssues(settings, draft, pending),
    );
  }
  return groups;
}

function selectedImageProviderIssues(
  settings: ModelSettings,
  draft: ModelRoutingDraft,
  pending?: PendingProviderConfiguration,
): string[] {
  const issues: string[] = [];
  const selections = [
    {
      id: draft.imageGeneration.provider.trim(),
      role: i18n.t("drawer:imageSettings.provider"),
    },
    {
      id: draft.imageGeneration.mapIconProvider.trim(),
      role: i18n.t("drawer:imageSettings.mapProvider"),
    },
  ].filter(
    (selection, index, all) =>
      Boolean(selection.id) &&
      all.findIndex((candidate) => candidate.id === selection.id) === index,
  );

  for (const selection of selections) {
    if (isOpenClawImageProvider(selection.id)) {
      if (!draft.imageGeneration.openClawBridgeUrl.trim()) {
        issues.push(`${selection.role}: ${i18n.t("model_issues:openClawUrl")}`);
      }
      continue;
    }
    if (isImagegenBridgeProvider(selection.id)) {
      if (!draft.imageGeneration.imagegenBridgeUrl.trim()) {
        issues.push(`${selection.role}: ${i18n.t("model_issues:nativeUrl")}`);
      }
      continue;
    }

    const catalog = settings.image_providers.find(
      (provider) => provider.id === selection.id,
    );
    if (!catalog) continue;
    const config = pending?.providerConfigs[selection.id];
    const provider = `${catalog.display_name || selection.id} · ${selection.role}`;
    if (!(config?.baseUrl.trim() || catalog.base_url.trim())) {
      issues.push(i18n.t("model_issues:providerBaseUrl", { provider }));
    }
    const keyReady = !config?.clearApiKey && (
      catalog.api_key_configured || Boolean(config?.apiKey.trim())
    );
    if (!keyReady) {
      issues.push(i18n.t("model_issues:providerApiKey", { provider }));
    }
  }
  return issues;
}

function imageGenerationDraft(settings: ModelSettings): ImageGenerationDraft {
  const imageSettings = settings.image_generation;
  const preferredModel = (providerId: string, saved: string) => {
    if (saved.trim()) return saved;
    return settings.image_providers.find((provider) => provider.id === providerId)?.models[0] ?? "";
  };
  return {
    provider: imageSettings.provider,
    mapIconProvider: imageSettings.map_icon_provider,
    baseUrl: imageSettings.base_url,
    model: preferredModel(imageSettings.provider, imageSettings.model),
    mapIconModel: preferredModel(imageSettings.map_icon_provider, imageSettings.map_icon_model),
    openClawBridgeUrl: imageSettings.openclaw_bridge_url,
    imagegenBridgeUrl: imageSettings.imagegen_bridge_url,
    imagegenBridgeProvider: imageSettings.imagegen_bridge_provider,
    imagegenBridgeMapIconProvider: imageSettings.imagegen_bridge_map_icon_provider,
    imagegenBridgeFallbacks: imageSettings.imagegen_bridge_fallbacks.join(", "),
    imagegenBridgeFallbackPolicy: imageSettings.imagegen_bridge_fallback_policy,
    imagegenBridgeCompatibility: imageSettings.imagegen_bridge_compatibility,
    defaultSize: imageSettings.default_size.trim() || DEFAULT_IMAGE_SIZE,
    locationSize: imageSettings.location_size.trim() || DEFAULT_LOCATION_IMAGE_SIZE,
    characterSize: imageSettings.character_size.trim() || DEFAULT_CHARACTER_IMAGE_SIZE,
    defaultResolution: imageSettings.default_resolution,
    locationResolution: imageSettings.location_resolution,
    characterResolution: imageSettings.character_resolution,
    defaultAspectRatio: imageSettings.default_aspect_ratio,
    locationAspectRatio: imageSettings.location_aspect_ratio,
    characterAspectRatio: imageSettings.character_aspect_ratio,
    quality: imageSettings.quality,
    outputFormat: imageSettings.output_format.trim() || DEFAULT_IMAGE_FORMAT,
    background: imageSettings.background,
    timeoutSeconds: imageSettings.timeout_seconds || 360,
    autoGenerate: imageSettings.auto_generate,
    appendNegativePrompt: imageSettings.append_negative_prompt,
  };
}

function imageGenerationUpdate(
  draft: ImageGenerationDraft,
): ImageGenerationUpdate {
  return {
    provider: draft.provider.trim(),
    map_icon_provider: draft.mapIconProvider.trim(),
    base_url: draft.baseUrl.trim(),
    model: draft.model.trim(),
    map_icon_model: draft.mapIconModel.trim(),
    openclaw_bridge_url: draft.openClawBridgeUrl.trim(),
    imagegen_bridge_url: draft.imagegenBridgeUrl.trim(),
    imagegen_bridge_provider: draft.imagegenBridgeProvider.trim(),
    imagegen_bridge_map_icon_provider: draft.imagegenBridgeMapIconProvider.trim(),
    imagegen_bridge_fallbacks: splitModelList(draft.imagegenBridgeFallbacks),
    imagegen_bridge_fallback_policy: draft.imagegenBridgeFallbackPolicy.trim(),
    imagegen_bridge_compatibility: draft.imagegenBridgeCompatibility.trim(),
    default_size: draft.defaultSize.trim(),
    location_size: draft.locationSize.trim(),
    character_size: draft.characterSize.trim(),
    default_resolution: draft.defaultResolution.trim(),
    location_resolution: draft.locationResolution.trim(),
    character_resolution: draft.characterResolution.trim(),
    default_aspect_ratio: draft.defaultAspectRatio.trim(),
    location_aspect_ratio: draft.locationAspectRatio.trim(),
    character_aspect_ratio: draft.characterAspectRatio.trim(),
    quality: draft.quality.trim(),
    output_format: draft.outputFormat.trim(),
    background: draft.background.trim(),
    timeout_seconds: Number.isFinite(draft.timeoutSeconds)
      ? Math.max(1, Math.round(draft.timeoutSeconds))
      : 360,
    auto_generate: draft.autoGenerate,
    append_negative_prompt: draft.appendNegativePrompt,
  };
}

function isOpenClawImageProvider(provider: string): boolean {
  return ["openclaw", "openclaw-bridge"].includes(
    provider.trim().toLowerCase(),
  );
}

function isImagegenBridgeProvider(provider: string): boolean {
  return ["codex-oauth", "imagegen-bridge", "imagegen_bridge", "bridge-native"].includes(
    provider.trim().toLowerCase(),
  );
}

export function promoteProvider(
  priority: string[],
  providerIds: string[],
  providerId: string,
): string[] {
  const id = providerId.trim();
  if (!providerIds.includes(id)) return completePriority(priority, providerIds);
  return [
    id,
    ...completePriority(priority, providerIds).filter((item) => item !== id),
  ];
}

export function splitModelList(value: string): string[] {
  return value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .filter((item, index, values) => values.indexOf(item) === index);
}

function completePriority(priority: string[], providerIds: string[]): string[] {
  const clean = priority
    .map((item) => item.trim())
    .filter((item) => providerIds.includes(item));
  return [...clean, ...providerIds.filter((id) => !clean.includes(id))];
}

function providerUpdate(
  provider: ModelProviderSetting,
  draft?: ModelProviderDraft,
) {
  return {
    id: provider.id,
    enabled: draft?.enabled ?? provider.enabled,
    ...(provider.supports_model
      ? { model: (draft?.model ?? provider.model ?? "").trim() }
      : {}),
    ...(provider.supports_reasoning
      ? { reasoning: (draft?.reasoning ?? provider.reasoning ?? "").trim() }
      : {}),
    ...(provider.supports_base_url
      ? { base_url: (draft?.baseUrl ?? provider.base_url ?? "").trim() }
      : {}),
    ...(provider.supports_api_key && draft?.apiKey.trim()
      ? { api_key: draft.apiKey.trim() }
      : {}),
    ...(provider.supports_api_key && draft?.clearApiKey
      ? { clear_api_key: true }
      : {}),
  };
}
