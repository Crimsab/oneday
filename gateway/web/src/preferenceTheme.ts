import { cssFontFamily } from "./fontLibrary";
import { DEFAULT_FONT_FAMILY } from "./preferences";
import type { AppPreferences } from "./types";

export type PreferenceCssVariables = Record<`--${string}`, string | number>;

export function preferenceCssVariables(preferences: AppPreferences): PreferenceCssVariables {
  const selectedFont = cssFontFamily(preferences.fontFamily);
  const defaultFont = cssFontFamily(DEFAULT_FONT_FAMILY);
  return {
    "--accent": preferences.accent,
    "--sans": preferences.fontScope === "interface" || preferences.fontScope === "all" ? selectedFont : defaultFont,
    "--reading": preferences.fontScope === "reading" || preferences.fontScope === "all" ? selectedFont : defaultFont,
    "--transcript-font-size": `${preferences.readingFontSize}px`,
    "--reading-font-weight": preferences.readingFontWeight,
    "--reading-font-style": preferences.readingFontStyle,
    "--reading-color": preferences.readingTextColor,
  };
}
