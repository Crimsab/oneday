import { Check } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { ImageGenerationDraft } from "../../modelRouting";
import type { ImageProviderCatalogEntry } from "../../types";
import type { ProviderConfigDraft } from "./imageGenerationDraft";
import { ProviderConfiguration } from "./ProviderConfiguration";
import { CustomSelect } from "../CustomSelect";
import { SettingsDisclosure } from "./SettingsDisclosure";

const customModelValue = "__oneday_custom_model__";
const customSizeValue = "__oneday_custom_size__";
const defaultSizePresets = ["1024x1024", "1536x1024", "1024x1536"];
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
  const otherProviders = catalog.filter((item) => item.id !== provider?.id);
  const modelList = (item?: ImageProviderCatalogEntry) => [...new Set([...(item?.models ?? []), ...(props.discoveredModels?.[item?.id ?? ""] ?? [])])];
  const preferredModel = (item?: ImageProviderCatalogEntry) => modelList(item)[0] ?? "";
  const aspectRatios = concreteValues(provider?.capabilities.aspect_ratios);
  const qualities = concreteValues(provider?.capabilities.qualities);
  const resolutions = resolutionOptionsFor(provider, draft.model);
  const providerChoice = (item: ImageProviderCatalogEntry, current = false) => {
    const selected = item.id === provider?.id;
    const nextModel = preferredModel(item);
    const mapFollowsPrimary = !draft.mapIconProvider || draft.mapIconProvider === provider?.id;
    return (
      <button
        key={item.id}
        type="button"
        role="radio"
        aria-checked={selected}
        className={`image-provider-choice${selected ? " active" : ""}${current ? " current" : ""}`}
        onClick={() => onImageChange(
          mapFollowsPrimary
            ? {
                provider: item.id,
                model: nextModel,
                mapIconProvider: item.id,
                mapIconModel: nextModel,
              }
            : { provider: item.id, model: nextModel },
        )}
      >
        <span className="image-provider-choice-copy">
          {current && <small className="image-provider-choice-label">{t("imageSettings.activeProvider")}</small>}
          <strong>{item.display_name}</strong>
          <small>
            {current
              ? item.configured
                ? t("imageSettings.configured")
                : t("imageSettings.notConfigured")
              : selected
                ? t("imageSettings.selected")
              : item.configured
                ? t("imageSettings.configured")
                : item.default
                  ? t("imageSettings.recommended")
                  : t("imageSettings.notConfigured")}
          </small>
        </span>
        {selected && <Check size={17} aria-hidden="true" />}
      </button>
    );
  };
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
        {provider && <div className="image-provider-current">{providerChoice(provider, true)}</div>}
        {otherProviders.length > 0 && (
          <SettingsDisclosure
            className="image-provider-more"
            title={t("imageSettings.changeProvider")}
            description={t("imageSettingsExtra.providerAlternativesHelp")}
            meta={t("imageSettingsExtra.providerCount", { count: otherProviders.length })}
          >
            <div className="image-provider-more-grid">
              {otherProviders.map((item) => providerChoice(item))}
            </div>
          </SettingsDisclosure>
        )}
      </div>
      <p className="image-provider-copy">
        {provider?.id === "codex-oauth"
          ? t("imageSettings.codexDescription")
          : t("imageSettings.vendorDescription", {
              provider: provider?.display_name,
            })}
      </p>
      <div className="settings-grid image-settings-grid">
        <label>
          <span>{t("imageSettings.sceneModel")}</span>
          <ImageModelPicker
            ariaLabel={t("imageSettings.sceneModel")}
            value={draft.model}
            models={modelList(provider)}
            customAllowed={provider?.model_validation !== "allowlist"}
            onChange={(model) => onImageChange({ model })}
          />
        </label>
        <label>
          <span>{t("imageSettings.mapProvider")}</span>
          <CustomSelect
            value={mapProvider?.id ?? ""}
            ariaLabel={t("imageSettings.mapProvider")}
            onChange={(providerId) => {
              const next = catalog.find(
                (item) => item.id === providerId,
              );
              onImageChange({
                mapIconProvider: providerId,
                mapIconModel: preferredModel(next),
              });
            }}
            options={catalog.map((item) => ({ value: item.id, label: item.display_name }))}
          />
        </label>
        <label>
          <span>{t("imageSettings.mapModel")}</span>
          <ImageModelPicker
            ariaLabel={t("imageSettings.mapModel")}
            value={draft.mapIconModel}
            models={modelList(mapProvider)}
            customAllowed={mapProvider?.model_validation !== "allowlist"}
            onChange={(mapIconModel) => onImageChange({ mapIconModel })}
          />
        </label>
        {provider?.capabilities.aspect_ratios.filter(isConcrete).length ? (
          <label>
            <span>{t("imageSettings.aspect")}</span>
            <CustomSelect
              value={draft.defaultAspectRatio}
              ariaLabel={t("imageSettings.aspect")}
              onChange={(defaultAspectRatio) => onImageChange({ defaultAspectRatio })}
              options={provider.capabilities.aspect_ratios.filter(isConcrete).map((value) => ({ value, label: value }))}
            />
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
      <section className="image-output-settings" aria-labelledby="image-output-settings-title">
        <header>
          <div>
            <h5 id="image-output-settings-title">{t("imageSettingsExtra.formatGroup")}</h5>
            <p>{t("imageSettingsExtra.formatHelp")}</p>
          </div>
          <span>{t("imageSettingsExtra.livePreview")}</span>
        </header>
        <div className="image-dimension-grid">
          <ImageSizePicker
            label={t("imageSettingsExtra.defaultFrame")}
            value={draft.defaultSize}
            presets={concreteValues(provider?.capabilities.sizes)}
            onChange={(defaultSize) => onImageChange({ defaultSize })}
          />
          <ImageSizePicker
            label={t("models.locationSize")}
            value={draft.locationSize}
            presets={concreteValues(provider?.capabilities.sizes)}
            onChange={(locationSize) => onImageChange({ locationSize })}
          />
          <ImageSizePicker
            label={t("models.characterSize")}
            value={draft.characterSize}
            presets={concreteValues(provider?.capabilities.sizes)}
            onChange={(characterSize) => onImageChange({ characterSize })}
          />
        </div>
        <label className="image-output-format">
          <span>{t("models.outputFormat")}</span>
          <CustomSelect
            value={draft.outputFormat}
            ariaLabel={t("models.outputFormat")}
            onChange={(outputFormat) => onImageChange({ outputFormat })}
            options={selectOptions(
              concreteValues(provider?.capabilities.output_formats).length
                ? concreteValues(provider?.capabilities.output_formats)
                : ["png"],
              draft.outputFormat,
              "PNG",
              false,
            ).map((option) => ({ ...option, label: option.label.toUpperCase() }))}
          />
        </label>
      </section>
      <div className="provider-configurations">
        {configuredProviders.map((item) => (
          <ProviderConfiguration key={item.id} provider={item} {...props} />
        ))}
      </div>
      <SettingsDisclosure
        className="image-provider-advanced"
        title={t("imageSettingsExtra.outputControls")}
        description={t("imageSettingsExtra.outputControlsHelp")}
      >
        <div className="settings-grid image-provider-advanced-grid">
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
          {resolutions.length ? (
            <label>
              <span>{t("imageAdvanced.defaultResolution")}</span>
              <CustomSelect
                value={draft.defaultResolution}
                ariaLabel={t("imageAdvanced.defaultResolution")}
                onChange={(defaultResolution) => onImageChange({ defaultResolution })}
                options={selectOptions(resolutions, draft.defaultResolution, t("imageSettingsExtra.automatic"))}
              />
            </label>
          ) : null}
          {resolutions.length ? (
            <label>
              <span>{t("imageAdvanced.locationResolution")}</span>
              <CustomSelect
                value={draft.locationResolution}
                ariaLabel={t("imageAdvanced.locationResolution")}
                onChange={(locationResolution) => onImageChange({ locationResolution })}
                options={selectOptions(resolutions, draft.locationResolution, t("imageSettingsExtra.automatic"))}
              />
            </label>
          ) : null}
          {resolutions.length ? (
            <label>
              <span>{t("imageAdvanced.characterResolution")}</span>
              <CustomSelect
                value={draft.characterResolution}
                ariaLabel={t("imageAdvanced.characterResolution")}
                onChange={(characterResolution) => onImageChange({ characterResolution })}
                options={selectOptions(resolutions, draft.characterResolution, t("imageSettingsExtra.automatic"))}
              />
            </label>
          ) : null}
          {aspectRatios.length ? (
            <label>
              <span>{t("imageAdvanced.locationAspect")}</span>
              <CustomSelect
                value={draft.locationAspectRatio}
                ariaLabel={t("imageAdvanced.locationAspect")}
                onChange={(locationAspectRatio) => onImageChange({ locationAspectRatio })}
                options={selectOptions(aspectRatios, draft.locationAspectRatio, t("imageSettingsExtra.automatic"))}
              />
            </label>
          ) : null}
          {aspectRatios.length ? (
            <label>
              <span>{t("imageAdvanced.characterAspect")}</span>
              <CustomSelect
                value={draft.characterAspectRatio}
                ariaLabel={t("imageAdvanced.characterAspect")}
                onChange={(characterAspectRatio) => onImageChange({ characterAspectRatio })}
                options={selectOptions(aspectRatios, draft.characterAspectRatio, t("imageSettingsExtra.automatic"))}
              />
            </label>
          ) : null}
          {qualities.length ? (
            <label>
              <span>{t("imageAdvanced.quality")}</span>
              <CustomSelect
                value={draft.quality}
                ariaLabel={t("imageAdvanced.quality")}
                onChange={(quality) => onImageChange({ quality })}
                options={selectOptions(qualities, draft.quality, t("imageSettingsExtra.automatic"))}
              />
            </label>
          ) : null}
          {provider?.capabilities.supports_transparency ? (
            <label>
              <span>{t("imageAdvanced.background")}</span>
              <CustomSelect
                value={draft.background}
                ariaLabel={t("imageAdvanced.background")}
                onChange={(background) => onImageChange({ background })}
                options={selectOptions(["opaque", "transparent"], draft.background, t("imageSettingsExtra.automatic"))}
              />
            </label>
          ) : null}
          <label className="toggle-row negative-prompt-toggle">
            <span>
              <strong>{t("models.appendNegative")}</strong>
              <small>{t("imageSettingsExtra.negativePromptHelp")}</small>
            </span>
            <input
              type="checkbox"
              checked={draft.appendNegativePrompt}
              onChange={(event) =>
                onImageChange({ appendNegativePrompt: event.target.checked })
              }
            />
          </label>
        </div>
      </SettingsDisclosure>
      <p className="model-note">{t("imageSettings.noConnectionTest")}</p>
    </section>
  );
}

function isConcrete(value: string) {
  return !value.endsWith("-defined");
}

function concreteValues(values?: string[]) {
  return [...new Set((values ?? []).filter(isConcrete))];
}

function resolutionOptionsFor(provider: ImageProviderCatalogEntry | undefined, model: string) {
  if (provider?.id !== "gemini") return [];
  const normalized = model.trim().toLowerCase();
  if (normalized.includes("3.1-flash-lite-image")) return ["1K"];
  if (normalized.includes("3.1-flash-image")) return ["0.5K", "1K", "2K", "4K"];
  if (normalized.includes("gemini-3") && normalized.includes("image")) {
    return ["1K", "2K", "4K"];
  }
  return [];
}

function selectOptions(values: string[], value: string, automaticLabel: string, includeAutomatic = true) {
  const options = [
    ...(includeAutomatic ? [{ value: "", label: automaticLabel }] : []),
    ...values.map((candidate) => ({ value: candidate, label: candidate })),
  ];
  if (value && !options.some((option) => option.value === value)) {
    options.splice(1, 0, { value, label: value });
  }
  return options;
}

function ImageSizePicker({ label, value, presets, onChange }: {
  label: string;
  value: string;
  presets: string[];
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation("drawer");
  const options = [...new Set([...(presets.length ? presets : defaultSizePresets), ...defaultSizePresets])];
  const custom = !options.includes(value);
  const [width, height] = imageDimensions(value);
  return (
    <div className="image-dimension-control">
      <label>
        <span>{label}</span>
        <CustomSelect
          value={custom ? customSizeValue : value}
          ariaLabel={label}
          onChange={(next) => onChange(next === customSizeValue ? "" : next)}
          options={[
            ...options.map((candidate) => ({ value: candidate, label: candidate })),
            { value: customSizeValue, label: t("imageSettingsExtra.customSize") },
          ]}
        />
      </label>
      {custom ? (
        <label className="image-custom-size">
          <span>{t("imageSettingsExtra.customSizeLabel")}</span>
          <input
            value={value}
            inputMode="numeric"
            onChange={(event) => onChange(event.target.value)}
            placeholder="1280x720"
            aria-label={t("imageSettingsExtra.customSizeFor", { field: label })}
          />
        </label>
      ) : null}
      <div className="image-size-preview" aria-label={t("imageSettingsExtra.previewFor", { field: label, size: value || t("imageSettingsExtra.customSize") })}>
        <span style={{ aspectRatio: `${width} / ${height}` }} aria-hidden="true" />
        <small>{value || t("imageSettingsExtra.enterCustomSize")}</small>
      </div>
    </div>
  );
}

function imageDimensions(value: string): [number, number] {
  const match = value.trim().match(/^(\d+)\s*[x×]\s*(\d+)$/i);
  const width = Number(match?.[1]);
  const height = Number(match?.[2]);
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return [1, 1];
  return [width, height];
}

function ImageModelPicker({ ariaLabel, value, models, customAllowed, onChange }: {
  ariaLabel: string;
  value: string;
  models: string[];
  customAllowed: boolean;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation("drawer");
  const options = [...new Set(models.filter(Boolean))];
  const custom = customAllowed && (!value || !options.includes(value));

  if (options.length === 0) {
    return <input value={value} aria-label={ariaLabel} onChange={(event) => onChange(event.target.value)} placeholder={t("imageSettings.customModelPlaceholder")} />;
  }
  if (options.length === 1 && !customAllowed) {
    return <output className="image-model-fixed" aria-label={ariaLabel}><strong>{options[0]}</strong><small>{t("imageSettings.onlySupportedModel")}</small></output>;
  }

  return <div className="image-model-picker">
    <CustomSelect
      value={custom ? customModelValue : value || options[0]}
      ariaLabel={ariaLabel}
      onChange={(next) => onChange(next === customModelValue ? "" : next)}
      options={[
        ...options.map((model) => ({ value: model, label: model })),
        ...(customAllowed ? [{ value: customModelValue, label: t("imageSettings.customModel") }] : []),
      ]}
    />
    {custom && <input value={value} aria-label={t("imageSettings.customModelId", { field: ariaLabel })} onChange={(event) => onChange(event.target.value)} placeholder={t("imageSettings.customModelPlaceholder")} />}
  </div>;
}
