import { useCallback, useEffect, useMemo, useState } from "react";
import { commandToAction, actionModeToText } from "./commands";
import { getHealth, getSnapshot, getStories, submitAction } from "./api";
import { Composer } from "./components/Composer";
import { Inspector } from "./components/Inspector";
import { LeftRail } from "./components/LeftRail";
import { PanelDrawer } from "./components/PanelDrawer";
import { RecentCommands } from "./components/RecentCommands";
import { SuggestedActions } from "./components/SuggestedActions";
import { TopBar } from "./components/TopBar";
import { Transcript } from "./components/Transcript";
import { recentFromMessages } from "./format";
import type { ChoiceView, ModuleTab, OverlayKind, PlayerAction, RecentCommand, StorySnapshot, StorySummary, SyncState } from "./types";

function App() {
  const [stories, setStories] = useState<StorySummary[]>([]);
  const [storyId, setStoryId] = useState("");
  const [snapshot, setSnapshot] = useState<StorySnapshot | null>(null);
  const [sync, setSync] = useState<SyncState>("Idle");
  const [healthText, setHealthText] = useState("Gateway starting");
  const [filter, setFilter] = useState("");
  const [selectedTab, setSelectedTab] = useState<ModuleTab>("history");
  const [overlay, setOverlay] = useState<OverlayKind>(null);
  const [draft, setDraft] = useState("");
  const [mode, setMode] = useState("action");
  const [notice, setNotice] = useState("");
  const [sending, setSending] = useState(false);
  const [paused, setPaused] = useState(false);
  const [hiddenBeforeMessageId, setHiddenBeforeMessageId] = useState(0);
  const [localCommands, setLocalCommands] = useState<RecentCommand[]>([]);

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
    void refreshStories();
  }, [refreshHealth, refreshStories]);

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

  const selectStory = (nextStoryId: string) => {
    setStoryId(nextStoryId);
    setSnapshot(null);
    setNotice("");
  };

  const executeDraft = async () => {
    if (!draft.trim() || !snapshot || !storyId || sending) return;
    setNotice("");
    const commandResult = commandToAction(draft);
    if (commandResult.tab) setSelectedTab(commandResult.tab);
    if (commandResult.overlay) setOverlay(commandResult.overlay);
    if (commandResult.notice) setNotice(commandResult.notice);
    if (commandResult.handled) {
      setDraft("");
      return;
    }

    const text = commandResult.text ?? actionModeToText(mode, draft);
    if (!text.trim()) return;
    await sendAction({ kind: "free_text", text }, draft);
    setDraft("");
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

  const sendAction = async (action: PlayerAction, sourceText: string) => {
    if (!snapshot || !storyId || sending) return;
    setSending(true);
    setSync("Sending");
    const currentTurn = snapshot.world.current_turn;
    try {
      const response = await submitAction(storyId, {
        session_id: snapshot.active_session.id,
        client_turn: currentTurn,
        idempotency_key: crypto.randomUUID(),
        action,
        capabilities: { images: true, ascii: true, roll_log: true },
      });
      setSnapshot(response.snapshot);
      setLocalCommands((items) => [
        { id: crypto.randomUUID(), text: sourceText.trim(), turn: currentTurn, source: "browser" as const },
        ...items,
      ].slice(0, 10));
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setSending(false);
    }
  };

  const clearTranscript = () => {
    const lastId = snapshot?.messages.at(-1)?.id ?? 0;
    setHiddenBeforeMessageId(lastId);
  };

  const openOverlay = (nextOverlay: OverlayKind) => {
    if (nextOverlay === "saves") setSelectedTab("saves");
    setOverlay(nextOverlay);
  };

  return (
    <div className="app-shell">
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
            onDraftChange={setDraft}
            onModeChange={setMode}
            onSubmit={executeDraft}
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
        <Inspector snapshot={snapshot} onRefresh={() => void loadSnapshot()} />
      </div>
      <PanelDrawer overlay={overlay} snapshot={snapshot} onClose={() => setOverlay(null)} onDraft={setDraft} />
    </div>
  );
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export default App;
