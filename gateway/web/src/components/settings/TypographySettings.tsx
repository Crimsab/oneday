import { Check, Link2, MousePointerClick, Pencil, RotateCcw, ScanSearch, Search, Trash2, Upload, X } from "lucide-react";
import { useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  bundledFontChoices,
  cssFontFamily,
  deleteStoredFont,
  importFontFile,
  listStoredFonts,
  loadStoredFonts,
  MAX_STORED_FONT_BYTES,
  querySystemFonts,
  saveOnlineFont,
  supportsLocalFontAccess,
  type FontChoice,
  type OnlineFontRecord,
  type StoredFontRecord,
} from "../../fontLibrary";
import { DEFAULT_FONT_FAMILY, DEFAULT_FONT_ID, DEFAULT_READING_COLOR, resetTypographyPreferences } from "../../preferences";
import type { AppPreferences } from "../../types";
import { CustomSelect } from "../CustomSelect";
import { DeferredColorPicker } from "./DeferredColorPicker";

type FontTarget = "interface" | "reading";

export function TypographySettings({ preferences, onChange }: { preferences: AppPreferences; onChange: (preferences: AppPreferences) => void }) {
  const { t } = useTranslation("settings_ui");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const onlineDialogRef = useRef<HTMLDialogElement>(null);
  const [storedFonts, setStoredFonts] = useState<StoredFontRecord[]>([]);
  const [libraryReady, setLibraryReady] = useState(false);
  const [systemFonts, setSystemFonts] = useState<FontChoice[]>([]);
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query.trim().toLocaleLowerCase());
  const [target, setTarget] = useState<FontTarget>("reading");
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState("");
  const [onlineOpen, setOnlineOpen] = useState(false);
  const [onlineUrl, setOnlineUrl] = useState("");
  const [onlineLabel, setOnlineLabel] = useState("");
  const [editingOnline, setEditingOnline] = useState<OnlineFontRecord | null>(null);
  const update = <K extends keyof AppPreferences>(key: K, value: AppPreferences[K]) => onChange({ ...preferences, [key]: value });

  useEffect(() => {
    let current = true;
    void loadStoredFonts()
      .then((records) => { if (current) setStoredFonts(records); })
      .catch((cause) => { if (current) setStatus(errorText(cause)); })
      .finally(() => { if (current) setLibraryReady(true); });
    return () => { current = false; };
  }, []);

  useEffect(() => {
    const dialog = onlineDialogRef.current;
    if (!dialog) return;
    if (onlineOpen && !dialog.open) dialog.showModal();
    if (!onlineOpen && dialog.open) dialog.close();
  }, [onlineOpen]);

  useEffect(() => {
    if (!libraryReady) return;
    const storedIds = new Set(storedFonts.map((font) => font.id));
    const interfaceMissing = isStoredSource(preferences.interfaceFontSource) && !storedIds.has(preferences.interfaceFontId);
    const readingMissing = isStoredSource(preferences.readingFontSource) && !storedIds.has(preferences.readingFontId);
    if (!interfaceMissing && !readingMissing) return;
    onChange({
      ...preferences,
      ...(interfaceMissing ? defaultFontAssignment("interface") : {}),
      ...(readingMissing ? defaultFontAssignment("reading") : {}),
    });
  }, [libraryReady, onChange, preferences, storedFonts]);

  const allFonts = useMemo(() => {
    const choices: FontChoice[] = [...bundledFontChoices, ...systemFonts, ...storedFonts];
    for (const assignment of [
      { id: preferences.interfaceFontId, family: preferences.interfaceFontFamily, source: preferences.interfaceFontSource },
      { id: preferences.readingFontId, family: preferences.readingFontFamily, source: preferences.readingFontSource },
    ]) {
      if (!choices.some((font) => font.id === assignment.id)) {
        choices.unshift({ ...assignment, label: assignment.family });
      }
    }
    return choices.filter((choice, index) => choices.findIndex((candidate) => candidate.id === choice.id) === index);
  }, [preferences, storedFonts, systemFonts]);

  const filteredFonts = useMemo(() => {
    if (!deferredQuery) return allFonts;
    return allFonts.filter((font) => `${font.label} ${font.family} ${font.detail ?? ""}`.toLocaleLowerCase().includes(deferredQuery));
  }, [allFonts, deferredQuery]);

  const selectFont = (font: FontChoice) => onChange({
    ...preferences,
    ...(target === "interface"
      ? { interfaceFontId: font.id, interfaceFontFamily: font.family, interfaceFontSource: font.source }
      : { readingFontId: font.id, readingFontFamily: font.family, readingFontSource: font.source }),
  });

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
      let selected: StoredFontRecord | null = null;
      for (const file of Array.from(files)) {
        if (file.size > MAX_STORED_FONT_BYTES) throw new Error(t("font.fileTooLarge", { name: file.name }));
        selected = await importFontFile(file);
      }
      setStoredFonts(await listStoredFonts());
      if (selected) selectFont(selected);
      setStatus(t("font.imported", { count: files.length }));
    } catch (cause) {
      setStatus(errorText(cause));
    } finally {
      setBusy(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const openOnlineCreate = () => {
    setEditingOnline(null);
    setOnlineUrl("");
    setOnlineLabel("");
    setOnlineOpen(true);
    setStatus("");
  };

  const openOnlineEdit = (font: OnlineFontRecord) => {
    setEditingOnline(font);
    setOnlineUrl(font.sourceUrl);
    setOnlineLabel(font.label);
    setOnlineOpen(true);
    setStatus("");
  };

  const closeOnlineEditor = () => {
    setOnlineOpen(false);
    setEditingOnline(null);
    setOnlineUrl("");
    setOnlineLabel("");
  };

  const persistOnlineFont = async () => {
    setBusy(true); setStatus("");
    try {
      const record = await saveOnlineFont({ id: editingOnline?.id, label: onlineLabel, sourceUrl: onlineUrl });
      setStoredFonts(await listStoredFonts());
      selectFont(record);
      setStatus(t(editingOnline ? "font.onlineUpdated" : "font.onlineImported", { name: record.label }));
      closeOnlineEditor();
    } catch (cause) {
      setStatus(errorText(cause));
    } finally { setBusy(false); }
  };

  const removeFont = async (font: StoredFontRecord) => {
    setBusy(true); setStatus("");
    try {
      await deleteStoredFont(font.id);
      setStoredFonts((items) => items.filter((item) => item.id !== font.id));
      onChange({
        ...preferences,
        ...(preferences.interfaceFontId === font.id ? defaultFontAssignment("interface") : {}),
        ...(preferences.readingFontId === font.id ? defaultFontAssignment("reading") : {}),
      });
      if (editingOnline?.id === font.id) closeOnlineEditor();
      setStatus(t("font.removed", { name: font.label }));
    } catch (cause) {
      setStatus(errorText(cause));
    } finally { setBusy(false); }
  };

  const resetTypography = () => {
    onChange(resetTypographyPreferences(preferences));
    setStatus(t("font.resetDone"));
  };

  const interfacePreviewStyle = { fontFamily: cssFontFamily(preferences.interfaceFontFamily) };
  const readingPreviewStyle = {
    fontFamily: cssFontFamily(preferences.readingFontFamily),
    fontSize: `${preferences.readingFontSize}px`,
    fontWeight: preferences.readingFontWeight,
    fontStyle: preferences.readingFontStyle,
    color: preferences.readingTextColor,
  };

  return (
    <section className="settings-group typography-settings" aria-labelledby="typography-title" data-setting-id="typography">
      <header>
        <div><h4 id="typography-title">{t("font.title")}</h4><p>{t("font.description")}</p></div>
        <button type="button" className="reset-typography-button" disabled={busy} onClick={resetTypography}><RotateCcw size={14} aria-hidden="true" /> {t("font.reset")}</button>
      </header>

      <div className="font-target-switcher" role="group" aria-label={t("font.targetLabel")}>
        <strong className="font-target-switcher-label">{t("font.targetLabel")}</strong>
        {(["interface", "reading"] as FontTarget[]).map((value) => <button type="button" key={value} className={target === value ? "active" : ""} aria-pressed={target === value} onClick={() => setTarget(value)}>
          <span>{t(`font.target.${value}`)} {target === value ? <Check size={14} aria-hidden="true" /> : <MousePointerClick size={14} aria-hidden="true" />}</span>
          <small>{value === "interface" ? preferences.interfaceFontFamily : preferences.readingFontFamily}</small>
        </button>)}
      </div>

      <div className="font-browser">
        <div className="font-browser-toolbar">
          <label className="font-search"><Search size={15} aria-hidden="true" /><span className="sr-only">{t("font.search")}</span><input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("font.search")} /></label>
          <button type="button" className="font-toolbar-action" disabled={busy || !supportsLocalFontAccess()} onClick={() => void detectFonts()} title={!supportsLocalFontAccess() ? t("font.unsupported") : undefined}><ScanSearch size={15} aria-hidden="true" /> {t("font.detect")}</button>
          <button type="button" className="font-toolbar-action" disabled={busy} onClick={() => fileInputRef.current?.click()}><Upload size={15} aria-hidden="true" /> {t("font.import")}</button>
          <button type="button" className="font-toolbar-action" disabled={busy} onClick={openOnlineCreate}><Link2 size={15} aria-hidden="true" /> {t("font.online")}</button>
          <input ref={fileInputRef} className="sr-only" type="file" accept=".woff,.woff2,.ttf,.otf,font/woff,font/woff2,font/ttf,font/otf" multiple onChange={(event) => void importFiles(event.target.files)} />
        </div>

        <div className="font-browser-body">
          <div className="font-choice-list" role="listbox" aria-label={t("font.available")}>
            {filteredFonts.slice(0, 120).map((font) => {
              const usedByInterface = preferences.interfaceFontId === font.id;
              const usedByReading = preferences.readingFontId === font.id;
              const selected = target === "interface" ? usedByInterface : usedByReading;
              const storedFont = isStoredSource(font.source) ? storedFonts.find((item) => item.id === font.id) : null;
              const onlineFont = storedFont?.source === "online" ? storedFont as OnlineFontRecord : null;
              return <div className={`font-choice-row ${selected ? "selected" : ""}`} key={font.id}>
                <button type="button" role="option" aria-selected={selected} onClick={() => selectFont(font)}>
                  <span className="font-choice-name" style={{ fontFamily: cssFontFamily(font.family) }}>{font.label}</span>
                  <small className="font-choice-meta"><span>{t(`font.source.${font.source}`)}</span>{font.detail && <span>{font.detail}</span>}</small>
                  {(usedByInterface || usedByReading) && <span className="font-use-badges" aria-label={t("font.usedBy")}>
                    {usedByInterface && <small>{t("font.badge.interface")}</small>}
                    {usedByReading && <small>{t("font.badge.reading")}</small>}
                  </span>}
                </button>
                {onlineFont && <button type="button" className="font-edit-button" disabled={busy} onClick={() => openOnlineEdit(onlineFont)} aria-label={t("font.edit", { name: font.label })} title={t("font.edit", { name: font.label })}><Pencil size={14} aria-hidden="true" /></button>}
                {storedFont && <button type="button" className="font-delete-button" disabled={busy} onClick={() => void removeFont(storedFont)} aria-label={t("font.remove", { name: font.label })} title={t("font.remove", { name: font.label })}><Trash2 size={14} aria-hidden="true" /></button>}
              </div>;
            })}
            {!filteredFonts.length && <p className="font-empty">{t("font.noResults")}</p>}
          </div>

          <div className="font-controls">
            <div className="font-target-summary"><span>{t(`font.target.${target}`)}</span><strong>{target === "interface" ? preferences.interfaceFontFamily : preferences.readingFontFamily}</strong><small>{t(`font.targetHint.${target}`)}</small></div>
            {target === "interface" ? (
              <label className="font-size-control settings-span-full"><span>{t("font.interfaceScale")}</span><output>{preferences.interfaceFontScale}%</output><input type="range" min="80" max="130" step="1" value={preferences.interfaceFontScale} onChange={(event) => update("interfaceFontScale", Number(event.target.value))} /><small>{t("font.interfaceScaleHint")}</small></label>
            ) : <>
              <label className="font-size-control settings-span-full"><span>{t("font.size")}</span><output>{preferences.readingFontSize}px</output><input type="range" min="13" max="26" step="1" value={preferences.readingFontSize} onChange={(event) => update("readingFontSize", Number(event.target.value))} /></label>
              <div className="settings-field"><span>{t("font.weight")}</span><CustomSelect value={String(preferences.readingFontWeight)} ariaLabel={t("font.weight")} onChange={(value) => update("readingFontWeight", Number(value))} options={[300, 400, 500, 600, 700].map((value) => ({ value: String(value), label: t(`font.weightOption.${value}`) }))} /></div>
              <div className="font-style-control"><span>{t("font.style")}</span><div role="group" aria-label={t("font.style")}><button type="button" aria-pressed={preferences.readingFontStyle === "normal"} onClick={() => update("readingFontStyle", "normal")}>{t("font.normal")}</button><button type="button" aria-pressed={preferences.readingFontStyle === "italic"} onClick={() => update("readingFontStyle", "italic")}><em>{t("font.italic")}</em></button></div></div>
              <DeferredColorPicker value={preferences.readingTextColor} defaultValue={DEFAULT_READING_COLOR} label={t("font.textColor")} description={t("font.textColorHint")} onApply={(color) => update("readingTextColor", color)} />
            </>}
          </div>
        </div>
        <p className="font-library-status" role="status" aria-live="polite">{busy ? t("font.working") : status}</p>
      </div>

      <div className="font-preview-grid">
        <div className="font-preview font-preview-interface" style={interfacePreviewStyle}><span>{t("font.previewInterfaceLabel")}</span><h5>{t("font.previewUiTitle")}</h5><p>{t("font.previewUi")}</p><button type="button">{t("font.previewButton")}</button></div>
        <div className="font-preview font-preview-reading" style={readingPreviewStyle}><span>{t("font.previewReadingLabel")}</span><h5>{t("font.previewTitle")}</h5><p>{t("font.previewBody")}</p><blockquote>{t("font.previewDialogue")}</blockquote></div>
      </div>

      <dialog ref={onlineDialogRef} className="font-online-dialog" aria-labelledby="font-online-dialog-title" onCancel={(event) => { event.preventDefault(); closeOnlineEditor(); }} onKeyDown={(event) => { if (event.key === "Escape") event.stopPropagation(); }} onPointerDown={(event) => { if (event.target === event.currentTarget) closeOnlineEditor(); }}>
        <form onSubmit={(event) => { event.preventDefault(); void persistOnlineFont(); }}>
          <header><div><strong id="font-online-dialog-title">{editingOnline ? t("font.onlineEditTitle") : t("font.onlineTitle")}</strong><small>{t("font.onlineHint")}</small></div><button type="button" className="font-online-close" onClick={closeOnlineEditor} aria-label={t("common.cancel")}><X size={15} aria-hidden="true" /></button></header>
          <div className="font-online-dialog-fields">
            <label><span>{t("font.onlineUrl")}</span><input type="url" required autoFocus value={onlineUrl} onChange={(event) => setOnlineUrl(event.target.value)} placeholder="https://example.com/font.woff2" /></label>
            <label><span>{t("font.onlineName")}</span><input type="text" maxLength={120} value={onlineLabel} onChange={(event) => setOnlineLabel(event.target.value)} placeholder={t("font.onlineNameOptional")} /></label>
          </div>
          <p className="font-online-privacy">{t("font.onlineStorageHint")}</p>
          <footer><button type="button" onClick={closeOnlineEditor}>{t("common.cancel")}</button><button type="submit" className="primary-action" disabled={busy || !onlineUrl.trim()}>{editingOnline ? t("font.onlineUpdate") : t("font.onlineSave")}</button></footer>
        </form>
      </dialog>
    </section>
  );
}

function defaultFontAssignment(target: FontTarget) {
  return target === "interface"
    ? { interfaceFontId: DEFAULT_FONT_ID, interfaceFontFamily: DEFAULT_FONT_FAMILY, interfaceFontSource: "bundled" as const }
    : { readingFontId: DEFAULT_FONT_ID, readingFontFamily: DEFAULT_FONT_FAMILY, readingFontSource: "bundled" as const };
}

function isStoredSource(source: FontChoice["source"]): source is "imported" | "online" {
  return source === "imported" || source === "online";
}

function errorText(cause: unknown): string {
  return cause instanceof Error ? cause.message : String(cause);
}
