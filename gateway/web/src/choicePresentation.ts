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
  const statText = stats.length ? ` Uses ${stats.map((stat) => stat.toUpperCase()).join(", ")}.` : "";
  switch (intent) {
    case "social":
      return `Builds rapport, pressure, or information through a character.${statText}`;
    case "stealth":
      return `Keeps attention low and tries to preserve initiative.${statText}`;
    case "flee":
      return `Prioritizes escape and survival over control of the scene.${statText}`;
    case "attack":
    case "combat":
    case "aggressive":
      return `Pushes for direct leverage or confrontation.${statText}`;
    case "explore":
    case "observe":
      return `Looks for new context, clues, routes, or scene details.${statText}`;
    case "craft":
    case "use_item":
      return `Uses resources or preparation to change the situation.${statText}`;
    case "survive":
      return `Protects core resources and tries to stay in the scene.${statText}`;
    case "lore":
      return `Trades tempo for world knowledge or hidden context.${statText}`;
    case "meta":
      return `Asks the narrator to steer framing or pacing.${statText}`;
    default:
      if (scope) return `Affects ${scope}; exact payoff depends on narration.${statText}`;
      return `Narrative suggestion; the exact payoff is not guaranteed.${statText}`;
  }
}

function tradeoffText(risk: string, certainty: string, hasStats: boolean): string {
  const statCaveat = hasStats ? " Related stats may influence outcome, but do not guarantee success." : " No related stat metadata was provided.";
  switch (risk) {
    case "low":
      return `Lower danger, usually lower immediate upside.${statCaveat}`;
    case "medium":
      return `Balanced upside and downside; consequences can still branch.${statCaveat}`;
    case "high":
    case "extreme":
      return `High danger or cost; stronger payoff if it lands.${statCaveat}`;
    case "unknown":
      return `Risk is intentionally unclear; expect hidden information.${statCaveat}`;
    default:
      if (certainty === "safe") return `Likely predictable, but not an automatic reward.${statCaveat}`;
      if (certainty === "uncertain") return `Outcome is uncertain and may reveal complications.${statCaveat}`;
      if (certainty === "desperate") return `Desperate move; likely costly even on success.${statCaveat}`;
      return `No explicit risk metadata; judge from the scene text.${statCaveat}`;
  }
}

function clean(value: string | undefined): string {
  return value?.trim().toLowerCase() ?? "";
}
