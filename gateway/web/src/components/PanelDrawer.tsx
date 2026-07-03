import { X } from "lucide-react";
import { slashCommands } from "../commands";
import { compactText } from "../format";
import type { AppPreferences, OverlayKind, StorySnapshot } from "../types";

interface PanelDrawerProps {
  overlay: OverlayKind;
  snapshot: StorySnapshot | null;
  preferences: AppPreferences;
  onClose: () => void;
  onPreferencesChange: (preferences: AppPreferences) => void;
}

export function PanelDrawer({ overlay, snapshot, preferences, onClose, onPreferencesChange }: PanelDrawerProps) {
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
        {overlay === "help" && <HelpContent />}
        {overlay === "options" && <OptionsContent snapshot={snapshot} preferences={preferences} onPreferencesChange={onPreferencesChange} />}
        {overlay === "saves" && <SavesContent snapshot={snapshot} />}
        {overlay === "new-story" && <NewStoryContent />}
      </section>
    </div>
  );
}

function overlayTitle(overlay: OverlayKind): string {
  if (overlay === "help") return "Help";
  if (overlay === "options") return "Options";
  if (overlay === "saves") return "Saves";
  return "New Story";
}

function HelpContent() {
  return (
    <div className="overlay-content command-help">
      {slashCommands.map((command) => (
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

function SavesContent({ snapshot }: { snapshot: StorySnapshot | null }) {
  const saves = snapshot?.panels.saves ?? [];
  return (
    <div className="overlay-content">
      <div className="inline-warning">
        Browser save writes are not exposed yet. Existing saves are read-only here; create new saves from the terminal for now.
      </div>
      <div className="save-list">
        {saves.length === 0 ? (
          <div className="empty-copy">No saved snapshots yet.</div>
        ) : (
          saves.map((save) => (
            <div className="save-row" key={save.id}>
              <strong>{save.name}</strong>
              <span>
                Turn {save.turn} - Chapter {save.chapter} - {compactText(save.location || "Unknown", 28)}
              </span>
              <small>{save.created_at}</small>
            </div>
          ))
        )}
      </div>
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
