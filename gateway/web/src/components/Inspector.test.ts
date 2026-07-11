import { describe, expect, it } from "vitest";
import { cardsFromValue, isLongInspectorRow, isPlayerHiddenField, meterRows, moduleTitle, npcDiscoverySummary, npcRelationSummary, sanitizePlayerVisibleValue } from "./Inspector";
import type { JsonValue, RecordView, StorySnapshot } from "../types";

describe("moduleTitle", () => {
  it("returns the visible label for module tabs", () => {
    expect(moduleTitle("inventory")).toBe("Inventory");
    expect(moduleTitle("craft")).toBe("Craft");
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
    expect(cardsFromValue({}, "Empty")).toEqual([]);
    expect(cardsFromValue("plain", "Value")).toEqual([{ title: "Value", rows: [["Value", "plain"]] }]);
  });

  it("filters player-hidden fields from cards and raw state", () => {
    expect(isPlayerHiddenField("Private Thoughts")).toBe(true);
    expect(isPlayerHiddenField("notes_on_protagonist")).toBe(true);
    expect(isPlayerHiddenField("Appearance")).toBe(false);
    expect(cardsFromValue([{ name: "Maren", private_thoughts: "betray them", appearance: "orange jacket" }], "Character")).toEqual([
      {
        title: "Maren",
        rows: [["Appearance", "orange jacket"]],
      },
    ]);
    expect(sanitizePlayerVisibleValue({ npcs: [{ name: "Maren", desires: ["hidden"], nested: { nemesis_json: { status: "active" } } }] })).toEqual({
      npcs: [{ name: "Maren", nested: {} }],
    });
  });
});

describe("isLongInspectorRow", () => {
  it("promotes prose-heavy inspector values to a full-width row", () => {
    expect(isLongInspectorRow("Text", "Short but narrative text")).toBe(true);
    expect(isLongInspectorRow("Details", "Station concourse, shared tables, overlapping announcements.")).toBe(true);
    expect(isLongInspectorRow("Location", "Stazione Centrale - atrio principale")).toBe(false);
    expect(isLongInspectorRow("Flag", "true")).toBe(false);
  });
});

describe("meterRows", () => {
  it("supports vitals and scalar 0-100 stats", () => {
    const rows = meterRows(snapshotWithStats({ vitals: { hp: { current: 15, max: 30 } }, focus: 82, currency: 10, deaths: 0 }));
    expect(rows).toEqual([
      { label: "Hp", value: 50, text: "15/30" },
      { label: "Focus", value: 82, text: "82/100" },
    ]);
  });
});

describe("npcRelationSummary", () => {
  it("uses explicit relationship labels without hardcoding NPC names", () => {
    expect(npcRelationSummary(npcRecord({ relationship: { status: "trusted ally" }, disposition: 92 }))).toMatchObject({
      label: "Trusted Ally",
      score: 92,
      tone: "ally",
      filledSegments: 9,
    });
  });

  it("falls back to score thresholds when no label exists", () => {
    expect(npcRelationSummary(npcRecord({ disposition: 22 }))).toMatchObject({
      label: "Hostile",
      score: 22,
      tone: "hostile",
      filledSegments: 2,
    });
    expect(npcRelationSummary(npcRecord({ relationship: { trust: 54 } }))).toMatchObject({
      label: "Neutral",
      score: 54,
      tone: "neutral",
      filledSegments: 5,
    });
  });
});

describe("npcDiscoverySummary", () => {
  it("surfaces progressive discovery and visual readiness", () => {
    expect(
      npcDiscoverySummary(
        npcRecord({
          discovery: {
            stage: "rumor",
            public_label: "Marek",
            visual_readiness: "none",
          },
        }),
      ),
    ).toEqual({
      stage: "rumor",
      label: "Rumor",
      publicLabel: "Marek",
      visualReadiness: "none",
      visualLabel: "",
    });
    expect(
      npcDiscoverySummary(
        npcRecord({
          discovery_stage: "established",
          public_label: "Serra Vale",
          visual_readiness: "canonical",
        }),
      ),
    ).toMatchObject({
      label: "Established",
      publicLabel: "Serra Vale",
      visualLabel: "Canonical",
    });
  });
});

function npcRecord(fields: Record<string, JsonValue>): RecordView {
  return { id: "npc", name: "Test NPC", fields };
}

function snapshotWithStats(stats: JsonValue): StorySnapshot {
  return {
    server_time: "2026-01-01T00:00:00Z",
    version: {
      turn: 1,
      revision: 0,
      story_updated_at: "now",
      active_session_id: "session",
      last_message_id: 1,
      world_updated_at: "now",
      character_updated_at: "now",
      npc_count: 0,
      npc_updated_at: "",
      chapter_count: 0,
      achievement_count: 0,
      latest_achievement_at: "",
      save_count: 0,
      latest_save_at: "",
      visual_asset_updated_at: "",
      visual_job_updated_at: "",
      active_visual_job_count: 0,
    },
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
