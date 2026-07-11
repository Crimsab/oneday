import { describe, expect, it } from "vitest";
import fixture from "../../../contracts/minigame-v1.json";
import type { MiniGameInput, MiniGameInstance } from "./types";

describe("portable minigame contract", () => {
  it("consumes the same player-safe v1 fixture as Go and Rust", () => {
    const instance = fixture.instance as MiniGameInstance;
    const input = fixture.input as MiniGameInput;
    expect(instance.protocol_version).toBe(1);
    expect(instance.runtime.phase).toBe("active");
    expect(input.action).toBe("submit");
    expect("answers" in instance.definition).toBe(false);
  });
});
