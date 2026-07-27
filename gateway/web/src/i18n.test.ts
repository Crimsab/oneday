import { afterEach, describe, expect, it } from "vitest";
import i18n, { formatInterfaceNumber, humanizeMissingKey, resources, setInterfaceLocale } from "./i18n";

function paths(value: unknown, prefix = ""): string[] {
  if (!value || typeof value !== "object") return [prefix];
  return Object.entries(value).flatMap(([key, child]) => paths(child, prefix ? `${prefix}.${key}` : key));
}

function messages(value: unknown, prefix = ""): Array<[string, string]> {
  if (typeof value === "string") return [[prefix, value]];
  if (!value || typeof value !== "object") return [];
  return Object.entries(value).flatMap(([key, child]) => messages(child, prefix ? `${prefix}.${key}` : key));
}

function interpolationVariables(message: string): string[] {
  return [...message.matchAll(/{{\s*([^},\s]+)[^}]*}}/g)].map((match) => match[1]).sort();
}

describe("interface catalogs", () => {
  afterEach(async () => { await setInterfaceLocale("en"); });

  it("keeps English and Italian catalogs in exact key parity", () => {
    expect(paths(resources.it).sort()).toEqual(paths(resources.en).sort());
  });

  it("keeps interpolation variables in exact parity", () => {
    const italian = new Map(messages(resources.it));
    for (const [key, english] of messages(resources.en)) {
      expect(interpolationVariables(italian.get(key) ?? ""), key).toEqual(interpolationVariables(english));
    }
  });

  it("covers every stable story-wizard action key", () => {
    expect(Object.keys(resources.en.wizard.actions).sort()).toEqual([
      "accept_rules", "accept_stats", "accept_world", "create_story", "crunchier_stats", "edit_final", "edit_rules", "edit_stats", "edit_world", "fewer_factions", "focus_input", "lighter_stats", "make_darker", "make_lighter", "more_danger", "no_combat", "preset_cozy", "preset_cyberpunk", "preset_dark_fantasy", "preset_horror", "regenerate_all", "regenerate_rules", "regenerate_world", "skip_background",
    ]);
  });

  it("falls back to English, then a readable label derived from the missing key", async () => {
    await setInterfaceLocale("it-IT");
    expect(i18n.t("common:save")).toBe("Salva");
    expect(i18n.t("common:not.a.real_key")).toBe("Real key");
    expect(humanizeMissingKey("drawer:assetKind.map_background")).toBe("Map background");
  });

  it("supports interpolation, plural forms, and locale number formatting", async () => {
    await setInterfaceLocale("it");
    expect(i18n.t("common:match", { count: 1, query: "audio" })).toContain("1 opzione");
    expect(i18n.t("common:match", { count: 2, query: "audio" })).toContain("2 opzioni");
    expect(formatInterfaceNumber(12345)).toBe("12.345");
    expect(i18n.t("drawer:mapArt.icons", { ready: 2, total: 5 })).toBe("2/5 icone");
    expect(i18n.t("library:manage", { name: "L’Ultimo Giorno" })).toBe("Gestisci L’Ultimo Giorno");
  });

  it("normalizes regional variants and updates the document language", async () => {
    const documentDescriptor = Object.getOwnPropertyDescriptor(globalThis, "document");
    Object.defineProperty(globalThis, "document", { configurable: true, value: { documentElement: { lang: "" } } });
    expect(await setInterfaceLocale("it-IT")).toBe("it");
    expect(document.documentElement.lang).toBe("it");
    if (documentDescriptor) Object.defineProperty(globalThis, "document", documentDescriptor);
    else Reflect.deleteProperty(globalThis, "document");
  });
});
