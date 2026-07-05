import { describe, expect, it } from "vitest";
import { actionFingerprint, resolvePendingActionIdentity } from "./actionIdentity";
import type { StorySnapshot } from "./types";

describe("action identity", () => {
  it("reuses the same idempotency key while the same action is pending", () => {
    const fingerprint = actionFingerprint("story-1", snapshot(), { kind: "free_text", text: "open door" });
    const first = resolvePendingActionIdentity(null, fingerprint, () => "turn-one");
    const retry = resolvePendingActionIdentity(first, fingerprint, () => "turn-two");

    expect(retry).toBe(first);
    expect(retry.idempotencyKey).toBe("turn-one");
  });

  it("creates a new key when the action or snapshot contract changes", () => {
    const base = snapshot();
    const firstFingerprint = actionFingerprint("story-1", base, { kind: "choice", choice_id: 1, text: "Open" });
    const secondFingerprint = actionFingerprint("story-1", { ...base, version: { ...base.version, revision: 8 } }, {
      kind: "choice",
      choice_id: 1,
      text: "Open",
    });
    const first = resolvePendingActionIdentity(null, firstFingerprint, () => "turn-one");
    const second = resolvePendingActionIdentity(first, secondFingerprint, () => "turn-two");

    expect(second).not.toBe(first);
    expect(second.idempotencyKey).toBe("turn-two");
  });
});

function snapshot(): StorySnapshot {
  return {
    server_time: "2026-01-01T00:00:00Z",
    version: {
      turn: 4,
      revision: 7,
      last_message_id: 10,
      world_updated_at: "2026-01-01T00:00:00Z",
      achievement_count: 0,
      save_count: 0,
    },
    story: {
      id: "story-1",
      name: "Story",
      description: "",
      genre: "",
      tone: "",
      language: "en",
      is_archived: false,
      updated_at: "2026-01-01T00:00:00Z",
    },
    character: { id: "character-1", name: "Hero", fields: {} },
    world: {
      id: "world-1",
      current_location: "",
      current_chapter: 1,
      current_turn: 4,
      known_locations: null,
      global_events: null,
      faction_standings: null,
      story_hooks: null,
      world_reactions: null,
      investigations: null,
      projects: null,
      guidance: null,
      fronts: null,
      timeline: null,
      scene_contract: null,
      updated_at: "2026-01-01T00:00:00Z",
    },
    active_session: {
      id: "session-1",
      story_id: "story-1",
      started_at: "2026-01-01T00:00:00Z",
      summary: "",
    },
    choices: [],
    messages: [],
    panels: { chapters: [], achievements: [], npcs: [], sessions: [], saves: [] },
  };
}
