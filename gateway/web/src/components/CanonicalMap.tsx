import type { JsonValue } from "../types";
import { normalizeKey, readyAssetUrl, type VisualCatalog } from "../visualAssets";

interface MapLocation { id: string; name: string; region_id?: string; description?: string; discovery_state?: string }
interface MapEdge { id: string; from_location_id: string; to_location_id: string; direction?: string; travel_minutes?: number }

export function CanonicalMap({ locationsValue, edgesValue, currentLocationId, visuals }: { locationsValue: JsonValue; edgesValue: JsonValue; currentLocationId: string; visuals?: VisualCatalog }) {
  const locations = mapLocations(locationsValue);
  const locationIDs = new Set(locations.map((location) => location.id));
  const edges = mapEdges(edgesValue).filter((edge) => locationIDs.has(edge.from_location_id) && locationIDs.has(edge.to_location_id));
  if (locations.length === 0) return <p className="empty-copy">No canonical known locations are available for the map.</p>;
  const columns = Math.min(3, Math.max(1, locations.length));
  const rows = Math.ceil(locations.length / columns);
  const width = 660;
  const height = Math.max(170, rows * 125);
  const positions = new Map(locations.map((location, index) => [location.id, { x: 90 + (index % columns) * (480 / Math.max(1, columns - 1)), y: 65 + Math.floor(index / columns) * 120 }]));
  const backgroundUrl = readyAssetUrl(visuals?.mapBackground);
  return (
    <div className={`canonical-map ${backgroundUrl ? "illustrated" : "graph-only"}`}>
      {backgroundUrl && <img className="canonical-map-art" src={backgroundUrl} alt="" aria-hidden="true" />}
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label={`Canonical map with ${locations.length} known locations and ${edges.length} known routes`}>
        {edges.map((edge) => { const from = positions.get(edge.from_location_id)!; const to = positions.get(edge.to_location_id)!; return <g key={edge.id}><line x1={from.x} y1={from.y} x2={to.x} y2={to.y} /><text x={(from.x + to.x) / 2} y={(from.y + to.y) / 2 - 6}>{edge.direction || (edge.travel_minutes ? `${edge.travel_minutes} min` : "route")}</text></g>; })}
        {locations.map((location) => { const point = positions.get(location.id)!; const current = location.id === currentLocationId; const icon = visuals?.mapIcons.get(normalizeKey(location.id)) ?? visuals?.mapIcons.get(normalizeKey(location.name)); const iconUrl = readyAssetUrl(icon); return <g key={location.id} className={`${current ? "current" : ""} ${iconUrl ? "has-icon" : ""}`}><circle cx={point.x} cy={point.y} r={current ? 25 : 20} />{iconUrl && <image href={iconUrl} x={point.x - 18} y={point.y - 18} width="36" height="36" preserveAspectRatio="xMidYMid meet" />}<text className="node-label" x={point.x} y={point.y + 40} textAnchor="middle">{location.name}</text><title>{`${location.name}${location.description ? ` - ${location.description}` : ""}`}</title></g>; })}
      </svg>
      <ul className="sr-only">{locations.map((location) => <li key={location.id}>{location.name}{location.id === currentLocationId ? " (current)" : ""}</li>)}</ul>
    </div>
  );
}

function mapLocations(value: JsonValue): MapLocation[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => item && typeof item === "object" && !Array.isArray(item) && typeof item.id === "string" && typeof item.name === "string" ? [{ id: item.id, name: item.name, region_id: text(item.region_id), description: text(item.description), discovery_state: text(item.discovery_state) }] : []);
}

function mapEdges(value: JsonValue): MapEdge[] {
  if (!Array.isArray(value)) return [];
  return value.flatMap((item) => item && typeof item === "object" && !Array.isArray(item) && typeof item.id === "string" && typeof item.from_location_id === "string" && typeof item.to_location_id === "string" ? [{ id: item.id, from_location_id: item.from_location_id, to_location_id: item.to_location_id, direction: text(item.direction), travel_minutes: typeof item.travel_minutes === "number" ? item.travel_minutes : undefined }] : []);
}

function text(value: unknown): string | undefined { return typeof value === "string" && value.trim() ? value : undefined; }
