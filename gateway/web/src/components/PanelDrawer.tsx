import { useEffect, useId, useMemo, useRef, useState, type FormEvent, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";
import {
  commandDescriptorsToSlashCommands,
  commandDescriptors as resolveCommandDescriptors,
} from "../commands";
import { compactText, displayTimestamp } from "../format";
import {
  draftFromModelSettings,
  hasModelRoutingChanges,
  modelRoutingIssues,
  promoteProvider,
  updateFromDraft,
  type ModelRoutingDraft,
} from "../modelRouting";
import { ModuleContent } from "./Inspector";
import type {
  AppPreferences,
  CommandDescriptor,
  MetaResult,
  ModelSettings,
  ModelDiscovery,
  ModelSettingsUpdate,
  ModuleTab,
  OverlayKind,
  SaveView,
  StoryEnhanceEnvelope,
  StoryEnhanceResponse,
  StorySnapshot,
  TimelineResponse,
  StoryWizardEnvelope,
  StoryWizardAction,
  StoryWizardResponse,
  StoryWizardResult,
  GenerateVisualAssetsRequest,
  ImageOperationCapability,
  ImageOperationView,
  VisualAsset,
  VisualAssetPromptUpdate,
  VisualAssetOperationRequest,
  VisualAssetsResponse,
  VisualAssetVersion,
  VisualGenerationJobView,
  VisualProfile,
  VisualProfileUpdate,
} from "../types";
import type { VisualCatalog } from "../visualAssets";
import type { SpatialEdge } from "../spatialMap";
import { readyAssetUrl } from "../visualAssets";
import {
  emptyProfile,
  visualProfileForStyle,
  visualStylePreset,
  visualStylePresets,
  type VisualStyleKey,
} from "../visualStylePresets";
import { VoiceAssignmentEditor } from "./VoiceAssignmentEditor";
import { SettingsWorkspace, type SettingsSection } from "./settings/SettingsWorkspace";
import type { SettingsSectionId } from "./settings/settingsRegistry";
import { GeneralSettings } from "./settings/GeneralSettings";
import { GameplaySettings } from "./settings/GameplaySettings";
import { TypographySettings } from "./settings/TypographySettings";
import { AdvancedSettings } from "./settings/AdvancedSettings";
import { ImageGenerationSettings as ImageProviderEditor } from "./settings/ImageGenerationSettings";
import type { ProviderConfigDraft } from "./settings/imageGenerationDraft";
import { buildProviderConfigUpdates } from "./settings/imageGenerationDraft";
import { CustomSelect } from "./CustomSelect";
import { visualGateReason } from "../presentation";
import i18n from "../i18n";
import { VisualAssetOperationEditor } from "./VisualAssetOperationEditor";
import { DialogDrawerShell } from "./dialog/DialogDrawerShell";
import { VisualAssetUpload } from "../features/visual-assets/upload/VisualAssetUpload";
import { NewVisualAssetUpload } from "../features/visual-assets/upload/NewVisualAssetUpload";
import { defaultMediaAssetFilters, filterMediaAssets, mediaActivity, type MediaStudioTab } from "../mediaStudioState";
import { getModelDiscovery } from "../api";

interface PanelDrawerProps {
  overlay: OverlayKind;
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  metaResult: MetaResult | null;
  modelSettings: ModelSettings | null;
  modelError: string;
  modelBusy: boolean;
  initialSettingsSection?: SettingsSectionId;
  visualProfile: VisualProfile | null;
  visualAssets: VisualAsset[];
  visualJobs: VisualGenerationJobView[];
  visualOperationCapabilities: ImageOperationCapability[];
  visualOperations: ImageOperationView[];
  visuals: VisualCatalog;
  visualAssetFocusId: string | null;
  visualProfileError: string;
  visualProfileBusy: boolean;
  selectedTab: ModuleTab;
  moduleTab?: ModuleTab | null;
  moduleFocusId?: string | null;
  commandDescriptors: CommandDescriptor[];
  busy: boolean;
  onClose: () => void;
  onPreferencesChange: (preferences: AppPreferences) => void;
  onModelSettingsSave: (payload: ModelSettingsUpdate) => Promise<void>;
  onModelSettingsReload: () => Promise<void> | void;
  onVisualProfileSave: (payload: VisualProfileUpdate) => Promise<void>;
  onVisualAssetsGenerate: (
    payload: GenerateVisualAssetsRequest,
  ) => Promise<void>;
  onVisualAssetsReload: () => Promise<void> | void;
  onVisualJobCancel: (jobId: number) => Promise<void>;
  onVisualAssetsCleanup: (dryRun?: boolean) => Promise<void>;
  onVisualAssetVersionsLoad: (assetId: string) => Promise<VisualAssetVersion[]>;
  onVisualAssetPromptSave: (
    assetId: string,
    payload: VisualAssetPromptUpdate,
  ) => Promise<void>;
  onVisualAssetVersionSelect: (
    assetId: string,
    versionId: number,
  ) => Promise<void>;
  onVisualAssetSelectionStep: (
    assetId: string,
    action: "undo" | "redo",
  ) => Promise<void>;
  onVisualAssetOperation: (
    assetId: string,
    payload: VisualAssetOperationRequest,
  ) => Promise<VisualAssetsResponse | void>;
  onOpenVisualAsset: (assetId: string) => void;
  onMapTravel: (locationName: string, route: SpatialEdge | null) => void;
  timeline?: TimelineResponse | null;
  onHistoryFork?: (sourceCommitId: string, turn: number) => void;
  onOpenHistoryModule?: (tab: "map" | "codex") => void;
  onRunStoryWizard: (
    payload: StoryWizardEnvelope,
  ) => Promise<StoryWizardResponse>;
  onEnhanceStoryText: (
    payload: StoryEnhanceEnvelope,
  ) => Promise<StoryEnhanceResponse>;
  onCreateSave: (name: string) => void;
  onLoadSave: (save: SaveView) => void;
  onDeleteSave: (save: SaveView) => void;
  saveFilter: string;
  onSaveFilterChange: (value: string) => void;
}

export function PanelDrawer({
  overlay,
  snapshot,
  preferences,
  metaResult,
  modelSettings,
  modelError,
  modelBusy,
  initialSettingsSection,
  visualProfile,
  visualAssets,
  visualJobs,
  visualOperationCapabilities,
  visualOperations,
  visuals,
  visualAssetFocusId,
  visualProfileError,
  visualProfileBusy,
  selectedTab,
  moduleTab,
  moduleFocusId,
  commandDescriptors,
  busy,
  onClose,
  onPreferencesChange,
  onModelSettingsSave,
  onModelSettingsReload,
  onVisualProfileSave,
  onVisualAssetsGenerate,
  onVisualAssetsReload,
  onVisualJobCancel,
  onVisualAssetsCleanup,
  onVisualAssetVersionsLoad,
  onVisualAssetPromptSave,
  onVisualAssetVersionSelect,
  onVisualAssetSelectionStep,
  onVisualAssetOperation,
  onOpenVisualAsset,
  onMapTravel,
  timeline,
  onHistoryFork,
  onOpenHistoryModule,
  onRunStoryWizard,
  onEnhanceStoryText,
  onCreateSave,
  onLoadSave,
  onDeleteSave,
  saveFilter,
  onSaveFilterChange,
}: PanelDrawerProps) {
  const { t } = useTranslation(["drawer", "library"]);
  if (!overlay) return null;
  const activeModuleTab = moduleTab ?? selectedTab;
  return (
    <DialogDrawerShell
      title={overlayTitle(overlay, activeModuleTab, t)}
      className={`${overlay === "module" ? "module-overlay" : ""} ${overlay === "new-story" ? "new-story-overlay" : ""} ${overlay === "options" ? "options-overlay" : ""}`}
      onClose={onClose}
    >
        {overlay === "help" && (
          <HelpContent commandDescriptors={commandDescriptors} />
        )}
        {overlay === "options" && (
          <OptionsContent
            snapshot={snapshot}
            preferences={preferences}
            modelSettings={modelSettings}
            modelError={modelError}
            modelBusy={modelBusy}
            initialSettingsSection={initialSettingsSection}
            visualProfile={visualProfile}
            visualAssets={visualAssets}
            visualJobs={visualJobs}
            visualOperationCapabilities={visualOperationCapabilities}
            visualOperations={visualOperations}
            visualAssetFocusId={visualAssetFocusId}
            visualProfileError={visualProfileError}
            visualProfileBusy={visualProfileBusy}
            onPreferencesChange={onPreferencesChange}
            onModelSettingsSave={onModelSettingsSave}
            onModelSettingsReload={onModelSettingsReload}
            onVisualProfileSave={onVisualProfileSave}
            onVisualAssetsGenerate={onVisualAssetsGenerate}
            onVisualAssetsReload={onVisualAssetsReload}
            onVisualJobCancel={onVisualJobCancel}
            onVisualAssetsCleanup={onVisualAssetsCleanup}
            onVisualAssetVersionsLoad={onVisualAssetVersionsLoad}
            onVisualAssetPromptSave={onVisualAssetPromptSave}
            onVisualAssetVersionSelect={onVisualAssetVersionSelect}
            onVisualAssetSelectionStep={onVisualAssetSelectionStep}
            onVisualAssetOperation={onVisualAssetOperation}
          />
        )}
        {overlay === "saves" && (
          <SavesContent
            snapshot={snapshot}
            busy={busy}
            saveFilter={saveFilter}
            onSaveFilterChange={onSaveFilterChange}
            onCreateSave={onCreateSave}
            onLoadSave={onLoadSave}
            onDeleteSave={onDeleteSave}
          />
        )}
        {overlay === "new-story" && (
          <NewStoryContent
            busy={busy}
            onRunStoryWizard={onRunStoryWizard}
            onEnhanceStoryText={onEnhanceStoryText}
          />
        )}
        {overlay === "meta" && <MetaContent metaResult={metaResult} />}
        {overlay === "module" && (
          <ModuleOverlayContent
            snapshot={snapshot}
            selectedTab={activeModuleTab}
            visuals={visuals}
            focusCardId={moduleFocusId}
            onOpenVisualAsset={onOpenVisualAsset}
            onMapTravel={onMapTravel}
            timeline={timeline}
            onHistoryFork={onHistoryFork}
            onOpenHistoryModule={onOpenHistoryModule}
          />
        )}
    </DialogDrawerShell>
  );
}

function overlayTitle(overlay: OverlayKind, selectedTab: ModuleTab, t: (key: string) => string): string {
  if (overlay === "help") return t("drawer:title.help");
  if (overlay === "options") return t("drawer:title.options");
  if (overlay === "saves") return t("drawer:title.saves");
  if (overlay === "meta") return t("drawer:title.meta");
  if (overlay === "module") return t(`library:tabs.${selectedTab}`);
  return t("drawer:title.newStory");
}

function HelpContent({
  commandDescriptors,
}: {
  commandDescriptors: CommandDescriptor[];
}) {
  const commands = commandDescriptorsToSlashCommands(
    resolveCommandDescriptors(commandDescriptors),
  );
  return (
    <div className="overlay-content command-help">
      {commands.map((command) => (
        <div key={command.name} className="help-row">
          <strong>{command.name}</strong>
          <span>{command.hint}</span>
          <small>{command.aliases.join(" ")}</small>
        </div>
      ))}
    </div>
  );
}

function OptionsContent({
  snapshot,
  preferences,
  modelSettings,
  modelError,
  modelBusy,
  initialSettingsSection,
  visualProfile,
  visualAssets,
  visualJobs,
  visualOperationCapabilities,
  visualOperations,
  visualAssetFocusId,
  visualProfileError,
  visualProfileBusy,
  onPreferencesChange,
  onModelSettingsSave,
  onModelSettingsReload,
  onVisualProfileSave,
  onVisualAssetsGenerate,
  onVisualAssetsReload,
  onVisualJobCancel,
  onVisualAssetsCleanup,
  onVisualAssetVersionsLoad,
  onVisualAssetPromptSave,
  onVisualAssetVersionSelect,
  onVisualAssetSelectionStep,
  onVisualAssetOperation,
}: {
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  modelSettings: ModelSettings | null;
  modelError: string;
  modelBusy: boolean;
  initialSettingsSection?: SettingsSectionId;
  visualProfile: VisualProfile | null;
  visualAssets: VisualAsset[];
  visualJobs: VisualGenerationJobView[];
  visualOperationCapabilities: ImageOperationCapability[];
  visualOperations: ImageOperationView[];
  visualAssetFocusId: string | null;
  visualProfileError: string;
  visualProfileBusy: boolean;
  onPreferencesChange: (preferences: AppPreferences) => void;
  onModelSettingsSave: (payload: ModelSettingsUpdate) => Promise<void>;
  onModelSettingsReload: () => Promise<void> | void;
  onVisualProfileSave: (payload: VisualProfileUpdate) => Promise<void>;
  onVisualAssetsGenerate: (
    payload: GenerateVisualAssetsRequest,
  ) => Promise<void>;
  onVisualAssetsReload: () => Promise<void> | void;
  onVisualJobCancel: (jobId: number) => Promise<void>;
  onVisualAssetsCleanup: (dryRun?: boolean) => Promise<void>;
  onVisualAssetVersionsLoad: (assetId: string) => Promise<VisualAssetVersion[]>;
  onVisualAssetPromptSave: (
    assetId: string,
    payload: VisualAssetPromptUpdate,
  ) => Promise<void>;
  onVisualAssetVersionSelect: (
    assetId: string,
    versionId: number,
  ) => Promise<void>;
  onVisualAssetSelectionStep: (
    assetId: string,
    action: "undo" | "redo",
  ) => Promise<void>;
  onVisualAssetOperation: (
    assetId: string,
    payload: VisualAssetOperationRequest,
  ) => Promise<VisualAssetsResponse | void>;
}) {
  const { t } = useTranslation(["options", "common", "drawer", "settings_ui"]);
  const mapBackground = visualAssets.find((asset) => asset.kind === "map_background");
  const mapIcons = visualAssets.filter((asset) => asset.kind === "map_icon");
  const readyMapIcons = mapIcons.filter((asset) => asset.status === "ready").length;

  const sections: SettingsSection[] = [
    {
      id: "appearance",
      content: <GeneralSettings preferences={preferences} onChange={onPreferencesChange} />,
    },
    {
      id: "typography",
      content: <TypographySettings preferences={preferences} onChange={onPreferencesChange} />,
    },
    {
      id: "gameplay",
      content: <GameplaySettings preferences={preferences} onChange={onPreferencesChange} />,
    },
    {
      id: "audio",
      content: snapshot ? <div data-setting-id="speech-mode"><VoiceAssignmentEditor storyId={snapshot.story.id} language={snapshot.story.language} revision={snapshot.version.revision} protagonist={snapshot.character} npcs={snapshot.panels.npcs} heading={t("drawer:audio.heading")} /></div> : <p className="empty-copy">{t("drawer:audio.empty")}</p>,
    },
    {
      id: "visuals",
      content: <div className="visual-settings-stack">
        <article className="map-art-settings" data-setting-id="map-art">
          <div><strong>{t("drawer:mapArt.title")}</strong><p>{t("drawer:mapArt.desc")}</p></div>
          <span className="settings-status">{mapBackground?.status === "ready" ? t("drawer:mapArt.ready") : mapBackground?.generation_eligible ? t("drawer:mapArt.queued") : t("drawer:mapArt.waiting")} · {t("drawer:mapArt.icons", { ready: readyMapIcons, total: mapIcons.length })}</span>
        </article>
        <div data-setting-id="visual-profile"><VisualDirectionSettings storyId={snapshot?.story.id || ""} profile={visualProfile} assets={visualAssets} jobs={visualJobs} operations={visualOperations} routeOperationCapabilities={visualOperationCapabilities} focusedAssetId={visualAssetFocusId} error={visualProfileError} busy={visualProfileBusy} onSave={onVisualProfileSave} onGenerate={onVisualAssetsGenerate} onReload={onVisualAssetsReload} onJobCancel={onVisualJobCancel} onCleanup={onVisualAssetsCleanup} onVersionsLoad={onVisualAssetVersionsLoad} onAssetPromptSave={onVisualAssetPromptSave} onVersionSelect={onVisualAssetVersionSelect} onSelectionStep={onVisualAssetSelectionStep} onAssetOperation={onVisualAssetOperation} /></div>
      </div>,
    },
    {
      id: "preferences",
      content: <AdvancedSettings scope="player" preferences={preferences} snapshot={snapshot} modelSettings={modelSettings} busy={modelBusy} onChange={onPreferencesChange} onReloadConfiguration={onModelSettingsReload} />,
    },
    {
      id: "operator",
      content: <div className="operator-settings-stack">
        <p className="operator-security-note" data-setting-id="operator-configuration">{t("settings_ui:operator.security")}</p>
        <div data-setting-id="provider-order"><ModelRoutingSettings modelSettings={modelSettings} modelError={modelError} busy={modelBusy} onSave={onModelSettingsSave} onReload={onModelSettingsReload} /></div>
        <AdvancedSettings scope="operator" preferences={preferences} snapshot={snapshot} modelSettings={modelSettings} busy={modelBusy} onChange={onPreferencesChange} onReloadConfiguration={onModelSettingsReload} />
      </div>,
    },
  ];

  return <div className="overlay-content options-content"><SettingsWorkspace sections={sections} initialSection={visualAssetFocusId ? "visuals" : initialSettingsSection ?? "appearance"} /></div>;
}

function VisualDirectionSettings({
  storyId,
  profile,
  assets,
  jobs,
  operations,
  routeOperationCapabilities,
  focusedAssetId,
  error,
  busy,
  onSave,
  onGenerate,
  onReload,
  onJobCancel,
  onCleanup,
  onVersionsLoad,
  onAssetPromptSave,
  onVersionSelect,
  onSelectionStep,
  onAssetOperation,
}: {
  storyId: string;
  profile: VisualProfile | null;
  assets: VisualAsset[];
  jobs: VisualGenerationJobView[];
  operations: ImageOperationView[];
  routeOperationCapabilities: ImageOperationCapability[];
  focusedAssetId: string | null;
  error: string;
  busy: boolean;
  onSave: (payload: VisualProfileUpdate) => Promise<void>;
  onGenerate: (payload: GenerateVisualAssetsRequest) => Promise<void>;
  onReload: () => Promise<void> | void;
  onJobCancel: (jobId: number) => Promise<void>;
  onCleanup: (dryRun?: boolean) => Promise<void>;
  onVersionsLoad: (assetId: string) => Promise<VisualAssetVersion[]>;
  onAssetPromptSave: (
    assetId: string,
    payload: VisualAssetPromptUpdate,
  ) => Promise<void>;
  onVersionSelect: (assetId: string, versionId: number) => Promise<void>;
  onSelectionStep: (assetId: string, action: "undo" | "redo") => Promise<void>;
  onAssetOperation: (
    assetId: string,
    payload: VisualAssetOperationRequest,
  ) => Promise<VisualAssetsResponse | void>;
}) {
  const { t } = useTranslation(["server", "common", "drawer"]);
  const [draft, setDraft] = useState<VisualProfileUpdate>(() =>
    profileDraft(profile),
  );
  const [selectedAssetId, setSelectedAssetId] = useState(focusedAssetId ?? "");
  const [assetDraft, setAssetDraft] = useState<VisualAssetPromptUpdate>({
    prompt: "",
    negative_prompt: "",
  });
  const [versions, setVersions] = useState<VisualAssetVersion[]>([]);
  const [versionIndex, setVersionIndex] = useState(0);
  const [versionsBusy, setVersionsBusy] = useState(false);
  const [saveError, setSaveError] = useState("");
  const [mediaTab, setMediaTab] = useState<MediaStudioTab>("library");
  const [mediaFilters, setMediaFilters] = useState(defaultMediaAssetFilters);
  const readyCount = assets.filter((asset) => asset.status === "ready").length;
  const pendingCount = assets.filter(
    (asset) => asset.status !== "ready",
  ).length;
  const activeJobs = jobs.filter(
    (job) => job.status === "queued" || job.status === "running",
  );
  const filteredAssets = useMemo(() => filterMediaAssets(assets, mediaFilters), [assets, mediaFilters]);
  const activityJobs = useMemo(() => mediaActivity(jobs), [jobs]);
  const selectedAsset = useMemo(
    () =>
      assets.find((asset) => asset.id === selectedAssetId) ??
      assets.find((asset) => asset.id === focusedAssetId) ??
      assets[0] ??
      null,
    [assets, focusedAssetId, selectedAssetId],
  );
  const selectedImageUrl = readyAssetUrl(selectedAsset);
  const activeVersion =
    versions[Math.min(versionIndex, Math.max(versions.length - 1, 0))] ?? null;
  const selectedJobActive = Boolean(
    selectedAsset &&
      activeJobs.some((job) => job.asset_id === selectedAsset.id),
  );
  const loadedAssetId = useRef("");

  useEffect(() => {
    setDraft(profileDraft(profile));
    setSaveError("");
  }, [profile]);

  useEffect(() => {
    if (focusedAssetId) setSelectedAssetId(focusedAssetId);
  }, [focusedAssetId]);

  useEffect(() => {
    if (!selectedAssetId && assets[0]) setSelectedAssetId(assets[0].id);
  }, [assets, selectedAssetId]);

  useEffect(() => {
    const asset = selectedAsset;
    if (!asset) {
      loadedAssetId.current = "";
      setAssetDraft({ prompt: "", negative_prompt: "" });
      setVersions([]);
      return;
    }
    const assetChanged = loadedAssetId.current !== asset.id;
    loadedAssetId.current = asset.id;
    if (assetChanged) {
      setAssetDraft({
        prompt: asset.prompt,
        negative_prompt: asset.negative_prompt,
      });
    }
    let cancelled = false;
    setVersionsBusy(true);
    setSaveError("");
    const previouslyShownId = assetChanged ? null : activeVersion?.id ?? null;
    onVersionsLoad(asset.id)
      .then((nextVersions) => {
        if (cancelled) return;
        setVersions(nextVersions);
        const preferredVersionId =
          asset.selected_version_id ?? previouslyShownId ?? nextVersions[0]?.id;
        const preferredIndex = nextVersions.findIndex(
          (version) => version.id === preferredVersionId,
        );
        setVersionIndex(preferredIndex >= 0 ? preferredIndex : 0);
      })
      .catch((failure) => {
        if (cancelled) return;
        setVersions([]);
        setSaveError(
          failure instanceof Error ? failure.message : String(failure),
        );
      })
      .finally(() => {
        if (!cancelled) setVersionsBusy(false);
      });
    return () => {
      cancelled = true;
    };
  }, [
    onVersionsLoad,
    selectedAsset?.id,
    selectedAsset?.selected_version_id,
    selectedAsset?.status,
  ]);

  const update = <K extends keyof VisualProfileUpdate>(
    key: K,
    value: VisualProfileUpdate[K],
  ) => {
    setSaveError("");
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const save = async () => {
    setSaveError("");
    try {
      await onSave(draft);
    } catch (saveFailure) {
      setSaveError(
        saveFailure instanceof Error
          ? saveFailure.message
          : String(saveFailure),
      );
    }
  };

  const generate = async (payload: GenerateVisualAssetsRequest) => {
    setSaveError("");
    try {
      await onGenerate(payload);
    } catch (failure) {
      setSaveError(
        failure instanceof Error ? failure.message : String(failure),
      );
    }
  };

  const cancelJob = async (jobId: number) => {
    setSaveError("");
    try {
      await onJobCancel(jobId);
    } catch (failure) {
      setSaveError(
        failure instanceof Error ? failure.message : String(failure),
      );
    }
  };

  const cleanup = async (dryRun: boolean) => {
    setSaveError("");
    try {
      await onCleanup(dryRun);
    } catch (failure) {
      setSaveError(
        failure instanceof Error ? failure.message : String(failure),
      );
    }
  };

  const saveAssetPrompt = async () => {
    if (!selectedAsset) return;
    setSaveError("");
    try {
      await onAssetPromptSave(selectedAsset.id, assetDraft);
    } catch (failure) {
      setSaveError(
        failure instanceof Error ? failure.message : String(failure),
      );
    }
  };

  const regenerateSelectedAsset = async () => {
    if (!selectedAsset) return;
    setSaveError("");
    try {
      await onAssetPromptSave(selectedAsset.id, assetDraft);
      await onGenerate({
        asset_ids: [selectedAsset.id],
        force: true,
        allow_silhouette: selectedAsset.gate_state === "silhouette_available",
        limit: 1,
      });
    } catch (failure) {
      setSaveError(
        failure instanceof Error ? failure.message : String(failure),
      );
    }
  };

  const selectVersion = async () => {
    if (!selectedAsset || !activeVersion) return;
    setSaveError("");
    try {
      await onVersionSelect(selectedAsset.id, activeVersion.id);
    } catch (failure) {
      setSaveError(
        failure instanceof Error ? failure.message : String(failure),
      );
    }
  };

  const stepSelection = async (action: "undo" | "redo") => {
    if (!selectedAsset) return;
    setSaveError("");
    try {
      await onSelectionStep(selectedAsset.id, action);
    } catch (failure) {
      setSaveError(failure instanceof Error ? failure.message : String(failure));
    }
  };

  const generationAllowed = Boolean(
    selectedAsset &&
      (selectedAsset.generation_eligible ||
        selectedAsset.gate_state === "explicit_request_available" ||
        selectedAsset.gate_state === "silhouette_available"),
  );

  return (
    <div className="visual-direction">
      <div className="model-routing-head">
        <span>{t("drawer:visuals.title")}</span>
        <strong>
          {t("drawer:visuals.counts", { ready: readyCount, pending: pendingCount })}
          {activeJobs.length ? t("drawer:visuals.activeJobs", { count: activeJobs.length }) : ""}
        </strong>
      </div>
      {!profile ? (
        <p className="model-error">
          {error || t("drawer:visuals.empty")}
        </p>
      ) : (
        <>
          <div className="media-studio-tabs" role="tablist" aria-label={t("drawer:mediaStudio.label")}>
            {(["library", "create", "activity"] as const).map((tab) => <button key={tab} type="button" role="tab" aria-selected={mediaTab === tab} className={mediaTab === tab ? "active" : ""} onClick={() => setMediaTab(tab)}>{t(`drawer:mediaStudio.tabs.${tab}`)}</button>)}
          </div>
          {mediaTab === "library" && <>
          <div className="media-studio-filters">
            <label><span className="sr-only">{t("drawer:mediaStudio.search")}</span><input type="search" value={mediaFilters.query} onChange={(event) => setMediaFilters((current) => ({ ...current, query: event.target.value }))} placeholder={t("drawer:mediaStudio.search")} /></label>
            <select aria-label={t("drawer:mediaStudio.kind")} value={mediaFilters.kind} onChange={(event) => setMediaFilters((current) => ({ ...current, kind: event.target.value }))}><option value="all">{t("drawer:mediaStudio.allKinds")}</option>{[...new Set(assets.map((asset) => asset.kind))].map((kind) => <option key={kind} value={kind}>{t(`drawer:assetKind.${kind}`, { defaultValue: kind.replaceAll("_", " ") })}</option>)}</select>
            <select aria-label={t("drawer:mediaStudio.status")} value={mediaFilters.status} onChange={(event) => setMediaFilters((current) => ({ ...current, status: event.target.value }))}><option value="all">{t("drawer:mediaStudio.allStatuses")}</option>{[...new Set(assets.map((asset) => asset.status))].map((status) => <option key={status} value={status}>{t(`drawer:assetStatus.${status}`, { defaultValue: status })}</option>)}</select>
            <select aria-label={t("drawer:mediaStudio.location")} value={mediaFilters.location} onChange={(event) => setMediaFilters((current) => ({ ...current, location: event.target.value }))}><option value="all">{t("drawer:mediaStudio.allLocations")}</option>{[...new Set(assets.map((asset) => asset.canonical_location_id).filter(Boolean))].map((location) => <option key={location} value={location}>{location}</option>)}</select>
            <select aria-label={t("drawer:mediaStudio.entity")} value={mediaFilters.entity} onChange={(event) => setMediaFilters((current) => ({ ...current, entity: event.target.value }))}><option value="all">{t("drawer:mediaStudio.allEntities")}</option>{[...new Set(assets.map((asset) => asset.canonical_entity_id).filter(Boolean))].map((entity) => <option key={entity} value={entity}>{entity}</option>)}</select>
            <select aria-label={t("drawer:mediaStudio.turn")} value={mediaFilters.turn} onChange={(event) => setMediaFilters((current) => ({ ...current, turn: event.target.value }))}><option value="all">{t("drawer:mediaStudio.allTurns")}</option>{[...new Set(assets.map((asset) => asset.turn).filter((turn) => turn > 0))].sort((a, b) => b - a).map((turn) => <option key={turn} value={turn}>{turn}</option>)}</select>
            <select aria-label={t("drawer:mediaStudio.sort")} value={mediaFilters.sort} onChange={(event) => setMediaFilters((current) => ({ ...current, sort: event.target.value as typeof current.sort }))}><option value="recent">{t("drawer:mediaStudio.recent")}</option><option value="turn">{t("drawer:mediaStudio.turn")}</option><option value="name">{t("drawer:mediaStudio.name")}</option></select>
          </div>
          <div className="visual-asset-list media-asset-gallery">
            {filteredAssets.map((asset) => (
              <button type="button" className={`visual-asset-row ${asset.status} ${asset.id === selectedAsset?.id ? "selected" : ""}`} key={asset.id} title={asset.prompt} onClick={() => setSelectedAssetId(asset.id)}>
                {readyAssetUrl(asset) ? <img src={readyAssetUrl(asset)} alt="" /> : <span className="media-asset-placeholder" aria-hidden="true" />}
                <span>{t(`drawer:assetKind.${asset.kind}`, { defaultValue: asset.kind.replaceAll("_", " ") })}</span><strong>{asset.subject}</strong><small>{t(`drawer:canonStatus.${asset.canon_status}`, { defaultValue: asset.canon_status })} · {t(`drawer:assetStatus.${asset.status}`, { defaultValue: asset.status })}</small>
              </button>
            ))}
          </div>
          {filteredAssets.length === 0 && <p className="empty-copy">{t("drawer:mediaStudio.empty")}</p>}
          </>}
          {mediaTab === "create" && <>
          <div className="settings-grid visual-settings">
            <label>
              <span>{t("drawer:visuals.worldPrompt")}</span>
              <textarea
                value={draft.world_style_prompt}
                onChange={(event) =>
                  update("world_style_prompt", event.target.value)
                }
                rows={4}
              />
            </label>
            <label>
              <span>{t("drawer:visuals.characterPrompt")}</span>
              <textarea
                value={draft.character_style_prompt}
                onChange={(event) =>
                  update("character_style_prompt", event.target.value)
                }
                rows={4}
              />
            </label>
            <label>
              <span>{t("drawer:visuals.palette")}</span>
              <input
                value={draft.palette}
                onChange={(event) => update("palette", event.target.value)}
              />
            </label>
            <label>
              <span>{t("drawer:visuals.negativePrompt")}</span>
              <input
                value={draft.negative_prompt}
                onChange={(event) =>
                  update("negative_prompt", event.target.value)
                }
              />
            </label>
          </div>
          {storyId && <NewVisualAssetUpload storyId={storyId} onUploaded={async (assetId) => { await onReload(); setSelectedAssetId(assetId); }} />}
          </>}
          {mediaTab === "activity" && activityJobs.length > 0 && (
            <div className="visual-job-list" aria-label={t("drawer:visuals.jobs")}>
              {activityJobs.map((job) => (
                <div className={`visual-job-row ${job.status}`} key={job.id}>
                  <span>{t(`drawer:assetStatus.${job.status}`, { defaultValue: job.status })}</span>
                  <strong>{assetLabel(assets, job.asset_id)}</strong>
                  <small title={job.error || job.provider || job.updated_at}>
                    {t("drawer:visuals.attempt", { attempts: job.attempts, max: job.max_attempts || 1 })}
                    {job.provider ? ` - ${job.provider}` : ""}
                  </small>
                  {(job.status === "queued" || job.status === "running") && (
                    <button type="button" onClick={() => void cancelJob(job.id)} disabled={busy}>
                      {t("drawer:visuals.cancel")}
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
          {mediaTab === "library" && selectedAsset && (
            <div className="visual-asset-editor">
              <div className="visual-asset-preview">
                {selectedImageUrl ? (
                  <img src={activeVersion?.url || selectedImageUrl} alt="" />
                ) : (
                  <div>{t(`drawer:assetStatus.${selectedAsset.status}`, { defaultValue: selectedAsset.status })}</div>
                )}
              </div>
              <div className="visual-asset-editor-main">
                <div className="visual-asset-editor-head">
                  <span>{t(`drawer:assetKind.${selectedAsset.kind}`, { defaultValue: selectedAsset.kind.replaceAll("_", " ") })}</span>
                  <strong>{selectedAsset.subject}</strong>
                  <small title={selectedAsset.provider}>
                    {t(`drawer:canonStatus.${selectedAsset.canon_status}`, { defaultValue: selectedAsset.canon_status })} · {t(`drawer:assetStatus.${selectedAsset.status}`, { defaultValue: selectedAsset.status })}
                  </small>
                </div>
                <div className="visual-lineage-note">
                  <strong>{t(`drawer:gate.${selectedAsset.gate_state}`, { defaultValue: selectedAsset.gate_state.replaceAll("_", " ") })}</strong>
                  <span>{visualGateReason(selectedAsset, t) || t("common:missing")}</span>
                  <small>
                    {t("drawer:visuals.profileRevision", { revision: profile.revision })}
                    {selectedAsset.form_id ? ` · ${t("drawer:visuals.form", { id: compactId(selectedAsset.form_id) })}` : ""}
                    {` · ${selectedAsset.inherited ? t("drawer:visuals.inherited") : t("drawer:visuals.currentBranch")}`}
                  </small>
                </div>
                <label>
                  <span>{t("drawer:visuals.assetPrompt")}</span>
                  <textarea
                    value={assetDraft.prompt}
                    onChange={(event) =>
                      setAssetDraft((current) => ({
                        ...current,
                        prompt: event.target.value,
                      }))
                    }
                    rows={5}
                  />
                </label>
                <label>
                  <span>{t("drawer:visuals.negativePrompt")}</span>
                  <input
                    value={assetDraft.negative_prompt}
                    onChange={(event) =>
                      setAssetDraft((current) => ({
                        ...current,
                        negative_prompt: event.target.value,
                      }))
                    }
                  />
                </label>
                <div className="visual-version-bar">
                  <button
                    type="button"
                    disabled={busy || versionIndex <= 0}
                    onClick={() =>
                      setVersionIndex((value) => Math.max(0, value - 1))
                    }
                  >
                    {t("drawer:visuals.newer")}
                  </button>
                  <span>
                    {versionsBusy
                      ? t("drawer:visuals.loadingVersions")
                      : versions.length
                        ? t("drawer:visuals.position", { current: versions.length - versionIndex, total: versions.length, state: activeVersion?.id === selectedAsset.selected_version_id ? t("drawer:visuals.selected") : t("drawer:visuals.preview") })
                        : t("drawer:visuals.noVersions")}
                  </span>
                  <button
                    type="button"
                    disabled={busy || versionIndex >= versions.length - 1}
                    onClick={() =>
                      setVersionIndex((value) =>
                        Math.min(versions.length - 1, value + 1),
                      )
                    }
                  >
                    {t("drawer:visuals.older")}
                  </button>
                </div>
                {activeVersion && (
                  <div className="model-note">
                    <p>
                      {t("drawer:visuals.versionFrom", { date: displayTimestamp(activeVersion.created_at), provider: activeVersion.provider || t("drawer:visuals.unknownProvider") })}
                    </p>
                    <p>
                      {t(`drawer:canonStatus.${activeVersion.canon_status}`, { defaultValue: activeVersion.canon_status })} · {activeVersion.form_id ? `${t("drawer:visuals.form", { id: compactId(activeVersion.form_id) })} · ` : ""}
                      {activeVersion.id === selectedAsset.selected_version_id ? t("drawer:visuals.currentlySelected") : t("drawer:visuals.previewOnly")}
                    </p>
                    {activeVersion.revised_prompt ? (
                      <p>{t("drawer:visuals.revised", { prompt: activeVersion.revised_prompt })}</p>
                    ) : null}
                  </div>
                )}
                {storyId && (
                  <VisualAssetUpload
                    storyId={storyId}
                    assetId={selectedAsset.id}
                    onUploaded={async () => {
                      await onReload();
                      const nextVersions = await onVersionsLoad(selectedAsset.id);
                      setVersions(nextVersions);
                      setVersionIndex(0);
                    }}
                  />
                )}
                <div className="model-actions">
                  <button
                    type="button"
                    onClick={() => void saveAssetPrompt()}
                    disabled={busy}
                  >
                    {t("drawer:visuals.savePrompt")}
                  </button>
                  <button
                    type="button"
                    onClick={() => void selectVersion()}
                    disabled={busy || !activeVersion}
                  >
                    {t("drawer:visuals.useVersion")}
                  </button>
                  <button
                    type="button"
                    onClick={() => void stepSelection("undo")}
                    disabled={busy || !selectedAsset.can_undo_selection}
                  >
                    {t("drawer:visuals.undo")}
                  </button>
                  <button
                    type="button"
                    onClick={() => void stepSelection("redo")}
                    disabled={busy || !selectedAsset.can_redo_selection}
                  >
                    {t("drawer:visuals.redo")}
                  </button>
                  <button
                    type="button"
                    className="primary-action"
                    onClick={() => void regenerateSelectedAsset()}
                    disabled={busy || selectedJobActive || !generationAllowed}
                  >
                    {selectedJobActive
                      ? t("drawer:visuals.generating")
                      : selectedAsset.gate_state === "silhouette_available"
                        ? t("drawer:visuals.silhouette")
                        : t("drawer:visuals.regenerate")}
                  </button>
                </div>
              </div>
              {activeVersion && (activeVersion.url || selectedImageUrl) && (
                <VisualAssetOperationEditor
                  asset={selectedAsset}
                  sourceVersionId={activeVersion.id}
                  sourceUrl={activeVersion.url || selectedImageUrl}
                  prompt={assetDraft.prompt}
                  negativePrompt={assetDraft.negative_prompt}
                  routeCapabilities={routeOperationCapabilities}
                  operations={operations}
                  disabled={busy || selectedJobActive}
                  onRun={(payload) => onAssetOperation(selectedAsset.id, payload)}
                />
              )}
            </div>
          )}
          <div className="model-actions">
            <button
              type="button"
              onClick={() => void onReload()}
              disabled={busy}
            >
              {t("drawer:visuals.reload")}
            </button>
            <button
              type="button"
              onClick={() => void cleanup(true)}
              disabled={busy}
            >
              {t("drawer:visuals.previewCleanup")}
            </button>
            <button
              type="button"
              onClick={() => void cleanup(false)}
              disabled={busy}
            >
              {t("drawer:visuals.clean")}
            </button>
            <button
              type="button"
              onClick={() => void generate({ force: false, limit: 6 })}
              disabled={busy || assets.length === 0}
            >
              {t("drawer:visuals.generateMissing")}
            </button>
            <button
              type="button"
              onClick={() => void generate({ force: true, limit: 6 })}
              disabled={busy || assets.length === 0}
            >
              {t("drawer:visuals.regenerateVisible")}
            </button>
            <button
              type="button"
              className="primary-action"
              onClick={() => void save()}
              disabled={busy}
            >
              {busy ? t("drawer:visuals.saving") : t("drawer:visuals.save")}
            </button>
          </div>
          {error && <p className="model-error">{error}</p>}
          {saveError && <p className="model-error">{saveError}</p>}
          <p className="model-note">
            {t("drawer:visuals.note")}
          </p>
        </>
      )}
    </div>
  );
}

function profileDraft(profile: VisualProfile | null): VisualProfileUpdate {
  return {
    world_style_prompt: profile?.world_style_prompt ?? "",
    character_style_prompt: profile?.character_style_prompt ?? "",
    negative_prompt: profile?.negative_prompt ?? "",
    palette: profile?.palette ?? "",
  };
}

function assetLabel(assets: VisualAsset[], assetId: string): string {
  const asset = assets.find((item) => item.id === assetId);
  if (!asset) return assetId;
  return `${asset.kind}: ${asset.subject}`;
}

const aiSections = ["connections", "models", "routing", "images", "diagnostics"] as const;
type AiSection = (typeof aiSections)[number];

function ModelRoutingSettings({
  modelSettings,
  modelError,
  busy,
  onSave,
  onReload,
}: {
  modelSettings: ModelSettings | null;
  modelError: string;
  busy: boolean;
  onSave: (payload: ModelSettingsUpdate) => Promise<void>;
  onReload: () => Promise<void> | void;
}) {
  const { t } = useTranslation("drawer");
  const [draft, setDraft] = useState<ModelRoutingDraft | null>(() =>
    modelSettings ? draftFromModelSettings(modelSettings) : null,
  );
  const [saveError, setSaveError] = useState("");
  const [saveMessage, setSaveMessage] = useState("");
  const [providerConfigs, setProviderConfigs] = useState<Record<string, ProviderConfigDraft>>({});
  const [dirtyProviderIds, setDirtyProviderIds] = useState<Set<string>>(new Set());
  const [bridgeToken, setBridgeToken] = useState("");
  const [clearBridgeToken, setClearBridgeToken] = useState(false);
  const [discovery, setDiscovery] = useState<ModelDiscovery | null>(null);
  const [discoveryBusy, setDiscoveryBusy] = useState(false);
  const [discoveryError, setDiscoveryError] = useState("");
  const [activeAiSection, setActiveAiSection] = useState<AiSection>("connections");
  const [aiNavHorizontal, setAiNavHorizontal] = useState(false);
  const resetProviderConfigs = (settings: ModelSettings | null) => {
    setProviderConfigs(Object.fromEntries((settings?.image_providers ?? []).map((provider) => [provider.id, { baseUrl: provider.base_url, apiVersion: provider.api_version ?? "", models: provider.models.join(", "), apiKey: "", clearApiKey: false }])));
    setDirtyProviderIds(new Set()); setBridgeToken(""); setClearBridgeToken(false);
  };

  useEffect(() => {
    setDraft(modelSettings ? draftFromModelSettings(modelSettings) : null);
    resetProviderConfigs(modelSettings);
    setSaveError("");
  }, [modelSettings]);

  const refreshDiscovery = async (refresh = false) => {
    setDiscoveryBusy(true); setDiscoveryError("");
    try { setDiscovery(await getModelDiscovery(refresh)); } catch (error) { setDiscoveryError(error instanceof Error ? error.message : String(error)); }
    finally { setDiscoveryBusy(false); }
  };

  useEffect(() => { void refreshDiscovery(); }, []);

  useEffect(() => {
    const media = window.matchMedia("(max-width: 1120px)");
    const updateOrientation = () => setAiNavHorizontal(media.matches);
    updateOrientation();
    media.addEventListener("change", updateOrientation);
    return () => media.removeEventListener("change", updateOrientation);
  }, []);

  const moveAiTab = (event: KeyboardEvent<HTMLButtonElement>, current: AiSection) => {
    const previousKey = aiNavHorizontal ? "ArrowLeft" : "ArrowUp";
    const nextKey = aiNavHorizontal ? "ArrowRight" : "ArrowDown";
    if (![previousKey, nextKey, "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    const currentIndex = aiSections.indexOf(current);
    const nextIndex = event.key === "Home" ? 0 : event.key === "End" ? aiSections.length - 1 : event.key === previousKey ? (currentIndex - 1 + aiSections.length) % aiSections.length : (currentIndex + 1) % aiSections.length;
    const nextSection = aiSections[nextIndex];
    const tabList = event.currentTarget.parentElement;
    setActiveAiSection(nextSection);
    window.requestAnimationFrame(() => tabList?.querySelector<HTMLButtonElement>(`[data-ai-section="${nextSection}"]`)?.focus());
  };

  const providerIds =
    modelSettings?.providers.map((provider) => provider.id) ?? [];
  const activeProvider =
    draft?.providerPriority[0] ?? modelSettings?.active.provider ?? "";

  const updateDraft = (
    updater: (value: ModelRoutingDraft) => ModelRoutingDraft,
  ) => {
    setSaveError("");
    setSaveMessage("");
    setDraft((value) => (value ? updater(value) : value));
  };

  const updateProvider = (
    id: string,
    patch: Partial<ModelRoutingDraft["providers"][string]>,
  ) => {
    updateDraft((value) => ({
      ...value,
      providers: {
        ...value.providers,
        [id]: { ...value.providers[id], ...patch },
      },
    }));
  };
  const updateImageGeneration = (
    patch: Partial<ModelRoutingDraft["imageGeneration"]>,
  ) => {
    updateDraft((value) => ({
      ...value,
      imageGeneration: { ...value.imageGeneration, ...patch },
    }));
  };

  const save = async () => {
    if (!modelSettings || !draft) return;
    setSaveError("");
    setSaveMessage("");
    try {
      const payload = updateFromDraft(modelSettings, draft);
      payload.image_generation = { ...payload.image_generation, imagegen_bridge_token: bridgeToken || undefined, clear_imagegen_bridge_token: clearBridgeToken || undefined, provider_configs: buildProviderConfigUpdates(modelSettings.image_providers, providerConfigs, dirtyProviderIds) };
      await onSave(payload);
      setSaveMessage(t("models.saved"));
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : String(error));
    }
  };

  if (!modelSettings || !draft) {
    return (
      <div className="model-routing">
        <div className="model-routing-head">
          <span>{t("models.title")}</span>
          <strong>{t("models.unavailable")}</strong>
        </div>
        {modelError && <p className="model-error">{modelError}</p>}
        <div className="model-actions">
          <button type="button" onClick={() => void onReload()} disabled={busy}>
            {t("models.reload")}
          </button>
        </div>
      </div>
    );
  }

  const issues = modelRoutingIssues(modelSettings, draft, { providerConfigs, bridgeToken, clearBridgeToken });
  const dirty = hasModelRoutingChanges(modelSettings, draft);
  const providerConfigDirty = dirtyProviderIds.size > 0 || Boolean(bridgeToken || clearBridgeToken);
  const selectedImageProviderId = modelSettings.image_generation.provider.toLowerCase();
  const selectedImageProvider = modelSettings.image_providers.find(
    (provider) => provider.id === selectedImageProviderId,
  );
  const codexImageProvider = [
    "codex-oauth",
    "imagegen-bridge",
    "imagegen_bridge",
    "bridge-native",
  ].includes(selectedImageProviderId);
  const selectedImageProviderReady = codexImageProvider
    ? Boolean(
        selectedImageProvider?.configured ||
          modelSettings.image_generation.imagegen_bridge_url.trim(),
      )
    : Boolean(
        selectedImageProvider?.configured ||
          modelSettings.image_generation.api_key_configured,
      );
  const revision = modelSettings.config_revision
    ? modelSettings.config_revision.slice(0, 12)
    : t("models.unknown");
  const discoveredModels = Object.fromEntries(
    (discovery?.sources ?? []).flatMap((source) =>
      source.id === "imagegen-bridge"
        ? [[source.id, source.models], ["codex-oauth", source.models]]
        : [[source.id, source.models]],
    ),
  );
  const allDiscoveredModels = Object.values(discoveredModels).flat();
  const providerModels = (providerId: string, models: string[]) =>
    [...new Set([...models, ...(discoveredModels[providerId] ?? [])])];

  return (
    <div className="model-routing">
      <div className="model-routing-head">
        <span>{t("models.title")}</span>
        <strong>{t("models.shared", { revision })}</strong>
      </div>
      <div className="model-active-strip">
        <div>
          <span>{t("models.effective")}</span>
          <strong>{modelSettings.active.provider || t("models.none")}</strong>
        </div>
        <div>
          <span>{t("models.narrator")}</span>
          <strong>
            {modelSettings.active.narrative_model || t("models.providerDefault")}
          </strong>
        </div>
        <div>
          <span>{t("models.path")}</span>
          <strong title={modelSettings.config_path}>
            {modelSettings.config_path}
          </strong>
        </div>
      </div>
      <div className="imagegen-status-strip">
        <div
          className={modelSettings.image_generation.available ? "ready" : "blocked"}
        >
          <span>{t("models.imageGeneration")}</span>
          <strong>{modelSettings.image_generation.available ? t("imageSettings.configured") : t("imageSettings.notConfigured")}</strong>
        </div>
        <div>
          <span>{t("models.provider")}</span>
          <strong>{modelSettings.image_generation.provider || t("models.notSet")}</strong>
        </div>
        <div>
          <span>
            {codexImageProvider
              ? t("imageSettings.bridgeConnection")
              : t("models.credential")}
          </span>
          <strong>
            {selectedImageProviderReady
              ? t("imageSettings.configured")
              : t("imageSettings.notConfigured")}
          </strong>
        </div>
      </div>
      <p className="model-note">
        {t("models.note")}
      </p>
      <div className="operator-console-body">
        <div className="model-section-nav" role="tablist" aria-label={t("models.title")} aria-orientation={aiNavHorizontal ? "horizontal" : "vertical"}>
          {aiSections.map((section) => <button id={`model-tab-${section}`} data-ai-section={section} key={section} type="button" role="tab" aria-selected={activeAiSection === section} aria-controls={`model-section-${section}`} tabIndex={activeAiSection === section ? 0 : -1} className={activeAiSection === section ? "active" : ""} onClick={() => setActiveAiSection(section)} onKeyDown={(event) => moveAiTab(event, section)}><span>{t(`modelSections.${section}`)}</span><small>{t(`modelSectionDescriptions.${section}`)}</small></button>)}
        </div>
        <div className="operator-console-panel">
          <header className="operator-panel-heading">
            <h4>{t(`modelSections.${activeAiSection}`)}</h4>
            <p>{t(`modelSectionDescriptions.${activeAiSection}`)}</p>
          </header>
      {activeAiSection === "routing" && <div className="settings-grid" id="model-section-routing" role="tabpanel" aria-labelledby="model-tab-routing">
        <label>
          <span>{t("models.priority")}</span>
          <CustomSelect
            value={activeProvider}
            ariaLabel={t("models.priority")}
            onChange={(nextProvider) =>
              updateDraft((value) => ({
                ...value,
                providerPriority: promoteProvider(
                  value.providerPriority,
                  providerIds,
                  nextProvider,
                ),
                providers: {
                  ...value.providers,
                  [nextProvider]: {
                    ...value.providers[nextProvider],
                    enabled: true,
                  },
                },
              }))
            }
            options={modelSettings.providers.map((provider) => ({ value: provider.id, label: provider.label }))}
          />
        </label>
      </div>}
      {activeAiSection === "models" && <div className="settings-grid" id="model-section-models" role="tabpanel" aria-labelledby="model-tab-models">
        <label>
          <span>{t("models.utility")}</span>
          <ModelInput
            value={draft.utilityModel}
            options={providerModels(activeProvider, modelSettings.utility_models)}
            onChange={(value) =>
              updateDraft((draft) => ({ ...draft, utilityModel: value }))
            }
          />
        </label>
        <label>
          <span>{t("models.repair")}</span>
          <ModelInput
            value={draft.repairModel}
            options={providerModels(activeProvider, modelSettings.repair_models)}
            onChange={(value) =>
              updateDraft((draft) => ({ ...draft, repairModel: value }))
            }
          />
        </label>
        <label>
          <span>{t("models.repairFallbacks")}</span>
          <input
            value={draft.repairFallbackModels}
            onChange={(event) =>
              updateDraft((draft) => ({
                ...draft,
                repairFallbackModels: event.target.value,
              }))
            }
            placeholder={t("models.fallbackPlaceholder")}
          />
        </label>
      </div>}
      {activeAiSection === "images" && <div className="settings-grid" id="model-section-images" role="tabpanel" aria-labelledby="model-tab-images">
        <ImageProviderEditor catalog={modelSettings.image_providers} draft={draft.imageGeneration} providerConfigs={providerConfigs} bridgeToken={bridgeToken} clearBridgeToken={clearBridgeToken} discoveredModels={discoveredModels} onImageChange={updateImageGeneration} onProviderConfig={(id, patch) => { setDirtyProviderIds((current) => new Set(current).add(id)); setProviderConfigs((current) => ({ ...current, [id]: { ...current[id], ...patch } })); }} onBridgeToken={setBridgeToken} onClearBridgeToken={setClearBridgeToken} />
        <label>
          <span>{t("models.locationSize")}</span>
          <input
            value={draft.imageGeneration.locationSize}
            onChange={(event) =>
              updateImageGeneration({ locationSize: event.target.value })
            }
            placeholder="1536x1024"
          />
        </label>
        <label>
          <span>{t("models.characterSize")}</span>
          <input
            value={draft.imageGeneration.characterSize}
            onChange={(event) =>
              updateImageGeneration({ characterSize: event.target.value })
            }
            placeholder="1024x1024"
          />
        </label>
        <label>
          <span>{t("models.outputFormat")}</span>
          <CustomSelect value={draft.imageGeneration.outputFormat} ariaLabel={t("models.outputFormat")} onChange={(value) => updateImageGeneration({ outputFormat: value })} options={["png", "jpeg", "webp"].map((format) => ({ value: format, label: format.toUpperCase() }))} />
        </label>
        <label>
          <span>{t("models.ascii")}</span>
          <ModelInput
            value={draft.asciiModel}
            options={[...new Set([...modelSettings.ascii_models, ...allDiscoveredModels])]}
            onChange={(value) =>
              updateDraft((draft) => ({ ...draft, asciiModel: value }))
            }
          />
        </label>
        <label>
          <span>{t("models.embeddingProvider")}</span>
          <CustomSelect
            value={draft.embeddingProvider}
            ariaLabel={t("models.embeddingProvider")}
            onChange={(nextProvider) =>
              updateDraft((draft) => ({
                ...draft,
                embeddingProvider: nextProvider,
              }))
            }
            options={modelSettings.embedding_providers.map((provider) => ({ value: provider, label: provider }))}
          />
        </label>
        <label>
          <span>{t("models.embeddingModel")}</span>
          <input
            value={draft.embeddingModel}
            onChange={(event) =>
              updateDraft((draft) => ({
                ...draft,
                embeddingModel: event.target.value,
              }))
            }
          />
        </label>
      </div>
      }
      {activeAiSection === "connections" && <section id="model-section-connections" role="tabpanel" aria-labelledby="model-tab-connections">
      <div className="provider-editor-grid">
        {modelSettings.providers.map((provider) => {
          const providerDraft = draft.providers[provider.id];
          return (
            <div
              className={`provider-card ${providerDraft?.enabled ? "enabled" : ""}`}
              key={provider.id}
            >
              <label className="toggle-row">
                <span>{provider.label}</span>
                <input
                  type="checkbox"
                  checked={providerDraft?.enabled ?? provider.enabled}
                  onChange={(event) =>
                    updateProvider(provider.id, {
                      enabled: event.target.checked,
                    })
                  }
                />
              </label>
              {provider.supports_model && (
                <label>
                  <span>{t("models.model")}</span>
                  <ModelInput
                    value={providerDraft?.model ?? ""}
                    options={providerModels(provider.id, modelSettings.narrative_models)}
                    onChange={(value) =>
                      updateProvider(provider.id, { model: value })
                    }
                  />
                </label>
              )}
              {provider.supports_reasoning && (
                <label>
                  <span>{t("models.reasoning")}</span>
                  <CustomSelect
                    value={providerDraft?.reasoning ?? "off"}
                    ariaLabel={t("models.reasoningFor", { provider: provider.label })}
                    onChange={(reasoning) =>
                      updateProvider(provider.id, {
                        reasoning,
                      })
                    }
                    options={[
                      "off",
                      "none",
                      "minimal",
                      "low",
                      "medium",
                      "high",
                      "xhigh",
                    ].map((level) => ({ value: level, label: level }))}
                  />
                </label>
              )}
            </div>
          );
        })}
      </div>
      </section>}
      {activeAiSection === "diagnostics" && <section id="model-section-diagnostics" role="tabpanel" aria-labelledby="model-tab-diagnostics">
      <div className="model-facts">
        <span>{t("models.providerChain", { chain: draft.providerPriority.join(" → ") })}</span>
        <span>
          {t("models.images", { provider: draft.imageGeneration.provider || t("models.none"), model: draft.imageGeneration.model || t("models.noModel") })}
        </span>
        <span>
          {t("models.embedding", { provider: draft.embeddingProvider, model: draft.embeddingModel || t("models.default") })}
        </span>
        <span>TTS: {modelSettings.tts_status}</span>
      </div>
      <div className="model-discovery" aria-live="polite">
        <div><strong>{t("modelDiscovery.title")}</strong><span>{(discovery?.sources ?? []).map((source) => `${source.id}: ${source.status} · ${displayTimestamp(source.checked_at)}`).join(" · ") || t("modelDiscovery.unavailable")}</span></div>
        <button type="button" onClick={() => void refreshDiscovery(true)} disabled={discoveryBusy}>{discoveryBusy ? t("modelDiscovery.refreshing") : t("modelDiscovery.refresh")}</button>
        {discoveryError && <p className="model-error">{discoveryError}</p>}
      </div>
      </section>}
        </div>
      </div>
      {issues.length > 0 && (
        <div className="model-warning">
          {issues.map((issue) => (
            <span key={issue}>{issue}</span>
          ))}
        </div>
      )}
      <div className="model-actions">
        <button
          type="button"
          onClick={() => { setDraft(draftFromModelSettings(modelSettings)); resetProviderConfigs(modelSettings); }}
          disabled={busy}
        >
          {t("models.reset")}
        </button>
        <button type="button" onClick={() => void onReload()} disabled={busy}>
          {t("models.reload")}
        </button>
        <button
          type="button"
          className="primary-action"
          onClick={() => void save()}
          disabled={busy || (!dirty && !providerConfigDirty) || issues.length > 0}
        >
          {busy ? t("models.saving") : t("models.save")}
        </button>
      </div>
      {modelError && <p className="model-error">{modelError}</p>}
      {saveError && <p className="model-error">{saveError}</p>}
      {saveMessage && <p className="model-success">{saveMessage}</p>}
      <p className="model-note">
        {t("models.saveNote")}
      </p>
    </div>
  );
}

function ModelInput({
  value,
  options,
  onChange,
}: {
  value: string;
  options: string[];
  onChange: (value: string) => void;
}) {
  const listId = useId();
  return (
    <>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        list={listId}
      />
      <datalist id={listId}>
        {options.map((option) => (
          <option value={option} key={option} />
        ))}
      </datalist>
    </>
  );
}

function SavesContent({
  snapshot,
  busy,
  saveFilter,
  onSaveFilterChange,
  onCreateSave,
  onLoadSave,
  onDeleteSave,
}: {
  snapshot: StorySnapshot | null;
  busy: boolean;
  saveFilter: string;
  onSaveFilterChange: (value: string) => void;
  onCreateSave: (name: string) => void;
  onLoadSave: (save: SaveView) => void;
  onDeleteSave: (save: SaveView) => void;
}) {
  const { t } = useTranslation("drawer");
  const [name, setName] = useState("");
  const saves = snapshot?.panels.saves ?? [];
  const query = saveFilter.trim().toLowerCase();
  const filteredSaves = query
    ? saves.filter((save) =>
        `${save.name} ${save.location} ${save.turn} ${save.chapter} ${save.id}`
          .toLowerCase()
          .includes(query),
      )
    : saves;
  const currentTurn = snapshot?.world.current_turn ?? 0;
  const submitSave = () => {
    onCreateSave(name);
    setName("");
  };
  return (
    <div className="overlay-content">
      <div className="save-create">
        <label>
          <span>{t("saves.manualName")}</span>
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") submitSave();
            }}
            placeholder={t("saves.defaultName", { turn: currentTurn })}
            disabled={busy || !snapshot}
          />
        </label>
        <button
          type="button"
          className="wide-action"
          onClick={submitSave}
          disabled={busy || !snapshot}
        >
          {t("saves.create")}
        </button>
      </div>
      <label className="save-filter">
        <span>{t("saves.find")}</span>
        <input
          value={saveFilter}
          onChange={(event) => onSaveFilterChange(event.target.value)}
          placeholder="/load filter"
          disabled={busy || !snapshot || saves.length === 0}
        />
      </label>
      <div className="save-list">
        {saves.length === 0 ? (
          <div className="empty-copy">{t("saves.empty")}</div>
        ) : filteredSaves.length === 0 ? (
          <div className="empty-copy">{t("saves.noMatch")}</div>
        ) : (
          filteredSaves.map((save) => (
            <div className="save-row" key={save.id}>
              <div>
                <strong>{save.name}</strong>
                <span>
                  {t("saves.summary", { turn: save.turn, chapter: save.chapter, location: compactText(save.location || t("saves.unknown"), 32) })}
                </span>
                <small>{displayTimestamp(save.created_at)}</small>
              </div>
              <div className="save-actions">
                <button
                  type="button"
                  className="save-load-button"
                  onClick={() => onLoadSave(save)}
                  disabled={busy}
                >
                  {t("saves.load")}
                </button>
                <button
                  type="button"
                  className="save-delete-button"
                  onClick={() => onDeleteSave(save)}
                  disabled={busy}
                >
                  {t("saves.delete")}
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function MetaContent({ metaResult }: { metaResult: MetaResult | null }) {
  const { t } = useTranslation("drawer");
  if (!metaResult) {
    return (
      <div className="overlay-content">
        <p className="overlay-copy muted">{t("metaEmpty")}</p>
      </div>
    );
  }
  return (
    <div className="overlay-content meta-content">
      <div className="meta-kind">{metaResult.kind}</div>
      <h3>{metaResult.title}</h3>
      <p>{metaResult.message}</p>
    </div>
  );
}

function ModuleOverlayContent({
  snapshot,
  selectedTab,
  visuals,
  focusCardId,
  onOpenVisualAsset,
  onMapTravel,
  timeline,
  onHistoryFork,
  onOpenHistoryModule,
}: {
  snapshot: StorySnapshot | null;
  selectedTab: ModuleTab;
  visuals: VisualCatalog;
  focusCardId?: string | null;
  onOpenVisualAsset?: (assetId: string) => void;
  onMapTravel?: (locationName: string, route: SpatialEdge | null) => void;
  timeline?: TimelineResponse | null;
  onHistoryFork?: (sourceCommitId: string, turn: number) => void;
  onOpenHistoryModule?: (tab: "map" | "codex") => void;
}) {
  const { t } = useTranslation("drawer");
  if (!snapshot) {
    return (
      <div className="overlay-content">
        <p className="overlay-copy muted">
          {t("moduleEmpty")}
        </p>
      </div>
    );
  }
  return (
    <div className="overlay-content module-content">
      <ModuleContent
        tab={selectedTab}
        snapshot={snapshot}
        visuals={visuals}
        expanded
        focusCardId={focusCardId}
        onOpenVisualAsset={onOpenVisualAsset}
        onMapTravel={onMapTravel}
        timeline={timeline}
        onHistoryFork={onHistoryFork}
        onOpenHistoryModule={onOpenHistoryModule}
      />
    </div>
  );
}

function NewStoryContent({
  busy,
  onRunStoryWizard,
  onEnhanceStoryText,
}: {
  busy: boolean;
  onRunStoryWizard: (
    payload: StoryWizardEnvelope,
  ) => Promise<StoryWizardResponse>;
  onEnhanceStoryText: (
    payload: StoryEnhanceEnvelope,
  ) => Promise<StoryEnhanceResponse>;
}) {
  const { t } = useTranslation(["onboarding", "wizard", "drawer"]);
  const [wizard, setWizard] = useState<StoryWizardResult | null>(null);
  const [input, setInput] = useState("");
  const [start, setStart] = useState(true);
  const [error, setError] = useState("");
  const [enhancing, setEnhancing] = useState(false);
  const [pendingPreset, setPendingPreset] = useState<StoryWizardAction | null>(null);
  const [visualStyle, setVisualStyle] = useState<VisualStyleKey>("photorealistic");
  const [customVisualProfile, setCustomVisualProfile] = useState<VisualProfileUpdate>(() => emptyProfile());
  const [operation, setOperation] = useState("");
  const [elapsedSeconds, setElapsedSeconds] = useState(0);
  const [log, setLog] = useState<Array<{ stage: string; message: string }>>([]);
  const initializedRef = useRef(false);

  useEffect(() => {
    if (!operation) {
      setElapsedSeconds(0);
      return;
    }
    const startedAt = Date.now();
    const update = () => setElapsedSeconds(Math.max(0, Math.floor((Date.now() - startedAt) / 1000)));
    update();
    const timer = window.setInterval(update, 1000);
    return () => window.clearInterval(timer);
  }, [operation]);

  const applyResponse = (response: StoryWizardResponse) => {
    const next = response.wizard;
    setWizard(next);
    setInput("");
    if (next.message) {
      setLog((items) =>
        [
          {
            stage: next.stage_label || next.stage || t("drawer:wizard.setup"),
            message: next.message,
          },
          ...items,
        ].slice(0, 4),
      );
    }
    setPendingPreset(null);
  };

  const runStep = async (payload: Omit<StoryWizardEnvelope, "start">) => {
    setError("");
    setOperation(wizardOperationLabel(wizard?.stage ?? "brief", payload));
    try {
      const response = await onRunStoryWizard({
        state: wizard?.state,
        ...payload,
        ...visualProfileForStyle(visualStyle, customVisualProfile),
        start,
      });
      applyResponse(response);
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : String(failure));
    } finally {
      setOperation("");
    }
  };

  useEffect(() => {
    if (initializedRef.current) return;
    initializedRef.current = true;
    setOperation(t("drawer:wizard.opening"));
    onRunStoryWizard({ start })
      .then((response) => {
        applyResponse(response);
      })
      .catch((failure) => {
        setError(
          failure instanceof Error ? failure.message : String(failure),
        );
      })
      .finally(() => {
        setOperation("");
      });
  }, []);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const text = input.trim();
    if (!text) {
      setError(currentInputRequiredMessage(wizard));
      return;
    }
    await runStep({ input: text });
  };

  const runAction = async (action: StoryWizardAction) => {
    if ((wizard?.stage ?? "brief") === "brief" && action.key.startsWith("preset_")) {
      setPendingPreset(action);
      setInput(action.seed || "");
      setError("");
      return;
    }
    await runStep({ action: action.key });
  };

  const stage = wizard?.stage ?? "brief";
  const phase = wizard?.phase ?? "conversation";
  const definition = storyDefinitionSummary(wizard?.definition);
  const step = wizardStep(stage);
  const selectedVisualPreset = visualStylePreset(visualStyle);

  const updateCustomVisualProfile = <K extends keyof VisualProfileUpdate>(
    key: K,
    value: VisualProfileUpdate[K],
  ) => {
    setCustomVisualProfile((current) => ({ ...current, [key]: value }));
  };

  const enhanceInput = async () => {
    if (stage === "done" || enhancing) return;
    const fallback = enhancedStoryInput(wizard, input);
    setError("");
    setEnhancing(true);
    try {
      const response = await onEnhanceStoryText({
        state: wizard?.state,
        stage,
        text: fallback,
        context: wizard?.message || "",
      });
      setInput(response.text?.trim() || fallback);
      if (response.model || response.provider) {
        setLog((items) =>
          [
            {
              stage: t("drawer:wizard.aiEnhance"),
              message: t("drawer:wizard.improved", { field: inputLabelForStage(stage).toLowerCase(), model: response.model || response.provider }),
            },
            ...items,
          ].slice(0, 8),
        );
      }
    } catch (failure) {
      setInput(fallback);
      setError(
        t("drawer:wizard.enhanceFailed", { error: failure instanceof Error ? failure.message : String(failure) }),
      );
    } finally {
      setEnhancing(false);
    }
  };

  return (
    <form
      className="overlay-content new-story-form story-wizard-form"
      onSubmit={submit}
    >
      <div className="story-wizard-status">
        <div>
          <span>
            {phase === "done"
              ? t("complete")
              : phase === "character"
                ? t("character")
                : t("step", { current: step.current, total: step.total })}
          </span>
          <strong>{wizard ? t(`wizard:stages.${wizard.stage}`, { defaultValue: wizard.stage_label || t("onboarding:loading") }) : t("onboarding:loading")}</strong>
        </div>
        <label className="checkbox-line">
          <input
            type="checkbox"
            checked={start}
            onChange={(event) => setStart(event.target.checked)}
            disabled={busy || stage === "done"}
          />
          <span>{t("start")}</span>
        </label>
      </div>

      <div className="story-wizard-grid">
        <div className="story-wizard-main">
          {operation && (
            <div className="story-wizard-generation" role="status" aria-live="polite">
              <span className="story-wizard-generation-pulse" aria-hidden="true" />
              <div>
                <strong>{generationProgressLabel(operation, elapsedSeconds)}</strong>
                <small>{t("onboarding:elapsed", { count: elapsedSeconds })}</small>
              </div>
            </div>
          )}
          <div className="story-wizard-message">
            <pre>
              {wizard?.stage === "brief" ? t("wizard:messages.brief") : wizard?.message || t("onboarding:starting")}
            </pre>
          </div>

          {wizard?.actions?.length ? (
            <div
              className="story-wizard-actions"
              aria-label={t("quick")}
            >
              {wizard.actions.map((action, index) => (
                <button
                  type="button"
                  key={action.key}
                  title={action.key}
                  aria-pressed={pendingPreset?.key === action.key}
                  onClick={() => void runAction(action)}
                  disabled={busy}
                >
                  <span>{index + 1}</span>
                  <strong>{t(`wizard:actions.${action.key}`, { defaultValue: action.label })}</strong>
                </button>
              ))}
            </div>
          ) : null}

          {pendingPreset && (
            <section className="story-wizard-confirmation" aria-label={t("confirm")}>
              <div>
                <span>{t("review")}</span>
                <strong>{t(`wizard:actions.${pendingPreset.key}`, { defaultValue: pendingPreset.label })}</strong>
                <p>{t("preset")}</p>
              </div>
              <div>
                <button type="button" onClick={() => { setPendingPreset(null); setInput(""); }} disabled={busy}>{t("common:cancel")}</button>
                <button type="button" className="primary" onClick={() => void runStep({ input: input.trim() })} disabled={busy || !input.trim()}>{t("generate")}</button>
              </div>
            </section>
          )}

          {stage !== "done" && (
            <label className="story-wizard-input">
              <span>{inputLabelForStage(stage)}</span>
              <textarea
                value={input}
                onChange={(event) => setInput(event.target.value)}
                rows={
                  stage === "brief"
                    ? 6
                    : stage === "character_background"
                      ? 4
                      : 5
                }
                placeholder={wizard?.placeholder || t("response")}
                disabled={busy}
              />
            </label>
          )}

          {error && <p className="form-error">{error}</p>}

          <div className="drawer-actions story-wizard-submit">
            <button
              type="button"
              onClick={() => void enhanceInput()}
              disabled={busy || enhancing || stage === "done"}
            >
              {enhancing ? t("enhancing") : t("enhance")}
            </button>
            {!pendingPreset && (
              <button type="submit" className="primary" disabled={busy || stage === "done"}>
              {busy ? t("drawer:wizard.working") : submitLabelForStage(stage)}
              </button>
            )}
          </div>
        </div>

        <aside className="story-wizard-side">
          <div className="story-wizard-card story-visual-style-card">
            <span>{t("visual")}</span>
            <CustomSelect
              value={visualStyle}
              ariaLabel={t("drawer:wizard.visualStyle")}
              disabled={busy || stage === "done"}
              onChange={(value) => setVisualStyle(value as VisualStyleKey)}
              options={visualStylePresets.map((preset) => ({
                value: preset.key,
                label: t(`drawer:style.presets.${preset.key}.label`),
              }))}
            />
            <p>{t(`drawer:style.presets.${selectedVisualPreset.key}.description`)}</p>
            {visualStyle === "custom" && (
              <div className="story-visual-custom-fields">
                <label>
                  <span>{t("onboarding:worldDirection")}</span>
                  <textarea
                    value={customVisualProfile.world_style_prompt}
                    onChange={(event) => updateCustomVisualProfile("world_style_prompt", event.target.value)}
                    placeholder={t("drawer:style.worldPlaceholder")}
                    rows={4}
                    disabled={busy}
                  />
                </label>
                <label>
                  <span>{t("onboarding:characterDirection")}</span>
                  <textarea
                    value={customVisualProfile.character_style_prompt}
                    onChange={(event) => updateCustomVisualProfile("character_style_prompt", event.target.value)}
                    placeholder={t("drawer:style.characterPlaceholder")}
                    rows={4}
                    disabled={busy}
                  />
                </label>
                <label>
                  <span>{t("onboarding:avoid")}</span>
                  <textarea
                    value={customVisualProfile.negative_prompt}
                    onChange={(event) => updateCustomVisualProfile("negative_prompt", event.target.value)}
                    placeholder={t("drawer:style.avoidPlaceholder")}
                    rows={3}
                    disabled={busy}
                  />
                </label>
                <label>
                  <span>{t("onboarding:palette")}</span>
                  <input
                    value={customVisualProfile.palette}
                    onChange={(event) => updateCustomVisualProfile("palette", event.target.value)}
                    placeholder={t("drawer:style.palettePlaceholder")}
                    disabled={busy}
                  />
                </label>
              </div>
            )}
            <small>{t("drawer:wizard.mapPipeline")}</small>
          </div>
          <div className="story-wizard-card">
            <span>{t("summary")}</span>
            {definition ? (
              <>
                <strong>{definition.name}</strong>
                <p>{definition.description}</p>
                <dl>
                  <dt>{t("genre")}</dt>
                  <dd>{definition.genre}</dd>
                  <dt>{t("tone")}</dt>
                  <dd>{definition.tone}</dd>
                  <dt>{t("language")}</dt>
                  <dd>{definition.language}</dd>
                  <dt>{t("world")}</dt>
                  <dd>{definition.worldName}</dd>
                  <dt>{t("combat")}</dt>
                  <dd>{definition.hasCombat ? t("drawer:wizard.enabled") : t("drawer:wizard.disabled")}</dd>
                </dl>
              </>
            ) : (
              <p>
                {t("onboarding:noDraft")}
              </p>
            )}
          </div>
          <div className="story-wizard-card">
            <span>{t("log")}</span>
            {log.length ? (
              <ol>
                {log.map((item, index) => (
                  <li key={`${item.stage}-${index}`}>
                    <strong>{item.stage}</strong>
                    <p>{compactText(item.message.replace(/\s+/g, " "), 180)}</p>
                  </li>
                ))}
              </ol>
            ) : (
              <p>{t("onboarding:emptyLog")}</p>
            )}
          </div>
        </aside>
      </div>
    </form>
  );
}

function wizardStep(stage: string): { current: number; total: number } {
  const stages = ["brief", "review_world", "review_rules", "review_stats", "confirm", "character_name", "character_background", "done"];
  return { current: Math.max(1, stages.indexOf(stage) + 1), total: stages.length - 1 };
}

function wizardOperationLabel(stage: string, payload: Omit<StoryWizardEnvelope, "start">): string {
  if (stage === "brief") return i18n.t("drawer:wizard.generating");
  if (payload.action === "create_story") return i18n.t("drawer:wizard.locking");
  if (stage.startsWith("review_") || stage === "confirm") return i18n.t("drawer:wizard.revising");
  if (stage === "character_background") return i18n.t("drawer:wizard.creating");
  return i18n.t("drawer:wizard.updating");
}

function generationProgressLabel(operation: string, elapsedSeconds: number): string {
  if (elapsedSeconds < 2) return operation;
  if (elapsedSeconds < 8) return i18n.t("drawer:wizard.worldRulesStats");
  return i18n.t("drawer:wizard.validating");
}

function currentInputRequiredMessage(wizard: StoryWizardResult | null): string {
  const stage = wizard?.stage ?? "brief";
  if (stage === "character_name") return i18n.t("drawer:wizard.required.name");
  if (stage === "character_background")
    return i18n.t("drawer:wizard.required.background");
  if (stage === "brief") return i18n.t("drawer:wizard.required.brief");
  return i18n.t("drawer:wizard.required.revision");
}

function inputLabelForStage(stage: string): string {
  switch (stage) {
    case "brief":
      return i18n.t("drawer:wizard.input.brief");
    case "character_name":
      return i18n.t("drawer:wizard.input.name");
    case "character_background":
      return i18n.t("drawer:wizard.input.background");
    default:
      return i18n.t("drawer:wizard.input.revision");
  }
}

function submitLabelForStage(stage: string): string {
  switch (stage) {
    case "brief":
      return i18n.t("drawer:wizard.submit.brief");
    case "character_name":
      return i18n.t("drawer:wizard.submit.name");
    case "character_background":
      return i18n.t("drawer:wizard.submit.background");
    default:
      return i18n.t("drawer:wizard.submit.revision");
  }
}

function enhancedStoryInput(
  wizard: StoryWizardResult | null,
  current: string,
): string {
  const stage = wizard?.stage ?? "brief";
  const text = current.trim();
  if (stage === "character_name") return text || "Mira";
  if (stage === "character_background") {
    if (text) {
      return `${text}\n\nAdd: a concrete personal stake, one useful skill, one fear, and one unresolved tie to the opening situation. Keep it playable, concise, and not overpowered.`;
    }
    return "A practical survivor with one useful skill, one unresolved debt, and a clear reason to enter the first scene instead of walking away.";
  }
  if (stage !== "brief") {
    return text
      ? `${text}\n\nPlease keep the current identity, reduce vague lore, add concrete tradeoffs, and preserve clear player agency.`
      : "Keep the current identity, reduce vague lore, add concrete tradeoffs, and preserve clear player agency.";
  }

  const preset = storyTextPresetFor(text);
  if (!text) return preset;
  return `${text}\n\nDesign constraints: compact prose, meaningful random outcomes, anti-loop memory, grounded consequences, no easy reward inflation, and choices that expose risk, scope, and likely tradeoffs.`;
}

function storyTextPresetFor(source: string): string {
  const text = source.toLowerCase();
  if (/(cyber|noir|neon|corporate|hacker)/.test(text)) {
    return "Italian cyberpunk noir with sharp dialogue, practical investigations, corporate pressure, visible consequences, compact prose, no lore sprawl, and choices that reveal risk before action.";
  }
  if (/(steam|clockwork|airship|brass)/.test(text)) {
    return "Italian steampunk mystery with industrial politics, dangerous machines, grounded travel, compact prose, social and technical problem solving, and clear costs for risky choices.";
  }
  if (/(fantasy|magic|magia|ruin|dragon|dungeon)/.test(text)) {
    return "Italian fantasy adventure with tactile places, costly magic, memorable factions, compact prose, no overpowered gifts, and choices that balance danger, discovery, and relationships.";
  }
  if (/(horror|occult|ghost|paura|orrore)/.test(text)) {
    return "Italian horror mystery with ordinary places turning unsafe, slow dread, limited resources, compact prose, no cheap shocks, and investigation choices with visible emotional and physical risk.";
  }
  return "Italian mystery adventure, compact prose, practical choices, strong anti-loop rules, no lore sprawl, no free advantages, meaningful randomness, and a first scene with a concrete problem.";
}

function storyDefinitionSummary(value: unknown): {
  name: string;
  description: string;
  genre: string;
  tone: string;
  language: string;
  worldName: string;
  hasCombat: boolean;
} | null {
  if (!value || typeof value !== "object") return null;
  const raw = value as Record<string, unknown>;
  const setting =
    typeof raw.setting === "object" && raw.setting
      ? (raw.setting as Record<string, unknown>)
      : {};
  const stats =
    typeof raw.stats_schema === "object" && raw.stats_schema
      ? (raw.stats_schema as Record<string, unknown>)
      : {};
  return {
    name: stringValue(raw.name, i18n.t("drawer:wizard.untitled")),
    description: stringValue(raw.description, i18n.t("drawer:wizard.noDescription")),
    genre: stringValue(raw.genre, "-"),
    tone: stringValue(raw.tone, "-"),
    language: stringValue(raw.language, "-"),
    worldName: stringValue(setting.world_name, "-"),
    hasCombat: Boolean(stats.has_combat),
  };
}

function stringValue(value: unknown, fallback: string): string {
  return typeof value === "string" && value.trim() ? value.trim() : fallback;
}

function compactId(value: string): string {
  if (value.length <= 20) return value;
  return `${value.slice(0, 9)}…${value.slice(-7)}`;
}
