import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError, getStories } from "./api";

const originalFetch = globalThis.fetch;

describe("api request handling", () => {
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns JSON payloads from the gateway", async () => {
    mockFetch(new Response(JSON.stringify([{ id: "story", name: "Story" }]), { status: 200 }));
    await expect(getStories()).resolves.toMatchObject([{ id: "story", name: "Story" }]);
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
