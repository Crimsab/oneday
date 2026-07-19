import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { HistoryEvent, HistoryReader } from "./HistoryReader";
import type { MessageView, StorySnapshot } from "../types";

describe("HistoryReader", () => {
  it("renders compact, localized filter controls and an accessible empty state", () => {
    const snapshot = { story: { id: "story-1" }, version: { revision: 1 }, world: { current_turn: 2 } } as StorySnapshot;
    const html = renderToStaticMarkup(<HistoryReader snapshot={snapshot} />);
    expect(html).toContain("History filters");
    expect(html).toContain("Search branch history");
    expect(html).toContain("No events match these filters.");
    expect(html).toContain('aria-live="polite"');
    expect(html).toContain("history-filter-grid");
  });

  it("renders only parent-backed actions and marks the canonical current event", () => {
    const message = { id: 3, content: "The bell rings.", turn: 5, role: "assistant", message_type: "narrative", branch_id: "main", source_commit_id: "commit-3" } as MessageView;
    const html = renderToStaticMarkup(<HistoryEvent message={message} currentTurn={5} actions={{ onFork: () => undefined, onOpenCodex: () => undefined }} />);
    expect(html).toContain("is-current");
    expect(html).toContain("Fork");
    expect(html).toContain("Codex");
    expect(html).not.toContain(">Map<");
    expect(html).not.toContain(">Jump<");

    const withoutCommit = renderToStaticMarkup(<HistoryEvent message={{ ...message, source_commit_id: "" }} currentTurn={5} actions={{ onFork: () => undefined }} />);
    expect(withoutCommit).not.toContain(">Fork<");
  });
});
