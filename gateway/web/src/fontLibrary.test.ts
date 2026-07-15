import { describe, expect, it } from "vitest";
import { dedupeSystemFonts, fontNameFromFile, isSupportedFontFile } from "./fontLibrary";

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
});
