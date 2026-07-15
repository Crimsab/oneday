import { defaultPreferences, normalizePreferences } from "../../preferences";
import type { AppPreferences } from "../../types";

export const THEME_KIND = "oneday-theme" as const;
export const THEME_VERSION = 1 as const;

export interface PortableThemeV1 {
  kind: typeof THEME_KIND;
  version: typeof THEME_VERSION;
  meta: { name: string; createdAt: string; generator: "OneDay" };
  tokens: {
    colors: { accent: string; readingText: string };
    typography: {
      interfaceFont: ThemeFontReference;
      readingFont: ThemeFontReference;
      interfaceScale: number;
      readingSize: number;
      readingWeight: number;
      readingStyle: AppPreferences["readingFontStyle"];
    };
    density: AppPreferences["density"];
  };
  fontRefs: string[];
}

export interface ThemeFontReference {
  id: string;
  family: string;
  source: AppPreferences["interfaceFontSource"];
}

export interface ThemePreview {
  theme: PortableThemeV1;
  preferences: AppPreferences;
  changes: Array<{ key: string; before: string; after: string }>;
  missingFontIds: string[];
}

export function exportTheme(preferences: AppPreferences, name = "OneDay theme"): PortableThemeV1 {
  const interfaceFont = fontReference(preferences.interfaceFontId, preferences.interfaceFontFamily, preferences.interfaceFontSource);
  const readingFont = fontReference(preferences.readingFontId, preferences.readingFontFamily, preferences.readingFontSource);
  return {
    kind: THEME_KIND,
    version: THEME_VERSION,
    meta: { name: normalizeName(name), createdAt: new Date().toISOString(), generator: "OneDay" },
    tokens: {
      colors: { accent: preferences.accent, readingText: preferences.readingTextColor },
      typography: {
        interfaceFont,
        readingFont,
        interfaceScale: preferences.interfaceFontScale,
        readingSize: preferences.readingFontSize,
        readingWeight: preferences.readingFontWeight,
        readingStyle: preferences.readingFontStyle,
      },
      density: preferences.density,
    },
    fontRefs: [...new Set([interfaceFont, readingFont].filter((font) => font.source === "imported" || font.source === "online").map((font) => font.id))],
  };
}

export function previewThemeImport(value: unknown, current: AppPreferences, availableFontIds: ReadonlySet<string>): ThemePreview {
  const theme = decodeTheme(value);
  const missingFontIds = theme.fontRefs.filter((id) => !availableFontIds.has(id));
  const interfaceFont = missingFontIds.includes(theme.tokens.typography.interfaceFont.id) ? fallbackFont() : theme.tokens.typography.interfaceFont;
  const readingFont = missingFontIds.includes(theme.tokens.typography.readingFont.id) ? fallbackFont() : theme.tokens.typography.readingFont;
  const preferences = normalizePreferences({
    ...current,
    accent: theme.tokens.colors.accent,
    readingTextColor: theme.tokens.colors.readingText,
    density: theme.tokens.density,
    interfaceFontId: interfaceFont.id,
    interfaceFontFamily: interfaceFont.family,
    interfaceFontSource: interfaceFont.source,
    readingFontId: readingFont.id,
    readingFontFamily: readingFont.family,
    readingFontSource: readingFont.source,
    interfaceFontScale: theme.tokens.typography.interfaceScale,
    readingFontSize: theme.tokens.typography.readingSize,
    readingFontWeight: theme.tokens.typography.readingWeight,
    readingFontStyle: theme.tokens.typography.readingStyle,
  });
  return { theme, preferences, missingFontIds, changes: themeChanges(current, preferences) };
}

export function decodeTheme(value: unknown): PortableThemeV1 {
  if (!isRecord(value) || value.kind !== THEME_KIND || value.version !== THEME_VERSION) throw new Error("invalid_theme_format");
  if (!isRecord(value.meta) || !isRecord(value.tokens) || !isRecord(value.tokens.colors) || !isRecord(value.tokens.typography)) throw new Error("invalid_theme_format");
  const typography = value.tokens.typography;
  const interfaceFont = decodeFont(typography.interfaceFont);
  const readingFont = decodeFont(typography.readingFont);
  const density = oneOf(value.tokens.density, ["compact", "balanced", "comfortable"] as const);
  const readingStyle = oneOf(typography.readingStyle, ["normal", "italic"] as const);
  const readingWeight = oneOfNumber(typography.readingWeight, [300, 400, 500, 600, 700]);
  const theme: PortableThemeV1 = {
    kind: THEME_KIND,
    version: THEME_VERSION,
    meta: { name: normalizeName(value.meta.name), createdAt: text(value.meta.createdAt, 80), generator: "OneDay" },
    tokens: {
      colors: { accent: color(value.tokens.colors.accent), readingText: color(value.tokens.colors.readingText) },
      typography: {
        interfaceFont,
        readingFont,
        interfaceScale: integer(typography.interfaceScale, 80, 130),
        readingSize: integer(typography.readingSize, 13, 26),
        readingWeight,
        readingStyle,
      },
      density,
    },
    fontRefs: Array.isArray(value.fontRefs) ? [...new Set(value.fontRefs.map((item) => text(item, 180)))].slice(0, 8) : [],
  };
  return theme;
}

function themeChanges(before: AppPreferences, after: AppPreferences): ThemePreview["changes"] {
  const keys: Array<keyof AppPreferences> = ["accent", "readingTextColor", "density", "interfaceFontFamily", "interfaceFontScale", "readingFontFamily", "readingFontSize", "readingFontWeight", "readingFontStyle"];
  return keys.flatMap((key) => before[key] === after[key] ? [] : [{ key: String(key), before: String(before[key]), after: String(after[key]) }]);
}

function fallbackFont(): ThemeFontReference {
  return fontReference(defaultPreferences.interfaceFontId, defaultPreferences.interfaceFontFamily, defaultPreferences.interfaceFontSource);
}

function fontReference(id: string, family: string, source: ThemeFontReference["source"]): ThemeFontReference {
  return { id: text(id, 180), family: text(family, 160), source };
}

function decodeFont(value: unknown): ThemeFontReference {
  if (!isRecord(value)) throw new Error("invalid_theme_font");
  return fontReference(text(value.id, 180), text(value.family, 160), oneOf(value.source, ["bundled", "system", "imported", "online"] as const));
}

function isRecord(value: unknown): value is Record<string, any> { return Boolean(value) && typeof value === "object" && !Array.isArray(value); }
function normalizeName(value: unknown): string { return text(value, 80) || "OneDay theme"; }
function text(value: unknown, max: number): string { if (typeof value !== "string") throw new Error("invalid_theme_value"); const result = value.trim(); if (!result || result.length > max) throw new Error("invalid_theme_value"); return result; }
function color(value: unknown): string { const result = text(value, 7).toLowerCase(); if (!/^#[0-9a-f]{6}$/.test(result)) throw new Error("invalid_theme_color"); return result; }
function integer(value: unknown, min: number, max: number): number { if (typeof value !== "number" || !Number.isInteger(value) || value < min || value > max) throw new Error("invalid_theme_number"); return value; }
function oneOf<T extends string>(value: unknown, options: readonly T[]): T { if (typeof value !== "string" || !options.includes(value as T)) throw new Error("invalid_theme_value"); return value as T; }
function oneOfNumber<T extends number>(value: unknown, options: readonly T[]): T { if (typeof value !== "number" || !options.includes(value as T)) throw new Error("invalid_theme_value"); return value as T; }
