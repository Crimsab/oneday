import type {
  ImageProviderCatalogEntry,
  ImageProviderConfigUpdate,
} from "../../types";

export interface ProviderConfigDraft {
  baseUrl: string;
  apiVersion: string;
  models: string;
  apiKey: string;
  clearApiKey: boolean;
}

export function buildProviderConfigUpdates(
  catalog: ImageProviderCatalogEntry[],
  configs: Record<string, ProviderConfigDraft>,
  dirtyIds: Set<string>,
): ImageProviderConfigUpdate[] {
  return catalog
    .filter(
      (provider) => provider.id !== "codex-oauth" && dirtyIds.has(provider.id),
    )
    .map((provider) => {
      const config = configs[provider.id];
      const update: ImageProviderConfigUpdate = {
        id: provider.id,
        api_key: config?.apiKey || undefined,
        clear_api_key: config?.clearApiKey || undefined,
      };
      if (config?.baseUrl !== provider.base_url)
        update.base_url = config?.baseUrl.trim();
      if (config?.apiVersion !== (provider.api_version ?? ""))
        update.api_version = config?.apiVersion.trim();
      if (config?.models !== provider.models.join(", "))
        update.models = (config?.models ?? "")
          .split(",")
          .map((value) => value.trim())
          .filter(Boolean);
      return update;
    });
}

export const editableImageDraftFields = [
  "provider",
  "mapIconProvider",
  "model",
  "mapIconModel",
  "imagegenBridgeUrl",
  "imagegenBridgeProvider",
  "imagegenBridgeMapIconProvider",
  "imagegenBridgeFallbacks",
  "imagegenBridgeFallbackPolicy",
  "imagegenBridgeCompatibility",
  "defaultSize",
  "locationSize",
  "characterSize",
  "defaultResolution",
  "locationResolution",
  "characterResolution",
  "defaultAspectRatio",
  "locationAspectRatio",
  "characterAspectRatio",
  "quality",
  "outputFormat",
  "background",
  "timeoutSeconds",
  "autoGenerate",
  "appendNegativePrompt",
] as const;
