import { Search } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { getChapters, getHistory, getStoryExport } from "../api";
import { readableStructuredText } from "../format";
import type { ChapterView, MessageView, StorySnapshot } from "../types";
import { MarkdownText } from "./MarkdownText";

export function HistoryReader({ snapshot }: { snapshot: StorySnapshot }) {
  const id = useId();
  const [messages, setMessages] = useState<MessageView[]>([]);
  const [chapters, setChapters] = useState<ChapterView[]>([]);
  const [messageCursor, setMessageCursor] = useState<number | null>(null);
  const [chapterCursor, setChapterCursor] = useState<number | null>(null);
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    setBusy(true);
    setError("");
    Promise.all([
      getHistory(snapshot.story.id, undefined, query),
      getChapters(snapshot.story.id, undefined, query),
    ])
      .then(([history, journal]) => {
        if (cancelled) return;
        setMessages(history.items);
        setMessageCursor(history.next_cursor ?? null);
        setChapters(journal.items);
        setChapterCursor(journal.next_cursor ?? null);
      })
      .catch((reason) => {
        if (!cancelled) setError(errorText(reason));
      })
      .finally(() => {
        if (!cancelled) setBusy(false);
      });
    return () => { cancelled = true; };
  }, [snapshot.story.id, snapshot.version.revision, query]);

  const loadOlder = async () => {
    if (!messageCursor) return;
    setBusy(true);
    try {
      const page = await getHistory(snapshot.story.id, messageCursor, query);
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
      const page = await getChapters(snapshot.story.id, chapterCursor, query);
      setChapters((items) => [...page.items, ...items]);
      setChapterCursor(page.next_cursor ?? null);
    } catch (reason) {
      setError(errorText(reason));
    } finally {
      setBusy(false);
    }
  };

  const exportAs = async (format: "markdown" | "json") => {
    setBusy(true);
    try {
      const result = await getStoryExport(snapshot.story.id, format);
      const url = URL.createObjectURL(new Blob([result.content], {
        type: format === "json" ? "application/json" : "text/markdown",
      }));
      const link = document.createElement("a");
      link.href = url;
      link.download = result.filename;
      link.click();
      URL.revokeObjectURL(url);
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
      <div className="history-export">
        <button type="button" disabled={busy} onClick={() => void exportAs("markdown")}>Export Markdown</button>
        <button type="button" disabled={busy} onClick={() => void exportAs("json")}>Export JSON</button>
      </div>
      {error && <p className="inline-error" role="alert">{error}</p>}
      <section aria-labelledby={`${id}-messages`}>
        <h3 id={`${id}-messages`}>Transcript</h3>
        {messages.length === 0 && !busy ? (
          <p className="empty-copy">No matching messages on this branch.</p>
        ) : (
          <div className="history-entries">
            {messages.map((message) => (
              <article key={message.id}>
                <header><strong>{message.role === "user" ? "You" : "Narrator"}</strong><span>Turn {message.turn}</span></header>
                <MarkdownText>{readableStructuredText(message.content, "(empty)")}</MarkdownText>
              </article>
            ))}
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

function errorText(reason: unknown): string {
  return reason instanceof Error ? reason.message : String(reason);
}
