import { Image as ImageIcon, LocateFixed, Minus, Plus } from "lucide-react";
import { useEffect, useId, useMemo, useRef, useState, type PointerEvent as ReactPointerEvent, type WheelEvent as ReactWheelEvent } from "react";
import type { JsonValue, VisualAsset } from "../types";
import { normalizeKey, readyAssetUrl, type VisualCatalog } from "../visualAssets";

interface MapLocation { id: string; name: string; region_id?: string; description?: string; discovery_state?: string }
interface MapEdge { id: string; from_location_id: string; to_location_id: string; direction?: string; travel_minutes?: number }
interface MapPoint { x: number; y: number }
interface MapView { scale: number; x: number; y: number }

interface CanonicalMapProps {
  locationsValue: JsonValue;
  edgesValue: JsonValue;
  currentLocationId: string;
  visuals?: VisualCatalog;
  expanded?: boolean;
  onOpenVisualAsset?: (assetId: string) => void;
}

const MIN_ZOOM = 0.8;
const MAX_ZOOM = 3;

export function CanonicalMap({ locationsValue, edgesValue, currentLocationId, visuals, expanded = false, onOpenVisualAsset }: CanonicalMapProps) {
  const clipPathPrefix = `map-node-${useId().replace(/[^a-zA-Z0-9_-]/g, "")}`;
  const locations = useMemo(() => mapLocations(locationsValue), [locationsValue]);
  const locationIDs = useMemo(() => new Set(locations.map((location) => location.id)), [locations]);
  const edges = useMemo(() => mapEdges(edgesValue).filter((edge) => locationIDs.has(edge.from_location_id) && locationIDs.has(edge.to_location_id)), [edgesValue, locationIDs]);
  const [selectedLocationId, setSelectedLocationId] = useState(currentLocationId || locations[0]?.id || "");
  const [zoomPercent, setZoomPercent] = useState(100);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const viewportRef = useRef<SVGGElement | null>(null);
  const viewRef = useRef<MapView>({ scale: 1, x: 0, y: 0 });
  const dragRef = useRef<{ pointerId: number; x: number; y: number; moved: boolean } | null>(null);

  const columns = Math.min(expanded ? 4 : 3, Math.max(1, locations.length));
  const rows = Math.ceil(locations.length / columns);
  const width = expanded ? 960 : 660;
  const height = Math.max(expanded ? 460 : 220, 120 + rows * (expanded ? 150 : 125));
  const horizontalSpan = width - (expanded ? 220 : 180);
  const positions = useMemo(() => new Map(locations.map((location, index) => [location.id, {
    x: columns === 1 ? width / 2 : (expanded ? 110 : 90) + (index % columns) * (horizontalSpan / (columns - 1)),
    y: (expanded ? 112 : 82) + Math.floor(index / columns) * (expanded ? 150 : 125),
  }])), [columns, expanded, horizontalSpan, locations, width]);
  const backgroundAsset = visuals?.mapBackground ?? null;
  const backgroundUrl = readyAssetUrl(backgroundAsset);
  const selectedLocation = locations.find((location) => location.id === selectedLocationId) ?? locations[0] ?? null;
  const selectedIcon = selectedLocation ? locationIcon(visuals, selectedLocation) : null;
  const selectedIconUrl = readyAssetUrl(selectedIcon);

  useEffect(() => {
    setSelectedLocationId(currentLocationId || locations[0]?.id || "");
  }, [currentLocationId]);

  useEffect(() => {
    if (!locations.some((location) => location.id === selectedLocationId)) {
      setSelectedLocationId(currentLocationId || locations[0]?.id || "");
    }
  }, [currentLocationId, locations, selectedLocationId]);

  useEffect(() => {
    resetView(viewportRef, viewRef, setZoomPercent);
  }, [expanded, locations.length, edges.length]);

  if (locations.length === 0) return <p className="empty-copy">No canonical known locations are available for the map.</p>;

  const updateView = (next: MapView) => {
    const normalized = { ...next, scale: clamp(next.scale, MIN_ZOOM, MAX_ZOOM) };
    viewRef.current = normalized;
    viewportRef.current?.setAttribute("transform", `translate(${normalized.x} ${normalized.y}) scale(${normalized.scale})`);
    setZoomPercent(Math.round(normalized.scale * 100));
  };

  const zoomAt = (factor: number, anchor: MapPoint = { x: width / 2, y: height / 2 }) => {
    const current = viewRef.current;
    const scale = clamp(current.scale * factor, MIN_ZOOM, MAX_ZOOM);
    const ratio = scale / current.scale;
    updateView({
      scale,
      x: anchor.x - (anchor.x - current.x) * ratio,
      y: anchor.y - (anchor.y - current.y) * ratio,
    });
  };

  const handleWheel = (event: ReactWheelEvent<SVGSVGElement>) => {
    event.preventDefault();
    const rect = event.currentTarget.getBoundingClientRect();
    zoomAt(event.deltaY < 0 ? 1.14 : 0.88, {
      x: ((event.clientX - rect.left) / rect.width) * width,
      y: ((event.clientY - rect.top) / rect.height) * height,
    });
  };

  const handlePointerDown = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (event.button !== 0) return;
    dragRef.current = { pointerId: event.pointerId, x: event.clientX, y: event.clientY, moved: false };
    event.currentTarget.setPointerCapture(event.pointerId);
    event.currentTarget.classList.add("dragging");
  };

  const handlePointerMove = (event: ReactPointerEvent<SVGSVGElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    const rect = event.currentTarget.getBoundingClientRect();
    const dx = ((event.clientX - drag.x) / rect.width) * width;
    const dy = ((event.clientY - drag.y) / rect.height) * height;
    if (Math.abs(event.clientX - drag.x) + Math.abs(event.clientY - drag.y) > 2) drag.moved = true;
    drag.x = event.clientX;
    drag.y = event.clientY;
    const current = viewRef.current;
    viewRef.current = { ...current, x: current.x + dx, y: current.y + dy };
    viewportRef.current?.setAttribute("transform", `translate(${viewRef.current.x} ${viewRef.current.y}) scale(${viewRef.current.scale})`);
  };

  const handlePointerEnd = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (dragRef.current?.pointerId !== event.pointerId) return;
    dragRef.current = null;
    event.currentTarget.classList.remove("dragging");
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
  };

  const chooseLocation = (locationId: string) => {
    setSelectedLocationId(locationId);
  };

  return (
    <div className={`canonical-map ${backgroundUrl ? "illustrated" : "graph-only"} ${expanded ? "expanded" : ""}`}>
      <div className="canonical-map-stage">
        {backgroundUrl && <img className="canonical-map-art" src={backgroundUrl} alt="" aria-hidden="true" />}
        <div className="canonical-map-toolbar" aria-label="Map controls">
          <button type="button" onClick={() => zoomAt(1.2)} title="Zoom in" aria-label="Zoom in"><Plus size={15} /></button>
          <span aria-live="polite">{zoomPercent}%</span>
          <button type="button" onClick={() => zoomAt(0.82)} title="Zoom out" aria-label="Zoom out"><Minus size={15} /></button>
          <button type="button" onClick={() => resetView(viewportRef, viewRef, setZoomPercent)} title="Reset map view" aria-label="Reset map view"><LocateFixed size={15} /></button>
          {backgroundAsset && onOpenVisualAsset && (
            <button type="button" onClick={() => onOpenVisualAsset(backgroundAsset.id)} title="Edit map background" aria-label="Edit map background"><ImageIcon size={15} /></button>
          )}
        </div>
        <svg
          ref={svgRef}
          viewBox={`0 0 ${width} ${height}`}
          role="img"
          aria-label={`Interactive canonical map with ${locations.length} known locations and ${edges.length} known routes`}
          onWheel={handleWheel}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerEnd}
          onPointerCancel={handlePointerEnd}
        >
          <defs>
            {locations.map((location, index) => {
              const point = positions.get(location.id)!;
              return <clipPath id={`${clipPathPrefix}-clip-${index}`} key={location.id}><circle cx={point.x} cy={point.y} r="18" /></clipPath>;
            })}
          </defs>
          <g ref={viewportRef} className="canonical-map-viewport">
            {edges.map((edge) => {
              const from = positions.get(edge.from_location_id)!;
              const to = positions.get(edge.to_location_id)!;
              return <g className="map-route" key={edge.id}><line x1={from.x} y1={from.y} x2={to.x} y2={to.y} /><text x={(from.x + to.x) / 2} y={(from.y + to.y) / 2 - 8}>{edge.direction || (edge.travel_minutes ? `${edge.travel_minutes} min` : "route")}</text></g>;
            })}
            {locations.map((location, index) => {
              const point = positions.get(location.id)!;
              const current = location.id === currentLocationId;
              const selected = location.id === selectedLocation?.id;
              const icon = locationIcon(visuals, location);
              const iconUrl = readyAssetUrl(icon);
              return (
                <g
                  key={location.id}
                  className={`map-node ${current ? "current" : ""} ${selected ? "selected" : ""} ${iconUrl ? "has-icon" : ""}`}
                  role="button"
                  tabIndex={0}
                  aria-label={`${location.name}${current ? ", current location" : ""}`}
                  onPointerDown={(event) => event.stopPropagation()}
                  onClick={() => chooseLocation(location.id)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      setSelectedLocationId(location.id);
                    }
                  }}
                >
                  <circle cx={point.x} cy={point.y} r={current ? 26 : 22} />
                  {iconUrl && <image href={iconUrl} x={point.x - 18} y={point.y - 18} width="36" height="36" preserveAspectRatio="xMidYMid slice" clipPath={`url(#${clipPathPrefix}-clip-${index})`} />}
                  <text className="node-label" x={point.x} y={point.y + 42} textAnchor="middle">{location.name}</text>
                  <title>{`${location.name}${location.description ? ` - ${location.description}` : ""}`}</title>
                </g>
              );
            })}
          </g>
        </svg>
        <p className="canonical-map-hint">Mouse wheel: zoom · Drag: move · Click a location: inspect</p>
      </div>
      {selectedLocation && (
        <div className="canonical-map-selection" aria-live="polite">
          <div className={`canonical-map-selection-icon ${selectedIconUrl ? "ready" : "empty"}`}>
            {selectedIconUrl ? <img src={selectedIconUrl} alt="" /> : <LocateFixed size={18} aria-hidden="true" />}
          </div>
          <div>
            <small>{selectedLocation.id === currentLocationId ? "Current location" : selectedLocation.discovery_state || "Known location"}</small>
            <strong>{selectedLocation.name}</strong>
            <p>{selectedLocation.description || "No player-known description is available yet."}</p>
          </div>
          {selectedIcon && onOpenVisualAsset && <button type="button" onClick={() => onOpenVisualAsset(selectedIcon.id)}>Edit image</button>}
        </div>
      )}
      <ul className="sr-only">{locations.map((location) => <li key={location.id}>{location.name}{location.id === currentLocationId ? " (current)" : ""}</li>)}</ul>
    </div>
  );
}

function locationIcon(visuals: VisualCatalog | undefined, location: MapLocation): VisualAsset | null {
  return visuals?.mapIcons.get(normalizeKey(location.id)) ?? visuals?.mapIcons.get(normalizeKey(location.name)) ?? null;
}

function resetView(viewportRef: { current: SVGGElement | null }, viewRef: { current: MapView }, setZoomPercent: (value: number) => void) {
  viewRef.current = { scale: 1, x: 0, y: 0 };
  viewportRef.current?.setAttribute("transform", "translate(0 0) scale(1)");
  setZoomPercent(100);
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
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
