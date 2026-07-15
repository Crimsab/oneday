import type { AppPreferences } from "./types";

const STORAGE_KEY = "oneday-browser-preferences-v2";

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
  accent: "amber",
  showLeftRail: false,
  showInspector: false,
  wrapTranscript: true,
  showChoiceDetails: false,
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
    accent: oneOf(value?.accent, ["amber", "green", "blue", "rose"], defaultPreferences.accent),
    showLeftRail: typeof value?.showLeftRail === "boolean" ? value.showLeftRail : defaultPreferences.showLeftRail,
    showInspector: typeof value?.showInspector === "boolean" ? value.showInspector : defaultPreferences.showInspector,
    wrapTranscript: typeof value?.wrapTranscript === "boolean" ? value.wrapTranscript : defaultPreferences.wrapTranscript,
    showChoiceDetails: typeof value?.showChoiceDetails === "boolean" ? value.showChoiceDetails : defaultPreferences.showChoiceDetails,
  };
}

function oneOf<T extends string>(value: unknown, options: readonly T[], fallback: T): T {
  return typeof value === "string" && (options as readonly string[]).includes(value) ? (value as T) : fallback;
}
