import { Archive, BarChart3, BookOpen, BriefcaseBusiness, Clock3, FileText, Flag, Search } from "lucide-react";
import type { CommandDescriptor, MetaCommand, ModuleTab } from "./types";

export interface CommandResult {
  handled?: boolean;
  tab?: ModuleTab;
  overlay?: "help" | "saves";
  text?: string;
  notice?: string;
  meta?: MetaCommand;
  saveName?: string;
  saveFilter?: string;
  saveDeleteFilter?: string;
}

export interface CommandContext {
  descriptors?: CommandDescriptor[];
  npcNames?: string[];
}

export interface SlashCommandItem {
  name: string;
  hint: string;
  aliases: string[];
  value: string;
  group: string;
  descriptor: CommandDescriptor;
}

export const moduleSpecs: Array<{
  tab: ModuleTab;
  label: string;
  hotkey: string;
  command: string;
  Icon: typeof Clock3;
}> = [
  { tab: "history", label: "History", hotkey: "H", command: "/history", Icon: Clock3 },
  { tab: "inventory", label: "Inventory", hotkey: "I", command: "/inventory", Icon: Archive },
  { tab: "stats", label: "Stats", hotkey: "S", command: "/stats", Icon: BarChart3 },
  { tab: "codex", label: "Codex", hotkey: "C", command: "/codex", Icon: BookOpen },
  { tab: "fronts", label: "Fronts", hotkey: "F", command: "/fronts", Icon: Flag },
  { tab: "investigations", label: "Investigations", hotkey: "G", command: "/investigations", Icon: Search },
  { tab: "projects", label: "Projects", hotkey: "P", command: "/projects", Icon: BriefcaseBusiness },
  { tab: "saves", label: "Saves", hotkey: "V", command: "/load", Icon: FileText },
];

export const fallbackCommandDescriptors: CommandDescriptor[] = [
  descriptor("inventory", "inventory", "Inventory", "Open inventory and crafting context.", "state", "shared", "open_panel", ["i"]),
  descriptor("stats", "stats", "Stats", "Open the character sheet.", "state", "shared", "open_panel", ["s"]),
  descriptor("map", "map", "Map", "Open known locations and travel context.", "state", "shared", "open_panel", ["m"]),
  descriptor("journal", "journal", "Journal", "Open chapter journal and story notes.", "state", "shared", "open_panel", ["j"]),
  descriptor("thoughts", "thoughts", "Thoughts", "Inspect saved NPC private thoughts when enabled.", "debug", "shared", "open_panel"),
  descriptor("codex", "codex", "Codex", "Open the story codex.", "state", "shared", "open_panel"),
  descriptor("characters", "characters", "Characters", "Open character records.", "state", "shared", "open_panel"),
  descriptor("fronts", "hooks", "Fronts", "Open fronts, hooks, fallout, and pressure clocks.", "state", "shared", "open_panel", ["hooks", "fronts", "front"]),
  descriptor("investigations", "investigations", "Investigations", "Open the investigation workspace.", "state", "shared", "open_panel", ["investigation"]),
  descriptor("projects", "projects", "Projects", "Open downtime projects and progress clocks.", "state", "shared", "open_panel", ["project"]),
  descriptor("achievements", "achievements", "Achievements", "Show earned achievements.", "state", "shared", "open_panel", ["a"]),
  descriptor("craft", "craft", "Craft", "Open the crafting station.", "play", "shared", "open_panel", ["crafting"]),
  descriptor("history", "history", "History", "Open transcript and session history.", "state", "shared", "open_panel"),
  descriptor("talk", "talk", "Talk", "Talk to a nearby NPC with an optional intent and message.", "talk", "shared", "submit_action", [], true, "nearby_npcs"),
  descriptor("btw", "btw", "BTW", "Ask a contextual side question without advancing the turn.", "meta", "shared", "submit_meta", [], true),
  descriptor("guide", "guide", "Guide", "Store soft future-facing story guidance.", "meta", "shared", "submit_meta", [], true),
  descriptor("narrator", "narrator", "Narrator Control", "Direct narrator canon or correct world state.", "meta", "shared", "submit_meta", ["n"], true),
  descriptor("advance", "advance", "Advance", "Push to the next meaningful beat without replaying filler.", "play", "shared", "submit_action", [], true),
  descriptor("timeskip", "timeskip", "Time Skip", "Jump ahead to a later meaningful moment.", "play", "shared", "submit_action", [], true),
  descriptor("downtime", "downtime", "Downtime", "Request a quieter scene around a focus.", "play", "shared", "submit_action", [], true),
  descriptor("save", "save", "Save", "Create a manual save.", "save", "shared", "save_create", [], true),
  descriptor("load", "load", "Load", "Open or filter saved snapshots.", "save", "shared", "save_load", [], true, "saves"),
  descriptor("delete-save", "delete-save", "Delete Save", "Filter saves and delete one through confirmation.", "save", "browser_only", "save_delete", ["delete"], true, "saves"),
  descriptor("help", "help", "Help", "Show available commands.", "system", "shared", "local_only"),
  descriptor("quit", "quit", "Quit", "Save and leave the terminal session.", "system", "terminal_only", "local_only", ["q"]),
];

export const slashCommands = commandDescriptorsToSlashCommands(fallbackCommandDescriptors);

const panelByCanonical: Record<string, ModuleTab> = {
  inventory: "inventory",
  stats: "stats",
  characters: "codex",
  codex: "codex",
  journal: "codex",
  thoughts: "codex",
  hooks: "fronts",
  fronts: "fronts",
  investigations: "investigations",
  projects: "projects",
  craft: "inventory",
  history: "history",
  achievements: "saves",
  map: "history",
};

export const tabHotkeys: Record<string, ModuleTab> = moduleSpecs.reduce<Record<string, ModuleTab>>((acc, item) => {
  acc[item.hotkey.toLowerCase()] = item.tab;
  return acc;
}, {});

const talkIntents = new Set(["ask", "probe", "bond", "bargain", "threaten", "promise", "lie", "confess"]);
const metaKinds: Record<string, MetaCommand["kind"]> = {
  btw: "btw",
  guide: "guide",
  narrator: "narrator",
};

export function commandToAction(rawText: string, context: CommandContext = {}): CommandResult {
  const text = rawText.trim();
  const parsed = parseSlashCommand(text);
  if (!parsed) return {};

  const descriptors = commandDescriptors(context.descriptors);
  const command = findCommandDescriptor(parsed.name, descriptors);
  if (!command) return {};

  const canonical = command.canonical || command.id;
  switch (command.behavior) {
    case "open_panel": {
      const tab = panelByCanonical[canonical] ?? panelByCanonical[command.id];
      return tab ? { handled: true, tab } : { handled: true };
    }
    case "save_load":
      return { handled: true, tab: "saves", overlay: "saves", saveFilter: parsed.argsText };
    case "save_create":
      return { tab: "saves", overlay: "saves", saveName: parsed.argsText };
    case "save_delete":
      return {
        handled: true,
        tab: "saves",
        overlay: "saves",
        saveDeleteFilter: parsed.argsText,
        notice: parsed.argsText ? `Filtered saves for deletion: ${parsed.argsText}` : "Select a save to delete from the save drawer.",
      };
    case "submit_meta":
      return metaCommandToAction(canonical, parsed.argsText);
    case "submit_action":
      return submitActionCommandToAction(canonical, parsed.argsText, context);
    case "local_only":
      return localCommandToAction(canonical, command.id);
    case "insert_template":
      return { handled: true, text: command.examples?.[0] ?? "" };
    default:
      return {};
  }
}

export function actionModeToText(mode: string, text: string): string {
  const clean = text.trim();
  if (mode === "advance") return buildAdvanceSceneAction(clean);
  if (mode === "timeskip") return buildTimeSkipAction(clean);
  if (mode === "talk" && !clean.toLowerCase().startsWith("/talk")) return `[Talk] ${clean}`;
  return clean;
}

export function commandSuggestions(draft: string, descriptors?: CommandDescriptor[]): SlashCommandItem[] {
  const trimmed = draft.trim();
  if (!trimmed.startsWith("/")) return [];

  const query = trimmed.slice(1).split(/\s+/)[0]?.toLowerCase() ?? "";
  return commandDescriptorsToSlashCommands(commandDescriptors(descriptors)).filter((command) => {
    if (!query) return true;
    return (
      command.name.slice(1).startsWith(query) ||
      command.descriptor.canonical.toLowerCase().startsWith(query) ||
      command.aliases.some((alias) => stripSlash(alias).startsWith(query))
    );
  });
}

export function commandDescriptors(descriptors?: CommandDescriptor[]): CommandDescriptor[] {
  return descriptors && descriptors.length > 0 ? descriptors : fallbackCommandDescriptors;
}

export function commandDescriptorsToSlashCommands(descriptors: CommandDescriptor[]): SlashCommandItem[] {
  return descriptors.map((item) => {
    const name = slashName(item);
    const value = item.trailing_space ? `${name} ` : name;
    return {
      name,
      hint: item.description,
      aliases: (item.aliases ?? []).map((alias) => `/${stripSlash(alias)}`),
      value,
      group: item.group,
      descriptor: item,
    };
  });
}

function metaCommandToAction(canonical: string, value: string): CommandResult {
  const kind = metaKinds[canonical];
  if (!kind) return {};
  if (!value) return { handled: true, notice: metaUsage(kind) };
  return { meta: { kind, text: value } };
}

function submitActionCommandToAction(canonical: string, argsText: string, context: CommandContext): CommandResult {
  if (canonical === "advance") return { text: buildAdvanceSceneAction(argsText) };
  if (canonical === "timeskip") return { text: buildTimeSkipAction(argsText) };
  if (canonical === "downtime") {
    if (!argsText) return { handled: true, notice: "Usage: /downtime <focus>" };
    return { text: `[Downtime Scene] ${argsText}` };
  }
  if (canonical === "talk") return talkCommandToAction(argsText, context);
  return {};
}

function localCommandToAction(canonical: string, id: string): CommandResult {
  if (canonical === "help" || id === "help") {
    return { handled: true, overlay: "help" };
  }
  if (canonical === "quit" || id === "quit") {
    return { handled: true, notice: "Quit remains a terminal session-menu action. Browser realtime sync stays connected." };
  }
  return { handled: true };
}

function talkCommandToAction(argsText: string, context: CommandContext): CommandResult {
  if (!argsText) {
    return { handled: true, tab: "codex", notice: "Use /talk <npc> [intent] [message]. Known NPCs are in Codex." };
  }

  const matched = matchKnownName(argsText, context.npcNames ?? []);
  const target = matched?.name ?? firstToken(argsText);
  let rest = matched ? matched.rest : argsText.slice(target.length).trim();
  let intent = "ask";
  const next = firstToken(rest).toLowerCase();
  if (talkIntents.has(next)) {
    intent = next;
    rest = rest.slice(next.length).trim();
  }

  if (!rest) {
    return { handled: true, tab: "codex", notice: `Talk target set: ${target} (${intent}). Add a message to send it.` };
  }
  return { text: `[Talk to ${target} | intent:${intent}] ${rest}` };
}

function buildAdvanceSceneAction(hint: string): string {
  let base =
    "[Advance Scene] Move to the next meaningful beat now. If this micro-scene is exhausted, do not replay it with near-identical prose or choices. Introduce a concrete change: reveal, consequence, interruption, pressure, location shift, or a natural time skip.";
  if (hint) {
    base += ` Requested timing or destination: ${hint}`;
  }
  return base;
}

function buildTimeSkipAction(hint: string): string {
  let base =
    "[Time Skip] Jump forward to a later meaningful moment instead of playing filler turn by turn. Keep continuity clear: show what changed, what stayed true, and why this later beat matters now.";
  if (hint) {
    base += ` Requested destination: ${hint}`;
  }
  return base;
}

function findCommandDescriptor(name: string, descriptors: CommandDescriptor[]): CommandDescriptor | undefined {
  const clean = stripSlash(name);
  return descriptors.find((descriptor) => {
    if (stripSlash(descriptor.id) === clean) return true;
    if (stripSlash(descriptor.canonical) === clean) return true;
    return (descriptor.aliases ?? []).some((alias) => stripSlash(alias) === clean);
  });
}

function parseSlashCommand(text: string): { name: string; argsText: string } | null {
  if (!text.startsWith("/")) return null;
  const body = text.slice(1).trim();
  if (!body) return { name: "", argsText: "" };
  const name = firstToken(body);
  return { name, argsText: body.slice(name.length).trim() };
}

function matchKnownName(argsText: string, names: string[]): { name: string; rest: string } | null {
  const lowerArgs = argsText.toLowerCase();
  let best: { name: string; rest: string } | null = null;
  for (const rawName of names) {
    const name = rawName.trim();
    if (!name) continue;
    const lowerName = name.toLowerCase();
    const exact = lowerArgs === lowerName;
    const prefix = lowerArgs.startsWith(`${lowerName} `);
    if (!exact && !prefix) continue;
    if (best && best.name.length >= name.length) continue;
    best = { name, rest: exact ? "" : argsText.slice(name.length).trim() };
  }
  return best;
}

function firstToken(value: string): string {
  return value.trim().split(/\s+/)[0] ?? "";
}

function slashName(descriptor: CommandDescriptor): string {
  return `/${stripSlash(descriptor.id || descriptor.canonical)}`;
}

function stripSlash(value: string): string {
  return value.replace(/^\/+/, "").trim().toLowerCase();
}

function metaUsage(kind: MetaCommand["kind"]): string {
  if (kind === "btw") return "Usage: /btw <quick question about the current story>";
  if (kind === "guide") return "Usage: /guide <future beat or chapter wish>";
  return "Usage: /n <narrator instruction or canon correction>";
}

function descriptor(
  id: CommandDescriptor["id"],
  canonical: CommandDescriptor["canonical"],
  title: CommandDescriptor["title"],
  description: CommandDescriptor["description"],
  group: CommandDescriptor["group"],
  parity: CommandDescriptor["parity"],
  behavior: CommandDescriptor["behavior"],
  aliases: string[] = [],
  trailingSpace = false,
  completionProvider = "",
): CommandDescriptor {
  return {
    id,
    canonical,
    title,
    description,
    group,
    parity,
    behavior,
    aliases,
    trailing_space: trailingSpace,
    completion_provider: completionProvider,
  };
}
