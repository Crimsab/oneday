import { afterEach, describe, expect, it } from "vitest";
import { defaultPreferences, loadPreferences, normalizePreferences, savePreferences } from "./preferences";

const originalLocalStorage = Object.getOwnPropertyDescriptor(globalThis, "localStorage");

describe("normalizePreferences", () => {
  it("keeps valid preferences and falls back for invalid values", () => {
    expect(
      normalizePreferences({
        density: "compact",
        fontSize: "large",
        accent: "blue",
        showLeftRail: false,
        showInspector: false,
        wrapTranscript: false,
      }),
    ).toEqual({
      density: "compact",
      fontSize: "large",
      accent: "blue",
      showLeftRail: false,
      showInspector: false,
      wrapTranscript: false,
    });

    expect(
      normalizePreferences({
        density: "wide" as never,
        fontSize: "huge" as never,
        accent: "orange" as never,
        showLeftRail: "yes" as never,
        showInspector: "yes" as never,
        wrapTranscript: "no" as never,
      }),
    ).toEqual(defaultPreferences);
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
    savePreferences({ ...defaultPreferences, density: "comfortable", accent: "rose" });
    expect(storage.get("oneday-browser-preferences-v2")).toContain("comfortable");
    expect(loadPreferences()).toMatchObject({ density: "comfortable", accent: "rose" });
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
