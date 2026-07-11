import { describe, expect, it } from "vitest";
import fixture from "../../../contracts/challenge-v1.json";
import type { ChallengeInstance, ChallengeInput, ChallengeResolution } from "./types";

describe("portable challenge contract", () => {
  it("consumes the same v1 fixture as Go and Rust", () => {
    const instance = fixture.instance as ChallengeInstance;
    const input = fixture.input as ChallengeInput;
    const resolution = fixture.resolution as ChallengeResolution;
    expect(instance.protocol_version).toBe(1);
    expect(resolution.instance_id).toBe(instance.id);
    expect(resolution.outcome.seed).toBe(instance.seed);
    expect(resolution.input.intent).toBe(input.intent);
  });
});
