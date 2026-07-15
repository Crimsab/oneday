import { describe, expect, it } from "vitest";
import { activeStoryCount, railPresentation, toggleDesktopRailMode } from "./railState";

describe("rail state", () => {
  it("keeps the last visible desktop mode when the rail is hidden", () => {
    expect(railPresentation(false, "collapsed")).toBe("hidden");
    expect(railPresentation(true, "collapsed")).toBe("collapsed");
  });

  it("toggles only between persistent visible modes", () => {
    expect(toggleDesktopRailMode("expanded")).toBe("collapsed");
    expect(toggleDesktopRailMode("collapsed")).toBe("expanded");
  });

  it("counts active stories independently from filters", () => {
    expect(activeStoryCount([{ is_archived: false }, { is_archived: true }, { is_archived: false }])).toBe(2);
  });
});
