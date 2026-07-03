import { ChevronDown, CornerDownLeft } from "lucide-react";
import { commandSuggestions } from "../commands";
import type { CommandDescriptor } from "../types";

interface ComposerProps {
  draft: string;
  mode: string;
  disabled: boolean;
  notice: string;
  commandDescriptors: CommandDescriptor[];
  onDraftChange: (value: string) => void;
  onModeChange: (value: string) => void;
  onSubmit: () => void;
  onHistoryStep: (direction: -1 | 1) => string | null;
}

export function Composer({
  draft,
  mode,
  disabled,
  notice,
  commandDescriptors,
  onDraftChange,
  onModeChange,
  onSubmit,
  onHistoryStep,
}: ComposerProps) {
  const suggestions = commandSuggestions(draft, commandDescriptors);

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
            if (event.key === "Tab" && suggestions[0]) {
              event.preventDefault();
              onDraftChange(suggestions[0].value);
            }
            if (event.key === "ArrowUp") {
              if (suggestions.length > 0) {
                event.preventDefault();
                return;
              }
              const next = onHistoryStep(-1);
              if (next !== null) {
                event.preventDefault();
                onDraftChange(next);
              }
            }
            if (event.key === "ArrowDown") {
              if (suggestions.length > 0) {
                event.preventDefault();
                return;
              }
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
        <span>Tab accepts slash command - Ctrl+Enter executes</span>
      </div>
      {suggestions.length > 0 && (
        <div className="slash-suggestions">
          {suggestions.slice(0, 7).map((command) => (
            <button type="button" key={command.name} onClick={() => onDraftChange(command.value)}>
              <strong>{command.name}</strong>
              <span>{command.hint}</span>
            </button>
          ))}
        </div>
      )}
    </form>
  );
}
