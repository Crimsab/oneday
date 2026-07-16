import { describe, expect, it, vi } from "vitest";
import { NETWORK_ONLY_PATH_PATTERNS, NETWORK_ONLY_URL_PATTERNS, isServerCanonicalPath, pwaManifest, pwaWorkbox } from "../../../pwa.config";
import { checkServerConnectivity } from "./PwaStatus";

describe("PWA manifest", () => {
  it("defines a standalone, scoped OneDay app with install icons", () => {
    expect(pwaManifest).toMatchObject({
      id: "/",
      name: "OneDay",
      short_name: "OneDay",
      start_url: "/",
      scope: "/",
      display: "standalone",
    });
    expect(pwaManifest.icons).toEqual(expect.arrayContaining([
      expect.objectContaining({ sizes: "192x192", type: "image/png" }),
      expect.objectContaining({ sizes: "512x512", type: "image/png" }),
    ]));
  });
});

describe("PWA cache boundary", () => {
  it.each([
    "/api/health",
    "/api/stories/story-1/events",
    "/api/stories/story-1/actions",
    "/generated/assets/story/scene.png",
  ])("keeps %s server-canonical", (pathname) => {
    expect(isServerCanonicalPath(pathname)).toBe(true);
  });

  it.each(["/", "/stories/story-1/history", "/assets/index-abc.js", "/brand/oneday-mark.png"])(
    "does not classify app-shell path %s as server state",
    (pathname) => expect(isServerCanonicalPath(pathname)).toBe(false),
  );

  it("uses NetworkOnly for every dynamic server path and no generic runtime cache", () => {
    expect(pwaWorkbox.navigateFallbackDenylist).toEqual(NETWORK_ONLY_PATH_PATTERNS);
    expect(pwaWorkbox.runtimeCaching.map((route) => route.urlPattern)).toEqual(NETWORK_ONLY_URL_PATTERNS);
    expect(pwaWorkbox.runtimeCaching.every((route) => route.handler === "NetworkOnly")).toBe(true);
  });
});

describe("server connectivity", () => {
  it("does not make a request while the browser is offline", async () => {
    const request = vi.fn<typeof fetch>();
    await expect(checkServerConnectivity(false, request)).resolves.toBe("offline");
    expect(request).not.toHaveBeenCalled();
  });

  it("requires a successful JSON health response", async () => {
    const healthy = vi.fn<typeof fetch>().mockResolvedValue(new Response("{}", {
      status: 200,
      headers: { "content-type": "application/json" },
    }));
    const fallbackHtml = vi.fn<typeof fetch>().mockResolvedValue(new Response("<html></html>", {
      status: 200,
      headers: { "content-type": "text/html" },
    }));

    await expect(checkServerConnectivity(true, healthy)).resolves.toBe("connected");
    await expect(checkServerConnectivity(true, fallbackHtml)).resolves.toBe("server-unreachable");
  });

  it("reports a reachable network with an unavailable server", async () => {
    const request = vi.fn<typeof fetch>().mockRejectedValue(new TypeError("network error"));
    await expect(checkServerConnectivity(true, request)).resolves.toBe("server-unreachable");
  });
});
