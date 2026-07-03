import { DoorOpen, FileSearch, MessageSquare, PackageSearch, Search, Users } from "lucide-react";
import type { ChoiceView, StorySnapshot } from "../types";
import { compactText } from "../format";
import { choicePresentation } from "../choicePresentation";

interface SuggestedActionsProps {
  choices: ChoiceView[];
  snapshot: StorySnapshot | null;
  disabled?: boolean;
  onChoice: (choice: ChoiceView) => void;
  onDraft: (value: string) => void;
}

const choiceIcons = [MessageSquare, FileSearch, Search, PackageSearch, Users, DoorOpen];

export function SuggestedActions({ choices, snapshot, disabled = false, onChoice, onDraft }: SuggestedActionsProps) {
  const fallback = fallbackActions(snapshot);
  return (
    <div className="choice-stack">
      {choices.length > 0
        ? choices.slice(0, 6).map((choice, index) => {
            const Icon = choiceIcons[index % choiceIcons.length];
            const presentation = choicePresentation(choice, index);
            const metadataTitle = [presentation.gain, presentation.tradeoff, presentation.meta.join(" ")]
              .filter(Boolean)
              .join("\n");
            return (
              <button
                type="button"
                className="choice-row"
                data-choice-tone={presentation.tone}
                disabled={disabled}
                key={choice.id}
                onClick={() => onChoice(choice)}
                title={metadataTitle || "Suggested action"}
              >
                <span className="choice-index" aria-label={`Choice ${choice.id}`}>
                  <Icon size={18} />
                  <kbd>{choice.id}</kbd>
                </span>
                <span className="choice-copy">
                  <strong>{choice.text}</strong>
                  {presentation.meta.length > 0 && (
                    <span className="choice-meta">
                      {presentation.meta.slice(0, 4).map((item) => <small key={item}>{item}</small>)}
                    </span>
                  )}
                </span>
                {presentation.hasMetadata && (
                  <span className="choice-effects" aria-hidden={!metadataTitle}>
                    {presentation.gain && <small className="choice-effect plus">{presentation.gain}</small>}
                    {presentation.tradeoff && <small className="choice-effect minus">{presentation.tradeoff}</small>}
                  </span>
                )}
              </button>
            );
          })
        : fallback.map((action, index) => {
            const Icon = choiceIcons[index % choiceIcons.length];
            return (
              <button type="button" className="choice-row ghost" disabled={disabled} key={action.command} onClick={() => onDraft(action.command)}>
                <span className="choice-index" aria-hidden="true">
                  <Icon size={18} />
                </span>
                <span className="choice-copy">
                  <strong>{action.label}</strong>
                  <small>{action.hint}</small>
                </span>
              </button>
            );
          })}
    </div>
  );
}

function fallbackActions(snapshot: StorySnapshot | null) {
  const location = snapshot?.world.current_location || "the area";
  const npc = snapshot?.panels.npcs[0]?.name;
  return [
    { label: "Look around", hint: `Survey ${compactText(location, 34)}`, command: "look around carefully" },
    { label: npc ? `Talk to ${npc}` : "Talk to someone", hint: "Start a social beat", command: npc ? `/talk ${npc} ask ` : "/talk " },
    { label: "Examine clues", hint: "Search for useful context", command: "examine the most suspicious detail nearby" },
    { label: "Advance scene", hint: "Move past filler", command: "/advance " },
  ];
}
