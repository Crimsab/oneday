import { describe, expect, it } from "vitest";
import {
  appendTurnEvent,
  parseStorySnapshotEvent,
  shouldSuppressStreamingDelta,
  streamingDeltaText,
  turnEventDetail,
  turnEventFromContract,
  turnEventTitle,
} from "./turnEvents";
import type { TurnStreamEvent } from "./types";

describe("turn event helpers", () => {
  it("deduplicates SSE and POST events by canonical event id", () => {
    const first = event({ created_at: "2026-01-01T00:00:00Z" });
    const duplicate = event({ created_at: "2026-01-01T00:00:02Z", message: "Updated label" });

    expect(appendTurnEvent(appendTurnEvent([], first), duplicate)).toEqual([first]);
  });

  it("keeps the newest bounded live events", () => {
    const events = Array.from({ length: 4 }, (_, index) =>
      event({ event: { id: `evt-${index}`, type: "turn.started" }, created_at: `2026-01-01T00:00:0${index}Z` }),
    ).reduce((items, next) => appendTurnEvent(items, next, 3), [] as TurnStreamEvent[]);

    expect(events.map((item) => item.event && typeof item.event === "object" && !Array.isArray(item.event) ? item.event.id : "")).toEqual([
      "evt-1",
      "evt-2",
      "evt-3",
    ]);
  });

  it("describes lifecycle states and contract events for the transcript", () => {
    expect(turnEventDetail(event({ status: "submitted", event_type: null }))).toContain("Rust gateway");
    expect(turnEventDetail(event({ event_type: "narrative.final" }))).toContain("Narrative generated");
    expect(turnEventTitle(event({ event_type: "turn.committed" }))).toBe("turn.committed");
  });

  it("extracts narrative delta text for provisional streaming display", () => {
    const delta = event({
      event_type: "narrative.delta",
      event: { id: "turn-1:live:2", type: "narrative.delta", payload: { text: "La porta si apre." } },
    });

    expect(streamingDeltaText(delta)).toBe("La porta si apre.");
    expect(turnEventDetail(delta)).toBe("La porta si apre.");
  });

  it("suppresses structured JSON deltas from provisional prose display", () => {
    expect(shouldSuppressStreamingDelta(undefined, '{"narrative":"La porta')).toBe(true);
    expect(shouldSuppressStreamingDelta("   ", "[{\"text\":\"raw\"}]")).toBe(true);
    expect(shouldSuppressStreamingDelta(undefined, "La porta si apre.")).toBe(false);
  });

  it("parses snapshot SSE payloads without throwing on malformed data", () => {
    expect(parseStorySnapshotEvent("{bad json")).toBeNull();
    expect(parseStorySnapshotEvent("null")).toBeNull();
    expect(parseStorySnapshotEvent(JSON.stringify({ story: { id: "story-1" }, version: { revision: 2 } }))).toMatchObject({
      story: { id: "story-1" },
      version: { revision: 2 },
    });
  });

  it("wraps contract events returned by the action response", () => {
    const wrapped = turnEventFromContract("story-1", 7, { kind: "choice", choice_id: 2, text: "Open" }, "Open", {
      id: "evt-1",
      type: "choices.updated",
    });

    expect(wrapped).toMatchObject({
      story_id: "story-1",
      status: "event",
      client_turn: 7,
      action_kind: "choice",
      event_type: "choices.updated",
    });
  });
});

function event(overrides: Partial<TurnStreamEvent> = {}): TurnStreamEvent {
  return {
    story_id: "story-1",
    status: "event",
    client_turn: 7,
    action_kind: "choice",
    action_text: "Open",
    event_type: "turn.started",
    event: { id: "evt-1", type: "turn.started" },
    message: "Turn accepted.",
    created_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}
