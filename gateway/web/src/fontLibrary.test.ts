import { describe, expect, it } from "vitest";
import { dedupeSystemFonts, fontFormatFromBytes, fontNameFromFile, fontNameFromUrl, isSupportedFontFile, normalizeOnlineFontUrl } from "./fontLibrary";

describe("font library helpers", () => {
  it("derives a readable name from common font files", () => {
    expect(fontNameFromFile("Atkinson-Hyperlegible_Bold.woff2")).toBe("Atkinson Hyperlegible Bold");
    expect(fontNameFromFile(".woff")).toBe("Font importato");
  });

  it("accepts supported font containers and rejects unrelated files", () => {
    expect(isSupportedFontFile({ name: "reader.otf", type: "font/otf" })).toBe(true);
    expect(isSupportedFontFile({ name: "reader.WOFF2", type: "" })).toBe(true);
    expect(isSupportedFontFile({ name: "cover.png", type: "image/png" })).toBe(false);
  });

  it("deduplicates local faces into searchable font families", () => {
    const choices = dedupeSystemFonts([
      { family: "Example Sans", fullName: "Example Sans Regular", postscriptName: "ExampleSans-Regular", style: "Regular" },
      { family: "Example Sans", fullName: "Example Sans Bold", postscriptName: "ExampleSans-Bold", style: "Bold" },
      { family: "Alpha Serif", fullName: "Alpha Serif", postscriptName: "AlphaSerif", style: "Regular" },
    ]);
    expect(choices.map((choice) => choice.label)).toEqual(["Alpha Serif", "Example Sans"]);
  });

  it("normalizes online font links and derives their labels", () => {
    expect(normalizeOnlineFontUrl(" https://fonts.example/Atkinson-Hyperlegible.woff2 ")).toBe("https://fonts.example/Atkinson-Hyperlegible.woff2");
    expect(fontNameFromUrl("https://fonts.example/Atkinson-Hyperlegible.woff2?v=2")).toBe("Atkinson Hyperlegible");
    expect(() => normalizeOnlineFontUrl("file:///tmp/font.woff2")).toThrow();
    expect(() => normalizeOnlineFontUrl("https://user:password@fonts.example/font.woff2")).toThrow();
  });

  it("detects real font containers from their binary signature", () => {
    expect(fontFormatFromBytes(new Uint8Array([0x77, 0x4f, 0x46, 0x32]))).toBe("woff2");
    expect(fontFormatFromBytes(new Uint8Array([0x4f, 0x54, 0x54, 0x4f]))).toBe("otf");
    expect(fontFormatFromBytes(new Uint8Array([0x89, 0x50, 0x4e, 0x47]))).toBeNull();
  });
});
