import type { AppPreferences } from "./types";

const STORAGE_KEY = "oneday-browser-preferences-v2";

export const defaultPreferences: AppPreferences = {
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
    if (!raw) return defaultPreferences;
    return normalizePreferences(JSON.parse(raw));
  } catch {
    return defaultPreferences;
  }
}

export function savePreferences(preferences: AppPreferences) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences));
}

export function normalizePreferences(value: Partial<AppPreferences> | null | undefined): AppPreferences {
  return {
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
