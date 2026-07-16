import { ChevronDown, Search } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { getChapters, getHistory, getStoryEpub, getStoryExport, getTelemetryExport } from "../api";
import { readableStructuredText } from "../format";
import { exportArchive, exportTemplate, type ArchiveOptions, type ReadableFormat, type ReadingMode } from "../features/portability/portabilityApi";
import { encodeTemplateCode } from "../features/portability/templateCode";
import type { ChapterView, MessageView, StorySnapshot } from "../types";
import { CustomSelect } from "./CustomSelect";
import { MarkdownText } from "./MarkdownText";

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.hidden = true;
  document.body.append(link);
  link.click();
  link.remove();
  globalThis.setTimeout(() => URL.revokeObjectURL(url), 0);
}

export function HistoryReader({ snapshot }: { snapshot: StorySnapshot }) {
  const { t } = useTranslation(["flow", "surfaces", "portability"]);
  const id = useId();
  const [messages, setMessages] = useState<MessageView[]>([]);
  const [chapters, setChapters] = useState<ChapterView[]>([]);
  const [messageCursor, setMessageCursor] = useState<number | null>(null);
  const [chapterCursor, setChapterCursor] = useState<number | null>(null);
  const [query, setQuery] = useState("");
	const [debouncedQuery, setDebouncedQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [format, setFormat] = useState<ReadableFormat>("markdown");
  const [readingMode, setReadingMode] = useState<ReadingMode>("original");
  const [targetLanguage, setTargetLanguage] = useState("en");
  const [archiveOptions, setArchiveOptions] = useState<ArchiveOptions>({ history: true, saves: true, visual_assets: true, audio: true, translations: true, world_detail: true });

	useEffect(() => {
		const timer = globalThis.setTimeout(() => setDebouncedQuery(query), 250);
		return () => globalThis.clearTimeout(timer);
	}, [query]);

  useEffect(() => {
		const controller = new AbortController();
    setBusy(true);
    setError("");
    Promise.all([
		getHistory(snapshot.story.id, undefined, debouncedQuery, controller.signal),
		getChapters(snapshot.story.id, undefined, debouncedQuery, controller.signal),
    ])
      .then(([history, journal]) => {
		if (controller.signal.aborted) return;
        setMessages(history.items);
        setMessageCursor(history.next_cursor ?? null);
        setChapters(journal.items);
        setChapterCursor(journal.next_cursor ?? null);
      })
      .catch((reason) => {
		if (!controller.signal.aborted) setError(errorText(reason));
      })
      .finally(() => {
		if (!controller.signal.aborted) setBusy(false);
      });
		return () => controller.abort();
	}, [snapshot.story.id, snapshot.version.revision, debouncedQuery]);

  const loadOlder = async () => {
    if (!messageCursor) return;
    setBusy(true);
    try {
		const page = await getHistory(snapshot.story.id, messageCursor, debouncedQuery);
      setMessages((items) => [...page.items, ...items]);
      setMessageCursor(page.next_cursor ?? null);
    } catch (reason) {
      setError(errorText(reason));
    } finally {
      setBusy(false);
    }
  };

  const loadOlderChapters = async () => {
    if (!chapterCursor) return;
    setBusy(true);
    try {
		const page = await getChapters(snapshot.story.id, chapterCursor, debouncedQuery);
      setChapters((items) => [...page.items, ...items]);
      setChapterCursor(page.next_cursor ?? null);
    } catch (reason) {
      setError(errorText(reason));
    } finally {
      setBusy(false);
    }
  };

  const exportAs = async (selectedFormat: ReadableFormat | "replay") => {
    setBusy(true);
    try {
		if (selectedFormat === "epub") {
			const result = await getStoryEpub(snapshot.story.id, targetLanguage, readingMode);
			downloadBlob(result.blob, result.filename);
			return;
		}
      const result = await getStoryExport(snapshot.story.id, selectedFormat, targetLanguage, readingMode);
      const bytes = result.encoding === "base64" ? Uint8Array.from(atob(result.content), (character) => character.charCodeAt(0)) : result.content;
      downloadBlob(
        new Blob([bytes], { type: result.content_type || (selectedFormat === "json" || selectedFormat === "replay" ? "application/json" : "text/markdown") }),
        result.filename,
      );
    } catch (reason) {
      setError(errorText(reason));
    } finally {
      setBusy(false);
    }
  };

  const exportPortableArchive = async () => {
    setBusy(true); setError("");
    try {
      const result = await exportArchive(snapshot.story.id, archiveOptions);
      downloadBlob(result.blob, result.filename);
    } catch (reason) { setError(errorText(reason)); } finally { setBusy(false); }
  };

  const exportWorldTemplate = async (asCode: boolean) => {
    setBusy(true); setError("");
    try {
      const result = await exportTemplate(snapshot.story.id);
      if (asCode) {
        await navigator.clipboard.writeText(await encodeTemplateCode(result.text));
      } else {
        downloadBlob(new Blob([result.text], { type: "application/json" }), result.filename);
      }
    } catch (reason) { setError(errorText(reason)); } finally { setBusy(false); }
  };

  const exportTelemetry = async () => {
    setBusy(true);
    try {
      const result = await getTelemetryExport(snapshot.story.id);
      downloadBlob(new Blob([result.content], { type: "application/x-ndjson" }), result.filename);
    } catch (reason) {
      setError(errorText(reason));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="history-reader">
      <label className="history-search">
        <Search size={14} />
        <span className="sr-only">{t("historySearch")}</span>
        <input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("historySearch")} />
      </label>
      <details className="history-export-menu">
        <summary>{t("exportBranch")}</summary>
        <div className="history-export history-export-workspace">
          <label><span>{t("portability:format")}</span><CustomSelect value={format} ariaLabel={t("portability:format")} onChange={(value) => setFormat(value as ReadableFormat)} options={["markdown", "html", "txt", "json", "epub"].map((value) => ({ value, label: value.toUpperCase() }))} /></label>
          <label><span>{t("portability:languageVersion")}</span><CustomSelect value={readingMode} ariaLabel={t("portability:languageVersion")} onChange={(value) => setReadingMode(value as ReadingMode)} options={[{ value: "original", label: t("portability:original") }, { value: "translated", label: t("portability:translated") }, { value: "bilingual", label: t("portability:bilingual") }]} /></label>
          {readingMode !== "original" && <label><span>{t("portability:targetLanguage")}</span><input value={targetLanguage} onChange={(event) => setTargetLanguage(event.target.value)} placeholder="en" /></label>}
          <button type="button" className="primary" disabled={busy || (readingMode !== "original" && !targetLanguage.trim())} onClick={() => void exportAs(format)}>{t("portability:download")}</button>
          <details className="history-portable-export">
            <summary>{t("portability:portableArchive")}</summary>
            <div className="history-archive-options">
              {(Object.keys(archiveOptions) as Array<keyof ArchiveOptions>).map((key) => <label key={key}><input type="checkbox" checked={archiveOptions[key]} onChange={(event) => setArchiveOptions((value) => ({ ...value, [key]: event.target.checked }))} />{t(`portability:archiveOptions.${key}`)}</label>)}
              <button type="button" disabled={busy} onClick={() => void exportPortableArchive()}>{t("portability:downloadArchive")}</button>
            </div>
          </details>
          <div className="history-template-actions"><button type="button" disabled={busy} onClick={() => void exportWorldTemplate(false)}>{t("portability:worldTemplate")}</button><button type="button" disabled={busy} onClick={() => void exportWorldTemplate(true)}>{t("portability:copyCode")}</button></div>
          <details className="history-technical-export">
            <summary>{t("surfaces:history.technical")}</summary>
            <div>
              <button type="button" disabled={busy} onClick={() => void exportAs("replay")}>{t("surfaces:history.replay")}</button>
              <button type="button" disabled={busy} onClick={() => void exportTelemetry()}>{t("surfaces:history.telemetry")}</button>
            </div>
          </details>
        </div>
      </details>
      {error && <p className="inline-error" role="alert">{error}</p>}
      <section aria-labelledby={`${id}-messages`}>
        <h3 id={`${id}-messages`}>{t("transcript")}</h3>
        {messages.length === 0 && !busy ? (
          <p className="empty-copy">{t("noMessages")}</p>
        ) : (
          <div className="history-entries">
            {messages.map((message) => <HistoryMessage key={message.id} message={message} />)}
          </div>
        )}
        {messageCursor && <button type="button" disabled={busy} onClick={() => void loadOlder()}>{t("olderMessages")}</button>}
      </section>
      <section aria-labelledby={`${id}-chapters`}>
        <h3 id={`${id}-chapters`}>{t("chapters")}</h3>
        <div className="history-chapters">
          {chapters.map((chapter) => (
            <article key={chapter.id}>
              <strong>{chapter.title || t("surfaces:history.chapter", { number: chapter.chapter_number })}</strong>
              <span>{t("surfaces:history.turns", { start: chapter.start_turn, end: chapter.end_turn ?? t("surfaces:history.current") })}</span>
              <p>{chapter.summary || t("surfaces:history.noSummary")}</p>
            </article>
          ))}
        </div>
        {chapterCursor && <button type="button" disabled={busy} onClick={() => void loadOlderChapters()}>{t("olderChapters")}</button>}
      </section>
    </div>
  );
}

const COLLAPSE_AFTER_WORDS = 90;
const PREVIEW_WORDS = 56;

function HistoryMessage({ message }: { message: MessageView }) {
  const { t } = useTranslation("surfaces");
  const [expanded, setExpanded] = useState(false);
  const content = readableStructuredText(message.content, t("history.empty"));
  const words = content.trim().split(/\s+/);
  const collapsible = words.length > COLLAPSE_AFTER_WORDS;
  const preview = collapsible && !expanded ? `${words.slice(0, PREVIEW_WORDS).join(" ")}…` : content;
  const role = message.role === "user" ? t("history.you") : message.role === "system" ? t("history.system") : t("history.narrator");

  return (
    <article className={`history-message ${message.role}-entry`} data-message-role={message.role}>
      <header className="history-message-header">
        <strong>{role}</strong>
        <span className="history-message-turn">{t("history.turn", { turn: message.turn })}</span>
      </header>
      <div className="history-message-body">
        <div className={!expanded && collapsible ? "history-message-preview" : undefined}>
          <MarkdownText>{preview}</MarkdownText>
        </div>
        {collapsible && (
          <button type="button" className="history-message-toggle" aria-expanded={expanded} onClick={() => setExpanded((value) => !value)}>
            {expanded ? t("history.less") : t("history.full", { count: words.length })}
            <ChevronDown size={16} aria-hidden="true" />
          </button>
        )}
      </div>
    </article>
  );
}

function errorText(reason: unknown): string {
  return reason instanceof Error ? reason.message : String(reason);
}
