import type { RecentCommand } from "./types";

export function stepHistoryIndex(currentIndex: number, direction: -1 | 1, commands: RecentCommand[]) {
  if (commands.length === 0) {
    return { index: -1, value: null };
  }

  const index =
    direction === -1
      ? Math.min(currentIndex + 1, commands.length - 1)
      : Math.max(currentIndex - 1, -1);

  return {
    index,
    value: index === -1 ? "" : commands[index].text,
  };
}
