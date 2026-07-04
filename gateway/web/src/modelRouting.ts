import type {
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
  imageModel: string;
  asciiModel: string;
  embeddingProvider: string;
  embeddingModel: string;
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
    imageModel: settings.active.image_model,
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
    image_model: draft.imageModel.trim(),
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
  return issues;
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
