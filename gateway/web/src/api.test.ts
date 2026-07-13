import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ApiRequestError,
  cancelVisualGenerationJob,
  cleanupVisualAssetFiles,
  createStory,
  deleteStory,
  generateVisualAssets,
  getChapters,
  getHistory,
  getMessageAudio,
  getMessageDiagnostics,
  getActiveMiniGame,
  getStories,
  getStoryExport,
  getTelemetryExport,
  getStoryDeletePlan,
  getTimeline,
  stepVisualAssetSelection,
  startMiniGame,
  inputMiniGame,
  updateTimeline,
  updateStory,
} from "./api";

const originalFetch = globalThis.fetch;

describe("api request handling", () => {
  afterEach(() => {
	vi.useRealTimers();
    globalThis.fetch = originalFetch;
  });

  it("returns JSON payloads from the gateway", async () => {
    mockFetch(new Response(JSON.stringify([{ id: "story", name: "Story" }]), { status: 200 }));
    await expect(getStories()).resolves.toMatchObject([{ id: "story", name: "Story" }]);
  });

	it("aborts stalled reads with a typed timeout error", async () => {
		vi.useFakeTimers();
		globalThis.fetch = vi.fn((_path: RequestInfo | URL, options?: RequestInit) => new Promise<Response>((_resolve, reject) => {
			options?.signal?.addEventListener("abort", () => reject(options.signal?.reason), { once: true });
		})) as typeof fetch;
		const pending = getStories();
		vi.advanceTimersByTime(30_000);
		await expect(pending).rejects.toMatchObject({ code: "request_timeout", status: 408 });
	});

  it("normalizes omitted empty audio collections", async () => {
    mockFetch(new Response(JSON.stringify({}), { status: 200 }));

    await expect(getMessageAudio("story-1", 42)).resolves.toEqual({ assets: [], jobs: [] });
  });

  it("posts browser story creation requests to the gateway", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          story: { story_id: "story-1", character_id: "char-1", started: true },
          snapshot: { story: { id: "story-1" } },
        }),
        { status: 200 },
      ),
    );

    await expect(
      createStory({
        brief: "short test",
        character_name: "Tester",
        character_background: "",
        start: true,
      }),
    ).resolves.toMatchObject({ story: { story_id: "story-1" } });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"brief\":\"short test\""),
      }),
    );
  });

  it("posts visual image generation requests to the story gateway", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          profile: { story_id: "story-1" },
          assets: [{ id: "asset-1", status: "running" }],
        }),
        { status: 200 },
      ),
    );

    await expect(generateVisualAssets("story-1", { asset_ids: ["asset-1"], force: true, limit: 1 })).resolves.toMatchObject({
      assets: [{ id: "asset-1", status: "running" }],
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1/visual-assets/generate",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"asset_ids\":[\"asset-1\"]"),
      }),
    );
  });

  it("cancels visual generation jobs through the story gateway", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          profile: { story_id: "story-1" },
          assets: [{ id: "asset-1", status: "pending" }],
          jobs: [{ id: 12, status: "cancelled" }],
        }),
        { status: 200 },
      ),
    );

    await expect(cancelVisualGenerationJob("story-1", 12)).resolves.toMatchObject({
      jobs: [{ id: 12, status: "cancelled" }],
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1/visual-assets/jobs/12/cancel",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("requests visual asset cleanup without exposing file contents", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          story_id: "story-1",
          dry_run: true,
          deleted_files: ["/tmp/stale.png"],
          kept_files: ["/tmp/kept.png"],
        }),
        { status: 200 },
      ),
    );

    await expect(cleanupVisualAssetFiles("story-1", { dry_run: true })).resolves.toMatchObject({
      story_id: "story-1",
      dry_run: true,
      deleted_files: ["/tmp/stale.png"],
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1/visual-assets/cleanup",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"dry_run\":true"),
      }),
    );
  });

  it("patches story metadata and archive state", async () => {
    mockFetch(new Response(JSON.stringify({ id: "story-1", name: "Renamed", is_archived: true }), { status: 200 }));

    await expect(updateStory("story-1", { name: "Renamed", is_archived: true })).resolves.toMatchObject({
      id: "story-1",
      name: "Renamed",
      is_archived: true,
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1",
      expect.objectContaining({
        method: "PATCH",
        body: expect.stringContaining("\"is_archived\":true"),
      }),
    );
  });

  it("deletes stories through the gateway", async () => {
    mockFetch(new Response(JSON.stringify({ story_id: "story-1" }), { status: 200 }));

    await expect(deleteStory("story-1")).resolves.toEqual({ story_id: "story-1" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1",
      expect.objectContaining({
        method: "DELETE",
      }),
    );
  });

  it("loads story delete plans before destructive deletion", async () => {
    mockFetch(new Response(JSON.stringify({ story_id: "story-1", total_rows: 42, counts: [], retained_asset_files: [] }), { status: 200 }));

    await expect(getStoryDeletePlan("story-1")).resolves.toMatchObject({ story_id: "story-1", total_rows: 42 });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/delete-plan", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("uses the branch timeline contract for reads and guarded mutations", async () => {
    mockFetch(new Response(JSON.stringify({ active_branch_id: "branch-main", revision: 7, branches: [] }), { status: 200 }));
    await expect(getTimeline("story/one")).resolves.toMatchObject({ active_branch_id: "branch-main", revision: 7 });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story%2Fone/timeline", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ timeline: { active_branch_id: "branch-alt" }, snapshot: { story: { id: "story/one" } } }), { status: 200 }));
    await updateTimeline("story/one", { action: "checkout", client_revision: 7, branch_id: "branch-alt" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/timeline",
      expect.objectContaining({ method: "POST", body: expect.stringContaining('"client_revision":7') }),
    );
  });

  it("recovers timeline reads from transient server failures", async () => {
    mockFetchSequence([
      new Response(JSON.stringify({ error: "temporary timeline failure" }), { status: 500 }),
      new Response(JSON.stringify({ active_branch_id: "branch-main", revision: 9, branches: [], head: null, commits: [] }), { status: 200 }),
    ]);

    await expect(getTimeline("story-1")).resolves.toMatchObject({ revision: 9 });
    expect(globalThis.fetch).toHaveBeenCalledTimes(2);
  });

  it("requests branch-scoped paginated history, chapters, and exports", async () => {
    mockFetch(new Response(JSON.stringify({ items: [], next_cursor: null }), { status: 200 }));
    await getHistory("story-1", 41, "glass seal");
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/history?limit=40&q=glass+seal&cursor=41", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ items: [], next_cursor: null }), { status: 200 }));
    await getChapters("story-1", 3, "arrival");
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/chapters?limit=30&q=arrival&cursor=3", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ format: "json", filename: "history.json", content: "{}" }), { status: 200 }));
    await expect(getStoryExport("story-1", "json")).resolves.toMatchObject({ filename: "history.json" });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/export?format=json", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("loads message diagnostics and redacted telemetry exports", async () => {
    mockFetch(new Response(JSON.stringify({ run_id: "run-1", attempts: [] }), { status: 200 }));
    await expect(getMessageDiagnostics("story/one", 42)).resolves.toMatchObject({ run_id: "run-1" });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story%2Fone/messages/42/diagnostics", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ format: "jsonl", filename: "telemetry.jsonl", content: "", count: 0, truncated: false }), { status: 200 }));
    await expect(getTelemetryExport("story/one", 250)).resolves.toMatchObject({ format: "jsonl", count: 0 });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story%2Fone/telemetry/export?limit=250", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("steps branch-local visual selection history", async () => {
    mockFetch(new Response(JSON.stringify({ profile: {}, assets: [], jobs: [] }), { status: 200 }));
    await stepVisualAssetSelection("story/one", "asset one", "undo");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/visual-assets/asset%20one/selection/undo",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("uses the branch-scoped minigame host endpoints", async () => {
    mockFetch(new Response(JSON.stringify({ instance: null }), { status: 200 }));
    await getActiveMiniGame("story/one");
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story%2Fone/minigames", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ instance: { id: "mini-1" } }), { status: 200 }));
    await startMiniGame("story/one", "pattern");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/minigames",
      expect.objectContaining({ method: "POST", body: expect.stringContaining('"kind":"pattern"') }),
    );

    mockFetch(new Response(JSON.stringify({ instance: { id: "mini-1", runtime: { phase: "resolved" } } }), { status: 200 }));
    await inputMiniGame("story/one", "mini/1", { action: "submit", value: "8" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/minigames/mini%2F1/input",
      expect.objectContaining({ method: "POST", body: expect.stringContaining('"value":"8"') }),
    );
  });

  it("rejects successful non-JSON responses before they crash React state", async () => {
    mockFetch(new Response("<html>vite fallback</html>", { status: 200, headers: { "content-type": "text/html" } }));

    await expect(getStories()).rejects.toMatchObject({
      name: "ApiRequestError",
      status: 200,
      message: "Gateway returned a non-JSON response.",
    });
  });

  it("keeps non-JSON error responses controlled", async () => {
    mockFetch(new Response("", { status: 502, statusText: "Bad Gateway" }));
    await expect(getStories()).rejects.toBeInstanceOf(ApiRequestError);
    await expect(getStories()).rejects.toMatchObject({ status: 502, message: "Bad Gateway" });
  });
});

function mockFetch(response: Response) {
  globalThis.fetch = vi.fn(async () => response.clone()) as typeof fetch;
}

function mockFetchSequence(responses: Response[]) {
  let index = 0;
  globalThis.fetch = vi.fn(async () => responses[Math.min(index++, responses.length - 1)].clone()) as typeof fetch;
}
