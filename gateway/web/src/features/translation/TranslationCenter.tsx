import { CheckCircle2, CircleStop, Languages, Pause, Play, Plus, RefreshCw, Trash2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getChapters } from "../../api";
import { CustomSelect } from "../../components/CustomSelect";
import { DialogDrawerShell } from "../../components/dialog/DialogDrawerShell";
import type { ChapterView, ModelSettings } from "../../types";
import { completeBrowserTranslationItem, createTranslationGlossary, createTranslationJob, deleteTranslationGlossary, deleteTranslationJob, estimateTranslationJob, listTranslationGlossary, listTranslationJobs, nextBrowserTranslationItem, runTranslationJobAction } from "./batchApi";
import type { TranslationEngine, TranslationEstimate, TranslationGlossaryEntry, TranslationJob, TranslationJobRequest, TranslationStyle } from "./batchTypes";
import { languageCatalog, languageFlagUrl } from "./languageCatalog";
import { prepareBrowserTranslator, supportsBrowserTranslation, translateInBrowser } from "./browserTranslator";

const activeStatus = new Set(["queued", "running"]);

export function TranslationCenter({ storyId, storyLanguage, modelSettings }: { storyId: string; storyLanguage: string; modelSettings: ModelSettings | null }) {
  const { t, i18n } = useTranslation("batch_translation");
  const copy = useCallback((key: string, values?: Record<string, unknown>) => t(key, values), [t]);
  const [open, setOpen] = useState(false);
  const [jobs, setJobs] = useState<TranslationJob[]>([]);
  const [chapters, setChapters] = useState<ChapterView[]>([]);
  const [glossary, setGlossary] = useState<TranslationGlossaryEntry[]>([]);
  const [scope, setScope] = useState<"story" | "chapter">("story");
  const [chapterId, setChapterId] = useState("");
  const [target, setTarget] = useState("en");
  const [engine, setEngine] = useState<TranslationEngine>("browser");
  const [provider, setProvider] = useState("");
  const [model, setModel] = useState("");
  const [style, setStyle] = useState<TranslationStyle>("faithful");
  const [estimate, setEstimate] = useState<TranslationEstimate | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [term, setTerm] = useState("");
  const [equivalent, setEquivalent] = useState("");
  const browserBusy = useRef(false);
  const languages = useMemo(() => languageCatalog(i18n.language), [i18n.language]);
  const enabledProviders = useMemo(() => modelSettings?.providers.filter((item) => item.enabled) ?? [], [modelSettings]);
  const models = useMemo(() => Array.from(new Set([modelSettings?.active.utility_model, modelSettings?.active.narrative_model, ...enabledProviders.map((item) => item.model), ...(modelSettings?.utility_models ?? []), ...(modelSettings?.narrative_models ?? [])].filter((item): item is string => Boolean(item)))), [enabledProviders, modelSettings]);

  const request = useMemo<TranslationJobRequest>(() => ({
    scope_kind: scope,
    scope_id: scope === "chapter" ? chapterId : "",
    source_language: storyLanguage,
    target_language: target,
    engine,
    provider: engine === "ai" ? provider : "",
    model: engine === "ai" ? model : "",
    style: engine === "ai" ? style : "faithful",
  }), [chapterId, engine, model, provider, scope, storyLanguage, style, target]);

  const refresh = useCallback(async () => {
    if (!storyId) { setJobs([]); return; }
    const [nextJobs, nextGlossary] = await Promise.all([listTranslationJobs(storyId), listTranslationGlossary(storyId)]);
    setJobs(nextJobs);
    setGlossary(nextGlossary);
  }, [storyId]);

  useEffect(() => {
    if (!storyId) return;
    let cancelled = false;
    void Promise.all([refresh(), getChapters(storyId)]).then(([, page]) => {
      if (cancelled) return;
      setChapters(page.items);
      setChapterId((current) => current || String(page.items.at(-1)?.id ?? ""));
    }).catch((reason) => { if (!cancelled) setError(String(reason)); });
    return () => { cancelled = true; };
  }, [refresh, storyId]);

  useEffect(() => {
    if (!storyId) return;
    let cancelled = false;
    const poll = async () => {
      try { const next = await listTranslationJobs(storyId); if (!cancelled) setJobs(next); } catch { /* keep last stable state */ }
      if (!cancelled) timer = globalThis.setTimeout(poll, 1500);
    };
    let timer = globalThis.setTimeout(poll, 1500);
    return () => { cancelled = true; globalThis.clearTimeout(timer); };
  }, [storyId]);

  const browserJob = jobs.find((item) => item.engine === "browser" && activeStatus.has(item.status));
  useEffect(() => {
    if (!storyId || browserBusy.current) return;
    const job = browserJob;
    if (!job) return;
    let cancelled = false;
    browserBusy.current = true;
    void nextBrowserTranslationItem(storyId, job.id).then(async (item) => {
      if (!item || cancelled) return;
      const translated = await translateInBrowser({ text: item.source_text, sourceLanguage: item.source_language, targetLanguage: item.target_language, allowDownload: false });
      if (!cancelled) await completeBrowserTranslationItem(storyId, job.id, item.id, translated);
    }).then(refresh).catch(async (reason) => {
      if (!cancelled) {
        setError(copy("browserPaused", { error: reason instanceof Error ? reason.message : String(reason) }));
        await runTranslationJobAction(storyId, job.id, "pause").catch(() => undefined);
        await refresh().catch(() => undefined);
      }
    }).finally(() => { browserBusy.current = false; });
    return () => { cancelled = true; };
  }, [browserJob?.id, browserJob?.status, copy, refresh, storyId]);

  useEffect(() => {
    if (!storyId || (scope === "chapter" && !chapterId) || !target || (engine === "ai" && !provider)) { setEstimate(null); return; }
    const controller = new AbortController();
    const timer = globalThis.setTimeout(() => {
      void estimateTranslationJob(storyId, request).then(setEstimate).catch(() => setEstimate(null));
    }, 250);
    return () => { controller.abort(); globalThis.clearTimeout(timer); };
  }, [chapterId, engine, provider, request, scope, storyId, target]);

  useEffect(() => {
    const first = enabledProviders[0];
    if (!provider && first) setProvider(first.id);
    if (!model) setModel(modelSettings?.active.utility_model || first?.model || "");
  }, [enabledProviders, model, modelSettings, provider]);

  const start = async () => {
    setBusy(true); setError("");
    try {
      if (engine === "browser") await prepareBrowserTranslator(storyLanguage, target);
      await createTranslationJob(storyId, request);
      await refresh();
    } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setBusy(false); }
  };

  const action = async (job: TranslationJob, value: "pause" | "resume" | "cancel" | "retry") => {
    setBusy(true); setError("");
    try {
      if (value === "resume" && job.engine === "browser") await prepareBrowserTranslator(job.source_language, job.target_language);
      await runTranslationJobAction(storyId, job.id, value); await refresh();
    } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setBusy(false); }
  };

  const addGlossary = async (mode: "translate" | "preserve") => {
    if (!term.trim()) return;
    setBusy(true);
    try { await createTranslationGlossary(storyId, { source_language: storyLanguage, target_language: target, source_term: term.trim(), target_term: mode === "preserve" ? "" : equivalent.trim(), mode }); setTerm(""); setEquivalent(""); await refresh(); }
    catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setBusy(false); }
  };

  const active = jobs.find((job) => activeStatus.has(job.status));
  const progress = active ? Math.round((active.completed_items / Math.max(1, active.total_items)) * 100) : 0;
  return (
    <>
      <button type="button" className={`chrome-button translation-center-trigger ${active ? "is-active" : ""}`} onClick={() => setOpen(true)} aria-label={copy("open")}>
        <Languages size={15} aria-hidden="true" />
        <span>{active ? `${active.completed_items}/${active.total_items}` : copy("label")}</span>
      </button>
      {open && <DialogDrawerShell title={copy("title")} className="translation-center-drawer" onClose={() => setOpen(false)}>
        <div className="translation-center">
          {active && <section className="translation-current" aria-live="polite">
            <div><strong>{copy("current")}</strong><span>{copy("progress", { done: active.completed_items, total: active.total_items, progress })}</span></div>
            <progress max={active.total_items} value={active.completed_items}>{progress}%</progress>
          </section>}
          <section className="translation-create">
            <h3>{copy("newJob")}</h3>
            <div className="translation-form-grid">
              <label>{copy("scope")}<CustomSelect value={scope} ariaLabel={copy("scope")} onChange={(value) => setScope(value as "story" | "chapter")} options={[{ value: "story", label: copy("wholeStory") }, { value: "chapter", label: copy("chapter") }]} /></label>
              {scope === "chapter" && <label>{copy("chapter")}<CustomSelect value={chapterId} ariaLabel={copy("chapter")} onChange={setChapterId} options={chapters.map((chapter) => ({ value: String(chapter.id), label: chapter.title || `#${chapter.chapter_number}` }))} /></label>}
              <label>{copy("language")}<CustomSelect value={target} ariaLabel={copy("language")} onChange={setTarget} options={languages.map((language) => ({ value: language.code, label: language.name, iconSrc: languageFlagUrl(language.code) }))} /></label>
              <label>{copy("engine")}<CustomSelect value={engine} ariaLabel={copy("engine")} onChange={(value) => setEngine(value as TranslationEngine)} options={[{ value: "browser", label: copy("browser"), disabled: !supportsBrowserTranslation() }, { value: "ai", label: copy("ai") }]} /></label>
              {engine === "ai" && <><label>{copy("provider")}<CustomSelect value={provider} ariaLabel={copy("provider")} onChange={setProvider} options={enabledProviders.map((item) => ({ value: item.id, label: item.label }))} /></label><label>{copy("model")}<CustomSelect value={model} ariaLabel={copy("model")} onChange={setModel} options={models.map((item) => ({ value: item, label: item }))} /></label></>}
              {engine === "ai" && <label>{copy("style")}<CustomSelect value={style} ariaLabel={copy("style")} onChange={(value) => setStyle(value as TranslationStyle)} options={[{ value: "faithful", label: copy("faithful") }, { value: "natural", label: copy("natural") }, { value: "literary", label: copy("literary") }]} /></label>}
            </div>
            {estimate && <p className="translation-estimate">{copy("estimate", { items: estimate.total_items, characters: estimate.total_characters.toLocaleString(i18n.language), cached: estimate.cache_hits })} {engine === "ai" ? copy("providerCost") : copy("localCost")}</p>}
            <button type="button" className="primary-button" disabled={busy || !estimate?.total_items || (engine === "ai" && !provider)} onClick={() => void start()}><Plus size={14} />{copy("start")}</button>
          </section>
          <section className="translation-jobs">
            <h3>{copy("jobs")}</h3>
            {jobs.length === 0 ? <p className="empty-copy">{copy("noJobs")}</p> : jobs.map((job) => <article key={job.id} className={`translation-job status-${job.status}`}>
              <div className="translation-job-main"><strong>{copy(job.scope_kind === "story" ? "wholeStory" : "chapter")}</strong><span>{job.target_language.toUpperCase()} / {copy(job.engine)} / {copy(job.status)}</span><small>{job.completed_items}/{job.total_items} {copy("items")}{job.failed_items ? `, ${job.failed_items} ${copy("failedItems")}` : ""}</small></div>
              <div className="translation-job-actions">
                {activeStatus.has(job.status) && <button type="button" disabled={busy} onClick={() => void action(job, "pause")} aria-label={copy("pause")}><Pause size={14} /></button>}
                {job.status === "paused" && <button type="button" disabled={busy} onClick={() => void action(job, "resume")} aria-label={copy("resume")}><Play size={14} /></button>}
                {activeStatus.has(job.status) && <button type="button" disabled={busy} onClick={() => void action(job, "cancel")} aria-label={copy("cancel")}><CircleStop size={14} /></button>}
                {(job.status === "failed" || job.status === "partial") && <button type="button" disabled={busy} onClick={() => void action(job, "retry")} aria-label={copy("retry")}><RefreshCw size={14} /></button>}
                <button type="button" disabled={busy || activeStatus.has(job.status)} onClick={() => void deleteTranslationJob(storyId, job.id, false).then(refresh)} aria-label={copy("deleteJob")}><Trash2 size={14} /></button>
                <button type="button" disabled={busy || activeStatus.has(job.status)} onClick={() => void deleteTranslationJob(storyId, job.id, true).then(refresh)} aria-label={copy("deleteTranslations")}><X size={14} /></button>
              </div>
              {job.status === "completed" && <CheckCircle2 size={15} aria-label={copy("completed")} />}
            </article>)}
          </section>
          <section className="translation-glossary">
            <h3>{copy("glossary")}</h3>
            <div className="translation-glossary-form"><input value={term} onChange={(event) => setTerm(event.target.value)} placeholder={copy("sourceTerm")} /><input value={equivalent} onChange={(event) => setEquivalent(event.target.value)} placeholder={copy("equivalent")} /><button type="button" disabled={busy || !term.trim()} onClick={() => void addGlossary("translate")}>{copy("add")}</button><button type="button" disabled={busy || !term.trim()} onClick={() => void addGlossary("preserve")}>{copy("preserve")}</button></div>
            <div className="translation-glossary-list">{glossary.map((entry) => <span key={entry.id}><strong>{entry.source_term}</strong>{entry.mode === "translate" ? ` → ${entry.target_term}` : ` ${copy("unchanged")}`}<button type="button" onClick={() => void deleteTranslationGlossary(storyId, entry.id).then(refresh)} aria-label={copy("removeTerm")}><X size={12} /></button></span>)}</div>
          </section>
          {error && <p className="inline-error" role="alert">{error}</p>}
        </div>
      </DialogDrawerShell>}
    </>
  );
}
