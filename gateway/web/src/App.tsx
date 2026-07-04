import { useCallback, useEffect, useMemo, useState } from "react";
import { actionModeToText, commandToAction, tabHotkeys } from "./commands";
import {
  ApiRequestError,
  createSave,
  deleteSave,
  enhanceStoryText,
  generateVisualAssets,
  getCommandDescriptors,
  getHealth,
  getModelSettings,
  getSnapshot,
  getStories,
  getVisualAssets,
  getVisualAssetVersions,
  loadSave,
  runStoryWizard,
  selectVisualAssetVersion,
  submitAction,
  submitMeta,
  updateModelSettings,
  updateVisualAssetPrompt,
  updateVisualProfile,
} from "./api";
import { Composer } from "./components/Composer";
import { Inspector } from "./components/Inspector";
import { CollapsedLeftRail, LeftRail } from "./components/LeftRail";
import { PanelDrawer } from "./components/PanelDrawer";
import { StoryPath } from "./components/StoryPath";
import { SuggestedActions } from "./components/SuggestedActions";
import { TopBar } from "./components/TopBar";
import { Transcript } from "./components/Transcript";
import { recentFromMessages } from "./format";
import { stepHistoryIndex } from "./history";
import { clientId } from "./ids";
import { defaultPreferences, loadPreferences, savePreferences } from "./preferences";
import { appendTurnEvent, turnEventDetail, turnEventFromContract } from "./turnEvents";
import type {
  AppPreferences,
  ChoiceView,
  CommandDescriptor,
  MetaCommand,
  MetaResult,
  ModelSettings,
  ModelSettingsUpdate,
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
  StoryWizardEnvelope,
  StoryWizardResponse,
  SyncState,
  TurnStreamEvent,
  VisualAssetsResponse,
  GenerateVisualAssetsRequest,
  VisualAssetPromptUpdate,
  VisualAssetVersion,
  VisualProfileUpdate,
} from "./types";
import { visualCatalog } from "./visualAssets";

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
  const [overlay, setOverlay] = useState<OverlayKind>(null);
  const [saveFilter, setSaveFilter] = useState("");
  const [draft, setDraft] = useState("");
  const [mode, setMode] = useState("action");
  const [notice, setNotice] = useState("");
  const [metaResult, setMetaResult] = useState<MetaResult | null>(null);
  const [sending, setSending] = useState(false);
  const [pendingTurn, setPendingTurn] = useState<PendingTurnView | null>(null);
  const [liveTurnEvents, setLiveTurnEvents] = useState<TurnStreamEvent[]>([]);
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

  const loadSnapshot = useCallback(async (nextStoryId = storyId) => {
    if (!nextStoryId) return;
    setSync("Loading");
    try {
      const nextSnapshot = await getSnapshot(nextStoryId);
      setSnapshot(nextSnapshot);
      setSync(paused ? "Paused" : "Live");
      void refreshVisualAssets(nextStoryId);
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
    }
  }, [paused, refreshVisualAssets, storyId]);

  useEffect(() => {
    void refreshHealth();
    void refreshCommandDescriptors();
    void refreshModelSettings();
    void refreshStories();
  }, [refreshCommandDescriptors, refreshHealth, refreshModelSettings, refreshStories]);

  useEffect(() => {
    if (!storyId) return;
    setHiddenBeforeMessageId(0);
    setLiveTurnEvents([]);
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
      void refreshVisualAssets(storyId);
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
      setLiveTurnEvents((items) => appendTurnEvent(items, liveEvent));
      setPendingTurn((pending) => (pending ? { ...pending, detail: turnEventDetail(liveEvent) } : pending));
      if (liveEvent.status === "failed") {
        setSync("Error");
        setNotice(liveEvent.message);
        return;
      }
      if (liveEvent.status === "completed" || liveEvent.status === "snapshot_changed") {
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
    setLiveTurnEvents([]);
    setNotice("");
  };

  const executeDraft = async () => {
    if (!draft.trim() || !snapshot || !storyId || sending) return;
    setNotice("");
    const sourceText = draft.trim();
    const commandResult = commandToAction(draft, {
      descriptors: commandDescriptors,
      npcNames: npcNamesFromSnapshot(snapshot),
      saveNames: saveNamesFromSnapshot(snapshot),
      visiblePrivateThoughts: false,
    });
    if (commandResult.tab) selectModuleTab(commandResult.tab);
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
      rememberLocalCommand(sourceText, snapshot.world.current_turn);
      setDraft("");
      setHistoryIndex(-1);
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
      if (!readySnapshot) return;
      setSync("Sending");
      const response = await submitMeta(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: readySnapshot.world.current_turn,
        client_revision: readySnapshot.version.revision,
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
        setLiveTurnEvents([]);
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

  const sendAction = async (action: PlayerAction, sourceText: string) => {
    if (!snapshot || !storyId || sending) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return;
      setSync("Sending");
      const currentTurn = readySnapshot.world.current_turn;
      setPendingTurn({
        id: clientId("pending"),
        turn: currentTurn,
        source: action.kind === "choice" ? action.text ?? sourceText.trim() : sourceText.trim(),
        detail: action.kind === "choice" ? "Resolving selected choice through the live engine..." : "Waiting for final bridge response from OneDay...",
        kind: action.kind,
      });
      const response = await submitAction(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: currentTurn,
        client_revision: readySnapshot.version.revision,
        idempotency_key: clientId("turn"),
        action,
        capabilities: { images: true, ascii: true, roll_log: true },
      });
      setLiveTurnEvents((items) =>
        response.events.reduce(
          (nextEvents, event) => appendTurnEvent(nextEvents, turnEventFromContract(storyId, currentTurn, action, sourceText, event)),
          items,
        ),
      );
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
      setPendingTurn(null);
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
      latest.version.revision !== baseSnapshot.version.revision ||
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
    if (nextOverlay !== "module") {
      setModuleFocusId(null);
      setModuleOverlayTab(null);
    }
    if (nextOverlay === "saves") {
      selectModuleTab("saves");
      setSaveFilter("");
    }
    if (nextOverlay === "options") {
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
      setNotice(`Visual generation finished. ${ready} ready${failed ? `, ${failed} failed` : ""}.`);
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

  const openVisualAssetEditor = useCallback((assetId: string) => {
    setVisualAssetFocusId(assetId);
    setOverlay("options");
  }, []);

  const toggleLeftRail = () => {
    setPreferences((value) => ({ ...defaultPreferences, ...value, showLeftRail: !value.showLeftRail }));
  };

  const toggleInspector = () => {
    setPreferences((value) => ({ ...defaultPreferences, ...value, showInspector: !value.showInspector }));
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
            onOpen={openOverlay}
          />
        ) : (
          <CollapsedLeftRail
            selectedTab={selectedTab}
            onSelectTab={selectModuleTab}
            onExpand={toggleLeftRail}
            onOpen={openOverlay}
          />
        )}
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
            <StoryPath snapshot={snapshot} locationAsset={visuals.location} onOpenVisualAsset={openVisualAssetEditor} />
            <Transcript
              messages={snapshot?.messages ?? []}
              hiddenBeforeId={hiddenBeforeMessageId}
              pendingTurn={pendingTurn}
              liveEvents={liveTurnEvents}
            />
          </section>

          {snapshot && snapshot.choices.length > 0 && (
            <section className="inline-choice-panel" aria-label="Suggested actions">
              <SuggestedActions choices={snapshot.choices} snapshot={snapshot} disabled={sending} onChoice={sendChoice} onDraft={setDraft} />
            </section>
          )}

          <Composer
            draft={draft}
            mode={mode}
            disabled={sending || !snapshot}
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
        visualAssets={visualAssets?.assets ?? []}
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
        onVisualAssetVersionsLoad={loadVisualAssetVersions}
        onVisualAssetPromptSave={saveVisualAssetPrompt}
        onVisualAssetVersionSelect={chooseVisualAssetVersion}
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

function npcNamesFromSnapshot(snapshot: StorySnapshot): string[] {
  return snapshot.panels.npcs.map((npc) => npc.name).filter(Boolean);
}

function saveNamesFromSnapshot(snapshot: StorySnapshot): string[] {
  return snapshot.panels.saves.map((save) => save.name).filter(Boolean);
}

export default App;
