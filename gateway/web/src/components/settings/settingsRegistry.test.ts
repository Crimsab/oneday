import { describe, expect, it } from "vitest";
import { searchSettings, settingsCategories, settingsNavigationGroups, settingsSearchEntries } from "./settingsRegistry";

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

  it("groups each settings category exactly once", () => {
    const grouped = settingsNavigationGroups.flatMap((group) => group.sections);
    expect(grouped).toHaveLength(settingsCategories.length);
    expect(new Set(grouped)).toEqual(new Set(settingsCategories.map((category) => category.id)));
  });
});
