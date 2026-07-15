import { describe, expect, it } from "vitest";
import { defaultPreferences } from "./preferences";
import { preferenceCssVariables } from "./preferenceTheme";

describe("preference CSS variables", () => {
  it("applies the selected font only to reading when requested", () => {
    const variables = preferenceCssVariables({ ...defaultPreferences, fontFamily: "Atkinson Hyperlegible", fontScope: "reading" });
    expect(variables["--reading"]).toContain("Atkinson Hyperlegible");
    expect(variables["--sans"]).toContain("IBM Plex Sans Variable");
  });

  it("applies the selected font to interface portals through root variables", () => {
    const variables = preferenceCssVariables({ ...defaultPreferences, fontFamily: "Example UI", fontScope: "interface" });
    expect(variables["--sans"]).toContain("Example UI");
    expect(variables["--reading"]).toContain("IBM Plex Sans Variable");
  });

  it("applies one selection to the entire app and exposes reading styling", () => {
    const variables = preferenceCssVariables({ ...defaultPreferences, fontFamily: "Entire App", fontScope: "all", readingFontSize: 22, readingFontWeight: 600, readingFontStyle: "italic", readingTextColor: "#abcdef" });
    expect(variables["--sans"]).toContain("Entire App");
    expect(variables["--reading"]).toContain("Entire App");
    expect(variables).toMatchObject({ "--transcript-font-size": "22px", "--reading-font-weight": 600, "--reading-font-style": "italic", "--reading-color": "#abcdef" });
  });
});
