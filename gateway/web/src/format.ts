import type { JsonObject, JsonValue, MessageView, RecentCommand, StorySnapshot } from "./types";

export function asObject(value: JsonValue | undefined): JsonObject {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as JsonObject) : {};
}

export function asArray(value: JsonValue | undefined): JsonValue[] {
  return Array.isArray(value) ? value : [];
}

export function valueToText(value: JsonValue | undefined, fallback = "-"): string {
  if (value === null || value === undefined || value === "") return fallback;
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) return arrayToText(value, fallback);
  return objectToText(value as JsonObject, fallback);
}

export function titleCase(value: string): string {
  return value
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/\b\w/g, (char) => char.toUpperCase());
}

export function compactText(value: string, max = 120): string {
  const text = value.replace(/\s+/g, " ").trim();
  if (text.length <= max) return text;
  return `${text.slice(0, max - 1).trim()}...`;
}

export function displayClock(turn = 0) {
  const day = Math.max(1, Math.floor(turn / 24) + 1);
  const hour = (8 + Math.floor(turn / 2)) % 24;
  const minute = (turn * 7) % 60;
  const cycle = hour < 6 ? "Night" : hour < 12 ? "Morning" : hour < 18 ? "Afternoon" : "Evening";
  return {
    day,
    time: `Day ${day}, ${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}`,
    cycle,
  };
}

export function messageClock(message: MessageView): string {
  const hour = (8 + Math.floor(message.turn / 2)) % 24;
  const minute = (message.turn * 7 + message.id) % 60;
  return `${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}`;
}

export function deriveCondition(snapshot: StorySnapshot | null): string {
  const stats = asObject(snapshot?.character.fields.stats);
  const health = numericStat(stats.health ?? stats.Health);
  const stamina = numericStat(stats.stamina ?? stats.Stamina);
  const focus = numericStat(stats.focus ?? stats.Focus);
  if (health !== null && health < 35) return "Injured";
  if (stamina !== null && stamina < 30) return "Exhausted";
  if (focus !== null && focus >= 65) return "Focused";
  return snapshot ? "Stable" : "Idle";
}

export function numericStat(value: JsonValue | undefined): number | null {
  if (typeof value === "number" && Number.isFinite(value)) return Math.max(0, Math.min(100, Math.round(value)));
  if (typeof value === "string") {
    const parsed = Number.parseFloat(value);
    if (Number.isFinite(parsed)) return Math.max(0, Math.min(100, Math.round(parsed)));
  }
  return null;
}

export function findString(value: JsonValue | undefined, keys: string[]): string | null {
  if (!value || typeof value !== "object") return null;
  const stack: JsonValue[] = [value];
  const wanted = new Set(keys.map((key) => key.toLowerCase()));
  while (stack.length) {
    const current = stack.shift();
    if (!current || typeof current !== "object") continue;
    if (Array.isArray(current)) {
      stack.push(...current);
      continue;
    }
    for (const [key, child] of Object.entries(current)) {
      if (wanted.has(key.toLowerCase()) && typeof child === "string" && child.trim()) {
        return child.trim();
      }
      if (child && typeof child === "object") stack.push(child);
    }
  }
  return null;
}

export function weatherLabel(snapshot: StorySnapshot | null): string {
  return (
    findString(snapshot?.world.scene_contract, ["weather", "forecast", "sky"]) ??
    findString(snapshot?.world.known_locations, ["weather", "forecast", "sky"]) ??
    "Untracked"
  );
}

export function recentFromMessages(messages: MessageView[]): RecentCommand[] {
  return messages
    .filter((message) => message.role === "user" && message.content.trim())
    .slice(-8)
    .reverse()
    .map((message) => ({
      id: `history-${message.id}`,
      text: message.content.trim(),
      turn: message.turn,
      source: "history" as const,
    }));
}

export function entryLabel(value: JsonValue, index: number): string {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    const object = value as JsonObject;
    for (const key of ["title", "name", "id", "label", "status"]) {
      const candidate = object[key];
      if (typeof candidate === "string" && candidate.trim()) return candidate;
    }
  }
  return `Item ${index + 1}`;
}

export function fieldRows(value: JsonValue | undefined): Array<[string, string]> {
  if (Array.isArray(value)) {
    return value.map((item, index) => [entryLabel(item, index), valueToText(item)]);
  }
  if (value && typeof value === "object") {
    return Object.entries(value as JsonObject).map(([key, child]) => [titleCase(key), valueToText(child)]);
  }
  if (value !== undefined && value !== null) return [["Value", valueToText(value)]];
  return [];
}

function arrayToText(value: JsonValue[], fallback: string): string {
  if (value.length === 0) return fallback;
  const primitive = value.every((item) => item === null || ["string", "number", "boolean"].includes(typeof item));
  if (primitive) return value.map((item) => valueToText(item, fallback)).join(", ");

  const labels = value.slice(0, 4).map((item, index) => {
    if (item && typeof item === "object" && !Array.isArray(item)) {
      const object = item as JsonObject;
      const label = entryLabel(item, index);
      const summary = objectSummary(object, ["name", "title", "label", "id"]);
      return summary ? `${label} (${summary})` : label;
    }
    return valueToText(item, fallback);
  });
  const suffix = value.length > labels.length ? `, +${value.length - labels.length} more` : "";
  return `${labels.join(", ")}${suffix}`;
}

function objectToText(value: JsonObject, fallback: string): string {
  const summary = objectSummary(value);
  if (summary) return summary;
  const keys = Object.keys(value);
  if (keys.length === 0) return fallback;
  return keys.slice(0, 5).map((key) => titleCase(key)).join(", ");
}

function objectSummary(value: JsonObject, skipKeys: string[] = []): string {
  const skip = new Set(skipKeys.map((key) => key.toLowerCase()));
  const preferred = [
    "name",
    "title",
    "label",
    "status",
    "type",
    "kind",
    "summary",
    "description",
    "detail",
    "details",
    "note",
    "value",
    "amount",
    "current",
    "max",
    "progress",
    "outcome",
    "risk",
    "intent",
  ];
  const pieces: string[] = [];
  for (const key of preferred) {
    if (skip.has(key.toLowerCase())) continue;
    const child = value[key] ?? value[titleCase(key)];
    if (child === undefined || child === null || child === "") continue;
    if (typeof child === "object") {
      if (Array.isArray(child) && child.length > 0 && child.every((item) => item === null || ["string", "number", "boolean"].includes(typeof item))) {
        pieces.push(`${titleCase(key)}: ${arrayToText(child, "-")}`);
      }
      continue;
    }
    pieces.push(`${titleCase(key)}: ${String(child)}`);
    if (pieces.length >= 4) break;
  }
  if (pieces.length > 0) return pieces.join("; ");

  for (const [key, child] of Object.entries(value)) {
    if (skip.has(key.toLowerCase())) continue;
    if (child === undefined || child === null || child === "") continue;
    if (typeof child === "object") continue;
    pieces.push(`${titleCase(key)}: ${String(child)}`);
    if (pieces.length >= 4) break;
  }
  return pieces.join("; ");
}
