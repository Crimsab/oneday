import { useCallback, useEffect, useMemo, useState } from "react";
import { actionModeToText, commandToAction, tabHotkeys } from "./commands";
import {
  ApiRequestError,
  createSave,
  deleteSave,
  getCommandDescriptors,
  getHealth,
  getSnapshot,
  getStories,
  loadSave,
  submitAction,
  submitMeta,
} from "./api";
import { Composer } from "./components/Composer";
import { Inspector } from "./components/Inspector";
import { LeftRail } from "./components/LeftRail";
import { PanelDrawer } from "./components/PanelDrawer";
import { RecentCommands } from "./components/RecentCommands";
import { SuggestedActions } from "./components/SuggestedActions";
import { TopBar } from "./components/TopBar";
import { Transcript } from "./components/Transcript";
import { recentFromMessages } from "./format";
import { stepHistoryIndex } from "./history";
import { clientId } from "./ids";
import { defaultPreferences, loadPreferences, savePreferences } from "./preferences";
import type {
  AppPreferences,
  ChoiceView,
  CommandDescriptor,
  MetaCommand,
  MetaResult,
  ModuleTab,
  OverlayKind,
  PlayerAction,
  RecentCommand,
  SaveView,
  StorySnapshot,
  StorySummary,
  SyncState,
} from "./types";

function App() {
  const [stories, setStories] = useState<StorySummary[]>([]);
  const [storyId, setStoryId] = useState("");
  const [snapshot, setSnapshot] = useState<StorySnapshot | null>(null);
  const [sync, setSync] = useState<SyncState>("Idle");
  const [healthText, setHealthText] = useState("Gateway starting");
  const [filter, setFilter] = useState("");
  const [selectedTab, setSelectedTab] = useState<ModuleTab>("history");
  const [overlay, setOverlay] = useState<OverlayKind>(null);
  const [saveFilter, setSaveFilter] = useState("");
  const [draft, setDraft] = useState("");
  const [mode, setMode] = useState("action");
  const [notice, setNotice] = useState("");
  const [metaResult, setMetaResult] = useState<MetaResult | null>(null);
  const [sending, setSending] = useState(false);
  const [paused, setPaused] = useState(false);
  const [hiddenBeforeMessageId, setHiddenBeforeMessageId] = useState(0);
  const [localCommands, setLocalCommands] = useState<RecentCommand[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [preferences, setPreferences] = useState<AppPreferences>(() => loadPreferences());
  const [commandDescriptors, setCommandDescriptors] = useState<CommandDescriptor[]>([]);

  const refreshHealth = useCallback(async () => {
    try {
      const health = await getHealth();
      setHealthText(`${health.stories} stories - Rust gateway`);
    } catch (error) {
      setHealthText(errorMessage(error));
    }
  }, []);

  const refreshStories = useCallback(async () => {
    setSync("Loading");
    try {
      const nextStories = await getStories();
      setStories(nextStories);
      setSync(storyId ? "Live" : "Idle");
      if (!storyId && nextStories[0]) setStoryId(nextStories[0].id);
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
    }
  }, [storyId]);

  const refreshCommandDescriptors = useCallback(async () => {
    try {
      setCommandDescriptors(await getCommandDescriptors());
    } catch {
      setCommandDescriptors([]);
    }
  }, []);

  const loadSnapshot = useCallback(async (nextStoryId = storyId) => {
    if (!nextStoryId) return;
    setSync("Loading");
    try {
      const nextSnapshot = await getSnapshot(nextStoryId);
      setSnapshot(nextSnapshot);
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
    }
  }, [paused, storyId]);

  useEffect(() => {
    void refreshHealth();
    void refreshCommandDescriptors();
    void refreshStories();
  }, [refreshCommandDescriptors, refreshHealth, refreshStories]);

  useEffect(() => {
    if (!storyId) return;
    setHiddenBeforeMessageId(0);
    void loadSnapshot(storyId);
  }, [loadSnapshot, storyId]);

  useEffect(() => {
    if (!storyId || paused) {
      if (storyId) setSync("Paused");
      return;
    }
    const source = new EventSource(`/api/stories/${encodeURIComponent(storyId)}/events`);
    source.addEventListener("open", () => setSync("Live"));
    source.addEventListener("snapshot", (event) => {
      setSnapshot(JSON.parse(event.data) as StorySnapshot);
      setSync("Live");
    });
    source.addEventListener("error", () => setSync("Reconnecting"));
    return () => source.close();
  }, [paused, storyId]);

  const filteredStories = useMemo(() => {
    const query = filter.trim().toLowerCase();
    if (!query) return stories;
    return stories.filter((story) => `${story.name} ${story.description} ${story.genre} ${story.tone}`.toLowerCase().includes(query));
  }, [filter, stories]);

  const recentCommands = useMemo(() => {
    const history = recentFromMessages(snapshot?.messages ?? []);
    const seen = new Set<string>();
    return [...localCommands, ...history].filter((command) => {
      const key = command.text.toLowerCase();
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
  }, [localCommands, snapshot?.messages]);

  useEffect(() => {
    savePreferences(preferences);
  }, [preferences]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const isEditing = target?.tagName === "INPUT" || target?.tagName === "TEXTAREA" || target?.tagName === "SELECT";
      if (isEditing) return;

      if (event.key === "?") {
        event.preventDefault();
        setOverlay("help");
        return;
      }
      if (event.key.toLowerCase() === "o") {
        event.preventDefault();
        setOverlay("options");
        return;
      }
      const tab = tabHotkeys[event.key.toLowerCase()];
      if (tab) {
        event.preventDefault();
        setSelectedTab(tab);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const selectStory = (nextStoryId: string) => {
    setStoryId(nextStoryId);
    setSnapshot(null);
    setNotice("");
  };

  const executeDraft = async () => {
    if (!draft.trim() || !snapshot || !storyId || sending) return;
    setNotice("");
    const commandResult = commandToAction(draft, {
      descriptors: commandDescriptors,
      npcNames: npcNamesFromSnapshot(snapshot),
    });
    if (commandResult.tab) setSelectedTab(commandResult.tab);
    if (commandResult.overlay) setOverlay(commandResult.overlay);
    if (commandResult.saveFilter !== undefined) setSaveFilter(commandResult.saveFilter);
    if (commandResult.saveDeleteFilter !== undefined) setSaveFilter(commandResult.saveDeleteFilter);
    if (commandResult.notice) setNotice(commandResult.notice);
    if (commandResult.meta) {
      await sendMetaCommand(commandResult.meta, draft);
      setDraft("");
      setHistoryIndex(-1);
      return;
    }
    if (commandResult.saveName !== undefined) {
      await createManualSave(commandResult.saveName, draft);
      setDraft("");
      setHistoryIndex(-1);
      return;
    }
    if (commandResult.handled) {
      setDraft("");
      return;
    }

    const text = commandResult.text ?? actionModeToText(mode, draft);
    if (!text.trim()) return;
    await sendAction({ kind: "free_text", text }, draft);
    setDraft("");
    setHistoryIndex(-1);
  };

  const sendChoice = async (choice: ChoiceView) => {
    await sendAction(
      {
        kind: "choice",
        choice_id: choice.id,
        text: `[Choice ${choice.id}] ${choice.text}`,
      },
      choice.text,
    );
  };

  const sendMetaCommand = async (meta: MetaCommand, sourceText: string) => {
    if (!snapshot || !storyId || sending) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return;
      setSync("Sending");
      const response = await submitMeta(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: readySnapshot.world.current_turn,
        kind: meta.kind,
        text: meta.text,
      });
      setSnapshot(response.snapshot);
      if (response.meta) {
        setMetaResult(response.meta);
        setOverlay("meta");
        setNotice(`${response.meta.title} answered.`);
      } else {
        setNotice("Meta command completed.");
      }
      setLocalCommands((items) => [
        { id: clientId("command"), text: sourceText.trim(), turn: response.snapshot.world.current_turn, source: "browser" as const },
        ...items,
      ].slice(0, 10));
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setSending(false);
    }
  };

  const createManualSave = async (name: string, sourceText = "") => {
    if (!snapshot || !storyId || sending) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return;
      setSync("Sending");
      const saveName = name.trim() || `Browser Save T${readySnapshot.world.current_turn}`;
      const response = await createSave(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: readySnapshot.world.current_turn,
        name: saveName,
        kind: "manual",
      });
      setSnapshot(response.snapshot);
      setSelectedTab("saves");
      setOverlay("saves");
      setSaveFilter("");
      setNotice(`Saved ${response.save?.name ?? saveName}.`);
      setLocalCommands((items) => [
        { id: clientId("command"), text: sourceText.trim() || `/save ${saveName}`, turn: response.snapshot.world.current_turn, source: "browser" as const },
        ...items,
      ].slice(0, 10));
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setSending(false);
    }
  };

  const loadManualSave = async (save: SaveView) => {
    if (!snapshot || !storyId || sending) return;
    const confirmed = window.confirm(`Load "${save.name}" from turn ${save.turn}? Current progress will roll back to that snapshot.`);
    if (!confirmed) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return;
      setSync("Sending");
      const response = await loadSave(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: readySnapshot.world.current_turn,
        save_id: save.id,
      });
      setSnapshot(response.snapshot);
      setSelectedTab("saves");
      setNotice(`${response.legacy ? "Legacy save loaded" : "Loaded"} ${response.save?.name ?? save.name}.`);
      setLocalCommands((items) => [
        { id: clientId("command"), text: `/load ${save.name}`, turn: response.snapshot.world.current_turn, source: "browser" as const },
        ...items,
      ].slice(0, 10));
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setSending(false);
    }
  };

  const deleteManualSave = async (save: SaveView) => {
    if (!snapshot || !storyId || sending) return;
    const confirmed = window.confirm(`Delete "${save.name}" from turn ${save.turn}? This removes only the saved snapshot.`);
    if (!confirmed) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return;
      setSync("Sending");
      const response = await deleteSave(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: readySnapshot.world.current_turn,
        save_id: save.id,
      });
      setSnapshot(response.snapshot);
      setSelectedTab("saves");
      setOverlay("saves");
      setNotice(`Deleted ${response.save?.name ?? save.name}.`);
      setLocalCommands((items) => [
        { id: clientId("command"), text: `/delete-save ${save.name}`, turn: response.snapshot.world.current_turn, source: "browser" as const },
        ...items,
      ].slice(0, 10));
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setSending(false);
    }
  };

  const sendAction = async (action: PlayerAction, sourceText: string) => {
    if (!snapshot || !storyId || sending) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return;
      setSync("Sending");
      const currentTurn = readySnapshot.world.current_turn;
      const response = await submitAction(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: currentTurn,
        idempotency_key: clientId("turn"),
        action,
        capabilities: { images: true, ascii: true, roll_log: true },
      });
      setSnapshot(response.snapshot);
      setLocalCommands((items) => [
        { id: clientId("command"), text: sourceText.trim(), turn: response.snapshot.world.current_turn, source: "browser" as const },
        ...items,
      ].slice(0, 10));
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setSending(false);
    }
  };

  const snapshotForSubmit = async (baseSnapshot: StorySnapshot): Promise<StorySnapshot | null> => {
    if (!paused) return baseSnapshot;
    setSync("Loading");
    const latest = await getSnapshot(storyId);
    setSnapshot(latest);
    if (
      latest.active_session.id !== baseSnapshot.active_session.id ||
      latest.world.current_turn !== baseSnapshot.world.current_turn ||
      latest.version.last_message_id !== baseSnapshot.version.last_message_id
    ) {
      setSync("Paused");
      setNotice("The story changed while sync was paused. Review the latest turn before sending.");
      return null;
    }
    return latest;
  };

  const clearTranscript = () => {
    const lastId = snapshot?.messages.at(-1)?.id ?? 0;
    setHiddenBeforeMessageId(lastId);
  };

  const openOverlay = (nextOverlay: OverlayKind) => {
    if (nextOverlay === "saves") {
      setSelectedTab("saves");
      setSaveFilter("");
    }
    setOverlay(nextOverlay);
  };

  const updatePreferences = (nextPreferences: AppPreferences) => {
    setPreferences({ ...defaultPreferences, ...nextPreferences });
  };

  const stepCommandHistory = (direction: -1 | 1): string | null => {
    const next = stepHistoryIndex(historyIndex, direction, recentCommands);
    setHistoryIndex(next.index);
    return next.value;
  };

  return (
    <div
      className={`app-shell ${preferences.showInspector ? "" : "inspector-hidden"} ${preferences.wrapTranscript ? "" : "transcript-nowrap"}`}
      data-density={preferences.density}
      data-font-size={preferences.fontSize}
      data-accent={preferences.accent}
    >
      <TopBar snapshot={snapshot} sync={sync} onOpen={openOverlay} />
      <div className="workspace">
        <LeftRail
          stories={filteredStories}
          activeStoryId={storyId}
          filter={filter}
          snapshot={snapshot}
          selectedTab={selectedTab}
          healthText={healthText}
          onFilterChange={setFilter}
          onSelectStory={selectStory}
          onSelectTab={setSelectedTab}
          onRefreshStories={refreshStories}
          onOpen={openOverlay}
        />
        <main className="center-stage">
          <section className="transcript-panel">
            <div className="panel-head">
              <h1>Narrative Transcript</h1>
              <div className="panel-actions">
                <button type="button" onClick={() => setPaused((value) => !value)}>
                  {paused ? "Resume" : "Pause"} <span>{paused ? ">" : "II"}</span>
                </button>
                <button type="button" onClick={clearTranscript}>Clear</button>
              </div>
            </div>
            <Transcript messages={snapshot?.messages ?? []} hiddenBeforeId={hiddenBeforeMessageId} />
          </section>

          <Composer
            draft={draft}
            mode={mode}
            disabled={sending || !snapshot}
            notice={notice}
            commandDescriptors={commandDescriptors}
            onDraftChange={setDraft}
            onModeChange={setMode}
            onSubmit={executeDraft}
            onHistoryStep={stepCommandHistory}
          />

          <section className="lower-grid">
            <section className="suggested-panel">
              <div className="panel-head compact">
                <h2>Suggested Actions</h2>
              </div>
              <SuggestedActions choices={snapshot?.choices ?? []} snapshot={snapshot} onChoice={sendChoice} onDraft={setDraft} />
            </section>
            <section className="recent-panel">
              <div className="panel-head compact">
                <h2>Recent Commands</h2>
              </div>
              <RecentCommands commands={recentCommands} onDraft={setDraft} />
            </section>
          </section>
        </main>
        {preferences.showInspector && <Inspector snapshot={snapshot} selectedTab={selectedTab} onRefresh={() => void loadSnapshot()} />}
      </div>
      <PanelDrawer
        overlay={overlay}
        snapshot={snapshot}
        preferences={preferences}
        metaResult={metaResult}
        commandDescriptors={commandDescriptors}
        busy={sending}
        onClose={() => setOverlay(null)}
        onPreferencesChange={updatePreferences}
        onCreateSave={(name) => void createManualSave(name, `/save ${name}`)}
        onLoadSave={(save) => void loadManualSave(save)}
        onDeleteSave={(save) => void deleteManualSave(save)}
        saveFilter={saveFilter}
        onSaveFilterChange={setSaveFilter}
      />
    </div>
  );
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function actionErrorMessage(error: unknown): string {
  if (error instanceof ApiRequestError && error.status === 409) {
    return "The story advanced elsewhere. Review the latest choices before sending again.";
  }
  const message = errorMessage(error);
  if (/stale|session/i.test(message)) {
    return "The story advanced elsewhere. Review the latest choices before sending again.";
  }
  return message;
}

function npcNamesFromSnapshot(snapshot: StorySnapshot): string[] {
  return snapshot.panels.npcs.map((npc) => npc.name).filter(Boolean);
}

export default App;
