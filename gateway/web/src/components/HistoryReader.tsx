import { ChevronDown, Search } from "lucide-react";
import { useEffect, useId, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getChapters, getHistory } from "../api";
import { StoryExportWorkspace } from "../features/portability/StoryExportWorkspace";
import { availableHistoryActions, eventType, historyGroups, messageMatchesFilters, type HistoryFilters } from "../features/history/historyTimeline";
import { readableStructuredText } from "../format";
import type { ChapterView, MessageView, StorySnapshot } from "../types";
import "./HistoryReader.css";
import { MarkdownText } from "./MarkdownText";

export interface HistoryReaderActions {
  onJump?: (message: MessageView) => void;
  onFork?: (message: MessageView) => void;
  onOpenMap?: (message: MessageView) => void;
  onOpenCodex?: (message: MessageView) => void;
  onOpenAsset?: (message: MessageView) => void;
}

export interface HistoryReaderProps {
  snapshot: StorySnapshot;
  /** Parent integration supplies the canonical branch when it is available. */
  activeBranchId?: string;
  /** Only supplied callbacks render as event actions; unavailable actions stay hidden. */
  actions?: HistoryReaderActions;
}

const initialFilters: HistoryFilters = { query: "", type: "", scope: "current", group: "" };

export function HistoryReader({ snapshot, activeBranchId, actions }: HistoryReaderProps) {
  const { t } = useTranslation(["flow", "surfaces", "portability", "history"]);
  const id = useId();
  const [messages, setMessages] = useState<MessageView[]>([]);
  const [chapters, setChapters] = useState<ChapterView[]>([]);
  const [messageCursor, setMessageCursor] = useState<number | null>(null);
  const [chapterCursor, setChapterCursor] = useState<number | null>(null);
  const [filters, setFilters] = useState(initialFilters);
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const timer = globalThis.setTimeout(() => setDebouncedQuery(filters.query), 250);
    return () => globalThis.clearTimeout(timer);
  }, [filters.query]);

  useEffect(() => {
    const controller = new AbortController();
    setBusy(true); setError("");
    Promise.all([getHistory(snapshot.story.id, undefined, debouncedQuery, controller.signal), getChapters(snapshot.story.id, undefined, debouncedQuery, controller.signal)])
      .then(([history, journal]) => { if (!controller.signal.aborted) { setMessages(history.items); setMessageCursor(history.next_cursor ?? null); setChapters(journal.items); setChapterCursor(journal.next_cursor ?? null); } })
      .catch((reason) => { if (!controller.signal.aborted) setError(errorText(reason)); })
      .finally(() => { if (!controller.signal.aborted) setBusy(false); });
    return () => controller.abort();
  }, [snapshot.story.id, snapshot.version.revision, debouncedQuery]);

  const types = useMemo(() => [...new Set(messages.map(eventType))].sort(), [messages]);
  const branches = useMemo(() => [...new Set(messages.map((message) => message.branch_id).filter(Boolean))].sort(), [messages]);
  const groups = useMemo(() => historyGroups(chapters), [chapters]);
  const filteredMessages = useMemo(() => messages.filter((message) => messageMatchesFilters(message, filters, chapters, activeBranchId)), [activeBranchId, chapters, filters, messages]);
  const hasFilters = Object.values(filters).some((value) => value !== "" && value !== "current");
  const updateFilter = <K extends keyof HistoryFilters>(key: K, value: HistoryFilters[K]) => setFilters((current) => ({ ...current, [key]: value }));

  const loadOlder = async () => {
    if (!messageCursor) return;
    setBusy(true);
    try { const page = await getHistory(snapshot.story.id, messageCursor, debouncedQuery); setMessages((items) => [...page.items, ...items]); setMessageCursor(page.next_cursor ?? null); } catch (reason) { setError(errorText(reason)); } finally { setBusy(false); }
  };
  const loadOlderChapters = async () => {
    if (!chapterCursor) return;
    setBusy(true);
    try { const page = await getChapters(snapshot.story.id, chapterCursor, debouncedQuery); setChapters((items) => [...page.items, ...items]); setChapterCursor(page.next_cursor ?? null); } catch (reason) { setError(errorText(reason)); } finally { setBusy(false); }
  };

  return <div className="history-reader">
    <div className="history-toolbar">
      <label className="history-search"><Search size={14} aria-hidden="true" /><span className="sr-only">{t("history:search")}</span><input type="search" value={filters.query} onChange={(event) => updateFilter("query", event.target.value)} placeholder={t("history:search")} /></label>
      <button type="button" className="history-reset" disabled={!hasFilters} onClick={() => setFilters(initialFilters)}>{t("history:reset")}</button>
    </div>
    <div className="history-filter-grid" aria-label={t("history:filters")}>
      <label>{t("history:type")}<select value={filters.type} onChange={(event) => updateFilter("type", event.target.value)}><option value="">{t("history:allTypes")}</option>{types.map((type) => <option key={type} value={type}>{type}</option>)}</select></label>
      <label>{t("history:scope")}<select value={filters.scope} onChange={(event) => updateFilter("scope", event.target.value)}><option value="current">{t("history:currentBranch")}</option><option value="all">{t("history:allBranches")}</option>{branches.map((branch) => <option key={branch} value={branch}>{branch}</option>)}</select></label>
      <label>{t("history:chapter")}<select value={filters.group} onChange={(event) => updateFilter("group", event.target.value)}><option value="">{t("history:allTurns")}</option>{groups.map((group) => <option key={group.id} value={group.id}>{group.label}</option>)}</select></label>
    </div>
    <p className="history-result-count" aria-live="polite">{t("history:results", { count: filteredMessages.length })}</p>
    <details className="history-export-menu"><summary>{t("exportBranch")}</summary><StoryExportWorkspace storyId={snapshot.story.id} compact /></details>
    {error && <p className="inline-error" role="alert">{error}</p>}
    <section aria-labelledby={`${id}-messages`}><h3 id={`${id}-messages`}>{t("transcript")}</h3>
      {filteredMessages.length === 0 && !busy ? <div className="history-empty"><p className="empty-copy">{t("history:emptyResults")}</p>{hasFilters && <button type="button" className="history-reset" onClick={() => setFilters(initialFilters)}>{t("history:reset")}</button>}</div> : <div className="history-entries">{filteredMessages.map((message) => <HistoryEvent key={message.id} message={message} currentTurn={snapshot.world.current_turn} actions={actions} />)}</div>}
      {messageCursor && <button type="button" disabled={busy} onClick={() => void loadOlder()}>{t("olderMessages")}</button>}
    </section>
    <section aria-labelledby={`${id}-chapters`}><h3 id={`${id}-chapters`}>{t("chapters")}</h3><div className="history-chapters">{chapters.map((chapter) => <article key={chapter.id}><strong>{chapter.title || t("surfaces:history.chapter", { number: chapter.chapter_number })}</strong><span>{t("surfaces:history.turns", { start: chapter.start_turn, end: chapter.end_turn ?? t("surfaces:history.current") })}</span><p>{chapter.summary || t("surfaces:history.noSummary")}</p></article>)}</div>{chapterCursor && <button type="button" disabled={busy} onClick={() => void loadOlderChapters()}>{t("olderChapters")}</button>}</section>
  </div>;
}

const COLLAPSE_AFTER_WORDS = 90;
const PREVIEW_WORDS = 56;

export function HistoryEvent({ message, currentTurn, actions }: { message: MessageView; currentTurn: number; actions?: HistoryReaderActions }) {
  const { t } = useTranslation(["surfaces", "history"]);
  const [expanded, setExpanded] = useState(false);
  const content = readableStructuredText(message.content, t("history.empty"));
  const words = content.trim().split(/\s+/);
  const collapsible = words.length > COLLAPSE_AFTER_WORDS;
  const preview = collapsible && !expanded ? `${words.slice(0, PREVIEW_WORDS).join(" ")}…` : content;
  const role = message.role === "user" ? t("history.you") : message.role === "system" ? t("history.system") : t("history.narrator");
  const isCurrent = message.turn === currentTurn;
  return <article className={`history-message ${message.role}-entry${isCurrent ? " is-current" : ""}`} data-message-role={message.role}>
    <header className="history-message-header"><div className="history-message-meta"><strong>{role}</strong><span className="history-event-type">{eventType(message)}</span>{isCurrent && <span className="history-current">{t("history:currentEvent")}</span>}</div><div className="history-message-meta"><span className="history-branch" title={message.branch_id}>{message.branch_id}</span><span className="history-message-turn">{t("history.turn", { turn: message.turn })}</span></div></header>
    <div className="history-message-body"><div className={!expanded && collapsible ? "history-message-preview" : undefined}><MarkdownText>{preview}</MarkdownText></div>{collapsible && <button type="button" className="history-message-toggle" aria-expanded={expanded} onClick={() => setExpanded((value) => !value)}>{expanded ? t("history.less") : t("history.full", { count: words.length })}<ChevronDown size={16} aria-hidden="true" /></button>}<HistoryActions message={message} actions={actions} /></div>
  </article>;
}

function HistoryActions({ message, actions }: { message: MessageView; actions?: HistoryReaderActions }) {
  const { t } = useTranslation(["surfaces", "history"]);
  if (!actions) return null;
  const callbacks = { jump: actions.onJump, fork: actions.onFork, map: actions.onOpenMap, codex: actions.onOpenCodex, asset: actions.onOpenAsset };
  const available = availableHistoryActions(callbacks);
  if (!available.length) return null;
  return <div className="history-actions" role="group" aria-label={t("history:eventActions", { turn: message.turn })}>{available.map((action) => <button key={action} type="button" className="history-action" onClick={() => callbacks[action]?.(message)}>{t(`history:actions.${action}`)}</button>)}</div>;
}

function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason); }
