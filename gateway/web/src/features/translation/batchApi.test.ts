import { afterEach, describe, expect, it, vi } from "vitest";
import { listTranslationJobs } from "./batchApi";

afterEach(() => vi.unstubAllGlobals());

describe("batch translation API", () => {
  it("returns job lists", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify([{ id: "job-1" }]), { status: 200 })));
    await expect(listTranslationJobs("story-1")).resolves.toEqual([{ id: "job-1" }]);
  });

  it("rejects malformed list payloads instead of crashing consumers", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({}), { status: 200 })));
    await expect(listTranslationJobs("story-1")).rejects.toThrow("unreadable response");
  });
});
