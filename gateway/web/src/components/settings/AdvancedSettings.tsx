import { Clipboard, Copy, Download, FileJson, RefreshCw, RotateCcw, Trash2, Upload } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { defaultPreferences, normalizePreferences } from "../../preferences";
import { commitThemeFontImport, listStoredFonts, rollbackThemeFontImport, stageThemeFonts, type StoredFontRecord } from "../../fontLibrary";
import { exportTheme, previewThemeImport, type ThemePreview } from "../../features/themes/themePortability";
import { exportThemeBundle, previewThemeBundle, type ThemeBundlePreview } from "../../features/themes/themeBundle";
import { buildSupportBundle, clearSupportEvents, getSupportEvents, type SupportEvent } from "../../supportDiagnostics";
import type { AppPreferences, ModelSettings, StorySnapshot } from "../../types";
import { SettingsToggle } from "./GeneralSettings";

const MAX_PREFERENCES_BYTES = 1024 * 1024;

export function AdvancedSettings({
  scope,
  preferences,
  snapshot,
  modelSettings,
  busy,
  onChange,
  onReloadConfiguration,
}: {
  scope: "player" | "operator";
  preferences: AppPreferences;
  snapshot: StorySnapshot | null;
  modelSettings: ModelSettings | null;
  busy: boolean;
  onChange: (preferences: AppPreferences) => void;
  onReloadConfiguration: () => Promise<void> | void;
}) {
  const { t } = useTranslation("settings_ui");
  const importRef = useRef<HTMLInputElement>(null);
  const themeImportRef = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState("");
  const [themePreview, setThemePreview] = useState<ThemePreview | null>(null);
  const [themeUndo, setThemeUndo] = useState<AppPreferences | null>(null);
  const [themeUndoFonts, setThemeUndoFonts] = useState<StoredFontRecord[]>([]);
  const [includeThemeFonts, setIncludeThemeFonts] = useState(false);
  const [confirmReset, setConfirmReset] = useState(false);
  const [logs, setLogs] = useState<SupportEvent[]>(() => getSupportEvents());

  const exportPreferences = () => {
    downloadJson({ version: 2, preferences }, "oneday-preferences.json");
    setStatus(t("advanced.exported"));
  };

  const copyPreferences = async () => {
    await copyText(JSON.stringify({ version: 2, preferences }, null, 2));
    setStatus(t("advanced.settingsCopied"));
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

  const exportPortableTheme = () => {
    downloadJson(exportTheme(preferences), "oneday-theme.json");
    setStatus(t("advanced.exported"));
  };

  const exportPortableThemeBundle = async () => {
    try {
      downloadBlob(await exportThemeBundle(preferences, await listStoredFonts(), includeThemeFonts), "oneday-theme.zip");
      setStatus(t("advanced.exported"));
    } catch (cause) { setStatus(cause instanceof Error ? cause.message : String(cause)); }
  };

  const previewPortableTheme = async (file: File | undefined) => {
    if (!file) return;
    try {
      const storedFonts = await listStoredFonts();
      const available = new Set(storedFonts.map((font) => font.id));
      if (file.name.toLowerCase().endsWith(".zip") || file.type === "application/zip") setThemePreview(await previewThemeBundle(file, preferences, available));
      else {
        if (file.size > MAX_PREFERENCES_BYTES) throw new Error(t("advanced.importTooLarge"));
        setThemePreview(previewThemeImport(JSON.parse(await file.text()), preferences, available));
      }
      setStatus("");
    } catch {
      setThemePreview(null);
      setStatus(t("advanced.invalidImport"));
    } finally {
      if (themeImportRef.current) themeImportRef.current.value = "";
    }
  };

  const applyPortableTheme = async () => {
    if (!themePreview) return;
    const stagedFonts = "stagedFonts" in themePreview ? (themePreview as ThemeBundlePreview).stagedFonts : [];
    try {
      if (stagedFonts.length) await stageThemeFonts(stagedFonts);
      setThemeUndo(preferences); setThemeUndoFonts(stagedFonts); onChange(themePreview.preferences);
      if (stagedFonts.length) await commitThemeFontImport();
      setThemePreview(null); setStatus(t("advanced.imported"));
    } catch (cause) {
      if (stagedFonts.length) await rollbackThemeFontImport(stagedFonts);
      setStatus(cause instanceof Error ? cause.message : String(cause));
    }
  };

  const undoPortableTheme = async () => {
    if (!themeUndo) return;
    onChange(themeUndo);
    if (themeUndoFonts.length) await rollbackThemeFontImport(themeUndoFonts);
    setThemeUndo(null);
    setThemeUndoFonts([]);
    setStatus(t("advanced.resetDone"));
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

  const supportBundle = () => buildSupportBundle({ preferences, snapshot, modelSettings });
  const copySupportBundle = async () => {
    setLogs(getSupportEvents());
    await copyText(JSON.stringify(supportBundle(), null, 2));
    setStatus(t("advanced.supportCopied"));
  };
  const downloadSupportBundle = () => {
    setLogs(getSupportEvents());
    downloadJson(supportBundle(), `oneday-support-${new Date().toISOString().slice(0, 10)}.json`);
    setStatus(t("advanced.supportDownloaded"));
  };
  const copyLogs = async () => {
    const current = getSupportEvents();
    setLogs(current);
    await copyText(JSON.stringify(current, null, 2));
    setStatus(t("advanced.logsCopied"));
  };
  const clearLogs = () => {
    clearSupportEvents();
    setLogs(getSupportEvents());
    setStatus(t("advanced.logsCleared"));
  };

  return (
    <div className="advanced-settings-stack">
      {scope === "operator" && <section className="settings-group" aria-labelledby="advanced-runtime-title" data-setting-id="runtime-status">
        <header><div><h4 id="advanced-runtime-title">{t("advanced.runtimeTitle")}</h4><p>{t("advanced.runtimeDesc")}</p></div><button type="button" data-setting-id="configuration-revision" disabled={busy} onClick={() => void reloadConfiguration()}><RefreshCw size={14} aria-hidden="true" /> {t("advanced.reload")}</button></header>
        <dl className="runtime-status-grid">
          <div><dt>{t("advanced.connection")}</dt><dd>{navigator.onLine ? t("advanced.online") : t("advanced.offline")}</dd></div>
          <div><dt>{t("advanced.liveUpdates")}</dt><dd>{snapshot ? t("advanced.live") : t("advanced.noStory")}</dd></div>
          <div><dt>{t("advanced.transport")}</dt><dd>gateway-turn + SSE</dd></div>
          <div><dt>{t("advanced.capabilities")}</dt><dd>{t("advanced.capabilitiesValue")}</dd></div>
          <div><dt>{t("advanced.activeFont")}</dt><dd>{preferences.interfaceFontFamily} / {preferences.readingFontFamily}</dd></div>
          <div><dt>{t("advanced.activeAccent")}</dt><dd><span className="runtime-color-swatch" style={{ backgroundColor: preferences.accent }} aria-hidden="true" />{preferences.accent}</dd></div>
        </dl>
      </section>}

      {scope === "player" && <section className="settings-group" aria-labelledby="advanced-diagnostics-title" data-setting-id="generation-diagnostics">
        <header><div><h4 id="advanced-diagnostics-title">{t("advanced.diagnosticsTitle")}</h4><p>{t("advanced.diagnosticsDesc")}</p></div></header>
        <div className="settings-toggle-list">
          <SettingsToggle id="generation-diagnostics-toggle" label={t("advanced.showDiagnostics")} description={t("advanced.showDiagnosticsDesc")} checked={preferences.showGenerationDiagnostics} onChange={(checked) => onChange({ ...preferences, showGenerationDiagnostics: checked })} />
        </div>
      </section>}

      {scope === "operator" && <section className="settings-group" aria-labelledby="advanced-support-title" data-setting-id="support-bundle">
        <header><div><h4 id="advanced-support-title">{t("advanced.supportTitle")}</h4><p>{t("advanced.supportDesc")}</p></div></header>
        <div className="support-bundle-body">
          <div className="support-bundle-copy"><FileJson size={20} aria-hidden="true" /><span><strong>{t("advanced.supportBundleTitle")}</strong><small>{t("advanced.supportBundleDesc")}</small></span></div>
          <div className="support-bundle-actions"><button type="button" className="primary-action" onClick={() => void copySupportBundle()}><Clipboard size={14} aria-hidden="true" /> {t("advanced.copySupport")}</button><button type="button" onClick={downloadSupportBundle}><Download size={14} aria-hidden="true" /> {t("advanced.downloadSupport")}</button></div>
          <p>{t("advanced.supportPrivacy")}</p>
        </div>
        <details className="support-log-accordion" onToggle={(event) => { if (event.currentTarget.open) setLogs(getSupportEvents()); }}>
          <summary><span>{t("advanced.logsTitle")}</span><small>{t("advanced.logsCount", { count: logs.length })}</small></summary>
          <div className="support-log-toolbar"><button type="button" onClick={() => void copyLogs()}><Copy size={13} aria-hidden="true" /> {t("advanced.copyLogs")}</button><button type="button" onClick={clearLogs}><Trash2 size={13} aria-hidden="true" /> {t("advanced.clearLogs")}</button></div>
          <div className="support-log-list" role="log" aria-label={t("advanced.logsTitle")}>
            {logs.length ? logs.map((event, index) => <div className={`support-log-entry ${event.level}`} key={`${event.timestamp}-${index}`}><time>{new Date(event.timestamp).toLocaleTimeString()}</time><strong>{event.source}</strong><code>{event.message}{event.detail ? `; ${event.detail}` : ""}</code></div>) : <p>{t("advanced.logsEmpty")}</p>}
          </div>
        </details>
      </section>}

      {scope === "player" && <section className="settings-group" aria-labelledby="advanced-portability-title" data-setting-id="preferences-portability">
        <header><div><h4 id="advanced-portability-title">{t("advanced.portabilityTitle")}</h4><p>{t("advanced.portabilityDesc")}</p></div></header>
        <div className="advanced-action-list">
          <div><span><strong>{t("advanced.exportTitle")}</strong><small>{t("advanced.exportDesc")}</small></span><div className="advanced-action-buttons"><button type="button" onClick={() => void copyPreferences()}><Copy size={14} aria-hidden="true" /> {t("advanced.copySettings")}</button><button type="button" onClick={exportPreferences}><Download size={14} aria-hidden="true" /> {t("advanced.export")}</button></div></div>
          <div><span><strong>{t("advanced.importTitle")}</strong><small>{t("advanced.importDesc")}</small></span><button type="button" onClick={() => importRef.current?.click()}><Upload size={14} aria-hidden="true" /> {t("advanced.import")}</button><input ref={importRef} className="sr-only" type="file" accept="application/json,.json" onChange={(event) => void importPreferences(event.target.files?.[0])} /></div>
          <div className="advanced-reset-row"><span><strong>{t("advanced.resetTitle")}</strong><small>{t("advanced.resetDesc")}</small></span>{confirmReset ? <div className="advanced-reset-confirm"><button type="button" onClick={() => setConfirmReset(false)}>{t("common.cancel")}</button><button type="button" className="danger-action" onClick={resetPreferences}>{t("advanced.confirmReset")}</button></div> : <button type="button" onClick={() => setConfirmReset(true)}><RotateCcw size={14} aria-hidden="true" /> {t("advanced.reset")}</button>}</div>
        </div>
        <p className="advanced-status" role="status" aria-live="polite">{status}</p>
      </section>}

      {scope === "player" && <section className="settings-group" aria-labelledby="advanced-theme-title" data-setting-id="theme-portability">
        <header><div><h4 id="advanced-theme-title">{t("options:theme")}</h4><p>{t("advanced.portabilityDesc")}</p></div></header>
        <div className="advanced-action-list">
          <div><span><strong>{t("advanced.exportTitle")}</strong><small>{t("advanced.exportDesc")}</small></span><div className="advanced-action-buttons"><button type="button" onClick={exportPortableTheme}><FileJson size={14} aria-hidden="true" /> JSON</button><button type="button" onClick={() => void exportPortableThemeBundle()}><Download size={14} aria-hidden="true" /> ZIP</button></div></div>
          <label className="theme-font-rights"><input type="checkbox" checked={includeThemeFonts} onChange={(event) => setIncludeThemeFonts(event.target.checked)} />{t("advanced.themeFontRights")}</label>
          <div><span><strong>{t("advanced.importTitle")}</strong><small>{t("advanced.importDesc")}</small></span><button type="button" onClick={() => themeImportRef.current?.click()}><Upload size={14} aria-hidden="true" /> {t("advanced.import")}</button><input ref={themeImportRef} className="sr-only" type="file" accept="application/json,application/zip,.json,.zip" onChange={(event) => void previewPortableTheme(event.target.files?.[0])} /></div>
          {themeUndo && <div><span><strong>{t("advanced.resetTitle")}</strong><small>{t("advanced.resetDesc")}</small></span><button type="button" onClick={() => void undoPortableTheme()}><RotateCcw size={14} aria-hidden="true" /> {t("advanced.reset")}</button></div>}
        </div>
        {themePreview && <div className="theme-import-preview" role="region" aria-label={themePreview.theme.meta.name}><strong>{themePreview.theme.meta.name}</strong>{themePreview.changes.length ? <dl>{themePreview.changes.map((change) => <div key={change.key}><dt>{change.key}</dt><dd><span>{change.before}</span><span>{change.after}</span></dd></div>)}</dl> : <p>{t("advanced.imported")}</p>}{themePreview.missingFontIds.length > 0 && <p className="settings-inline-warning">{themePreview.missingFontIds.join(", ")}</p>}<div className="advanced-reset-confirm"><button type="button" onClick={() => setThemePreview(null)}>{t("common:cancel")}</button><button type="button" className="primary-action" onClick={() => void applyPortableTheme()}>{t("advanced.import")}</button></div></div>}
      </section>}
    </div>
  );
}

function downloadJson(value: unknown, filename: string): void {
  const blob = new Blob([JSON.stringify(value, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

async function copyText(value: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch {
      // Fall through to the browser-compatible selection copy below.
    }
  }
  const textarea = document.createElement("textarea");
  textarea.value = value;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.append(textarea);
  textarea.select();
  document.execCommand("copy");
  textarea.remove();
}
