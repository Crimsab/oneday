import { Check, Languages, Search, X } from "lucide-react";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { detectLanguageInBrowser, primaryLanguage, supportsBrowserTranslation, translateInBrowser } from "./browserTranslator";
import { languageCatalog } from "./languageCatalog";
import { readingLanguageForStory, recentTranslationLanguages, rememberTranslationLanguage, setReadingLanguageForStory } from "./translationPreferences";

export function MessageLanguageControl({ storyId, text, storyLanguage, isUser, onTranslationChange }: {
  storyId: string;
  text: string;
  storyLanguage: string;
  isUser: boolean;
  onTranslationChange: (translation: string | null) => void;
}) {
  const { t, i18n } = useTranslation("surfaces");
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [target, setTarget] = useState(() => readingLanguageForStory(storyId));
  const [recent, setRecent] = useState(recentTranslationLanguages);
  const [translated, setTranslated] = useState(false);
  const [busy, setBusy] = useState(false);
  const [downloadProgress, setDownloadProgress] = useState<number | null>(null);
  const [error, setError] = useState("");
  const abortRef = useRef<AbortController | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const pickerRef = useRef<HTMLDivElement>(null);
  const [pickerPosition, setPickerPosition] = useState({ top: 8, left: 8 });
  const options = useMemo(() => languageCatalog(i18n.language), [i18n.language]);

  useEffect(() => {
    const preferred = readingLanguageForStory(storyId);
    setTarget(preferred);
    setTranslated(false);
    onTranslationChange(null);
  }, [storyId, text, onTranslationChange]);

  useEffect(() => {
    if (!target || translated || busy) return;
    void runTranslation(target, false);
    // The automatic path intentionally runs only when a saved reading language changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target, text]);

  useLayoutEffect(() => {
    if (!open) return;
    const reposition = () => {
      const trigger = triggerRef.current;
      if (!trigger) return;
      const rect = trigger.getBoundingClientRect();
      const width = Math.min(320, window.innerWidth - 16);
      const estimatedHeight = 340;
      const below = rect.bottom + 6;
      const top = below + estimatedHeight <= window.innerHeight - 8
        ? below
        : Math.max(8, rect.top - estimatedHeight - 6);
      setPickerPosition({ top, left: Math.max(8, Math.min(rect.left, window.innerWidth - width - 8)) });
    };
    const closeOutside = (event: PointerEvent) => {
      const node = event.target as Node;
      if (!triggerRef.current?.contains(node) && !pickerRef.current?.contains(node)) setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.stopPropagation();
      setOpen(false);
      triggerRef.current?.focus();
    };
    reposition();
    window.addEventListener("resize", reposition);
    window.addEventListener("scroll", reposition, true);
    window.addEventListener("pointerdown", closeOutside);
    window.addEventListener("keydown", closeOnEscape);
    return () => {
      window.removeEventListener("resize", reposition);
      window.removeEventListener("scroll", reposition, true);
      window.removeEventListener("pointerdown", closeOutside);
      window.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  if (!supportsBrowserTranslation() || !storyId || !text.trim()) return null;

  const runTranslation = async (language: string, userInitiated: boolean) => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setBusy(true);
    setError("");
    setDownloadProgress(null);
    try {
      const source = isUser && userInitiated
        ? await detectLanguageInBrowser(text, controller.signal) || primaryLanguage(storyLanguage)
        : primaryLanguage(storyLanguage);
      const result = await translateInBrowser({ text, sourceLanguage: source, targetLanguage: language, signal: controller.signal, onDownloadProgress: setDownloadProgress, allowDownload: userInitiated });
      onTranslationChange(result);
      setTarget(language);
      setTranslated(true);
      if (userInitiated) setRecent(rememberTranslationLanguage(language));
      setOpen(false);
    } catch (cause) {
      const code = cause instanceof Error ? cause.message : "generic";
      if (!controller.signal.aborted && (userInitiated || code !== "language_pack_needs_user_action")) setError(t(`transcript.translation.errors.${code}`));
    } finally {
      if (abortRef.current === controller) abortRef.current = null;
      setBusy(false);
      setDownloadProgress(null);
    }
  };

  const showOriginal = () => {
    abortRef.current?.abort();
    onTranslationChange(null);
    setTranslated(false);
    setBusy(false);
  };
  const toggleRemember = () => {
    const next = readingLanguageForStory(storyId) === target ? "" : target;
    setReadingLanguageForStory(storyId, next);
  };
  const filtered = options.filter((option) => `${option.code} ${option.name}`.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()));
  const ordered = [...recent.map((code) => options.find((option) => option.code === code)).filter((option): option is NonNullable<typeof option> => Boolean(option)), ...filtered.filter((option) => !recent.includes(option.code))];

  return (
    <div className="message-language-control">
      <button ref={triggerRef} type="button" className="message-language-trigger" onClick={() => setOpen((value) => !value)} aria-expanded={open} aria-haspopup="dialog" disabled={busy}>
        <Languages size={14} aria-hidden="true" />
        <span>{target ? options.find((option) => option.code === target)?.name || target.toUpperCase() : t("transcript.translation.translate")}</span>
      </button>
      {translated && <button type="button" className="message-language-secondary" onClick={showOriginal}>{t("transcript.translation.original")}</button>}
      {translated && <button type="button" className="message-language-secondary" onClick={toggleRemember}>{readingLanguageForStory(storyId) === target ? <Check size={13} /> : null}{t("transcript.translation.remember")}</button>}
      {busy && <span className="message-language-status" role="status">{downloadProgress === null ? t("transcript.translation.translating") : t("transcript.translation.downloading", { progress: Math.round(downloadProgress * 100) })}</span>}
      {error && <span className="message-language-error" role="alert">{error}</span>}
      {open && createPortal(
        <div ref={pickerRef} className="language-picker" role="dialog" aria-label={t("transcript.translation.choose")} style={pickerPosition}>
          <label><Search size={14} aria-hidden="true" /><span className="sr-only">{t("transcript.translation.search")}</span><input autoFocus value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("transcript.translation.search")} /></label>
          <button type="button" className="language-picker-close" onClick={() => setOpen(false)} aria-label={t("transcript.translation.close")}><X size={14} /></button>
          <div className="language-picker-list">
            {ordered.map((option) => <button type="button" key={option.code} onClick={() => void runTranslation(option.code, true)}><span aria-hidden="true">{option.code.toUpperCase()}</span><strong>{option.name}</strong></button>)}
          </div>
        </div>,
        document.body,
      )}
    </div>
  );
}
