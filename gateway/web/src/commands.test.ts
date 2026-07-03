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

  it("opens overlays for help, saves, and save preparation", () => {
    expect(commandToAction("/help")).toMatchObject({ handled: true, overlay: "help" });
    expect(commandToAction("/load")).toMatchObject({ handled: true, tab: "saves", overlay: "saves" });
    expect(commandToAction("/save Camp")).toMatchObject({ handled: true, tab: "saves", overlay: "saves" });
  });

  it("maps narrator, guide, btw, advance, timeskip, downtime, and talk commands into actions", () => {
    expect(commandToAction("/btw what changed?")).toEqual({ text: "[BTW] what changed?" });
    expect(commandToAction("/guide foreshadow the storm")).toEqual({ text: "[Guide] foreshadow the storm" });
    expect(commandToAction("/narrator keep it weird")).toEqual({ text: "[Narrator] keep it weird" });
    expect(commandToAction("/n keep it short")).toEqual({ text: "[Narrator] keep it short" });
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
