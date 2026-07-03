import { ChevronDown, CornerDownLeft } from "lucide-react";
import { slashCommands } from "../commands";

interface ComposerProps {
  draft: string;
  mode: string;
  disabled: boolean;
  notice: string;
  onDraftChange: (value: string) => void;
  onModeChange: (value: string) => void;
  onSubmit: () => void;
  onHistoryStep: (direction: -1 | 1) => string | null;
}

export function Composer({ draft, mode, disabled, notice, onDraftChange, onModeChange, onSubmit, onHistoryStep }: ComposerProps) {
  const suggestions = draft.trim().startsWith("/")
    ? slashCommands.filter((command) => {
        const query = draft.trim().slice(1).toLowerCase();
        return command.name.slice(1).startsWith(query) || command.aliases.some((alias) => alias.slice(1).startsWith(query));
      })
    : [];

  return (
    <form
      className="composer"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      <div className="composer-title">Action Composer</div>
      <div className="composer-row">
        <span className="prompt-marker">&gt;</span>
        <textarea
          value={draft}
          onChange={(event) => onDraftChange(event.target.value)}
          onKeyDown={(event) => {
            if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
              event.preventDefault();
              onSubmit();
            }
            if (event.key === "ArrowUp") {
              const next = onHistoryStep(-1);
              if (next !== null) {
                event.preventDefault();
                onDraftChange(next);
              }
            }
            if (event.key === "ArrowDown") {
              const next = onHistoryStep(1);
              if (next !== null) {
                event.preventDefault();
                onDraftChange(next);
              }
            }
          }}
          placeholder="Enter command or action..."
          rows={2}
        />
        <label className="select-wrap">
          <select value={mode} onChange={(event) => onModeChange(event.target.value)}>
            <option value="action">Action</option>
            <option value="talk">Talk</option>
            <option value="advance">Advance</option>
            <option value="timeskip">Time Skip</option>
          </select>
          <ChevronDown size={14} />
        </label>
        <button type="submit" className="execute-button" disabled={disabled || !draft.trim()}>
          <CornerDownLeft size={16} />
          Execute
        </button>
      </div>
      <div className="composer-tip">
        <span>{notice || "Tip: /advance, /timeskip, /downtime, /talk, /inventory, /stats, /characters, /fronts, /projects, /history, /saves"}</span>
        <span>Ctrl+Enter to execute</span>
      </div>
      {suggestions.length > 0 && (
        <div className="slash-suggestions">
          {suggestions.slice(0, 7).map((command) => (
            <button type="button" key={command.name} onClick={() => onDraftChange(`${command.name} `)}>
              <strong>{command.name}</strong>
              <span>{command.hint}</span>
            </button>
          ))}
        </div>
      )}
    </form>
  );
}
