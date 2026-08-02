import { useTranslation } from "react-i18next";
import type { ImageGenerationDraft } from "../../modelRouting";
import type { ImageProviderCatalogEntry } from "../../types";
import type { ProviderConfigDraft } from "./imageGenerationDraft";
import { CustomSelect } from "../CustomSelect";
import { SettingsDisclosure } from "./SettingsDisclosure";

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
            <SettingsDisclosure
              className="provider-connection-details settings-span-full"
              title={t("imageSettingsExtra.connection")}
              description={t("imageSettingsExtra.connectionHelp")}
            >
              <div className="settings-grid">
                <label>
                  <span>{t("imageSettings.bridgeUrl")}</span>
                  <input type="url" value={draft.imagegenBridgeUrl} onChange={(event) => onImageChange({ imagegenBridgeUrl: event.target.value })} />
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
                {provider.configured && (
                  <div className={`saved-secret-action ${clearBridgeToken ? "pending" : ""}`}>
                    <span>
                      <strong>{clearBridgeToken ? t("imageSettings.clearSecretPending") : t("imageSettings.keyConfigured")}</strong>
                      <small>{t("imageSettings.savedSecretHelp")}</small>
                    </span>
                    <button
                      type="button"
                      aria-pressed={clearBridgeToken}
                      onClick={() => {
                        onClearBridgeToken(!clearBridgeToken);
                        if (!clearBridgeToken) onBridgeToken("");
                      }}
                    >
                      {clearBridgeToken ? t("imageSettings.keepSecret") : t("imageSettings.clearSecretShort")}
                    </button>
                  </div>
                )}
              </div>
            </SettingsDisclosure>
            <label>
              <span>{t("imageSettings.route")}</span>
              <CustomSelect
                value={draft.imagegenBridgeProvider}
                ariaLabel={t("imageSettings.route")}
                onChange={(imagegenBridgeProvider) => onImageChange({ imagegenBridgeProvider })}
                options={[
                  { value: "codex-responses", label: `${t("imageSettings.routeRecommended")} · codex-responses` },
                  { value: "codex-app-server", label: `${t("imageSettings.routeFallback")} · codex-app-server` },
                ]}
              />
            </label>
            <SettingsDisclosure
              className="provider-connection-details settings-span-full"
              title={t("imageSettingsExtra.bridgeRouting")}
              description={t("imageSettingsExtra.bridgeRoutingHelp")}
            >
              <div className="settings-grid">
                {draft.mapIconProvider === "codex-oauth" && (
                  <label>
                    <span>{t("imageAdvanced.mapRoute")}</span>
                    <CustomSelect
                      value={draft.imagegenBridgeMapIconProvider}
                      ariaLabel={t("imageAdvanced.mapRoute")}
                      onChange={(imagegenBridgeMapIconProvider) => onImageChange({ imagegenBridgeMapIconProvider })}
                      options={[
                        { value: "codex-responses", label: "codex-responses" },
                        { value: "codex-app-server", label: "codex-app-server" },
                      ]}
                    />
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
                  <CustomSelect
                    value={draft.imagegenBridgeFallbackPolicy}
                    ariaLabel={t("models.fallbackPolicy")}
                    onChange={(imagegenBridgeFallbackPolicy) => onImageChange({ imagegenBridgeFallbackPolicy })}
                    options={[
                      { value: "on_unavailable", label: t("models.onUnavailable") },
                      { value: "on_error", label: t("models.onError") },
                    ]}
                  />
                </label>
                <label>
                  <span>{t("models.compatibility")}</span>
                  <CustomSelect
                    value={draft.imagegenBridgeCompatibility}
                    ariaLabel={t("models.compatibility")}
                    onChange={(imagegenBridgeCompatibility) => onImageChange({ imagegenBridgeCompatibility })}
                    options={[
                      { value: "strict", label: t("models.strict") },
                      { value: "normalize", label: t("models.normalize") },
                      { value: "best_effort", label: t("models.bestEffort") },
                    ]}
                  />
                </label>
              </div>
            </SettingsDisclosure>
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
