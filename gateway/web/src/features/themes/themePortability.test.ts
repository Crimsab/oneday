import { describe, expect, it } from "vitest";
import { defaultPreferences } from "../../preferences";
import { decodeTheme, exportTheme, previewThemeImport } from "./themePortability";

describe("portable themes", () => {
  it("round-trips only visual preferences", () => {
    const preferences = { ...defaultPreferences, locale: "it" as const, accent: "#123456", automaticChallenges: false, readingFontSize: 21 };
    const theme = exportTheme(preferences, "Midnight Paper");
    expect(theme).not.toHaveProperty("preferences");
    expect(JSON.stringify(theme)).not.toContain("automaticChallenges");
    const preview = previewThemeImport(theme, defaultPreferences, new Set());
    expect(preview.preferences).toMatchObject({ accent: "#123456", readingFontSize: 21, locale: defaultPreferences.locale, automaticChallenges: defaultPreferences.automaticChallenges });
  });

  it("rejects unknown formats and invalid colors", () => {
    expect(() => decodeTheme({ kind: "preferences", version: 1 })).toThrow("invalid_theme_format");
    const theme = exportTheme(defaultPreferences);
    theme.tokens.colors.accent = "red";
    expect(() => decodeTheme(theme)).toThrow("invalid_theme_color");
  });

  it("falls back when a referenced font is missing", () => {
    const theme = exportTheme({ ...defaultPreferences, readingFontId: "imported:missing", readingFontFamily: "Missing", readingFontSource: "imported" });
    const preview = previewThemeImport(theme, defaultPreferences, new Set());
    expect(preview.missingFontIds).toEqual(["imported:missing"]);
    expect(preview.preferences.readingFontId).toBe(defaultPreferences.interfaceFontId);
  });
});
