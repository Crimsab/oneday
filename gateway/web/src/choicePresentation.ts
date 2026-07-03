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
  const statText = stats.length ? ` Stats: ${stats.map((stat) => stat.toUpperCase()).join(", ")}.` : "";
  switch (intent) {
    case "social":
      return `Social leverage, rapport, or information.${statText}`;
    case "stealth":
      return `Low-profile move; preserves initiative.${statText}`;
    case "flee":
      return `Escape/safety over scene control.${statText}`;
    case "attack":
    case "combat":
    case "aggressive":
      return `Direct pressure or confrontation.${statText}`;
    case "explore":
    case "observe":
      return `Finds clues, routes, or scene context.${statText}`;
    case "craft":
    case "use_item":
      return `Uses resources or preparation.${statText}`;
    case "survive":
      return `Protects core resources and position.${statText}`;
    case "lore":
      return `World knowledge or hidden context.${statText}`;
    case "meta":
      return `Narrator framing or pacing.${statText}`;
    default:
      if (scope) return `Affects ${scope}; payoff depends on narration.${statText}`;
      return `Freeform narrative angle; payoff depends on scene.${statText}`;
  }
}

function tradeoffText(risk: string, certainty: string, hasStats: boolean): string {
  const statCaveat = hasStats ? " Stats influence, not guarantee." : " No stat metadata.";
  switch (risk) {
    case "low":
      return `Lower danger, usually lower upside.${statCaveat}`;
    case "medium":
      return `Balanced upside/downside.${statCaveat}`;
    case "high":
    case "extreme":
      return `High danger/cost; bigger payoff if it lands.${statCaveat}`;
    case "unknown":
      return `Risk unclear; expect hidden information.${statCaveat}`;
    default:
      if (certainty === "safe") return `Predictable, not automatic reward.${statCaveat}`;
      if (certainty === "uncertain") return `Uncertain; may reveal complications.${statCaveat}`;
      if (certainty === "desperate") return `Desperate; likely costly even on success.${statCaveat}`;
      return `No risk metadata; judge from scene text.${statCaveat}`;
  }
}

function clean(value: string | undefined): string {
  return value?.trim().toLowerCase() ?? "";
}
