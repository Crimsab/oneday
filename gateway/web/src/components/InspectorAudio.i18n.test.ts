import { readFileSync } from "node:fs";
import { afterEach, describe, expect, it } from "vitest";
import i18n, { setInterfaceLocale } from "../i18n";
import { moduleTitle } from "./Inspector";

describe("Inspector and audio-tools localization", () => {
  afterEach(async () => { await setInterfaceLocale("en"); });

  it("switches stable inspector presentation without changing canonical identifiers", async () => {
    await setInterfaceLocale("it-IT");
    expect(moduleTitle("inventory")).toBe("Inventario");
    expect(i18n.t("inspector_extra:openLarge", { title: "Inventario" })).toBe("Apri Inventario in una vista più ampia");
    expect(i18n.t("audio_tools:deleteEntry", { source: "Lyanna" })).toBe("Elimina la pronuncia di Lyanna");
    const audioSource = readFileSync(new URL("AudioLanguageTools.tsx", import.meta.url), "utf8");
    expect(audioSource).toContain('{ value: "provider", label: t("audio_tools:providerGuidance") }');
    expect(audioSource).toContain('{ value: "x-sampa", label: "X-SAMPA" }');
  });

  it.each([
    ["Inspector.tsx", ["Story details", "Crafting station ready", "No matching characters.", "Snapshots and play sessions"]],
    ["AudioLanguageTools.tsx", ["Pronunciation saved.", "Audit cache", "Remove orphaned files", "Working…"]],
  ])("keeps important %s copy out of component literals", (file, literals) => {
    const source = readFileSync(new URL(file, import.meta.url), "utf8");
    for (const literal of literals) expect(source).not.toContain(literal);
  });
});
