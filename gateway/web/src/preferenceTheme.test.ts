import { describe, expect, it } from "vitest";
import { defaultPreferences } from "./preferences";
import { preferenceCssVariables } from "./preferenceTheme";

describe("preference CSS variables", () => {
  it("applies independent interface and reading fonts", () => {
    const variables = preferenceCssVariables({ ...defaultPreferences, interfaceFontFamily: "Example UI", readingFontFamily: "Atkinson Hyperlegible" });
    expect(variables["--reading"]).toContain("Atkinson Hyperlegible");
    expect(variables["--sans"]).toContain("Example UI");
  });

  it("exposes interface scaling through a root variable", () => {
    const variables = preferenceCssVariables({ ...defaultPreferences, interfaceFontScale: 118 });
    expect(variables["--ui-root-font-size"]).toBe("118%");
  });

  it("exposes reading styling without changing the interface font", () => {
    const variables = preferenceCssVariables({ ...defaultPreferences, readingFontFamily: "Story Font", readingFontSize: 22, readingFontWeight: 600, readingFontStyle: "italic", readingTextColor: "#abcdef" });
    expect(variables["--sans"]).toContain("IBM Plex Sans Variable");
    expect(variables["--reading"]).toContain("Story Font");
    expect(variables).toMatchObject({ "--transcript-font-size": "22px", "--reading-font-weight": 600, "--reading-font-style": "italic", "--reading-color": "#abcdef" });
  });
});
