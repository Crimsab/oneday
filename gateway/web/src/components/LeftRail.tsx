import { useTranslation } from "react-i18next";
import { ChevronDown, LibraryBig, PanelLeftClose, PanelLeftOpen, Users } from "lucide-react";
import { moduleSpecs } from "../commands";
import { asArray, compactText, displayTimestamp, entryLabel } from "../format";
import type { ModuleTab, StorySnapshot, TimelineResponse } from "../types";
import type { DesktopRailMode } from "../types";
import { BranchNavigator } from "./BranchNavigator";

const moduleGroups: Array<{ label: "story" | "character" | "threads" | "library"; tabs: ModuleTab[] }> = [
  { label: "story", tabs: ["history", "map", "codex"] },
  { label: "character", tabs: ["inventory", "stats", "craft"] },
  { label: "threads", tabs: ["fronts", "investigations", "projects"] },
  { label: "library", tabs: ["achievements", "saves"] },
];

interface LeftRailProps {
  activeStoryCount: number;
  presentation: DesktopRailMode;
  snapshot: StorySnapshot | null;
  selectedTab: ModuleTab;
  healthText: string;
  onOpenStoryLibrary: () => void;
  onSelectTab: (tab: ModuleTab) => void;
  onToggleMode: () => void;
  busyStoryId: string;
  timeline: TimelineResponse | null;
  onForkBranch: (name: string) => Promise<void>;
  onRenameBranch: (branchId: string, name: string) => Promise<void>;
  onCheckoutBranch: (branchId: string) => Promise<void>;
}

export function LeftRail({
  activeStoryCount,
  presentation,
  snapshot,
  selectedTab,
  healthText,
  onOpenStoryLibrary,
  onSelectTab,
  onToggleMode,
  busyStoryId,
  timeline,
  onForkBranch,
  onRenameBranch,
  onCheckoutBranch,
}: LeftRailProps) {
  const { t } = useTranslation(["chrome", "library"]);
  const expanded = presentation === "expanded";
  const notes = storyNotes(snapshot, t);
  const chapters = snapshot?.panels.chapters.slice(-5).reverse() ?? [];
  const storiesLabel = t("library:activeStoryCount", { count: activeStoryCount });

  return (
    <aside className={`left-rail rail-${presentation}`} id="story-navigation" aria-label={t("library")}>
      <div className="rail-brand">
        {!expanded ? <button type="button" className="rail-brand-expand" onClick={onToggleMode} aria-label={t("expandLibrary")} title={t("expandLibrary")}><img src="/brand/oneday-mark.png" alt="" /></button> : <img src="/brand/oneday-mark.png" alt="" />}
        {expanded && (
          <span>
            <strong>OneDay</strong>
            <small>{healthText}</small>
          </span>
        )}
      </div>

      <button
        type="button"
        className="rail-stories-button"
        onClick={onOpenStoryLibrary}
        aria-label={storiesLabel}
        title={storiesLabel}
      >
        <LibraryBig size={18} aria-hidden="true" />
        {expanded ? <span>{t("library:storiesCount", { count: activeStoryCount })}</span> : <strong className="rail-stories-count">{activeStoryCount > 99 ? "99+" : activeStoryCount}</strong>}
      </button>

      {expanded && (
        <BranchNavigator
          timeline={timeline}
          busy={Boolean(busyStoryId)}
          onFork={onForkBranch}
          onRename={onRenameBranch}
          onCheckout={onCheckoutBranch}
        />
      )}

      <nav className="module-nav" aria-label={t("tools")}>
        {moduleGroups.map((group) => (
          <div className="module-group" key={group.label}>
            {expanded && <span className="module-group-label">{t(`library:groups.${group.label}`)}</span>}
            {group.tabs.map((tab) => {
              const spec = moduleSpecs.find((item) => item.tab === tab)!;
              const Icon = spec.Icon;
              const label = t(`library:tabs.${tab}`);
              return (
                <button
                  type="button"
                  key={tab}
                  className={selectedTab === tab ? "active" : ""}
                  onClick={() => onSelectTab(tab)}
                  title={`${label} (${spec.hotkey})`}
                  aria-label={label}
                  aria-current={selectedTab === tab ? "page" : undefined}
                >
                  <Icon size={17} />
                  {expanded && <span>{label}</span>}
                </button>
              );
            })}
          </div>
        ))}
      </nav>

      {expanded && (
        <details className="rail-block notes-block">
          <summary className="rail-title split">
            <span>{t("notes")}</span>
            <span className="rail-summary-end"><Users size={15} /><ChevronDown className="disclosure-chevron" size={14} /></span>
          </summary>
          <div className="notes-content">
            {chapters.length > 0 && (
              <div className="chapter-trail">
                {chapters.map((chapter) => (
                  <div className="chapter-summary" key={chapter.id} title={chapter.summary || chapter.title}>
                    <strong>{t("library:chapter", { number: chapter.chapter_number })}</strong>
                    <span>{chapter.title || t("library:untitled")}</span>
                    <small>
                      {t("library:turn", { turn: chapter.start_turn })}
                      {chapter.end_turn ? `-${chapter.end_turn}` : "+"}
                    </small>
                  </div>
                ))}
              </div>
            )}
            {notes.length === 0 ? (
              <div className="empty-copy">{t("library:emptyNotes")}</div>
            ) : (
              <div className="notes-copy">
                {notes.map((note) => <p key={note}>{note}</p>)}
              </div>
            )}
          </div>
        </details>
      )}

      <button type="button" className="rail-collapse-toggle" onClick={onToggleMode} aria-label={t(expanded ? "collapseLibrary" : "expandLibrary")} title={t(expanded ? "collapseLibrary" : "expandLibrary")}>
        {expanded ? <PanelLeftClose size={17} /> : <PanelLeftOpen size={17} />}
        {expanded && <span>{t("collapseLibrary")}</span>}
      </button>
    </aside>
  );
}

function storyNotes(snapshot: StorySnapshot | null, t: (key: string, options?: Record<string, unknown>) => string): string[] {
  if (!snapshot) return [];
  const notes: string[] = [];
  const hooks = asArray(snapshot.world.story_hooks);
  if (hooks[0]) notes.push(t("library:note.hook", { value: compactText(entryLabel(hooks[0], 0), 90) }));
  const fronts = asArray(snapshot.world.fronts);
  if (fronts[0]) notes.push(t("library:note.front", { value: compactText(entryLabel(fronts[0], 0), 90) }));
  const npc = snapshot.panels.npcs[0];
  if (npc) notes.push(t("library:note.contact", { value: npc.name }));
  const guidance = asArray(snapshot.world.guidance)[0];
  if (guidance) notes.push(t("library:note.lead", { value: compactText(entryLabel(guidance, 0), 90) }));
  notes.push(t("library:note.updated", { value: compactTimestamp(snapshot.world.updated_at || snapshot.server_time) }));
  return notes;
}

function compactTimestamp(value: string | undefined): string {
  const timestamp = displayTimestamp(value);
  const match = timestamp.match(/^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2})/);
  return match ? `${match[1]} ${match[2]}` : timestamp;
}
