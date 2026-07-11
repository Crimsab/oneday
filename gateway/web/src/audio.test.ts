import { afterEach, describe, expect, it, vi } from "vitest";
import { createMessageAudio, getMessageAudio, getTTSCatalog, updateTTSSettings } from "./api";
import { assetUrl } from "./components/AudioControls";
import type { AudioAsset, StoryTTSSettings } from "./types";

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; vi.restoreAllMocks(); });

describe("canonical audio client", () => {
  it("uses encoded branch-safe API routes and carries story revision", async () => {
    globalThis.fetch = vi.fn(async () => new Response(JSON.stringify({ assets: [], jobs: [], providers: [], voices: [], settings: {} }), { status: 200, headers: { "content-type": "application/json" } })) as typeof fetch;
    await getTTSCatalog("it-IT");
    await getMessageAudio("story/one", 42);
    await createMessageAudio("story/one", 42);
    const settings = { story_id: "story/one", mode: "all", autoplay: false, default_language_tag: "it-IT", provider_policy: {} } satisfies StoryTTSSettings;
    await updateTTSSettings("story/one", settings, 7);
    expect(globalThis.fetch).toHaveBeenNthCalledWith(1, "/api/tts/voices?language=it-IT", {});
    expect(globalThis.fetch).toHaveBeenNthCalledWith(2, "/api/stories/story%2Fone/messages/42/audio", {});
    expect(globalThis.fetch).toHaveBeenNthCalledWith(3, "/api/stories/story%2Fone/messages/42/audio", { method: "POST" });
    const update = vi.mocked(globalThis.fetch).mock.calls[3];
    expect(JSON.parse(String((update[1] as RequestInit).body))).toMatchObject({ client_revision: 7, mode: "all" });
  });

  it("serves an immutable asset by opaque encoded id", () => {
    expect(assetUrl({ id: "audio/id", status: "ready" } as AudioAsset)).toBe("/api/audio/audio%2Fid");
  });
});
