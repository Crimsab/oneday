import { Archive, BarChart3, BookOpen, BriefcaseBusiness, Clock3, FileText, Flag, Search } from "lucide-react";
import type { ModuleTab } from "./types";

export interface CommandResult {
  handled?: boolean;
  tab?: ModuleTab;
  overlay?: "help" | "saves";
  text?: string;
  notice?: string;
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

export const slashCommands = [
  { name: "/inventory", hint: "Open inventory", aliases: ["/i"] },
  { name: "/stats", hint: "Open character sheet", aliases: ["/s"] },
  { name: "/map", hint: "Open known locations", aliases: ["/m"] },
  { name: "/journal", hint: "Open chapter journal", aliases: ["/j"] },
  { name: "/thoughts", hint: "Inspect NPC private thoughts", aliases: [] },
  { name: "/codex", hint: "Open codex and characters", aliases: ["/characters"] },
  { name: "/fronts", hint: "Open fronts and hooks", aliases: ["/hooks", "/front"] },
  { name: "/investigations", hint: "Open investigations", aliases: [] },
  { name: "/projects", hint: "Open projects", aliases: [] },
  { name: "/btw", hint: "Ask a contextual side question", aliases: [] },
  { name: "/guide", hint: "Store soft future story guidance", aliases: [] },
  { name: "/advance", hint: "Push to the next meaningful beat", aliases: [] },
  { name: "/timeskip", hint: "Jump ahead to a later meaningful moment", aliases: [] },
  { name: "/downtime", hint: "Request a quieter scene", aliases: [] },
  { name: "/narrator", hint: "Speak to the game master", aliases: ["/n"] },
  { name: "/craft", hint: "Open crafting context", aliases: ["/crafting"] },
  { name: "/talk", hint: "Talk to an NPC: /talk <npc> [intent] [message]", aliases: [] },
  { name: "/save", hint: "Prepare a save command", aliases: [] },
  { name: "/load", hint: "Open saved snapshots", aliases: [] },
  { name: "/help", hint: "Show browser command help", aliases: [] },
  { name: "/quit", hint: "Leave from the terminal session menu", aliases: ["/q"] },
] as const;

const tabCommands: Record<string, ModuleTab> = {
  "/inventory": "inventory",
  "/i": "inventory",
  "/stats": "stats",
  "/s": "stats",
  "/characters": "codex",
  "/codex": "codex",
  "/journal": "codex",
  "/j": "codex",
  "/thoughts": "codex",
  "/fronts": "fronts",
  "/front": "fronts",
  "/hooks": "fronts",
  "/investigations": "investigations",
  "/investigation": "investigations",
  "/projects": "projects",
  "/project": "projects",
  "/craft": "inventory",
  "/crafting": "inventory",
  "/history": "history",
  "/achievements": "saves",
  "/a": "saves",
  "/saves": "saves",
  "/map": "history",
  "/m": "history",
};

export const tabHotkeys: Record<string, ModuleTab> = moduleSpecs.reduce<Record<string, ModuleTab>>((acc, item) => {
  acc[item.hotkey.toLowerCase()] = item.tab;
  return acc;
}, {});

const talkIntents = new Set(["ask", "probe", "bond", "bargain", "threaten", "promise", "lie", "confess"]);

export function commandToAction(rawText: string): CommandResult {
  const text = rawText.trim();
  const lower = text.toLowerCase();
  if (!lower.startsWith("/")) return {};

  if (tabCommands[lower]) {
    return { handled: true, tab: tabCommands[lower] };
  }
  if (lower === "/help") {
    return { handled: true, overlay: "help" };
  }
  if (lower === "/load") {
    return { handled: true, tab: "saves", overlay: "saves" };
  }
  if (lower.startsWith("/save")) {
    const name = text.slice("/save".length).trim();
    return {
      handled: true,
      tab: "saves",
      overlay: "saves",
      notice: name
        ? `Save command prepared as "${name}". Dedicated browser save endpoint is not exposed yet.`
        : "Saved snapshots are visible here. Type a save name after /save when the gateway exposes save writes.",
    };
  }
  if (lower === "/quit" || lower === "/q") {
    return { handled: true, notice: "Quit remains a terminal session-menu action. Browser realtime sync stays connected." };
  }
  if (lower.startsWith("/btw")) {
    const question = text.slice("/btw".length).trim();
    if (!question) return { handled: true, notice: "Usage: /btw <question>" };
    return { text: `[BTW] ${question}` };
  }
  if (lower.startsWith("/guide")) {
    const guidance = text.slice("/guide".length).trim();
    if (!guidance) return { handled: true, notice: "Usage: /guide <guidance>" };
    return { text: `[Guide] ${guidance}` };
  }
  if (lower.startsWith("/narrator") || lower.startsWith("/n ")) {
    const prefix = lower.startsWith("/n ") ? "/n" : "/narrator";
    const message = text.slice(prefix.length).trim();
    if (!message) return { handled: true, notice: "Usage: /narrator <message>" };
    return { text: `[Narrator] ${message}` };
  }
  if (lower.startsWith("/advance")) {
    return { text: buildAdvanceSceneAction(text.slice("/advance".length).trim()) };
  }
  if (lower.startsWith("/timeskip")) {
    return { text: buildTimeSkipAction(text.slice("/timeskip".length).trim()) };
  }
  if (lower.startsWith("/downtime")) {
    const hint = text.slice("/downtime".length).trim();
    if (!hint) return { handled: true, notice: "Usage: /downtime <focus>" };
    return { text: `[Downtime Scene] ${hint}` };
  }
  if (lower.startsWith("/talk")) {
    return talkCommandToAction(text);
  }
  return {};
}

export function actionModeToText(mode: string, text: string): string {
  const clean = text.trim();
  if (mode === "advance") return buildAdvanceSceneAction(clean);
  if (mode === "timeskip") return buildTimeSkipAction(clean);
  if (mode === "talk" && !clean.toLowerCase().startsWith("/talk")) return `[Talk] ${clean}`;
  return clean;
}

function talkCommandToAction(text: string): CommandResult {
  const parts = text.trim().split(/\s+/).slice(1);
  if (!parts.length) {
    return { handled: true, tab: "codex", notice: "Use /talk <npc> [intent] [message]. Known NPCs are in Codex." };
  }
  const target = parts.shift() ?? "";
  let intent = "ask";
  if (parts.length && talkIntents.has(parts[0].toLowerCase())) {
    intent = parts.shift()!.toLowerCase();
  }
  const message = parts.join(" ").trim();
  if (!message) {
    return { handled: true, tab: "codex", notice: `Talk target set: ${target} (${intent}). Add a message to send it.` };
  }
  return { text: `[Talk to ${target} | intent:${intent}] ${message}` };
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
