import { useTranslation } from "react-i18next";
import type { ImageGenerationDraft } from "../../modelRouting";
import type { ImageProviderCatalogEntry } from "../../types";
import type { ProviderConfigDraft } from "./imageGenerationDraft";
import { ProviderConfiguration } from "./ProviderConfiguration";
interface Props {
  catalog: ImageProviderCatalogEntry[];
  draft: ImageGenerationDraft;
  providerConfigs: Record<string, ProviderConfigDraft>;
  bridgeToken: string;
  clearBridgeToken: boolean;
  discoveredModels?: Record<string, string[]>;
  onImageChange: (patch: Partial<ImageGenerationDraft>) => void;
  onProviderConfig: (id: string, patch: Partial<ProviderConfigDraft>) => void;
  onBridgeToken: (value: string) => void;
  onClearBridgeToken: (value: boolean) => void;
}

export function ImageGenerationSettings(props: Props) {
  const { t } = useTranslation("drawer");
  const { catalog, draft, onImageChange } = props;
  const provider =
    catalog.find((item) => item.id === draft.provider) ?? catalog[0];
  const mapProvider =
    catalog.find((item) => item.id === draft.mapIconProvider) ?? provider;
  const configuredProviders = [provider, mapProvider].filter(
    (item, index, all): item is ImageProviderCatalogEntry =>
      Boolean(item) &&
      all.findIndex((other) => other?.id === item?.id) === index,
  );
  const modelList = (item?: ImageProviderCatalogEntry) => [...new Set([...(item?.models ?? []), ...(props.discoveredModels?.[item?.id ?? ""] ?? [])])];
  return (
    <section
      className="image-provider-settings settings-span-full"
      aria-labelledby="image-provider-title"
    >
      <div className="settings-section-head">
        <div>
          <h4 id="image-provider-title">{t("imageSettings.title")}</h4>
          <p>{t("imageSettings.description")}</p>
        </div>
        <span
          className={`provider-state ${provider?.configured ? "ready" : "blocked"}`}
        >
          {provider?.configured
            ? t("imageSettings.configured")
            : t("imageSettings.notConfigured")}
        </span>
      </div>
      <div
        className="image-provider-picker"
        role="radiogroup"
        aria-label={t("imageSettings.provider")}
      >
        {catalog.map((item) => (
          <button
            key={item.id}
            type="button"
            role="radio"
            aria-checked={item.id === provider?.id}
            className={item.id === provider?.id ? "active" : ""}
            onClick={() =>
              onImageChange({
                provider: item.id,
                model: item.models[0] ?? draft.model,
              })
            }
          >
            <strong>{item.display_name}</strong>
            <small>
              {item.default
                ? t("imageSettings.recommended")
                : item.configured
                  ? t("imageSettings.configured")
                  : t("imageSettings.notConfigured")}
            </small>
          </button>
        ))}
      </div>
      <p className="image-provider-copy">
        {provider?.id === "codex-oauth"
          ? t("imageSettings.codexDescription")
          : t("imageSettings.vendorDescription", {
              provider: provider?.display_name,
            })}
      </p>
      <div className="settings-grid image-settings-grid">
        <datalist id="scene-image-models">
          {modelList(provider).map((model) => (
            <option key={model} value={model} />
          ))}
        </datalist>
        <datalist id="map-image-models">
          {modelList(mapProvider).map((model) => (
            <option key={model} value={model} />
          ))}
        </datalist>
        <label>
          <span>{t("imageSettings.sceneModel")}</span>
          <input
            list="scene-image-models"
            value={draft.model}
            onChange={(event) => onImageChange({ model: event.target.value })}
          />
        </label>
        <label>
          <span>{t("imageSettings.mapProvider")}</span>
          <select
            value={mapProvider?.id ?? ""}
            onChange={(event) => {
              const next = catalog.find(
                (item) => item.id === event.target.value,
              );
              onImageChange({
                mapIconProvider: event.target.value,
                mapIconModel: next?.models[0] ?? draft.mapIconModel,
              });
            }}
          >
            {catalog.map((item) => (
              <option key={item.id} value={item.id}>
                {item.display_name}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>{t("imageSettings.mapModel")}</span>
          <input
            list="map-image-models"
            value={draft.mapIconModel}
            onChange={(event) =>
              onImageChange({ mapIconModel: event.target.value })
            }
          />
        </label>
        {provider?.capabilities.sizes.filter(isConcrete).length ? (
          <label>
            <span>{t("imageSettings.size")}</span>
            <select
              value={draft.defaultSize}
              onChange={(event) =>
                onImageChange({ defaultSize: event.target.value })
              }
            >
              {provider.capabilities.sizes.filter(isConcrete).map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
          </label>
        ) : null}
        {provider?.capabilities.aspect_ratios.filter(isConcrete).length ? (
          <label>
            <span>{t("imageSettings.aspect")}</span>
            <select
              value={draft.defaultAspectRatio}
              onChange={(event) =>
                onImageChange({ defaultAspectRatio: event.target.value })
              }
            >
              {provider.capabilities.aspect_ratios
                .filter(isConcrete)
                .map((value) => (
                  <option key={value}>{value}</option>
                ))}
            </select>
          </label>
        ) : null}
        <label className="toggle-row">
          <span>{t("models.auto")}</span>
          <input
            type="checkbox"
            checked={draft.autoGenerate}
            onChange={(event) =>
              onImageChange({ autoGenerate: event.target.checked })
            }
          />
        </label>
      </div>
      <div className="provider-configurations">
        {configuredProviders.map((item) => (
          <ProviderConfiguration key={item.id} provider={item} {...props} />
        ))}
      </div>
      <details className="image-provider-advanced">
        <summary>{t("imageSettingsExtra.outputControls")}</summary>
        <div className="settings-grid">
          <label>
            <span>{t("models.timeout")}</span>
            <input
              type="number"
              min={1}
              value={draft.timeoutSeconds}
              onChange={(event) =>
                onImageChange({ timeoutSeconds: Number(event.target.value) })
              }
            />
          </label>
          <label>
            <span>{t("imageAdvanced.defaultResolution")}</span>
            <input
              value={draft.defaultResolution}
              onChange={(event) =>
                onImageChange({ defaultResolution: event.target.value })
              }
            />
          </label>
          <label>
            <span>{t("imageAdvanced.locationResolution")}</span>
            <input
              value={draft.locationResolution}
              onChange={(event) =>
                onImageChange({ locationResolution: event.target.value })
              }
            />
          </label>
          <label>
            <span>{t("imageAdvanced.characterResolution")}</span>
            <input
              value={draft.characterResolution}
              onChange={(event) =>
                onImageChange({ characterResolution: event.target.value })
              }
            />
          </label>
          <label>
            <span>{t("imageAdvanced.locationAspect")}</span>
            <input
              value={draft.locationAspectRatio}
              onChange={(event) =>
                onImageChange({ locationAspectRatio: event.target.value })
              }
            />
          </label>
          <label>
            <span>{t("imageAdvanced.characterAspect")}</span>
            <input
              value={draft.characterAspectRatio}
              onChange={(event) =>
                onImageChange({ characterAspectRatio: event.target.value })
              }
            />
          </label>
          <label>
            <span>{t("imageAdvanced.quality")}</span>
            <input
              value={draft.quality}
              onChange={(event) =>
                onImageChange({ quality: event.target.value })
              }
            />
          </label>
          <label>
            <span>{t("imageAdvanced.background")}</span>
            <input
              value={draft.background}
              onChange={(event) =>
                onImageChange({ background: event.target.value })
              }
            />
          </label>
          <label className="toggle-row">
            <span>{t("models.appendNegative")}</span>
            <input
              type="checkbox"
              checked={draft.appendNegativePrompt}
              onChange={(event) =>
                onImageChange({ appendNegativePrompt: event.target.checked })
              }
            />
          </label>
        </div>
      </details>
      <p className="model-note">{t("imageSettings.noConnectionTest")}</p>
    </section>
  );
}

function isConcrete(value: string) {
  return !value.endsWith("-defined");
}
