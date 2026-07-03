import { describe, expect, it } from "vitest";
import {
  actionModeToText,
  commandDescriptors as resolveCommandDescriptors,
  commandSuggestions,
  commandToAction,
  fallbackCommandDescriptors,
  groupCommandSuggestions,
  isCommandEnabled,
  moduleSpecs,
  tabHotkeys,
} from "./commands";

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
    expect(commandToAction("/craft")).toMatchObject({ handled: true, tab: "craft", overlay: "module" });
    expect(commandToAction("/crafting")).toMatchObject({ handled: true, tab: "craft", overlay: "module" });
  });

  it("opens overlays for help and creates browser saves", () => {
    expect(commandToAction("/help")).toMatchObject({ handled: true, overlay: "help" });
    expect(commandToAction("/load")).toMatchObject({ handled: true, tab: "saves", overlay: "saves", saveFilter: "" });
    expect(commandToAction("/load Camp")).toMatchObject({ handled: true, tab: "saves", overlay: "saves", saveFilter: "Camp" });
    expect(commandToAction("/saves")).toMatchObject({ handled: true, tab: "saves", overlay: "saves", saveFilter: "" });
    expect(commandToAction("/delete-save Camp")).toMatchObject({
      handled: true,
      tab: "saves",
      overlay: "saves",
      saveDeleteFilter: "Camp",
    });
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
    expect(commandToAction("/talk Maren Lo probe the ledger", { npcNames: ["Maren Lo"] })).toEqual({
      text: "[Talk to Maren Lo | intent:probe] the ledger",
    });
  });

  it("returns usage notices for incomplete commands", () => {
    expect(commandToAction("/btw")).toMatchObject({ handled: true, notice: expect.stringContaining("Usage") });
    expect(commandToAction("/guide")).toMatchObject({ handled: true, notice: expect.stringContaining("Usage") });
    expect(commandToAction("/narrator")).toMatchObject({ handled: true, notice: expect.stringContaining("Usage") });
    expect(commandToAction("/downtime")).toMatchObject({ handled: true, notice: expect.stringContaining("Usage") });
    expect(commandToAction("/talk")).toMatchObject({ handled: true, tab: "codex", notice: expect.stringContaining("/talk") });
    expect(commandToAction("/talk Maren ask", { npcNames: ["Maren"] })).toMatchObject({
      handled: true,
      tab: "codex",
      notice: expect.stringContaining("Maren"),
    });
    expect(commandToAction("/quit")).toMatchObject({ handled: true, notice: expect.stringContaining("terminal") });
  });

  it("keeps browser save deletion out of freeform action fallback", () => {
    const result = commandToAction("/delete-save Before docks");
    expect(result.text).toBeUndefined();
    expect(result.handled).toBe(true);
    expect(result.saveDeleteFilter).toBe("Before docks");
  });

  it("honors player-safe enabled_when gates before opening debug panels", () => {
    expect(commandToAction("/thoughts")).toMatchObject({
      handled: true,
      notice: expect.stringContaining("player-safe browser mode"),
    });
    expect(commandToAction("/thoughts", { visiblePrivateThoughts: true })).toMatchObject({ handled: true, tab: "codex" });
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

describe("commandSuggestions", () => {
  it("lists descriptor-backed commands and aliases", () => {
    expect(commandSuggestions("/del").map((command) => command.name)).toContain("/delete-save");
    expect(commandSuggestions("/fro").map((command) => command.name)).toContain("/fronts");
    expect(commandSuggestions("/hooks").map((command) => command.name)).toContain("/fronts");
    expect(commandSuggestions("/sav").find((command) => command.name === "/load")?.aliases).toContain("/saves");
  });

  it("completes multi-word NPC names and talk intents", () => {
    const npcSuggestions = commandSuggestions("/talk Mar", undefined, { npcNames: ["Maren Lo", "Dockworker"] });
    expect(npcSuggestions[0]).toMatchObject({
      kind: "completion",
      name: "Maren Lo",
      value: "/talk Maren Lo ",
      group: "talk",
    });

    const intentSuggestions = commandSuggestions("/talk Maren Lo pr", undefined, { npcNames: ["Maren Lo"] });
    expect(intentSuggestions.map((item) => item.value)).toContain("/talk Maren Lo probe ");
  });

  it("completes save names for load and delete-save commands", () => {
    const saveContext = { saveNames: ["Before the docks", "Chapter 2 checkpoint"] };
    expect(commandSuggestions("/load dock", undefined, saveContext)).toMatchObject([
      { kind: "completion", name: "Before the docks", value: "/load Before the docks", group: "save" },
    ]);
    expect(commandSuggestions("/delete-save chapter", undefined, saveContext)).toMatchObject([
      { kind: "completion", name: "Chapter 2 checkpoint", value: "/delete-save Chapter 2 checkpoint", group: "save" },
    ]);
  });

  it("groups palette rows by command family with recent commands last", () => {
    const groups = groupCommandSuggestions(commandSuggestions("/", undefined, { recentCommands: ["/advance docks"] }));
    expect(groups.map((group) => group.key)).toContain("play");
    expect(groups.at(-1)).toMatchObject({ key: "recent", label: "Recent" });
  });

  it("hides disabled debug commands from the browser palette", () => {
    expect(commandSuggestions("/tho").map((command) => command.name)).not.toContain("/thoughts");
    expect(commandSuggestions("/tho", undefined, { visiblePrivateThoughts: true }).map((command) => command.name)).toContain("/thoughts");
  });
});

describe("isCommandEnabled", () => {
  it("supports the enabled_when requirements exposed by the terminal contract", () => {
    expect(isCommandEnabled({ ...moduleCommand("thoughts"), enabled_when: "visible_private_thoughts" })).toBe(false);
    expect(isCommandEnabled({ ...moduleCommand("thoughts"), enabled_when: "visible_private_thoughts" }, { visiblePrivateThoughts: true })).toBe(true);
    expect(isCommandEnabled({ ...moduleCommand("talk"), enabled_when: "nearby_npcs" }, { npcNames: ["Maren"] })).toBe(true);
    expect(isCommandEnabled({ ...moduleCommand("load"), enabled_when: "saves" }, { saveNames: [] })).toBe(false);
  });
});

describe("browser command coverage", () => {
  it("maps every descriptor trigger to a concrete browser result", () => {
    const context = {
      descriptors: fallbackCommandDescriptors,
      npcNames: ["Maren Lo"],
      saveNames: ["Before docks"],
      visiblePrivateThoughts: true,
    };

    for (const descriptor of resolveCommandDescriptors(fallbackCommandDescriptors)) {
      const triggers = new Set([descriptor.id, descriptor.canonical, ...(descriptor.aliases ?? [])].filter(Boolean));
      for (const trigger of triggers) {
        const result = commandToAction(commandSample(trigger, descriptor.behavior), context);
        expect(result, `${descriptor.id} via /${trigger}`).not.toEqual({});
        expect(
          Boolean(
            result.handled ||
              result.tab ||
              result.overlay ||
              result.text ||
              result.meta ||
              result.notice ||
              result.saveName !== undefined ||
              result.saveFilter !== undefined ||
              result.saveDeleteFilter !== undefined,
          ),
          `${descriptor.id} via /${trigger} produced no actionable browser result`,
        ).toBe(true);
      }
    }
  });
});

function moduleCommand(id: string) {
  return {
    id,
    canonical: id,
    aliases: [],
    title: id,
    description: "",
    group: "state" as const,
    parity: "shared" as const,
    behavior: "open_panel" as const,
  };
}

function commandSample(trigger: string, behavior: string): string {
  if (behavior === "submit_meta") return `/${trigger} note`;
  if (behavior === "submit_action") {
    if (trigger === "talk") return "/talk Maren Lo ask about the ledger";
    if (trigger === "downtime") return `/${trigger} repair gear`;
    return `/${trigger} toward the next clue`;
  }
  if (behavior === "save_create" || behavior === "save_load" || behavior === "save_delete") return `/${trigger} Before docks`;
  return `/${trigger}`;
}
