import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import { useTranslation } from "react-i18next";
import {
  actionFingerprint,
  resolvePendingActionIdentity,
  type PendingActionIdentity,
} from "./actionIdentity";
import { actionModeToText, commandToAction, tabHotkeys } from "./commands";
import {
  ApiRequestError,
  AUTHENTICATION_REQUIRED_EVENT,
  getAuthSession,
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
  getSetupReadiness,
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
  runVisualAssetOperation,
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
import { AuthenticationGate } from "./components/AuthenticationGate";
import { Composer } from "./components/Composer";
import { Inspector } from "./components/Inspector";
import { MiniGameHost } from "./components/MiniGameHost";
import { LeftRail } from "./components/LeftRail";
import { StoryPath } from "./components/StoryPath";
import { SuggestedActions } from "./components/SuggestedActions";
import { TopBar } from "./components/TopBar";
import { Transcript } from "./components/Transcript";
import {
  InstallationOnboarding,
  InstallationReadinessError,
  InstallationReadinessPending,
} from "./features/installation-onboarding/InstallationOnboarding";
import { recentFromMessages } from "./format";
import { stepHistoryIndex } from "./history";
import { restoreFailedDraft } from "./draftLifecycle";
import { coalesceRequest, isCurrentAsyncSelection } from "./asyncState";
import { clientId } from "./ids";
import {
  defaultPreferences,
  loadPreferences,
  savePreferences,
} from "./preferences";
import { loadStoredFonts } from "./fontLibrary";
import { preferenceCssVariables } from "./preferenceTheme";
import i18n, { formatInterfaceNumber, setInterfaceLocale } from "./i18n";
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
  SetupReadinessReport,
  StoryUpdatePayload,
  StoryWizardEnvelope,
  StoryWizardResponse,
  TimelineResponse,
  SyncState,
  TurnStreamEvent,
  VisualAssetsResponse,
  GenerateVisualAssetsRequest,
  VisualAssetPromptUpdate,
  VisualAssetOperationRequest,
  VisualAssetVersion,
  VisualProfileUpdate,
} from "./types";
import { visualCatalog } from "./visualAssets";
import { visualPollingDelayMs } from "./visualJobs";
import { effectiveRouteCapabilities } from "./imageOperations";
import type { SpatialEdge } from "./spatialMap";
import { turnEventMessage } from "./presentation";
import {
  activeStoryCount,
  railPresentation,
  toggleDesktopRailMode,
} from "./features/navigation/railState";
import {
  appRoutePath,
  historyReturnRoute,
  isModuleSection,
  parseAppRoute,
  resolveAppRoute,
  sameAppRoute,
  writeAppRoute,
  type AppRoute,
} from "./appRoute";
import type { SettingsSectionId } from "./components/settings/settingsRegistry";

const deepLinkOverlays = new Set<OverlayKind>([
  "help",
  "options",
  "saves",
  "new-story",
  "meta",
  "module",
]);
const initialAppRoute =
  typeof window === "undefined"
    ? null
    : parseAppRoute(window.location.pathname);
const loadPanelDrawer = () => import("./components/PanelDrawer");
const PanelDrawer = lazy(() =>
  loadPanelDrawer().then((module) => ({ default: module.PanelDrawer })),
);
const StoryLibraryDrawer = lazy(() =>
  import("./features/story-library/StoryLibraryDrawer").then((module) => ({
    default: module.StoryLibraryDrawer,
  })),
);
type HealthState =
  | { kind: "starting" }
  | { kind: "stories"; count: number }
  | { kind: "error"; message: string };

function initialOverlayFromLocation(): OverlayKind {
  if (typeof window === "undefined") return null;
  const overlay = new URLSearchParams(window.location.search).get(
    "overlay",
  ) as OverlayKind;
  return deepLinkOverlays.has(overlay) ? overlay : null;
}

function initialSettingsSectionFromLocation(): SettingsSectionId {
  if (typeof window === "undefined") return "appearance";
  return new URLSearchParams(window.location.search).get("section") ===
    "operator"
    ? "operator"
    : "appearance";
}

type AuthenticationState =
  | { kind: "checking"; bootstrapAvailable: false }
  | { kind: "authenticated"; bootstrapAvailable: boolean }
  | { kind: "required"; bootstrapAvailable: boolean };

function App() {
  const [authentication, setAuthentication] = useState<AuthenticationState>({
    kind: "checking",
    bootstrapAvailable: false,
  });
  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadPanelDrawer();
    }, 700);
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    let active = true;
    const requireAuthentication = () => {
      setAuthentication((current) => ({
        kind: "required",
        bootstrapAvailable: current.bootstrapAvailable,
      }));
    };

    window.addEventListener(
      AUTHENTICATION_REQUIRED_EVENT,
      requireAuthentication,
    );
    void getAuthSession()
      .then((session) => {
        if (!active) return;
        setAuthentication({
          kind: session.authenticated ? "authenticated" : "required",
          bootstrapAvailable: session.bootstrap_available,
        });
      })
      .catch(() => {
        if (!active) return;
        setAuthentication({ kind: "required", bootstrapAvailable: false });
      });

    return () => {
      active = false;
      window.removeEventListener(
        AUTHENTICATION_REQUIRED_EVENT,
        requireAuthentication,
      );
    };
  }, []);

  if (authentication.kind !== "authenticated") {
    return (
      <AuthenticationGate
        checking={authentication.kind === "checking"}
        bootstrapAvailable={authentication.bootstrapAvailable}
      />
    );
  }

  return <AuthenticatedApp />;
}

function AuthenticatedApp() {
  const { t } = useTranslation(["common", "story", "notifications"]);
  const [stories, setStories] = useState<StorySummary[]>([]);
  const [storyId, setStoryId] = useState(() =>
    initialAppRoute?.kind === "story" ? initialAppRoute.storyId : "",
  );
  const [snapshot, setSnapshot] = useState<StorySnapshot | null>(null);
  const [sync, setSync] = useState<SyncState>("Idle");
  const [health, setHealth] = useState<HealthState>({ kind: "starting" });
  const [selectedTab, setSelectedTab] = useState<ModuleTab>(() =>
    initialAppRoute?.kind === "story" &&
    isModuleSection(initialAppRoute.section)
      ? initialAppRoute.section
      : "history",
  );
  const [moduleFocusId, setModuleFocusId] = useState<string | null>(null);
  const [moduleOverlayTab, setModuleOverlayTab] = useState<ModuleTab | null>(
    null,
  );
  const [overlay, setOverlay] = useState<OverlayKind>(() =>
    initialOverlayFromLocation(),
  );
  const [storyLibraryOpen, setStoryLibraryOpen] = useState(
    () => initialAppRoute?.kind === "library",
  );
  const [setupRouteOpen, setSetupRouteOpen] = useState(
    () => initialAppRoute?.kind === "setup",
  );
  const [translationCenterOpen, setTranslationCenterOpen] = useState(
    () =>
      initialAppRoute?.kind === "story" &&
      initialAppRoute.section === "translations",
  );
  const [mobileRailOpen, setMobileRailOpen] = useState(false);
  const [isMobileLayout, setIsMobileLayout] = useState(
    () =>
      typeof window !== "undefined" &&
      window.matchMedia("(max-width: 860px)").matches,
  );
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
  const [preferences, setPreferences] = useState<AppPreferences>(() =>
    loadPreferences(),
  );
  const [commandDescriptors, setCommandDescriptors] = useState<
    CommandDescriptor[]
  >([]);
  const [modelSettings, setModelSettings] = useState<ModelSettings | null>(
    null,
  );
  const [modelSettingsError, setModelSettingsError] = useState("");
  const [modelSaving, setModelSaving] = useState(false);
  const [optionsInitialSection, setOptionsInitialSection] =
    useState<SettingsSectionId>(() => initialSettingsSectionFromLocation());
  const quickSaveRef = useRef<() => void>(() => undefined);
  const quickLoadRef = useRef<() => void>(() => undefined);
  const [setupReadiness, setSetupReadiness] =
    useState<SetupReadinessReport | null>(null);
  const [setupReadinessState, setSetupReadinessState] = useState<
    "loading" | "ready" | "error"
  >("loading");
  const [visualAssets, setVisualAssets] = useState<VisualAssetsResponse | null>(
    null,
  );
  const [visualAssetsError, setVisualAssetsError] = useState("");
  const [visualProfileSaving, setVisualProfileSaving] = useState(false);
  const [visualGenerationBusy, setVisualGenerationBusy] = useState(false);
  const [visualAssetFocusId, setVisualAssetFocusId] = useState<string | null>(
    null,
  );
  const [activeMiniGame, setActiveMiniGame] = useState<MiniGameInstance | null>(
    null,
  );
  const [miniGameBusy, setMiniGameBusy] = useState(false);
  const [miniGameError, setMiniGameError] = useState("");
  const [storyMutatingId, setStoryMutatingId] = useState("");
  const [timeline, setTimeline] = useState<TimelineResponse | null>(null);
  const pendingActionIdentity = useRef<PendingActionIdentity | null>(null);
  const actionSubmitInFlight = useRef(false);
  const storyIdRef = useRef(storyId);
  const selectedTabRef = useRef(selectedTab);
  const storiesRef = useRef(stories);
  const storiesLoadedRef = useRef(false);
  const pausedRef = useRef(paused);
  const snapshotRequestVersion = useRef(0);
  const timelineRequestVersion = useRef(0);
  const snapshotRequests = useRef(
    new Map<string, ReturnType<typeof getSnapshot>>(),
  );
  const timelineRequests = useRef(
    new Map<string, ReturnType<typeof getTimeline>>(),
  );
  const bootstrapStartedRef = useRef(false);
  const visualAssetsRequestVersion = useRef(0);
  const miniGameRequestVersion = useRef(0);
  storyIdRef.current = storyId;
  selectedTabRef.current = selectedTab;
  storiesRef.current = stories;
  pausedRef.current = paused;
  const healthText =
    health.kind === "starting"
      ? t("notifications:health.starting")
      : health.kind === "stories"
        ? t("notifications:health.stories", {
            count: health.count,
            formattedCount: formatInterfaceNumber(health.count),
          })
        : health.message;
  const syncLabel = t(`notifications:sync.${sync.toLowerCase()}`);

  const refreshHealth = useCallback(async () => {
    try {
      const health = await getHealth();
      setHealth({ kind: "stories", count: health.stories });
    } catch (error) {
      setHealth({ kind: "error", message: errorMessage(error) });
    }
  }, []);

  const applyAppRoute = useCallback((route: AppRoute) => {
    setSetupRouteOpen(route.kind === "setup");
    setStoryLibraryOpen(route.kind === "library");
    setTranslationCenterOpen(
      route.kind === "story" && route.section === "translations",
    );
    setMobileRailOpen(false);
    if (route.kind === "setup") {
      setOverlay((current) => (current === "module" ? null : current));
      setModuleOverlayTab(null);
      setModuleFocusId(null);
      return;
    }
    if (route.kind === "library") {
      setOverlay((current) => (current === "module" ? null : current));
      setModuleOverlayTab(null);
      setModuleFocusId(null);
      return;
    }
    if (route.storyId !== storyIdRef.current) setStoryId(route.storyId);
    if (!isModuleSection(route.section)) {
      setOverlay((current) => (current === "module" ? null : current));
      setModuleOverlayTab(null);
      setModuleFocusId(null);
      return;
    }
    setSelectedTab(route.section);
    setModuleFocusId(null);
    if (
      route.section !== "history" &&
      window.matchMedia("(max-width: 1240px)").matches
    ) {
      setModuleOverlayTab(route.section);
      setOverlay("module");
    } else {
      setModuleOverlayTab(null);
      setOverlay((current) => (current === "module" ? null : current));
      setPreferences((value) => ({
        ...defaultPreferences,
        ...value,
        showInspector: true,
      }));
    }
  }, []);

  const navigateAppRoute = useCallback(
    (
      route: AppRoute,
      mode: "push" | "replace" = "push",
      returnTo?: AppRoute,
    ) => {
      const current = parseAppRoute(window.location.pathname);
      if (!current || !sameAppRoute(current, route) || mode === "replace") {
        writeAppRoute(
          route,
          mode,
          returnTo ? { returnTo: appRoutePath(returnTo) } : {},
        );
      }
      applyAppRoute(route);
    },
    [applyAppRoute],
  );

  const refreshStories = useCallback(async () => {
    setSync("Loading");
    try {
      const nextStories = await getStories();
      setStories(nextStories);
      storiesRef.current = nextStories;
      storiesLoadedRef.current = true;
      const requested = parseAppRoute(window.location.pathname);
      const resolved = resolveAppRoute(requested, nextStories);
      const returnTo = historyReturnRoute(window.history.state);
      writeAppRoute(
        resolved,
        "replace",
        returnTo ? { returnTo: appRoutePath(returnTo) } : {},
      );
      applyAppRoute(resolved);
      setSync(
        resolved.kind === "story"
          ? pausedRef.current
            ? "Paused"
            : "Live"
          : "Idle",
      );
      return nextStories;
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
      return [] as StorySummary[];
    }
  }, [applyAppRoute]);

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

  const refreshSetupReadiness = useCallback(async () => {
    setSetupReadinessState("loading");
    try {
      setSetupReadiness(await getSetupReadiness());
      setSetupReadinessState("ready");
    } catch (error) {
      setSetupReadiness(null);
      setSetupReadinessState("error");
      setNotice(errorMessage(error));
    }
  }, []);

  const refreshVisualAssets = useCallback(
    async (nextStoryId = storyIdRef.current) => {
      const requestVersion = ++visualAssetsRequestVersion.current;
      if (!nextStoryId) {
        setVisualAssets(null);
        return;
      }
      try {
        const nextAssets = await getVisualAssets(nextStoryId);
        if (
          !isCurrentAsyncSelection(
            nextStoryId,
            storyIdRef.current,
            requestVersion,
            visualAssetsRequestVersion.current,
          )
        )
          return;
        setVisualAssets(nextAssets);
        setVisualAssetsError("");
      } catch (error) {
        if (
          !isCurrentAsyncSelection(
            nextStoryId,
            storyIdRef.current,
            requestVersion,
            visualAssetsRequestVersion.current,
          )
        )
          return;
        setVisualAssetsError(errorMessage(error));
      }
    },
    [],
  );

  const refreshMiniGame = useCallback(
    async (nextStoryId = storyIdRef.current) => {
      const requestVersion = ++miniGameRequestVersion.current;
      if (!nextStoryId) {
        setActiveMiniGame(null);
        return;
      }
      try {
        const response = await getActiveMiniGame(nextStoryId);
        if (
          !isCurrentAsyncSelection(
            nextStoryId,
            storyIdRef.current,
            requestVersion,
            miniGameRequestVersion.current,
          )
        )
          return;
        setActiveMiniGame(response.instance ?? null);
        setMiniGameError("");
      } catch (error) {
        if (
          !isCurrentAsyncSelection(
            nextStoryId,
            storyIdRef.current,
            requestVersion,
            miniGameRequestVersion.current,
          )
        )
          return;
        setMiniGameError(errorMessage(error));
      }
    },
    [],
  );

  const loadSnapshot = useCallback(
    async (nextStoryId = storyIdRef.current) => {
      if (!nextStoryId) return;
      const requestVersion = ++snapshotRequestVersion.current;
      setSync("Loading");
      try {
        const nextSnapshot = await coalesceRequest(
          snapshotRequests.current,
          nextStoryId,
          () => getSnapshot(nextStoryId),
        );
        if (
          !isCurrentAsyncSelection(
            nextStoryId,
            storyIdRef.current,
            requestVersion,
            snapshotRequestVersion.current,
          )
        )
          return;
        setSnapshot(nextSnapshot);
        setSync(pausedRef.current ? "Paused" : "Live");
        void refreshVisualAssets(nextStoryId);
        void refreshMiniGame(nextStoryId);
      } catch (error) {
        if (
          !isCurrentAsyncSelection(
            nextStoryId,
            storyIdRef.current,
            requestVersion,
            snapshotRequestVersion.current,
          )
        )
          return;
        setSync("Error");
        setNotice(errorMessage(error));
      }
    },
    [refreshMiniGame, refreshVisualAssets],
  );

  const refreshTimeline = useCallback(
    async (nextStoryId = storyIdRef.current, reportError = false) => {
      if (!nextStoryId) return;
      const requestVersion = ++timelineRequestVersion.current;
      try {
        const nextTimeline = await coalesceRequest(
          timelineRequests.current,
          nextStoryId,
          () => getTimeline(nextStoryId),
        );
        if (
          !isCurrentAsyncSelection(
            nextStoryId,
            storyIdRef.current,
            requestVersion,
            timelineRequestVersion.current,
          )
        )
          return;
        setTimeline(nextTimeline);
      } catch (error) {
        if (
          reportError &&
          isCurrentAsyncSelection(
            nextStoryId,
            storyIdRef.current,
            requestVersion,
            timelineRequestVersion.current,
          )
        ) {
          setNotice(errorMessage(error));
        }
      }
    },
    [],
  );

  useEffect(() => {
    if (bootstrapStartedRef.current) return;
    bootstrapStartedRef.current = true;
    void refreshHealth();
    void refreshCommandDescriptors();
    void refreshModelSettings();
    void refreshSetupReadiness();
    void refreshStories();
  }, [
    refreshCommandDescriptors,
    refreshHealth,
    refreshModelSettings,
    refreshSetupReadiness,
    refreshStories,
  ]);

  useEffect(() => {
    const onPopState = () => {
      if (!storiesLoadedRef.current) return;
      const requested = parseAppRoute(window.location.pathname);
      const resolved = resolveAppRoute(requested, storiesRef.current);
      if (!requested || !sameAppRoute(requested, resolved))
        writeAppRoute(resolved, "replace");
      applyAppRoute(resolved);
    };
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, [applyAppRoute]);

  useEffect(() => {
    if (!storyId) return;
    setTimeline(null);
    setSnapshot(null);
    setVisualAssets(null);
    setActiveMiniGame(null);
    setHiddenBeforeMessageId(0);
    void loadSnapshot(storyId);
    void refreshTimeline(storyId, true);
  }, [loadSnapshot, refreshTimeline, storyId]);

  const mutateTimeline = async (
    payload: Parameters<typeof updateTimeline>[1],
  ) => {
    if (!storyId || !snapshot || storyMutatingId) return;
    setStoryMutatingId(storyId);
    try {
      const response = await updateTimeline(storyId, payload);
      setTimeline(response.timeline);
      setSnapshot(response.snapshot);
      setPendingTurn(null);
      void refreshMiniGame(storyId);
      pendingActionIdentity.current = null;
      setNotice(
        t("notifications:branch.active", {
          name:
            response.timeline.branches.find(
              (branch) => branch.id === response.timeline.active_branch_id,
            )?.name ?? t("notifications:branch.updated"),
        }),
      );
    } catch (error) {
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setStoryMutatingId("");
    }
  };

  const forkBranch = (name: string) =>
    timeline?.head
      ? mutateTimeline({
          action: "fork",
          client_revision: snapshot?.version.revision ?? timeline.revision,
          from_commit_id: timeline.head.id,
          name,
        })
      : Promise.resolve();
  const renameBranch = (branchId: string, name: string) =>
    mutateTimeline({
      action: "rename",
      client_revision: snapshot?.version.revision ?? timeline?.revision ?? 0,
      branch_id: branchId,
      name,
    });
  const checkoutBranch = (branchId: string) =>
    mutateTimeline({
      action: "checkout",
      client_revision: snapshot?.version.revision ?? timeline?.revision ?? 0,
      branch_id: branchId,
    });

  const restoreDecision = async (fromCommitId: string, turn: number) => {
    if (!storyId || !snapshot || !timeline || !fromCommitId || storyMutatingId)
      return;
    const siblingCount = timeline.branches.filter(
      (branch) => branch.fork_commit_id === fromCommitId,
    ).length;
    const name = `Turn ${Math.max(0, turn)} alternative ${siblingCount + 1}`;
    setStoryMutatingId(storyId);
    try {
      const checkedOut = await updateTimeline(storyId, {
        action: "fork_checkout",
        client_revision: snapshot.version.revision,
        from_commit_id: fromCommitId,
        name,
      });
      setTimeline(checkedOut.timeline);
      setSnapshot(checkedOut.snapshot);
      setPendingTurn(null);
      pendingActionIdentity.current = null;
      setNotice(t("notifications:branch.previousDecision", { name }));
    } catch (error) {
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
      void refreshTimeline(storyId);
    } finally {
      setStoryMutatingId("");
    }
  };

  useEffect(() => {
    if (!storyId || paused) {
      if (storyId) setSync("Paused");
      return;
    }
    let source: EventSource | null = null;
    const connectTimer = window.setTimeout(() => {
      const nextSource = new EventSource(
        `/api/stories/${encodeURIComponent(storyId)}/events`,
      );
      source = nextSource;
      nextSource.addEventListener("open", () => setSync("Live"));
      nextSource.addEventListener("snapshot", (event) => {
        const nextSnapshot = parseStorySnapshotEvent(event.data);
        if (!nextSnapshot) {
          setNotice(t("story:unreadableSnapshot"));
          setSync("Reconnecting");
          return;
        }
        snapshotRequestVersion.current += 1;
        setSnapshot(nextSnapshot);
        void refreshVisualAssets(storyId);
        void refreshTimeline(storyId);
        setSync("Live");
      });
      nextSource.addEventListener("turn", (event) => {
        let liveEvent: TurnStreamEvent;
        try {
          liveEvent = JSON.parse(event.data) as TurnStreamEvent;
        } catch {
          setNotice(t("story:unreadableEvent"));
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
          if (
            pending.streamingSuppressed ||
            shouldSuppressStreamingDelta(pending.streamingText, delta)
          ) {
            return {
              ...pending,
              detail: t("story:preparing"),
              streamingText: undefined,
              streamingSuppressed: true,
            };
          }
          return {
            ...pending,
            detail: t("story:streaming"),
            streamingText: `${pending.streamingText ?? ""}${delta}`,
          };
        });
        if (liveEvent.status === "failed") {
          setSync("Error");
          setNotice(turnEventMessage(liveEvent, t));
          return;
        }
        if (
          liveEvent.status === "completed" ||
          liveEvent.status === "snapshot_changed"
        ) {
          pendingActionIdentity.current = null;
          setSync(paused ? "Paused" : "Live");
          return;
        }
        if (liveEvent.status === "submitted" || liveEvent.status === "event") {
          setSync(paused ? "Paused" : "Sending");
        }
      });
      nextSource.addEventListener("error", () => setSync("Reconnecting"));
    }, 0);
    return () => {
      window.clearTimeout(connectTimer);
      source?.close();
    };
  }, [paused, refreshTimeline, refreshVisualAssets, storyId, t]);

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

  useEffect(() => {
    const media = window.matchMedia("(max-width: 860px)");
    const update = () => {
      setIsMobileLayout(media.matches);
      if (!media.matches) setMobileRailOpen(false);
    };
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);

  useEffect(() => {
    void loadStoredFonts().catch(() => undefined);
  }, []);

  useEffect(() => {
    void setInterfaceLocale(preferences.locale).then(async (locale) => {
      // Transient errors are already presentation strings. Clear and refresh
      // them so a live locale change cannot leave stale copy in the old language.
      setNotice("");
      setModelSettingsError("");
      setVisualAssetsError("");
      setMiniGameError("");
      setHealth({ kind: "starting" });
      const descriptors = await getCommandDescriptors(locale).catch(() => []);
      setCommandDescriptors(descriptors);
      await refreshHealth();
    });
  }, [preferences.locale, refreshHealth]);

  const selectModuleTab = useCallback(
    (tab: ModuleTab) => {
      const activeStoryId = storyIdRef.current;
      if (!activeStoryId) return;
      navigateAppRoute({ kind: "story", storyId: activeStoryId, section: tab });
    },
    [navigateAppRoute],
  );

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const isEditing =
        target?.tagName === "INPUT" ||
        target?.tagName === "TEXTAREA" ||
        target?.tagName === "SELECT";
      if (isEditing) return;

      if (event.key === "?") {
        event.preventDefault();
        setOverlay("help");
        return;
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s") {
        event.preventDefault();
        quickSaveRef.current();
        return;
      }
      if (
        (event.ctrlKey || event.metaKey) &&
        event.shiftKey &&
        event.key.toLowerCase() === "l"
      ) {
        event.preventDefault();
        quickLoadRef.current();
        return;
      }
      if (event.key.toLowerCase() === "o") {
        event.preventDefault();
        setVisualAssetFocusId(null);
        setOptionsInitialSection("appearance");
        setOverlay("options");
        return;
      }
      if (event.key === "[") {
        event.preventDefault();
        if (isMobileLayout) setMobileRailOpen((value) => !value);
        else
          setPreferences((value) => ({
            ...defaultPreferences,
            ...value,
            showLeftRail: !value.showLeftRail,
          }));
        return;
      }
      if (event.key === "]") {
        event.preventDefault();
        setPreferences((value) => ({
          ...defaultPreferences,
          ...value,
          showInspector: !value.showInspector,
        }));
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
  }, [isMobileLayout, selectModuleTab]);

  const selectStory = (nextStoryId: string) => {
    if (nextStoryId === storyId) {
      void loadSnapshot(nextStoryId);
    } else {
      setSnapshot(null);
      setVisualAssets(null);
      setVisualAssetsError("");
      pendingActionIdentity.current = null;
      setNotice("");
    }
    navigateAppRoute({
      kind: "story",
      storyId: nextStoryId,
      section: selectedTabRef.current,
    });
  };

  const handleUpdateStory = async (
    targetStoryId: string,
    payload: StoryUpdatePayload,
  ) => {
    if (!targetStoryId || storyMutatingId) return;
    setStoryMutatingId(targetStoryId);
    setSync("Saving");
    try {
      const updated = await updateStory(targetStoryId, payload);
      setStories((items) =>
        items.map((story) => (story.id === targetStoryId ? updated : story)),
      );
      if (targetStoryId === storyId) {
        await loadSnapshot(targetStoryId);
      }
      setNotice(
        t(
          updated.is_archived
            ? "notifications:story.archived"
            : "notifications:story.updated",
          { name: updated.name },
        ),
      );
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
    } finally {
      setStoryMutatingId("");
    }
  };

  const handleSetStoryArchived = async (
    targetStoryId: string,
    archived: boolean,
  ) => {
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
        .map((count) => `${count.table}: ${formatInterfaceNumber(count.rows)}`)
        .join("\n");
      const retainedFiles = plan.retained_asset_files.length;
      const message = [
        t("notifications:story.deletePrompt", {
          name: plan.story_name || targetStoryId,
        }),
        "",
        t("notifications:story.rowsAffected", {
          count: plan.total_rows,
          formattedCount: formatInterfaceNumber(plan.total_rows),
        }),
        topCounts,
        retainedFiles
          ? t("notifications:story.retainedFiles", {
              count: retainedFiles,
              formattedCount: formatInterfaceNumber(retainedFiles),
            })
          : t("notifications:story.noGeneratedFiles"),
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
        const nextActive =
          nextStories.find((story) => !story.is_archived) ?? null;
        setSnapshot(null);
        setVisualAssets(null);
        setVisualAssetsError("");
        pendingActionIdentity.current = null;
        if (nextActive) {
          navigateAppRoute(
            { kind: "story", storyId: nextActive.id, section: "history" },
            "replace",
          );
          await loadSnapshot(nextActive.id);
        } else {
          navigateAppRoute({ kind: "library" }, "replace");
          setSync("Idle");
        }
      } else {
        setSync(paused ? "Paused" : "Live");
      }
      setNotice(t("notifications:story.deleted"));
    } catch (error) {
      setSync("Error");
      setNotice(errorMessage(error));
    } finally {
      setStoryMutatingId("");
    }
  };

  const executeDraft = async (draftOverride?: string) => {
    const currentDraft = draftOverride ?? draft;
    if (
      !currentDraft.trim() ||
      !snapshot ||
      !storyId ||
      sending ||
      actionSubmitInFlight.current
    )
      return;
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
    if (commandResult.saveFilter !== undefined)
      setSaveFilter(commandResult.saveFilter);
    if (commandResult.saveDeleteFilter !== undefined)
      setSaveFilter(commandResult.saveDeleteFilter);
    if (commandResult.notice) setNotice(commandResult.notice);
    if (commandResult.timeline) {
      const details =
        document.querySelector<HTMLDetailsElement>(".branch-navigator");
      if (details) {
        details.open = true;
        details.querySelector<HTMLElement>("summary")?.focus();
      }
      const value = commandResult.timeline.value?.trim() ?? "";
      if (commandResult.timeline.action === "list")
        setNotice(t("notifications:branch.navigatorOpened"));
      if (commandResult.timeline.action === "fork")
        value
          ? await forkBranch(value)
          : setNotice(t("notifications:branch.forkUsage"));
      if (commandResult.timeline.action === "rename")
        value
          ? await renameBranch(timeline?.active_branch_id ?? "", value)
          : setNotice(t("notifications:branch.renameUsage"));
      if (commandResult.timeline.action === "checkout") {
        const target = timeline?.branches.find(
          (branch) =>
            branch.id === value ||
            branch.name.toLowerCase() === value.toLowerCase(),
        );
        target
          ? await checkoutBranch(target.id)
          : setNotice(
              t("notifications:branch.notFound", {
                name: value || t("notifications:branch.missingName"),
              }),
            );
      }
      if (commandResult.timeline.action === "retry") {
        const decision = timeline?.head?.parent_commit_id ?? "";
        decision
          ? await restoreDecision(
              decision,
              timeline?.head?.canonical_turn ?? snapshot.world.current_turn,
            )
          : setNotice(t("notifications:branch.noDecision"));
      }
      setDraft("");
      setHistoryIndex(-1);
      return;
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
      const isEditing =
        target?.tagName === "INPUT" ||
        target?.tagName === "TEXTAREA" ||
        target?.tagName === "SELECT";
      if (isEditing || sending || !snapshot || !/^[1-6]$/.test(event.key))
        return;

      const keyNumber = Number.parseInt(event.key, 10);
      const choice =
        snapshot.choices.find((item) => item.id === keyNumber) ??
        snapshot.choices[keyNumber - 1];
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
        setNotice(
          t("notifications:meta.answered", { title: response.meta.title }),
        );
      } else {
        setNotice(t("notifications:meta.completed"));
      }
      setLocalCommands((items) =>
        [
          {
            id: clientId("command"),
            text: sourceText.trim(),
            turn: response.snapshot.world.current_turn,
            source: "browser" as const,
          },
          ...items,
        ].slice(0, 10),
      );
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
        const degree =
          result.outcome?.degree ??
          (result.passed ? "full_success" : "hard_failure");
        const continuation = `[Challenge Result: mini_game ${degree.toUpperCase()} - ${result.detail}]`;
        const continued = await sendAction(
          { kind: "free_text", text: continuation },
          t("notifications:challenge.resolved"),
        );
        if (continued) {
          setActiveMiniGame(null);
          await refreshMiniGame(storyId);
        } else {
          setMiniGameError(t("notifications:challenge.continuationRetry"));
        }
      }
    } catch (error) {
      setMiniGameError(errorMessage(error));
    } finally {
      setMiniGameBusy(false);
    }
  };

  const createManualSave = async (
    name: string,
    sourceText = "",
    options: { kind?: "manual" | "quicksave"; reveal?: boolean } = {},
  ) => {
    if (!snapshot || !storyId || sending) return;
    setSending(true);
    const baseSnapshot = snapshot;
    try {
      const readySnapshot = await snapshotForSubmit(baseSnapshot);
      if (!readySnapshot) return false;
      setSync("Sending");
      const kind = options.kind ?? "manual";
      const saveName =
        name.trim() ||
        (kind === "quicksave"
          ? `Quicksave T${readySnapshot.world.current_turn}`
          : `Browser Save T${readySnapshot.world.current_turn}`);
      const response = await createSave(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: readySnapshot.world.current_turn,
        client_revision: readySnapshot.version.revision,
        name: saveName,
        kind,
      });
      setSnapshot(response.snapshot);
      if (options.reveal !== false) {
        selectModuleTab("saves");
        setOverlay("saves");
        setSaveFilter("");
      }
      setNotice(
        t("notifications:save.saved", {
          name: response.save?.name ?? saveName,
        }),
      );
      setLocalCommands((items) =>
        [
          {
            id: clientId("command"),
            text: sourceText.trim() || `/save ${saveName}`,
            turn: response.snapshot.world.current_turn,
            source: "browser" as const,
          },
          ...items,
        ].slice(0, 10),
      );
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setSending(false);
    }
  };

  const runBrowserStoryWizard = async (
    payload: StoryWizardEnvelope,
  ): Promise<StoryWizardResponse> => {
    if (sending) throw new Error(t("notifications:request.alreadyRunning"));
    setSending(true);
    setNotice("");
    setSync("Sending");
    try {
      const response = await runStoryWizard(payload);
      if (response.snapshot) {
        setSnapshot(response.snapshot);
        setVisualAssets(null);
        setStories((items) => [
          response.snapshot!.story,
          ...items.filter((story) => story.id !== response.snapshot!.story.id),
        ]);
        navigateAppRoute(
          {
            kind: "story",
            storyId: response.snapshot.story.id,
            section: "history",
          },
          "replace",
        );
        setOverlay(null);
        setHiddenBeforeMessageId(0);
        pendingActionIdentity.current = null;
        setNotice(
          response.wizard.start_error
            ? t("notifications:wizard.createdStartFailed", {
                name: response.snapshot.story.name,
                error: response.wizard.start_error,
              })
            : t(
                response.wizard.started
                  ? "notifications:wizard.createdStarted"
                  : "notifications:wizard.created",
                { name: response.snapshot.story.name },
              ),
        );
      } else {
        setNotice(
          response.wizard.stage_label || t("notifications:wizard.updated"),
        );
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

  const runBrowserStoryEnhance = async (
    payload: StoryEnhanceEnvelope,
  ): Promise<StoryEnhanceResponse> => {
    if (sending) throw new Error(t("notifications:request.alreadyRunning"));
    setSending(true);
    setNotice("");
    setSync("Sending");
    try {
      const response = await enhanceStoryText(payload);
      setNotice(
        response.model
          ? t("notifications:enhance.withModel", { model: response.model })
          : t("notifications:enhance.completed"),
      );
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
    const confirmed = window.confirm(
      t("notifications:save.loadConfirm", {
        name: save.name,
        turn: formatInterfaceNumber(save.turn),
      }),
    );
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
      const loadedName = response.save?.name ?? save.name;
      setNotice(
        response.snapshot_state === "legacy_partial"
          ? response.snapshot_detail
            ? t("notifications:save.legacyLoadedWithDetail", {
                name: loadedName,
                detail: response.snapshot_detail,
              })
            : t("notifications:save.legacyLoaded", { name: loadedName })
          : t("notifications:save.loaded", { name: loadedName }),
      );
      setLocalCommands((items) =>
        [
          {
            id: clientId("command"),
            text: `/load ${save.name}`,
            turn: response.snapshot.world.current_turn,
            source: "browser" as const,
          },
          ...items,
        ].slice(0, 10),
      );
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
    const confirmed = window.confirm(
      t("notifications:save.deleteConfirm", {
        name: save.name,
        turn: formatInterfaceNumber(save.turn),
      }),
    );
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
      setNotice(
        t("notifications:save.deleted", {
          name: response.save?.name ?? save.name,
        }),
      );
      setLocalCommands((items) =>
        [
          {
            id: clientId("command"),
            text: `/delete-save ${save.name}`,
            turn: response.snapshot.world.current_turn,
            source: "browser" as const,
          },
          ...items,
        ].slice(0, 10),
      );
      setSync(paused ? "Paused" : "Live");
    } catch (error) {
      setSync("Error");
      setNotice(actionErrorMessage(error));
      await loadSnapshot().catch(() => undefined);
    } finally {
      setSending(false);
    }
  };

  const quickSave = () => {
    if (!snapshot || sending) return;
    void createManualSave("", "/quicksave", {
      kind: "quicksave",
      reveal: false,
    });
  };

  const quickLoad = () => {
    if (!snapshot || sending) return;
    const latest = snapshot.panels.saves[0];
    if (!latest) {
      setSaveFilter("");
      setOverlay("saves");
      setNotice(t("notifications:save.noneAvailable"));
      return;
    }
    void loadManualSave(latest);
  };
  quickSaveRef.current = quickSave;
  quickLoadRef.current = quickLoad;

  const sendAction = async (
    action: PlayerAction,
    sourceText: string,
  ): Promise<boolean> => {
    if (!snapshot || !storyId || sending || actionSubmitInFlight.current)
      return false;
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
        source:
          action.kind === "choice"
            ? (action.text ?? sourceText.trim())
            : sourceText.trim(),
        detail:
          action.kind === "choice"
            ? t("notifications:action.resolvingChoice")
            : t("notifications:action.resolvingAction"),
        kind: action.kind,
      });
      const fingerprint = actionFingerprint(storyId, readySnapshot, action);
      const identity = resolvePendingActionIdentity(
        pendingActionIdentity.current,
        fingerprint,
        () => clientId("turn"),
      );
      pendingActionIdentity.current = identity;
      const response = await submitAction(storyId, {
        session_id: readySnapshot.active_session.id,
        client_turn: currentTurn,
        client_revision: readySnapshot.version.revision,
        idempotency_key: identity.idempotencyKey,
        action,
        stream: true,
        capabilities: {
          images: true,
          ascii: true,
          roll_log: true,
          automatic_challenges: preferences.automaticChallenges,
          timing_free_challenges: preferences.timingFreeChallenges,
          challenge_cooldown: preferences.challengeCooldown,
          excluded_minigames: preferences.disabledMiniGames,
        },
      });
      setSnapshot(response.snapshot);
      void refreshMiniGame(storyId);
      setLocalCommands((items) =>
        [
          {
            id: clientId("command"),
            text: sourceText.trim(),
            turn: response.snapshot.world.current_turn,
            source: "browser" as const,
          },
          ...items,
        ].slice(0, 10),
      );
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

  const snapshotForSubmit = async (
    baseSnapshot: StorySnapshot,
  ): Promise<StorySnapshot | null> => {
    const shouldRefresh =
      paused || sync !== "Live" || isSnapshotStale(baseSnapshot);
    if (!shouldRefresh) return baseSnapshot;
    setSync("Loading");
    const latest = await getSnapshot(storyId);
    setSnapshot(latest);
    if (submitSnapshotChanged(baseSnapshot, latest)) {
      setSync(paused ? "Paused" : "Live");
      setNotice(t("story:changed"));
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
    setStoryLibraryOpen(false);
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
      setOptionsInitialSection("appearance");
      if (!modelSettings) void refreshModelSettings();
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

  const openHistoryModule = (tab: "map" | "codex") => {
    setSelectedTab(tab);
    setModuleFocusId(null);
    setModuleOverlayTab(tab);
    setOverlay("module");
  };

  const closeOverlay = () => {
    setOverlay(null);
    setVisualAssetFocusId(null);
    setModuleFocusId(null);
    setModuleOverlayTab(null);
    setOptionsInitialSection("appearance");
  };

  const openSetupReview = () => {
    void refreshSetupReadiness();
    navigateAppRoute({ kind: "setup" }, "push");
  };

  const openInstallationConfiguration = () => {
    setStoryLibraryOpen(false);
    setVisualAssetFocusId(null);
    setOptionsInitialSection("operator");
    if (!modelSettings) void refreshModelSettings();
    setOverlay("options");
  };

  const handleMapTravel = (locationName: string, route: SpatialEdge | null) => {
    const routeDetail = route
      ? [
          route.travel_mode,
          route.direction,
          route.travel_minutes ? `${route.travel_minutes} minutes` : "",
        ]
          .filter(Boolean)
          .join(", ")
      : "an unexplored route";
    const text = route
      ? `Travel to ${locationName} via ${routeDetail}.`
      : `Find a safe route to ${locationName} and travel there.`;
    closeOverlay();
    void sendAction({ kind: "free_text", text }, text);
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
      void refreshSetupReadiness();
      setNotice(
        t("notifications:model.saved", {
          provider:
            nextSettings.active.provider || t("notifications:model.none"),
        }),
      );
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
      setNotice(t("notifications:visual.profileSaved"));
    } catch (error) {
      setVisualAssetsError(errorMessage(error));
      setNotice(errorMessage(error));
      throw error;
    } finally {
      setVisualProfileSaving(false);
    }
  };

  const generateMissingVisualAssets = async (
    payload: GenerateVisualAssetsRequest = {},
  ) => {
    if (!storyId) return;
    setVisualGenerationBusy(true);
    setNotice("");
    try {
      const nextAssets = await generateVisualAssets(storyId, payload);
      setVisualAssets(nextAssets);
      setVisualAssetsError("");
      const ready = nextAssets.assets.filter(
        (asset) => asset.status === "ready",
      ).length;
      const failed = nextAssets.assets.filter(
        (asset) => asset.status === "failed",
      ).length;
      const active = nextAssets.assets.filter(
        (asset) => asset.status === "queued" || asset.status === "running",
      ).length;
      const summary = [
        t("notifications:visual.ready", {
          count: ready,
          formattedCount: formatInterfaceNumber(ready),
        }),
        active
          ? t("notifications:visual.active", {
              count: active,
              formattedCount: formatInterfaceNumber(active),
            })
          : "",
        failed
          ? t("notifications:visual.failed", {
              count: failed,
              formattedCount: formatInterfaceNumber(failed),
            })
          : "",
      ]
        .filter(Boolean)
        .join(", ");
      setNotice(t("notifications:visual.generationQueued", { summary }));
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
      setNotice(t("notifications:visual.jobCancelled", { jobId }));
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
      const result = await cleanupVisualAssetFiles(storyId, {
        dry_run: dryRun,
      });
      setNotice(
        dryRun
          ? t("notifications:visual.cleanupPreview", {
              count: result.deleted_files.length,
              formattedCount: formatInterfaceNumber(
                result.deleted_files.length,
              ),
            })
          : t("notifications:visual.cleanupRemoved", {
              count: result.deleted_files.length,
              formattedCount: formatInterfaceNumber(
                result.deleted_files.length,
              ),
            }),
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

  const loadVisualAssetVersions = useCallback(
    async (assetId: string): Promise<VisualAssetVersion[]> => {
      if (!storyId) return [];
      return getVisualAssetVersions(storyId, assetId);
    },
    [storyId],
  );

  const saveVisualAssetPrompt = useCallback(
    async (assetId: string, payload: VisualAssetPromptUpdate) => {
      if (!storyId) return;
      setVisualProfileSaving(true);
      setNotice("");
      try {
        const nextAssets = await updateVisualAssetPrompt(
          storyId,
          assetId,
          payload,
        );
        setVisualAssets(nextAssets);
        setVisualAssetsError("");
        setNotice(t("notifications:visual.promptSaved"));
      } catch (error) {
        setVisualAssetsError(errorMessage(error));
        setNotice(errorMessage(error));
        throw error;
      } finally {
        setVisualProfileSaving(false);
      }
    },
    [storyId, t],
  );

  const chooseVisualAssetVersion = useCallback(
    async (assetId: string, versionId: number) => {
      if (!storyId) return;
      setVisualProfileSaving(true);
      setNotice("");
      try {
        const nextAssets = await selectVisualAssetVersion(
          storyId,
          assetId,
          versionId,
        );
        setVisualAssets(nextAssets);
        setVisualAssetsError("");
        setNotice(t("notifications:visual.versionSelected"));
      } catch (error) {
        setVisualAssetsError(errorMessage(error));
        setNotice(errorMessage(error));
        throw error;
      } finally {
        setVisualProfileSaving(false);
      }
    },
    [storyId, t],
  );

  const stepVisualSelection = useCallback(
    async (assetId: string, action: "undo" | "redo") => {
      if (!storyId) return;
      setVisualProfileSaving(true);
      setNotice("");
      try {
        const nextAssets = await stepVisualAssetSelection(
          storyId,
          assetId,
          action,
        );
        setVisualAssets(nextAssets);
        setVisualAssetsError("");
        setNotice(
          t(
            action === "undo"
              ? "notifications:visual.selectionUndone"
              : "notifications:visual.selectionRestored",
          ),
        );
      } catch (error) {
        setVisualAssetsError(errorMessage(error));
        setNotice(errorMessage(error));
        throw error;
      } finally {
        setVisualProfileSaving(false);
      }
    },
    [storyId, t],
  );

  const runImageOperation = useCallback(
    async (assetId: string, payload: VisualAssetOperationRequest) => {
      if (!storyId) return;
      setVisualGenerationBusy(true);
      setNotice("");
      try {
        const nextAssets = await runVisualAssetOperation(
          storyId,
          assetId,
          payload,
        );
        setVisualAssets(nextAssets);
        setVisualAssetsError("");
        setNotice(i18n.t("image_editing:queued"));
        return nextAssets;
      } catch (error) {
        setVisualAssetsError(errorMessage(error));
        setNotice(errorMessage(error));
        throw error;
      } finally {
        setVisualGenerationBusy(false);
      }
    },
    [storyId],
  );

  const openVisualAssetEditor = useCallback((assetId: string) => {
    setVisualAssetFocusId(assetId);
    setOverlay("options");
  }, []);

  const toggleLeftRail = () => {
    if (isMobileLayout) {
      setMobileRailOpen((value) => !value);
      return;
    }
    setPreferences((value) => {
      const opening = !value.showLeftRail;
      return {
        ...defaultPreferences,
        ...value,
        showLeftRail: opening,
        showInspector:
          opening && window.matchMedia("(max-width: 1600px)").matches
            ? false
            : value.showInspector,
      };
    });
  };

  const toggleRailMode = () => {
    setPreferences((value) => ({
      ...defaultPreferences,
      ...value,
      desktopRailMode: toggleDesktopRailMode(value.desktopRailMode),
      showLeftRail: true,
    }));
  };

  const openStoryLibrary = () => {
    const returnTo: AppRoute | undefined = storyIdRef.current
      ? {
          kind: "story",
          storyId: storyIdRef.current,
          section: selectedTabRef.current,
        }
      : undefined;
    navigateAppRoute({ kind: "library" }, "push", returnTo);
  };

  const closeRoutedSurface = () => {
    const current = parseAppRoute(window.location.pathname);
    if (current?.kind === "story" && current.section !== "translations") {
      applyAppRoute(current);
      return;
    }
    const returnTo = historyReturnRoute(window.history.state);
    if (returnTo) {
      navigateAppRoute(returnTo, "replace");
      return;
    }
    navigateAppRoute(resolveAppRoute(null, storiesRef.current), "replace");
  };

  const setTranslationRouteOpen = (open: boolean) => {
    if (!open) {
      closeRoutedSurface();
      return;
    }
    if (!storyIdRef.current) return;
    navigateAppRoute(
      { kind: "story", storyId: storyIdRef.current, section: "translations" },
      "push",
      {
        kind: "story",
        storyId: storyIdRef.current,
        section: selectedTabRef.current,
      },
    );
  };

  const openNewStoryFromLibrary = () => {
    setStoryLibraryOpen(false);
    openOverlay("new-story");
  };

  const toggleInspector = () => {
    setPreferences((value) => {
      const opening = !value.showInspector;
      return {
        ...defaultPreferences,
        ...value,
        showLeftRail:
          opening && window.matchMedia("(max-width: 1600px)").matches
            ? false
            : value.showLeftRail,
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
    setLocalCommands((items) =>
      [
        {
          id: clientId("command"),
          text: clean,
          turn,
          source: "browser" as const,
        },
        ...items.filter(
          (item) => item.text.trim().toLowerCase() !== clean.toLowerCase(),
        ),
      ].slice(0, 10),
    );
  };

  const visuals = useMemo(
    () => visualCatalog(visualAssets, snapshot),
    [snapshot, visualAssets],
  );
  const themeVariables = useMemo(
    () => preferenceCssVariables(preferences),
    [preferences],
  );
  useEffect(() => {
    const root = document.documentElement;
    const previous = new Map<string, string>();
    for (const [name, value] of Object.entries(themeVariables)) {
      previous.set(name, root.style.getPropertyValue(name));
      root.style.setProperty(name, String(value));
    }
    return () => {
      for (const [name, value] of previous) {
        if (value) root.style.setProperty(name, value);
        else root.style.removeProperty(name);
      }
    };
  }, [themeVariables]);
  const appStyle = themeVariables as CSSProperties;
  const desktopRailPresentation = railPresentation(
    preferences.showLeftRail,
    preferences.desktopRailMode,
  );
  const leftRailVisible = isMobileLayout
    ? mobileRailOpen
    : desktopRailPresentation !== "hidden";
  const railMode = isMobileLayout ? "expanded" : preferences.desktopRailMode;
  const activeStories = Math.max(
    activeStoryCount(stories),
    snapshot && !snapshot.story.is_archived ? 1 : 0,
  );
  const isFreshInstallation = sync === "Idle" && stories.length === 0;
  const showSetupSurface = isFreshInstallation || setupRouteOpen;
  const showInstallationOnboarding =
    showSetupSurface &&
    setupReadinessState === "ready" &&
    setupReadiness !== null;

  useEffect(() => {
    if (!showSetupSurface) return;
    setStoryLibraryOpen(false);
    if (!setupRouteOpen || isFreshInstallation) return;

    const activeStory = stories.find((story) => !story.is_archived);
    if (activeStory && !storyIdRef.current) {
      setStoryId(activeStory.id);
      setSelectedTab("history");
    }
  }, [isFreshInstallation, setupRouteOpen, showSetupSurface, stories]);

  return (
    <div
      className={`app-shell rail-${desktopRailPresentation} ${mobileRailOpen ? "mobile-rail-open" : "mobile-rail-closed"} ${preferences.showInspector ? "" : "inspector-hidden"} ${preferences.wrapTranscript ? "" : "transcript-nowrap"}`}
      data-density={preferences.density}
      data-accent="custom"
      style={appStyle}
    >
      <TopBar
        snapshot={snapshot}
        sync={sync}
        syncLabel={syncLabel}
        syncTitle={t("notifications:sync.connection", { status: syncLabel })}
        leftRailVisible={leftRailVisible}
        showInspector={preferences.showInspector}
        onToggleLeftRail={toggleLeftRail}
        onToggleInspector={toggleInspector}
        onOpen={openOverlay}
        onOpenSetup={openSetupReview}
        modelSettings={modelSettings}
        translationCenterOpen={translationCenterOpen}
        onTranslationCenterOpenChange={setTranslationRouteOpen}
        onQuickSave={quickSave}
        onQuickLoad={quickLoad}
        saveBusy={sending}
        hasSaves={Boolean(snapshot?.panels.saves.length)}
      />
      <div className="workspace">
        {mobileRailOpen && (
          <button
            type="button"
            className="mobile-rail-backdrop"
            aria-label={t("common:close")}
            onClick={() => setMobileRailOpen(false)}
          />
        )}
        <LeftRail
          activeStoryCount={activeStories}
          presentation={railMode}
          snapshot={snapshot}
          selectedTab={selectedTab}
          healthText={healthText}
          onOpenStoryLibrary={openStoryLibrary}
          onSelectTab={selectModuleTab}
          onToggleMode={toggleRailMode}
          busyStoryId={storyMutatingId}
          timeline={timeline}
          onForkBranch={forkBranch}
          onRenameBranch={renameBranch}
          onCheckoutBranch={checkoutBranch}
        />
        <main className="center-stage">
          {showInstallationOnboarding ? (
            <InstallationOnboarding
              readiness={setupReadiness}
              preferences={preferences}
              onPreferencesChange={setPreferences}
              reopened={setupRouteOpen && !isFreshInstallation}
              onConfigure={openInstallationConfiguration}
              onStartStory={() =>
                setupRouteOpen && !isFreshInstallation
                  ? navigateAppRoute(
                      resolveAppRoute(null, storiesRef.current),
                      "replace",
                    )
                  : openOverlay("new-story")
              }
              onRetry={() => void refreshSetupReadiness()}
            />
          ) : showSetupSurface && setupReadinessState === "loading" ? (
            <InstallationReadinessPending />
          ) : showSetupSurface && setupReadinessState === "error" ? (
            <InstallationReadinessError
              onRetry={() => void refreshSetupReadiness()}
            />
          ) : (
            <>
              <section
                className="transcript-panel"
                aria-labelledby="story-surface-title"
              >
                <h1 className="sr-only" id="story-surface-title">
                  {snapshot?.story.name || t("notifications:surface.yourStory")}
                </h1>
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
                  storyLanguage={snapshot?.story.language ?? ""}
                  modelSettings={modelSettings}
                  messages={snapshot?.messages ?? []}
                  hiddenBeforeId={hiddenBeforeMessageId}
                  pendingTurn={pendingTurn}
                  timeline={timeline}
                  timelineBusy={sending || Boolean(storyMutatingId)}
                  showDiagnostics={preferences.showGenerationDiagnostics}
                  onCheckoutBranch={checkoutBranch}
                  onRestoreDecision={restoreDecision}
                />
              </section>

              {snapshot && snapshot.choices.length > 0 && (
                <section
                  className="inline-choice-panel"
                  aria-label={t("notifications:surface.suggestedActions")}
                >
                  <SuggestedActions
                    choices={snapshot.choices}
                    snapshot={snapshot}
                    disabled={sending}
                    showDetails={preferences.showChoiceDetails}
                    onChoice={sendChoice}
                    onDraft={setDraft}
                  />
                </section>
              )}

              {snapshot && activeMiniGame && (
                <MiniGameHost
                  instance={activeMiniGame}
                  busy={miniGameBusy || sending}
                  error={miniGameError}
                  onInput={sendMiniGameInput}
                />
              )}

              <Composer
                draft={draft}
                mode={mode}
                disabled={
                  sending ||
                  !snapshot ||
                  Boolean(
                    activeMiniGame &&
                    activeMiniGame.runtime.phase !== "resolved",
                  )
                }
                notice={notice}
                commandDescriptors={commandDescriptors}
                commandContext={commandContext}
                onDraftChange={setDraft}
                onModeChange={setMode}
                onSubmit={executeDraft}
                onHistoryStep={stepCommandHistory}
              />
            </>
          )}
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
            onMapTravel={handleMapTravel}
            timeline={timeline}
            onHistoryFork={restoreDecision}
            onOpenHistoryModule={openHistoryModule}
          />
        )}
      </div>
      {storyLibraryOpen && (
        <Suspense
          fallback={<DrawerLoadingFallback label={t("common:loading")} />}
        >
          <StoryLibraryDrawer
            stories={stories}
            activeStoryId={storyId}
            activeStoryTurn={snapshot?.world.current_turn}
            busyStoryId={storyMutatingId}
            onClose={closeRoutedSurface}
            onNewStory={openNewStoryFromLibrary}
            onRefresh={refreshStories}
            onSelectStory={selectStory}
            onUpdateStory={handleUpdateStory}
            onSetStoryArchived={handleSetStoryArchived}
            onDeleteStory={handleDeleteStory}
            timeFormat={preferences.timeFormat}
            onTimeFormatChange={(timeFormat) =>
              setPreferences((value) => ({ ...value, timeFormat }))
            }
          />
        </Suspense>
      )}
      {overlay && (
        <Suspense
          fallback={<DrawerLoadingFallback label={t("common:loading")} />}
        >
          <PanelDrawer
            overlay={overlay}
            snapshot={snapshot}
            preferences={preferences}
            metaResult={metaResult}
            modelSettings={modelSettings}
            modelError={modelSettingsError}
            modelBusy={modelSaving}
            setupReadiness={setupReadiness}
            setupReadinessState={setupReadinessState}
            initialSettingsSection={optionsInitialSection}
            visualProfile={visualAssets?.profile ?? null}
            visualAssets={visuals.assets}
            visualJobs={visualAssets?.jobs ?? []}
            visualOperationCapabilities={effectiveRouteCapabilities(
              visualAssets?.operation_capabilities ??
                modelSettings?.image_providers.find(
                  (provider) =>
                    provider.id === modelSettings.image_generation.provider,
                )?.capabilities.operations ??
                [],
              modelSettings?.image_generation.available ?? false,
            )}
            visualOperations={visualAssets?.operations ?? []}
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
            onSetupReadinessReload={() => refreshSetupReadiness()}
            onVisualProfileSave={(payload) => saveVisualProfile(payload)}
            onVisualAssetsGenerate={(payload) =>
              generateMissingVisualAssets(payload)
            }
            onVisualAssetsReload={() => refreshVisualAssets()}
            onVisualJobCancel={(jobId) => cancelVisualJob(jobId)}
            onVisualAssetsCleanup={(dryRun) => cleanVisualAssetFiles(dryRun)}
            onVisualAssetVersionsLoad={loadVisualAssetVersions}
            onVisualAssetPromptSave={saveVisualAssetPrompt}
            onVisualAssetVersionSelect={chooseVisualAssetVersion}
            onVisualAssetSelectionStep={stepVisualSelection}
            onVisualAssetOperation={runImageOperation}
            onOpenVisualAsset={openVisualAssetEditor}
            onMapTravel={handleMapTravel}
            timeline={timeline}
            onHistoryFork={restoreDecision}
            onOpenHistoryModule={openHistoryModule}
            onRunStoryWizard={(payload) => runBrowserStoryWizard(payload)}
            onEnhanceStoryText={(payload) => runBrowserStoryEnhance(payload)}
            onCreateSave={(name) =>
              void createManualSave(name, `/save ${name}`)
            }
            onLoadSave={(save) => void loadManualSave(save)}
            onDeleteSave={(save) => void deleteManualSave(save)}
            saveFilter={saveFilter}
            onSaveFilterChange={setSaveFilter}
          />
        </Suspense>
      )}
    </div>
  );
}

function DrawerLoadingFallback({ label }: { label: string }) {
  return (
    <div className="lazy-drawer-loading" role="status" aria-live="polite">
      <span aria-hidden="true" />
      <strong>{label}</strong>
    </div>
  );
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function actionErrorMessage(error: unknown): string {
  if (error instanceof ApiRequestError && error.status === 409) {
    return i18n.t("notifications:action.stale");
  }
  const message = errorMessage(error);
  if (/stale|session/i.test(message)) {
    return i18n.t("notifications:action.stale");
  }
  return message;
}

function isSnapshotStale(snapshot: StorySnapshot, maxAgeMs = 30_000): boolean {
  const serverTime = Date.parse(snapshot.server_time);
  if (!Number.isFinite(serverTime)) return false;
  return Date.now() - serverTime > maxAgeMs;
}

function submitSnapshotChanged(
  previous: StorySnapshot,
  latest: StorySnapshot,
): boolean {
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
