import { ChevronDown, Search } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { getChapters, getHistory } from "../api";
import { readableStructuredText } from "../format";
import { StoryExportWorkspace } from "../features/portability/StoryExportWorkspace";
import type { ChapterView, MessageView, StorySnapshot } from "../types";
import { MarkdownText } from "./MarkdownText";

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

  return (
    <div className="history-reader">
      <label className="history-search">
        <Search size={14} />
        <span className="sr-only">{t("historySearch")}</span>
        <input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("historySearch")} />
      </label>
      <details className="history-export-menu">
        <summary>{t("exportBranch")}</summary>
        <StoryExportWorkspace storyId={snapshot.story.id} compact />
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
