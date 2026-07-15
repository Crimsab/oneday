import type { DesktopRailMode } from "../../types";

export type RailPresentation = DesktopRailMode | "hidden";

export function railPresentation(visible: boolean, mode: DesktopRailMode): RailPresentation {
  return visible ? mode : "hidden";
}

export function toggleDesktopRailMode(mode: DesktopRailMode): DesktopRailMode {
  return mode === "expanded" ? "collapsed" : "expanded";
}

export function activeStoryCount(stories: ReadonlyArray<{ is_archived: boolean }>): number {
  return stories.reduce((count, story) => count + (story.is_archived ? 0 : 1), 0);
}
