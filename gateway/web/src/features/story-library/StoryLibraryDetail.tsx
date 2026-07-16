import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ArrowLeft, BookOpen, Boxes, GitBranch, History, Save } from "lucide-react";
import { getChapters, getHistory, getStoryOverview, getStorySaves, getTimeline, getVisualAssets } from "../../api";
import { compactText, displayTimestamp } from "../../format";
import type { ChapterPage, HistoryPage, SaveView, StoryOverview, StorySummary, TimelineResponse, VisualAssetsResponse } from "../../types";

type DetailTab = "overview" | "branches" | "chapters" | "saves" | "timeline" | "assets";
type DetailData = StoryOverview | TimelineResponse | ChapterPage | SaveView[] | HistoryPage | VisualAssetsResponse;

const tabs: Array<{ id: DetailTab; icon: typeof BookOpen }> = [
  { id: "overview", icon: BookOpen }, { id: "branches", icon: GitBranch }, { id: "chapters", icon: BookOpen },
  { id: "saves", icon: Save }, { id: "timeline", icon: History }, { id: "assets", icon: Boxes },
];

export function StoryLibraryDetail({ story, onBack, onOpen }: { story: StorySummary; onBack: () => void; onOpen: () => void }) {
  const { t } = useTranslation("library");
  const [tab, setTab] = useState<DetailTab>("overview");
  const [cache, setCache] = useState<Partial<Record<DetailTab, DetailData>>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const pending = loading || cache[tab] === undefined;

  useEffect(() => { setTab("overview"); setCache({}); }, [story.id]);
  useEffect(() => {
    if (cache[tab]) { setLoading(false); return; }
    const controller = new AbortController();
    setLoading(true); setError("");
    const request = tab === "overview" ? getStoryOverview(story.id, controller.signal)
      : tab === "branches" ? getTimeline(story.id, controller.signal)
      : tab === "chapters" ? getChapters(story.id, undefined, "", controller.signal)
      : tab === "saves" ? getStorySaves(story.id, controller.signal)
      : tab === "timeline" ? getHistory(story.id, undefined, "", controller.signal)
      : getVisualAssets(story.id, controller.signal);
    void request.then((data) => setCache((current) => ({ ...current, [tab]: data }))).catch((reason) => {
      if (reason instanceof DOMException && reason.name === "AbortError") return;
      setError(reason instanceof Error ? reason.message : String(reason));
    }).finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [cache, story.id, tab]);

  return <section className="story-library-detail" aria-label={story.name}>
    <header>
      <button type="button" className="story-detail-back" onClick={onBack}><ArrowLeft size={15} />{t("detail.back")}</button>
      <div><strong>{story.name}</strong><small>{story.genre} · {story.language}</small></div>
      <button type="button" className="story-detail-open" onClick={onOpen}>{t("detail.open")}</button>
    </header>
    <nav className="story-detail-tabs" aria-label="Story details">
      {tabs.map(({ id, icon: Icon }) => <button key={id} type="button" className={tab === id ? "active" : ""} aria-pressed={tab === id} onClick={() => setTab(id)}><Icon size={14} />{t(`detail.tabs.${id}`)}</button>)}
    </nav>
    <div className="story-detail-body">
      {pending && <p className="story-detail-state">{t("detail.loading")}</p>}
      {error && <p className="story-detail-state error">{error}</p>}
      {!pending && !error && <DetailContent tab={tab} value={cache[tab]} story={story} />}
    </div>
  </section>;
}

function DetailContent({ tab, value, story }: { tab: DetailTab; value?: DetailData; story: StorySummary }) {
  const { t } = useTranslation("library");
  if (tab === "overview") {
    const overview = value as StoryOverview;
    return <><p className="story-detail-description">{story.description || story.tone || t("detail.noDescription")}</p><dl className="story-overview-grid">
      <Stat label={t("detail.stats.turn")} value={overview.current_turn} /><Stat label={t("detail.stats.branches")} value={overview.branch_count} /><Stat label={t("detail.stats.chapters")} value={overview.chapter_count} />
      <Stat label={t("detail.stats.saves")} value={overview.save_count} /><Stat label={t("detail.stats.messages")} value={overview.message_count} /><Stat label={t("detail.stats.assets")} value={overview.asset_count} />
    </dl></>;
  }
  if (tab === "branches") return <DetailList empty={t("detail.empty")} items={(value as TimelineResponse).branches.map((item) => ({ title: item.name, meta: t("detail.branchMeta", { count: item.head_turn, state: item.id === (value as TimelineResponse).active_branch_id ? t("detail.active") : t("detail.branch") }) }))} />;
  if (tab === "chapters") return <DetailList empty={t("detail.empty")} items={(value as ChapterPage).items.map((item) => ({ title: item.title || t("chapter", { number: item.chapter_number }), meta: t("detail.turnRange", { start: item.start_turn, end: item.end_turn ?? "…" }), text: item.summary }))} />;
  if (tab === "saves") return <DetailList empty={t("detail.empty")} items={(value as SaveView[]).map((item) => ({ title: item.name, meta: `${t("turn", { turn: item.turn })} · ${displayTimestamp(item.created_at)}`, text: item.location }))} />;
  if (tab === "timeline") return <DetailList empty={t("detail.empty")} items={(value as HistoryPage).items.map((item) => ({ title: `${item.role} · ${t("turn", { turn: item.turn })}`, meta: displayTimestamp(item.created_at), text: compactText(item.content, 180) }))} />;
  return <DetailList empty={t("detail.empty")} items={(value as VisualAssetsResponse).assets.map((item) => ({ title: item.subject || item.kind, meta: `${item.kind} · ${item.status}`, text: compactText(item.prompt, 160) }))} />;
}

function Stat({ label, value }: { label: string; value: number }) { return <div><dt>{label}</dt><dd>{value}</dd></div>; }
function DetailList({ items, empty }: { items: Array<{ title: string; meta: string; text?: string }>; empty: string }) {
  if (!items.length) return <p className="story-detail-state">{empty}</p>;
  return <div className="story-detail-list">{items.map((item, index) => <article key={`${item.title}-${index}`}><strong>{item.title}</strong><small>{item.meta}</small>{item.text && <p>{item.text}</p>}</article>)}</div>;
}
