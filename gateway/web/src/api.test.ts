import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ApiRequestError,
  cancelVisualGenerationJob,
  cleanupVisualAssetFiles,
  createStory,
  deleteStory,
  generateVisualAssets,
  getStories,
  getStoryDeletePlan,
  updateStory,
} from "./api";

const originalFetch = globalThis.fetch;

describe("api request handling", () => {
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns JSON payloads from the gateway", async () => {
    mockFetch(new Response(JSON.stringify([{ id: "story", name: "Story" }]), { status: 200 }));
    await expect(getStories()).resolves.toMatchObject([{ id: "story", name: "Story" }]);
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
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/delete-plan", {});
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
