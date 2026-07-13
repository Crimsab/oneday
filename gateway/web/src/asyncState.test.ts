import { describe, expect, it } from "vitest";
import { coalesceRequest, isCurrentAsyncSelection } from "./asyncState";

describe("async story selection guard", () => {
  it("accepts only the latest request for the selected story", () => {
    expect(isCurrentAsyncSelection("story-2", "story-2", 4, 4)).toBe(true);
    expect(isCurrentAsyncSelection("story-1", "story-2", 4, 4)).toBe(false);
    expect(isCurrentAsyncSelection("story-2", "story-2", 3, 4)).toBe(false);
  });
});

describe("coalesceRequest", () => {
  it("shares concurrent reads and clears the key after completion", async () => {
    const inFlight = new Map<string, Promise<number>>();
    let calls = 0;
    let resolve!: (value: number) => void;
    const load = () => {
      calls += 1;
      return new Promise<number>((done) => { resolve = done; });
    };
    const first = coalesceRequest(inFlight, "story", load);
    const second = coalesceRequest(inFlight, "story", load);
    expect(second).toBe(first);
    expect(calls).toBe(1);
    resolve(7);
    await expect(first).resolves.toBe(7);
    await Promise.resolve();
    expect(inFlight.has("story")).toBe(false);
    await coalesceRequest(inFlight, "story", async () => {
      calls += 1;
      return 8;
    });
    expect(calls).toBe(2);
  });

  it("clears failed reads so a retry can run", async () => {
    const inFlight = new Map<string, Promise<number>>();
    await expect(coalesceRequest(inFlight, "story", async () => { throw new Error("offline"); })).rejects.toThrow("offline");
    await Promise.resolve();
    await expect(coalesceRequest(inFlight, "story", async () => 9)).resolves.toBe(9);
  });
});
