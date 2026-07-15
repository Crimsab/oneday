import { useEffect, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import { Download, Trash2, Wrench } from "lucide-react";
import { cleanupAudio, deletePronunciation, getAudioExport, getPronunciations, updatePronunciation } from "../api";
import type { AudioCleanupResult, PronunciationEntry } from "../types";
import { CustomSelect } from "./CustomSelect";

export function AudioLanguageTools({ storyId, language, revision }: { storyId: string; language: string; revision: number }) {
  const { t } = useTranslation(["audio", "audio_tools", "common"]);
  const [entries, setEntries] = useState<PronunciationEntry[]>([]);
  const [source, setSource] = useState("");
  const [spoken, setSpoken] = useState("");
  const [alphabet, setAlphabet] = useState<PronunciationEntry["alphabet"]>("provider");
  const [caseSensitive, setCaseSensitive] = useState(false);
  const [audit, setAudit] = useState<AudioCleanupResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState("");

  const reload = () => getPronunciations(storyId, language).then((response) => setEntries(response.pronunciations || []));
  useEffect(() => { void reload().catch((cause) => setFeedback(errorText(cause, t))); }, [storyId, language, t]);

  const save = async (event: FormEvent) => {
    event.preventDefault();
    if (!source.trim() || !spoken.trim()) return;
    setBusy(true); setFeedback("");
    try {
      await updatePronunciation(storyId, {
        id: crypto.randomUUID(), story_id: storyId, language_tag: language || "en",
        source_text: source.trim(), pronunciation: spoken.trim(), alphabet,
        case_sensitive: caseSensitive, revision: 1,
      }, revision);
      setSource(""); setSpoken("");
      await reload();
      setFeedback(t("audio_tools:saved"));
    } catch (cause) { setFeedback(errorText(cause, t)); }
    finally { setBusy(false); }
  };

  const remove = async (entry: PronunciationEntry) => {
    setBusy(true); setFeedback("");
    try { await deletePronunciation(storyId, entry.id, revision); await reload(); setFeedback(t("audio_tools:removed")); }
    catch (cause) { setFeedback(errorText(cause, t)); }
    finally { setBusy(false); }
  };

  const inspectCache = async () => {
    setBusy(true); setFeedback("");
    try { const response = await cleanupAudio(storyId, true); setAudit(response.cleanup); setFeedback(cleanupText(response.cleanup, t)); }
    catch (cause) { setFeedback(errorText(cause, t)); }
    finally { setBusy(false); }
  };

  const cleanCache = async () => {
    setBusy(true); setFeedback("");
    try { const response = await cleanupAudio(storyId, false); setAudit(null); setFeedback(cleanupText(response.cleanup, t)); }
    catch (cause) { setFeedback(errorText(cause, t)); }
    finally { setBusy(false); }
  };

  const exportManifest = async () => {
    setBusy(true); setFeedback("");
    try {
      const manifest = (await getAudioExport(storyId)).export;
      const anchor = document.createElement("a");
      anchor.href = URL.createObjectURL(new Blob([JSON.stringify(manifest, null, 2)], { type: "application/json" }));
      anchor.download = manifest.filename;
      anchor.click();
      URL.revokeObjectURL(anchor.href);
      setFeedback(t("audio_tools:exported"));
    } catch (cause) { setFeedback(errorText(cause, t)); }
    finally { setBusy(false); }
  };

  const removable = Boolean(audit && audit.orphan_files + audit.invalid_cache_rows + audit.prunable_cache_rows > 0);
  return (
    <div className="audio-language-tools">
      <div className="settings-section-head"><div><h4>{t("audio:pronunciation")}</h4><p>{t("audio:pronunciationHint")}</p></div></div>
      <form className="pronunciation-form" onSubmit={save}>
        <label><span>{t("audio_tools:written")}</span><input value={source} onChange={(event) => setSource(event.target.value)} placeholder="Lyanna" required /></label>
        <label><span>{t("audio_tools:spoken")}</span><input value={spoken} onChange={(event) => setSpoken(event.target.value)} placeholder="Lee-ah-na" required /></label>
        <label><span>{t("audio_tools:alphabet")}</span><CustomSelect value={alphabet} ariaLabel={t("audio_tools:alphabetLabel")} onChange={(value) => setAlphabet(value as PronunciationEntry["alphabet"])} options={[{ value: "provider", label: t("audio_tools:providerGuidance") }, { value: "ipa", label: "IPA" }, { value: "x-sampa", label: "X-SAMPA" }]} /></label>
        <label className="toggle-row"><span>{t("audio_tools:caseSensitive")}</span><input type="checkbox" checked={caseSensitive} onChange={(event) => setCaseSensitive(event.target.checked)} /></label>
        <button type="submit" disabled={busy || !source.trim() || !spoken.trim()}>{t("audio_tools:add")}</button>
      </form>
      {entries.length > 0 && <div className="pronunciation-list" aria-label={t("audio_tools:entries")}>{entries.map((entry) => <div key={entry.id}><span><strong>{entry.source_text}</strong><small>{entry.pronunciation} · {entry.alphabet} · {t("audio_tools:entryRevision", { revision: entry.revision })}</small></span><button type="button" className="square-button" disabled={busy} onClick={() => void remove(entry)} title={t("audio_tools:deleteEntry", { source: entry.source_text })} aria-label={t("audio_tools:deleteEntry", { source: entry.source_text })}><Trash2 size={14} /></button></div>)}</div>}
      <div className="audio-maintenance-actions">
        <button type="button" disabled={busy} onClick={inspectCache}><Wrench size={14} aria-hidden="true" /> {t("audio_tools:audit")}</button>
        <button type="button" disabled={busy || !removable} onClick={cleanCache}>{t("audio_tools:removeOrphans")}</button>
        <button type="button" disabled={busy} onClick={exportManifest}><Download size={14} aria-hidden="true" /> {t("audio_tools:export")}</button>
      </div>
      <p className="settings-feedback" role="status" aria-live="polite">{busy ? t("audio_tools:working") : feedback}</p>
    </div>
  );
}

function cleanupText(result: AudioCleanupResult, t: TFunction): string {
  if (result.dry_run) return t("audio_tools:auditResult", { files: result.files_scanned, orphaned: result.orphan_files, invalid: result.invalid_cache_rows, expired: result.prunable_cache_rows });
  return t("audio_tools:cleanupResult", { files: result.files_removed, expired: result.cache_rows_removed, invalid: result.invalid_cache_rows });
}

function errorText(cause: unknown, t: TFunction): string { return cause instanceof Error ? cause.message : t("audio_tools:failed"); }
