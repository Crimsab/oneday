import { useEffect, useId, useMemo, useRef, useState, type FormEvent } from "react";
import { X } from "lucide-react";
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
import { ModuleContent, moduleTitle } from "./Inspector";
import type {
  AppPreferences,
  CommandDescriptor,
  MetaResult,
  ModelSettings,
  ModelSettingsUpdate,
  ModuleTab,
  OverlayKind,
  SaveView,
  StoryEnhanceEnvelope,
  StoryEnhanceResponse,
  StorySnapshot,
  StoryWizardEnvelope,
  StoryWizardAction,
  StoryWizardResponse,
  StoryWizardResult,
  GenerateVisualAssetsRequest,
  VisualAsset,
  VisualAssetPromptUpdate,
  VisualAssetVersion,
  VisualGenerationJobView,
  VisualProfile,
  VisualProfileUpdate,
} from "../types";
import type { VisualCatalog } from "../visualAssets";
import { readyAssetUrl } from "../visualAssets";
import { VoiceAssignmentEditor } from "./VoiceAssignmentEditor";
import { SettingsWorkspace, type SettingsSection } from "./settings/SettingsWorkspace";
import { CustomSelect } from "./CustomSelect";

interface PanelDrawerProps {
  overlay: OverlayKind;
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  metaResult: MetaResult | null;
  modelSettings: ModelSettings | null;
  modelError: string;
  modelBusy: boolean;
  visualProfile: VisualProfile | null;
  visualAssets: VisualAsset[];
  visualJobs: VisualGenerationJobView[];
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
  visualProfile,
  visualAssets,
  visualJobs,
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
  onRunStoryWizard,
  onEnhanceStoryText,
  onCreateSave,
  onLoadSave,
  onDeleteSave,
  saveFilter,
  onSaveFilterChange,
}: PanelDrawerProps) {
  if (!overlay) return null;
  const activeModuleTab = moduleTab ?? selectedTab;
  return (
    <div className="overlay-backdrop" role="presentation" onMouseDown={onClose}>
      <section
        className={`overlay-panel ${overlay === "module" ? "module-overlay" : ""} ${overlay === "new-story" ? "new-story-overlay" : ""} ${overlay === "options" ? "options-overlay" : ""}`}
        role="dialog"
        aria-modal="true"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="overlay-head">
          <h2>{overlayTitle(overlay, activeModuleTab)}</h2>
          <button
            type="button"
            className="square-button"
            onClick={onClose}
            title="Close"
          >
            <X size={16} />
          </button>
        </div>
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
            visualProfile={visualProfile}
            visualAssets={visualAssets}
            visualJobs={visualJobs}
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
          />
        )}
      </section>
    </div>
  );
}

function overlayTitle(overlay: OverlayKind, selectedTab: ModuleTab): string {
  if (overlay === "help") return "Help";
  if (overlay === "options") return "Options";
  if (overlay === "saves") return "Saves";
  if (overlay === "meta") return "Meta Command";
  if (overlay === "module") return moduleTitle(selectedTab);
  return "New Story";
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
  visualProfile,
  visualAssets,
  visualJobs,
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
}: {
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  modelSettings: ModelSettings | null;
  modelError: string;
  modelBusy: boolean;
  visualProfile: VisualProfile | null;
  visualAssets: VisualAsset[];
  visualJobs: VisualGenerationJobView[];
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
}) {
  const update = <K extends keyof AppPreferences>(
    key: K,
    value: AppPreferences[K],
  ) => {
    onPreferencesChange({ ...preferences, [key]: value });
  };

  const mapBackground = visualAssets.find((asset) => asset.kind === "map_background");
  const mapIcons = visualAssets.filter((asset) => asset.kind === "map_icon");
  const readyMapIcons = mapIcons.filter((asset) => asset.status === "ready").length;

  const sections: SettingsSection[] = [
    {
      id: "general",
      content: <div className="settings-grid general-settings">
        <label data-setting-id="density"><span>Density</span><CustomSelect value={preferences.density} ariaLabel="Interface density" onChange={(value) => update("density", value as AppPreferences["density"])} options={[{ value: "compact", label: "Compact" }, { value: "balanced", label: "Balanced" }, { value: "comfortable", label: "Comfortable" }]} /></label>
        <label data-setting-id="font-size"><span>Font size</span><CustomSelect value={preferences.fontSize} ariaLabel="Font size" onChange={(value) => update("fontSize", value as AppPreferences["fontSize"])} options={[{ value: "small", label: "Small" }, { value: "base", label: "Base" }, { value: "large", label: "Large" }]} /></label>
        <label data-setting-id="accent"><span>Accent</span><CustomSelect value={preferences.accent} ariaLabel="Accent color" onChange={(value) => update("accent", value as AppPreferences["accent"])} options={[{ value: "amber", label: "Amber" }, { value: "green", label: "Green" }, { value: "blue", label: "Blue" }, { value: "rose", label: "Rose" }]} /></label>
        <label className="toggle-row" data-setting-id="stories-sidebar"><span>Stories sidebar</span><input type="checkbox" checked={preferences.showLeftRail} onChange={(event) => update("showLeftRail", event.target.checked)} /></label>
        <label className="toggle-row" data-setting-id="inspector"><span>Inspector panel</span><input type="checkbox" checked={preferences.showInspector} onChange={(event) => update("showInspector", event.target.checked)} /></label>
        <label className="toggle-row" data-setting-id="transcript-wrap"><span>Transcript wrap</span><input type="checkbox" checked={preferences.wrapTranscript} onChange={(event) => update("wrapTranscript", event.target.checked)} /></label>
      </div>,
    },
    {
      id: "gameplay",
      content: <div className="settings-policy-list">
        <article data-setting-id="automatic-challenges"><strong>Automatic challenges</strong><p>NPC situations can open an interactive challenge. OneDay selects the family automatically, so the player never chooses the easiest mechanic.</p><span className="settings-status">Enabled</span></article>
        <article data-setting-id="timing-free"><strong>Timing-free selection</strong><p>The selector excludes reflex-only challenges when a timing-free mechanic can represent the same scene.</p><span className="settings-status">Required</span></article>
        <article data-setting-id="challenge-cooldown"><strong>Challenge cooldown</strong><p>Recent branch history reduces repetition and blocks a family during its cooldown window.</p><span className="settings-status">Active branch</span></article>
        <article className="choice-detail-setting" data-setting-id="choice-details">
          <strong>Choice details</strong>
          <p>Show intent, risk, scope, certainty, and related attributes beneath each suggested action.</p>
          <label className="settings-policy-toggle"><input type="checkbox" checked={preferences.showChoiceDetails} onChange={(event) => update("showChoiceDetails", event.target.checked)} /><span>Show details</span></label>
        </article>
      </div>,
    },
    {
      id: "audio",
      content: snapshot ? <div data-setting-id="speech-mode"><VoiceAssignmentEditor storyId={snapshot.story.id} language={snapshot.story.language} revision={snapshot.version.revision} protagonist={snapshot.character} npcs={snapshot.panels.npcs} heading="Story speech policy" /></div> : <p className="empty-copy">Select a story to configure spoken audio and character voices.</p>,
    },
    {
      id: "visuals",
      content: <div className="visual-settings-stack">
        <article className="map-art-settings" data-setting-id="map-art">
          <div><strong>Illustrated known-location map</strong><p>OneDay generates one decorative world layer and one reusable symbol for each canonical location. Routes, labels and discovery state stay in the live SVG, so generated art can never invent map facts.</p></div>
          <span className="settings-status">{mapBackground?.status === "ready" ? "Art ready" : mapBackground?.generation_eligible ? "Art queued" : "Awaits 2 locations"} · {readyMapIcons}/{mapIcons.length} icons</span>
        </article>
        <div data-setting-id="visual-profile"><VisualDirectionSettings profile={visualProfile} assets={visualAssets} jobs={visualJobs} focusedAssetId={visualAssetFocusId} error={visualProfileError} busy={visualProfileBusy} onSave={onVisualProfileSave} onGenerate={onVisualAssetsGenerate} onReload={onVisualAssetsReload} onJobCancel={onVisualJobCancel} onCleanup={onVisualAssetsCleanup} onVersionsLoad={onVisualAssetVersionsLoad} onAssetPromptSave={onVisualAssetPromptSave} onVersionSelect={onVisualAssetVersionSelect} onSelectionStep={onVisualAssetSelectionStep} /></div>
      </div>,
    },
    {
      id: "models",
      content: <div data-setting-id="provider-order"><ModelRoutingSettings modelSettings={modelSettings} modelError={modelError} busy={modelBusy} onSave={onModelSettingsSave} onReload={onModelSettingsReload} /></div>,
    },
    {
      id: "advanced",
      content: <div className="option-grid" data-setting-id="runtime-status">
        <div><span>Live updates</span><strong>{snapshot ? "SSE snapshots + turn events" : "No story selected"}</strong></div>
        <div><span>Action transport</span><strong>gateway-turn</strong></div>
        <div><span>Capabilities</span><strong>images, ascii, roll log</strong></div>
        <div><span>Theme</span><strong>Reference Amber Noir</strong></div>
      </div>,
    },
  ];

  return <div className="overlay-content options-content"><SettingsWorkspace sections={sections} /></div>;
}

function VisualDirectionSettings({
  profile,
  assets,
  jobs,
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
}: {
  profile: VisualProfile | null;
  assets: VisualAsset[];
  jobs: VisualGenerationJobView[];
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
}) {
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
  const readyCount = assets.filter((asset) => asset.status === "ready").length;
  const pendingCount = assets.filter(
    (asset) => asset.status !== "ready",
  ).length;
  const activeJobs = jobs.filter(
    (job) => job.status === "queued" || job.status === "running",
  );
  const visibleJobs = [
    ...activeJobs,
    ...jobs.filter((job) => job.status !== "queued" && job.status !== "running"),
  ].filter((job, index, list) => list.findIndex((item) => item.id === job.id) === index).slice(0, 4);
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
    if (!selectedAsset) {
      setAssetDraft({ prompt: "", negative_prompt: "" });
      setVersions([]);
      return;
    }
    setAssetDraft({
      prompt: selectedAsset.prompt,
      negative_prompt: selectedAsset.negative_prompt,
    });
    let cancelled = false;
    setVersionsBusy(true);
    setSaveError("");
    onVersionsLoad(selectedAsset.id)
      .then((nextVersions) => {
        if (cancelled) return;
        setVersions(nextVersions);
        setVersionIndex(0);
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
  }, [onVersionsLoad, selectedAsset]);

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
      const nextVersions = await onVersionsLoad(selectedAsset.id);
      setVersions(nextVersions);
      setVersionIndex(0);
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
        <span>Visual Direction</span>
        <strong>
          {readyCount} ready / {pendingCount} pending
          {activeJobs.length ? ` / ${activeJobs.length} active jobs` : ""}
        </strong>
      </div>
      {!profile ? (
        <p className="model-error">
          {error || "Select a story to edit visual prompts."}
        </p>
      ) : (
        <>
          <div className="settings-grid visual-settings">
            <label>
              <span>World/location prompt</span>
              <textarea
                value={draft.world_style_prompt}
                onChange={(event) =>
                  update("world_style_prompt", event.target.value)
                }
                rows={4}
              />
            </label>
            <label>
              <span>Character prompt</span>
              <textarea
                value={draft.character_style_prompt}
                onChange={(event) =>
                  update("character_style_prompt", event.target.value)
                }
                rows={4}
              />
            </label>
            <label>
              <span>Palette</span>
              <input
                value={draft.palette}
                onChange={(event) => update("palette", event.target.value)}
              />
            </label>
            <label>
              <span>Negative prompt</span>
              <input
                value={draft.negative_prompt}
                onChange={(event) =>
                  update("negative_prompt", event.target.value)
                }
              />
            </label>
          </div>
          <div className="visual-asset-list">
            {assets.slice(0, 8).map((asset) => (
              <button
                type="button"
                className={`visual-asset-row ${asset.status} ${asset.id === selectedAsset?.id ? "selected" : ""}`}
                key={asset.id}
                title={asset.prompt}
                onClick={() => setSelectedAssetId(asset.id)}
              >
                <span>{asset.kind}</span>
                <strong>{asset.subject}</strong>
                <small title={asset.error || asset.provider}>
                  {asset.canon_status} · {asset.status}
                  {asset.error ? " !" : ""}
                </small>
              </button>
            ))}
          </div>
          {visibleJobs.length > 0 && (
            <div className="visual-job-list" aria-label="Visual generation jobs">
              {visibleJobs.map((job) => (
                <div className={`visual-job-row ${job.status}`} key={job.id}>
                  <span>{job.status}</span>
                  <strong>{assetLabel(assets, job.asset_id)}</strong>
                  <small title={job.error || job.provider || job.updated_at}>
                    attempt {job.attempts}/{job.max_attempts || 1}
                    {job.provider ? ` - ${job.provider}` : ""}
                  </small>
                  {(job.status === "queued" || job.status === "running") && (
                    <button type="button" onClick={() => void cancelJob(job.id)} disabled={busy}>
                      Cancel
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
          {selectedAsset && (
            <div className="visual-asset-editor">
              <div className="visual-asset-preview">
                {selectedImageUrl ? (
                  <img src={activeVersion?.url || selectedImageUrl} alt="" />
                ) : (
                  <div>{selectedAsset.status}</div>
                )}
              </div>
              <div className="visual-asset-editor-main">
                <div className="visual-asset-editor-head">
                  <span>{selectedAsset.kind}</span>
                  <strong>{selectedAsset.subject}</strong>
                  <small title={selectedAsset.provider}>
                    {selectedAsset.canon_status} · {selectedAsset.status}
                  </small>
                </div>
                <div className="visual-lineage-note">
                  <strong>{gateLabel(selectedAsset.gate_state)}</strong>
                  <span>{selectedAsset.gate_reason || "No gating detail."}</span>
                  <small>
                    Profile rev {profile.revision}
                    {selectedAsset.form_id ? ` · form ${compactId(selectedAsset.form_id)}` : ""}
                    {selectedAsset.inherited ? " · inherited from ancestor branch" : " · current branch"}
                  </small>
                </div>
                <label>
                  <span>Asset prompt</span>
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
                  <span>Negative prompt</span>
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
                    disabled={busy || versionIndex >= versions.length - 1}
                    onClick={() =>
                      setVersionIndex((value) =>
                        Math.min(versions.length - 1, value + 1),
                      )
                    }
                  >
                    Previous
                  </button>
                  <span>
                    {versionsBusy
                      ? "Loading versions"
                      : versions.length
                        ? `${versionIndex + 1} / ${versions.length}`
                        : "No versions yet"}
                  </span>
                  <button
                    type="button"
                    disabled={busy || versionIndex <= 0}
                    onClick={() =>
                      setVersionIndex((value) => Math.max(0, value - 1))
                    }
                  >
                    Next
                  </button>
                </div>
                {activeVersion && (
                  <div className="model-note">
                    <p>
                      Version from {displayTimestamp(activeVersion.created_at)}{" "}
                      via {activeVersion.provider || "unknown provider"}.
                    </p>
                    <p>
                      {activeVersion.canon_status} · {activeVersion.form_id ? `form ${compactId(activeVersion.form_id)} · ` : ""}
                      {activeVersion.id === selectedAsset.selected_version_id ? "currently selected" : "preview only"}
                    </p>
                    {activeVersion.revised_prompt ? (
                      <p>Revised: {activeVersion.revised_prompt}</p>
                    ) : null}
                  </div>
                )}
                <div className="model-actions">
                  <button
                    type="button"
                    onClick={() => void saveAssetPrompt()}
                    disabled={busy}
                  >
                    Save prompt
                  </button>
                  <button
                    type="button"
                    onClick={() => void selectVersion()}
                    disabled={busy || !activeVersion}
                  >
                    Use shown version
                  </button>
                  <button
                    type="button"
                    onClick={() => void stepSelection("undo")}
                    disabled={busy || !selectedAsset.can_undo_selection}
                  >
                    Undo selection
                  </button>
                  <button
                    type="button"
                    onClick={() => void stepSelection("redo")}
                    disabled={busy || !selectedAsset.can_redo_selection}
                  >
                    Redo selection
                  </button>
                  <button
                    type="button"
                    className="primary-action"
                    onClick={() => void regenerateSelectedAsset()}
                    disabled={busy || !generationAllowed}
                  >
                    {selectedAsset.gate_state === "silhouette_available" ? "Generate silhouette" : "Regenerate"}
                  </button>
                </div>
              </div>
            </div>
          )}
          <div className="model-actions">
            <button
              type="button"
              onClick={() => void onReload()}
              disabled={busy}
            >
              Reload assets
            </button>
            <button
              type="button"
              onClick={() => void cleanup(true)}
              disabled={busy}
            >
              Preview cleanup
            </button>
            <button
              type="button"
              onClick={() => void cleanup(false)}
              disabled={busy}
            >
              Clean files
            </button>
            <button
              type="button"
              onClick={() => void generate({ force: false, limit: 6 })}
              disabled={busy || assets.length === 0}
            >
              Generate missing
            </button>
            <button
              type="button"
              onClick={() => void generate({ force: true, limit: 6 })}
              disabled={busy || assets.length === 0}
            >
              Regenerate visible
            </button>
            <button
              type="button"
              className="primary-action"
              onClick={() => void save()}
              disabled={busy}
            >
              {busy ? "Saving..." : "Save visual prompts"}
            </button>
          </div>
          {error && <p className="model-error">{error}</p>}
          {saveError && <p className="model-error">{saveError}</p>}
          <p className="model-note">
            Missing images become pending asset requests. Ready images are
            served without blocking story turns.
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
  const [draft, setDraft] = useState<ModelRoutingDraft | null>(() =>
    modelSettings ? draftFromModelSettings(modelSettings) : null,
  );
  const [saveError, setSaveError] = useState("");
  const [saveMessage, setSaveMessage] = useState("");

  useEffect(() => {
    setDraft(modelSettings ? draftFromModelSettings(modelSettings) : null);
    setSaveError("");
  }, [modelSettings]);

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
      await onSave(updateFromDraft(modelSettings, draft));
      setSaveMessage("Model routing saved to shared config.");
    } catch (error) {
      setSaveError(error instanceof Error ? error.message : String(error));
    }
  };

  if (!modelSettings || !draft) {
    return (
      <div className="model-routing">
        <div className="model-routing-head">
          <span>Model Routing</span>
          <strong>Config unavailable</strong>
        </div>
        {modelError && <p className="model-error">{modelError}</p>}
        <div className="model-actions">
          <button type="button" onClick={() => void onReload()} disabled={busy}>
            Reload from disk
          </button>
        </div>
      </div>
    );
  }

  const issues = modelRoutingIssues(modelSettings, draft);
  const dirty = hasModelRoutingChanges(modelSettings, draft);
  const revision = modelSettings.config_revision
    ? modelSettings.config_revision.slice(0, 12)
    : "unknown";

  return (
    <div className="model-routing">
      <div className="model-routing-head">
        <span>Model Routing</span>
        <strong>Shared config · {revision}</strong>
      </div>
      <div className="model-active-strip">
        <div>
          <span>Effective provider from saved config</span>
          <strong>{modelSettings.active.provider || "none"}</strong>
        </div>
        <div>
          <span>Configured narrator model</span>
          <strong>
            {modelSettings.active.narrative_model || "provider default"}
          </strong>
        </div>
        <div>
          <span>Config path</span>
          <strong title={modelSettings.config_path}>
            {modelSettings.config_path}
          </strong>
        </div>
      </div>
      <div className="imagegen-status-strip">
        <div
          className={modelSettings.image_generation.available ? "ready" : "blocked"}
        >
          <span>Image generation</span>
          <strong>{modelSettings.image_generation.status}</strong>
        </div>
        <div>
          <span>Provider</span>
          <strong>{modelSettings.image_generation.provider || "not set"}</strong>
        </div>
        <div>
          <span>API key</span>
          <strong>
            {modelSettings.image_generation.api_key_configured
              ? "configured"
              : "not configured"}
          </strong>
        </div>
      </div>
      <p className="model-note">
        Image generation keys stay in the gateway environment or shared service
        config. The browser only shows whether a usable key or bridge is already
        configured.
      </p>
      <div className="settings-grid">
        <label>
          <span>Provider priority</span>
          <CustomSelect
            value={activeProvider}
            ariaLabel="Provider priority"
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
        <label>
          <span>Utility model</span>
          <ModelInput
            value={draft.utilityModel}
            options={modelSettings.utility_models}
            onChange={(value) =>
              updateDraft((draft) => ({ ...draft, utilityModel: value }))
            }
          />
        </label>
        <label>
          <span>Repair model</span>
          <ModelInput
            value={draft.repairModel}
            options={modelSettings.repair_models}
            onChange={(value) =>
              updateDraft((draft) => ({ ...draft, repairModel: value }))
            }
          />
        </label>
        <label>
          <span>Repair fallbacks</span>
          <input
            value={draft.repairFallbackModels}
            onChange={(event) =>
              updateDraft((draft) => ({
                ...draft,
                repairFallbackModels: event.target.value,
              }))
            }
            placeholder="comma-separated fallback models"
          />
        </label>
        <label>
          <span>Image provider</span>
          <input
            value={draft.imageGeneration.provider}
            onChange={(event) =>
              updateImageGeneration({ provider: event.target.value })
            }
            placeholder="openclaw-bridge"
          />
        </label>
        <label>
          <span>Image generation model</span>
          <ModelInput
            value={draft.imageGeneration.model}
            options={modelSettings.image_models}
            onChange={(value) => updateImageGeneration({ model: value })}
          />
        </label>
        <label>
          <span>OpenClaw bridge URL</span>
          <input
            value={draft.imageGeneration.openClawBridgeUrl}
            onChange={(event) =>
              updateImageGeneration({ openClawBridgeUrl: event.target.value })
            }
            placeholder="http://openclaw-imagegen:8099/generate"
          />
        </label>
        <label>
          <span>OpenAI-compatible base URL</span>
          <input
            value={draft.imageGeneration.baseUrl}
            onChange={(event) =>
              updateImageGeneration({ baseUrl: event.target.value })
            }
            placeholder="http://llm.example.com/v1"
          />
        </label>
        <label>
          <span>Default image size</span>
          <input
            value={draft.imageGeneration.defaultSize}
            onChange={(event) =>
              updateImageGeneration({ defaultSize: event.target.value })
            }
            placeholder="1024x1024"
          />
        </label>
        <label>
          <span>Location image size</span>
          <input
            value={draft.imageGeneration.locationSize}
            onChange={(event) =>
              updateImageGeneration({ locationSize: event.target.value })
            }
            placeholder="1536x1024"
          />
        </label>
        <label>
          <span>Character image size</span>
          <input
            value={draft.imageGeneration.characterSize}
            onChange={(event) =>
              updateImageGeneration({ characterSize: event.target.value })
            }
            placeholder="1024x1024"
          />
        </label>
        <label>
          <span>Image output format</span>
          <CustomSelect value={draft.imageGeneration.outputFormat} ariaLabel="Image output format" onChange={(value) => updateImageGeneration({ outputFormat: value })} options={["png", "jpeg", "webp"].map((format) => ({ value: format, label: format.toUpperCase() }))} />
        </label>
        <label>
          <span>Image timeout seconds</span>
          <input
            type="number"
            min={1}
            value={draft.imageGeneration.timeoutSeconds}
            onChange={(event) =>
              updateImageGeneration({
                timeoutSeconds: Number(event.target.value),
              })
            }
          />
        </label>
        <label className="toggle-row">
          <span>Auto-generate visuals</span>
          <input
            type="checkbox"
            checked={draft.imageGeneration.autoGenerate}
            onChange={(event) =>
              updateImageGeneration({ autoGenerate: event.target.checked })
            }
          />
        </label>
        <label className="toggle-row">
          <span>Append negative prompt</span>
          <input
            type="checkbox"
            checked={draft.imageGeneration.appendNegativePrompt}
            onChange={(event) =>
              updateImageGeneration({
                appendNegativePrompt: event.target.checked,
              })
            }
          />
        </label>
        <label>
          <span>ASCII art model</span>
          <ModelInput
            value={draft.asciiModel}
            options={modelSettings.ascii_models}
            onChange={(value) =>
              updateDraft((draft) => ({ ...draft, asciiModel: value }))
            }
          />
        </label>
        <label>
          <span>Embedding provider</span>
          <CustomSelect
            value={draft.embeddingProvider}
            ariaLabel="Embedding provider"
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
          <span>Embedding model</span>
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
                  <span>Model</span>
                  <ModelInput
                    value={providerDraft?.model ?? ""}
                    options={modelSettings.narrative_models}
                    onChange={(value) =>
                      updateProvider(provider.id, { model: value })
                    }
                  />
                </label>
              )}
              {provider.supports_reasoning && (
                <label>
                  <span>Reasoning</span>
                  <CustomSelect
                    value={providerDraft?.reasoning ?? "off"}
                    ariaLabel={`Reasoning for ${provider.label}`}
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
      <div className="model-facts">
        <span>Provider chain: {draft.providerPriority.join(" -> ")}</span>
        <span>
          Images: {draft.imageGeneration.provider || "none"}/
          {draft.imageGeneration.model || "no model"}
        </span>
        <span>
          Embedding: {draft.embeddingProvider}/
          {draft.embeddingModel || "default"}
        </span>
        <span>TTS: {modelSettings.tts_status}</span>
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
          onClick={() => setDraft(draftFromModelSettings(modelSettings))}
          disabled={busy}
        >
          Reset
        </button>
        <button type="button" onClick={() => void onReload()} disabled={busy}>
          Reload from disk
        </button>
        <button
          type="button"
          className="primary-action"
          onClick={() => void save()}
          disabled={busy || !dirty || issues.length > 0}
        >
          {busy ? "Saving..." : "Save model routing"}
        </button>
      </div>
      {modelError && <p className="model-error">{modelError}</p>}
      {saveError && <p className="model-error">{saveError}</p>}
      {saveMessage && <p className="model-success">{saveMessage}</p>}
      <p className="model-note">
        Saved changes write to the shared config used by the terminal and by the
        next browser turn bridge process.
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
          <span>Manual save name</span>
          <input
            value={name}
            onChange={(event) => setName(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") submitSave();
            }}
            placeholder={`Browser Save T${currentTurn}`}
            disabled={busy || !snapshot}
          />
        </label>
        <button
          type="button"
          className="wide-action"
          onClick={submitSave}
          disabled={busy || !snapshot}
        >
          Create Save
        </button>
      </div>
      <label className="save-filter">
        <span>Find save</span>
        <input
          value={saveFilter}
          onChange={(event) => onSaveFilterChange(event.target.value)}
          placeholder="/load filter"
          disabled={busy || !snapshot || saves.length === 0}
        />
      </label>
      <div className="save-list">
        {saves.length === 0 ? (
          <div className="empty-copy">No saved snapshots yet.</div>
        ) : filteredSaves.length === 0 ? (
          <div className="empty-copy">No saves match this filter.</div>
        ) : (
          filteredSaves.map((save) => (
            <div className="save-row" key={save.id}>
              <div>
                <strong>{save.name}</strong>
                <span>
                  Turn {save.turn} - Chapter {save.chapter} -{" "}
                  {compactText(save.location || "Unknown", 32)}
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
                  Load
                </button>
                <button
                  type="button"
                  className="save-delete-button"
                  onClick={() => onDeleteSave(save)}
                  disabled={busy}
                >
                  Delete
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
  if (!metaResult) {
    return (
      <div className="overlay-content">
        <p className="overlay-copy muted">No meta response is available yet.</p>
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
}: {
  snapshot: StorySnapshot | null;
  selectedTab: ModuleTab;
  visuals: VisualCatalog;
  focusCardId?: string | null;
}) {
  if (!snapshot) {
    return (
      <div className="overlay-content">
        <p className="overlay-copy muted">
          Select a story to inspect this module.
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
  const [wizard, setWizard] = useState<StoryWizardResult | null>(null);
  const [input, setInput] = useState("");
  const [start, setStart] = useState(true);
  const [error, setError] = useState("");
  const [enhancing, setEnhancing] = useState(false);
  const [pendingPreset, setPendingPreset] = useState<StoryWizardAction | null>(null);
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
            stage: next.stage_label || next.stage || "Story setup",
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
    setOperation("Opening guided setup");
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
              stage: "AI enhance",
              message: `Improved ${inputLabelForStage(stage).toLowerCase()} with ${response.model || response.provider}.`,
            },
            ...items,
          ].slice(0, 8),
        );
      }
    } catch (failure) {
      setInput(fallback);
      setError(
        `AI enhance failed; used local fallback. ${failure instanceof Error ? failure.message : String(failure)}`,
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
              ? "Complete"
              : phase === "character"
                ? "Character setup"
                : `Story setup - Step ${step.current} of ${step.total}`}
          </span>
          <strong>{wizard?.stage_label || "Loading wizard"}</strong>
        </div>
        <label className="checkbox-line">
          <input
            type="checkbox"
            checked={start}
            onChange={(event) => setStart(event.target.checked)}
            disabled={busy || stage === "done"}
          />
          <span>Start first turn after creation</span>
        </label>
      </div>

      <div className="story-wizard-grid">
        <div className="story-wizard-main">
          {operation && (
            <div className="story-wizard-generation" role="status" aria-live="polite">
              <span className="story-wizard-generation-pulse" aria-hidden="true" />
              <div>
                <strong>{generationProgressLabel(operation, elapsedSeconds)}</strong>
                <small>{elapsedSeconds}s elapsed. The structured draft appears after schema validation.</small>
              </div>
            </div>
          )}
          <div className="story-wizard-message">
            <pre>
              {wizard?.message ||
                "Starting the same guided setup used by the terminal..."}
            </pre>
          </div>

          {wizard?.actions?.length ? (
            <div
              className="story-wizard-actions"
              aria-label="Wizard quick choices"
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
                  <strong>{action.label}</strong>
                </button>
              ))}
            </div>
          ) : null}

          {pendingPreset && (
            <section className="story-wizard-confirmation" aria-label="Confirm story preset">
              <div>
                <span>Review before generation</span>
                <strong>{pendingPreset.label}</strong>
                <p>The preset has only filled the brief. Nothing has been generated or created yet. Edit it below, then confirm when it reads correctly.</p>
              </div>
              <div>
                <button type="button" onClick={() => { setPendingPreset(null); setInput(""); }} disabled={busy}>Cancel</button>
                <button type="button" className="primary" onClick={() => void runStep({ input: input.trim() })} disabled={busy || !input.trim()}>Generate draft</button>
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
                placeholder={wizard?.placeholder || "Type your response..."}
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
              {enhancing ? "Enhancing..." : "Enhance text"}
            </button>
            {!pendingPreset && (
              <button type="submit" className="primary" disabled={busy || stage === "done"}>
                {busy ? "Working..." : submitLabelForStage(stage)}
              </button>
            )}
          </div>
        </div>

        <aside className="story-wizard-side">
          <div className="story-wizard-card">
            <span>Draft Summary</span>
            {definition ? (
              <>
                <strong>{definition.name}</strong>
                <p>{definition.description}</p>
                <dl>
                  <dt>Genre</dt>
                  <dd>{definition.genre}</dd>
                  <dt>Tone</dt>
                  <dd>{definition.tone}</dd>
                  <dt>Language</dt>
                  <dd>{definition.language}</dd>
                  <dt>World</dt>
                  <dd>{definition.worldName}</dd>
                  <dt>Combat</dt>
                  <dd>{definition.hasCombat ? "enabled" : "disabled"}</dd>
                </dl>
              </>
            ) : (
              <p>
                No draft yet. Start with a brief or a terminal-compatible
                preset.
              </p>
            )}
          </div>
          <div className="story-wizard-card">
            <span>Recent Wizard Log</span>
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
              <p>Steps will appear here as you build the story.</p>
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
  if (stage === "brief") return "Generating structured story draft";
  if (payload.action === "create_story") return "Locking reviewed story";
  if (stage.startsWith("review_") || stage === "confirm") return "Revising structured story draft";
  if (stage === "character_background") return "Creating story and protagonist";
  return "Updating guided setup";
}

function generationProgressLabel(operation: string, elapsedSeconds: number): string {
  if (elapsedSeconds < 2) return operation;
  if (elapsedSeconds < 8) return "Generating world, rules, and playable stats";
  return "Validating the structured draft";
}

function currentInputRequiredMessage(wizard: StoryWizardResult | null): string {
  const stage = wizard?.stage ?? "brief";
  if (stage === "character_name") return "Protagonist name is required.";
  if (stage === "character_background")
    return "Write a background or use Skip background.";
  if (stage === "brief") return "Story brief is required.";
  return "Type a revision or use one of the quick choices.";
}

function inputLabelForStage(stage: string): string {
  switch (stage) {
    case "brief":
      return "Story brief";
    case "character_name":
      return "Protagonist name";
    case "character_background":
      return "Background";
    default:
      return "Revision";
  }
}

function submitLabelForStage(stage: string): string {
  switch (stage) {
    case "brief":
      return "Draft world";
    case "character_name":
      return "Set name";
    case "character_background":
      return "Create story";
    default:
      return "Send revision";
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
    name: stringValue(raw.name, "Untitled story"),
    description: stringValue(raw.description, "No description yet."),
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

function gateLabel(value: string): string {
  const labels: Record<string, string> = {
    insufficient_observation: "Observation required",
    silhouette_available: "Silhouette available",
    identified_draft: "Identity draft",
    established_canonical: "Canonical identity",
    form_changed: "New canonical form",
    identity_contradiction: "Identity contradiction",
    insufficient_canon: "Location canon required",
    explicit_request_available: "Available on request",
    narrative_significance: "Narratively significant",
    meaningful_stay: "Meaningful stay",
    chapter_milestone: "Chapter milestone",
  };
  return labels[value] ?? value.replaceAll("_", " ");
}

function compactId(value: string): string {
  if (value.length <= 20) return value;
  return `${value.slice(0, 9)}…${value.slice(-7)}`;
}
