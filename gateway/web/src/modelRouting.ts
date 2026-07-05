import type {
  ImageGenerationSetting,
  ImageGenerationUpdate,
  ModelProviderSetting,
  ModelSettings,
  ModelSettingsUpdate,
} from "./types";

export interface ModelProviderDraft {
  enabled: boolean;
  model: string;
  reasoning: string;
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
  baseUrl: string;
  model: string;
  openClawBridgeUrl: string;
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
  return {
    providerPriority: completePriority(
      settings.provider_priority,
      settings.providers.map((provider) => provider.id),
    ),
    providers: Object.fromEntries(
      settings.providers.map((provider) => [
        provider.id,
        {
          enabled: provider.enabled,
          model: provider.model ?? "",
          reasoning: provider.reasoning ?? "",
        },
      ]),
    ),
    utilityModel: settings.active.utility_model,
    repairModel: settings.active.repair_model,
    repairFallbackModels: settings.active.repair_fallback_models.join(", "),
    imageGeneration: imageGenerationDraft(settings.image_generation),
    asciiModel: settings.active.ascii_model,
    embeddingProvider: settings.active.embedding_provider,
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
): string[] {
  const providerIds = settings.providers.map((provider) => provider.id);
  const priority = completePriority(draft.providerPriority, providerIds);
  const enabledProviders = settings.providers.filter(
    (provider) => draft.providers[provider.id]?.enabled,
  );
  const issues: string[] = [];
  if (enabledProviders.length === 0) {
    issues.push("Enable at least one provider.");
  }
  const selectedProvider = priority[0];
  if (selectedProvider && !draft.providers[selectedProvider]?.enabled) {
    issues.push("The first provider in the priority chain must be enabled.");
  }
  for (const provider of enabledProviders) {
    const value = draft.providers[provider.id];
    if (provider.supports_model && !value?.model.trim()) {
      issues.push(`${provider.label} needs a model name.`);
    }
  }
  if (!draft.utilityModel.trim()) {
    issues.push("Utility model is required.");
  }
  if (!draft.embeddingProvider.trim()) {
    issues.push("Embedding provider is required.");
  }
  if (draft.imageGeneration.autoGenerate) {
    if (!draft.imageGeneration.provider.trim()) {
      issues.push("Image generation provider is required when auto-generate is enabled.");
    }
    if (!draft.imageGeneration.model.trim()) {
      issues.push("Image generation model is required when auto-generate is enabled.");
    }
    if (draft.imageGeneration.timeoutSeconds <= 0) {
      issues.push("Image generation timeout must be positive.");
    }
    if (
      isOpenClawImageProvider(draft.imageGeneration.provider) &&
      !draft.imageGeneration.openClawBridgeUrl.trim()
    ) {
      issues.push("OpenClaw image generation needs a bridge URL.");
    }
    if (
      !isOpenClawImageProvider(draft.imageGeneration.provider) &&
      !draft.imageGeneration.baseUrl.trim()
    ) {
      issues.push("OpenAI-compatible image generation needs a base URL.");
    }
    if (
      !isOpenClawImageProvider(draft.imageGeneration.provider) &&
      !settings.image_generation.api_key_configured
    ) {
      issues.push("OpenAI-compatible image generation needs an API key configured outside the browser.");
    }
  }
  return issues;
}

function imageGenerationDraft(
  settings: ImageGenerationSetting,
): ImageGenerationDraft {
  return {
    provider: settings.provider,
    baseUrl: settings.base_url,
    model: settings.model,
    openClawBridgeUrl: settings.openclaw_bridge_url,
    defaultSize: settings.default_size,
    locationSize: settings.location_size,
    characterSize: settings.character_size,
    defaultResolution: settings.default_resolution,
    locationResolution: settings.location_resolution,
    characterResolution: settings.character_resolution,
    defaultAspectRatio: settings.default_aspect_ratio,
    locationAspectRatio: settings.location_aspect_ratio,
    characterAspectRatio: settings.character_aspect_ratio,
    quality: settings.quality,
    outputFormat: settings.output_format,
    background: settings.background,
    timeoutSeconds: settings.timeout_seconds || 180,
    autoGenerate: settings.auto_generate,
    appendNegativePrompt: settings.append_negative_prompt,
  };
}

function imageGenerationUpdate(
  draft: ImageGenerationDraft,
): ImageGenerationUpdate {
  return {
    provider: draft.provider.trim(),
    base_url: draft.baseUrl.trim(),
    model: draft.model.trim(),
    openclaw_bridge_url: draft.openClawBridgeUrl.trim(),
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
      : 180,
    auto_generate: draft.autoGenerate,
    append_negative_prompt: draft.appendNegativePrompt,
  };
}

function isOpenClawImageProvider(provider: string): boolean {
  return ["openclaw", "openclaw-bridge", "codex-oauth"].includes(
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
  };
}
