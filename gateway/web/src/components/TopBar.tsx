import type { ReactNode } from "react";
import { BookOpen, CircleHelp, Clock3, Hash, PanelLeftClose, PanelLeftOpen, PanelRightClose, PanelRightOpen, Settings } from "lucide-react";
import { displayClock } from "../format";
import type { OverlayKind, StorySnapshot, SyncState } from "../types";

interface TopBarProps {
  snapshot: StorySnapshot | null;
  sync: SyncState;
  showLeftRail: boolean;
  showInspector: boolean;
  onToggleLeftRail: () => void;
  onToggleInspector: () => void;
  onOpen: (overlay: OverlayKind) => void;
}

export function TopBar({ snapshot, sync, showLeftRail, showInspector, onToggleLeftRail, onToggleInspector, onOpen }: TopBarProps) {
  const clock = displayClock(snapshot);
  return (
    <header className="top-bar">
      <div className="top-status" aria-label="Current story status">
        <StatusCell icon={<Hash size={14} />} label="Turn" value={snapshot ? String(snapshot.world.current_turn) : "-"} />
        <StatusCell icon={<BookOpen size={14} />} label="Story" value={snapshot?.story.name || "Choose a story"} strong />
        <StatusCell icon={<Clock3 size={14} />} label="Story time" value={snapshot ? clock.time : "-"} />
        <div className={`status-cell sync-cell ${sync.toLowerCase()}`}>
          <i aria-hidden="true" />
          <strong title={`Connection: ${sync}`}>{sync}</strong>
        </div>
      </div>
      <div className="top-actions">
        <button
          className="square-button"
          type="button"
          onClick={onToggleLeftRail}
          title={`${showLeftRail ? "Hide" : "Show"} library ([)`}
          aria-label={`${showLeftRail ? "Hide" : "Show"} library`}
        >
          {showLeftRail ? <PanelLeftClose size={16} /> : <PanelLeftOpen size={16} />}
          <span>Library</span>
        </button>
        <button
          className="square-button"
          type="button"
          onClick={onToggleInspector}
          title={`${showInspector ? "Hide" : "Show"} story details (])`}
          aria-label={`${showInspector ? "Hide" : "Show"} story details`}
        >
          {showInspector ? <PanelRightClose size={16} /> : <PanelRightOpen size={16} />}
          <span>Details</span>
        </button>
        <button className="chrome-button" type="button" onClick={() => onOpen("options")}>
          <Settings size={15} />
          Options
        </button>
        <button className="chrome-button" type="button" onClick={() => onOpen("help")}>
          <CircleHelp size={16} />
          <span className="sr-only">Help</span>
        </button>
      </div>
    </header>
  );
}

function StatusCell({
  icon,
  label,
  value,
  strong = false,
}: {
  icon?: ReactNode;
  label: string;
  value: string;
  strong?: boolean;
}) {
  return (
    <div className="status-cell" aria-label={`${label}: ${value}`}>
      {icon && <span className="status-icon" aria-hidden="true">{icon}</span>}
      <span>{label}</span>
      <strong className={strong ? "status-strong" : undefined} title={value}>
        {value}
      </strong>
    </div>
  );
}
