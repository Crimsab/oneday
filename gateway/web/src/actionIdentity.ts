import type { PlayerAction, StorySnapshot } from "./types";

export interface PendingActionIdentity {
  fingerprint: string;
  idempotencyKey: string;
}

export function actionFingerprint(storyId: string, snapshot: StorySnapshot, action: PlayerAction): string {
  return [
    storyId,
    snapshot.active_session.id,
    snapshot.world.current_turn,
    snapshot.version.revision,
    action.kind,
    action.choice_id ?? "",
    action.text?.trim() ?? "",
  ].join("\u001f");
}

export function resolvePendingActionIdentity(
  pending: PendingActionIdentity | null,
  fingerprint: string,
  createKey: () => string,
): PendingActionIdentity {
  if (pending?.fingerprint === fingerprint) return pending;
  return {
    fingerprint,
    idempotencyKey: createKey(),
  };
}
