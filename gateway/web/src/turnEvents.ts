import type { JsonObject, JsonValue, PlayerAction, StorySnapshot, TurnStreamEvent } from "./types";
import i18n from "./i18n";

export function appendTurnEvent(events: TurnStreamEvent[], next: TurnStreamEvent, limit = 8): TurnStreamEvent[] {
	const key = turnEventKey(next);
	if (events.some((event) => turnEventKey(event) === key)) return events;
	return [...events, next].slice(-limit);
}

export function turnEventDetail(event: TurnStreamEvent): string {
  if (event.status === "submitted") return i18n.t("turn_progress:submitted");
  if (event.status === "completed") return i18n.t("turn_progress:completed");
	if (event.status === "failed") return i18n.t("turn_progress:failed");
  if (event.status === "snapshot_changed") return i18n.t("turn_progress:snapshot");
  if (event.status === "lagged") return i18n.t("turn_progress:lagged");

  switch (event.event_type) {
    case "turn.started":
	  return i18n.t("turn_progress:resolving");
    case "narrative.delta":
      return streamingDeltaText(event) || i18n.t("turn_progress:streaming");
    case "narrative.final":
      return i18n.t("turn_progress:final");
    case "challenge.started":
	  return i18n.t("turn_progress:challengeSet");
    case "challenge.resolved":
	  return i18n.t("turn_progress:outcome");
    case "state.delta":
      return i18n.t("turn_progress:stateDelta");
    case "choices.updated":
      return i18n.t("turn_progress:choices");
    case "asset.queued":
      return event.message || i18n.t("turn_progress:imageQueued");
    case "asset.running":
      return event.message || i18n.t("turn_progress:imageStarted");
    case "asset.ready":
      return event.message || i18n.t("turn_progress:imageReady");
    case "asset.failed":
      return event.message || i18n.t("turn_progress:imageFailed");
    case "asset.cancelled":
      return event.message || i18n.t("turn_progress:imageCancelled");
    case "turn.committed":
      return i18n.t("turn_progress:committed");
    default:
	  return i18n.t("turn_progress:updated");
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
	  return i18n.t("turn_progress:accepted");
    case "narrative.final":
      return i18n.t("turn_progress:final");
    case "challenge.started":
	  return i18n.t("turn_progress:challengeSet");
    case "challenge.resolved":
      return i18n.t("turn_progress:outcomeBefore");
    case "choices.updated":
	  return i18n.t("turn_progress:choices");
    case "state.delta":
      return i18n.t("turn_progress:canonicalChanged");
    case "asset.queued":
      return i18n.t("events:asset.queued");
    case "asset.running":
      return i18n.t("events:asset.running");
    case "asset.ready":
      return i18n.t("events:asset.ready");
    case "asset.failed":
      return i18n.t("events:asset.failed");
    case "asset.cancelled":
      return i18n.t("events:asset.cancelled");
    case "turn.committed":
      return i18n.t("turn_progress:committed");
    default:
	  return i18n.t("turn_progress:updated");
  }
}

function isObject(value: JsonValue | undefined): value is JsonObject {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}
