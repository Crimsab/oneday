import { HelpCircle, Save, Settings } from "lucide-react";
import { deriveCondition, displayClock, weatherLabel } from "../format";
import type { OverlayKind, StorySnapshot, SyncState } from "../types";

interface TopBarProps {
  snapshot: StorySnapshot | null;
  sync: SyncState;
  onOpen: (overlay: OverlayKind) => void;
}

export function TopBar({ snapshot, sync, onOpen }: TopBarProps) {
  const clock = displayClock(snapshot?.world.current_turn ?? 0);
  const condition = deriveCondition(snapshot);
  const weather = weatherLabel(snapshot);

  return (
    <header className="top-bar">
      <div className="brand-mark">OneDay</div>
      <div className="top-status" aria-label="Current story status">
        <StatusCell label="Turn" value={snapshot ? String(snapshot.world.current_turn) : "-"} />
        <StatusCell label="Location" value={snapshot?.world.current_location || "Select a story"} strong />
        <StatusCell label="Time" value={snapshot ? clock.time : "-"} />
        <StatusCell label="Weather" value={weather} />
        <StatusCell label="Condition" value={condition} strong />
        <div className={`status-cell sync-cell ${sync.toLowerCase()}`}>
          <span>Sync</span>
          <strong>
            {sync}
            <i aria-hidden="true" />
          </strong>
        </div>
      </div>
      <div className="top-actions">
        <button className="chrome-button" type="button" onClick={() => onOpen("saves")}>
          <Save size={15} />
          Saves
        </button>
        <button className="chrome-button" type="button" onClick={() => onOpen("options")}>
          <Settings size={15} />
          Options
        </button>
        <button className="chrome-button" type="button" onClick={() => onOpen("help")}>
          <HelpCircle size={16} />
          Help
        </button>
      </div>
    </header>
  );
}

function StatusCell({ label, value, strong = false }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className="status-cell">
      <span>{label}</span>
      <strong className={strong ? "status-strong" : undefined}>{value}</strong>
    </div>
  );
}
