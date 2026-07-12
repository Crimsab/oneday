import type { JsonValue } from "./types";

export type MapScopeKind = "world" | "region" | "location";

export interface SpatialRegion {
  id: string;
  name: string;
  kind?: string;
  parent_region_id?: string;
}

export interface SpatialLocation {
  id: string;
  name: string;
  kind?: string;
  region_id?: string;
  parent_location_id?: string;
  description?: string;
  discovery_state?: string;
}

export interface SpatialEdge {
  id: string;
  from_location_id: string;
  to_location_id: string;
  direction?: string;
  travel_minutes?: number;
  travel_mode?: string;
  bidirectional?: boolean;
  conditions?: JsonValue;
}

export interface MapScope {
  kind: MapScopeKind;
  id: string;
  name: string;
}

export interface SpatialMapNode {
  id: string;
  layoutId: string;
  nodeKind: "region" | "location";
  name: string;
  kind: string;
  description?: string;
  discoveryState?: string;
  hasChildren: boolean;
  childScope?: MapScope;
  canonicalLocationId?: string;
}

export interface SpatialMapModel {
  regions: SpatialRegion[];
  locations: SpatialLocation[];
  edges: SpatialEdge[];
  defaultScope: MapScope;
}

export function parseSpatialMap(
  regionsValue: JsonValue | undefined,
  locationsValue: JsonValue,
  edgesValue: JsonValue,
  currentLocationId: string,
): SpatialMapModel {
  const regions = parseRegions(regionsValue);
  const locations = parseLocations(locationsValue);
  const locationIds = new Set(locations.map((location) => location.id));
  const edges = parseEdges(edgesValue).filter((edge) => locationIds.has(edge.from_location_id) && locationIds.has(edge.to_location_id));
  return { regions, locations, edges, defaultScope: containingScope(regions, locations, currentLocationId) };
}

export function nodesForScope(model: SpatialMapModel, scope: MapScope): SpatialMapNode[] {
  const { regions, locations } = model;
  let regionNodes: SpatialRegion[] = [];
  let locationNodes: SpatialLocation[] = [];
  if (scope.kind === "world") {
    regionNodes = regions.filter((region) => !region.parent_region_id);
    locationNodes = locations.filter((location) => !location.region_id && !location.parent_location_id);
    if (regionNodes.length === 0 && locationNodes.length === 0) {
      locationNodes = locations.filter((location) => !location.parent_location_id);
    }
  } else if (scope.kind === "region") {
    regionNodes = regions.filter((region) => region.parent_region_id === scope.id);
    locationNodes = locations.filter((location) => location.region_id === scope.id && !location.parent_location_id);
  } else {
    locationNodes = locations.filter((location) => location.parent_location_id === scope.id);
  }

  return [
    ...regionNodes.map((region) => ({
      id: region.id,
      layoutId: `region:${region.id}`,
      nodeKind: "region" as const,
      name: region.name,
      kind: region.kind || "region",
      hasChildren: regions.some((candidate) => candidate.parent_region_id === region.id)
        || locations.some((location) => location.region_id === region.id && !location.parent_location_id),
      childScope: { kind: "region" as const, id: region.id, name: region.name },
    })),
    ...locationNodes.map((location) => {
      const hasChildren = locations.some((candidate) => candidate.parent_location_id === location.id);
      return {
        id: location.id,
        layoutId: location.id,
        nodeKind: "location" as const,
        name: location.name,
        kind: location.kind || "place",
        description: location.description,
        discoveryState: location.discovery_state,
        hasChildren,
        childScope: hasChildren ? { kind: "location" as const, id: location.id, name: location.name } : undefined,
        canonicalLocationId: location.id,
      };
    }),
  ].sort((left, right) => left.name.localeCompare(right.name));
}

export function edgesForScope(model: SpatialMapModel, nodes: SpatialMapNode[]): SpatialEdge[] {
  const visible = new Set(nodes.filter((node) => node.nodeKind === "location").map((node) => node.id));
  return model.edges.filter((edge) => visible.has(edge.from_location_id) && visible.has(edge.to_location_id));
}

export function breadcrumbsForScope(model: SpatialMapModel, scope: MapScope): MapScope[] {
  const crumbs: MapScope[] = [{ kind: "world", id: "root", name: "World" }];
  if (scope.kind === "world") return crumbs;
  if (scope.kind === "region") return [...crumbs, ...regionChain(model.regions, scope.id)];

  const location = model.locations.find((candidate) => candidate.id === scope.id);
  if (!location) return crumbs;
  if (location.region_id) crumbs.push(...regionChain(model.regions, location.region_id));
  const chain: MapScope[] = [];
  let cursor: SpatialLocation | undefined = location;
  const seen = new Set<string>();
  while (cursor && !seen.has(cursor.id)) {
    seen.add(cursor.id);
    chain.unshift({ kind: "location", id: cursor.id, name: cursor.name });
    cursor = model.locations.find((candidate) => candidate.id === cursor?.parent_location_id);
  }
  return [...crumbs, ...chain];
}

export function routeFromCurrent(model: SpatialMapModel, currentLocationId: string, targetLocationId: string): SpatialEdge | null {
  return model.edges.find((edge) => edge.from_location_id === currentLocationId && edge.to_location_id === targetLocationId)
    ?? model.edges.find((edge) => edge.bidirectional && edge.to_location_id === currentLocationId && edge.from_location_id === targetLocationId)
    ?? null;
}

function containingScope(regions: SpatialRegion[], locations: SpatialLocation[], currentLocationId: string): MapScope {
  const current = locations.find((location) => location.id === currentLocationId);
  if (!current) return { kind: "world", id: "root", name: "World" };
  if (current.parent_location_id) {
    const parent = locations.find((location) => location.id === current.parent_location_id);
    if (parent) return { kind: "location", id: parent.id, name: parent.name };
  }
  if (current.region_id) {
    const region = regions.find((candidate) => candidate.id === current.region_id);
    if (region) return { kind: "region", id: region.id, name: region.name };
  }
  return { kind: "world", id: "root", name: "World" };
}

function regionChain(regions: SpatialRegion[], regionId: string): MapScope[] {
  const chain: MapScope[] = [];
  let cursor = regions.find((region) => region.id === regionId);
  const seen = new Set<string>();
  while (cursor && !seen.has(cursor.id)) {
    seen.add(cursor.id);
    chain.unshift({ kind: "region", id: cursor.id, name: cursor.name });
    cursor = regions.find((region) => region.id === cursor?.parent_region_id);
  }
  return chain;
}

function parseRegions(value: JsonValue | undefined): SpatialRegion[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => isObject(item) && string(item.id) && string(item.name) ? [{
    id: string(item.id), name: string(item.name), kind: optionalString(item.kind), parent_region_id: optionalString(item.parent_region_id),
  }] : []);
}

function parseLocations(value: JsonValue): SpatialLocation[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => isObject(item) && string(item.id) && string(item.name) ? [{
    id: string(item.id), name: string(item.name), kind: optionalString(item.kind), region_id: optionalString(item.region_id),
    parent_location_id: optionalString(item.parent_location_id), description: optionalString(item.description), discovery_state: optionalString(item.discovery_state),
  }] : []);
}

function parseEdges(value: JsonValue): SpatialEdge[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => isObject(item) && string(item.id) && string(item.from_location_id) && string(item.to_location_id) ? [{
    id: string(item.id), from_location_id: string(item.from_location_id), to_location_id: string(item.to_location_id),
    direction: optionalString(item.direction), travel_minutes: typeof item.travel_minutes === "number" ? item.travel_minutes : undefined,
    travel_mode: optionalString(item.travel_mode), bidirectional: item.bidirectional === true, conditions: item.conditions as JsonValue,
  }] : []);
}

function isObject(value: JsonValue): value is Record<string, JsonValue> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}

function string(value: JsonValue | undefined): string {
  return typeof value === "string" ? value.trim() : "";
}

function optionalString(value: JsonValue | undefined): string | undefined {
  return string(value) || undefined;
}
