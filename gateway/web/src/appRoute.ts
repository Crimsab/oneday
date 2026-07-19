import type { ModuleTab, StorySummary } from "./types";

export const storySections = [
  "history",
  "map",
  "codex",
  "inventory",
  "stats",
  "craft",
  "fronts",
  "investigations",
  "projects",
  "achievements",
  "saves",
  "translations",
] as const;

export type StorySection = ModuleTab | "translations";
export type AppRoute =
  | { kind: "library" }
  | { kind: "setup" }
  | { kind: "story"; storyId: string; section: StorySection };

const sectionSet = new Set<string>(storySections);

export function isModuleSection(section: StorySection): section is ModuleTab {
  return section !== "translations";
}

export function parseAppRoute(pathname: string): AppRoute | null {
  const segments = pathname.split("/").filter(Boolean);
  if (segments.length === 1 && segments[0] === "setup") return { kind: "setup" };
  if (segments.length === 1 && segments[0] === "stories") return { kind: "library" };
  if (segments.length !== 3 || segments[0] !== "stories" || !sectionSet.has(segments[2])) return null;
  try {
    const storyId = decodeURIComponent(segments[1]);
    if (!storyId) return null;
    return { kind: "story", storyId, section: segments[2] as StorySection };
  } catch {
    return null;
  }
}

export function appRoutePath(route: AppRoute): string {
  if (route.kind === "library") return "/stories";
  if (route.kind === "setup") return "/setup";
  return `/stories/${encodeURIComponent(route.storyId)}/${route.section}`;
}

export function resolveAppRoute(requested: AppRoute | null, stories: StorySummary[]): AppRoute {
  if (requested?.kind === "setup") return requested;
  if (requested?.kind === "library") return requested;
  if (requested?.kind === "story") {
    const story = stories.find((item) => item.id === requested.storyId);
    if (story && !story.is_archived) return requested;
  }
  const active = stories.find((story) => !story.is_archived);
  return active ? { kind: "story", storyId: active.id, section: "history" } : { kind: "library" };
}

export function sameAppRoute(left: AppRoute, right: AppRoute): boolean {
  return left.kind === right.kind
    && (left.kind === "library" || left.kind === "setup" || (right.kind === "story" && left.storyId === right.storyId && left.section === right.section));
}

export interface OneDayHistoryState {
  oneday?: true;
  returnTo?: string;
}

export function historyReturnRoute(state: unknown): AppRoute | null {
  if (!state || typeof state !== "object") return null;
  const value = state as OneDayHistoryState;
  return value.oneday && typeof value.returnTo === "string" ? parseAppRoute(value.returnTo) : null;
}

export function writeAppRoute(route: AppRoute, mode: "push" | "replace", state: OneDayHistoryState = {}): void {
  const method = mode === "push" ? window.history.pushState : window.history.replaceState;
  method.call(window.history, { ...state, oneday: true }, "", appRoutePath(route));
}
