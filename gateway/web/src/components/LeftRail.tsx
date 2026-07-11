import { useState } from "react";
import { Archive, ArchiveRestore, Check, MoreHorizontal, PanelLeftOpen, Pencil, Plus, RefreshCw, Search, Trash2, Users, X } from "lucide-react";
import { moduleSpecs } from "../commands";
import { asArray, compactText, displayTimestamp, entryLabel } from "../format";
import type { ModuleTab, OverlayKind, StorySnapshot, StorySummary, StoryUpdatePayload } from "../types";
import type { TimelineResponse } from "../types";
import { BranchNavigator } from "./BranchNavigator";

const moduleGroups: Array<{ label: string; tabs: ModuleTab[] }> = [
  { label: "Story", tabs: ["history", "map", "codex"] },
  { label: "Character", tabs: ["inventory", "stats", "craft"] },
  { label: "Active threads", tabs: ["fronts", "investigations", "projects"] },
  { label: "Library", tabs: ["saves"] },
];

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
  onUpdateStory: (storyId: string, payload: StoryUpdatePayload) => Promise<void>;
  onSetStoryArchived: (storyId: string, archived: boolean) => Promise<void>;
  onDeleteStory: (storyId: string) => Promise<void>;
  onOpen: (overlay: OverlayKind) => void;
  busyStoryId: string;
	timeline: TimelineResponse | null;
	onForkBranch: (name:string)=>Promise<void>;
	onRenameBranch: (branchId:string,name:string)=>Promise<void>;
	onCheckoutBranch: (branchId:string)=>Promise<void>;
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
  onUpdateStory,
  onSetStoryArchived,
  onDeleteStory,
  onOpen,
  busyStoryId,
	timeline,
	onForkBranch,
	onRenameBranch,
	onCheckoutBranch,
}: LeftRailProps) {
  const notes = storyNotes(snapshot);
  const chapters = snapshot?.panels.chapters.slice(-5).reverse() ?? [];
  const [editingStoryId, setEditingStoryId] = useState("");

  return (
    <aside className="left-rail">
      <div className="rail-brand">
        <strong>OneDay</strong>
        <small>{healthText}</small>
      </div>
      <section className="rail-block stories-block">
        <div className="rail-title">
          <span>Stories</span>
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
              <StoryRow
                key={story.id}
                story={story}
                active={story.id === activeStoryId}
                turnLabel={storyTurnLabel(snapshot, story.id, activeStoryId)}
                editing={editingStoryId === story.id}
                busy={busyStoryId === story.id}
                onSelect={() => onSelectStory(story.id)}
                onEdit={() => setEditingStoryId(story.id)}
                onCancelEdit={() => setEditingStoryId("")}
                onSave={async (payload) => {
                  await onUpdateStory(story.id, payload);
                  setEditingStoryId("");
                }}
                onSetArchived={(archived) => onSetStoryArchived(story.id, archived)}
                onDelete={() => onDeleteStory(story.id)}
              />
            ))
          )}
        </div>
      </section>

	  <BranchNavigator timeline={timeline} busy={Boolean(busyStoryId)} onFork={onForkBranch} onRename={onRenameBranch} onCheckout={onCheckoutBranch} />

      <nav className="module-nav" aria-label="Story tools">
        {moduleGroups.map((group) => (
          <div className="module-group" key={group.label}>
            <span className="module-group-label">{group.label}</span>
            {group.tabs.map((tab) => {
              const spec = moduleSpecs.find((item) => item.tab === tab)!;
              const Icon = spec.Icon;
              return <button type="button" key={tab} className={selectedTab === tab ? "active" : ""} onClick={() => onSelectTab(tab)} title={`${spec.label} (${spec.hotkey})`}><Icon size={17} /><span>{spec.label}</span></button>;
            })}
          </div>
        ))}
      </nav>

      <details className="rail-block notes-block">
        <summary className="rail-title split">
          <span>Story Notes</span>
          <Users size={15} />
        </summary>
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
      </details>
    </aside>
  );
}

interface StoryRowProps {
  story: StorySummary;
  active: boolean;
  turnLabel: string;
  editing: boolean;
  busy: boolean;
  onSelect: () => void;
  onEdit: () => void;
  onCancelEdit: () => void;
  onSave: (payload: StoryUpdatePayload) => Promise<void>;
  onSetArchived: (archived: boolean) => Promise<void>;
  onDelete: () => Promise<void>;
}

function StoryRow({
  story,
  active,
  turnLabel,
  editing,
  busy,
  onSelect,
  onEdit,
  onCancelEdit,
  onSave,
  onSetArchived,
  onDelete,
}: StoryRowProps) {
  const [draft, setDraft] = useState(() => storyDraft(story));

  const resetDraft = () => setDraft(storyDraft(story));
  const save = async () => {
    await onSave({
      name: draft.name,
      description: draft.description,
      genre: draft.genre,
      tone: draft.tone,
    });
  };
  const deleteWithConfirm = async () => {
    await onDelete();
  };

  if (editing) {
    return (
      <form
        className={`story-row story-row-edit ${active ? "active" : ""}`}
        onSubmit={(event) => {
          event.preventDefault();
          void save();
        }}
      >
        <label>
          <span>Name</span>
          <input value={draft.name} onChange={(event) => setDraft((value) => ({ ...value, name: event.target.value }))} />
        </label>
        <label>
          <span>Description</span>
          <textarea
            value={draft.description}
            onChange={(event) => setDraft((value) => ({ ...value, description: event.target.value }))}
            rows={3}
          />
        </label>
        <div className="story-edit-grid">
          <label>
            <span>Genre</span>
            <input value={draft.genre} onChange={(event) => setDraft((value) => ({ ...value, genre: event.target.value }))} />
          </label>
          <label>
            <span>Tone</span>
            <input value={draft.tone} onChange={(event) => setDraft((value) => ({ ...value, tone: event.target.value }))} />
          </label>
        </div>
        <div className="story-edit-actions">
          <button type="button" className="ghost-button" onClick={() => { resetDraft(); onCancelEdit(); }} disabled={busy}>
            <X size={14} />
            Cancel
          </button>
          <button type="submit" className="accent-button" disabled={busy || !draft.name.trim()}>
            <Check size={14} />
            Save
          </button>
        </div>
      </form>
    );
  }

  return (
    <div className={`story-row ${active ? "active" : ""} ${story.is_archived ? "archived" : ""}`}>
      <button type="button" className="story-select" onClick={onSelect} disabled={busy}>
        <strong>{story.name || story.id}</strong>
        <span>
          Turn {turnLabel} - {story.genre || "Story"}
          {story.is_archived ? " - Archived" : ""}
        </span>
        <small title={story.description || story.tone || story.id}>{compactText(story.description || story.tone || story.id, 54)}</small>
      </button>
      <details className="story-row-actions">
        <summary aria-label={`Manage ${story.name || story.id}`} title={`Manage ${story.name || story.id}`}><MoreHorizontal size={16} /></summary>
        <div>
          <button type="button" onClick={() => { resetDraft(); onEdit(); }} disabled={busy}><Pencil size={14} />Edit</button>
          <button type="button" onClick={() => void onSetArchived(!story.is_archived)} disabled={busy}>{story.is_archived ? <ArchiveRestore size={14} /> : <Archive size={14} />}{story.is_archived ? "Restore" : "Archive"}</button>
          <button type="button" className="danger-action" onClick={() => void deleteWithConfirm()} disabled={busy}><Trash2 size={14} />Delete</button>
        </div>
      </details>
    </div>
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
      <button type="button" className="rail-brand-compact" onClick={onExpand} title="Open stories sidebar ([)">
        OD
      </button>
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

function storyDraft(story: StorySummary): Required<Pick<StoryUpdatePayload, "name" | "description" | "genre" | "tone">> {
  return {
    name: story.name || story.id,
    description: story.description || "",
    genre: story.genre || "",
    tone: story.tone || "",
  };
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
