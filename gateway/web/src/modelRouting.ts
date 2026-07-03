import type { ModelProviderSetting, ModelSettings, ModelSettingsUpdate } from "./types";

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
  imageModel: string;
  embeddingProvider: string;
  embeddingModel: string;
}

export function draftFromModelSettings(settings: ModelSettings): ModelRoutingDraft {
  return {
    providerPriority: completePriority(settings.provider_priority, settings.providers.map((provider) => provider.id)),
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
    imageModel: settings.active.image_model,
    embeddingProvider: settings.active.embedding_provider,
    embeddingModel: settings.active.embedding_model,
  };
}

export function updateFromDraft(settings: ModelSettings, draft: ModelRoutingDraft): ModelSettingsUpdate {
  return {
    provider_priority: completePriority(draft.providerPriority, settings.providers.map((provider) => provider.id)),
    providers: settings.providers.map((provider) => providerUpdate(provider, draft.providers[provider.id])),
    utility_model: draft.utilityModel.trim(),
    repair_model: draft.repairModel.trim(),
    repair_fallback_models: splitModelList(draft.repairFallbackModels),
    image_model: draft.imageModel.trim(),
    embedding_provider: draft.embeddingProvider.trim(),
    embedding_model: draft.embeddingModel.trim(),
  };
}

export function promoteProvider(priority: string[], providerIds: string[], providerId: string): string[] {
  const id = providerId.trim();
  if (!providerIds.includes(id)) return completePriority(priority, providerIds);
  return [id, ...completePriority(priority, providerIds).filter((item) => item !== id)];
}

export function splitModelList(value: string): string[] {
  return value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .filter((item, index, values) => values.indexOf(item) === index);
}

function completePriority(priority: string[], providerIds: string[]): string[] {
  const clean = priority.map((item) => item.trim()).filter((item) => providerIds.includes(item));
  return [...clean, ...providerIds.filter((id) => !clean.includes(id))];
}

function providerUpdate(provider: ModelProviderSetting, draft?: ModelProviderDraft) {
  return {
    id: provider.id,
    enabled: draft?.enabled ?? provider.enabled,
    ...(provider.supports_model ? { model: (draft?.model ?? provider.model ?? "").trim() } : {}),
    ...(provider.supports_reasoning ? { reasoning: (draft?.reasoning ?? provider.reasoning ?? "").trim() } : {}),
  };
}
