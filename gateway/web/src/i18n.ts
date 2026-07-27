import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import { en } from "./locales/en";
import { it } from "./locales/it";
import { loadPreferences, normalizeLocale, type InterfaceLocale } from "./preferences";

export const resources = { en, it } as const;
export const namespaces = Object.keys(en) as Array<keyof typeof en>;
const initialLocale = typeof window === "undefined" ? "en" : loadPreferences().locale;

void i18n.use(initReactI18next).init({ resources, lng: initialLocale, fallbackLng: "en", supportedLngs: ["en", "it"], defaultNS: "common", ns: namespaces, fallbackNS: "common", interpolation: { escapeValue: false }, returnNull: false, parseMissingKeyHandler: (key) => humanizeMissingKey(key) });

export function humanizeMissingKey(key: string): string {
  const leaf = key.split(/[.:]/).at(-1)?.trim() ?? "";
  const words = leaf
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/[_-]+/g, " ")
    .trim();
  if (!words) return en.common.missing;
  return words.charAt(0).toLocaleUpperCase("en") + words.slice(1).toLocaleLowerCase("en");
}

export function formatInterfaceNumber(value: number): string {
  return new Intl.NumberFormat(i18n.resolvedLanguage ?? "en").format(value);
}

export function formatInterfaceDateTime(value: Date): string {
  const timeFormat = typeof window === "undefined" ? "system" : loadPreferences().timeFormat;
  return new Intl.DateTimeFormat(i18n.resolvedLanguage ?? "en", {
    dateStyle: "medium",
    timeStyle: "short",
    ...(timeFormat === "system" ? {} : { hour12: timeFormat === "12" }),
  }).format(value);
}

export async function setInterfaceLocale(value: string): Promise<InterfaceLocale> {
  const locale = normalizeLocale(value) ?? "en";
  await i18n.changeLanguage(locale);
  if (typeof document !== "undefined") document.documentElement.lang = locale;
  return locale;
}

if (typeof document !== "undefined") document.documentElement.lang = initialLocale;
export default i18n;
