import { Archive, BarChart3, BookOpen, BriefcaseBusiness, Clock3, FileText, Flag, Hammer, Search } from "lucide-react";
import type { CommandDescriptor, MetaCommand, ModuleTab, OverlayKind } from "./types";

export interface CommandResult {
  handled?: boolean;
  tab?: ModuleTab;
  overlay?: Exclude<OverlayKind, null>;
  text?: string;
  notice?: string;
  meta?: MetaCommand;
  saveName?: string;
  saveFilter?: string;
  saveDeleteFilter?: string;
	timeline?: { action: "list" | "fork" | "rename" | "checkout"; value?: string };
}

export interface CommandContext {
  descriptors?: CommandDescriptor[];
  npcNames?: string[];
  saveNames?: string[];
  visiblePrivateThoughts?: boolean;
}

export interface CommandSuggestionContext {
  npcNames?: string[];
  saveNames?: string[];
  recentCommands?: string[];
  visiblePrivateThoughts?: boolean;
}

export type CommandSuggestionKind = "command" | "completion" | "recent";

export interface SlashCommandItem {
  name: string;
  hint: string;
  aliases: string[];
  value: string;
  group: string;
  kind: CommandSuggestionKind;
  badge?: string;
  descriptor?: CommandDescriptor;
}

export interface CommandSuggestionGroup {
  key: string;
  label: string;
  items: SlashCommandItem[];
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
  { tab: "craft", label: "Craft", hotkey: "R", command: "/craft", Icon: Hammer },
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
  descriptor(
    "thoughts",
    "thoughts",
    "Thoughts",
    "Inspect saved NPC private thoughts when enabled.",
    "debug",
    "shared",
    "open_panel",
    [],
    false,
    "",
    "visible_private_thoughts",
  ),
  descriptor("codex", "codex", "Codex", "Open the story codex.", "state", "shared", "open_panel"),
  descriptor("characters", "characters", "Characters", "Open character records.", "state", "shared", "open_panel"),
  descriptor("fronts", "hooks", "Fronts", "Open fronts, hooks, fallout, and pressure clocks.", "state", "shared", "open_panel", ["fronts", "front"]),
  descriptor("investigations", "investigations", "Investigations", "Open the investigation workspace.", "state", "shared", "open_panel", ["investigation"]),
  descriptor("projects", "projects", "Projects", "Open downtime projects and progress clocks.", "state", "shared", "open_panel", ["project"]),
  descriptor("achievements", "achievements", "Achievements", "Show earned achievements.", "state", "shared", "open_panel", ["a"]),
  descriptor("craft", "craft", "Craft", "Open the crafting station.", "play", "shared", "open_panel", ["crafting"]),
  descriptor("history", "history", "History", "Open transcript and session history.", "state", "shared", "open_panel"),
	descriptor("branches", "branches", "Branches", "List and navigate alternate story branches.", "state", "shared", "timeline", ["branch"]),
	descriptor("fork", "fork", "Fork branch", "Fork the current story head into a named alternate.", "state", "shared", "timeline", [], true),
	descriptor("branch-rename", "branch-rename", "Rename branch", "Rename the active story branch.", "state", "shared", "timeline", ["rename-branch"], true),
	descriptor("checkout", "checkout", "Checkout branch", "Switch to a named branch without deleting the current one.", "state", "shared", "timeline", [], true),
  descriptor("talk", "talk", "Talk", "Talk to a nearby NPC with an optional intent and message.", "talk", "shared", "submit_action", [], true, "nearby_npcs"),
  descriptor("btw", "btw", "BTW", "Ask a contextual side question without advancing the turn.", "meta", "shared", "submit_meta", [], true),
  descriptor("guide", "guide", "Guide", "Store soft future-facing story guidance.", "meta", "shared", "submit_meta", [], true),
  descriptor("narrator", "narrator", "Narrator Control", "Direct narrator canon or correct world state.", "meta", "shared", "submit_meta", ["n"], true),
  descriptor("advance", "advance", "Advance", "Push to the next meaningful beat without replaying filler.", "play", "shared", "submit_action", [], true),
  descriptor("timeskip", "timeskip", "Time Skip", "Jump ahead to a later meaningful moment.", "play", "shared", "submit_action", [], true),
  descriptor("downtime", "downtime", "Downtime", "Request a quieter scene around a focus.", "play", "shared", "submit_action", [], true),
  descriptor("save", "save", "Save", "Create a manual save.", "save", "shared", "save_create", [], true),
  descriptor("load", "load", "Load", "Open or filter saved snapshots.", "save", "shared", "save_load", ["saves"], true, "saves"),
  descriptor("delete-save", "delete-save", "Delete Save", "Filter saves and delete one through confirmation.", "save", "browser_only", "save_delete", ["delete"], true, "saves"),
  descriptor("help", "help", "Help", "Show available commands.", "system", "shared", "local_only"),
  descriptor("quit", "quit", "Quit", "Save and leave the terminal session.", "system", "terminal_only", "local_only", ["q"]),
];

export const slashCommands = commandDescriptorsToSlashCommands(fallbackCommandDescriptors);

export const commandGroupLabels: Record<string, string> = {
  play: "Play",
  talk: "Talk",
  state: "State",
  save: "Saves",
  meta: "Meta",
  system: "System",
  debug: "Debug",
  recent: "Recent",
};

const commandGroupOrder = ["play", "talk", "state", "save", "meta", "system", "debug", "recent"];

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
  craft: "craft",
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
  if (!command) return { handled: true, notice: unknownCommandNotice(parsed.name, descriptors) };
  if (!isCommandEnabled(command, context)) {
    return { handled: true, notice: disabledCommandNotice(command) };
  }

  const canonical = command.canonical || command.id;
  switch (command.behavior) {
    case "open_panel": {
      const tab = panelByCanonical[canonical] ?? panelByCanonical[command.id];
      if (tab === "craft") return { handled: true, tab, overlay: "module" };
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
	case "timeline":
		return { handled: true, timeline: { action: canonical === "branches" ? "list" : canonical === "branch-rename" ? "rename" : canonical as "fork" | "checkout", value: parsed.argsText } };
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

export function commandSuggestions(
  draft: string,
  descriptors?: CommandDescriptor[],
  context: CommandSuggestionContext = {},
): SlashCommandItem[] {
  const trimmed = draft.trimStart();
  if (!trimmed.startsWith("/")) return [];

  const parsed = parseCommandDraft(trimmed);
  if (!parsed) return [];

  const allDescriptors = commandDescriptors(descriptors).filter((descriptor) => isCommandEnabled(descriptor, context));
  const command = parsed.commandName ? findCommandDescriptor(parsed.commandName, allDescriptors) : undefined;
  const canonical = command?.canonical ?? command?.id;
  if (command && parsed.hasArgs && canonical === "talk") {
    return talkCompletionSuggestions(command, parsed.argsText, context);
  }
  if (command && parsed.hasArgs && (canonical === "load" || canonical === "delete-save")) {
    return saveCompletionSuggestions(command, parsed.argsText, context.saveNames ?? []);
  }

  const query = parsed.commandName.toLowerCase();
  const commands = commandDescriptorsToSlashCommands(allDescriptors).filter((item) => {
    if (!query) return true;
    const descriptor = item.descriptor;
    return (
      item.name.slice(1).startsWith(query) ||
      descriptor?.canonical.toLowerCase().startsWith(query) ||
      descriptor?.title.toLowerCase().includes(query) ||
      item.aliases.some((alias) => stripSlash(alias).startsWith(query))
    );
  }).sort((left, right) => compareSuggestions(left, right, query));

  const recent = recentCommandSuggestions(query, context.recentCommands ?? []);
  return [...commands, ...recent];
}

export function commandDescriptors(descriptors?: CommandDescriptor[]): CommandDescriptor[] {
  if (!descriptors || descriptors.length === 0) return fallbackCommandDescriptors;

  const fallbackByID = new Map(fallbackCommandDescriptors.map((descriptor) => [descriptor.id, descriptor]));
  const seen = new Set<string>();
  const merged = descriptors.map((descriptor) => {
    seen.add(descriptor.id);
    const fallback = fallbackByID.get(descriptor.id);
    if (!fallback) return descriptor;
    return {
      ...fallback,
      ...descriptor,
      aliases: uniqueNames([...(descriptor.aliases ?? []), ...(fallback.aliases ?? [])]),
      examples: descriptor.examples?.length ? descriptor.examples : fallback.examples,
      enabled_when: descriptor.enabled_when ?? fallback.enabled_when,
    };
  });
  for (const fallback of fallbackCommandDescriptors) {
    if (!seen.has(fallback.id) && fallback.parity === "browser_only") merged.push(fallback);
  }
  return merged;
}

export function isCommandEnabled(descriptor: CommandDescriptor, context: CommandSuggestionContext = {}): boolean {
  const requirement = descriptor.enabled_when?.trim();
  if (!requirement) return true;
  if (requirement === "nearby_npcs") return (context.npcNames ?? []).length > 0;
  if (requirement === "saves") return (context.saveNames ?? []).length > 0;
  if (requirement === "visible_private_thoughts") return context.visiblePrivateThoughts === true;
  return false;
}

export function groupCommandSuggestions(items: SlashCommandItem[]): CommandSuggestionGroup[] {
  const groups = new Map<string, SlashCommandItem[]>();
  for (const item of items) {
    const key = item.group || "system";
    groups.set(key, [...(groups.get(key) ?? []), item]);
  }
  return [...groups.entries()]
    .sort(([left], [right]) => groupIndex(left) - groupIndex(right))
    .map(([key, groupedItems]) => ({
      key,
      label: commandGroupLabels[key] ?? titleCase(key),
      items: groupedItems,
    }));
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
      kind: "command",
      badge: item.parity === "browser_only" ? "Browser" : item.parity === "terminal_only" ? "Terminal" : undefined,
      descriptor: item,
    };
  });
}

function talkCompletionSuggestions(
  command: CommandDescriptor,
  argsText: string,
  context: CommandSuggestionContext,
): SlashCommandItem[] {
  const names = uniqueNames(context.npcNames ?? []);
  const matched = matchKnownName(argsText, names);
  if (!matched) {
    const query = argsText.toLowerCase();
    return names
      .filter((name) => !query || name.toLowerCase().includes(query))
      .slice(0, 8)
      .map((name) => completionSuggestion({
        command,
        group: "talk",
        name,
        value: `/talk ${name} `,
        hint: "Set talk target, then add intent and message.",
        badge: "NPC",
      }));
  }

  const intentQuery = firstToken(matched.rest).toLowerCase();
  if (matched.rest && !intentQuery) return [];
  if (matched.rest && talkIntents.has(intentQuery) && matched.rest.trim().split(/\s+/).length > 1) return [];

  return [...talkIntents]
    .filter((intent) => !intentQuery || intent.startsWith(intentQuery))
    .map((intent) => completionSuggestion({
      command,
      group: "talk",
      name: intent,
      value: `/talk ${matched.name} ${intent} `,
      hint: `Talk to ${matched.name} with ${intent} intent.`,
      badge: "Intent",
    }));
}

function saveCompletionSuggestions(command: CommandDescriptor, argsText: string, saveNames: string[]): SlashCommandItem[] {
  const query = argsText.toLowerCase();
  return uniqueNames(saveNames)
    .filter((name) => !query || name.toLowerCase().includes(query))
    .slice(0, 8)
    .map((name) => completionSuggestion({
      command,
      group: "save",
      name,
      value: `/${stripSlash(command.id)} ${name}`,
      hint: command.behavior === "save_delete" ? "Filter saves for deletion confirmation." : "Filter or load this saved snapshot.",
      badge: "Save",
    }));
}

function recentCommandSuggestions(query: string, recentCommands: string[]): SlashCommandItem[] {
  if (recentCommands.length === 0) return [];
  return uniqueNames(recentCommands)
    .filter((command) => command.trim().startsWith("/") && (!query || command.toLowerCase().includes(query)))
    .slice(0, 5)
    .map((command) => ({
      name: command,
      hint: "Reuse recent command.",
      aliases: [],
      value: command,
      group: "recent",
      kind: "recent" as const,
      badge: "Recent",
    }));
}

function completionSuggestion({
  command,
  group,
  name,
  value,
  hint,
  badge,
}: {
  command: CommandDescriptor;
  group: string;
  name: string;
  value: string;
  hint: string;
  badge: string;
}): SlashCommandItem {
  return {
    name,
    hint,
    aliases: [],
    value,
    group,
    kind: "completion",
    badge,
    descriptor: command,
  };
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

function disabledCommandNotice(command: CommandDescriptor): string {
  const name = slashName(command);
  if (command.enabled_when === "visible_private_thoughts") {
    return `${name} is disabled in player-safe browser mode. Enable visible_private_thoughts only for debug runs.`;
  }
  if (command.enabled_when === "nearby_npcs") {
    return `${name} needs at least one known nearby character. Open Codex or continue the scene first.`;
  }
  if (command.enabled_when === "saves") {
    return `${name} needs a saved snapshot. Use /save <name> first.`;
  }
  return `${name} is not available in the current browser state.`;
}

function unknownCommandNotice(name: string, descriptors: CommandDescriptor[]): string {
  if (!name.trim()) {
    return "Type a command after /. Press Ctrl+K or use /help to browse available commands.";
  }

  const commandName = `/${stripSlash(name)}`;
  const suggestions = commandSuggestions(commandName, descriptors)
    .filter((item) => item.kind === "command")
    .slice(0, 3)
    .map((item) => item.name);
  if (suggestions.length > 0) {
    return `Unknown command ${commandName}. Did you mean ${suggestions.join(", ")}?`;
  }
  return `Unknown command ${commandName}. Press Ctrl+K or use /help to browse available commands.`;
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

function parseCommandDraft(text: string): { commandName: string; argsText: string; hasArgs: boolean } | null {
  if (!text.startsWith("/")) return null;
  const body = text.slice(1);
  const commandName = firstToken(body);
  const afterCommand = body.slice(commandName.length);
  return {
    commandName,
    argsText: afterCommand.trimStart(),
    hasArgs: afterCommand.length > 0,
  };
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

function groupIndex(group: string): number {
  const index = commandGroupOrder.indexOf(group);
  return index === -1 ? commandGroupOrder.length : index;
}

function compareSuggestions(left: SlashCommandItem, right: SlashCommandItem, query = ""): number {
  const scoreDelta = suggestionQueryScore(left, query) - suggestionQueryScore(right, query);
  if (scoreDelta !== 0) return scoreDelta;
  const groupDelta = groupIndex(left.group) - groupIndex(right.group);
  if (groupDelta !== 0) return groupDelta;
  const kindDelta = suggestionKindIndex(left.kind) - suggestionKindIndex(right.kind);
  if (kindDelta !== 0) return kindDelta;
  return left.name.localeCompare(right.name);
}

function suggestionQueryScore(item: SlashCommandItem, query: string): number {
  if (!query) return 0;
  const name = item.name.slice(1).toLowerCase();
  const canonical = item.descriptor?.canonical.toLowerCase() ?? "";
  const title = item.descriptor?.title.toLowerCase() ?? "";
  const aliases = item.aliases.map((alias) => stripSlash(alias));
  if (name === query || canonical === query || aliases.some((alias) => alias === query)) return 0;
  if (name.startsWith(query)) return 1;
  if (canonical.startsWith(query)) return 2;
  if (aliases.some((alias) => alias.startsWith(query))) return 3;
  if (title.includes(query)) return 4;
  return 5;
}

function suggestionKindIndex(kind: CommandSuggestionKind): number {
  if (kind === "command") return 0;
  if (kind === "completion") return 1;
  return 2;
}

function titleCase(value: string): string {
  return value
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join(" ");
}

function uniqueNames(values: string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const value of values) {
    const clean = value.trim();
    const key = clean.toLowerCase();
    if (!clean || seen.has(key)) continue;
    seen.add(key);
    result.push(clean);
  }
  return result;
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
  enabledWhen = "",
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
    enabled_when: enabledWhen || undefined,
  };
}
