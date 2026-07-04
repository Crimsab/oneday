import { useEffect, useId, useState, type FormEvent } from "react";
import { X } from "lucide-react";
import { commandDescriptorsToSlashCommands, commandDescriptors as resolveCommandDescriptors } from "../commands";
import { compactText, displayTimestamp } from "../format";
import { draftFromModelSettings, hasModelRoutingChanges, modelRoutingIssues, promoteProvider, updateFromDraft, type ModelRoutingDraft } from "../modelRouting";
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
  StoryCreateEnvelope,
  StorySnapshot,
  GenerateVisualAssetsRequest,
  VisualAsset,
  VisualProfile,
  VisualProfileUpdate,
} from "../types";
import type { VisualCatalog } from "../visualAssets";

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
  visuals: VisualCatalog;
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
  onVisualAssetsGenerate: (payload: GenerateVisualAssetsRequest) => Promise<void>;
  onVisualAssetsReload: () => Promise<void> | void;
  onCreateStory: (payload: StoryCreateEnvelope) => Promise<void> | void;
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
  visuals,
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
  onCreateStory,
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
        className={`overlay-panel ${overlay === "module" ? "module-overlay" : ""}`}
        role="dialog"
        aria-modal="true"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="overlay-head">
          <h2>{overlayTitle(overlay, activeModuleTab)}</h2>
          <button type="button" className="square-button" onClick={onClose} title="Close">
            <X size={16} />
          </button>
        </div>
        {overlay === "help" && <HelpContent commandDescriptors={commandDescriptors} />}
        {overlay === "options" && (
          <OptionsContent
            snapshot={snapshot}
            preferences={preferences}
            modelSettings={modelSettings}
            modelError={modelError}
            modelBusy={modelBusy}
            visualProfile={visualProfile}
            visualAssets={visualAssets}
            visualProfileError={visualProfileError}
            visualProfileBusy={visualProfileBusy}
            onPreferencesChange={onPreferencesChange}
            onModelSettingsSave={onModelSettingsSave}
            onModelSettingsReload={onModelSettingsReload}
            onVisualProfileSave={onVisualProfileSave}
            onVisualAssetsGenerate={onVisualAssetsGenerate}
            onVisualAssetsReload={onVisualAssetsReload}
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
        {overlay === "new-story" && <NewStoryContent busy={busy} onCreateStory={onCreateStory} />}
        {overlay === "meta" && <MetaContent metaResult={metaResult} />}
        {overlay === "module" && <ModuleOverlayContent snapshot={snapshot} selectedTab={activeModuleTab} visuals={visuals} focusCardId={moduleFocusId} />}
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

function HelpContent({ commandDescriptors }: { commandDescriptors: CommandDescriptor[] }) {
  const commands = commandDescriptorsToSlashCommands(resolveCommandDescriptors(commandDescriptors));
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
  visualProfileError,
  visualProfileBusy,
  onPreferencesChange,
  onModelSettingsSave,
  onModelSettingsReload,
  onVisualProfileSave,
  onVisualAssetsGenerate,
  onVisualAssetsReload,
}: {
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  modelSettings: ModelSettings | null;
  modelError: string;
  modelBusy: boolean;
  visualProfile: VisualProfile | null;
  visualAssets: VisualAsset[];
  visualProfileError: string;
  visualProfileBusy: boolean;
  onPreferencesChange: (preferences: AppPreferences) => void;
  onModelSettingsSave: (payload: ModelSettingsUpdate) => Promise<void>;
  onModelSettingsReload: () => Promise<void> | void;
  onVisualProfileSave: (payload: VisualProfileUpdate) => Promise<void>;
  onVisualAssetsGenerate: (payload: GenerateVisualAssetsRequest) => Promise<void>;
  onVisualAssetsReload: () => Promise<void> | void;
}) {
  const update = <K extends keyof AppPreferences>(key: K, value: AppPreferences[K]) => {
    onPreferencesChange({ ...preferences, [key]: value });
  };

  return (
    <div className="overlay-content options-content">
      <div className="option-grid">
        <div>
          <span>Realtime bridge</span>
          <strong>{snapshot ? "SSE snapshots + turn events" : "No story selected"}</strong>
        </div>
        <div>
          <span>Action transport</span>
          <strong>gateway-turn</strong>
        </div>
        <div>
          <span>Capabilities</span>
          <strong>images, ascii, roll log</strong>
        </div>
        <div>
          <span>Theme</span>
          <strong>Reference Amber Noir</strong>
        </div>
      </div>
      <div className="settings-grid">
        <label>
          <span>Density</span>
          <select value={preferences.density} onChange={(event) => update("density", event.target.value as AppPreferences["density"])}>
            <option value="compact">Compact</option>
            <option value="balanced">Balanced</option>
            <option value="comfortable">Comfortable</option>
          </select>
        </label>
        <label>
          <span>Font size</span>
          <select value={preferences.fontSize} onChange={(event) => update("fontSize", event.target.value as AppPreferences["fontSize"])}>
            <option value="small">Small</option>
            <option value="base">Base</option>
            <option value="large">Large</option>
          </select>
        </label>
        <label>
          <span>Accent</span>
          <select value={preferences.accent} onChange={(event) => update("accent", event.target.value as AppPreferences["accent"])}>
            <option value="amber">Amber</option>
            <option value="green">Green</option>
            <option value="blue">Blue</option>
            <option value="rose">Rose</option>
          </select>
        </label>
        <label className="toggle-row">
          <span>Stories sidebar</span>
          <input type="checkbox" checked={preferences.showLeftRail} onChange={(event) => update("showLeftRail", event.target.checked)} />
        </label>
        <label className="toggle-row">
          <span>Inspector panel</span>
          <input type="checkbox" checked={preferences.showInspector} onChange={(event) => update("showInspector", event.target.checked)} />
        </label>
        <label className="toggle-row">
          <span>Transcript wrap</span>
          <input type="checkbox" checked={preferences.wrapTranscript} onChange={(event) => update("wrapTranscript", event.target.checked)} />
        </label>
      </div>
      <ModelRoutingSettings modelSettings={modelSettings} modelError={modelError} busy={modelBusy} onSave={onModelSettingsSave} onReload={onModelSettingsReload} />
      <VisualDirectionSettings
        profile={visualProfile}
        assets={visualAssets}
        error={visualProfileError}
        busy={visualProfileBusy}
        onSave={onVisualProfileSave}
        onGenerate={onVisualAssetsGenerate}
        onReload={onVisualAssetsReload}
      />
    </div>
  );
}

function VisualDirectionSettings({
  profile,
  assets,
  error,
  busy,
  onSave,
  onGenerate,
  onReload,
}: {
  profile: VisualProfile | null;
  assets: VisualAsset[];
  error: string;
  busy: boolean;
  onSave: (payload: VisualProfileUpdate) => Promise<void>;
  onGenerate: (payload: GenerateVisualAssetsRequest) => Promise<void>;
  onReload: () => Promise<void> | void;
}) {
  const [draft, setDraft] = useState<VisualProfileUpdate>(() => profileDraft(profile));
  const [saveError, setSaveError] = useState("");
  const readyCount = assets.filter((asset) => asset.status === "ready").length;
  const pendingCount = assets.filter((asset) => asset.status !== "ready").length;

  useEffect(() => {
    setDraft(profileDraft(profile));
    setSaveError("");
  }, [profile]);

  const update = <K extends keyof VisualProfileUpdate>(key: K, value: VisualProfileUpdate[K]) => {
    setSaveError("");
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const save = async () => {
    setSaveError("");
    try {
      await onSave(draft);
    } catch (saveFailure) {
      setSaveError(saveFailure instanceof Error ? saveFailure.message : String(saveFailure));
    }
  };

  const generate = async (payload: GenerateVisualAssetsRequest) => {
    setSaveError("");
    try {
      await onGenerate(payload);
    } catch (failure) {
      setSaveError(failure instanceof Error ? failure.message : String(failure));
    }
  };

  return (
    <div className="visual-direction">
      <div className="model-routing-head">
        <span>Visual Direction</span>
        <strong>
          {readyCount} ready / {pendingCount} pending
        </strong>
      </div>
      {!profile ? (
        <p className="model-error">{error || "Select a story to edit visual prompts."}</p>
      ) : (
        <>
          <div className="settings-grid visual-settings">
            <label>
              <span>World/location prompt</span>
              <textarea value={draft.world_style_prompt} onChange={(event) => update("world_style_prompt", event.target.value)} rows={4} />
            </label>
            <label>
              <span>Character prompt</span>
              <textarea value={draft.character_style_prompt} onChange={(event) => update("character_style_prompt", event.target.value)} rows={4} />
            </label>
            <label>
              <span>Palette</span>
              <input value={draft.palette} onChange={(event) => update("palette", event.target.value)} />
            </label>
            <label>
              <span>Negative prompt</span>
              <input value={draft.negative_prompt} onChange={(event) => update("negative_prompt", event.target.value)} />
            </label>
          </div>
          <div className="visual-asset-list">
            {assets.slice(0, 8).map((asset) => (
              <div className={`visual-asset-row ${asset.status}`} key={asset.id} title={asset.prompt}>
                <span>{asset.kind}</span>
                <strong>{asset.subject}</strong>
                <small title={asset.error || asset.provider}>
                  {asset.status}
                  {asset.error ? " !" : ""}
                </small>
                <button
                  type="button"
                  onClick={() => void generate({ asset_ids: [asset.id], force: true, limit: 1 })}
                  disabled={busy}
                  title={asset.error || `Regenerate ${asset.subject}`}
                >
                  regen
                </button>
              </div>
            ))}
          </div>
          <div className="model-actions">
            <button type="button" onClick={() => void onReload()} disabled={busy}>
              Reload assets
            </button>
            <button type="button" onClick={() => void generate({ force: false, limit: 6 })} disabled={busy || assets.length === 0}>
              Generate missing
            </button>
            <button type="button" onClick={() => void generate({ force: true, limit: 6 })} disabled={busy || assets.length === 0}>
              Regenerate visible
            </button>
            <button type="button" className="primary-action" onClick={() => void save()} disabled={busy}>
              {busy ? "Saving..." : "Save visual prompts"}
            </button>
          </div>
          {error && <p className="model-error">{error}</p>}
          {saveError && <p className="model-error">{saveError}</p>}
          <p className="model-note">Missing images become pending asset requests. Ready images are served without blocking story turns.</p>
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
  const [draft, setDraft] = useState<ModelRoutingDraft | null>(() => (modelSettings ? draftFromModelSettings(modelSettings) : null));
  const [saveError, setSaveError] = useState("");
  const [saveMessage, setSaveMessage] = useState("");

  useEffect(() => {
    setDraft(modelSettings ? draftFromModelSettings(modelSettings) : null);
    setSaveError("");
  }, [modelSettings]);

  const providerIds = modelSettings?.providers.map((provider) => provider.id) ?? [];
  const activeProvider = draft?.providerPriority[0] ?? modelSettings?.active.provider ?? "";

  const updateDraft = (updater: (value: ModelRoutingDraft) => ModelRoutingDraft) => {
    setSaveError("");
    setSaveMessage("");
    setDraft((value) => (value ? updater(value) : value));
  };

  const updateProvider = (id: string, patch: Partial<ModelRoutingDraft["providers"][string]>) => {
    updateDraft((value) => ({
      ...value,
      providers: {
        ...value.providers,
        [id]: { ...value.providers[id], ...patch },
      },
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
  const revision = modelSettings.config_revision ? modelSettings.config_revision.slice(0, 12) : "unknown";

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
          <strong>{modelSettings.active.narrative_model || "provider default"}</strong>
        </div>
        <div>
          <span>Config path</span>
          <strong title={modelSettings.config_path}>{modelSettings.config_path}</strong>
        </div>
      </div>
      <div className="settings-grid">
        <label>
          <span>Provider priority</span>
          <select
            value={activeProvider}
            onChange={(event) =>
              updateDraft((value) => ({
                ...value,
                providerPriority: promoteProvider(value.providerPriority, providerIds, event.target.value),
                providers: {
                  ...value.providers,
                  [event.target.value]: {
                    ...value.providers[event.target.value],
                    enabled: true,
                  },
                },
              }))
            }
          >
            {modelSettings.providers.map((provider) => (
              <option key={provider.id} value={provider.id}>
                {provider.label}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>Utility model</span>
          <ModelInput value={draft.utilityModel} options={modelSettings.utility_models} onChange={(value) => updateDraft((draft) => ({ ...draft, utilityModel: value }))} />
        </label>
        <label>
          <span>Repair model</span>
          <ModelInput value={draft.repairModel} options={modelSettings.repair_models} onChange={(value) => updateDraft((draft) => ({ ...draft, repairModel: value }))} />
        </label>
        <label>
          <span>Repair fallbacks</span>
          <input
            value={draft.repairFallbackModels}
            onChange={(event) => updateDraft((draft) => ({ ...draft, repairFallbackModels: event.target.value }))}
            placeholder="comma-separated fallback models"
          />
        </label>
        <label>
          <span>ASCII art model</span>
          <ModelInput value={draft.imageModel} options={modelSettings.image_models} onChange={(value) => updateDraft((draft) => ({ ...draft, imageModel: value }))} />
        </label>
        <label>
          <span>Embedding provider</span>
          <select value={draft.embeddingProvider} onChange={(event) => updateDraft((draft) => ({ ...draft, embeddingProvider: event.target.value }))}>
            {modelSettings.embedding_providers.map((provider) => (
              <option key={provider} value={provider}>
                {provider}
              </option>
            ))}
          </select>
        </label>
        <label>
          <span>Embedding model</span>
          <input value={draft.embeddingModel} onChange={(event) => updateDraft((draft) => ({ ...draft, embeddingModel: event.target.value }))} />
        </label>
      </div>
      <div className="provider-editor-grid">
        {modelSettings.providers.map((provider) => {
          const providerDraft = draft.providers[provider.id];
          return (
            <div className={`provider-card ${providerDraft?.enabled ? "enabled" : ""}`} key={provider.id}>
              <label className="toggle-row">
                <span>{provider.label}</span>
                <input
                  type="checkbox"
                  checked={providerDraft?.enabled ?? provider.enabled}
                  onChange={(event) => updateProvider(provider.id, { enabled: event.target.checked })}
                />
              </label>
              {provider.supports_model && (
                <label>
                  <span>Model</span>
                  <ModelInput
                    value={providerDraft?.model ?? ""}
                    options={modelSettings.narrative_models}
                    onChange={(value) => updateProvider(provider.id, { model: value })}
                  />
                </label>
              )}
              {provider.supports_reasoning && (
                <label>
                  <span>Reasoning</span>
                  <select value={providerDraft?.reasoning ?? "off"} onChange={(event) => updateProvider(provider.id, { reasoning: event.target.value })}>
                    {["off", "none", "minimal", "low", "medium", "high", "xhigh"].map((level) => (
                      <option key={level} value={level}>
                        {level}
                      </option>
                    ))}
                  </select>
                </label>
              )}
            </div>
          );
        })}
      </div>
      <div className="model-facts">
        <span>Provider chain: {draft.providerPriority.join(" -> ")}</span>
        <span>Embedding: {draft.embeddingProvider}/{draft.embeddingModel || "default"}</span>
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
        <button type="button" onClick={() => setDraft(draftFromModelSettings(modelSettings))} disabled={busy}>
          Reset
        </button>
        <button type="button" onClick={() => void onReload()} disabled={busy}>
          Reload from disk
        </button>
        <button type="button" className="primary-action" onClick={() => void save()} disabled={busy || !dirty || issues.length > 0}>
          {busy ? "Saving..." : "Save model routing"}
        </button>
      </div>
      {modelError && <p className="model-error">{modelError}</p>}
      {saveError && <p className="model-error">{saveError}</p>}
      {saveMessage && <p className="model-success">{saveMessage}</p>}
      <p className="model-note">Saved changes write to the shared config used by the terminal and by the next browser turn bridge process.</p>
    </div>
  );
}

function ModelInput({ value, options, onChange }: { value: string; options: string[]; onChange: (value: string) => void }) {
  const listId = useId();
  return (
    <>
      <input value={value} onChange={(event) => onChange(event.target.value)} list={listId} />
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
    ? saves.filter((save) => `${save.name} ${save.location} ${save.turn} ${save.chapter} ${save.id}`.toLowerCase().includes(query))
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
        <button type="button" className="wide-action" onClick={submitSave} disabled={busy || !snapshot}>
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
                  Turn {save.turn} - Chapter {save.chapter} - {compactText(save.location || "Unknown", 32)}
                </span>
                <small>{displayTimestamp(save.created_at)}</small>
              </div>
              <div className="save-actions">
                <button type="button" className="save-load-button" onClick={() => onLoadSave(save)} disabled={busy}>
                  Load
                </button>
                <button type="button" className="save-delete-button" onClick={() => onDeleteSave(save)} disabled={busy}>
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
        <p className="overlay-copy muted">Select a story to inspect this module.</p>
      </div>
    );
  }
  return (
    <div className="overlay-content module-content">
      <ModuleContent tab={selectedTab} snapshot={snapshot} visuals={visuals} expanded focusCardId={focusCardId} />
    </div>
  );
}

function NewStoryContent({
  busy,
  onCreateStory,
}: {
  busy: boolean;
  onCreateStory: (payload: StoryCreateEnvelope) => Promise<void> | void;
}) {
  const [brief, setBrief] = useState(
    "Italian mystery adventure, compact prose, practical choices, strong anti-loop rules, no lore sprawl.",
  );
  const [characterName, setCharacterName] = useState("Tester");
  const [characterBackground, setCharacterBackground] = useState("Created from the browser to validate OneDay parity and UI flows.");
  const [worldStylePrompt, setWorldStylePrompt] = useState("");
  const [characterStylePrompt, setCharacterStylePrompt] = useState("");
  const [palette, setPalette] = useState("");
  const [negativePrompt, setNegativePrompt] = useState("no text, no logos, no watermark, no UI, no unreadable signage");
  const [start, setStart] = useState(true);
  const [error, setError] = useState("");

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!brief.trim()) {
      setError("Story brief is required.");
      return;
    }
    if (!characterName.trim()) {
      setError("Character name is required.");
      return;
    }
    setError("");
    await onCreateStory({
      brief: brief.trim(),
      character_name: characterName.trim(),
      character_background: characterBackground.trim(),
      world_style_prompt: worldStylePrompt.trim(),
      character_style_prompt: characterStylePrompt.trim(),
      negative_prompt: negativePrompt.trim(),
      palette: palette.trim(),
      start,
    });
  };

  const enhanceVisuals = () => {
    const enhanced = enhancedVisualDirection({
      brief,
      characterBackground,
      worldStylePrompt,
      characterStylePrompt,
      palette,
      negativePrompt,
    });
    setWorldStylePrompt(enhanced.world_style_prompt);
    setCharacterStylePrompt(enhanced.character_style_prompt);
    setPalette(enhanced.palette);
    setNegativePrompt(enhanced.negative_prompt);
  };

  return (
    <form className="overlay-content new-story-form" onSubmit={submit}>
      <label>
        <span>Story brief</span>
        <textarea value={brief} onChange={(event) => setBrief(event.target.value)} rows={5} disabled={busy} />
      </label>
      <div className="two-column-form">
        <label>
          <span>Protagonist</span>
          <input value={characterName} onChange={(event) => setCharacterName(event.target.value)} disabled={busy} />
        </label>
        <label className="checkbox-line">
          <input type="checkbox" checked={start} onChange={(event) => setStart(event.target.checked)} disabled={busy} />
          <span>Start first turn</span>
        </label>
      </div>
      <label>
        <span>Background</span>
        <textarea value={characterBackground} onChange={(event) => setCharacterBackground(event.target.value)} rows={3} disabled={busy} />
      </label>
      <div className="visual-create-block">
        <div className="model-routing-head">
          <span>Image Style</span>
          <button type="button" onClick={enhanceVisuals} disabled={busy}>
            Enhance style
          </button>
        </div>
        <label>
          <span>World and location style</span>
          <textarea
            value={worldStylePrompt}
            onChange={(event) => setWorldStylePrompt(event.target.value)}
            rows={4}
            placeholder="Optional. Leave empty to auto-derive from the story."
            disabled={busy}
          />
        </label>
        <label>
          <span>Character style</span>
          <textarea
            value={characterStylePrompt}
            onChange={(event) => setCharacterStylePrompt(event.target.value)}
            rows={3}
            placeholder="Optional. Used for NPC and protagonist portraits."
            disabled={busy}
          />
        </label>
        <div className="two-column-form">
          <label>
            <span>Palette</span>
            <input value={palette} onChange={(event) => setPalette(event.target.value)} placeholder="Optional palette" disabled={busy} />
          </label>
          <label>
            <span>Negative prompt</span>
            <input value={negativePrompt} onChange={(event) => setNegativePrompt(event.target.value)} disabled={busy} />
          </label>
        </div>
      </div>
      {error && <p className="form-error">{error}</p>}
      <div className="drawer-actions">
        <button type="submit" className="primary" disabled={busy}>
          {busy ? "Creating..." : "Create Story"}
        </button>
      </div>
    </form>
  );
}

function enhancedVisualDirection({
  brief,
  characterBackground,
  worldStylePrompt,
  characterStylePrompt,
  palette,
  negativePrompt,
}: {
  brief: string;
  characterBackground: string;
  worldStylePrompt: string;
  characterStylePrompt: string;
  palette: string;
  negativePrompt: string;
}): VisualProfileUpdate {
  const context = [brief, characterBackground].map((part) => part.trim()).filter(Boolean).join(" ");
  const preset = visualPresetFor(context || worldStylePrompt || characterStylePrompt);
  const worldBase = worldStylePrompt.trim() || preset.world;
  const characterBase = characterStylePrompt.trim() || preset.character;
  const contextLine = context
    ? ` Story context: ${compactText(context, 260)}.`
    : " Use a flexible original setting direction suitable for sci-fi, cyberpunk, steampunk, fantasy, or mystery stories.";

  return {
    world_style_prompt: `${worldBase}${contextLine} Composition: cinematic game key art, strong readable silhouettes, concrete places, no generic stock mood.`,
    character_style_prompt: `${characterBase}${contextLine} Composition: square portrait, expressive face, outfit and props derived from role and world, coherent with location lighting.`,
    palette: palette.trim() || preset.palette,
    negative_prompt:
      negativePrompt.trim() ||
      "no text, no logos, no watermark, no UI, no unreadable signage, no extra limbs, no generic stock photo, no anime unless requested",
  };
}

function visualPresetFor(source: string): Pick<VisualProfileUpdate, "world_style_prompt" | "character_style_prompt" | "palette"> & {
  world: string;
  character: string;
} {
  const text = source.toLowerCase();
  if (/(cyber|neon|corporate|hacker|noir)/.test(text)) {
    return {
      world: "Cyberpunk noir concept art with lived-in streets, reflective rain, practical industrial detail, and restrained neon.",
      character: "Grounded cyberpunk character portraits with specific faces, worn clothing, practical tech, and cinematic rim light.",
      world_style_prompt: "",
      character_style_prompt: "",
      palette: "oil black, rain blue, sodium amber, muted teal, restrained neon rose",
    };
  }
  if (/(steam|brass|victorian|clockwork|airship)/.test(text)) {
    return {
      world: "Steampunk adventure concept art with brass machinery, soot, hand-built interiors, and believable period texture.",
      character: "Steampunk character portraits with tailored silhouettes, tools, goggles or period props only when context supports them.",
      world_style_prompt: "",
      character_style_prompt: "",
      palette: "ink black, aged brass, oxidized green, coal gray, warm lamp light",
    };
  }
  if (/(fantasy|magia|magic|elf|ruin|dragon|dungeon)/.test(text)) {
    return {
      world: "Dark fantasy concept art with ancient materials, tactile ruins, weathered magic, and grounded natural light.",
      character: "Painterly fantasy portraits with believable anatomy, expressive eyes, specific costume details, and non-generic silhouettes.",
      world_style_prompt: "",
      character_style_prompt: "",
      palette: "deep forest, bone ivory, tarnished gold, storm gray, ember amber",
    };
  }
  if (/(horror|dread|ghost|occult|paura|orrore)/.test(text)) {
    return {
      world: "Atmospheric horror mystery concept art with ordinary places made tense, soft practical light, and unsettling negative space.",
      character: "Horror mystery portraits with tired expressions, realistic skin, subtle dread, and no exaggerated monster styling unless requested.",
      world_style_prompt: "",
      character_style_prompt: "",
      palette: "cold gray, old paper, sickly green, candle amber, dried red",
    };
  }
  return {
    world: "Cinematic sci-fi adventure concept art with functional architecture, tactile materials, readable geography, and human-scale drama.",
    character: "Grounded sci-fi character portraits with practical outfits, memorable faces, role-specific props, and coherent world lighting.",
    world_style_prompt: "",
    character_style_prompt: "",
    palette: "graphite, soft white, signal amber, desaturated teal, muted blue",
  };
}
