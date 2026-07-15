import { describe, expect, it } from "vitest";
import {
  clampBrushSize,
  fitMaskView,
  invertCoverage,
  panForAnchoredZoom,
  sourcePointFromClient,
} from "./maskGeometry";

describe("mask editor geometry", () => {
  it("maps viewport pointers back into full-resolution source pixels", () => {
    const transform = fitMaskView({ width: 500, height: 300 }, { width: 1000, height: 1000 }, 2, 25, -10);
    const point = sourcePointFromClient(285, 170, { left: 10, top: 20 }, transform);
    expect(transform).toEqual({ originX: -25, originY: -160, scale: 0.6 });
    expect(point.x).toBeCloseTo(500);
    expect(point.y).toBeCloseTo(516.6667);
  });

  it("keeps the source pixel below the pointer stable while zooming", () => {
    const viewport = { width: 720, height: 480 };
    const source = { width: 1600, height: 900 };
    const anchor = { x: 1120, y: 280 };
    const pointer = { x: 430, y: 170 };
    const pan = panForAnchoredZoom(viewport, source, anchor, pointer, 1.8);
    const transformed = fitMaskView(viewport, source, 1.8, pan.x, pan.y);
    expect(transformed.originX + anchor.x * transformed.scale).toBeCloseTo(pointer.x);
    expect(transformed.originY + anchor.y * transformed.scale).toBeCloseTo(pointer.y);
  });

  it("normalizes coverage inversion and bounds source-pixel brush sizes", () => {
    expect([...invertCoverage(new Uint8ClampedArray([0, 1, 127, 255]))]).toEqual([255, 254, 128, 0]);
    expect(clampBrushSize(0, { width: 32, height: 20 })).toBe(2);
    expect(clampBrushSize(99, { width: 32, height: 20 })).toBe(20);
  });
});
