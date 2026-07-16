import { describe, expect, it } from "vitest";
import { friendlyError } from "./bridge";

describe("friendlyError", () => {
  it("keeps actionable native errors", () => {
    expect(friendlyError(new Error("Server did not answer"))).toBe("Server did not answer");
    expect(friendlyError("Invalid URL")).toBe("Invalid URL");
  });

  it("does not stringify arbitrary objects", () => {
    expect(friendlyError({ secret: "not for UI" })).toBe("The operation could not be completed.");
  });
});
