import { afterEach, describe, expect, it, vi } from "vitest";
import { cancelAudioJob, cleanupAudio, createMessageAudio, deletePronunciation, getAudioExport, getMessageAudio, getPronunciations, getTTSCatalog, retryAudioJob, updatePronunciation, updateTTSSettings } from "./api";
import { assetUrl, firstReadyAudioAsset } from "./components/AudioControls";
import type { AudioAsset, PronunciationEntry, StoryTTSSettings } from "./types";

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; vi.restoreAllMocks(); });

describe("canonical audio client", () => {
  it("uses encoded branch-safe API routes and carries story revision", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(JSON.stringify({ assets: [], jobs: [], providers: [], voices: [], settings: {} }), { status: 200, headers: { "content-type": "application/json" } }));
    globalThis.fetch = fetchMock as typeof fetch;
    await getTTSCatalog("it-IT");
    await getMessageAudio("story/one", 42);
    await createMessageAudio("story/one", 42);
    const settings = { story_id: "story/one", mode: "all", autoplay: false, default_language_tag: "it-IT", provider_policy: {} } satisfies StoryTTSSettings;
    await updateTTSSettings("story/one", settings, 7);
    expect(globalThis.fetch).toHaveBeenNthCalledWith(1, "/api/tts/voices?language=it-IT", expect.objectContaining({ signal: expect.any(AbortSignal) }));
    expect(globalThis.fetch).toHaveBeenNthCalledWith(2, "/api/stories/story%2Fone/messages/42/audio", expect.objectContaining({ signal: expect.any(AbortSignal) }));
    expect(globalThis.fetch).toHaveBeenNthCalledWith(3, "/api/stories/story%2Fone/messages/42/audio", expect.objectContaining({ method: "POST", signal: expect.any(AbortSignal) }));
    const update = fetchMock.mock.calls[3];
    expect(JSON.parse(String((update[1] as RequestInit).body))).toMatchObject({ client_revision: 7, mode: "all" });
  });

  it("serves an immutable asset by opaque encoded id", () => {
    expect(assetUrl({ id: "audio/id", status: "ready" } as AudioAsset)).toBe("/api/audio/audio%2Fid");
  });

  it("selects one stable ready asset for autoplay", () => {
    const assets = [
      { id: "queued", status: "queued" },
      { id: "ready-first", status: "ready" },
      { id: "ready-second", status: "ready" },
    ] as AudioAsset[];
    expect(firstReadyAudioAsset(assets)?.id).toBe("ready-first");
  });

  it("encodes retry, cancel, lexicon, cleanup, and export operations", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => new Response(JSON.stringify({ assets: [], jobs: [], pronunciations: [], cleanup: {}, export: {} }), { status: 200, headers: { "content-type": "application/json" } }));
    globalThis.fetch = fetchMock as typeof fetch;
    await retryAudioJob("story/one", "job/1");
    await cancelAudioJob("story/one", "job/1");
    await getPronunciations("story/one", "it-IT");
    const entry = { id: "pron/1", story_id: "story/one", language_tag: "it-IT", source_text: "Lyanna", pronunciation: "Lianna", alphabet: "provider", case_sensitive: false, revision: 1 } satisfies PronunciationEntry;
    await updatePronunciation("story/one", entry, 9);
    await deletePronunciation("story/one", entry.id, 9);
    await cleanupAudio("story/one", true);
    await getAudioExport("story/one");
    expect(globalThis.fetch).toHaveBeenNthCalledWith(1, "/api/stories/story%2Fone/audio/jobs/job%2F1/retry", expect.objectContaining({ method: "POST", signal: expect.any(AbortSignal) }));
    expect(globalThis.fetch).toHaveBeenNthCalledWith(2, "/api/stories/story%2Fone/audio/jobs/job%2F1/cancel", expect.objectContaining({ method: "POST", signal: expect.any(AbortSignal) }));
    expect(globalThis.fetch).toHaveBeenNthCalledWith(3, "/api/stories/story%2Fone/pronunciations?language=it-IT", expect.objectContaining({ signal: expect.any(AbortSignal) }));
    expect(globalThis.fetch).toHaveBeenNthCalledWith(5, "/api/stories/story%2Fone/pronunciations/pron%2F1?client_revision=9", expect.objectContaining({ method: "DELETE", signal: expect.any(AbortSignal) }));
    expect(globalThis.fetch).toHaveBeenNthCalledWith(7, "/api/stories/story%2Fone/audio/export", expect.objectContaining({ signal: expect.any(AbortSignal) }));
    const cleanup = fetchMock.mock.calls[5][1] as RequestInit;
    expect(JSON.parse(String(cleanup.body))).toEqual({ dry_run: true });
  });
});
