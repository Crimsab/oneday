import { describe, expect, it } from "vitest";
import { actionModeToText, commandToAction, moduleSpecs, tabHotkeys } from "./commands";

describe("commandToAction", () => {
  it("ignores normal prose", () => {
    expect(commandToAction("look around")).toEqual({});
  });

  it("opens module tabs for terminal-compatible slash commands", () => {
    expect(commandToAction("/inventory")).toMatchObject({ handled: true, tab: "inventory" });
    expect(commandToAction("/i")).toMatchObject({ handled: true, tab: "inventory" });
    expect(commandToAction("/stats")).toMatchObject({ handled: true, tab: "stats" });
    expect(commandToAction("/characters")).toMatchObject({ handled: true, tab: "codex" });
    expect(commandToAction("/front")).toMatchObject({ handled: true, tab: "fronts" });
    expect(commandToAction("/projects")).toMatchObject({ handled: true, tab: "projects" });
    expect(commandToAction("/history")).toMatchObject({ handled: true, tab: "history" });
  });

  it("opens overlays for help and creates browser saves", () => {
    expect(commandToAction("/help")).toMatchObject({ handled: true, overlay: "help" });
    expect(commandToAction("/load")).toMatchObject({ handled: true, tab: "saves", overlay: "saves", saveFilter: "" });
    expect(commandToAction("/load Camp")).toMatchObject({ handled: true, tab: "saves", overlay: "saves", saveFilter: "Camp" });
    expect(commandToAction("/save Camp")).toMatchObject({
      tab: "saves",
      overlay: "saves",
      saveName: "Camp",
    });
    expect(commandToAction("/save")).toMatchObject({ tab: "saves", overlay: "saves", saveName: "" });
  });

  it("maps meta commands to browser meta operations instead of narrative actions", () => {
    expect(commandToAction("/btw what changed?")).toMatchObject({ meta: { kind: "btw", text: "what changed?" } });
    expect(commandToAction("/guide foreshadow the storm")).toMatchObject({ meta: { kind: "guide", text: "foreshadow the storm" } });
    expect(commandToAction("/narrator keep it weird")).toMatchObject({ meta: { kind: "narrator", text: "keep it weird" } });
    expect(commandToAction("/n keep it short")).toMatchObject({ meta: { kind: "narrator", text: "keep it short" } });
  });

  it("maps advance, timeskip, downtime, and talk commands into actions", () => {
    expect(commandToAction("/advance docks")).toMatchObject({ text: expect.stringContaining("[Advance Scene]") });
    expect(commandToAction("/timeskip tomorrow")).toMatchObject({ text: expect.stringContaining("[Time Skip]") });
    expect(commandToAction("/downtime train")).toEqual({ text: "[Downtime Scene] train" });
    expect(commandToAction("/talk Maren probe the ledger")).toEqual({ text: "[Talk to Maren | intent:probe] the ledger" });
  });

  it("returns usage notices for incomplete commands", () => {
    expect(commandToAction("/btw")).toMatchObject({ handled: true, notice: expect.stringContaining("Usage") });
    expect(commandToAction("/guide")).toMatchObject({ handled: true, notice: expect.stringContaining("Usage") });
    expect(commandToAction("/narrator")).toMatchObject({ handled: true, notice: expect.stringContaining("Usage") });
    expect(commandToAction("/downtime")).toMatchObject({ handled: true, notice: expect.stringContaining("Usage") });
    expect(commandToAction("/talk")).toMatchObject({ handled: true, tab: "codex", notice: expect.stringContaining("/talk") });
    expect(commandToAction("/talk Maren ask")).toMatchObject({ handled: true, tab: "codex", notice: expect.stringContaining("Maren") });
    expect(commandToAction("/quit")).toMatchObject({ handled: true, notice: expect.stringContaining("terminal") });
  });
});

describe("actionModeToText", () => {
  it("wraps composer modes with terminal-compatible intent text", () => {
    expect(actionModeToText("action", " open door ")).toBe("open door");
    expect(actionModeToText("talk", "hello")).toBe("[Talk] hello");
    expect(actionModeToText("advance", "next room")).toContain("[Advance Scene]");
    expect(actionModeToText("timeskip", "night")).toContain("[Time Skip]");
  });
});

describe("module hotkeys", () => {
  it("has one hotkey for every module spec", () => {
    for (const spec of moduleSpecs) {
      expect(tabHotkeys[spec.hotkey.toLowerCase()]).toBe(spec.tab);
    }
  });
});
