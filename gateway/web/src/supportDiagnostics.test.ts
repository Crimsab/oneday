import { beforeEach, describe, expect, it } from "vitest";
import { defaultPreferences } from "./preferences";
import {
  buildSupportBundle,
  clearSupportEvents,
  getSupportEvents,
  recordApiSupportEvent,
  recordSupportEvent,
  redactSupportText,
} from "./supportDiagnostics";
import type { StorySnapshot } from "./types";

describe("support diagnostics", () => {
  beforeEach(() => clearSupportEvents());

  it("redacts common credentials and bounds captured text", () => {
    const secret = "super-secret-value";
    const redacted = redactSupportText(`Authorization: Bearer ${secret} token=${secret} password=${secret} https://example.test/?access_token=${secret} ${"x".repeat(3_000)}`);
    expect(redacted).not.toContain(secret);
    expect(redacted).toContain("[REDACTED]");
    expect(redacted.length).toBeLessThanOrEqual(2_000);
  });

  it("sanitizes story identifiers and query strings in API events", () => {
    recordApiSupportEvent("get", "/api/stories/private-story-id/snapshot?token=secret", 500, 12.7, "failed");
    const event = getSupportEvents().at(-1);
    expect(event?.message).toBe("GET /api/stories/:story/snapshot -> 500");
    expect(event?.detail).toBe("13 ms; failed");
  });

  it("includes technical story context without copying narrative text", () => {
    recordSupportEvent("error", "application", "Example failure");
    const snapshot = {
      world: { current_turn: 9, secret_story_text: "The hidden ending" },
      version: { revision: 4 },
      messages: [{ branch_id: "branch-main", content: "Private story text" }],
    } as unknown as StorySnapshot;
    const bundle = buildSupportBundle({ preferences: defaultPreferences, snapshot, modelSettings: null });
    const serialized = JSON.stringify(bundle);

    expect(bundle.story_context).toMatchObject({ selected: true, turn: 9, revision: 4, branch_id: "branch-main", message_count: 1 });
    expect(serialized).not.toContain("The hidden ending");
    expect(serialized).not.toContain("Private story text");
    expect(serialized).toContain("Example failure");
  });
});
