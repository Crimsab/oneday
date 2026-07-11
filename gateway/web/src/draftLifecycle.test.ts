import { describe, expect, it } from "vitest";
import { restoreFailedDraft } from "./draftLifecycle";

describe("optimistic composer lifecycle", () => {
  it("restores a failed submission when the cleared composer is untouched", () => {
    expect(restoreFailedDraft("", "Open the sealed gate")).toBe("Open the sealed gate");
  });

  it("preserves replacement text typed while the failed request was pending", () => {
    expect(restoreFailedDraft("Actually, wait", "Open the sealed gate")).toBe("Actually, wait");
  });
});
