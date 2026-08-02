import type { AppPreferences } from "../../types";
import { DEFAULT_ACCENT } from "../../preferences";
import { CustomSelect } from "../CustomSelect";
import { DeferredColorPicker } from "./DeferredColorPicker";
import { useTranslation } from "react-i18next";
import { useMemo } from "react";
import { languageCatalog } from "../../features/translation/languageCatalog";

export function GeneralSettings({
  preferences,
  onChange,
}: {
  preferences: AppPreferences;
  onChange: (preferences: AppPreferences) => void;
}) {
  const { t } = useTranslation(["options", "settings_ui"]);
  const storyLanguages = useMemo(
    () =>
      languageCatalog(preferences.locale).map((language) => ({
        value: language.code,
        label: language.name,
      })),
    [preferences.locale],
  );
  const update = <K extends keyof AppPreferences>(
    key: K,
    value: AppPreferences[K],
  ) => onChange({ ...preferences, [key]: value });
  const applyAccent = (accent: string) => {
    const accentHistory = [
      accent,
      preferences.accent,
      ...preferences.accentHistory,
    ]
      .filter((color, index, values) => values.indexOf(color) === index)
      .slice(0, 10);
    onChange({ ...preferences, accent, accentHistory });
  };

  return (
    <div className="general-settings-stack">
      <section
        className="settings-group"
        aria-labelledby="general-interface-title"
      >
        <header>
          <div>
            <h4 id="general-interface-title">
              {t("settings_ui:general.interfaceTitle")}
            </h4>
            <p>{t("settings_ui:general.interfaceDesc")}</p>
          </div>
        </header>
        <div className="settings-field-grid">
          <div
            className="settings-field settings-span-full"
            data-setting-id="interface-language"
          >
            <span>{t("options:interfaceLanguage")}</span>
            <CustomSelect
              value={preferences.locale}
              ariaLabel={t("options:interfaceLanguage")}
              onChange={(value) =>
                update("locale", value as AppPreferences["locale"])
              }
              options={[
                { value: "en", label: t("options:english") },
                { value: "it", label: t("options:italian") },
              ]}
            />
            <small>{t("options:languageHint")}</small>
          </div>
          <div
            className="settings-field settings-span-full"
            data-setting-id="story-language"
          >
            <span>{t("options:defaultStoryLanguage")}</span>
            <CustomSelect
              value={preferences.defaultStoryLanguage}
              ariaLabel={t("options:defaultStoryLanguage")}
              onChange={(value) => update("defaultStoryLanguage", value)}
              options={storyLanguages}
            />
            <small>{t("options:defaultStoryLanguageHint")}</small>
          </div>
          <div className="settings-field" data-setting-id="density">
            <span>{t("options:density")}</span>
            <CustomSelect
              value={preferences.density}
              ariaLabel={t("options:density")}
              onChange={(value) =>
                update("density", value as AppPreferences["density"])
              }
              options={[
                { value: "compact", label: t("options:compact") },
                { value: "balanced", label: t("options:balanced") },
                { value: "comfortable", label: t("options:comfortable") },
              ]}
            />
          </div>
        </div>
      </section>

      <section
        className="settings-group"
        aria-labelledby="general-color-title"
        data-setting-id="accent"
      >
        <header>
          <div>
            <h4 id="general-color-title">
              {t("settings_ui:general.colorTitle")}
            </h4>
            <p>{t("settings_ui:general.colorDesc")}</p>
          </div>
        </header>
        <DeferredColorPicker
          value={preferences.accent}
          defaultValue={DEFAULT_ACCENT}
          history={preferences.accentHistory}
          label={t("options:accent")}
          description={t("settings_ui:color.triggerHint")}
          onApply={applyAccent}
        />
      </section>

      <section
        className="settings-group"
        aria-labelledby="general-layout-title"
      >
        <header>
          <div>
            <h4 id="general-layout-title">
              {t("settings_ui:general.layoutTitle")}
            </h4>
            <p>{t("settings_ui:general.layoutDesc")}</p>
          </div>
        </header>
        <div className="settings-toggle-list">
          <SettingsToggle
            id="stories-sidebar"
            label={t("options:storiesSidebar")}
            description={t("settings_ui:general.storiesSidebarDesc")}
            checked={preferences.showLeftRail}
            onChange={(checked) => update("showLeftRail", checked)}
          />
          <SettingsToggle
            id="inspector"
            label={t("options:inspector")}
            description={t("settings_ui:general.inspectorDesc")}
            checked={preferences.showInspector}
            onChange={(checked) => update("showInspector", checked)}
          />
          <SettingsToggle
            id="transcript-wrap"
            label={t("options:wrap")}
            description={t("settings_ui:general.wrapDesc")}
            checked={preferences.wrapTranscript}
            onChange={(checked) => update("wrapTranscript", checked)}
          />
        </div>
      </section>
    </div>
  );
}

export function SettingsToggle({
  id,
  label,
  description,
  checked,
  onChange,
}: {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="settings-toggle" data-setting-id={id}>
      <span className="settings-toggle-copy">
        <strong>{label}</strong>
        <small>{description}</small>
      </span>
      <span className="switch-control">
        <input
          type="checkbox"
          checked={checked}
          onChange={(event) => onChange(event.target.checked)}
        />
        <span aria-hidden="true" />
      </span>
    </label>
  );
}
