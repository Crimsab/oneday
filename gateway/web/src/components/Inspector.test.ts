import { describe, expect, it } from "vitest";
import { cardsFromValue, meterRows, moduleTitle } from "./Inspector";
import type { JsonValue, StorySnapshot } from "../types";

describe("moduleTitle", () => {
  it("returns the visible label for module tabs", () => {
    expect(moduleTitle("inventory")).toBe("Inventory");
    expect(moduleTitle("fronts")).toBe("Fronts");
  });
});

describe("cardsFromValue", () => {
  it("renders arrays as cards", () => {
    expect(cardsFromValue([{ name: "Ledger", type: "clue", detail: "wet ink" }], "Item")).toEqual([
      {
        title: "Ledger",
        rows: [
          ["Type", "clue"],
          ["Detail", "wet ink"],
        ],
      },
    ]);
  });

  it("renders primitive objects as a single detail card", () => {
    expect(cardsFromValue({ status: "active", timer_turns: 3 }, "Hook")).toEqual([
      {
        title: "Hook",
        rows: [
          ["Status", "active"],
          ["Timer Turns", "3"],
        ],
      },
    ]);
  });

  it("renders nested objects as separate cards", () => {
    expect(cardsFromValue({ lead_a: { title: "Harbor", status: "active" } }, "Lead")).toEqual([
      {
        title: "Lead A",
        rows: [["Status", "active"]],
      },
    ]);
  });

  it("handles empty and primitive values", () => {
    expect(cardsFromValue(undefined, "Empty")).toEqual([]);
    expect(cardsFromValue("plain", "Value")).toEqual([{ title: "Value", rows: [["Value", "plain"]] }]);
  });
});

describe("meterRows", () => {
  it("supports vitals and scalar 0-100 stats", () => {
    const rows = meterRows(snapshotWithStats({ vitals: { hp: { current: 15, max: 30 } }, focus: 82 }));
    expect(rows).toEqual([
      { label: "Hp", value: 50, text: "15/30" },
      { label: "Focus", value: 82, text: "82/100" },
    ]);
  });
});

function snapshotWithStats(stats: JsonValue): StorySnapshot {
  return {
    server_time: "2026-01-01T00:00:00Z",
    version: { turn: 1, last_message_id: 1, world_updated_at: "now", achievement_count: 0, save_count: 0 },
    story: { id: "story", name: "Story", description: "", genre: "", tone: "", language: "en", is_archived: false, updated_at: "now" },
    character: { id: "character", name: "Hero", fields: { stats } },
    world: {
      id: "world",
      current_location: "Dock",
      current_chapter: 1,
      current_turn: 1,
      known_locations: {},
      global_events: {},
      faction_standings: {},
      story_hooks: {},
      world_reactions: {},
      investigations: {},
      projects: {},
      guidance: {},
      fronts: {},
      timeline: {},
      scene_contract: {},
      updated_at: "now",
    },
    active_session: { id: "session", story_id: "story", started_at: "now", ended_at: null, summary: "" },
    choices: [],
    messages: [],
    panels: { chapters: [], achievements: [], npcs: [], sessions: [], saves: [] },
  };
}
