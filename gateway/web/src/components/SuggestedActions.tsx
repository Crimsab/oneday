import { DoorOpen, FileSearch, MessageSquare, PackageSearch, Search, Users } from "lucide-react";
import type { ChoiceView, StorySnapshot } from "../types";
import { compactText } from "../format";
import { choicePresentation } from "../choicePresentation";

interface SuggestedActionsProps {
  choices: ChoiceView[];
  snapshot: StorySnapshot | null;
  onChoice: (choice: ChoiceView) => void;
  onDraft: (value: string) => void;
}

const choiceIcons = [MessageSquare, FileSearch, Search, PackageSearch, Users, DoorOpen];

export function SuggestedActions({ choices, snapshot, onChoice, onDraft }: SuggestedActionsProps) {
  const fallback = fallbackActions(snapshot);
  return (
    <div className="suggested-grid">
      {choices.length > 0
        ? choices.slice(0, 6).map((choice, index) => {
            const Icon = choiceIcons[index % choiceIcons.length];
            const presentation = choicePresentation(choice, index);
            return (
              <button
                type="button"
                className="suggested-card choice-card"
                data-choice-tone={presentation.tone}
                key={choice.id}
                onClick={() => onChoice(choice)}
                title={`${presentation.gain}\n${presentation.tradeoff}`}
              >
                <div className="choice-card-top">
                  <Icon size={18} />
                  <span>{presentation.title}</span>
                </div>
                <strong>{choice.text}</strong>
                <div className="choice-meta">
                  {presentation.meta.length > 0 ? presentation.meta.map((item) => <small key={item}>{item}</small>) : <small>suggested action</small>}
                </div>
                <span className="choice-effect plus">{presentation.gain}</span>
                <span className="choice-effect minus">{presentation.tradeoff}</span>
              </button>
            );
          })
        : fallback.map((action, index) => {
            const Icon = choiceIcons[index % choiceIcons.length];
            return (
              <button type="button" className="suggested-card ghost" key={action.command} onClick={() => onDraft(action.command)}>
                <Icon size={18} />
                <strong>{action.label}</strong>
                <span>{action.hint}</span>
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
