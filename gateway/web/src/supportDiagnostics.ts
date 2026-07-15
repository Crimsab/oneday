import type { AppPreferences, ModelSettings, StorySnapshot } from "./types";

export type SupportEventLevel = "info" | "warning" | "error";

export interface SupportEvent {
  timestamp: string;
  level: SupportEventLevel;
  source: "browser" | "console" | "api" | "application";
  message: string;
  detail?: string;
}

const MAX_EVENTS = 200;
const MAX_TEXT_LENGTH = 2_000;
const events: SupportEvent[] = [];
let installed = false;

export function installSupportDiagnostics(): void {
  if (installed || typeof window === "undefined") return;
  installed = true;
  recordSupportEvent("info", "application", "OneDay web client started");

  window.addEventListener("error", (event) => {
    recordSupportEvent("error", "browser", event.message || "Uncaught browser error", `${event.filename || "unknown"}:${event.lineno || 0}:${event.colno || 0}`);
  });
  window.addEventListener("unhandledrejection", (event) => {
    recordSupportEvent("error", "browser", "Unhandled promise rejection", serializeForSupport(event.reason));
  });
  window.addEventListener("online", () => recordSupportEvent("info", "browser", "Browser connection restored"));
  window.addEventListener("offline", () => recordSupportEvent("warning", "browser", "Browser connection lost"));

  for (const level of ["warn", "error"] as const) {
    const original = console[level].bind(console);
    console[level] = (...values: unknown[]) => {
      recordSupportEvent(level === "warn" ? "warning" : "error", "console", values.map(serializeForSupport).join(" "));
      original(...values);
    };
  }
}

export function recordApiSupportEvent(method: string, path: string, status: number, durationMs: number, message = ""): void {
  const level: SupportEventLevel = status === 0 || status >= 500 ? "error" : status >= 400 ? "warning" : "info";
  recordSupportEvent(level, "api", `${method.toUpperCase()} ${safeApiPath(path)} -> ${status}`, `${Math.max(0, Math.round(durationMs))} ms${message ? `; ${message}` : ""}`);
}

export function recordSupportEvent(level: SupportEventLevel, source: SupportEvent["source"], message: unknown, detail?: unknown): void {
  events.push({
    timestamp: new Date().toISOString(),
    level,
    source,
    message: redactSupportText(serializeForSupport(message)),
    ...(detail === undefined ? {} : { detail: redactSupportText(serializeForSupport(detail)) }),
  });
  if (events.length > MAX_EVENTS) events.splice(0, events.length - MAX_EVENTS);
}

export function getSupportEvents(): SupportEvent[] {
  return events.map((event) => ({ ...event }));
}

export function clearSupportEvents(): void {
  events.length = 0;
  recordSupportEvent("info", "application", "Support log cleared");
}

export function buildSupportBundle({
  preferences,
  snapshot,
  modelSettings,
}: {
  preferences: AppPreferences;
  snapshot: StorySnapshot | null;
  modelSettings: ModelSettings | null;
}) {
  return {
    schema_version: 1,
    generated_at: new Date().toISOString(),
    application: {
      name: "OneDay",
      location: typeof window === "undefined" ? "" : `${window.location.origin}${window.location.pathname}`,
      language: preferences.locale,
    },
    environment: typeof navigator === "undefined" ? {} : {
      user_agent: navigator.userAgent,
      platform: navigator.platform,
      language: navigator.language,
      online: navigator.onLine,
      viewport: typeof window === "undefined" ? "" : `${window.innerWidth}x${window.innerHeight}@${window.devicePixelRatio}`,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    },
    story_context: snapshot ? {
      selected: true,
      turn: snapshot.world.current_turn,
      revision: snapshot.version.revision,
      branch_id: snapshot.messages.at(-1)?.branch_id || undefined,
      message_count: snapshot.messages.length,
    } : { selected: false },
    routing: modelSettings ? {
      config_revision: modelSettings.config_revision,
      active: modelSettings.active,
      providers: modelSettings.providers.map(({ id, enabled, model, reasoning }) => ({ id, enabled, model, reasoning })),
      image_providers: modelSettings.image_providers.map(({ id, configured, status }) => ({ id, configured, status })),
      tts_status: modelSettings.tts_status,
    } : null,
    preferences,
    logs: getSupportEvents(),
  };
}

export function redactSupportText(value: string): string {
  return value
    .replace(/(authorization\s*[:=]\s*bearer\s+)[^\s,;]+/gi, "$1[REDACTED]")
    .replace(/(\bbearer\s+)[a-z0-9._~+/=-]+/gi, "$1[REDACTED]")
    .replace(/([?&](?:api[_-]?key|access[_-]?token|refresh[_-]?token|token|secret|password)=)[^&#\s]+/gi, "$1[REDACTED]")
    .replace(/(["']?(?:api[_-]?key|access[_-]?token|refresh[_-]?token|auth[_-]?token|token|secret|password)["']?\s*[:=]\s*["']?)[^"'\s,;}]+/gi, "$1[REDACTED]")
    .slice(0, MAX_TEXT_LENGTH);
}

function safeApiPath(path: string): string {
  const pathname = path.split("?", 1)[0];
  return pathname.replace(/(\/api\/stories\/)[^/]+/i, "$1:story");
}

function serializeForSupport(value: unknown): string {
  if (typeof value === "string") return value;
  if (value instanceof Error) return `${value.name}: ${value.message}${value.stack ? `\n${value.stack}` : ""}`;
  try {
    return JSON.stringify(value, circularReplacer());
  } catch {
    return String(value);
  }
}

function circularReplacer() {
  const seen = new WeakSet<object>();
  return (_key: string, value: unknown) => {
    if (!value || typeof value !== "object") return value;
    if (seen.has(value)) return "[Circular]";
    seen.add(value);
    return value;
  };
}
