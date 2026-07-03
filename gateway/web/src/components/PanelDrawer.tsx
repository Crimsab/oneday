import { useState } from "react";
import { X } from "lucide-react";
import { commandDescriptorsToSlashCommands, commandDescriptors as resolveCommandDescriptors } from "../commands";
import { compactText } from "../format";
import { ModuleContent, moduleTitle } from "./Inspector";
import type { AppPreferences, CommandDescriptor, MetaResult, ModelSettings, ModuleTab, OverlayKind, SaveView, StorySnapshot } from "../types";

interface PanelDrawerProps {
  overlay: OverlayKind;
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  metaResult: MetaResult | null;
  modelSettings: ModelSettings | null;
  selectedTab: ModuleTab;
  commandDescriptors: CommandDescriptor[];
  busy: boolean;
  onClose: () => void;
  onPreferencesChange: (preferences: AppPreferences) => void;
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
  selectedTab,
  commandDescriptors,
  busy,
  onClose,
  onPreferencesChange,
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
            onPreferencesChange={onPreferencesChange}
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
  onPreferencesChange,
}: {
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  modelSettings: ModelSettings | null;
  onPreferencesChange: (preferences: AppPreferences) => void;
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
      <ModelRoutingSettings preferences={preferences} modelSettings={modelSettings} onUpdate={update} />
    </div>
  );
}

function ModelRoutingSettings({
  preferences,
  modelSettings,
  onUpdate,
}: {
  preferences: AppPreferences;
  modelSettings: ModelSettings | null;
  onUpdate: <K extends keyof AppPreferences>(key: K, value: AppPreferences[K]) => void;
}) {
  return (
    <div className="model-routing">
      <div className="model-routing-head">
        <span>Model Routing</span>
        <strong>{modelSettings ? "Staged only" : "Config unavailable"}</strong>
      </div>
      <div className="settings-grid">
        <ModelSelect
          label="Narrator"
          value={preferences.narrativeModel}
          options={modelSettings?.narrative_models ?? []}
          onChange={(value) => onUpdate("narrativeModel", value)}
        />
        <ModelSelect
          label="Meta/utility"
          value={preferences.utilityModel}
          options={modelSettings?.utility_models ?? []}
          onChange={(value) => onUpdate("utilityModel", value)}
        />
        <ModelSelect
          label="Repair"
          value={preferences.repairModel}
          options={modelSettings?.repair_models ?? []}
          onChange={(value) => onUpdate("repairModel", value)}
        />
        <ModelSelect
          label="Images/ascii"
          value={preferences.imageModel}
          options={modelSettings?.image_models ?? []}
          onChange={(value) => onUpdate("imageModel", value)}
        />
        <label>
          <span>OpenAI endpoint</span>
          <input
            value={preferences.openAIEndpoint}
            onChange={(event) => onUpdate("openAIEndpoint", event.target.value)}
            placeholder="future /v1 endpoint override"
          />
        </label>
        <label>
          <span>TTS voice</span>
          <input
            value={preferences.ttsVoice}
            onChange={(event) => onUpdate("ttsVoice", event.target.value)}
            placeholder="future per-character voice"
          />
        </label>
      </div>
      <p className="model-note">
        Browser selections are saved locally. The active terminal-compatible engine still follows config.yaml until a model_routing backend contract is added.
      </p>
      {modelSettings && (
        <div className="model-facts">
          <span>Provider chain: {modelSettings.provider_priority.join(" -> ") || "not configured"}</span>
          <span>Embedding: {modelSettings.embedding_provider}/{modelSettings.embedding_model || "default"}</span>
          <span>TTS: {modelSettings.tts_status}</span>
        </div>
      )}
    </div>
  );
}

function ModelSelect({ label, value, options, onChange }: { label: string; value: string; options: string[]; onChange: (value: string) => void }) {
  return (
    <label>
      <span>{label}</span>
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="config">Config default</option>
        {options.map((option) => (
          <option value={option} key={option}>
            {option}
          </option>
        ))}
      </select>
    </label>
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
