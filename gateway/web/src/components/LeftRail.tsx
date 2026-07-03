import {
  Archive,
  BarChart3,
  BookOpen,
  BriefcaseBusiness,
  Clock3,
  FileText,
  Flag,
  Plus,
  RefreshCw,
  Search,
  Users,
} from "lucide-react";
import { asArray, compactText, entryLabel, fieldRows } from "../format";
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

const modules: Array<{ tab: ModuleTab; label: string; Icon: typeof Clock3 }> = [
  { tab: "history", label: "History", Icon: Clock3 },
  { tab: "inventory", label: "Inventory", Icon: Archive },
  { tab: "stats", label: "Stats", Icon: BarChart3 },
  { tab: "codex", label: "Codex", Icon: BookOpen },
  { tab: "fronts", label: "Fronts", Icon: Flag },
  { tab: "investigations", label: "Investigations", Icon: Search },
  { tab: "projects", label: "Projects", Icon: BriefcaseBusiness },
  { tab: "saves", label: "Saves", Icon: FileText },
];

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
                <small>{compactText(story.description || story.tone || story.id, 54)}</small>
              </button>
            ))
          )}
        </div>
      </section>

      <nav className="module-nav" aria-label="Story modules">
        {modules.map(({ tab, label, Icon }) => (
          <button
            type="button"
            key={tab}
            className={selectedTab === tab ? "active" : ""}
            onClick={() => onSelectTab(tab)}
          >
            <Icon size={17} />
            {label}
          </button>
        ))}
      </nav>

      <section className="rail-block notes-block">
        <div className="rail-title split">
          <span>Story Notes</span>
          <Users size={15} />
        </div>
        {notes.length === 0 ? (
          <div className="empty-copy">Select a story to see hooks, contacts, and next leads.</div>
        ) : (
          <div className="notes-copy">
            {notes.map((note) => (
              <p key={note}>{note}</p>
            ))}
          </div>
        )}
      </section>
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
  const guidance = fieldRows(snapshot.world.guidance)[0];
  if (guidance) notes.push(`Next Lead: ${compactText(guidance[1], 90)}`);
  notes.push(`Updated: ${snapshot.world.updated_at || snapshot.server_time}`);
  return notes;
}
