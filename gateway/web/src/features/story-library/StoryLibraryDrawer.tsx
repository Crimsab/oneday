import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { Archive, ArchiveRestore, Check, FileDown, FileUp, Info, MoreHorizontal, Pencil, Plus, RefreshCw, Search, Trash2, X } from "lucide-react";
import { DialogDrawerShell } from "../../components/dialog/DialogDrawerShell";
import { compactText, displayTimestamp } from "../../format";
import type { AppPreferences, StorySummary, StoryUpdatePayload } from "../../types";
import { activeStoryCount } from "../navigation/railState";
import { filterStoryLibrary, type StoryLibraryStatus } from "./storyLibraryState";
import { StoryLibraryDetail } from "./StoryLibraryDetail";
import { importArchive, importTemplate } from "../portability/portabilityApi";
import { decodeTemplateCode } from "../portability/templateCode";
import { StoryExportDialog } from "../portability/StoryExportWorkspace";

interface StoryLibraryDrawerProps {
  stories: StorySummary[];
  activeStoryId: string;
  activeStoryTurn?: number;
  busyStoryId: string;
  onClose: () => void;
  onNewStory: () => void;
  onRefresh: () => Promise<StorySummary[]> | void;
  onSelectStory: (storyId: string) => void;
  onUpdateStory: (storyId: string, payload: StoryUpdatePayload) => Promise<void>;
  onSetStoryArchived: (storyId: string, archived: boolean) => Promise<void>;
  onDeleteStory: (storyId: string) => Promise<void>;
  timeFormat: AppPreferences["timeFormat"];
  onTimeFormatChange: (timeFormat: AppPreferences["timeFormat"]) => void;
}

export function StoryLibraryDrawer({
  stories,
  activeStoryId,
  activeStoryTurn,
  busyStoryId,
  onClose,
  onNewStory,
  onRefresh,
  onSelectStory,
  onUpdateStory,
  onSetStoryArchived,
  onDeleteStory,
  timeFormat,
  onTimeFormatChange,
}: StoryLibraryDrawerProps) {
  const { t } = useTranslation(["chrome", "library"]);
  const [status, setStatus] = useState<StoryLibraryStatus>("active");
  const [query, setQuery] = useState("");
  const [editingStoryId, setEditingStoryId] = useState("");
  const [detailStoryId, setDetailStoryId] = useState("");
  const [importOpen, setImportOpen] = useState(false);
  const [importCode, setImportCode] = useState("");
  const [importBusy, setImportBusy] = useState(false);
  const [importError, setImportError] = useState("");
  const [exportStoryId, setExportStoryId] = useState("");
  const visibleStories = useMemo(() => filterStoryLibrary(stories, status, query), [query, status, stories]);
  const activeCount = activeStoryCount(stories);
  const archivedCount = stories.length - activeCount;
  const activeStory = stories.find((story) => story.id === activeStoryId);
  const exportStory = stories.find((story) => story.id === exportStoryId);

  const chooseStory = (storyId: string) => {
    onSelectStory(storyId);
    onClose();
  };

  const finishImport = async (storyId: string) => {
    await onRefresh();
    setImportOpen(false);
    setImportCode("");
    onSelectStory(storyId);
    onClose();
  };

  const importFile = async (file?: File) => {
    if (!file) return;
    setImportBusy(true); setImportError("");
    try {
      const result = file.name.toLowerCase().endsWith(".zip") ? await importArchive(file) : await importTemplate(await file.text());
      await finishImport(result.story_id);
    } catch (reason) { setImportError(reason instanceof Error ? reason.message : String(reason)); } finally { setImportBusy(false); }
  };

  const importShareCode = async () => {
    setImportBusy(true); setImportError("");
    try { await finishImport((await importTemplate(await decodeTemplateCode(importCode))).story_id); }
    catch (reason) { setImportError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setImportBusy(false); }
  };

  if (exportStory) {
    return <StoryExportDialog storyId={exportStory.id} storyName={exportStory.name || exportStory.id} onClose={() => setExportStoryId("")} />;
  }

  return (
    <DialogDrawerShell title={t("chrome:library")} className="story-library-drawer" onClose={onClose}>
      <div className="story-library-toolbar">
        <div className="story-library-tabs" role="group" aria-label={t("library:filterStatus")}>
          <button type="button" className={status === "active" ? "active" : ""} aria-pressed={status === "active"} onClick={() => setStatus("active")}>
            {t("library:activeStories", { count: activeCount })}
          </button>
          <button type="button" className={status === "archived" ? "active" : ""} aria-pressed={status === "archived"} onClick={() => setStatus("archived")}>
            {t("library:archivedStories", { count: archivedCount })}
          </button>
        </div>
        <div className="story-library-actions">
          <button type="button" onClick={() => void onRefresh()} aria-label={t("chrome:refresh")} title={t("chrome:refresh")}>
            <RefreshCw size={16} />
          </button>
          <button type="button" onClick={() => setImportOpen((value) => !value)} aria-expanded={importOpen}>
            <FileUp size={16} />{t("library:importStory")}
          </button>
          <button type="button" disabled={!activeStory} onClick={() => activeStory && setExportStoryId(activeStory.id)}>
            <FileDown size={16} />{t("library:exportStory")}
          </button>
          <button type="button" className="story-library-new" onClick={onNewStory}>
            <Plus size={16} />
            {t("chrome:newStory")}
          </button>
        </div>
        {importOpen && <div className="story-library-import">
          <label className="story-library-import-file"><span>{t("library:importFile")}</span><span className="story-library-file-picker"><FileUp size={15} />{t("library:chooseFile")}<input className="sr-only" type="file" accept=".zip,.json,.oneday.json,application/zip,application/json" disabled={importBusy} onChange={(event) => void importFile(event.target.files?.[0])} /></span></label>
          <span>{t("library:orShareCode")}</span>
          <textarea value={importCode} onChange={(event) => setImportCode(event.target.value)} rows={3} placeholder="OD1:…" />
          <button type="button" disabled={importBusy || !importCode.trim()} onClick={() => void importShareCode()}>{t("library:importCode")}</button>
          {importError && <p role="alert" className="inline-error">{importError}</p>}
        </div>}
        <label className="story-library-search">
          <Search size={16} aria-hidden="true" />
          <span className="sr-only">{t("chrome:filter")}</span>
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t("chrome:filter")} />
        </label>
        <label className="story-library-time-format"><span>{t("library:timeFormat")}</span><select value={timeFormat} onChange={(event) => onTimeFormatChange(event.target.value as AppPreferences["timeFormat"])}><option value="system">{t("library:timeFormats.system")}</option><option value="12">{t("library:timeFormats.12")}</option><option value="24">{t("library:timeFormats.24")}</option></select></label>
      </div>
      <div className={`story-library-content ${detailStoryId ? "has-detail" : ""}`}>
        {visibleStories.length === 0 ? (
          <div className="story-library-empty">
            <strong>{query.trim() ? t("chrome:noMatches") : status === "archived" ? t("library:noArchivedStories") : t("chrome:noStories")}</strong>
            {!query.trim() && status === "active" && <button type="button" onClick={onNewStory}><Plus size={16} />{t("chrome:newStory")}</button>}
          </div>
        ) : (
          <div className="story-library-list">
            {visibleStories.map((story) => (
              <StoryLibraryRow
                key={story.id}
                story={story}
                active={story.id === activeStoryId}
                turn={story.id === activeStoryId ? activeStoryTurn : undefined}
                editing={editingStoryId === story.id}
                busy={busyStoryId === story.id}
                onSelect={() => chooseStory(story.id)}
                onEdit={() => setEditingStoryId(story.id)}
                onDetails={() => setDetailStoryId(story.id)}
                onExport={() => setExportStoryId(story.id)}
                onCancelEdit={() => setEditingStoryId("")}
                onSave={async (payload) => {
                  await onUpdateStory(story.id, payload);
                  setEditingStoryId("");
                }}
                onSetArchived={(archived) => onSetStoryArchived(story.id, archived)}
                onDelete={() => onDeleteStory(story.id)}
              />
            ))}
          </div>
        )}
        {detailStoryId && stories.find((story) => story.id === detailStoryId) && <StoryLibraryDetail
          story={stories.find((story) => story.id === detailStoryId)!}
          onBack={() => setDetailStoryId("")}
          onOpen={() => chooseStory(detailStoryId)}
        />}
      </div>
    </DialogDrawerShell>
  );
}

interface StoryLibraryRowProps {
  story: StorySummary;
  active: boolean;
  turn?: number;
  editing: boolean;
  busy: boolean;
  onSelect: () => void;
  onEdit: () => void;
  onDetails: () => void;
  onExport: () => void;
  onCancelEdit: () => void;
  onSave: (payload: StoryUpdatePayload) => Promise<void>;
  onSetArchived: (archived: boolean) => Promise<void>;
  onDelete: () => Promise<void>;
}

function StoryLibraryRow({ story, active, turn, editing, busy, onSelect, onEdit, onDetails, onExport, onCancelEdit, onSave, onSetArchived, onDelete }: StoryLibraryRowProps) {
  const { t } = useTranslation("library");
  const [draft, setDraft] = useState(() => storyDraft(story));
  const resetDraft = () => setDraft(storyDraft(story));

  if (editing) {
    return (
      <form className={`story-library-row story-library-row-edit ${active ? "active" : ""}`} onSubmit={(event) => {
        event.preventDefault();
        if (draft.language.trim() !== story.language.trim() && !window.confirm(t("languageChangeConfirm"))) return;
        void onSave(draft);
      }}>
        <label><span>{t("name")}</span><input value={draft.name} onChange={(event) => setDraft((value) => ({ ...value, name: event.target.value }))} /></label>
        <label><span>{t("description")}</span><textarea value={draft.description} onChange={(event) => setDraft((value) => ({ ...value, description: event.target.value }))} rows={3} /></label>
        <div className="story-library-edit-grid">
          <label><span>{t("genre")}</span><input value={draft.genre} onChange={(event) => setDraft((value) => ({ ...value, genre: event.target.value }))} /></label>
          <label><span>{t("tone")}</span><input value={draft.tone} onChange={(event) => setDraft((value) => ({ ...value, tone: event.target.value }))} /></label>
          <label><span>{t("language")}</span><input value={draft.language} onChange={(event) => setDraft((value) => ({ ...value, language: event.target.value }))} placeholder="it-IT" /></label>
        </div>
        <div className="story-library-edit-actions">
          <button type="button" onClick={() => { resetDraft(); onCancelEdit(); }} disabled={busy}><X size={14} />{t("cancel")}</button>
          <button type="submit" className="primary" disabled={busy || !draft.name.trim()}><Check size={14} />{t("save")}</button>
        </div>
      </form>
    );
  }

  return (
    <article className={`story-library-row ${active ? "active" : ""} ${story.is_archived ? "archived" : ""}`}>
      <button type="button" className="story-library-select" onClick={onSelect} disabled={busy} aria-current={active ? "page" : undefined}>
        <span className="story-library-row-heading"><strong>{story.name || story.id}</strong>{active && <small>{t("current")}</small>}</span>
        <span>{t("storyMeta", { genre: story.genre || t("storyFallback"), language: story.language || "-" })}</span>
        <p>{compactText(story.description || story.tone || story.id, 150)}</p>
        <small className="story-library-row-facts">{turn !== undefined && <span>{t("turn", { turn })}</span>}{story.updated_at && <span>{t("updated", { value: displayTimestamp(story.updated_at, t("unknownUpdated")) })}</span>}</small>
      </button>
      <StoryActionsMenu
        label={t("manage", { name: story.name || story.id })}
        archived={story.is_archived}
        busy={busy}
        onDetails={onDetails}
        onExport={onExport}
        onEdit={() => { resetDraft(); onEdit(); }}
        onSetArchived={() => onSetArchived(!story.is_archived)}
        onDelete={onDelete}
      />
    </article>
  );
}

function StoryActionsMenu({ label, archived, busy, onEdit, onDetails, onExport, onSetArchived, onDelete }: { label: string; archived: boolean; busy: boolean; onEdit: () => void; onDetails: () => void; onExport: () => void; onSetArchived: () => Promise<void>; onDelete: () => Promise<void> }) {
  const { t } = useTranslation("library");
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const reposition = () => {
    const trigger = triggerRef.current;
    if (!trigger) return;
    const rect = trigger.getBoundingClientRect();
    const width = 180;
    const estimatedHeight = 202;
    const top = rect.bottom + estimatedHeight > window.innerHeight - 8 ? rect.top - estimatedHeight - 4 : rect.bottom + 4;
    setPosition({ top: Math.max(8, top), left: Math.max(8, Math.min(rect.right - width, window.innerWidth - width - 8)) });
  };

  useLayoutEffect(() => { if (open) reposition(); }, [open]);
  useEffect(() => {
    if (!open) return;
    const closeOutside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!triggerRef.current?.contains(target) && !menuRef.current?.contains(target)) setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    window.addEventListener("pointerdown", closeOutside);
    window.addEventListener("resize", reposition);
    window.addEventListener("scroll", reposition, true);
    document.addEventListener("keydown", closeOnEscape, true);
    return () => {
      window.removeEventListener("pointerdown", closeOutside);
      window.removeEventListener("resize", reposition);
      window.removeEventListener("scroll", reposition, true);
      document.removeEventListener("keydown", closeOnEscape, true);
    };
  }, [open]);

  const run = (action: () => void | Promise<void>) => { setOpen(false); void action(); };
  return (
    <>
      <button ref={triggerRef} type="button" className="story-library-menu-trigger" aria-label={label} title={label} aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((value) => !value)}><MoreHorizontal size={17} /></button>
      {open && createPortal(
        <div ref={menuRef} className="story-row-menu" role="menu" style={position}>
          <button type="button" role="menuitem" onClick={() => run(onDetails)} disabled={busy}><Info size={14} />{t("details")}</button>
          <button type="button" role="menuitem" onClick={() => run(onExport)} disabled={busy}><FileDown size={14} />{t("exportStory")}</button>
          <button type="button" role="menuitem" onClick={() => run(onEdit)} disabled={busy}><Pencil size={14} />{t("edit")}</button>
          <button type="button" role="menuitem" onClick={() => run(onSetArchived)} disabled={busy}>{archived ? <ArchiveRestore size={14} /> : <Archive size={14} />}{archived ? t("restore") : t("archive")}</button>
          <button type="button" role="menuitem" className="danger-action" onClick={() => run(onDelete)} disabled={busy}><Trash2 size={14} />{t("delete")}</button>
        </div>,
        document.body,
      )}
    </>
  );
}

function storyDraft(story: StorySummary): Required<Pick<StoryUpdatePayload, "name" | "description" | "genre" | "tone" | "language">> {
  return { name: story.name || story.id, description: story.description || "", genre: story.genre || "", tone: story.tone || "", language: story.language || "" };
}
