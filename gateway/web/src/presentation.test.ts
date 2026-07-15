import { afterEach, describe, expect, it } from "vitest";
import i18n, { setInterfaceLocale } from "./i18n";
import { turnEventMessage } from "./presentation";
import type { TurnStreamEvent } from "./types";

const emitted = ["turn.started", "narrative.delta", "narrative.final", "choices.updated", "state.delta", "roll.resolved", "challenge.started", "challenge.resolved", "combat.started", "combat.updated", "social.started", "social.updated", "asset.queued", "asset.running", "asset.ready", "asset.failed", "asset.cancelled", "turn.committed", "error"];

describe("semantic gateway presentation", () => {
  afterEach(async () => { await setInterfaceLocale("en"); });
  it("translates every emitted turn event key and preserves legacy fallback", async () => {
    await setInterfaceLocale("it");
    for (const type of emitted) {
      const event = { message_key: `turn.event.${type}`, message_args: {}, message: `legacy ${type}` } as TurnStreamEvent;
      expect(turnEventMessage(event, i18n.t)).not.toBe(event.message);
      expect(turnEventMessage(event, i18n.t)).not.toBe("Unavailable");
    }
    expect(turnEventMessage({ message: "Legacy message" } as TurnStreamEvent, i18n.t)).toBe("Legacy message");
  });
});
