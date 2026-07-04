import { PanelLeftOpen, Plus, RefreshCw, Search, Users } from "lucide-react";
import { moduleSpecs } from "../commands";
import { asArray, compactText, displayTimestamp, entryLabel } from "../format";
import type { ModuleTab, OverlayKind, StorySnapshot, StorySummary } from "../types";

interface LeftRailProps {
  stories: StorySummary[];
  activeStoryId: string;
  filter: string;
  snapshot: StorySnapshot | null;
  selectedTab: ModuleTab;
  healthText: string;
  onFilterChange: (value: string) => void;
  onSelectStory: (storyId: string) => void;
  onSelectTab: (tab: ModuleTab) => void;
  onRefreshStories: () => void;
  onOpen: (overlay: OverlayKind) => void;
}

export function LeftRail({
  stories,
  activeStoryId,
  filter,
  snapshot,
  selectedTab,
  healthText,
  onFilterChange,
  onSelectStory,
  onSelectTab,
  onRefreshStories,
  onOpen,
}: LeftRailProps) {
  const notes = storyNotes(snapshot);
  const chapters = snapshot?.panels.chapters.slice(-5).reverse() ?? [];

  return (
    <aside className="left-rail">
      <section className="rail-block stories-block">
        <div className="rail-title">
          <span>Stories</span>
          <small>{healthText}</small>
        </div>
        <div className="new-story-row">
          <button type="button" className="new-story-button" onClick={() => onOpen("new-story")}>
            <Plus size={16} />
            New Story
          </button>
          <button type="button" className="square-button" onClick={onRefreshStories} title="Refresh stories">
            <RefreshCw size={15} />
          </button>
        </div>
        <label className="search-wrap">
          <Search size={14} />
          <input value={filter} onChange={(event) => onFilterChange(event.target.value)} placeholder="Filter stories" />
        </label>
        <div className="story-list">
          {stories.length === 0 ? (
            <div className="empty-copy">No stories found.</div>
          ) : (
            stories.map((story) => (
              <button
                type="button"
                key={story.id}
                className={`story-row ${story.id === activeStoryId ? "active" : ""}`}
                onClick={() => onSelectStory(story.id)}
              >
                <strong>{story.name || story.id}</strong>
                <span>
                  Turn {storyTurnLabel(snapshot, story.id, activeStoryId)} - {story.genre || "Story"}
                </span>
                <small title={story.description || story.tone || story.id}>{compactText(story.description || story.tone || story.id, 54)}</small>
              </button>
            ))
          )}
        </div>
      </section>

      <nav className="module-nav" aria-label="Story modules">
        {moduleSpecs.map(({ tab, label, hotkey, Icon }) => (
          <button
            type="button"
            key={tab}
            className={selectedTab === tab ? "active" : ""}
            onClick={() => onSelectTab(tab)}
          >
            <Icon size={17} />
            <span>{label}</span>
            <kbd>{hotkey}</kbd>
          </button>
        ))}
      </nav>

      <section className="rail-block notes-block">
        <div className="rail-title split">
          <span>Story Notes</span>
          <Users size={15} />
        </div>
        <div className="notes-content">
          {chapters.length > 0 && (
            <div className="chapter-trail">
              {chapters.map((chapter) => (
                <button type="button" key={chapter.id} title={chapter.summary || chapter.title}>
                  <strong>Chapter {chapter.chapter_number}</strong>
                  <span>{chapter.title || "Untitled"}</span>
                  <small>
                    Turn {chapter.start_turn}
                    {chapter.end_turn ? `-${chapter.end_turn}` : "+"}
                  </small>
                </button>
              ))}
            </div>
          )}
          {notes.length === 0 ? (
            <div className="empty-copy">Select a story to see hooks, contacts, and next leads.</div>
          ) : (
            <div className="notes-copy">
              {notes.map((note) => (
                <p key={note}>{note}</p>
              ))}
            </div>
          )}
        </div>
      </section>
    </aside>
  );
}

interface CollapsedLeftRailProps {
  selectedTab: ModuleTab;
  onSelectTab: (tab: ModuleTab) => void;
  onExpand: () => void;
  onOpen: (overlay: OverlayKind) => void;
}

export function CollapsedLeftRail({ selectedTab, onSelectTab, onExpand, onOpen }: CollapsedLeftRailProps) {
  return (
    <aside className="left-rail-collapsed" aria-label="Collapsed story rail">
      <button type="button" className="rail-icon-button" onClick={onExpand} title="Open stories sidebar ([)">
        <PanelLeftOpen size={18} />
      </button>
      <button type="button" className="rail-icon-button" onClick={() => onOpen("new-story")} title="New story">
        <Plus size={17} />
      </button>
      <div className="collapsed-module-stack">
        {moduleSpecs.map(({ tab, label, hotkey, Icon }) => (
          <button
            type="button"
            key={tab}
            className={selectedTab === tab ? "active" : ""}
            onClick={() => onSelectTab(tab)}
            title={`${label} (${hotkey})`}
          >
            <Icon size={17} />
            <kbd>{hotkey}</kbd>
          </button>
        ))}
      </div>
    </aside>
  );
}

function storyTurnLabel(snapshot: StorySnapshot | null, storyId: string, activeStoryId: string): string {
  if (!snapshot || storyId !== activeStoryId) return "--";
  return String(snapshot.world.current_turn);
}

function storyNotes(snapshot: StorySnapshot | null): string[] {
  if (!snapshot) return [];
  const notes: string[] = [];
  const hooks = asArray(snapshot.world.story_hooks);
  if (hooks[0]) notes.push(`Hook: ${compactText(entryLabel(hooks[0], 0), 90)}`);
  const fronts = asArray(snapshot.world.fronts);
  if (fronts[0]) notes.push(`Active Front: ${compactText(entryLabel(fronts[0], 0), 90)}`);
  const npc = snapshot.panels.npcs[0];
  if (npc) notes.push(`Key Contact: ${npc.name}`);
  const guidance = asArray(snapshot.world.guidance)[0];
  if (guidance) notes.push(`Next Lead: ${compactText(entryLabel(guidance, 0), 90)}`);
  notes.push(`Updated: ${compactTimestamp(snapshot.world.updated_at || snapshot.server_time)}`);
  return notes;
}

function compactTimestamp(value: string | undefined): string {
  const timestamp = displayTimestamp(value);
  const match = timestamp.match(/^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2})/);
  return match ? `${match[1]} ${match[2]}` : timestamp;
}
