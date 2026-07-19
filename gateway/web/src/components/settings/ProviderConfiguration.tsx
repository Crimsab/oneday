import { useTranslation } from "react-i18next";
import type { ImageGenerationDraft } from "../../modelRouting";
import type { ImageProviderCatalogEntry } from "../../types";
import type { ProviderConfigDraft } from "./imageGenerationDraft";

interface Props {
  provider: ImageProviderCatalogEntry;
  draft: ImageGenerationDraft;
  providerConfigs: Record<string, ProviderConfigDraft>;
  bridgeToken: string;
  clearBridgeToken: boolean;
  onImageChange: (patch: Partial<ImageGenerationDraft>) => void;
  onProviderConfig: (id: string, patch: Partial<ProviderConfigDraft>) => void;
  onBridgeToken: (value: string) => void;
  onClearBridgeToken: (value: boolean) => void;
}

export function ProviderConfiguration({
  provider,
  draft,
  providerConfigs,
  bridgeToken,
  clearBridgeToken,
  onImageChange,
  onProviderConfig,
  onBridgeToken,
  onClearBridgeToken,
}: Props) {
  const { t } = useTranslation("drawer");
  const config = providerConfigs[provider.id];
  const codex = provider.id === "codex-oauth";
  return (
    <section
      className="provider-configuration"
      role="group"
      aria-label={`${t("imageSettings.provider")}: ${provider.display_name}`}
    >
      <h5>
        {t("imageSettings.provider")}: {provider.display_name}
      </h5>
      <div className="settings-grid">
        {codex ? (
          <>
            <label>
              <span>{t("imageSettings.bridgeUrl")}</span>
              <input
                type="url"
                value={draft.imagegenBridgeUrl}
                onChange={(event) =>
                  onImageChange({ imagegenBridgeUrl: event.target.value })
                }
              />
            </label>
            <label>
              <span>{t("imageSettings.bridgeTokenOptional")}</span>
              <input
                type="password"
                autoComplete="new-password"
                value={bridgeToken}
                onChange={(event) => {
                  onBridgeToken(event.target.value);
                  if (event.target.value) onClearBridgeToken(false);
                }}
                placeholder={t("imageSettings.secretPlaceholder")}
              />
            </label>
            <label className="toggle-row">
              <span>{t("imageSettings.clearSecret")}</span>
              <input
                type="checkbox"
                checked={clearBridgeToken}
                onChange={(event) => {
                  onClearBridgeToken(event.target.checked);
                  if (event.target.checked) onBridgeToken("");
                }}
              />
            </label>
            <label>
              <span>{t("imageSettings.route")}</span>
              <select
                value={draft.imagegenBridgeProvider}
                onChange={(event) =>
                  onImageChange({ imagegenBridgeProvider: event.target.value })
                }
              >
                <option value="codex-responses">
                  codex-responses · {t("imageSettings.recommended")}
                </option>
                <option value="codex-app-server">
                  codex-app-server · {t("imageSettings.fallback")}
                </option>
              </select>
            </label>
            <details className="settings-span-full">
              <summary>{t("imageSettingsExtra.bridgeRouting")}</summary>
              <div className="settings-grid">
                {draft.mapIconProvider === "codex-oauth" && (
                  <label>
                    <span>{t("imageAdvanced.mapRoute")}</span>
                    <select
                      value={draft.imagegenBridgeMapIconProvider}
                      onChange={(event) =>
                        onImageChange({
                          imagegenBridgeMapIconProvider: event.target.value,
                        })
                      }
                    >
                      <option value="codex-responses">codex-responses</option>
                      <option value="codex-app-server">codex-app-server</option>
                    </select>
                  </label>
                )}
                <label>
                  <span>{t("imageAdvanced.fallbacks")}</span>
                  <input
                    value={draft.imagegenBridgeFallbacks}
                    onChange={(event) =>
                      onImageChange({
                        imagegenBridgeFallbacks: event.target.value,
                      })
                    }
                    placeholder="codex-app-server:gpt-image-2"
                  />
                </label>
                <label>
                  <span>{t("models.fallbackPolicy")}</span>
                  <select
                    value={draft.imagegenBridgeFallbackPolicy}
                    onChange={(event) =>
                      onImageChange({
                        imagegenBridgeFallbackPolicy: event.target.value,
                      })
                    }
                  >
                    <option value="on_unavailable">
                      {t("models.onUnavailable")}
                    </option>
                    <option value="on_error">{t("models.onError")}</option>
                  </select>
                </label>
                <label>
                  <span>{t("models.compatibility")}</span>
                  <select
                    value={draft.imagegenBridgeCompatibility}
                    onChange={(event) =>
                      onImageChange({
                        imagegenBridgeCompatibility: event.target.value,
                      })
                    }
                  >
                    <option value="strict">{t("models.strict")}</option>
                    <option value="normalize">{t("models.normalize")}</option>
                    <option value="best_effort">
                      {t("models.bestEffort")}
                    </option>
                  </select>
                </label>
              </div>
            </details>
          </>
        ) : (
          <>
            <label>
              <span>{t("imageSettings.baseUrl")}</span>
              <input
                type="url"
                value={config?.baseUrl ?? ""}
                onChange={(event) =>
                  onProviderConfig(provider.id, { baseUrl: event.target.value })
                }
              />
            </label>
            {provider.id === "azure-openai" && (
              <label>
                <span>{t("imageSettings.apiVersion")}</span>
                <input
                  value={config?.apiVersion ?? ""}
                  onChange={(event) =>
                    onProviderConfig(provider.id, {
                      apiVersion: event.target.value,
                    })
                  }
                />
              </label>
            )}
            {provider.model_validation === "configured" && (
              <label>
                <span>{t("models.model")}</span>
                <input
                  value={config?.models ?? ""}
                  onChange={(event) =>
                    onProviderConfig(provider.id, {
                      models: event.target.value,
                    })
                  }
                />
              </label>
            )}
            <label>
              <span>{t("imageSettings.apiKey")}</span>
              <input
                type="password"
                autoComplete="new-password"
                value={config?.apiKey ?? ""}
                onChange={(event) =>
                  onProviderConfig(provider.id, {
                    apiKey: event.target.value,
                    clearApiKey: false,
                  })
                }
                placeholder={t("imageSettings.secretPlaceholder")}
              />
              <small>
                {provider.api_key_configured
                  ? t("imageSettings.keyConfigured")
                  : t("imageSettings.keyMissing")}
              </small>
            </label>
            <label className="toggle-row">
              <span>{t("imageSettings.clearSecret")}</span>
              <input
                type="checkbox"
                checked={config?.clearApiKey ?? false}
                onChange={(event) =>
                  onProviderConfig(provider.id, {
                    clearApiKey: event.target.checked,
                    apiKey: event.target.checked ? "" : config?.apiKey,
                  })
                }
              />
            </label>
          </>
        )}
      </div>
    </section>
  );
}
