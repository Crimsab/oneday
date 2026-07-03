import { CheckCircle2, CornerDownRight } from "lucide-react";
import { groupCommandSuggestions, type SlashCommandItem } from "../commands";

interface CommandPaletteProps {
  items: SlashCommandItem[];
  activeIndex: number;
  variant: "inline" | "full";
  onActiveIndexChange: (index: number) => void;
  onPick: (item: SlashCommandItem) => void;
}

export function CommandPalette({ items, activeIndex, variant, onActiveIndexChange, onPick }: CommandPaletteProps) {
  if (items.length === 0) return null;

  let index = 0;
  return (
    <div className={`command-palette ${variant}`} role="listbox" aria-label="Command palette">
      <div className="command-palette-head">
        <span>{variant === "full" ? "Command Palette" : "Slash Commands"}</span>
        <kbd>Enter</kbd>
      </div>
      {groupCommandSuggestions(items).map((group) => (
        <div className="command-group" key={group.key}>
          <div className="command-group-title">{group.label}</div>
          {group.items.map((item) => {
            const rowIndex = index;
            index += 1;
            const active = rowIndex === activeIndex;
            return (
              <button
                type="button"
                className={`command-row ${active ? "active" : ""}`}
                key={`${item.group}-${item.kind}-${item.name}-${item.value}`}
                role="option"
                aria-selected={active}
                onMouseEnter={() => onActiveIndexChange(rowIndex)}
                onClick={() => onPick(item)}
              >
                <span className="command-row-icon">{item.kind === "command" ? <CornerDownRight size={14} /> : <CheckCircle2 size={14} />}</span>
                <span className="command-row-main">
                  <strong>{item.name}</strong>
                  <small>{item.hint}</small>
                </span>
                <span className="command-row-meta">
                  {item.badge && <em>{item.badge}</em>}
                  {item.aliases.length > 0 && <small>{item.aliases.slice(0, 3).join(" ")}</small>}
                </span>
              </button>
            );
          })}
        </div>
      ))}
    </div>
  );
}
