import type { ChapterView, MessageView } from "../../types";

export type HistoryScope = "all" | "current" | string;

export interface HistoryFilters { query: string; type: string; scope: HistoryScope; group: string; }
export interface HistoryGroup { id: string; label: string; }
export type HistoryAction = "jump" | "fork" | "map" | "codex" | "asset";

export function eventType(message: MessageView): string { return message.message_type || message.role || "unknown"; }

export function historyGroups(chapters: ChapterView[]): HistoryGroup[] {
  return chapters.map((chapter) => ({ id: String(chapter.id), label: chapter.title || `Chapter ${chapter.chapter_number}` }));
}

export function messageMatchesFilters(message: MessageView, filters: HistoryFilters, chapters: ChapterView[], activeBranchId?: string): boolean {
  const query = filters.query.trim().toLocaleLowerCase();
  const matchesQuery = !query || [message.content, message.role, message.message_type, String(message.turn)].some((value) => value.toLocaleLowerCase().includes(query));
  const matchesType = !filters.type || eventType(message) === filters.type;
  const matchesScope = filters.scope === "all" || (filters.scope === "current" && (!activeBranchId || message.branch_id === activeBranchId)) || message.branch_id === filters.scope;
  const chapter = chapters.find((item) => message.turn >= item.start_turn && message.turn <= (item.end_turn ?? Number.MAX_SAFE_INTEGER));
  return matchesQuery && matchesType && matchesScope && (!filters.group || chapter?.id === Number(filters.group));
}

export function availableHistoryActions(actions: Partial<Record<HistoryAction, unknown>>): HistoryAction[] {
  return (["jump", "fork", "map", "codex", "asset"] as HistoryAction[]).filter((action) => Boolean(actions[action]));
}
