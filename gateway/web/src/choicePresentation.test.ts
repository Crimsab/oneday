import { describe, expect, it } from "vitest";
import { choicePresentation, toneForChoice } from "./choicePresentation";

describe("toneForChoice", () => {
  it("maps semantic intent and risk into distinct visual tones", () => {
    expect(toneForChoice("social", "")).toBe("social");
    expect(toneForChoice("observe", "")).toBe("explore");
    expect(toneForChoice("stealth", "")).toBe("stealth");
    expect(toneForChoice("attack", "")).toBe("force");
    expect(toneForChoice("", "high")).toBe("force");
  });

  it("uses a neutral tone without real metadata", () => {
    expect(toneForChoice("", "")).toBe("neutral");
  });
});

describe("choicePresentation", () => {
  it("summarizes what a rich choice signals and does not guarantee", () => {
    const presentation = choicePresentation(
      {
        id: 2,
        text: "Talk your way past the guard",
        intent: "social",
        risk: "medium",
        scope: "npc",
        certainty: "uncertain",
        related_stats: ["cha", "wil"],
      },
    );

    expect(presentation.tone).toBe("social");
    expect(presentation.title).toBe("Choice 2");
    expect(presentation.hasMetadata).toBe(true);
    expect(presentation.meta).toEqual(["intent:social", "risk:medium", "certainty:uncertain", "scope:npc", "CHA", "WIL"]);
    expect(presentation.gain).toContain("Social leverage");
    expect(presentation.tradeoff).toContain("Medium risk");
  });

  it("keeps fallback debug copy out of choices without metadata", () => {
    const presentation = choicePresentation({ id: 1, text: "Do something" });
    expect(presentation.tone).toBe("neutral");
    expect(presentation.hasMetadata).toBe(false);
    expect(presentation.meta).toEqual([]);
    expect(presentation.gain).toBe("");
    expect(presentation.tradeoff).toBe("");
  });
});
