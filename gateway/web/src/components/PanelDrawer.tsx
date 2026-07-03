import { useEffect, useId, useState } from "react";
import { X } from "lucide-react";
import { commandDescriptorsToSlashCommands, commandDescriptors as resolveCommandDescriptors } from "../commands";
import { compactText } from "../format";
import { draftFromModelSettings, promoteProvider, updateFromDraft, type ModelRoutingDraft } from "../modelRouting";
import { ModuleContent, moduleTitle } from "./Inspector";
import type { AppPreferences, CommandDescriptor, MetaResult, ModelSettings, ModelSettingsUpdate, ModuleTab, OverlayKind, SaveView, StorySnapshot } from "../types";

interface PanelDrawerProps {
  overlay: OverlayKind;
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  metaResult: MetaResult | null;
  modelSettings: ModelSettings | null;
  modelBusy: boolean;
  selectedTab: ModuleTab;
  commandDescriptors: CommandDescriptor[];
  busy: boolean;
  onClose: () => void;
  onPreferencesChange: (preferences: AppPreferences) => void;
  onModelSettingsSave: (payload: ModelSettingsUpdate) => Promise<void>;
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
  modelBusy,
  selectedTab,
  commandDescriptors,
  busy,
  onClose,
  onPreferencesChange,
  onModelSettingsSave,
  onCreateSave,
  onLoadSave,
  onDeleteSave,
  saveFilter,
  onSaveFilterChange,
}: PanelDrawerProps) {
  if (!overlay) return null;
  return (
    <div className="overlay-backdrop" role="presentation" onMouseDown={onClose}>
      <section
        className={`overlay-panel ${overlay === "module" ? "module-overlay" : ""}`}
        role="dialog"
        aria-modal="true"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="overlay-head">
          <h2>{overlayTitle(overlay, selectedTab)}</h2>
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
            modelBusy={modelBusy}
            onPreferencesChange={onPreferencesChange}
            onModelSettingsSave={onModelSettingsSave}
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
        {overlay === "new-story" && <NewStoryContent />}
        {overlay === "meta" && <MetaContent metaResult={metaResult} />}
        {overlay === "module" && <ModuleOverlayContent snapshot={snapshot} selectedTab={selectedTab} />}
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
  modelBusy,
  onPreferencesChange,
  onModelSettingsSave,
}: {
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  modelSettings: ModelSettings | null;
  modelBusy: boolean;
  onPreferencesChange: (preferences: AppPreferences) => void;
  onModelSettingsSave: (payload: ModelSettingsUpdate) => Promise<void>;
}) {
  const update = <K extends keyof AppPreferences>(key: K, value: AppPreferences[K]) => {
    onPreferencesChange({ ...preferences, [key]: value });
  };

  return (
    <div className="overlay-content options-content">
      <div className="option-grid">
        <div>
          <span>Realtime bridge</span>
          <strong>{snapshot ? "SSE snapshots active" : "No story selected"}</strong>
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
          <strong>Dark cockpit</strong>
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
      <ModelRoutingSettings modelSettings={modelSettings} busy={modelBusy} onSave={onModelSettingsSave} />
    </div>
  );
}

function ModelRoutingSettings({
  modelSettings,
  busy,
  onSave,
}: {
  modelSettings: ModelSettings | null;
  busy: boolean;
  onSave: (payload: ModelSettingsUpdate) => Promise<void>;
}) {
  const [draft, setDraft] = useState<ModelRoutingDraft | null>(() => (modelSettings ? draftFromModelSettings(modelSettings) : null));
  const [saveError, setSaveError] = useState("");

  useEffect(() => {
    setDraft(modelSettings ? draftFromModelSettings(modelSettings) : null);
    setSaveError("");
  }, [modelSettings]);

  const providerIds = modelSettings?.providers.map((provider) => provider.id) ?? [];
  const activeProvider = draft?.providerPriority[0] ?? modelSettings?.active.provider ?? "";

  const updateDraft = (updater: (value: ModelRoutingDraft) => ModelRoutingDraft) => {
    setSaveError("");
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
    try {
      await onSave(updateFromDraft(modelSettings, draft));
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
      </div>
    );
  }

  return (
    <div className="model-routing">
      <div className="model-routing-head">
        <span>Model Routing</span>
        <strong>Shared config</strong>
      </div>
      <div className="model-active-strip">
        <div>
          <span>Active provider</span>
          <strong>{modelSettings.active.provider || "none"}</strong>
        </div>
        <div>
          <span>Narrator model</span>
          <strong>{modelSettings.active.narrative_model || "provider default"}</strong>
        </div>
        <div>
          <span>Config path</span>
          <strong title={modelSettings.config_path}>{modelSettings.config_path}</strong>
        </div>
      </div>
      <div className="settings-grid">
        <label>
          <span>Primary provider</span>
          <select
            value={activeProvider}
            onChange={(event) =>
              updateDraft((value) => ({
                ...value,
                providerPriority: promoteProvider(value.providerPriority, providerIds, event.target.value),
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
          <span>Images/ascii model</span>
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
      <div className="model-actions">
        <button type="button" onClick={() => setDraft(draftFromModelSettings(modelSettings))} disabled={busy}>
          Reset
        </button>
        <button type="button" className="primary-action" onClick={() => void save()} disabled={busy}>
          {busy ? "Saving..." : "Save model routing"}
        </button>
      </div>
      {saveError && <p className="model-error">{saveError}</p>}
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
                <small>{save.created_at}</small>
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

function ModuleOverlayContent({ snapshot, selectedTab }: { snapshot: StorySnapshot | null; selectedTab: ModuleTab }) {
  if (!snapshot) {
    return (
      <div className="overlay-content">
        <p className="overlay-copy muted">Select a story to inspect this module.</p>
      </div>
    );
  }
  return (
    <div className="overlay-content module-content">
      <ModuleContent tab={selectedTab} snapshot={snapshot} expanded />
    </div>
  );
}

function NewStoryContent() {
  return (
    <div className="overlay-content">
      <p className="overlay-copy">
        Story creation still lives in the terminal flow. The browser gateway currently reads existing stories and submits realtime turns against the same canonical session.
      </p>
      <p className="overlay-copy muted">
        Keeping this disabled avoids creating a second incompatible story path before the backend exposes the same creation contract.
      </p>
    </div>
  );
}
