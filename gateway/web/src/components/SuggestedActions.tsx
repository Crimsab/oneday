import { ChevronRight, DoorOpen, FileSearch, MessageSquare, PackageSearch, Search, Users } from "lucide-react";
import type { ChoiceView, StorySnapshot } from "../types";
import { compactText } from "../format";
import { choicePresentation, type ChoicePresentation } from "../choicePresentation";

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
  const visibleChoices = choices.slice(0, 3);
  return (
    <div className="choice-surface">
      <header className="choice-heading">
        <div><span>Your move</span><h2>What do you do?</h2></div>
        <small>Choose a path or write your own action below.</small>
      </header>
      <div className="choice-stack">
      {visibleChoices.length > 0 ? (
          <>
            {visibleChoices.map((choice) => {
              const presentation = choicePresentation(choice, choice.id - 1);
              const outcome = choiceOutcome(presentation);
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
                    <kbd>{choice.id}</kbd>
                  </span>
                  <span className="choice-copy">
                    <strong>{choice.text}</strong>
                    <ChoiceMetadata choice={choice} presentation={presentation} />
                    {outcome && (
                      <small className="choice-outcome" title={metadataTitle}>
                        {outcome}
                      </small>
                    )}
                  </span>
                  <ChevronRight className="choice-arrow" size={18} aria-hidden="true" />
                </button>
              );
            })}
          </>
        ) : (
          fallback.map((action, index) => {
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
                <ChevronRight className="choice-arrow" size={18} aria-hidden="true" />
              </button>
            );
          })
        )}
      </div>
    </div>
  );
}

function choiceOutcome(presentation: ChoicePresentation): string {
  const gain = firstSentence(presentation.gain);
  const tradeoff = firstSentence(presentation.tradeoff);
  if (gain && tradeoff) return `Gain: ${gain}. Risk: ${tradeoff}.`;
  if (gain) return `Gain: ${gain}.`;
  if (tradeoff) return `Risk: ${tradeoff}.`;
  return "";
}

function firstSentence(value: string): string {
  const clean = value.trim();
  if (!clean) return "";
  return clean.split(".")[0]?.trim() ?? "";
}

function ChoiceMetadata({ choice, presentation }: { choice: ChoiceView; presentation: ChoicePresentation }) {
  const chips = [
    ["INT", choice.intent],
    ["RISK", choice.risk],
    ["SCOPE", choice.scope],
    ["CERT", choice.certainty],
  ].filter((item): item is [string, string] => Boolean(item[1]));
  if (!chips.length && !presentation.meta.length) return null;
  return (
    <span className="choice-meta">
      {chips.slice(0, 4).map(([label, value]) => (
        <small key={`${label}:${value}`}>
          <b>{label}</b> {value}
        </small>
      ))}
    </span>
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
