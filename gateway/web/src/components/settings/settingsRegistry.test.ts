import { describe, expect, it } from "vitest";
import { searchSettings, settingsCategories, settingsSearchEntries } from "./settingsRegistry";

describe("settings registry", () => {
  it("indexes every entry under a real category", () => {
    const categories = new Set(settingsCategories.map((item) => item.id));
    expect(settingsSearchEntries.length).toBeGreaterThan(20);
    expect(settingsSearchEntries.every((item) => categories.has(item.section))).toBe(true);
  });

  it("finds labels and synonyms without exposing field values", () => {
    expect(searchSettings("npc voice").map((item) => item.id)).toContain("voice-assignments");
    expect(searchSettings("known location icons").map((item) => item.id)).toContain("map-art");
    expect(searchSettings("provider fallback").map((item) => item.id)).toContain("provider-order");
    expect(searchSettings("   ")).toEqual([]);
  });
});
