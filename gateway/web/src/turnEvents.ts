import type { JsonObject, JsonValue, PlayerAction, StorySnapshot, TurnStreamEvent } from "./types";

export function appendTurnEvent(events: TurnStreamEvent[], next: TurnStreamEvent, limit = 8): TurnStreamEvent[] {
	const key = turnEventKey(next);
	if (events.some((event) => turnEventKey(event) === key)) return events;
	return [...events, next].slice(-limit);
}

export function turnEventDetail(event: TurnStreamEvent): string {
  if (event.status === "submitted") return "Action accepted by the Rust gateway; waiting for OneDay...";
  if (event.status === "completed") return "Turn committed. Syncing the canonical snapshot...";
  if (event.status === "failed") return event.message || "The live engine failed.";
  if (event.status === "snapshot_changed") return event.message || "Canonical state changed from another client.";
  if (event.status === "lagged") return event.message || "Live events were skipped; snapshot sync will recover.";

  switch (event.event_type) {
    case "turn.started":
      return "Engine started resolving the turn.";
    case "narrative.delta":
      return streamingDeltaText(event) || "Narrative is streaming.";
    case "narrative.final":
      return "Narrative generated; applying state updates.";
    case "state.delta":
      return "State delta received.";
    case "choices.updated":
      return "Choices refreshed.";
    case "asset.queued":
      return event.message || "Image generation queued.";
    case "asset.running":
      return event.message || "Image generation started.";
    case "asset.ready":
      return event.message || "Generated image is ready.";
    case "asset.failed":
      return event.message || "Image generation failed.";
    case "asset.cancelled":
      return event.message || "Image generation cancelled.";
    case "turn.committed":
      return "Turn committed to the shared story.";
    default:
      return event.message || event.event_type || "Live engine event received.";
  }
}

export function isVisualAssetTurnEvent(event: TurnStreamEvent): boolean {
  return typeof event.event_type === "string" && event.event_type.startsWith("asset.");
}

export function streamingDeltaText(event: TurnStreamEvent): string {
  if (event.event_type !== "narrative.delta") return "";
  const eventObject = isObject(event.event) ? event.event : {};
  const payload = isObject(eventObject.payload) ? eventObject.payload : {};
  return typeof payload.text === "string" ? payload.text : "";
}

export function parseStorySnapshotEvent(data: string): StorySnapshot | null {
  try {
    const snapshot = JSON.parse(data) as StorySnapshot;
    if (!snapshot || typeof snapshot !== "object") return null;
    return snapshot;
  } catch {
    return null;
  }
}

export function shouldSuppressStreamingDelta(previousText: string | undefined, delta: string): boolean {
  const combined = `${previousText ?? ""}${delta}`.trimStart();
  if (!combined) return false;
  return combined.startsWith("{") || combined.startsWith("[") || combined.startsWith("```json");
}

export function turnEventTitle(event: TurnStreamEvent): string {
  if (event.event_type) return event.event_type;
  return event.status.replace(/_/g, " ");
}

export function turnEventFromContract(
  storyId: string,
  clientTurn: number,
  action: PlayerAction,
  sourceText: string,
  event: JsonValue,
): TurnStreamEvent {
  const eventObject = isObject(event) ? event : {};
  const eventType = typeof eventObject.type === "string" ? eventObject.type : "turn.event";
  return {
    story_id: storyId,
    status: "event",
    client_turn: clientTurn,
    action_kind: action.kind,
    action_text: action.text ?? sourceText,
    event_type: eventType,
    event,
    message: messageForEventType(eventType),
    created_at: new Date().toISOString(),
  };
}

function turnEventKey(event: TurnStreamEvent): string {
  const eventObject = isObject(event.event) ? event.event : {};
  const id = typeof eventObject.id === "string" ? eventObject.id : "";
  if (id) return `id:${id}`;
  return [event.story_id, event.status, event.event_type ?? "", event.client_turn ?? "", event.created_at].join(":");
}

function messageForEventType(eventType: string): string {
  switch (eventType) {
    case "turn.started":
      return "Turn accepted by the live engine.";
    case "narrative.final":
      return "Narrative generated; applying state changes.";
    case "choices.updated":
      return "Choices refreshed from the engine.";
    case "state.delta":
      return "Canonical state changed.";
    case "asset.queued":
      return "Visual asset request queued.";
    case "asset.running":
      return "Visual asset generation started.";
    case "asset.ready":
      return "Visual asset is ready.";
    case "asset.failed":
      return "Visual asset generation failed.";
    case "asset.cancelled":
      return "Visual asset generation cancelled.";
    case "turn.committed":
      return "Turn committed to the shared story.";
    default:
      return `Engine event: ${eventType}.`;
  }
}

function isObject(value: JsonValue | undefined): value is JsonObject {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}
