import { readFileSync } from "node:fs";
import { afterEach, describe, expect, it } from "vitest";
import i18n, { setInterfaceLocale } from "./i18n";

describe("application notifications", () => {
  afterEach(async () => {
    await setInterfaceLocale("en");
  });

  it("provides natural Italian interpolation and plural forms", async () => {
    await setInterfaceLocale("it-IT");

    expect(i18n.t("notifications:health.stories", { count: 1, formattedCount: "1" })).toBe("1 storia");
    expect(i18n.t("notifications:health.stories", { count: 2, formattedCount: "2" })).toBe("2 storie");
    expect(i18n.t("notifications:story.rowsAffected", { count: 1, formattedCount: "1" })).toContain("Riga del database");
    expect(i18n.t("notifications:story.rowsAffected", { count: 4, formattedCount: "4" })).toContain("Righe del database");
    expect(i18n.t("notifications:visual.failed", { count: 1, formattedCount: "1" })).toBe("1 non riuscita");
    expect(i18n.t("notifications:visual.failed", { count: 3, formattedCount: "3" })).toBe("3 non riuscite");
    expect(i18n.t("notifications:save.loadConfirm", { name: "Prima del porto", turn: "12" })).toContain("«Prima del porto» dal turno 12");
  });

  it("localizes sync presentation without replacing stable internal states", async () => {
    await setInterfaceLocale("it");
    expect(i18n.t("notifications:sync.live")).toBe("In diretta");
    expect(i18n.t("notifications:sync.connection", { status: i18n.t("notifications:sync.reconnecting") })).toBe("Connessione: Riconnessione");

    const appSource = readFileSync(new URL("App.tsx", import.meta.url), "utf8");
    const topBarSource = readFileSync(new URL("components/TopBar.tsx", import.meta.url), "utf8");
    expect(appSource).toContain('setSync("Live")');
    expect(appSource).toContain("notifications:sync.${sync.toLowerCase()}");
    expect(appSource).not.toContain('setSync("In diretta")');
    expect(topBarSource).toContain("syncLabel");
    expect(topBarSource).toContain("sync.toLowerCase()");
  });

  it("routes the important App action feedback through catalogs", () => {
    const source = readFileSync(new URL("App.tsx", import.meta.url), "utf8");
    for (const untranslated of [
      "Story deleted.",
      "Meta command completed.",
      "Resolving your action...",
      "Visual profile saved.",
      "Visual version selected.",
      "Another OneDay request is already running.",
    ]) {
      expect(source).not.toContain(`"${untranslated}"`);
    }
    expect(source).toContain("notifications:story.deletePrompt");
    expect(source).toContain("notifications:save.loadConfirm");
    expect(source).toContain("notifications:visual.generationQueued");
  });
});
