export type TranslationAvailability = "unavailable" | "downloadable" | "downloading" | "available";

interface DownloadProgressEvent extends Event { loaded: number }
interface DownloadMonitor extends EventTarget {}
interface TranslatorInstance {
  translate(text: string, options?: { signal?: AbortSignal }): Promise<string>;
  destroy?: () => void;
}
interface LanguageDetectorInstance {
  detect(text: string): Promise<Array<{ detectedLanguage: string; confidence: number }>>;
  destroy?: () => void;
}
interface TranslatorFactory {
  availability(options: { sourceLanguage: string; targetLanguage: string }): Promise<TranslationAvailability>;
  create(options: { sourceLanguage: string; targetLanguage: string; signal?: AbortSignal; monitor?: (monitor: DownloadMonitor) => void }): Promise<TranslatorInstance>;
}
interface LanguageDetectorFactory {
  availability(): Promise<TranslationAvailability>;
  create(options?: { signal?: AbortSignal; monitor?: (monitor: DownloadMonitor) => void }): Promise<LanguageDetectorInstance>;
}

type BuiltInAiGlobal = typeof globalThis & {
  Translator?: TranslatorFactory;
  LanguageDetector?: LanguageDetectorFactory;
};

const cache = new Map<string, string>();

export function supportsBrowserTranslation(): boolean {
  return typeof globalThis !== "undefined" && Boolean((globalThis as BuiltInAiGlobal).Translator);
}

export async function browserTranslationAvailability(sourceLanguage: string, targetLanguage: string): Promise<TranslationAvailability> {
  const source = primaryLanguage(sourceLanguage);
  const target = primaryLanguage(targetLanguage);
  if (!source || !target) return "unavailable";
  if (source === target) return "available";
  const factory = (globalThis as BuiltInAiGlobal).Translator;
  if (!factory) return "unavailable";
  try {
    return await factory.availability({ sourceLanguage: source, targetLanguage: target });
  } catch {
    return "unavailable";
  }
}

export async function translateInBrowser({
  text,
  sourceLanguage,
  targetLanguage,
  signal,
  onDownloadProgress,
  allowDownload = true,
}: {
  text: string;
  sourceLanguage: string;
  targetLanguage: string;
  signal?: AbortSignal;
  onDownloadProgress?: (progress: number) => void;
  allowDownload?: boolean;
}): Promise<string> {
  const source = primaryLanguage(sourceLanguage);
  const target = primaryLanguage(targetLanguage);
  if (!text.trim() || source === target) return text;
  const key = `${source}:${target}:${text}`;
  const cached = cache.get(key);
  if (cached !== undefined) return cached;

  const factory = (globalThis as BuiltInAiGlobal).Translator;
  if (!factory) throw new Error("browser_translation_unavailable");
  const availability = await factory.availability({ sourceLanguage: source, targetLanguage: target });
  if (availability === "unavailable") throw new Error("language_pair_unavailable");
  if (!allowDownload && availability !== "available") throw new Error("language_pack_needs_user_action");
  const translator = await factory.create({
    sourceLanguage: source,
    targetLanguage: target,
    signal,
    monitor: onDownloadProgress ? (monitor) => monitor.addEventListener("downloadprogress", (event) => {
      onDownloadProgress(Math.max(0, Math.min(1, (event as DownloadProgressEvent).loaded)));
    }) : undefined,
  });
  try {
    const translated = await translator.translate(text, { signal });
    cache.set(key, translated);
    return translated;
  } finally {
    translator.destroy?.();
  }
}

export async function prepareBrowserTranslator(sourceLanguage: string, targetLanguage: string, signal?: AbortSignal): Promise<void> {
  const source = primaryLanguage(sourceLanguage);
  const target = primaryLanguage(targetLanguage);
  if (source === target) return;
  const factory = (globalThis as BuiltInAiGlobal).Translator;
  if (!factory) throw new Error("browser_translation_unavailable");
  const availability = await factory.availability({ sourceLanguage: source, targetLanguage: target });
  if (availability === "unavailable") throw new Error("language_pair_unavailable");
  const translator = await factory.create({ sourceLanguage: source, targetLanguage: target, signal });
  translator.destroy?.();
}

export async function detectLanguageInBrowser(text: string, signal?: AbortSignal): Promise<string | null> {
  const factory = (globalThis as BuiltInAiGlobal).LanguageDetector;
  if (!factory || !text.trim()) return null;
  const availability = await factory.availability();
  if (availability === "unavailable") return null;
  const detector = await factory.create({ signal });
  try {
    return primaryLanguage((await detector.detect(text))[0]?.detectedLanguage || "") || null;
  } finally {
    detector.destroy?.();
  }
}

export function primaryLanguage(value: string): string {
  const normalized = value.trim().toLowerCase().replace(/_/g, "-");
  const aliases: Record<string, string> = {
    english: "en", inglese: "en", italian: "it", italiano: "it",
    french: "fr", francese: "fr", german: "de", tedesco: "de",
    spanish: "es", spagnolo: "es", portuguese: "pt", portoghese: "pt",
    japanese: "ja", giapponese: "ja", korean: "ko", coreano: "ko", chinese: "zh", cinese: "zh",
  };
  const candidate = aliases[normalized] || normalized;
  if (!candidate) return "";
  try {
    return Intl.getCanonicalLocales(candidate)[0]?.split("-")[0]?.toLowerCase() || "";
  } catch {
    return "";
  }
}

export function clearTranslationCache(): void {
  cache.clear();
}
