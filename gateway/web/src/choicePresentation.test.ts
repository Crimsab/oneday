import { describe, expect, it } from "vitest";
import { choicePresentation, toneForChoice } from "./choicePresentation";

describe("toneForChoice", () => {
  it("maps semantic intent and risk into distinct visual tones", () => {
    expect(toneForChoice("social", "", 0)).toBe("social");
    expect(toneForChoice("observe", "", 0)).toBe("explore");
    expect(toneForChoice("stealth", "", 0)).toBe("stealth");
    expect(toneForChoice("attack", "", 0)).toBe("force");
    expect(toneForChoice("", "high", 0)).toBe("force");
  });

  it("uses index-based fallback tones for choices without metadata", () => {
    expect(toneForChoice("", "", 0)).toBe("social");
    expect(toneForChoice("", "", 1)).toBe("explore");
    expect(toneForChoice("", "", 2)).toBe("stealth");
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
      0,
    );

    expect(presentation.tone).toBe("social");
    expect(presentation.title).toBe("Choice 2");
    expect(presentation.meta).toEqual(["intent:social", "risk:medium", "certainty:uncertain", "scope:npc", "CHA", "WIL"]);
    expect(presentation.gain).toContain("Social leverage");
    expect(presentation.tradeoff).toContain("Medium risk");
  });

  it("is explicit when metadata is missing", () => {
    const presentation = choicePresentation({ id: 1, text: "Do something" }, 1);
    expect(presentation.tone).toBe("explore");
    expect(presentation.meta).toEqual([]);
    expect(presentation.gain).toContain("Freeform angle");
    expect(presentation.tradeoff).toContain("No risk metadata");
  });
});
