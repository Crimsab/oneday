import { useState } from "react";
import { X } from "lucide-react";
import { commandDescriptorsToSlashCommands, commandDescriptors as resolveCommandDescriptors } from "../commands";
import { compactText } from "../format";
import type { AppPreferences, CommandDescriptor, MetaResult, OverlayKind, SaveView, StorySnapshot } from "../types";

interface PanelDrawerProps {
  overlay: OverlayKind;
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  metaResult: MetaResult | null;
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
      <section className="overlay-panel" role="dialog" aria-modal="true" onMouseDown={(event) => event.stopPropagation()}>
        <div className="overlay-head">
          <h2>{overlayTitle(overlay)}</h2>
          <button type="button" className="square-button" onClick={onClose} title="Close">
            <X size={16} />
          </button>
        </div>
        {overlay === "help" && <HelpContent commandDescriptors={commandDescriptors} />}
        {overlay === "options" && <OptionsContent snapshot={snapshot} preferences={preferences} onPreferencesChange={onPreferencesChange} />}
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
      </section>
    </div>
  );
}

function overlayTitle(overlay: OverlayKind): string {
  if (overlay === "help") return "Help";
  if (overlay === "options") return "Options";
  if (overlay === "saves") return "Saves";
  if (overlay === "meta") return "Meta Command";
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
  onPreferencesChange,
}: {
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
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
    </div>
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
