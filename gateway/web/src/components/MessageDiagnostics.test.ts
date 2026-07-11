import { describe, expect, it } from "vitest";
import { generationSummary } from "./MessageDiagnostics";

describe("generation summary", () => {
  it("shows only player-safe provider aggregates", () => {
    const metadata = {
      provider: "codex",
      model: "gpt-5.5",
      latency_ms: 1525,
      streamed: true,
      usage: { total_tokens: 1234, reasoning_tokens: 321 },
      prompt: "never expose this",
      chain_of_thought: "nor this",
    };

    const summary = generationSummary(metadata);
    expect(summary).toContain("codex · gpt-5.5 · 1.5 s");
    expect(summary).toContain("1,234 tokens · streamed");
    expect(summary).not.toContain("never expose");
    expect(summary).not.toContain("321");
  });

  it("stays absent for legacy messages without generation aggregates", () => {
    expect(generationSummary({ output: { narrative: "A quiet room." } })).toBe("");
  });
});
