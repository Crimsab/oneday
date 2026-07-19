import { describe, expect, it } from "vitest";
import { availableHistoryActions, messageMatchesFilters } from "./historyTimeline";
import type { ChapterView, MessageView } from "../../types";

const chapters = [{ id: 4, title: "Harbor", chapter_number: 2, start_turn: 3, end_turn: 5 }] as ChapterView[];
const message = { id: 1, content: "The harbor bell rings", turn: 4, role: "assistant", message_type: "narrative", branch_id: "main" } as MessageView;

describe("history timeline filters", () => {
  it("combines real event type, chapter, branch scope, and search fields", () => {
    expect(messageMatchesFilters(message, { query: "bell", type: "narrative", scope: "current", group: "4" }, chapters, "main")).toBe(true);
    expect(messageMatchesFilters(message, { query: "bell", type: "narrative", scope: "current", group: "4" }, chapters, "alternate")).toBe(false);
    expect(messageMatchesFilters(message, { query: "", type: "user", scope: "all", group: "" }, chapters)).toBe(false);
    expect(messageMatchesFilters(message, { query: "turn 4", type: "", scope: "all", group: "" }, chapters)).toBe(false);
  });

  it("only exposes actions backed by parent callbacks", () => {
    expect(availableHistoryActions({ jump: () => undefined, codex: () => undefined })).toEqual(["jump", "codex"]);
    expect(availableHistoryActions({})).toEqual([]);
  });
});
