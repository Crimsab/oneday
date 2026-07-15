import type { TFunction } from "i18next";
import type { TurnStreamEvent, VisualAsset } from "./types";

export function turnEventMessage(event: TurnStreamEvent, t: TFunction): string {
  if (!event.message_key) return event.message;
  const key = event.message_key.startsWith("turn.event.") ? `events:${event.message_key.slice("turn.event.".length)}` : `server:${event.message_key}`;
  return t(key, { ...(event.message_args ?? {}), defaultValue: event.message || t("common:genericError") });
}

export function visualGateReason(asset: VisualAsset, t: TFunction): string {
  if (!asset.gate_reason_code) return asset.gate_reason;
  return t(`server:${asset.gate_reason_code}`, { defaultValue: asset.gate_reason });
}
