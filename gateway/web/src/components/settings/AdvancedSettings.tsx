import { Download, RefreshCw, RotateCcw, Upload } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { defaultPreferences, normalizePreferences } from "../../preferences";
import type { AppPreferences } from "../../types";
import { SettingsToggle } from "./GeneralSettings";

const MAX_PREFERENCES_BYTES = 1024 * 1024;

export function AdvancedSettings({
  preferences,
  hasSnapshot,
  busy,
  onChange,
  onReloadConfiguration,
}: {
  preferences: AppPreferences;
  hasSnapshot: boolean;
  busy: boolean;
  onChange: (preferences: AppPreferences) => void;
  onReloadConfiguration: () => Promise<void> | void;
}) {
  const { t } = useTranslation("settings_ui");
  const importRef = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState("");
  const [confirmReset, setConfirmReset] = useState(false);

  const exportPreferences = () => {
    const blob = new Blob([JSON.stringify({ version: 1, preferences }, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "oneday-preferences.json";
    link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 0);
    setStatus(t("advanced.exported"));
  };

  const importPreferences = async (file: File | undefined) => {
    if (!file) return;
    try {
      if (file.size > MAX_PREFERENCES_BYTES) throw new Error(t("advanced.importTooLarge"));
      const parsed: unknown = JSON.parse(await file.text());
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error(t("advanced.invalidImport"));
      const envelope = parsed as Record<string, unknown>;
      const candidate = "preferences" in envelope ? envelope.preferences : envelope;
      if (!candidate || typeof candidate !== "object" || Array.isArray(candidate)) throw new Error(t("advanced.invalidImport"));
      onChange(normalizePreferences(candidate as Partial<AppPreferences>));
      setStatus(t("advanced.imported"));
    } catch (cause) {
      setStatus(cause instanceof Error ? cause.message : String(cause));
    } finally {
      if (importRef.current) importRef.current.value = "";
    }
  };

  const reloadConfiguration = async () => {
    setStatus("");
    try {
      await onReloadConfiguration();
      setStatus(t("advanced.reloaded"));
    } catch (cause) {
      setStatus(cause instanceof Error ? cause.message : String(cause));
    }
  };

  const resetPreferences = () => {
    onChange({ ...defaultPreferences, locale: preferences.locale });
    setConfirmReset(false);
    setStatus(t("advanced.resetDone"));
  };

  return (
    <div className="advanced-settings-stack">
      <section className="settings-group" aria-labelledby="advanced-runtime-title" data-setting-id="runtime-status">
        <header><div><h4 id="advanced-runtime-title">{t("advanced.runtimeTitle")}</h4><p>{t("advanced.runtimeDesc")}</p></div><button type="button" data-setting-id="configuration-revision" disabled={busy} onClick={() => void reloadConfiguration()}><RefreshCw size={14} aria-hidden="true" /> {t("advanced.reload")}</button></header>
        <dl className="runtime-status-grid">
          <div><dt>{t("advanced.connection")}</dt><dd>{navigator.onLine ? t("advanced.online") : t("advanced.offline")}</dd></div>
          <div><dt>{t("advanced.liveUpdates")}</dt><dd>{hasSnapshot ? t("advanced.live") : t("advanced.noStory")}</dd></div>
          <div><dt>{t("advanced.transport")}</dt><dd>gateway-turn + SSE</dd></div>
          <div><dt>{t("advanced.capabilities")}</dt><dd>{t("advanced.capabilitiesValue")}</dd></div>
          <div><dt>{t("advanced.activeFont")}</dt><dd>{preferences.fontFamily}</dd></div>
          <div><dt>{t("advanced.activeAccent")}</dt><dd><span className="runtime-color-swatch" style={{ backgroundColor: preferences.accent }} aria-hidden="true" />{preferences.accent}</dd></div>
        </dl>
      </section>

      <section className="settings-group" aria-labelledby="advanced-diagnostics-title" data-setting-id="generation-diagnostics">
        <header><div><h4 id="advanced-diagnostics-title">{t("advanced.diagnosticsTitle")}</h4><p>{t("advanced.diagnosticsDesc")}</p></div></header>
        <div className="settings-toggle-list">
          <SettingsToggle id="generation-diagnostics-toggle" label={t("advanced.showDiagnostics")} description={t("advanced.showDiagnosticsDesc")} checked={preferences.showGenerationDiagnostics} onChange={(checked) => onChange({ ...preferences, showGenerationDiagnostics: checked })} />
        </div>
      </section>

      <section className="settings-group" aria-labelledby="advanced-portability-title" data-setting-id="preferences-portability">
        <header><div><h4 id="advanced-portability-title">{t("advanced.portabilityTitle")}</h4><p>{t("advanced.portabilityDesc")}</p></div></header>
        <div className="advanced-action-list">
          <div><span><strong>{t("advanced.exportTitle")}</strong><small>{t("advanced.exportDesc")}</small></span><button type="button" onClick={exportPreferences}><Download size={14} aria-hidden="true" /> {t("advanced.export")}</button></div>
          <div><span><strong>{t("advanced.importTitle")}</strong><small>{t("advanced.importDesc")}</small></span><button type="button" onClick={() => importRef.current?.click()}><Upload size={14} aria-hidden="true" /> {t("advanced.import")}</button><input ref={importRef} className="sr-only" type="file" accept="application/json,.json" onChange={(event) => void importPreferences(event.target.files?.[0])} /></div>
          <div className="advanced-reset-row"><span><strong>{t("advanced.resetTitle")}</strong><small>{t("advanced.resetDesc")}</small></span>{confirmReset ? <div className="advanced-reset-confirm"><button type="button" onClick={() => setConfirmReset(false)}>{t("common.cancel")}</button><button type="button" className="danger-action" onClick={resetPreferences}>{t("advanced.confirmReset")}</button></div> : <button type="button" onClick={() => setConfirmReset(true)}><RotateCcw size={14} aria-hidden="true" /> {t("advanced.reset")}</button>}</div>
        </div>
        <p className="advanced-status" role="status" aria-live="polite">{status}</p>
      </section>
    </div>
  );
}
