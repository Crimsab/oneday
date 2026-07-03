import { describe, expect, it } from "vitest";
import {
  asArray,
  asObject,
  compactText,
  deriveCondition,
  displayClock,
  entryLabel,
  fieldRows,
  findString,
  messageClock,
  numericStat,
  recentFromMessages,
  titleCase,
  valueToText,
  weatherLabel,
} from "./format";
import type { JsonValue, MessageView, StorySnapshot } from "./types";

describe("basic JSON helpers", () => {
  it("normalizes objects, arrays, and printable values", () => {
    expect(asObject({ a: 1 })).toEqual({ a: 1 });
    expect(asObject(["x"])).toEqual({});
    expect(asArray(["x"])).toEqual(["x"]);
    expect(asArray({ a: 1 })).toEqual([]);
    expect(valueToText("")).toBe("-");
    expect(valueToText({ a: 1 })).toBe(JSON.stringify({ a: 1 }, null, 2));
  });

  it("formats labels and compact text", () => {
    expect(titleCase("known_location-id")).toBe("Known Location Id");
    expect(compactText("  one \n two  three  ", 40)).toBe("one two three");
    expect(compactText("abcdef", 4)).toBe("abc...");
  });
});

describe("clock and stat helpers", () => {
  it("derives display clocks from turns and messages", () => {
    expect(displayClock(0)).toEqual({ day: 1, time: "Day 1, 08:00", cycle: "Morning" });
    expect(displayClock(24).day).toBe(2);
    expect(messageClock(message({ id: 5, turn: 2 }))).toBe("09:19");
  });

  it("clamps and parses numeric stats", () => {
    expect(numericStat(42.4)).toBe(42);
    expect(numericStat("99.6")).toBe(100);
    expect(numericStat(180)).toBe(100);
    expect(numericStat(-20)).toBe(0);
    expect(numericStat("nope")).toBeNull();
  });
});

describe("snapshot derivation helpers", () => {
  it("derives conditions from character stats", () => {
    expect(deriveCondition(null)).toBe("Idle");
    expect(deriveCondition(snapshotWithStats({ health: 20 }))).toBe("Injured");
    expect(deriveCondition(snapshotWithStats({ stamina: 12 }))).toBe("Exhausted");
    expect(deriveCondition(snapshotWithStats({ focus: 70 }))).toBe("Focused");
    expect(deriveCondition(snapshotWithStats({ focus: 20 }))).toBe("Stable");
  });

  it("finds nested strings and weather labels", () => {
    expect(findString({ outer: [{ sky: "Overcast" }] }, ["sky"])).toBe("Overcast");
    expect(findString({ outer: [{ sky: "" }] }, ["sky"])).toBeNull();
    expect(weatherLabel(snapshotWithWorld({ scene_contract: { weather: "Rain" } }))).toBe("Rain");
    expect(weatherLabel(snapshotWithWorld({ known_locations: { forecast: "Clear" } }))).toBe("Clear");
  });

  it("builds recent command history from user messages", () => {
    const commands = recentFromMessages([
      message({ id: 1, role: "assistant", content: "no" }),
      message({ id: 2, role: "user", content: " look " }),
      message({ id: 3, role: "user", content: "open door" }),
    ]);
    expect(commands.map((command) => command.text)).toEqual(["open door", "look"]);
  });

  it("labels entries and field rows", () => {
    expect(entryLabel({ title: "A lead", id: "x" }, 0)).toBe("A lead");
    expect(entryLabel("plain", 1)).toBe("Item 2");
    expect(fieldRows(["a", "b"])).toEqual([
      ["Item 1", "a"],
      ["Item 2", "b"],
    ]);
    expect(fieldRows({ known_place: "Docks" })).toEqual([["Known Place", "Docks"]]);
    expect(fieldRows("raw")).toEqual([["Value", "raw"]]);
  });
});

function message(overrides: Partial<MessageView>): MessageView {
  return {
    id: 1,
    session_id: "session",
    story_id: "story",
    turn: 0,
    role: "user",
    content: "look",
    message_type: "narrative",
    metadata: {},
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function snapshotWithStats(stats: Record<string, JsonValue>): StorySnapshot {
  return snapshotWithWorld({ characterStats: stats });
}

function snapshotWithWorld(overrides: Record<string, JsonValue>): StorySnapshot {
  const characterStats = overrides.characterStats ?? { focus: 80 };
  return {
    server_time: "2026-01-01T00:00:00Z",
    version: { turn: 1, last_message_id: 1, world_updated_at: "now", achievement_count: 0, save_count: 0 },
    story: { id: "story", name: "Story", description: "", genre: "", tone: "", language: "en", is_archived: false, updated_at: "now" },
    character: { id: "character", name: "Hero", fields: { stats: characterStats } },
    world: {
      id: "world",
      current_location: "Dock",
      current_chapter: 1,
      current_turn: 1,
      known_locations: (overrides.known_locations as StorySnapshot["world"]["known_locations"]) ?? {},
      global_events: {},
      faction_standings: {},
      story_hooks: {},
      world_reactions: {},
      investigations: {},
      projects: {},
      guidance: {},
      fronts: {},
      timeline: {},
      scene_contract: (overrides.scene_contract as StorySnapshot["world"]["scene_contract"]) ?? {},
      updated_at: "now",
    },
    active_session: { id: "session", story_id: "story", started_at: "now", ended_at: null, summary: "" },
    choices: [],
    messages: [],
    panels: { chapters: [], achievements: [], npcs: [], sessions: [], saves: [] },
  };
}
