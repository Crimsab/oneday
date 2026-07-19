import { describe, expect, it } from "vitest";
import { settings_search as englishSearch } from "../../locales/en/settings_search";
import { settings_search as italianSearch } from "../../locales/it/settings_search";
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

  it("keeps player preferences and protected operator configuration in distinct sections", () => {
    const playerSections = new Set(settingsCategories.filter((item) => item.scope === "player").map((item) => item.id));
    const operatorSections = new Set(settingsCategories.filter((item) => item.scope === "operator").map((item) => item.id));
    expect(playerSections).toContain("preferences");
    expect(playerSections).not.toContain("operator");
    expect(operatorSections).toEqual(new Set(["operator"]));
    expect(searchSettings("api key").every((item) => item.section === "operator")).toBe(true);
    expect(searchSettings("reset browser preferences").every((item) => item.section === "preferences")).toBe(true);
  });

  it("keeps renamed player and operator search results localized in English and Italian", () => {
    expect(englishSearch["generation-diagnostics"][0]).toBe("Message diagnostics");
    expect(italianSearch["generation-diagnostics"][0]).toBe("Diagnostica messaggi");
    for (const catalog of [englishSearch, italianSearch]) {
      expect(catalog["theme-portability"]).toHaveLength(2);
      expect(catalog["support-bundle"]).toHaveLength(2);
    }
  });

  it("groups each settings category exactly once", () => {
    const grouped = settingsNavigationGroups.flatMap((group) => group.sections);
    expect(grouped).toHaveLength(settingsCategories.length);
    expect(new Set(grouped)).toEqual(new Set(settingsCategories.map((category) => category.id)));
  });
});
