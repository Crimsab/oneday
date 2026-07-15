import { BadgeDollarSign, Binary, BrainCircuit, MessageCircle, Scale, Theater, Timer } from "lucide-react";
import { useState, type ComponentType } from "react";
import { useTranslation } from "react-i18next";
import { AUTOMATIC_MINI_GAME_KINDS } from "../../preferences";
import type { AppPreferences, MiniGameKind } from "../../types";
import { SettingsToggle } from "./GeneralSettings";

const miniGameIcons: Record<(typeof AUTOMATIC_MINI_GAME_KINDS)[number], ComponentType<{ size?: number; "aria-hidden"?: boolean }>> = {
  deduction: BrainCircuit,
  negotiation: MessageCircle,
  pattern: Binary,
  bidding: BadgeDollarSign,
  courtroom: Scale,
  comedy: Theater,
  quicktime: Timer,
};

export function GameplaySettings({ preferences, onChange }: { preferences: AppPreferences; onChange: (preferences: AppPreferences) => void }) {
  const { t } = useTranslation(["drawer", "settings_ui"]);
  const [catalogStatus, setCatalogStatus] = useState("");
  const update = <K extends keyof AppPreferences>(key: K, value: AppPreferences[K]) => onChange({ ...preferences, [key]: value });
  const disabled = new Set(preferences.disabledMiniGames);
  const enabledCount = AUTOMATIC_MINI_GAME_KINDS.length - disabled.size;
  const enabledTimingFreeCount = AUTOMATIC_MINI_GAME_KINDS.filter((kind) => kind !== "quicktime" && !disabled.has(kind)).length;
  const toggleMiniGame = (kind: MiniGameKind) => {
    const currentlyEnabled = !disabled.has(kind);
    if (currentlyEnabled && (enabledCount === 1 || (preferences.timingFreeChallenges && kind !== "quicktime" && enabledTimingFreeCount === 1))) {
      setCatalogStatus(t("settings_ui:gameplay.keepOne"));
      return;
    }
    setCatalogStatus("");
    const next = currentlyEnabled
      ? [...preferences.disabledMiniGames, kind]
      : preferences.disabledMiniGames.filter((item) => item !== kind);
    update("disabledMiniGames", next);
  };
  const setTimingFree = (checked: boolean) => {
    const mustRestoreFallback = checked && enabledTimingFreeCount === 0;
    onChange({
      ...preferences,
      timingFreeChallenges: checked,
      disabledMiniGames: mustRestoreFallback ? preferences.disabledMiniGames.filter((kind) => kind !== "deduction") : preferences.disabledMiniGames,
    });
    setCatalogStatus("");
  };
  return (
    <div className="gameplay-settings-stack">
      <section className="settings-group" aria-labelledby="challenge-behavior-title">
        <header><div><h4 id="challenge-behavior-title">{t("settings_ui:gameplay.behaviorTitle")}</h4><p>{t("settings_ui:gameplay.behaviorDesc")}</p></div><span className="settings-apply-note">{t("settings_ui:gameplay.nextTurn")}</span></header>
        <div className="settings-toggle-list">
          <SettingsToggle id="automatic-challenges" label={t("drawer:gameplay.automatic")} description={t("drawer:gameplay.automaticDesc")} checked={preferences.automaticChallenges} onChange={(checked) => update("automaticChallenges", checked)} />
          <SettingsToggle id="timing-free" label={t("drawer:gameplay.timing")} description={t("drawer:gameplay.timingDesc")} checked={preferences.timingFreeChallenges} onChange={setTimingFree} />
          <SettingsToggle id="challenge-cooldown" label={t("drawer:gameplay.cooldown")} description={t("drawer:gameplay.cooldownDesc")} checked={preferences.challengeCooldown} onChange={(checked) => update("challengeCooldown", checked)} />
        </div>
      </section>
      <section className="settings-group" aria-labelledby="minigame-catalog-title" data-setting-id="minigame-catalog">
        <header><div><h4 id="minigame-catalog-title">{t("settings_ui:gameplay.catalogTitle")}</h4><p>{t("settings_ui:gameplay.catalogDesc")}</p></div><span className="settings-apply-note">{t("settings_ui:gameplay.catalogCount", { enabled: enabledCount, total: AUTOMATIC_MINI_GAME_KINDS.length })}</span></header>
        <div className="minigame-preference-grid">
          {AUTOMATIC_MINI_GAME_KINDS.map((kind) => {
            const Icon = miniGameIcons[kind];
            const enabled = !disabled.has(kind);
            const timingExcluded = kind === "quicktime" && preferences.timingFreeChallenges;
            return <label className={`minigame-preference ${enabled ? "enabled" : "disabled"}`} key={kind}>
              <span className="minigame-preference-icon" aria-hidden="true"><Icon size={17} /></span>
              <span className="minigame-preference-copy"><strong>{t(`settings_ui:gameplay.miniGames.${kind}.name`)}</strong><small>{t(`settings_ui:gameplay.miniGames.${kind}.desc`)}</small>{timingExcluded && <em>{t("settings_ui:gameplay.reflexWarning")}</em>}</span>
              <span className="switch-control"><input type="checkbox" checked={enabled} onChange={() => toggleMiniGame(kind)} /><span aria-hidden="true" /></span>
            </label>;
          })}
        </div>
        <p className="minigame-catalog-status" role="status" aria-live="polite">{catalogStatus}</p>
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
