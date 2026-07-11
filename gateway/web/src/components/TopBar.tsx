import type { ReactNode } from "react";
import { Activity, Cloud, Gauge, HelpCircle, MapPin, PanelLeftClose, PanelLeftOpen, PanelRightClose, PanelRightOpen, Save, Settings, Sun } from "lucide-react";
import { deriveCondition, displayClock, weatherLabel } from "../format";
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
  const condition = deriveCondition(snapshot);
  const weather = weatherLabel(snapshot);

  return (
    <header className="top-bar">
      <div className="top-status" aria-label="Current story status">
        <StatusCell icon={<Gauge size={14} />} label="Turn" value={snapshot ? String(snapshot.world.current_turn) : "-"} />
        <StatusCell icon={<MapPin size={14} />} label="Loc" fullLabel="Location" value={snapshot?.world.current_location || "Select a story"} strong />
        <StatusCell icon={<Sun size={14} />} label="Time" value={snapshot ? clock.time : "-"} />
        <StatusCell icon={<Cloud size={14} />} label="Sky" fullLabel="Weather" value={weather} />
        <StatusCell icon={<Activity size={14} />} label="State" fullLabel="Condition" value={condition} strong />
        <div className={`status-cell sync-cell ${sync.toLowerCase()}`}>
          <span>Sync</span>
          <strong title={sync}>
            {sync}
            <i aria-hidden="true" />
          </strong>
        </div>
      </div>
      <div className="top-actions">
        <button
          className="square-button"
          type="button"
          onClick={onToggleLeftRail}
          title={`${showLeftRail ? "Collapse" : "Open"} stories sidebar ([)`}
          aria-label={`${showLeftRail ? "Collapse" : "Open"} stories sidebar`}
        >
          {showLeftRail ? <PanelLeftClose size={16} /> : <PanelLeftOpen size={16} />}
        </button>
        <button
          className="square-button"
          type="button"
          onClick={onToggleInspector}
          title={`${showInspector ? "Collapse" : "Open"} inspector (])`}
          aria-label={`${showInspector ? "Collapse" : "Open"} inspector`}
        >
          {showInspector ? <PanelRightClose size={16} /> : <PanelRightOpen size={16} />}
        </button>
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

function StatusCell({
  icon,
  label,
  fullLabel,
  value,
  strong = false,
}: {
  icon?: ReactNode;
  label: string;
  fullLabel?: string;
  value: string;
  strong?: boolean;
}) {
  return (
    <div className="status-cell" aria-label={`${fullLabel || label}: ${value}`}>
      {icon && <span className="status-icon" aria-hidden="true">{icon}</span>}
      <span title={fullLabel || label}>{label}</span>
      <strong className={strong ? "status-strong" : undefined} title={value}>
        {value}
      </strong>
    </div>
  );
}
