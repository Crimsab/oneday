import { cssFontFamily } from "./fontLibrary";
import { DEFAULT_FONT_FAMILY } from "./preferences";
import type { AppPreferences } from "./types";

export type PreferenceCssVariables = Record<`--${string}`, string | number>;

export function preferenceCssVariables(preferences: AppPreferences): PreferenceCssVariables {
  return {
    "--accent": preferences.accent,
    "--sans": cssFontFamily(preferences.interfaceFontFamily || DEFAULT_FONT_FAMILY),
    "--reading": cssFontFamily(preferences.readingFontFamily || DEFAULT_FONT_FAMILY),
    "--ui-root-font-size": `${preferences.interfaceFontScale}%`,
    "--transcript-font-size": `${preferences.readingFontSize}px`,
    "--reading-font-weight": preferences.readingFontWeight,
    "--reading-font-style": preferences.readingFontStyle,
    "--reading-color": preferences.readingTextColor,
  };
}
