import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError, createStory, getStories } from "./api";

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
