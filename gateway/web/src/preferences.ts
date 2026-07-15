import type { AppPreferences, FontSourcePreference, MiniGameKind } from "./types";

const STORAGE_KEY = "oneday-browser-preferences-v2";
export const DEFAULT_ACCENT = "#d09a48";
export const DEFAULT_READING_COLOR = "#dfe5e8";
export const DEFAULT_FONT_FAMILY = "IBM Plex Sans Variable";
export const DEFAULT_FONT_ID = "bundled:ibm-plex-sans";
export const DEFAULT_INTERFACE_FONT_SCALE = 100;

export const AUTOMATIC_MINI_GAME_KINDS = [
  "deduction",
  "negotiation",
  "pattern",
  "bidding",
  "courtroom",
  "comedy",
  "quicktime",
] as const satisfies readonly MiniGameKind[];

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
  accent: DEFAULT_ACCENT,
  accentHistory: [],
  interfaceFontId: DEFAULT_FONT_ID,
  interfaceFontFamily: DEFAULT_FONT_FAMILY,
  interfaceFontSource: "bundled",
  interfaceFontScale: DEFAULT_INTERFACE_FONT_SCALE,
  readingFontId: DEFAULT_FONT_ID,
  readingFontFamily: DEFAULT_FONT_FAMILY,
  readingFontSource: "bundled",
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
  disabledMiniGames: [],
  showGenerationDiagnostics: false,
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

type LegacyPreferences = Partial<AppPreferences> & {
  fontSize?: "small" | "base" | "large";
  fontId?: string;
  fontFamily?: string;
  fontSource?: FontSourcePreference;
  fontScope?: "reading" | "interface" | "all";
};

export function normalizePreferences(value: LegacyPreferences | null | undefined): AppPreferences {
  const legacyFont = {
    id: normalizeText(value?.fontId, DEFAULT_FONT_ID),
    family: normalizeText(value?.fontFamily, DEFAULT_FONT_FAMILY),
    source: oneOf(value?.fontSource, ["bundled", "system", "imported", "online"], "bundled" as const),
  };
  const legacyScope = oneOf(value?.fontScope, ["reading", "interface", "all"], "reading" as const);
  const interfaceFont = normalizedFontAssignment(value?.interfaceFontId, value?.interfaceFontFamily, value?.interfaceFontSource,
    legacyScope === "interface" || legacyScope === "all" ? legacyFont : null);
  const readingFont = normalizedFontAssignment(value?.readingFontId, value?.readingFontFamily, value?.readingFontSource,
    legacyScope === "reading" || legacyScope === "all" ? legacyFont : null);
  const timingFreeChallenges = typeof value?.timingFreeChallenges === "boolean" ? value.timingFreeChallenges : defaultPreferences.timingFreeChallenges;
  const disabledMiniGames = normalizeMiniGameKinds(value?.disabledMiniGames, timingFreeChallenges);
  return {
    locale: resolveLocale(value?.locale, typeof navigator === "undefined" ? [] : navigator.languages),
    density: oneOf(value?.density, ["compact", "balanced", "comfortable"], defaultPreferences.density),
    accent: normalizeColor(value?.accent, defaultPreferences.accent),
    accentHistory: normalizeColorHistory(value?.accentHistory),
    interfaceFontId: interfaceFont.id,
    interfaceFontFamily: interfaceFont.family,
    interfaceFontSource: interfaceFont.source,
    interfaceFontScale: boundedNumber(value?.interfaceFontScale, 80, 130, legacyInterfaceScale(value?.fontSize)),
    readingFontId: readingFont.id,
    readingFontFamily: readingFont.family,
    readingFontSource: readingFont.source,
    readingFontSize: boundedNumber(value?.readingFontSize, 13, 26, defaultPreferences.readingFontSize),
    readingFontWeight: oneOfNumber(value?.readingFontWeight, [300, 400, 500, 600, 700], defaultPreferences.readingFontWeight),
    readingFontStyle: oneOf(value?.readingFontStyle, ["normal", "italic"], defaultPreferences.readingFontStyle),
    readingTextColor: normalizeColor(value?.readingTextColor, defaultPreferences.readingTextColor),
    showLeftRail: typeof value?.showLeftRail === "boolean" ? value.showLeftRail : defaultPreferences.showLeftRail,
    showInspector: typeof value?.showInspector === "boolean" ? value.showInspector : defaultPreferences.showInspector,
    wrapTranscript: typeof value?.wrapTranscript === "boolean" ? value.wrapTranscript : defaultPreferences.wrapTranscript,
    showChoiceDetails: typeof value?.showChoiceDetails === "boolean" ? value.showChoiceDetails : defaultPreferences.showChoiceDetails,
    automaticChallenges: typeof value?.automaticChallenges === "boolean" ? value.automaticChallenges : defaultPreferences.automaticChallenges,
    timingFreeChallenges,
    challengeCooldown: typeof value?.challengeCooldown === "boolean" ? value.challengeCooldown : defaultPreferences.challengeCooldown,
    disabledMiniGames,
    showGenerationDiagnostics: typeof value?.showGenerationDiagnostics === "boolean" ? value.showGenerationDiagnostics : defaultPreferences.showGenerationDiagnostics,
  };
}

export function resetTypographyPreferences(preferences: AppPreferences): AppPreferences {
  return {
    ...preferences,
    interfaceFontId: defaultPreferences.interfaceFontId,
    interfaceFontFamily: defaultPreferences.interfaceFontFamily,
    interfaceFontSource: defaultPreferences.interfaceFontSource,
    interfaceFontScale: defaultPreferences.interfaceFontScale,
    readingFontId: defaultPreferences.readingFontId,
    readingFontFamily: defaultPreferences.readingFontFamily,
    readingFontSource: defaultPreferences.readingFontSource,
    readingFontSize: defaultPreferences.readingFontSize,
    readingFontWeight: defaultPreferences.readingFontWeight,
    readingFontStyle: defaultPreferences.readingFontStyle,
    readingTextColor: defaultPreferences.readingTextColor,
  };
}

function normalizedFontAssignment(
  id: unknown,
  family: unknown,
  source: unknown,
  legacy: { id: string; family: string; source: FontSourcePreference } | null,
) {
  return {
    id: normalizeText(id, legacy?.id ?? DEFAULT_FONT_ID),
    family: normalizeText(family, legacy?.family ?? DEFAULT_FONT_FAMILY),
    source: oneOf(source, ["bundled", "system", "imported", "online"], legacy?.source ?? "bundled"),
  };
}

function legacyInterfaceScale(value: LegacyPreferences["fontSize"]): number {
  if (value === "small") return 88;
  if (value === "large") return 113;
  return DEFAULT_INTERFACE_FONT_SCALE;
}

function normalizeMiniGameKinds(value: unknown, timingFreeOnly: boolean): MiniGameKind[] {
  if (!Array.isArray(value)) return [];
  const allowed = new Set<MiniGameKind>(AUTOMATIC_MINI_GAME_KINDS);
  const unique = [...new Set(value.filter((item): item is MiniGameKind => typeof item === "string" && allowed.has(item as MiniGameKind)))];
  const hasUsableFamily = AUTOMATIC_MINI_GAME_KINDS.some((kind) => !unique.includes(kind) && (!timingFreeOnly || kind !== "quicktime"));
  return hasUsableFamily ? unique : unique.filter((kind) => kind !== "deduction");
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
