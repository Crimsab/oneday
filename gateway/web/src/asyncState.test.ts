import { describe, expect, it } from "vitest";
import { isCurrentAsyncSelection } from "./asyncState";

describe("async story selection guard", () => {
  it("accepts only the latest request for the selected story", () => {
    expect(isCurrentAsyncSelection("story-2", "story-2", 4, 4)).toBe(true);
    expect(isCurrentAsyncSelection("story-1", "story-2", 4, 4)).toBe(false);
    expect(isCurrentAsyncSelection("story-2", "story-2", 3, 4)).toBe(false);
  });
});
