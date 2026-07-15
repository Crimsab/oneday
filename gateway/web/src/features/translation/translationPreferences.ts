const READING_LANGUAGE_KEY = "oneday-reading-languages-v1";
const RECENT_LANGUAGE_KEY = "oneday-recent-translation-languages-v1";

export function readingLanguageForStory(storyId: string): string {
  if (!storyId) return "";
  try {
    const value = JSON.parse(localStorage.getItem(READING_LANGUAGE_KEY) || "{}") as Record<string, unknown>;
    return typeof value[storyId] === "string" ? value[storyId] : "";
  } catch {
    return "";
  }
}

export function setReadingLanguageForStory(storyId: string, language: string): void {
  if (!storyId) return;
  let value: Record<string, string> = {};
  try { value = JSON.parse(localStorage.getItem(READING_LANGUAGE_KEY) || "{}"); } catch { value = {}; }
  if (language) value[storyId] = language;
  else delete value[storyId];
  localStorage.setItem(READING_LANGUAGE_KEY, JSON.stringify(value));
}

export function recentTranslationLanguages(): string[] {
  try {
    const value = JSON.parse(localStorage.getItem(RECENT_LANGUAGE_KEY) || "[]");
    return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string").slice(0, 6) : [];
  } catch {
    return [];
  }
}

export function rememberTranslationLanguage(language: string): string[] {
  const next = [language, ...recentTranslationLanguages().filter((item) => item !== language)].slice(0, 6);
  localStorage.setItem(RECENT_LANGUAGE_KEY, JSON.stringify(next));
  return next;
}
