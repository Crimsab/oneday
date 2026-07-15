import { ChevronDown, Search } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { getChapters, getHistory, getStoryEpub, getStoryExport, getTelemetryExport } from "../api";
import { readableStructuredText } from "../format";
import type { ChapterView, MessageView, StorySnapshot } from "../types";
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
  const id = useId();
  const [messages, setMessages] = useState<MessageView[]>([]);
  const [chapters, setChapters] = useState<ChapterView[]>([]);
  const [messageCursor, setMessageCursor] = useState<number | null>(null);
  const [chapterCursor, setChapterCursor] = useState<number | null>(null);
  const [query, setQuery] = useState("");
	const [debouncedQuery, setDebouncedQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

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

  const exportAs = async (format: "markdown" | "json" | "epub" | "replay") => {
    setBusy(true);
    try {
		if (format === "epub") {
			const result = await getStoryEpub(snapshot.story.id);
			downloadBlob(result.blob, result.filename);
			return;
		}
      const result = await getStoryExport(snapshot.story.id, format);
      const bytes = result.encoding === "base64" ? Uint8Array.from(atob(result.content), (character) => character.charCodeAt(0)) : result.content;
      downloadBlob(
        new Blob([bytes], { type: result.content_type || (format === "json" || format === "replay" ? "application/json" : "text/markdown") }),
        result.filename,
      );
    } catch (reason) {
      setError(errorText(reason));
    } finally {
      setBusy(false);
    }
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
        <span className="sr-only">Search branch history</span>
        <input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search this branch" />
      </label>
      <details className="history-export-menu">
        <summary>Export this branch</summary>
        <div className="history-export">
          <button type="button" disabled={busy} onClick={() => void exportAs("markdown")}>Export Markdown</button>
          <button type="button" disabled={busy} onClick={() => void exportAs("json")}>Export JSON</button>
          <button type="button" disabled={busy} onClick={() => void exportAs("epub")}>Export EPUB</button>
          <details className="history-technical-export">
            <summary>Technical exports</summary>
            <div>
              <button type="button" disabled={busy} onClick={() => void exportAs("replay")}>Export media replay</button>
              <button type="button" disabled={busy} onClick={() => void exportTelemetry()}>Export telemetry</button>
            </div>
          </details>
        </div>
      </details>
      {error && <p className="inline-error" role="alert">{error}</p>}
      <section aria-labelledby={`${id}-messages`}>
        <h3 id={`${id}-messages`}>Transcript</h3>
        {messages.length === 0 && !busy ? (
          <p className="empty-copy">No matching messages on this branch.</p>
        ) : (
          <div className="history-entries">
            {messages.map((message) => <HistoryMessage key={message.id} message={message} />)}
          </div>
        )}
        {messageCursor && <button type="button" disabled={busy} onClick={() => void loadOlder()}>Load older messages</button>}
      </section>
      <section aria-labelledby={`${id}-chapters`}>
        <h3 id={`${id}-chapters`}>Chapters</h3>
        <div className="history-chapters">
          {chapters.map((chapter) => (
            <article key={chapter.id}>
              <strong>{chapter.title || `Chapter ${chapter.chapter_number}`}</strong>
              <span>Turns {chapter.start_turn}–{chapter.end_turn ?? "current"}</span>
              <p>{chapter.summary || "No summary yet."}</p>
            </article>
          ))}
        </div>
        {chapterCursor && <button type="button" disabled={busy} onClick={() => void loadOlderChapters()}>Load older chapters</button>}
      </section>
    </div>
  );
}

const COLLAPSE_AFTER_WORDS = 90;
const PREVIEW_WORDS = 56;

function HistoryMessage({ message }: { message: MessageView }) {
  const [expanded, setExpanded] = useState(false);
  const content = readableStructuredText(message.content, "(empty)");
  const words = content.trim().split(/\s+/);
  const collapsible = words.length > COLLAPSE_AFTER_WORDS;
  const preview = collapsible && !expanded ? `${words.slice(0, PREVIEW_WORDS).join(" ")}…` : content;
  const role = message.role === "user" ? "You" : message.role === "system" ? "System" : "Narrator";

  return (
    <article className={`history-message ${message.role}-entry`} data-message-role={message.role}>
      <header className="history-message-header">
        <strong>{role}</strong>
        <span className="history-message-turn">Turn {message.turn}</span>
      </header>
      <div className="history-message-body">
        <div className={!expanded && collapsible ? "history-message-preview" : undefined}>
          <MarkdownText>{preview}</MarkdownText>
        </div>
        {collapsible && (
          <button type="button" className="history-message-toggle" aria-expanded={expanded} onClick={() => setExpanded((value) => !value)}>
            {expanded ? "Show less" : `Show full message · ${words.length} words`}
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
