import { describe, expect, it } from "vitest";
import { isServerReady, lifecycleLabel, profileLabel } from "./profile";

describe("desktop profiles", () => {
  it("keeps the first-run choice explicit", () => {
    expect(profileLabel(null)).toBe("Choose a profile");
    expect(profileLabel({ mode: "remote", serverUrl: "https://oneday.example.com/" })).toBe("Remote server");
    expect(profileLabel({ mode: "standalone", profileId: "a".repeat(32) })).toBe("Standalone on this device");
  });

  it("does not expose a server until its lifecycle is ready", () => {
    expect(isServerReady({ state: "starting" })).toBe(false);
    expect(isServerReady({ state: "draining" })).toBe(false);
    expect(isServerReady({ state: "failed", message: "missing sidecar" })).toBe(false);
    expect(isServerReady({ state: "ready", endpoint: "http://127.0.0.1:49152/" })).toBe(true);
    expect(lifecycleLabel({ state: "failed", message: "missing sidecar" })).toBe("Failed");
  });
});
