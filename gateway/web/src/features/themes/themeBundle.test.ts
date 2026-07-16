import { describe, expect, it } from "vitest";
import { strToU8, zipSync } from "fflate";
import { defaultPreferences } from "../../preferences";
import type { StoredFontRecord } from "../../fontLibrary";
import { exportThemeBundle, previewThemeBundle } from "./themeBundle";

describe("portable theme bundles", () => {
  it("round-trips an opted-in WOFF2 font under a fresh local identity", async () => {
    const bytes = new Uint8Array([0x77, 0x4f, 0x46, 0x32, 0, 1, 2, 3]);
    const font: StoredFontRecord = { id: "imported:source", family: "Source", label: "Source font", source: "imported", fileName: "source.woff2", mimeType: "font/woff2", createdAt: new Date(0).toISOString(), data: new Blob([bytes], { type: "font/woff2" }) };
    const preferences = { ...defaultPreferences, readingFontId: font.id, readingFontFamily: font.family, readingFontSource: "imported" as const };
    const blob = await exportThemeBundle(preferences, [font], true);
    const preview = await previewThemeBundle(new File([blob], "theme.zip", { type: "application/zip" }), defaultPreferences, new Set());
    expect(preview.stagedFonts).toHaveLength(1);
    expect(preview.stagedFonts[0].id).not.toBe(font.id);
    expect(preview.preferences.readingFontId).toBe(preview.stagedFonts[0].id);
    expect(preview.missingFontIds).toEqual([]);
  });

  it("rejects traversal entries before extraction", async () => {
    const blob = new Blob([zipSync({ "../theme.json": strToU8("{}") })], { type: "application/zip" });
    await expect(previewThemeBundle(new File([blob], "bad.zip"), defaultPreferences, new Set())).rejects.toThrow("invalid_theme_bundle");
  });
});
