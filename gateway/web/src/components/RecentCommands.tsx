import type { RecentCommand } from "../types";
import { compactText } from "../format";

interface RecentCommandsProps {
  commands: RecentCommand[];
  onDraft: (value: string) => void;
}

export function RecentCommands({ commands, onDraft }: RecentCommandsProps) {
  return (
    <div className="recent-list">
      {commands.length === 0 ? (
        <div className="empty-copy">No commands yet.</div>
      ) : (
        commands.slice(0, 7).map((command) => (
          <button type="button" key={command.id} onClick={() => onDraft(command.text)}>
            <span>{compactText(command.text, 42)}</span>
            <small>Turn {command.turn}</small>
          </button>
        ))
      )}
    </div>
  );
}
