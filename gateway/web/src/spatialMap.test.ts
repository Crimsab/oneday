import { describe, expect, it } from "vitest";
import { breadcrumbsForScope, edgesForScope, nodesForScope, parseSpatialMap, routeFromCurrent } from "./spatialMap";

const regions = [
  { id: "vharrow", name: "Vharrow", kind: "macroregion", parent_region_id: "" },
  { id: "port", name: "Port District", kind: "district", parent_region_id: "vharrow" },
];
const locations = [
  { id: "dock", name: "Dock 7", kind: "site", region_id: "port", parent_location_id: "" },
  { id: "pump", name: "Pump House", kind: "landmark", region_id: "port", parent_location_id: "" },
  { id: "lane", name: "Access Lane", kind: "subzone", region_id: "port", parent_location_id: "dock" },
];
const edges = [
  { id: "dock-pump", from_location_id: "dock", to_location_id: "pump", travel_minutes: 8, travel_mode: "walk", bidirectional: true },
];

describe("spatial map scopes", () => {
  it("opens the containing scope and exposes hierarchy breadcrumbs", () => {
    const model = parseSpatialMap(regions, locations, edges, "lane");
    expect(model.defaultScope).toEqual({ kind: "location", id: "dock", name: "Dock 7" });
    expect(breadcrumbsForScope(model, model.defaultScope).map((crumb) => crumb.name)).toEqual([
      "World", "Vharrow", "Port District", "Dock 7",
    ]);
    expect(nodesForScope(model, model.defaultScope).map((node) => node.name)).toEqual(["Access Lane"]);
  });

  it("shows only direct children and routes belonging to the current scope", () => {
    const model = parseSpatialMap(regions, locations, edges, "dock");
    const portScope = { kind: "region" as const, id: "port", name: "Port District" };
    const nodes = nodesForScope(model, portScope);
    expect(nodes.map((node) => node.name)).toEqual(["Dock 7", "Pump House"]);
    expect(edgesForScope(model, nodes)).toHaveLength(1);
    expect(routeFromCurrent(model, "pump", "dock")?.id).toBe("dock-pump");
  });
});
