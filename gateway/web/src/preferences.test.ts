import { afterEach, describe, expect, it } from "vitest";
import { defaultPreferences, loadPreferences, normalizeLocale, normalizePreferences, resolveLocale, savePreferences } from "./preferences";

const originalLocalStorage = Object.getOwnPropertyDescriptor(globalThis, "localStorage");

describe("normalizePreferences", () => {
  it("keeps valid preferences and falls back for invalid values", () => {
    expect(
      normalizePreferences({
        locale: "it",
        density: "compact",
        fontSize: "large",
        accent: "#73c7ff",
        accentHistory: ["#112233", "bad", "#abc"],
        fontId: "system:atkinson",
        fontFamily: "Atkinson Hyperlegible",
        fontSource: "system",
        fontScope: "all",
        readingFontSize: 21,
        readingFontWeight: 600,
        readingFontStyle: "italic",
        readingTextColor: "#eeeeee",
        showLeftRail: false,
        showInspector: false,
        wrapTranscript: false,
        showChoiceDetails: true,
        automaticChallenges: false,
        timingFreeChallenges: false,
        challengeCooldown: false,
      }),
    ).toEqual({
      locale: "it",
      density: "compact",
      fontSize: "large",
      accent: "#73c7ff",
      accentHistory: ["#112233", "#aabbcc"],
      fontId: "system:atkinson",
      fontFamily: "Atkinson Hyperlegible",
      fontSource: "system",
      fontScope: "all",
      readingFontSize: 21,
      readingFontWeight: 600,
      readingFontStyle: "italic",
      readingTextColor: "#eeeeee",
      showLeftRail: false,
      showInspector: false,
      wrapTranscript: false,
      showChoiceDetails: true,
      automaticChallenges: false,
      timingFreeChallenges: false,
      challengeCooldown: false,
    });

    expect(
      normalizePreferences({
        density: "wide" as never,
        fontSize: "huge" as never,
        accent: "orange" as never,
        showLeftRail: "yes" as never,
        showInspector: "yes" as never,
        wrapTranscript: "no" as never,
        showChoiceDetails: "yes" as never,
        readingFontSize: 99,
        readingFontWeight: 450,
        readingTextColor: "tomato",
      }),
    ).toEqual(defaultPreferences);
  });
});

describe("interface locale resolution", () => {
  it("normalizes supported regional variants", () => {
    expect(normalizeLocale("it-IT")).toBe("it");
    expect(normalizeLocale("en_US")).toBe("en");
    expect(normalizeLocale("fr-FR")).toBeNull();
  });

  it("prefers a saved locale, then browser order, then English", () => {
    expect(resolveLocale("en-US", ["it-IT"])).toBe("en");
    expect(resolveLocale(undefined, ["fr-FR", "it-CH"])).toBe("it");
    expect(resolveLocale(undefined, ["fr-FR"])).toBe("en");
  });
});

describe("loadPreferences and savePreferences", () => {
  afterEach(() => {
    if (originalLocalStorage) {
      Object.defineProperty(globalThis, "localStorage", originalLocalStorage);
    } else {
      Reflect.deleteProperty(globalThis, "localStorage");
    }
  });

  it("loads defaults when storage is empty or malformed", () => {
    stubLocalStorage();
    expect(loadPreferences()).toEqual(defaultPreferences);
    localStorage.setItem("oneday-browser-preferences-v2", "{bad json");
    expect(loadPreferences()).toEqual(defaultPreferences);
  });

  it("persists normalized preferences", () => {
    const storage = stubLocalStorage();
    savePreferences({ ...defaultPreferences, density: "comfortable", accent: "#ff91ad" });
    expect(storage.get("oneday-browser-preferences-v2")).toContain("comfortable");
    expect(loadPreferences()).toMatchObject({ density: "comfortable", accent: "#ff91ad" });
  });

  it("migrates legacy named accent colors", () => {
    stubLocalStorage();
    localStorage.setItem("oneday-browser-preferences-v2", JSON.stringify({ accent: "green" }));
    expect(loadPreferences().accent).toBe("#8ed979");
  });
});

function stubLocalStorage() {
  const storage = new Map<string, string>();
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, value);
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => storage.clear(),
    key: (index: number) => [...storage.keys()][index] ?? null,
    get length() {
      return storage.size;
    },
    },
  });
  return storage;
}
