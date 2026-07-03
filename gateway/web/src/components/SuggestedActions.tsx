import { useEffect, useState } from "react";
import { DoorOpen, FileSearch, MessageSquare, PackageSearch, Search, Users } from "lucide-react";
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
  const visibleChoices = choices.slice(0, 6);
  const choiceKey = visibleChoices.map((choice) => choice.id).join("|");
  const [activeChoiceId, setActiveChoiceId] = useState<number | null>(visibleChoices[0]?.id ?? null);

  useEffect(() => {
    setActiveChoiceId(visibleChoices[0]?.id ?? null);
  }, [choiceKey]);

  return (
    <div className="choice-stack">
      {visibleChoices.length > 0
        ? visibleChoices.map((choice, index) => {
            const Icon = choiceIcons[index % choiceIcons.length];
            const presentation = choicePresentation(choice, index);
            const isActive = choice.id === activeChoiceId;
            const metadataTitle = [presentation.gain, presentation.tradeoff, presentation.meta.join(" ")]
              .filter(Boolean)
              .join("\n");
            return (
              <button
                type="button"
                className="choice-row"
                data-active={isActive ? "true" : "false"}
                data-choice-tone={presentation.tone}
                disabled={disabled}
                key={choice.id}
                onFocus={() => setActiveChoiceId(choice.id)}
                onMouseEnter={() => setActiveChoiceId(choice.id)}
                onClick={() => onChoice(choice)}
                title={metadataTitle || "Suggested action"}
              >
                <span className="choice-index" aria-label={`Choice ${choice.id}`}>
                  <Icon size={18} />
                  <kbd>{choice.id}</kbd>
                </span>
                <span className="choice-copy">
                  <strong>{choice.text}</strong>
                  <ChoiceMetadata choice={choice} presentation={presentation} />
                </span>
                {presentation.hasMetadata && (
                  <span className="choice-effects" aria-hidden={!metadataTitle}>
                    {presentation.gain && <small className="choice-effect plus">{presentation.gain}</small>}
                    {presentation.tradeoff && <small className="choice-effect minus">{presentation.tradeoff}</small>}
                  </span>
                )}
                {isActive && <ChoiceDetails choice={choice} presentation={presentation} />}
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

function ChoiceDetails({ choice, presentation }: { choice: ChoiceView; presentation: ChoicePresentation }) {
  const stats = (choice.related_stats ?? []).slice(0, 4);
  return (
    <span className="choice-detail">
      <span className="choice-detail-cell">
        <b>Description</b>
        <small>{choice.text}</small>
      </span>
      <span className="choice-detail-cell">
        <b>Stats involved</b>
        {stats.length > 0 ? (
          <span className="choice-stat-list">
            {stats.map((stat, index) => (
              <span className="choice-stat" key={stat}>
                <span>{compactText(stat, 16)}</span>
                <span className="stat-dots" aria-hidden="true">
                  {Array.from({ length: 6 }, (_, dot) => (
                    <i className={dot <= Math.min(5, 2 + index) ? "filled" : ""} key={dot} />
                  ))}
                </span>
              </span>
            ))}
          </span>
        ) : (
          <small>No stat metadata.</small>
        )}
      </span>
      <span className="choice-detail-cell">
        <b>Possible outcomes</b>
        {presentation.gain && <small className="outcome-good">{presentation.gain}</small>}
        {presentation.tradeoff && <small className="outcome-risk">{presentation.tradeoff}</small>}
        <small>Not guaranteed. The scene context still decides.</small>
      </span>
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
