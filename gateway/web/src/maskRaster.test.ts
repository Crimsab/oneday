import { describe, expect, it } from "vitest";
import { coverageRgbaFromAlpha, MaskCommandHistory, type MaskCommand } from "./maskRaster";

const command = (size: number): MaskCommand => ({
  kind: "stroke",
  tool: "brush",
  size,
  points: [{ x: size, y: size }],
});

describe("mask raster contract", () => {
  it("exports coverage as opaque grayscale: 0 preserves and 255 edits", () => {
    const source = new Uint8ClampedArray([
      255, 255, 255, 0,
      255, 255, 255, 127,
      255, 255, 255, 255,
    ]);
    expect([...coverageRgbaFromAlpha(source)]).toEqual([
      0, 0, 0, 255,
      127, 127, 127, 255,
      255, 255, 255, 255,
    ]);
  });

  it("keeps local command undo/redo and drops redo after a new stroke", () => {
    const history = new MaskCommandHistory(2);
    history.commit(command(10));
    history.commit(command(20));
    expect(history.replayPlan().commands.map((item) => item.kind === "stroke" ? item.size : 0)).toEqual([10, 20]);
    expect(history.undo()).toBe(true);
    expect(history.canRedo).toBe(true);
    history.commit(command(30));
    expect(history.canRedo).toBe(false);
    expect(history.replayPlan().commands.map((item) => item.kind === "stroke" ? item.size : 0)).toEqual([10, 30]);
  });

  it("starts replay after the most recent periodic checkpoint", () => {
    const history = new MaskCommandHistory(2);
    const checkpoint = { width: 1, height: 1, data: new Uint8ClampedArray(4), colorSpace: "srgb" } as ImageData;
    history.commit(command(10));
    history.commit(command(20), checkpoint);
    history.commit(command(30));
    expect(history.replayPlan()).toMatchObject({ checkpoint, checkpointIndex: 2 });
    expect(history.replayPlan().commands).toEqual([command(30)]);
  });
});
