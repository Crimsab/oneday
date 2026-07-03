import type { ChoiceView } from "./types";

export type ChoiceTone = "social" | "explore" | "stealth" | "force" | "craft" | "survive" | "lore" | "meta" | "neutral";

export interface ChoicePresentation {
  tone: ChoiceTone;
  title: string;
  meta: string[];
  gain: string;
  tradeoff: string;
}

const forceIntents = new Set(["attack", "combat", "aggressive", "force"]);
const exploreIntents = new Set(["explore", "observe"]);

export function choicePresentation(choice: ChoiceView, index = 0): ChoicePresentation {
  const intent = clean(choice.intent);
  const risk = clean(choice.risk);
  const scope = clean(choice.scope);
  const certainty = clean(choice.certainty);
  const stats = (choice.related_stats ?? []).map((stat) => stat.trim()).filter(Boolean);
  const tone = toneForChoice(intent, risk, index);

  const meta = [
    intent ? `intent:${intent}` : "",
    risk ? `risk:${risk}` : "",
    certainty ? `certainty:${certainty}` : "",
    scope ? `scope:${scope}` : "",
    ...stats.map((stat) => stat.toUpperCase()),
  ].filter(Boolean);

  return {
    tone,
    title: `Choice ${choice.id}`,
    meta,
    gain: gainText(intent, scope, stats),
    tradeoff: tradeoffText(risk, certainty, stats.length > 0),
  };
}

export function toneForChoice(intent: string, risk: string, index = 0): ChoiceTone {
  if (forceIntents.has(intent)) return "force";
  if (intent === "social") return "social";
  if (intent === "stealth" || intent === "flee") return "stealth";
  if (intent === "craft" || intent === "use_item") return "craft";
  if (intent === "survive") return "survive";
  if (intent === "lore") return "lore";
  if (exploreIntents.has(intent)) return "explore";
  if (intent === "meta") return "meta";
  if (risk === "high" || risk === "extreme" || risk === "desperate") return "force";
  const fallback: ChoiceTone[] = ["social", "explore", "stealth", "craft", "survive", "lore"];
  return fallback[index % fallback.length] ?? "neutral";
}

function gainText(intent: string, scope: string, stats: string[]): string {
  const statText = stats.length ? ` ${stats.map((stat) => stat.toUpperCase()).join(", ")}.` : "";
  switch (intent) {
    case "social":
      return `Social leverage.${statText}`;
    case "stealth":
      return `Low profile.${statText}`;
    case "flee":
      return `Escape/safety.${statText}`;
    case "attack":
    case "combat":
    case "aggressive":
      return `Direct pressure.${statText}`;
    case "explore":
    case "observe":
      return `Clues/context.${statText}`;
    case "craft":
    case "use_item":
      return `Resources/prep.${statText}`;
    case "survive":
      return `Preserve resources.${statText}`;
    case "lore":
      return `World knowledge.${statText}`;
    case "meta":
      return `Narrator framing.${statText}`;
    default:
      if (scope) return `Affects ${scope}.${statText}`;
      return `Freeform angle.${statText}`;
  }
}

function tradeoffText(risk: string, certainty: string, hasStats: boolean): string {
  const statCaveat = hasStats ? " Stats not guaranteed." : " No stat metadata.";
  switch (risk) {
    case "low":
      return `Low risk.${statCaveat}`;
    case "medium":
      return `Medium risk.${statCaveat}`;
    case "high":
    case "extreme":
      return `High risk/cost.${statCaveat}`;
    case "unknown":
      return `Unknown risk.${statCaveat}`;
    default:
      if (certainty === "safe") return `Predictable, not automatic.${statCaveat}`;
      if (certainty === "uncertain") return `Uncertain outcome.${statCaveat}`;
      if (certainty === "desperate") return `Desperate/costly.${statCaveat}`;
      return `No risk metadata.${statCaveat}`;
  }
}

function clean(value: string | undefined): string {
  return value?.trim().toLowerCase() ?? "";
}
