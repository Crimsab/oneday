import { useTranslation } from "react-i18next";
import type { AppPreferences } from "../../types";
import { SettingsToggle } from "./GeneralSettings";

export function GameplaySettings({ preferences, onChange }: { preferences: AppPreferences; onChange: (preferences: AppPreferences) => void }) {
  const { t } = useTranslation(["drawer", "settings_ui"]);
  const update = <K extends keyof AppPreferences>(key: K, value: AppPreferences[K]) => onChange({ ...preferences, [key]: value });
  return (
    <div className="gameplay-settings-stack">
      <section className="settings-group" aria-labelledby="challenge-behavior-title">
        <header><div><h4 id="challenge-behavior-title">{t("settings_ui:gameplay.behaviorTitle")}</h4><p>{t("settings_ui:gameplay.behaviorDesc")}</p></div><span className="settings-apply-note">{t("settings_ui:gameplay.nextTurn")}</span></header>
        <div className="settings-toggle-list">
          <SettingsToggle id="automatic-challenges" label={t("drawer:gameplay.automatic")} description={t("drawer:gameplay.automaticDesc")} checked={preferences.automaticChallenges} onChange={(checked) => update("automaticChallenges", checked)} />
          <SettingsToggle id="timing-free" label={t("drawer:gameplay.timing")} description={t("drawer:gameplay.timingDesc")} checked={preferences.timingFreeChallenges} onChange={(checked) => update("timingFreeChallenges", checked)} />
          <SettingsToggle id="challenge-cooldown" label={t("drawer:gameplay.cooldown")} description={t("drawer:gameplay.cooldownDesc")} checked={preferences.challengeCooldown} onChange={(checked) => update("challengeCooldown", checked)} />
        </div>
      </section>
      <section className="settings-group" aria-labelledby="choice-presentation-title">
        <header><div><h4 id="choice-presentation-title">{t("settings_ui:gameplay.presentationTitle")}</h4><p>{t("settings_ui:gameplay.presentationDesc")}</p></div></header>
        <div className="settings-toggle-list">
          <SettingsToggle id="choice-details" label={t("drawer:gameplay.context")} description={t("drawer:gameplay.contextDesc")} checked={preferences.showChoiceDetails} onChange={(checked) => update("showChoiceDetails", checked)} />
        </div>
      </section>
    </div>
  );
}
