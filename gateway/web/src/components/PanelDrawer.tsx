import { X } from "lucide-react";
import { slashCommands } from "../commands";
import { compactText } from "../format";
import type { OverlayKind, StorySnapshot } from "../types";

interface PanelDrawerProps {
  overlay: OverlayKind;
  snapshot: StorySnapshot | null;
  onClose: () => void;
  onDraft: (value: string) => void;
}

export function PanelDrawer({ overlay, snapshot, onClose, onDraft }: PanelDrawerProps) {
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
        {overlay === "options" && <OptionsContent snapshot={snapshot} />}
        {overlay === "saves" && <SavesContent snapshot={snapshot} onDraft={onDraft} onClose={onClose} />}
        {overlay === "new-story" && <NewStoryContent />}
      </section>
    </div>
  );
}

function overlayTitle(overlay: OverlayKind): string {
  if (overlay === "help") return "Help";
  if (overlay === "options") return "Options";
  if (overlay === "saves") return "Save & Load";
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

function OptionsContent({ snapshot }: { snapshot: StorySnapshot | null }) {
  return (
    <div className="overlay-content">
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
    </div>
  );
}

function SavesContent({
  snapshot,
  onDraft,
  onClose,
}: {
  snapshot: StorySnapshot | null;
  onDraft: (value: string) => void;
  onClose: () => void;
}) {
  const saves = snapshot?.panels.saves ?? [];
  return (
    <div className="overlay-content">
      <button
        type="button"
        className="wide-action"
        onClick={() => {
          onDraft(`/save Quicksave T${snapshot?.world.current_turn ?? ""}`.trim());
          onClose();
        }}
      >
        Prepare /save command
      </button>
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
