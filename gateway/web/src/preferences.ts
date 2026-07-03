import type { AppPreferences } from "./types";

const STORAGE_KEY = "oneday-browser-preferences-v1";

export const defaultPreferences: AppPreferences = {
  density: "balanced",
  fontSize: "base",
  accent: "amber",
  showLeftRail: true,
  showInspector: true,
  wrapTranscript: true,
  narrativeModel: "config",
  utilityModel: "config",
  repairModel: "config",
  imageModel: "config",
  openAIEndpoint: "",
  ttsVoice: "",
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
    narrativeModel: stringPref(value?.narrativeModel, defaultPreferences.narrativeModel),
    utilityModel: stringPref(value?.utilityModel, defaultPreferences.utilityModel),
    repairModel: stringPref(value?.repairModel, defaultPreferences.repairModel),
    imageModel: stringPref(value?.imageModel, defaultPreferences.imageModel),
    openAIEndpoint: stringPref(value?.openAIEndpoint, defaultPreferences.openAIEndpoint),
    ttsVoice: stringPref(value?.ttsVoice, defaultPreferences.ttsVoice),
  };
}

function oneOf<T extends string>(value: unknown, options: readonly T[], fallback: T): T {
  return typeof value === "string" && (options as readonly string[]).includes(value) ? (value as T) : fallback;
}

function stringPref(value: unknown, fallback: string): string {
  return typeof value === "string" ? value : fallback;
}
