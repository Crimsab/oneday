import { afterEach, describe, expect, it } from "vitest";
import { clearTranslationCache, primaryLanguage, supportsBrowserTranslation, translateInBrowser } from "./browserTranslator";

describe("browser translator", () => {
  afterEach(() => { delete (globalThis as { Translator?: unknown }).Translator; clearTranslationCache(); });

  it("normalizes BCP-47 language tags", () => {
    expect(primaryLanguage("IT-it")).toBe("it");
    expect(primaryLanguage("pt_BR")).toBe("pt");
  });

  it("stays hidden when the browser API is unavailable", () => {
    expect(supportsBrowserTranslation()).toBe(false);
  });

  it("reuses an in-memory translation", async () => {
    let calls = 0;
    (globalThis as { Translator?: unknown }).Translator = {
      availability: async () => "available",
      create: async () => ({ translate: async () => { calls += 1; return "Ciao"; } }),
    };
    await expect(translateInBrowser({ text: "Hello", sourceLanguage: "en", targetLanguage: "it" })).resolves.toBe("Ciao");
    await expect(translateInBrowser({ text: "Hello", sourceLanguage: "en", targetLanguage: "it" })).resolves.toBe("Ciao");
    expect(calls).toBe(1);
  });
});
