import { describe, expect, it } from "vitest";
import {
  asArray,
  asObject,
  compactText,
  deriveCondition,
  displayClock,
  displayTimestamp,
  entryLabel,
  fieldRows,
  findString,
  messageClock,
  numericStat,
  readableStructuredText,
  recentFromMessages,
  titleCase,
  valueToText,
  weatherLabel,
} from "./format";
import type { JsonValue, MessageView, StorySnapshot } from "./types";
import { setInterfaceLocale } from "./i18n";

describe("basic JSON helpers", () => {
  it("normalizes objects, arrays, and printable values", () => {
    expect(asObject({ a: 1 })).toEqual({ a: 1 });
    expect(asObject(["x"])).toEqual({});
    expect(asArray(["x"])).toEqual(["x"]);
    expect(asArray({ a: 1 })).toEqual([]);
    expect(valueToText("")).toBe("-");
    expect(valueToText({ a: 1 })).toBe("A: 1");
    expect(valueToText({ name: "Ledger", detail: "wet ink" })).toBe("Name: Ledger; Detail: wet ink");
    expect(valueToText([{ name: "Moon Key", kind: "quest" }, "rope"])).toBe("Moon Key (Kind: quest), rope");
  });

  it("formats labels and compact text", () => {
    expect(titleCase("known_location-id")).toBe("Known Location Id");
    expect(compactText("  one \n two  three  ", 40)).toBe("one two three");
    expect(compactText("abcdef", 4)).toBe("abc...");
  });
});

describe("clock and stat helpers", () => {
  it("uses canonical world clocks and turn labels", () => {
	const snapshot = snapshotWithWorld({ world_time: { day: 2, minute_of_day: 540, display_text: "Day 2, 09:00" } });
    expect(displayClock(snapshot)).toEqual({ day: 2, time: "Day 2, 09:00", cycle: "Morning" });
	expect(displayClock(null)).toEqual({ day: null, time: "Not tracked", cycle: "Not tracked" });
    expect(messageClock(message({ id: 5, turn: 2 }))).toBe("T2");
    const timestamp = displayTimestamp("2026-07-03 23:44:18.215210603 +0200 CEST m=+25.421624674");
    expect(timestamp).not.toContain("2026-07-03");
    expect(timestamp).not.toContain("CEST");
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
  it("does not present inferred character conditions as canon", () => {
    expect(deriveCondition(null)).toBe("Not tracked");
	expect(deriveCondition(snapshotWithStats({ health: 20 }))).toBe("Not tracked");
	expect(deriveCondition(snapshotWithWorld({ condition: "Wounded" }))).toBe("Wounded");
  });

  it("finds nested strings and weather labels", () => {
    expect(findString({ outer: [{ sky: "Overcast" }] }, ["sky"])).toBe("Overcast");
    expect(findString({ outer: [{ sky: "" }] }, ["sky"])).toBeNull();
	expect(weatherLabel(snapshotWithWorld({ scene_contract: { weather: "Rain" } }))).toBe("Not tracked");
	expect(weatherLabel(snapshotWithWorld({ weather: { tracked: true, label: "Clear" } }))).toBe("Clear");
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

describe("structured text display", () => {
  it("turns raw narrator JSON into player-readable prose", () => {
    const readable = readableStructuredText(
      '{"narrative":"The dock bell rings.","choices":[{"id":1,"text":"Follow the bell"},{"id":2,"text":"Wait"}],"location":"Harbor"}',
    );

    expect(readable).toContain("The dock bell rings.");
    expect(readable).toContain("Location: Harbor");
    expect(readable).toContain("1. Follow the bell");
    expect(readable).not.toContain('{"narrative"');
  });

  it("supports fenced JSON while leaving normal prose unchanged", () => {
    expect(readableStructuredText("```json\n{\"summary\":\"A short note.\"}\n```")).toBe("A short note.");
    expect(readableStructuredText("look around")).toBe("look around");
  });
});

describe("localized presentation fallbacks", () => {
  it("translates fallback and structured field labels without changing canonical values", async () => {
    await setInterfaceLocale("it");
    expect(deriveCondition(null)).toBe("Non monitorato");
    expect(weatherLabel(null)).toBe("Non monitorato");
    expect(entryLabel("plain", 1)).toBe("Elemento 2");
    expect(fieldRows("raw")).toEqual([["Valore", "raw"]]);
    expect(valueToText([{ name: "Chiave", kind: "quest" }])).toBe("Chiave (Categoria: quest)");
    expect(readableStructuredText('{"location":"Porto","mood":"Teso","choices":["Vai"]}')).toContain("Luogo: Porto - Atmosfera: Teso");
    await setInterfaceLocale("en");
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
	branch_id: "branch-main",
	source_commit_id: "commit-main",
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
    character: { id: "character", name: "Hero", fields: { stats: characterStats, condition: overrides.condition } },
    world: {
      id: "world",
      current_location: "Dock",
      current_chapter: 1,
      current_turn: 1,
	  current_location_id: "location-dock",
	  spatial_locations: [],
	  spatial_edges: [],
	  world_time: overrides.world_time ?? {},
	  weather: overrides.weather ?? { tracked: false, label: "Not tracked" },
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
