import { ScanSearch, Search, Trash2, Upload } from "lucide-react";
import { useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  bundledFontChoices,
  cssFontFamily,
  deleteImportedFont,
  importFontFile,
  listImportedFonts,
  loadImportedFonts,
  querySystemFonts,
  supportsLocalFontAccess,
  type FontChoice,
  type ImportedFontRecord,
} from "../../fontLibrary";
import { DEFAULT_FONT_FAMILY, DEFAULT_READING_COLOR } from "../../preferences";
import type { AppPreferences } from "../../types";
import { CustomSelect } from "../CustomSelect";
import { DeferredColorPicker } from "./DeferredColorPicker";

const MAX_FONT_BYTES = 20 * 1024 * 1024;

export function TypographySettings({ preferences, onChange }: { preferences: AppPreferences; onChange: (preferences: AppPreferences) => void }) {
  const { t } = useTranslation("settings_ui");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [imported, setImported] = useState<ImportedFontRecord[]>([]);
  const [systemFonts, setSystemFonts] = useState<FontChoice[]>([]);
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query.trim().toLocaleLowerCase());
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState("");
  const update = <K extends keyof AppPreferences>(key: K, value: AppPreferences[K]) => onChange({ ...preferences, [key]: value });

  useEffect(() => {
    let current = true;
    void loadImportedFonts()
      .then((records) => { if (current) setImported(records); })
      .catch((cause) => { if (current) setStatus(errorText(cause)); });
    return () => { current = false; };
  }, []);

  const allFonts = useMemo(() => {
    const choices: FontChoice[] = [...bundledFontChoices, ...systemFonts, ...imported];
    if (!choices.some((font) => font.id === preferences.fontId)) {
      choices.unshift({ id: preferences.fontId, family: preferences.fontFamily, label: preferences.fontFamily, source: preferences.fontSource });
    }
    return choices.filter((choice, index) => choices.findIndex((candidate) => candidate.id === choice.id) === index);
  }, [imported, preferences.fontFamily, preferences.fontId, preferences.fontSource, systemFonts]);

  const filteredFonts = useMemo(() => {
    if (!deferredQuery) return allFonts;
    return allFonts.filter((font) => `${font.label} ${font.family} ${font.detail ?? ""}`.toLocaleLowerCase().includes(deferredQuery));
  }, [allFonts, deferredQuery]);

  const selectFont = (font: FontChoice) => onChange({ ...preferences, fontId: font.id, fontFamily: font.family, fontSource: font.source });

  const detectFonts = async () => {
    setBusy(true); setStatus("");
    try {
      const fonts = await querySystemFonts();
      setSystemFonts(fonts);
      setStatus(t("font.detected", { count: fonts.length }));
    } catch (cause) {
      setStatus(cause instanceof DOMException && cause.name === "NotAllowedError" ? t("font.permissionDenied") : errorText(cause));
    } finally { setBusy(false); }
  };

  const importFiles = async (files: FileList | null) => {
    if (!files?.length) return;
    setBusy(true); setStatus("");
    try {
      let selected: ImportedFontRecord | null = null;
      for (const file of Array.from(files)) {
        if (file.size > MAX_FONT_BYTES) throw new Error(t("font.fileTooLarge", { name: file.name }));
        selected = await importFontFile(file);
      }
      const records = await listImportedFonts();
      setImported(records);
      if (selected) selectFont(selected);
      setStatus(t("font.imported", { count: files.length }));
    } catch (cause) {
      setStatus(errorText(cause));
    } finally {
      setBusy(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const removeFont = async (font: ImportedFontRecord) => {
    setBusy(true); setStatus("");
    try {
      await deleteImportedFont(font.id);
      setImported((items) => items.filter((item) => item.id !== font.id));
      if (preferences.fontId === font.id) {
        onChange({ ...preferences, fontId: "bundled:ibm-plex-sans", fontFamily: DEFAULT_FONT_FAMILY, fontSource: "bundled" });
      }
      setStatus(t("font.removed", { name: font.label }));
    } catch (cause) {
      setStatus(errorText(cause));
    } finally { setBusy(false); }
  };

  const previewStyle = {
    fontFamily: cssFontFamily(preferences.fontFamily),
    fontSize: `${preferences.readingFontSize}px`,
    fontWeight: preferences.readingFontWeight,
    fontStyle: preferences.readingFontStyle,
    color: preferences.readingTextColor,
  };

  return (
    <section className="settings-group typography-settings" aria-labelledby="typography-title" data-setting-id="typography">
      <header><div><h4 id="typography-title">{t("font.title")}</h4><p>{t("font.description")}</p></div></header>
      <div className="font-browser">
        <div className="font-browser-toolbar">
          <label className="font-search"><Search size={15} aria-hidden="true" /><span className="sr-only">{t("font.search")}</span><input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("font.search")} /></label>
          <button type="button" disabled={busy || !supportsLocalFontAccess()} onClick={() => void detectFonts()} title={!supportsLocalFontAccess() ? t("font.unsupported") : undefined}><ScanSearch size={15} aria-hidden="true" /> {t("font.detect")}</button>
          <button type="button" disabled={busy} onClick={() => fileInputRef.current?.click()}><Upload size={15} aria-hidden="true" /> {t("font.import")}</button>
          <input ref={fileInputRef} className="sr-only" type="file" accept=".woff,.woff2,.ttf,.otf,font/woff,font/woff2,font/ttf,font/otf" multiple onChange={(event) => void importFiles(event.target.files)} />
        </div>
        <div className="font-browser-body">
          <div className="font-choice-list" role="listbox" aria-label={t("font.available")}>
            {filteredFonts.slice(0, 120).map((font) => {
              const importedFont = font.source === "imported" ? imported.find((item) => item.id === font.id) : null;
              return <div className={`font-choice-row ${preferences.fontId === font.id ? "selected" : ""}`} key={font.id}>
                <button type="button" role="option" aria-selected={preferences.fontId === font.id} onClick={() => selectFont(font)}>
                  <span style={{ fontFamily: cssFontFamily(font.family) }}>{font.label}</span>
                  <small>{t(`font.source.${font.source}`)}{font.detail ? ` / ${font.detail}` : ""}</small>
                </button>
                {importedFont && <button type="button" className="font-delete-button" disabled={busy} onClick={() => void removeFont(importedFont)} aria-label={t("font.remove", { name: font.label })} title={t("font.remove", { name: font.label })}><Trash2 size={14} aria-hidden="true" /></button>}
              </div>;
            })}
            {!filteredFonts.length && <p className="font-empty">{t("font.noResults")}</p>}
          </div>
          <div className="font-controls">
            <div className="settings-field"><span>{t("font.scope")}</span><CustomSelect value={preferences.fontScope} ariaLabel={t("font.scope")} onChange={(value) => update("fontScope", value as AppPreferences["fontScope"])} options={["reading", "interface", "all"].map((value) => ({ value, label: t(`font.scopeOption.${value}`) }))} /></div>
            <label className="font-size-control"><span>{t("font.size")}</span><output>{preferences.readingFontSize}px</output><input type="range" min="13" max="26" step="1" value={preferences.readingFontSize} onChange={(event) => update("readingFontSize", Number(event.target.value))} /></label>
            <div className="settings-field"><span>{t("font.weight")}</span><CustomSelect value={String(preferences.readingFontWeight)} ariaLabel={t("font.weight")} onChange={(value) => update("readingFontWeight", Number(value))} options={[300, 400, 500, 600, 700].map((value) => ({ value: String(value), label: t(`font.weightOption.${value}`) }))} /></div>
            <div className="font-style-control"><span>{t("font.style")}</span><div role="group" aria-label={t("font.style")}><button type="button" aria-pressed={preferences.readingFontStyle === "normal"} onClick={() => update("readingFontStyle", "normal")}>{t("font.normal")}</button><button type="button" aria-pressed={preferences.readingFontStyle === "italic"} onClick={() => update("readingFontStyle", "italic")}><em>{t("font.italic")}</em></button></div></div>
            <DeferredColorPicker value={preferences.readingTextColor} defaultValue={DEFAULT_READING_COLOR} label={t("font.textColor")} description={t("font.textColorHint")} onApply={(color) => update("readingTextColor", color)} />
          </div>
        </div>
        <p className="font-library-status" role="status" aria-live="polite">{busy ? t("font.working") : status}</p>
      </div>
      <div className="font-preview" style={previewStyle}>
        <span>{t("font.previewLabel")}</span>
        <h5>{t("font.previewTitle")}</h5>
        <p>{t("font.previewBody")}</p>
        <blockquote>{t("font.previewDialogue")}</blockquote>
        <small>{t("font.previewUi")}</small>
      </div>
    </section>
  );
}

function errorText(cause: unknown): string {
  return cause instanceof Error ? cause.message : String(cause);
}
