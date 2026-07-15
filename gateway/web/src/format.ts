import type { JsonObject, JsonValue, MessageView, RecentCommand, StorySnapshot } from "./types";
import i18n, { formatInterfaceDateTime } from "./i18n";

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

export function readableStructuredText(value: string, fallback = ""): string {
  const text = value.trim();
  if (!text) return fallback;

  const jsonText = extractJSONText(text);
  if (!jsonText) return text;

  try {
    const parsed = JSON.parse(jsonText) as JsonValue;
    const readable = readableJSONValue(parsed);
    return readable || fallback || text;
  } catch {
    return text;
  }
}

export function displayClock(snapshot: StorySnapshot | null) {
  const clock = asObject(snapshot?.world.world_time);
  const day = typeof clock.day === "number" ? clock.day : null;
  const minuteOfDay = typeof clock.minute_of_day === "number" ? clock.minute_of_day : null;
  const cycle = minuteOfDay === null ? i18n.t("common:notTracked") : minuteOfDay < 360 ? i18n.t("format:night") : minuteOfDay < 720 ? i18n.t("format:morning") : minuteOfDay < 1080 ? i18n.t("format:afternoon") : i18n.t("format:evening");
  return {
    day,
	time: typeof clock.display_text === "string" && clock.display_text.trim() ? clock.display_text : i18n.t("common:notTracked"),
    cycle,
  };
}

export function displayTimestamp(value: string | undefined, fallback = "-"): string {
  const text = (value ?? "").trim();
  if (!text) return fallback;
  const cleaned = text
    .replace(/\s+m=\+[0-9.]+s?$/i, "")
    .replace(/\s+m=\+[^\s]+$/i, "")
  const parsed = /^\d{4}-\d{2}-\d{2}T/.test(cleaned) ? new Date(cleaned) : null;
  if (parsed && !Number.isNaN(parsed.getTime())) return formatInterfaceDateTime(parsed);
  return cleaned.replace(/^(\d{4}-\d{2}-\d{2})T/, "$1 ").replace(/Z$/, " UTC");
}

export function messageClock(message: MessageView): string {
  return `T${message.turn}`;
}

export function deriveCondition(snapshot: StorySnapshot | null): string {
	if (!snapshot) return i18n.t("common:notTracked");
	return findString(snapshot.character.fields, ["condition", "status"]) ?? i18n.t("common:notTracked");
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
	const weather = asObject(snapshot?.world.weather);
	return typeof weather.label === "string" && weather.label.trim() ? weather.label : i18n.t("common:notTracked");
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
  return i18n.t("common:item", { number: index + 1 });
}

export function fieldRows(value: JsonValue | undefined): Array<[string, string]> {
  if (Array.isArray(value)) {
    return value.map((item, index) => [entryLabel(item, index), valueToText(item)]);
  }
  if (value && typeof value === "object") {
    return Object.entries(value as JsonObject).map(([key, child]) => [titleCase(key), valueToText(child)]);
  }
  if (value !== undefined && value !== null) return [[i18n.t("format_extra:fields.value"), valueToText(value)]];
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
  const suffix = value.length > labels.length ? `, ${i18n.t("common:more", { count: value.length - labels.length })}` : "";
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
        pieces.push(`${localizedFieldLabel(key)}: ${arrayToText(child, "-")}`);
      }
      continue;
    }
    pieces.push(`${localizedFieldLabel(key)}: ${String(child)}`);
    if (pieces.length >= 4) break;
  }
  if (pieces.length > 0) return pieces.join("; ");

  for (const [key, child] of Object.entries(value)) {
    if (skip.has(key.toLowerCase())) continue;
    if (child === undefined || child === null || child === "") continue;
    if (typeof child === "object") continue;
    pieces.push(`${localizedFieldLabel(key)}: ${String(child)}`);
    if (pieces.length >= 4) break;
  }
  return pieces.join("; ");
}

function extractJSONText(text: string): string | null {
  const fenced = text.match(/^```(?:json)?\s*([\s\S]*?)\s*```$/i);
  const candidate = (fenced?.[1] ?? text).trim();
  if (!candidate.startsWith("{") && !candidate.startsWith("[")) return null;
  return candidate;
}

function readableJSONValue(value: JsonValue): string {
  if (Array.isArray(value)) return valueToText(value, "");
  if (!value || typeof value !== "object") return valueToText(value, "");

  const object = value as JsonObject;
  const output = asObject(object.output);
  const lines: string[] = [];
  const primary =
    firstReadableString(object, ["narrative", "message", "text", "summary", "description"]) ||
    firstReadableString(output, ["narrative", "message", "text", "summary", "description"]);
  if (primary) lines.push(primary);

  const location = firstReadableString(object, ["location", "place"]);
  const mood = firstReadableString(object, ["mood", "tone"]);
  if (location || mood) {
    lines.push([location ? `${i18n.t("format_extra:fields.location")}: ${location}` : "", mood ? `${i18n.t("format_extra:fields.mood")}: ${mood}` : ""].filter(Boolean).join(" - "));
  }

  const choices = readableChoices(object.choices ?? output.choices);
  if (choices.length > 0) lines.push(`${i18n.t("format_extra:choices")}:\n${choices.map((choice, index) => `${index + 1}. ${choice}`).join("\n")}`);

  if (lines.length > 0) return lines.join("\n\n");
  return fieldRows(object)
    .slice(0, 8)
    .map(([key, child]) => `- ${key}: ${child}`)
    .join("\n");
}

function localizedFieldLabel(key: string): string {
  const normalized = key.toLowerCase();
  const known = new Set(["name", "title", "label", "status", "type", "kind", "summary", "description", "detail", "details", "note", "value", "amount", "current", "max", "progress", "outcome", "risk", "intent", "location", "mood"]);
  return known.has(normalized) ? i18n.t(`format_extra:fields.${normalized}`) : titleCase(key);
}

function firstReadableString(object: JsonObject, keys: string[]): string {
  for (const key of keys) {
    const value = object[key];
    if (typeof value === "string" && value.trim()) return value.trim();
  }
  return "";
}

function readableChoices(value: JsonValue | undefined): string[] {
  if (!Array.isArray(value)) return [];
  return value
    .map((choice, index) => {
      if (typeof choice === "string") return choice.trim();
      if (choice && typeof choice === "object" && !Array.isArray(choice)) {
        const object = choice as JsonObject;
        const text = firstReadableString(object, ["text", "label", "title", "description"]);
        return text || entryLabel(choice, index);
      }
      return valueToText(choice, "").trim();
    })
    .filter(Boolean)
    .slice(0, 6);
}
