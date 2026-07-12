import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { actionFingerprint, resolvePendingActionIdentity, type PendingActionIdentity } from "./actionIdentity";
import { actionModeToText, commandToAction, tabHotkeys } from "./commands";
import {
  ApiRequestError,
  cancelVisualGenerationJob,
  cleanupVisualAssetFiles,
  createSave,
  deleteStory,
  deleteSave,
  enhanceStoryText,
  generateVisualAssets,
  getCommandDescriptors,
  getHealth,
  getModelSettings,
  getActiveMiniGame,
  getSnapshot,
	getTimeline,
  getStoryDeletePlan,
  getStories,
  getVisualAssets,
  getVisualAssetVersions,
  inputMiniGame,
  loadSave,
  runStoryWizard,
  selectVisualAssetVersion,
  stepVisualAssetSelection,
  submitAction,
  submitMeta,
  updateStory,
	updateTimeline,
  updateModelSettings,
  updateVisualAssetPrompt,
  updateVisualProfile,
} from "./api";
import { Composer } from "./components/Composer";
import { Inspector } from "./components/Inspector";
import { MiniGameHost } from "./components/MiniGameHost";
import { LeftRail } from "./components/LeftRail";
import { PanelDrawer } from "./components/PanelDrawer";
import { StoryPath } from "./components/StoryPath";
import { SuggestedActions } from "./components/SuggestedActions";
import { TopBar } from "./components/TopBar";
import { Transcript } from "./components/Transcript";
import { recentFromMessages } from "./format";
import { stepHistoryIndex } from "./history";
import { restoreFailedDraft } from "./draftLifecycle";
import { clientId } from "./ids";
import { defaultPreferences, loadPreferences, savePreferences } from "./preferences";
import {
  isVisualAssetTurnEvent,
  parseStorySnapshotEvent,
  shouldSuppressStreamingDelta,
  streamingDeltaText,
  turnEventDetail,
} from "./turnEvents";
import type {
  AppPreferences,
  ChoiceView,
  CommandDescriptor,
  MetaCommand,
  MetaResult,
  ModelSettings,
  ModelSettingsUpdate,
  MiniGameInput,
  MiniGameInstance,
  MiniGameKind,
  ModuleTab,
  OverlayKind,
  PendingTurnView,
  PlayerAction,
  RecentCommand,
  SaveView,
  StoryEnhanceEnvelope,
  StoryEnhanceResponse,
  StorySnapshot,
  StorySummary,
  StoryUpdatePayload,
  StoryWizardEnvelope,
  StoryWizardResponse,
	TimelineResponse,
  SyncState,
  TurnStreamEvent,
  VisualAssetsResponse,
  GenerateVisualAssetsRequest,
  VisualAssetPromptUpdate,
  VisualAssetVersion,
  VisualProfileUpdate,
} from "./types";
import { visualCatalog } from "./visualAssets";
import { visualPollingDelayMs } from "./visualJobs";

const deepLinkOverlays = new Set<OverlayKind>(["help", "options", "saves", "new-story", "meta", "module"]);

function initialOverlayFromLocation(): OverlayKind {
  if (typeof window === "undefined") return null;
  const overlay = new URLSearchParams(window.location.search).get("overlay") as OverlayKind;
  return deepLinkOverlays.has(overlay) ? overlay : null;
}

function App() {
  const [stories, setStories] = useState<StorySummary[]>([]);
  const [storyId, setStoryId] = useState("");
  const [snapshot, setSnapshot] = useState<StorySnapshot | null>(null);
  const [sync, setSync] = useState<SyncState>("Idle");
  const [healthText, setHealthText] = useState("Gateway starting");
  const [filter, setFilter] = useState("");
  const [selectedTab, setSelectedTab] = useState<ModuleTab>("history");
  const [moduleFocusId, setModuleFocusId] = useState<string | null>(null);
  const [moduleOverlayTab, setModuleOverlayTab] = useState<ModuleTab | null>(null);
  const [overlay, setOverlay] = useState<OverlayKind>(() => initialOverlayFromLocation());
  const [saveFilter, setSaveFilter] = useState("");
  const [draft, setDraft] = useState("");
  const [mode, setMode] = useState("action");
  const [notice, setNotice] = useState("");
  const [metaResult, setMetaResult] = useState<MetaResult | null>(null);
  const [sending, setSending] = useState(false);
  const [pendingTurn, setPendingTurn] = useState<PendingTurnView | null>(null);
  const [paused, setPaused] = useState(false);
  const [hiddenBeforeMessageId, setHiddenBeforeMessageId] = useState(0);
  const [localCommands, setLocalCommands] = useState<RecentCommand[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [preferences, setPreferences] = useState<AppPreferences>(() => loadPreferences());
  const [commandDescriptors, setCommandDescriptors] = useState<CommandDescriptor[]>([]);
  const [modelSettings, setModelSettings] = useState<ModelSettings | null>(null);
  const [modelSettingsError, setModelSettingsError] = useState("");
  const [modelSaving, setModelSaving] = useState(false);
  const [visualAssets, setVisualAssets] = useState<VisualAssetsResponse | null>(null);
  const [visualAssetsError, setVisualAssetsError] = useState("");
  const [visualProfileSaving, setVisualProfileSaving] = useState(false);
  const [visualGenerationBusy, setVisualGenerationBusy] = useState(false);
  const [visualAssetFocusId, setVisualAssetFocusId] = useState<string | null>(null);
  const [activeMiniGame, setActiveMiniGame] = useState<MiniGameInstance | null>(null);
  const [miniGameBusy, setMiniGameBusy] = useState(false);
  const [miniGameError, setMiniGameError] = useState("");
  const [storyMutatingId, setStoryMutatingId] = useState("");
	const [timeline, setTimeline] = useState<TimelineResponse | null>(null);
  const pendingActionIdentity = useRef<PendingActionIdentity | null>(null);
  const actionSubmitInFlight = useRef(false);

  const refreshHealth = useCallback(async () => {
    try {
      const health = await getHealth();
      setHealthText(`${health.stories} ${health.stories === 1 ? "story" : "stories"}`);
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
      return nextStories;
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
      return [] as StorySummary[];
    }
  }, [storyId]);

  const refreshCommandDescriptors = useCallback(async () => {
    try {
      setCommandDescriptors(await getCommandDescriptors());
    } catch {
      setCommandDescriptors([]);
    }
  }, []);

  const refreshModelSettings = useCallback(async () => {
    try {
      setModelSettings(await getModelSettings());
      setModelSettingsError("");
    } catch (error) {
      setModelSettings(null);
      setModelSettingsError(errorMessage(error));
    }
  }, []);

  const refreshVisualAssets = useCallback(async (nextStoryId = storyId) => {
    if (!nextStoryId) {
      setVisualAssets(null);
      return;
    }
    try {
      const nextAssets = await getVisualAssets(nextStoryId);
      setVisualAssets(nextAssets);
      setVisualAssetsError("");
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
    }
  }, [storyId]);

  const refreshMiniGame = useCallback(async (nextStoryId = storyId) => {
    if (!nextStoryId) {
      setActiveMiniGame(null);
      return;
    }
    try {
      const response = await getActiveMiniGame(nextStoryId);
      setActiveMiniGame(response.instance ?? null);
      setMiniGameError("");
    } catch (error) {
      setMiniGameError(errorMessage(error));
    }
  }, [storyId]);

  const loadSnapshot = useCallback(async (nextStoryId = storyId) => {
    if (!nextStoryId) return;
    setSync("Loading");
    try {
      const nextSnapshot = await getSnapshot(nextStoryId);
      setSnapshot(nextSnapshot);
      setSync(paused ? "Paused" : "Live");
      void refreshVisualAssets(nextStoryId);
      void refreshMiniGame(nextStoryId);
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
    }
  }, [paused, refreshMiniGame, refreshVisualAssets, storyId]);

  useEffect(() => {
    void refreshHealth();
    void refreshCommandDescriptors();
    void refreshModelSettings();
    void refreshStories();
  }, [refreshCommandDescriptors, refreshHealth, refreshModelSettings, refreshStories]);

  useEffect(() => {
    if (!storyId) return;
	setTimeline(null);
    setHiddenBeforeMessageId(0);
    void loadSnapshot(storyId);
	void getTimeline(storyId).then(setTimeline).catch((error) => setNotice(errorMessage(error)));
  }, [loadSnapshot, storyId]);

	const mutateTimeline = async (payload: Parameters<typeof updateTimeline>[1]) => {
		if (!storyId || !snapshot || storyMutatingId) return;
		setStoryMutatingId(storyId);
		try {
			const response = await updateTimeline(storyId, payload);
			setTimeline(response.timeline);
			setSnapshot(response.snapshot);
			setPendingTurn(null);
			void refreshMiniGame(storyId);
			pendingActionIdentity.current = null;
			setNotice(`Active branch: ${response.timeline.branches.find((branch) => branch.id === response.timeline.active_branch_id)?.name ?? "updated"}.`);
		} catch (error) {
			setNotice(actionErrorMessage(error));
			await loadSnapshot().catch(() => undefined);
		} finally { setStoryMutatingId(""); }
	};

	const forkBranch = (name:string) => timeline?.head ? mutateTimeline({action:"fork",client_revision:snapshot?.version.revision ?? timeline.revision,from_commit_id:timeline.head.id,name}) : Promise.resolve();
	const renameBranch = (branchId:string,name:string) => mutateTimeline({action:"rename",client_revision:snapshot?.version.revision ?? timeline?.revision ?? 0,branch_id:branchId,name});
  const checkoutBranch = (branchId:string) => mutateTimeline({action:"checkout",client_revision:snapshot?.version.revision ?? timeline?.revision ?? 0,branch_id:branchId});

  const restoreDecision = async (fromCommitId: string, turn: number) => {
    if (!storyId || !snapshot || !timeline || !fromCommitId || storyMutatingId) return;
    const siblingCount = timeline.branches.filter((branch) => branch.fork_commit_id === fromCommitId).length;
    const name = `Turn ${Math.max(0, turn)} alternative ${siblingCount + 1}`;
    setStoryMutatingId(storyId);
    try {
      const forked = await updateTimeline(storyId, {
        action: "fork",
        client_revision: snapshot.version.revision,
        from_commit_id: fromCommitId,
        name,
      });
      const branch = forked.timeline.branches.find((item) => item.fork_commit_id === fromCommitId && item.name === name);
      if (!branch) throw new Error("The alternative branch was created but could not be selected.");
      const checkedOut = await updateTimeline(storyId, {
        action: "checkout",
        client_revision: forked.timeline.revision,
        branch_id: branch.id,
      });
      setTimeline(checkedOut.timeline);
      setSnapshot(checkedOut.snapshot);
      setPendingTurn(null);
      pendingActionIdentity.current = null;
      setNotice(`Back at the previous decision on ${name}.`);
    } catch (error) {
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
      void getTimeline(storyId).then(setTimeline).catch(() => undefined);
    } finally {
      setStoryMutatingId("");
    }
  };

  useEffect(() => {
    if (!storyId || paused) {
      if (storyId) setSync("Paused");
      return;
    }
    const source = new EventSource(`/api/stories/${encodeURIComponent(storyId)}/events`);
    source.addEventListener("open", () => setSync("Live"));
    source.addEventListener("snapshot", (event) => {
      const nextSnapshot = parseStorySnapshotEvent(event.data);
      if (!nextSnapshot) {
        setNotice("Received an unreadable live snapshot; keeping the current story state.");
        setSync("Reconnecting");
        return;
      }
      setSnapshot(nextSnapshot);
      void refreshVisualAssets(storyId);
      void getTimeline(storyId).then(setTimeline).catch(() => undefined);
      setSync("Live");
    });
    source.addEventListener("turn", (event) => {
      let liveEvent: TurnStreamEvent;
      try {
        liveEvent = JSON.parse(event.data) as TurnStreamEvent;
      } catch {
        setNotice("Received an unreadable live turn event.");
        return;
      }
      if (isVisualAssetTurnEvent(liveEvent)) {
        void refreshVisualAssets(storyId);
        setSync(paused ? "Paused" : "Live");
        return;
      }
      setPendingTurn((pending) => {
        if (!pending) return pending;
        const delta = streamingDeltaText(liveEvent);
        if (!delta) return { ...pending, detail: turnEventDetail(liveEvent) };
        if (pending.streamingSuppressed || shouldSuppressStreamingDelta(pending.streamingText, delta)) {
          return {
            ...pending,
			detail: "Preparing the final story response...",
            streamingText: undefined,
            streamingSuppressed: true,
          };
        }
        return {
          ...pending,
          detail: "Assistant draft streaming. This text becomes canonical only after commit.",
          streamingText: `${pending.streamingText ?? ""}${delta}`,
        };
      });
      if (liveEvent.status === "failed") {
        setSync("Error");
        setNotice(liveEvent.message);
        return;
      }
      if (liveEvent.status === "completed" || liveEvent.status === "snapshot_changed") {
        pendingActionIdentity.current = null;
        setSync(paused ? "Paused" : "Live");
        return;
      }
      if (liveEvent.status === "submitted" || liveEvent.status === "event") {
        setSync(paused ? "Paused" : "Sending");
      }
    });
    source.addEventListener("error", () => setSync("Reconnecting"));
    return () => source.close();
  }, [paused, refreshVisualAssets, storyId]);

  useEffect(() => {
    if (!storyId) return;
    const delay = visualPollingDelayMs(visualAssets);
    if (!delay) return;
    let inFlight = false;
    const timer = window.setInterval(() => {
      if (inFlight) return;
      inFlight = true;
      refreshVisualAssets(storyId)
        .catch(() => undefined)
        .finally(() => {
          inFlight = false;
        });
    }, delay);
    return () => window.clearInterval(timer);
  }, [refreshVisualAssets, storyId, visualAssets]);

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

  const commandContext = useMemo(
    () => ({
      npcNames: snapshot ? npcNamesFromSnapshot(snapshot) : [],
      saveNames: snapshot ? saveNamesFromSnapshot(snapshot) : [],
      recentCommands: recentCommands.map((command) => command.text),
      visiblePrivateThoughts: false,
    }),
    [recentCommands, snapshot],
  );

  useEffect(() => {
    savePreferences(preferences);
  }, [preferences]);

  const selectModuleTab = useCallback((tab: ModuleTab) => {
    setModuleFocusId(null);
    setModuleOverlayTab(null);
    setSelectedTab(tab);
    if (window.matchMedia("(max-width: 1240px)").matches) {
      setModuleOverlayTab(tab);
      setOverlay("module");
    } else {
      setPreferences((value) => ({
        ...defaultPreferences,
        ...value,
        showLeftRail: window.matchMedia("(max-width: 1600px)").matches ? false : value.showLeftRail,
        showInspector: true,
      }));
    }
  }, []);

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
      if (event.key === "[") {
        event.preventDefault();
        setPreferences((value) => ({ ...defaultPreferences, ...value, showLeftRail: !value.showLeftRail }));
        return;
      }
      if (event.key === "]") {
        event.preventDefault();
        setPreferences((value) => ({ ...defaultPreferences, ...value, showInspector: !value.showInspector }));
        return;
      }
      const tab = tabHotkeys[event.key.toLowerCase()];
      if (tab) {
        event.preventDefault();
        selectModuleTab(tab);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [selectModuleTab]);

  const selectStory = (nextStoryId: string) => {
    if (nextStoryId === storyId) {
      void loadSnapshot(nextStoryId);
      return;
    }
    setStoryId(nextStoryId);
    setSnapshot(null);
    setVisualAssets(null);
    setVisualAssetsError("");
    pendingActionIdentity.current = null;
    setNotice("");
  };

  const handleUpdateStory = async (targetStoryId: string, payload: StoryUpdatePayload) => {
    if (!targetStoryId || storyMutatingId) return;
    setStoryMutatingId(targetStoryId);
    setSync("Saving");
    try {
      const updated = await updateStory(targetStoryId, payload);
      setStories((items) => items.map((story) => (story.id === targetStoryId ? updated : story)));
      if (targetStoryId === storyId) {
        await loadSnapshot(targetStoryId);
      }
      setNotice(updated.is_archived ? `Archived ${updated.name}.` : `Updated ${updated.name}.`);
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
    } finally {
      setStoryMutatingId("");
    }
  };

  const handleSetStoryArchived = async (targetStoryId: string, archived: boolean) => {
    await handleUpdateStory(targetStoryId, { is_archived: archived });
  };

  const handleDeleteStory = async (targetStoryId: string) => {
    if (!targetStoryId || storyMutatingId) return;
    setStoryMutatingId(targetStoryId);
    setSync("Saving");
    try {
      const plan = await getStoryDeletePlan(targetStoryId);
      const topCounts = plan.counts
        .filter((count) => count.rows > 0)
        .slice(0, 6)
        .map((count) => `${count.table}: ${count.rows}`)
        .join("\n");
      const retainedFiles = plan.retained_asset_files.length;
      const message = [
        `Delete "${plan.story_name || targetStoryId}"?`,
        "",
        `Database rows affected: ${plan.total_rows}`,
        topCounts,
        retainedFiles ? `Generated image files retained on disk: ${retainedFiles}` : "No generated image files are linked.",
      ]
        .filter(Boolean)
        .join("\n");
      if (!window.confirm(message)) {
        setSync(paused ? "Paused" : "Live");
        return;
      }
      await deleteStory(targetStoryId);
      const nextStories = await getStories();
      setStories(nextStories);
      void refreshHealth();
      if (targetStoryId === storyId) {
        const nextActive = nextStories.find((story) => !story.is_archived) ?? nextStories[0] ?? null;
        setStoryId(nextActive?.id ?? "");
        setSnapshot(null);
        setVisualAssets(null);
        setVisualAssetsError("");
        pendingActionIdentity.current = null;
        if (nextActive) {
          await loadSnapshot(nextActive.id);
        } else {
          setSync("Idle");
        }
      } else {
        setSync(paused ? "Paused" : "Live");
      }
      setNotice("Story deleted.");
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
    } finally {
      setStoryMutatingId("");
    }
  };

  const executeDraft = async (draftOverride?: string) => {
    const currentDraft = draftOverride ?? draft;
    if (!currentDraft.trim() || !snapshot || !storyId || sending || actionSubmitInFlight.current) return;
    setNotice("");
    const sourceText = currentDraft.trim();
    const commandResult = commandToAction(currentDraft, {
      descriptors: commandDescriptors,
      npcNames: npcNamesFromSnapshot(snapshot),
      saveNames: saveNamesFromSnapshot(snapshot),
      visiblePrivateThoughts: false,
    });
    if (commandResult.tab) {
      selectModuleTab(commandResult.tab);
      if (window.matchMedia("(max-width: 1240px)").matches) {
        setModuleOverlayTab(commandResult.tab);
        setOverlay("module");
      }
    }
    if (commandResult.overlay) setOverlay(commandResult.overlay);
    if (commandResult.saveFilter !== undefined) setSaveFilter(commandResult.saveFilter);
    if (commandResult.saveDeleteFilter !== undefined) setSaveFilter(commandResult.saveDeleteFilter);
    if (commandResult.notice) setNotice(commandResult.notice);
	if (commandResult.timeline) {
		const details = document.querySelector<HTMLDetailsElement>(".branch-navigator");
		if (details) { details.open = true; details.querySelector<HTMLElement>("summary")?.focus(); }
		const value = commandResult.timeline.value?.trim() ?? "";
		if (commandResult.timeline.action === "list") setNotice("Story branch navigator opened in the sidebar.");
		if (commandResult.timeline.action === "fork") value ? await forkBranch(value) : setNotice("Usage: /fork <branch name>");
		if (commandResult.timeline.action === "rename") value ? await renameBranch(timeline?.active_branch_id ?? "", value) : setNotice("Usage: /branch-rename <name>");
		if (commandResult.timeline.action === "checkout") {
			const target = timeline?.branches.find((branch) => branch.id === value || branch.name.toLowerCase() === value.toLowerCase());
			target ? await checkoutBranch(target.id) : setNotice(`Branch not found: ${value || "(missing name)"}`);
		}
		setDraft(""); setHistoryIndex(-1); return;
	}
    if (commandResult.meta) {
      await sendMetaCommand(commandResult.meta, currentDraft);
      setDraft("");
      setHistoryIndex(-1);
      return;
    }
    if (commandResult.saveName !== undefined) {
      await createManualSave(commandResult.saveName, currentDraft);
      setDraft("");
      setHistoryIndex(-1);
      return;
    }
    if (commandResult.handled) {
      rememberLocalCommand(sourceText, snapshot.world.current_turn);
      setDraft("");
      setHistoryIndex(-1);
      return;
    }

    const text = commandResult.text ?? actionModeToText(mode, currentDraft);
    if (!text.trim()) return;
	setDraft("");
    setHistoryIndex(-1);
	const sent = await sendAction({ kind: "free_text", text }, currentDraft);
	if (!sent) setDraft((current) => restoreFailedDraft(current, currentDraft));
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

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const isEditing = target?.tagName === "INPUT" || target?.tagName === "TEXTAREA" || target?.tagName === "SELECT";
      if (isEditing || sending || !snapshot || !/^[1-6]$/.test(event.key)) return;

      const keyNumber = Number.parseInt(event.key, 10);
      const choice = snapshot.choices.find((item) => item.id === keyNumber) ?? snapshot.choices[keyNumber - 1];
      if (!choice) return;

      event.preventDefault();
      void sendChoice(choice);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [sending, snapshot]);

  const sendMetaCommand = async (meta: MetaCommand, sourceText: string) => {
    if (!snapshot || !storyId || sending) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
	  if (!readySnapshot) return false;
      setSync("Sending");
      const response = await submitMeta(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: readySnapshot.world.current_turn,
        client_revision: readySnapshot.version.revision,
        kind: meta.kind,
        text: meta.text,
      });
      setSnapshot(response.snapshot);
	  void refreshMiniGame(storyId);
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

  const sendMiniGameInput = async (input: MiniGameInput) => {
    if (!storyId || !activeMiniGame || miniGameBusy || sending) return;
    setMiniGameBusy(true);
    setMiniGameError("");
    try {
      const response = await inputMiniGame(storyId, activeMiniGame.id, input);
      const instance = response.instance ?? null;
      setActiveMiniGame(instance);
      const result = instance?.runtime.result;
      if (instance?.runtime.phase === "resolved" && result) {
        const degree = result.outcome?.degree ?? (result.passed ? "full_success" : "hard_failure");
        const continuation = `[Challenge Result: mini_game ${degree.toUpperCase()} - ${result.detail}]`;
        const continued = await sendAction({ kind: "free_text", text: continuation }, "Challenge resolved");
        if (continued) {
          setActiveMiniGame(null);
          await refreshMiniGame(storyId);
        } else {
          setMiniGameError("The challenge was resolved, but narrative continuation needs a retry.");
        }
      }
    } catch (error) {
      setMiniGameError(errorMessage(error));
    } finally {
      setMiniGameBusy(false);
    }
  };

  const createManualSave = async (name: string, sourceText = "") => {
    if (!snapshot || !storyId || sending) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return false;
      setSync("Sending");
      const saveName = name.trim() || `Browser Save T${readySnapshot.world.current_turn}`;
      const response = await createSave(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: readySnapshot.world.current_turn,
        client_revision: readySnapshot.version.revision,
        name: saveName,
        kind: "manual",
      });
      setSnapshot(response.snapshot);
      selectModuleTab("saves");
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

  const runBrowserStoryWizard = async (payload: StoryWizardEnvelope): Promise<StoryWizardResponse> => {
    if (sending) throw new Error("Another OneDay request is already running.");
    setSending(true);
    setNotice("");
    setSync("Sending");
    try {
      const response = await runStoryWizard(payload);
      if (response.snapshot) {
        setSnapshot(response.snapshot);
        setStoryId(response.snapshot.story.id);
        setVisualAssets(null);
        setStories((items) => [response.snapshot!.story, ...items.filter((story) => story.id !== response.snapshot!.story.id)]);
        selectModuleTab("history");
        setOverlay(null);
        setHiddenBeforeMessageId(0);
        pendingActionIdentity.current = null;
        const startNotice = response.wizard.start_error
          ? ` created, but first turn did not start: ${response.wizard.start_error}`
          : ` created${response.wizard.started ? " and started" : ""}.`;
        setNotice(`${response.snapshot.story.name}${startNotice}`);
      } else {
        setNotice(response.wizard.stage_label || "Story wizard updated.");
      }
      setSync(paused ? "Paused" : "Live");
      return response;
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      throw error;
    } finally {
      setSending(false);
    }
  };

  const runBrowserStoryEnhance = async (payload: StoryEnhanceEnvelope): Promise<StoryEnhanceResponse> => {
    if (sending) throw new Error("Another OneDay request is already running.");
    setSending(true);
    setNotice("");
    setSync("Sending");
    try {
      const response = await enhanceStoryText(payload);
      setNotice(response.model ? `Enhanced text with ${response.model}.` : "Enhanced text.");
      setSync(paused ? "Paused" : "Live");
      return response;
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      throw error;
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
        client_revision: readySnapshot.version.revision,
        save_id: save.id,
      });
      setSnapshot(response.snapshot);
      selectModuleTab("saves");
      const loadNotice = response.snapshot_state === "legacy_partial"
        ? `Legacy save loaded with partial rollback${response.snapshot_detail ? `: ${response.snapshot_detail}` : ""}`
        : "Loaded";
      setNotice(`${loadNotice} ${response.save?.name ?? save.name}.`);
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
        client_revision: readySnapshot.version.revision,
        save_id: save.id,
      });
      setSnapshot(response.snapshot);
      selectModuleTab("saves");
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

  const sendAction = async (action: PlayerAction, sourceText: string): Promise<boolean> => {
    if (!snapshot || !storyId || sending || actionSubmitInFlight.current) return false;
	actionSubmitInFlight.current = true;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return false;
      setSync("Sending");
      const currentTurn = readySnapshot.world.current_turn;
      setPendingTurn({
        id: clientId("pending"),
        turn: currentTurn,
        source: action.kind === "choice" ? action.text ?? sourceText.trim() : sourceText.trim(),
		detail: action.kind === "choice" ? "Resolving your selected choice..." : "Resolving your action...",
        kind: action.kind,
      });
      const fingerprint = actionFingerprint(storyId, readySnapshot, action);
      const identity = resolvePendingActionIdentity(pendingActionIdentity.current, fingerprint, () => clientId("turn"));
      pendingActionIdentity.current = identity;
      const response = await submitAction(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: currentTurn,
        client_revision: readySnapshot.version.revision,
        idempotency_key: identity.idempotencyKey,
        action,
        stream: true,
        capabilities: { images: true, ascii: true, roll_log: true },
      });
      setSnapshot(response.snapshot);
      void refreshMiniGame(storyId);
      setLocalCommands((items) => [
        { id: clientId("command"), text: sourceText.trim(), turn: response.snapshot.world.current_turn, source: "browser" as const },
        ...items,
      ].slice(0, 10));
      setSync(paused ? "Paused" : "Live");
      pendingActionIdentity.current = null;
	  return true;
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
	  return false;
    } finally {
	  actionSubmitInFlight.current = false;
      setSending(false);
      setPendingTurn(null);
    }
  };

  const snapshotForSubmit = async (baseSnapshot: StorySnapshot): Promise<StorySnapshot | null> => {
    const shouldRefresh = paused || sync !== "Live" || isSnapshotStale(baseSnapshot);
    if (!shouldRefresh) return baseSnapshot;
    setSync("Loading");
    const latest = await getSnapshot(storyId);
    setSnapshot(latest);
    if (submitSnapshotChanged(baseSnapshot, latest)) {
      setSync(paused ? "Paused" : "Live");
      setNotice("The story changed before sending. Review the latest turn before sending again.");
      return null;
    }
    setSync(paused ? "Paused" : "Live");
    return latest;
  };

  const clearTranscript = () => {
    const lastId = snapshot?.messages.at(-1)?.id ?? 0;
    setHiddenBeforeMessageId(lastId);
  };

  const openOverlay = (nextOverlay: OverlayKind) => {
    if (nextOverlay !== "module") {
      setModuleFocusId(null);
      setModuleOverlayTab(null);
    }
    if (nextOverlay === "saves") {
      selectModuleTab("saves");
      setSaveFilter("");
    }
    if (nextOverlay === "options") {
      setVisualAssetFocusId(null);
      void refreshModelSettings();
    }
    setOverlay(nextOverlay);
  };

  const openModuleOverlay = () => {
    setModuleFocusId(null);
    setModuleOverlayTab(selectedTab);
    setOverlay("module");
  };

  const openNpcCodex = (npcId: string) => {
    setModuleFocusId(npcId);
    setModuleOverlayTab("codex");
    setOverlay("module");
  };

  const closeOverlay = () => {
    setOverlay(null);
    setModuleFocusId(null);
    setModuleOverlayTab(null);
  };

  const updatePreferences = (nextPreferences: AppPreferences) => {
    setPreferences({ ...defaultPreferences, ...nextPreferences });
  };

  const saveModelSettings = async (payload: ModelSettingsUpdate) => {
    setModelSaving(true);
    setNotice("");
    try {
      const nextSettings = await updateModelSettings(payload);
      setModelSettings(nextSettings);
      setModelSettingsError("");
      setNotice(`Model routing saved. Active provider: ${nextSettings.active.provider || "none"}.`);
    } catch (error) {
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setModelSaving(false);
    }
  };

  const saveVisualProfile = async (payload: VisualProfileUpdate) => {
    if (!storyId) return;
    setVisualProfileSaving(true);
    setNotice("");
    try {
      const nextAssets = await updateVisualProfile(storyId, payload);
      setVisualAssets(nextAssets);
      setVisualAssetsError("");
      setNotice("Visual profile saved. New missing assets will use the updated prompt direction.");
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setVisualProfileSaving(false);
    }
  };

  const generateMissingVisualAssets = async (payload: GenerateVisualAssetsRequest = {}) => {
    if (!storyId) return;
    setVisualGenerationBusy(true);
    setNotice("");
    try {
      const nextAssets = await generateVisualAssets(storyId, payload);
      setVisualAssets(nextAssets);
      setVisualAssetsError("");
      const ready = nextAssets.assets.filter((asset) => asset.status === "ready").length;
      const failed = nextAssets.assets.filter((asset) => asset.status === "failed").length;
      const active = nextAssets.assets.filter((asset) => asset.status === "queued" || asset.status === "running").length;
      setNotice(`Visual generation queued. ${ready} ready${active ? `, ${active} queued/running` : ""}${failed ? `, ${failed} failed` : ""}.`);
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setVisualGenerationBusy(false);
    }
  };

  const cancelVisualJob = async (jobId: number) => {
    if (!storyId) return;
    setVisualGenerationBusy(true);
    setNotice("");
    try {
      const nextAssets = await cancelVisualGenerationJob(storyId, jobId);
      setVisualAssets(nextAssets);
      setVisualAssetsError("");
      setNotice(`Visual generation job ${jobId} cancelled.`);
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setVisualGenerationBusy(false);
    }
  };

  const cleanVisualAssetFiles = async (dryRun = false) => {
    if (!storyId) return;
    setVisualGenerationBusy(true);
    setNotice("");
    try {
      const result = await cleanupVisualAssetFiles(storyId, { dry_run: dryRun });
      setNotice(
        dryRun
          ? `Visual cleanup preview: ${result.deleted_files.length} stale files can be removed.`
          : `Visual cleanup removed ${result.deleted_files.length} stale files.`,
      );
      void refreshVisualAssets(storyId);
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setVisualGenerationBusy(false);
    }
  };

  const loadVisualAssetVersions = useCallback(async (assetId: string): Promise<VisualAssetVersion[]> => {
    if (!storyId) return [];
    return getVisualAssetVersions(storyId, assetId);
  }, [storyId]);

  const saveVisualAssetPrompt = useCallback(async (assetId: string, payload: VisualAssetPromptUpdate) => {
    if (!storyId) return;
    setVisualProfileSaving(true);
    setNotice("");
    try {
      const nextAssets = await updateVisualAssetPrompt(storyId, assetId, payload);
      setVisualAssets(nextAssets);
      setVisualAssetsError("");
      setNotice("Image prompt saved. Regenerate the asset to create a new version from this prompt.");
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setVisualProfileSaving(false);
    }
  }, [storyId]);

  const chooseVisualAssetVersion = useCallback(async (assetId: string, versionId: number) => {
    if (!storyId) return;
    setVisualProfileSaving(true);
    setNotice("");
    try {
      const nextAssets = await selectVisualAssetVersion(storyId, assetId, versionId);
      setVisualAssets(nextAssets);
      setVisualAssetsError("");
      setNotice("Visual version selected.");
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setVisualProfileSaving(false);
    }
  }, [storyId]);

  const stepVisualSelection = useCallback(async (assetId: string, action: "undo" | "redo") => {
    if (!storyId) return;
    setVisualProfileSaving(true);
    setNotice("");
    try {
      const nextAssets = await stepVisualAssetSelection(storyId, assetId, action);
      setVisualAssets(nextAssets);
      setVisualAssetsError("");
      setNotice(`Visual selection ${action === "undo" ? "undone" : "restored"}.`);
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setVisualProfileSaving(false);
    }
  }, [storyId]);

  const openVisualAssetEditor = useCallback((assetId: string) => {
    setVisualAssetFocusId(assetId);
    setOverlay("options");
  }, []);

  const toggleLeftRail = () => {
    setPreferences((value) => {
      const opening = !value.showLeftRail;
      return {
        ...defaultPreferences,
        ...value,
        showLeftRail: opening,
        showInspector: opening && window.matchMedia("(max-width: 1600px)").matches ? false : value.showInspector,
      };
    });
  };

  const toggleInspector = () => {
    setPreferences((value) => {
      const opening = !value.showInspector;
      return {
        ...defaultPreferences,
        ...value,
        showLeftRail: opening && window.matchMedia("(max-width: 1600px)").matches ? false : value.showLeftRail,
        showInspector: opening,
      };
    });
  };

  const stepCommandHistory = (direction: -1 | 1): string | null => {
    const next = stepHistoryIndex(historyIndex, direction, recentCommands);
    setHistoryIndex(next.index);
    return next.value;
  };

  const rememberLocalCommand = (text: string, turn: number) => {
    const clean = text.trim();
    if (!clean) return;
    setLocalCommands((items) => [
      { id: clientId("command"), text: clean, turn, source: "browser" as const },
      ...items.filter((item) => item.text.trim().toLowerCase() !== clean.toLowerCase()),
    ].slice(0, 10));
  };

  const visuals = useMemo(() => visualCatalog(visualAssets, snapshot), [snapshot, visualAssets]);

  return (
    <div
      className={`app-shell ${preferences.showLeftRail ? "" : "left-rail-hidden"} ${preferences.showInspector ? "" : "inspector-hidden"} ${preferences.wrapTranscript ? "" : "transcript-nowrap"}`}
      data-density={preferences.density}
      data-font-size={preferences.fontSize}
      data-accent={preferences.accent}
    >
      <TopBar
        snapshot={snapshot}
        sync={sync}
        showLeftRail={preferences.showLeftRail}
        showInspector={preferences.showInspector}
        onToggleLeftRail={toggleLeftRail}
        onToggleInspector={toggleInspector}
        onOpen={openOverlay}
      />
      <div className="workspace">
        {preferences.showLeftRail ? (
          <LeftRail
            stories={filteredStories}
            activeStoryId={storyId}
            filter={filter}
            snapshot={snapshot}
            selectedTab={selectedTab}
            healthText={healthText}
            onFilterChange={setFilter}
            onSelectStory={selectStory}
            onSelectTab={selectModuleTab}
            onRefreshStories={refreshStories}
            onUpdateStory={handleUpdateStory}
            onSetStoryArchived={handleSetStoryArchived}
            onDeleteStory={handleDeleteStory}
            onOpen={openOverlay}
            busyStoryId={storyMutatingId}
			timeline={timeline}
			onForkBranch={forkBranch}
			onRenameBranch={renameBranch}
			onCheckoutBranch={checkoutBranch}
          />
        ) : null}
        <main className="center-stage">
          <section className="transcript-panel" aria-labelledby="story-surface-title">
            <h1 className="sr-only" id="story-surface-title">{snapshot?.story.name || "Your story"}</h1>
            <StoryPath
              snapshot={snapshot}
              locationAsset={visuals.location}
              paused={paused}
              onTogglePaused={() => setPaused((value) => !value)}
              onClearTranscript={clearTranscript}
              onOpenVisualAsset={openVisualAssetEditor}
            />
            <Transcript
              storyId={snapshot?.story.id ?? ""}
              messages={snapshot?.messages ?? []}
              hiddenBeforeId={hiddenBeforeMessageId}
              pendingTurn={pendingTurn}
              timeline={timeline}
              timelineBusy={sending || Boolean(storyMutatingId)}
              onCheckoutBranch={checkoutBranch}
              onRestoreDecision={restoreDecision}
            />
          </section>

          {snapshot && snapshot.choices.length > 0 && (
            <section className="inline-choice-panel" aria-label="Suggested actions">
              <SuggestedActions choices={snapshot.choices} snapshot={snapshot} disabled={sending} showDetails={preferences.showChoiceDetails} onChoice={sendChoice} onDraft={setDraft} />
            </section>
          )}

          {snapshot && activeMiniGame && (
            <MiniGameHost instance={activeMiniGame} busy={miniGameBusy || sending} error={miniGameError} onInput={sendMiniGameInput} />
          )}

          <Composer
            draft={draft}
            mode={mode}
            disabled={sending || !snapshot || Boolean(activeMiniGame && activeMiniGame.runtime.phase !== "resolved")}
            notice={notice}
            commandDescriptors={commandDescriptors}
            commandContext={commandContext}
            onDraftChange={setDraft}
            onModeChange={setMode}
            onSubmit={executeDraft}
            onHistoryStep={stepCommandHistory}
          />
        </main>
        {preferences.showInspector && (
          <Inspector
            snapshot={snapshot}
            selectedTab={selectedTab}
            visuals={visuals}
            onRefresh={() => void loadSnapshot()}
            onOpenModule={openModuleOverlay}
            onOpenNpcCodex={openNpcCodex}
            onOpenVisualAsset={openVisualAssetEditor}
          />
        )}
      </div>
      <PanelDrawer
        overlay={overlay}
        snapshot={snapshot}
        preferences={preferences}
        metaResult={metaResult}
        modelSettings={modelSettings}
        modelError={modelSettingsError}
        modelBusy={modelSaving}
        visualProfile={visualAssets?.profile ?? null}
        visualAssets={visuals.assets}
        visualJobs={visualAssets?.jobs ?? []}
        visuals={visuals}
        visualAssetFocusId={visualAssetFocusId}
        visualProfileError={visualAssetsError}
        visualProfileBusy={visualProfileSaving || visualGenerationBusy}
        selectedTab={selectedTab}
        moduleTab={moduleOverlayTab}
        moduleFocusId={moduleFocusId}
        commandDescriptors={commandDescriptors}
        busy={sending}
        onClose={closeOverlay}
        onPreferencesChange={updatePreferences}
        onModelSettingsSave={(payload) => saveModelSettings(payload)}
        onModelSettingsReload={() => refreshModelSettings()}
        onVisualProfileSave={(payload) => saveVisualProfile(payload)}
        onVisualAssetsGenerate={(payload) => generateMissingVisualAssets(payload)}
        onVisualAssetsReload={() => refreshVisualAssets()}
        onVisualJobCancel={(jobId) => cancelVisualJob(jobId)}
        onVisualAssetsCleanup={(dryRun) => cleanVisualAssetFiles(dryRun)}
        onVisualAssetVersionsLoad={loadVisualAssetVersions}
        onVisualAssetPromptSave={saveVisualAssetPrompt}
        onVisualAssetVersionSelect={chooseVisualAssetVersion}
        onVisualAssetSelectionStep={stepVisualSelection}
        onOpenVisualAsset={openVisualAssetEditor}
        onRunStoryWizard={(payload) => runBrowserStoryWizard(payload)}
        onEnhanceStoryText={(payload) => runBrowserStoryEnhance(payload)}
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

function isSnapshotStale(snapshot: StorySnapshot, maxAgeMs = 30_000): boolean {
  const serverTime = Date.parse(snapshot.server_time);
  if (!Number.isFinite(serverTime)) return false;
  return Date.now() - serverTime > maxAgeMs;
}

function submitSnapshotChanged(previous: StorySnapshot, latest: StorySnapshot): boolean {
  return (
    latest.active_session.id !== previous.active_session.id ||
    latest.version.active_session_id !== previous.version.active_session_id ||
    latest.world.current_turn !== previous.world.current_turn ||
    latest.version.revision !== previous.version.revision ||
    latest.version.last_message_id !== previous.version.last_message_id ||
    latest.version.world_updated_at !== previous.version.world_updated_at
  );
}

function npcNamesFromSnapshot(snapshot: StorySnapshot): string[] {
  return snapshot.panels.npcs.map((npc) => npc.name).filter(Boolean);
}

function saveNamesFromSnapshot(snapshot: StorySnapshot): string[] {
  return snapshot.panels.saves.map((save) => save.name).filter(Boolean);
}

export default App;
