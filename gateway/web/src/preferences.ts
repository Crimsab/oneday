import type { AppPreferences } from "./types";

const STORAGE_KEY = "oneday-browser-preferences-v2";
export const DEFAULT_ACCENT = "#d09a48";
export const DEFAULT_READING_COLOR = "#dfe5e8";
export const DEFAULT_FONT_FAMILY = "IBM Plex Sans Variable";

const LEGACY_ACCENTS: Record<string, string> = {
  amber: DEFAULT_ACCENT,
  green: "#8ed979",
  blue: "#73c7ff",
  rose: "#ff91ad",
};

export type InterfaceLocale = "en" | "it";

export function normalizeLocale(value: unknown): InterfaceLocale | null {
  if (typeof value !== "string") return null;
  const primary = value.trim().toLowerCase().replace(/_/g, "-").split("-")[0];
  return primary === "en" || primary === "it" ? primary : null;
}

export function resolveLocale(saved: unknown, browserLocales: readonly string[] = []): InterfaceLocale {
  return normalizeLocale(saved) ?? browserLocales.map(normalizeLocale).find((value): value is InterfaceLocale => value !== null) ?? "en";
}

export const defaultPreferences: AppPreferences = {
  locale: "en",
  density: "balanced",
  fontSize: "base",
  accent: DEFAULT_ACCENT,
  accentHistory: [],
  fontId: "bundled:ibm-plex-sans",
  fontFamily: DEFAULT_FONT_FAMILY,
  fontSource: "bundled",
  fontScope: "reading",
  readingFontSize: 17,
  readingFontWeight: 400,
  readingFontStyle: "normal",
  readingTextColor: DEFAULT_READING_COLOR,
  showLeftRail: false,
  showInspector: false,
  wrapTranscript: true,
  showChoiceDetails: false,
  automaticChallenges: true,
  timingFreeChallenges: true,
  challengeCooldown: true,
};

export function loadPreferences(): AppPreferences {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return normalizePreferences(null);
    return normalizePreferences(JSON.parse(raw));
  } catch {
    return normalizePreferences(null);
  }
}

export function savePreferences(preferences: AppPreferences) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences));
}

export function normalizePreferences(value: Partial<AppPreferences> | null | undefined): AppPreferences {
  return {
    locale: resolveLocale(value?.locale, typeof navigator === "undefined" ? [] : navigator.languages),
    density: oneOf(value?.density, ["compact", "balanced", "comfortable"], defaultPreferences.density),
    fontSize: oneOf(value?.fontSize, ["small", "base", "large"], defaultPreferences.fontSize),
    accent: normalizeColor(value?.accent, defaultPreferences.accent),
    accentHistory: normalizeColorHistory(value?.accentHistory),
    fontId: normalizeText(value?.fontId, defaultPreferences.fontId),
    fontFamily: normalizeText(value?.fontFamily, defaultPreferences.fontFamily),
    fontSource: oneOf(value?.fontSource, ["bundled", "system", "imported"], defaultPreferences.fontSource),
    fontScope: oneOf(value?.fontScope, ["reading", "interface", "all"], defaultPreferences.fontScope),
    readingFontSize: boundedNumber(value?.readingFontSize, 13, 26, defaultPreferences.readingFontSize),
    readingFontWeight: oneOfNumber(value?.readingFontWeight, [300, 400, 500, 600, 700], defaultPreferences.readingFontWeight),
    readingFontStyle: oneOf(value?.readingFontStyle, ["normal", "italic"], defaultPreferences.readingFontStyle),
    readingTextColor: normalizeColor(value?.readingTextColor, defaultPreferences.readingTextColor),
    showLeftRail: typeof value?.showLeftRail === "boolean" ? value.showLeftRail : defaultPreferences.showLeftRail,
    showInspector: typeof value?.showInspector === "boolean" ? value.showInspector : defaultPreferences.showInspector,
    wrapTranscript: typeof value?.wrapTranscript === "boolean" ? value.wrapTranscript : defaultPreferences.wrapTranscript,
    showChoiceDetails: typeof value?.showChoiceDetails === "boolean" ? value.showChoiceDetails : defaultPreferences.showChoiceDetails,
    automaticChallenges: typeof value?.automaticChallenges === "boolean" ? value.automaticChallenges : defaultPreferences.automaticChallenges,
    timingFreeChallenges: typeof value?.timingFreeChallenges === "boolean" ? value.timingFreeChallenges : defaultPreferences.timingFreeChallenges,
    challengeCooldown: typeof value?.challengeCooldown === "boolean" ? value.challengeCooldown : defaultPreferences.challengeCooldown,
  };
}

function oneOf<T extends string>(value: unknown, options: readonly T[], fallback: T): T {
  return typeof value === "string" && (options as readonly string[]).includes(value) ? (value as T) : fallback;
}

function normalizeColor(value: unknown, fallback: string): string {
  if (typeof value !== "string") return fallback;
  const legacy = LEGACY_ACCENTS[value.toLowerCase()];
  if (legacy) return legacy;
  const normalized = value.trim().toLowerCase();
  if (/^#[0-9a-f]{3}$/.test(normalized)) {
    return `#${normalized.slice(1).split("").map((digit) => digit + digit).join("")}`;
  }
  return /^#[0-9a-f]{6}$/.test(normalized) ? normalized : fallback;
}

function normalizeColorHistory(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  const colors = value
    .map((item) => normalizeColor(item, ""))
    .filter((item): item is string => Boolean(item));
  return [...new Set(colors)].slice(0, 10);
}

function normalizeText(value: unknown, fallback: string): string {
  if (typeof value !== "string") return fallback;
  const normalized = value.trim();
  return normalized && normalized.length <= 160 ? normalized : fallback;
}

function boundedNumber(value: unknown, min: number, max: number, fallback: number): number {
  if (typeof value !== "number" || !Number.isFinite(value) || value < min || value > max) return fallback;
  return Math.round(value);
}

function oneOfNumber(value: unknown, options: readonly number[], fallback: number): number {
  return typeof value === "number" && options.includes(value) ? value : fallback;
}
