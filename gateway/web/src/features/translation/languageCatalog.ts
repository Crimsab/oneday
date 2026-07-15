const COMMON_LANGUAGE_CODES = ["en", "it", "fr", "de", "es", "pt", "nl", "pl", "ro", "sv", "da", "no", "fi", "cs", "el", "tr", "uk", "ru", "ar", "hi", "id", "vi", "th", "ja", "ko", "zh"] as const;

export interface LanguageOption { code: string; name: string }

export function languageCatalog(locale: string): LanguageOption[] {
  const names = new Intl.DisplayNames([locale], { type: "language" });
  return COMMON_LANGUAGE_CODES.map((code) => ({ code, name: names.of(code) || code.toUpperCase() }))
    .sort((left, right) => left.name.localeCompare(right.name, locale));
}
