import { describe, expect, it } from "vitest";
import { displayModelName, generationSummary } from "./MessageDiagnostics";

describe("generation summary", () => {
  it("keeps dated provider deployments out of the compact model label", () => {
    expect(displayModelName("gpt-5.4-mini-2026-03-17")).toBe("gpt-5.4-mini");
    expect(displayModelName("chatgpt-gpt-5.4-mini")).toBe("gpt-5.4-mini");
  });
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
