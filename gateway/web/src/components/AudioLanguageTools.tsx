import { useEffect, useState, type FormEvent } from "react";
import { Download, Trash2, Wrench } from "lucide-react";
import { cleanupAudio, deletePronunciation, getAudioExport, getPronunciations, updatePronunciation } from "../api";
import type { AudioCleanupResult, PronunciationEntry } from "../types";
import { CustomSelect } from "./CustomSelect";

export function AudioLanguageTools({ storyId, language, revision }: { storyId: string; language: string; revision: number }) {
  const [entries, setEntries] = useState<PronunciationEntry[]>([]);
  const [source, setSource] = useState("");
  const [spoken, setSpoken] = useState("");
  const [alphabet, setAlphabet] = useState<PronunciationEntry["alphabet"]>("provider");
  const [caseSensitive, setCaseSensitive] = useState(false);
  const [audit, setAudit] = useState<AudioCleanupResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [feedback, setFeedback] = useState("");

  const reload = () => getPronunciations(storyId, language).then((response) => setEntries(response.pronunciations || []));
  useEffect(() => { void reload().catch((cause) => setFeedback(errorText(cause))); }, [storyId, language]);

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
      setFeedback("Pronunciation saved. New synthesis uses a fresh cache identity.");
    } catch (cause) { setFeedback(errorText(cause)); }
    finally { setBusy(false); }
  };

  const remove = async (entry: PronunciationEntry) => {
    setBusy(true); setFeedback("");
    try { await deletePronunciation(storyId, entry.id, revision); await reload(); setFeedback("Pronunciation removed."); }
    catch (cause) { setFeedback(errorText(cause)); }
    finally { setBusy(false); }
  };

  const inspectCache = async () => {
    setBusy(true); setFeedback("");
    try { const response = await cleanupAudio(storyId, true); setAudit(response.cleanup); setFeedback(cleanupText(response.cleanup)); }
    catch (cause) { setFeedback(errorText(cause)); }
    finally { setBusy(false); }
  };

  const cleanCache = async () => {
    setBusy(true); setFeedback("");
    try { const response = await cleanupAudio(storyId, false); setAudit(null); setFeedback(cleanupText(response.cleanup)); }
    catch (cause) { setFeedback(errorText(cause)); }
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
      setFeedback("Branch-safe audio manifest exported.");
    } catch (cause) { setFeedback(errorText(cause)); }
    finally { setBusy(false); }
  };

  const removable = Boolean(audit && audit.orphan_files + audit.invalid_cache_rows > 0);
  return (
    <div className="audio-language-tools">
      <div className="settings-section-head"><div><h4>Pronunciation lexicon</h4><p>Applies by language before synthesis and invalidates matching cache identities.</p></div></div>
      <form className="pronunciation-form" onSubmit={save}>
        <label><span>Written text</span><input value={source} onChange={(event) => setSource(event.target.value)} placeholder="Lyanna" required /></label>
        <label><span>Spoken form</span><input value={spoken} onChange={(event) => setSpoken(event.target.value)} placeholder="Lee-ah-na" required /></label>
        <label><span>Alphabet</span><CustomSelect value={alphabet} ariaLabel="Pronunciation alphabet" onChange={(value) => setAlphabet(value as PronunciationEntry["alphabet"])} options={[{ value: "provider", label: "Provider guidance" }, { value: "ipa", label: "IPA" }, { value: "x-sampa", label: "X-SAMPA" }]} /></label>
        <label className="toggle-row"><span>Case sensitive</span><input type="checkbox" checked={caseSensitive} onChange={(event) => setCaseSensitive(event.target.checked)} /></label>
        <button type="submit" disabled={busy || !source.trim() || !spoken.trim()}>Add pronunciation</button>
      </form>
      {entries.length > 0 && <div className="pronunciation-list" aria-label="Pronunciation entries">{entries.map((entry) => <div key={entry.id}><span><strong>{entry.source_text}</strong><small>{entry.pronunciation} · {entry.alphabet} · rev {entry.revision}</small></span><button type="button" className="square-button" disabled={busy} onClick={() => void remove(entry)} title={`Delete pronunciation for ${entry.source_text}`}><Trash2 size={14} /></button></div>)}</div>}
      <div className="audio-maintenance-actions">
        <button type="button" disabled={busy} onClick={inspectCache}><Wrench size={14} aria-hidden="true" /> Audit cache</button>
        <button type="button" disabled={busy || !removable} onClick={cleanCache}>Remove orphaned files</button>
        <button type="button" disabled={busy} onClick={exportManifest}><Download size={14} aria-hidden="true" /> Export audio manifest</button>
      </div>
      <p className="settings-feedback" role="status" aria-live="polite">{busy ? "Working…" : feedback}</p>
    </div>
  );
}

function cleanupText(result: AudioCleanupResult): string {
  if (result.dry_run) return `Audit: ${result.files_scanned} audio files, ${result.orphan_files} orphaned, ${result.invalid_cache_rows} invalid cache rows.`;
  return `Cleanup: ${result.files_removed} orphaned files removed; ${result.invalid_cache_rows} invalid cache rows handled.`;
}

function errorText(cause: unknown): string { return cause instanceof Error ? cause.message : "Audio operation failed"; }
