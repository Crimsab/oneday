import { describe, expect, it } from "vitest";
import { appRoutePath, historyReturnRoute, parseAppRoute, resolveAppRoute, sameAppRoute } from "./appRoute";
import type { StorySummary } from "./types";

const stories = [
  { id: "story one", name: "One", is_archived: false },
  { id: "archived", name: "Old", is_archived: true },
] as StorySummary[];

describe("app routes", () => {
  it("parses the library and every canonical story section", () => {
    expect(parseAppRoute("/stories")).toEqual({ kind: "library" });
    for (const section of ["history", "map", "codex", "inventory", "stats", "craft", "fronts", "investigations", "projects", "achievements", "saves", "translations"]) {
      expect(parseAppRoute(`/stories/story%20one/${section}`)).toEqual({ kind: "story", storyId: "story one", section });
    }
  });

  it("rejects noncanonical and malformed paths while preserving encoded story ids", () => {
    expect(parseAppRoute("/")).toBeNull();
    expect(parseAppRoute("/stories/story-one/unknown")).toBeNull();
    expect(parseAppRoute("/stories/story%2Fone/map")).toEqual({ kind: "story", storyId: "story/one", section: "map" });
    expect(parseAppRoute("/stories/%E0%A4%A/map")).toBeNull();
  });

  it("serializes stable story ids and recognizes identical routes", () => {
    const route = { kind: "story", storyId: "story one", section: "map" } as const;
    expect(appRoutePath(route)).toBe("/stories/story%20one/map");
    expect(sameAppRoute(route, parseAppRoute(appRoutePath(route))!)).toBe(true);
    expect(sameAppRoute(route, { ...route, section: "codex" })).toBe(false);
  });

  it("falls back from missing or archived stories to the first active story", () => {
    expect(resolveAppRoute({ kind: "story", storyId: "archived", section: "map" }, stories)).toEqual({ kind: "story", storyId: "story one", section: "history" });
    expect(resolveAppRoute({ kind: "story", storyId: "missing", section: "map" }, stories)).toEqual({ kind: "story", storyId: "story one", section: "history" });
    expect(resolveAppRoute(null, [])).toEqual({ kind: "library" });
  });

  it("round-trips modal return routes through history state", () => {
    expect(historyReturnRoute({ oneday: true, returnTo: "/stories/story%20one/map" })).toEqual({ kind: "story", storyId: "story one", section: "map" });
    expect(historyReturnRoute({ oneday: true, returnTo: "/invalid" })).toBeNull();
    expect(historyReturnRoute({ returnTo: "/stories/story%20one/map" })).toBeNull();
  });
});
