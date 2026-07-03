import { describe, expect, it } from "vitest";
import { stepHistoryIndex } from "./history";
import type { RecentCommand } from "./types";

const commands: RecentCommand[] = [
  { id: "1", text: "open door", turn: 3, source: "history" },
  { id: "2", text: "look around", turn: 2, source: "history" },
];

describe("stepHistoryIndex", () => {
  it("returns null when command history is empty", () => {
    expect(stepHistoryIndex(-1, -1, [])).toEqual({ index: -1, value: null });
  });

  it("walks backward through recent commands with ArrowUp semantics", () => {
    expect(stepHistoryIndex(-1, -1, commands)).toEqual({ index: 0, value: "open door" });
    expect(stepHistoryIndex(0, -1, commands)).toEqual({ index: 1, value: "look around" });
    expect(stepHistoryIndex(1, -1, commands)).toEqual({ index: 1, value: "look around" });
  });

  it("walks forward and clears the draft after the newest command", () => {
    expect(stepHistoryIndex(1, 1, commands)).toEqual({ index: 0, value: "open door" });
    expect(stepHistoryIndex(0, 1, commands)).toEqual({ index: -1, value: "" });
    expect(stepHistoryIndex(-1, 1, commands)).toEqual({ index: -1, value: "" });
  });
});
