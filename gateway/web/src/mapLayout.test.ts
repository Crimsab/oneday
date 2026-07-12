import { describe, expect, it } from "vitest";
import { layoutMapTopology } from "./mapLayout";

const locations = ["home", "dock", "market", "tower", "island"].map((id) => ({ id }));
const edges = [
  { from_location_id: "home", to_location_id: "dock", direction: "east" },
  { from_location_id: "dock", to_location_id: "market", direction: "north" },
  { from_location_id: "market", to_location_id: "tower" },
];

describe("layoutMapTopology", () => {
  it("is deterministic, bounded, and anchors the current location", () => {
    const first = layoutMapTopology(locations, edges, 960, 460);
    const second = layoutMapTopology(locations, edges, 960, 460);

    expect([...first]).toEqual([...second]);
    expect(first.get("dock")).toEqual({ x: 480, y: 218.5 });
    for (const point of first.values()) {
      expect(point.x).toBeGreaterThanOrEqual(58);
      expect(point.x).toBeLessThanOrEqual(902);
      expect(point.y).toBeGreaterThanOrEqual(38);
      expect(point.y).toBeLessThanOrEqual(408);
    }
  });

  it("honours cardinal route direction and avoids a single-row index grid", () => {
    const points = layoutMapTopology(locations, edges, 960, 460);

    expect(points.get("home")!.x).toBeLessThan(points.get("dock")!.x);
    expect(points.get("market")!.y).toBeLessThan(points.get("dock")!.y);
    expect(new Set([...points.values()].map((point) => Math.round(point.y))).size).toBeGreaterThan(2);
  });

  it("keeps nodes separated in the compact map", () => {
    const points = [...layoutMapTopology(locations, edges, 660, 220).values()];
    const closest = Math.min(...points.flatMap((left, index) => points.slice(index + 1).map((right) => Math.hypot(right.x - left.x, right.y - left.y))));

    expect(closest).toBeGreaterThan(45);
  });
});
