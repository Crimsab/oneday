import { ChevronRight, Image as ImageIcon, LocateFixed, Map as MapIcon, Minus, Plus, Route } from "lucide-react";
import { useEffect, useId, useMemo, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import { layoutMapTopology } from "../mapLayout";
import {
  breadcrumbsForScope,
  edgesForScope,
  nodesForScope,
  parseSpatialMap,
  routeFromCurrent,
  type MapScope,
  type SpatialEdge,
  type SpatialMapNode,
} from "../spatialMap";
import type { JsonValue, VisualAsset } from "../types";
import { mapBackgroundForScope, normalizeKey, readyAssetUrl, type VisualCatalog } from "../visualAssets";

interface MapPoint { x: number; y: number }
interface MapView { scale: number; x: number; y: number }

interface CanonicalMapProps {
  regionsValue?: JsonValue;
  locationsValue: JsonValue;
  edgesValue: JsonValue;
  currentLocationId: string;
  visuals?: VisualCatalog;
  expanded?: boolean;
  onOpenVisualAsset?: (assetId: string) => void;
  onTravel?: (locationName: string, route: SpatialEdge | null) => void;
}

const MIN_ZOOM = 0.8;
const MAX_ZOOM = 3;

export function CanonicalMap({
  regionsValue,
  locationsValue,
  edgesValue,
  currentLocationId,
  visuals,
  expanded = false,
  onOpenVisualAsset,
  onTravel,
}: CanonicalMapProps) {
  const clipPathPrefix = `map-node-${useId().replace(/[^a-zA-Z0-9_-]/g, "")}`;
  const model = useMemo(
    () => parseSpatialMap(regionsValue, locationsValue, edgesValue, currentLocationId),
    [currentLocationId, edgesValue, locationsValue, regionsValue],
  );
  const [scope, setScope] = useState<MapScope>(model.defaultScope);
  const nodes = useMemo(() => nodesForScope(model, scope), [model, scope]);
  const edges = useMemo(() => edgesForScope(model, nodes), [model, nodes]);
  const [selectedNodeKey, setSelectedNodeKey] = useState("");
  const [zoomPercent, setZoomPercent] = useState(100);
  const stageRef = useRef<HTMLDivElement | null>(null);
  const viewportRef = useRef<SVGGElement | null>(null);
  const viewRef = useRef<MapView>({ scale: 1, x: 0, y: 0 });
  const dragRef = useRef<{ pointerId: number; x: number; y: number; moved: boolean } | null>(null);

  const width = expanded ? 960 : 660;
  const height = expanded ? 460 : 220;
  const positions = useMemo(
    () => layoutMapTopology(
      nodes.map((node) => ({ id: node.layoutId })),
      edges.map((edge) => ({ from_location_id: edge.from_location_id, to_location_id: edge.to_location_id, direction: edge.direction })),
      width,
      height,
    ),
    [edges, height, nodes, width],
  );
  const breadcrumbs = useMemo(() => breadcrumbsForScope(model, scope), [model, scope]);
  const selectedNode = nodes.find((node) => nodeKey(node) === selectedNodeKey) ?? nodes.find((node) => node.id === currentLocationId) ?? nodes[0] ?? null;
  const selectedIcon = selectedNode?.nodeKind === "location" ? locationIcon(visuals, selectedNode.id, selectedNode.name) : null;
  const selectedIconUrl = readyAssetUrl(selectedIcon);
  const selectedRoute = selectedNode?.canonicalLocationId ? routeFromCurrent(model, currentLocationId, selectedNode.canonicalLocationId) : null;
  const backgroundAsset = mapBackgroundForScope(visuals, scope.kind, scope.id);
  const backgroundUrl = readyAssetUrl(backgroundAsset);

  useEffect(() => {
    setScope(model.defaultScope);
    setSelectedNodeKey("");
  }, [currentLocationId, model.defaultScope.id, model.defaultScope.kind]);

  useEffect(() => {
    if (selectedNodeKey && !nodes.some((node) => nodeKey(node) === selectedNodeKey)) setSelectedNodeKey("");
  }, [nodes, selectedNodeKey]);

  useEffect(() => {
    resetView(viewportRef, viewRef, setZoomPercent);
  }, [expanded, scope.id, scope.kind]);

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
    updateView({ scale, x: anchor.x - (anchor.x - current.x) * ratio, y: anchor.y - (anchor.y - current.y) * ratio });
  };

  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) return;
    const handleWheel = (event: WheelEvent) => {
      event.preventDefault();
      event.stopPropagation();
      const rect = stage.getBoundingClientRect();
      const current = viewRef.current;
      const scale = clamp(current.scale * (event.deltaY < 0 ? 1.14 : 0.88), MIN_ZOOM, MAX_ZOOM);
      const ratio = scale / current.scale;
      const anchor = { x: ((event.clientX - rect.left) / rect.width) * width, y: ((event.clientY - rect.top) / rect.height) * height };
      const next = { scale, x: anchor.x - (anchor.x - current.x) * ratio, y: anchor.y - (anchor.y - current.y) * ratio };
      viewRef.current = next;
      viewportRef.current?.setAttribute("transform", `translate(${next.x} ${next.y}) scale(${next.scale})`);
      setZoomPercent(Math.round(next.scale * 100));
    };
    stage.addEventListener("wheel", handleWheel, { passive: false });
    return () => stage.removeEventListener("wheel", handleWheel);
  }, [height, width]);

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

  const inspectNode = (node: SpatialMapNode) => setSelectedNodeKey(nodeKey(node));
  const openNode = (node: SpatialMapNode) => {
    inspectNode(node);
    if (node.hasChildren && node.childScope) setScope(node.childScope);
  };

  return (
    <div className={`canonical-map ${backgroundUrl ? "illustrated" : "graph-only"} ${expanded ? "expanded" : ""}`}>
      <nav className="canonical-map-breadcrumbs" aria-label="Map hierarchy">
        {breadcrumbs.map((crumb, index) => (
          <span key={`${crumb.kind}:${crumb.id}`}>
            {index > 0 && <ChevronRight size={12} aria-hidden="true" />}
            <button type="button" aria-current={index === breadcrumbs.length - 1 ? "page" : undefined} onClick={() => setScope(crumb)}>{crumb.name}</button>
          </span>
        ))}
      </nav>
      <div ref={stageRef} className="canonical-map-stage">
        {backgroundUrl && <img className="canonical-map-art" src={backgroundUrl} alt="" aria-hidden="true" />}
        <div className="canonical-map-toolbar" aria-label="Map controls">
          <button type="button" onClick={() => zoomAt(1.2)} title="Zoom in" aria-label="Zoom in"><Plus size={15} /></button>
          <span aria-live="polite">{zoomPercent}%</span>
          <button type="button" onClick={() => zoomAt(0.82)} title="Zoom out" aria-label="Zoom out"><Minus size={15} /></button>
          <button type="button" onClick={() => resetView(viewportRef, viewRef, setZoomPercent)} title="Reset map view" aria-label="Reset map view"><LocateFixed size={15} /></button>
          {backgroundAsset && onOpenVisualAsset && (
            <button type="button" onClick={() => onOpenVisualAsset(backgroundAsset.id)} title="Edit this map background" aria-label="Edit this map background"><ImageIcon size={15} /></button>
          )}
        </div>
        {nodes.length === 0 ? (
          <div className="canonical-map-empty"><MapIcon size={22} /><strong>No mapped places inside {scope.name}</strong><span>Discover a child location to expand this map.</span></div>
        ) : (
          <svg
            className="canonical-map-canvas"
            viewBox={`0 0 ${width} ${height}`}
            role="img"
            aria-label={`Interactive ${scope.name} map with ${nodes.length} known places and ${edges.length} known routes`}
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointerMove}
            onPointerUp={handlePointerEnd}
            onPointerCancel={handlePointerEnd}
          >
            <defs>
              {nodes.map((node, index) => {
                const point = positions.get(node.layoutId)!;
                return <clipPath id={`${clipPathPrefix}-clip-${index}`} key={nodeKey(node)}><circle cx={point.x} cy={point.y} r="18" /></clipPath>;
              })}
            </defs>
            <g ref={viewportRef} className="canonical-map-viewport">
              {edges.map((edge) => {
                const from = positions.get(edge.from_location_id)!;
                const to = positions.get(edge.to_location_id)!;
                return <g className="map-route" key={edge.id}><line x1={from.x} y1={from.y} x2={to.x} y2={to.y} /><text x={(from.x + to.x) / 2} y={(from.y + to.y) / 2 - 8}>{routeLabel(edge)}</text></g>;
              })}
              {nodes.map((node, index) => {
                const point = positions.get(node.layoutId)!;
                const current = node.canonicalLocationId === currentLocationId;
                const selected = nodeKey(node) === nodeKey(selectedNode);
                const icon = node.nodeKind === "location" ? locationIcon(visuals, node.id, node.name) : null;
                const iconUrl = readyAssetUrl(icon);
                return (
                  <g
                    key={nodeKey(node)}
                    className={`map-node ${node.nodeKind} ${current ? "current" : ""} ${selected ? "selected" : ""} ${iconUrl ? "has-icon" : ""} ${node.hasChildren ? "has-children" : ""}`}
                    role="button"
                    tabIndex={0}
                    aria-label={`${node.name}, ${node.kind}${current ? ", current location" : ""}${node.hasChildren ? ", contains another map" : ""}`}
                    onPointerDown={(event) => event.stopPropagation()}
                    onClick={() => inspectNode(node)}
                    onDoubleClick={() => openNode(node)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        node.hasChildren ? openNode(node) : inspectNode(node);
                      }
                    }}
                  >
                    <circle cx={point.x} cy={point.y} r={current ? 26 : 22} />
                    {iconUrl && <image href={iconUrl} x={point.x - 18} y={point.y - 18} width="36" height="36" preserveAspectRatio="xMidYMid slice" clipPath={`url(#${clipPathPrefix}-clip-${index})`} />}
                    {!iconUrl && node.nodeKind === "region" && <text className="map-node-glyph" x={point.x} y={point.y + 5} textAnchor="middle">M</text>}
                    {node.hasChildren && <circle className="map-node-child-marker" cx={point.x + 18} cy={point.y - 18} r="5" />}
                    <text className="node-label" x={point.x} y={point.y + 42} textAnchor="middle">{node.name}</text>
                    <title>{`${node.name}, ${node.kind}${node.description ? ` - ${node.description}` : ""}`}</title>
                  </g>
                );
              })}
            </g>
          </svg>
        )}
        <p className="canonical-map-hint">Wheel to zoom, drag to move, double-click an area to enter</p>
      </div>
      {selectedNode && (
        <div className="canonical-map-selection" aria-live="polite">
          <div className={`canonical-map-selection-icon ${selectedIconUrl ? "ready" : "empty"}`}>
            {selectedIconUrl ? <img src={selectedIconUrl} alt="" /> : selectedNode.nodeKind === "region" ? <MapIcon size={18} aria-hidden="true" /> : <LocateFixed size={18} aria-hidden="true" />}
          </div>
          <div className="canonical-map-selection-copy">
            <small>{selectedNode.canonicalLocationId === currentLocationId ? "Current location" : selectedNode.kind}</small>
            <strong>{selectedNode.name}</strong>
            <p>{selectedNode.description || (selectedNode.hasChildren ? "Open this area to inspect its known places." : "No player-known description is available yet.")}</p>
            {selectedRoute && <span className="canonical-map-route-summary"><Route size={12} />{routeLabel(selectedRoute)}</span>}
          </div>
          <div className="canonical-map-selection-actions">
            {selectedNode.hasChildren && selectedNode.childScope && <button type="button" onClick={() => setScope(selectedNode.childScope!)}>Open area</button>}
            {selectedNode.nodeKind === "location" && selectedNode.id !== currentLocationId && onTravel && (
              <button type="button" onClick={() => onTravel(selectedNode.name, selectedRoute)}>{selectedRoute ? "Travel" : "Explore route"}</button>
            )}
            {selectedIcon && onOpenVisualAsset && <button type="button" onClick={() => onOpenVisualAsset(selectedIcon.id)}>Edit image</button>}
          </div>
        </div>
      )}
      <ul className="sr-only">{nodes.map((node) => <li key={nodeKey(node)}>{node.name}{node.canonicalLocationId === currentLocationId ? " (current)" : ""}</li>)}</ul>
    </div>
  );
}

function nodeKey(node: SpatialMapNode | null): string {
  return node ? `${node.nodeKind}:${node.id}` : "";
}

function routeLabel(edge: SpatialEdge): string {
  const parts = [edge.direction, edge.travel_mode, edge.travel_minutes ? `${edge.travel_minutes} min` : ""].filter(Boolean);
  return parts.join(" / ") || "route";
}

function locationIcon(visuals: VisualCatalog | undefined, id: string, name: string): VisualAsset | null {
  return visuals?.mapIcons.get(normalizeKey(id)) ?? visuals?.mapIcons.get(normalizeKey(name)) ?? null;
}

function resetView(viewportRef: { current: SVGGElement | null }, viewRef: { current: MapView }, setZoomPercent: (value: number) => void) {
  viewRef.current = { scale: 1, x: 0, y: 0 };
  viewportRef.current?.setAttribute("transform", "translate(0 0) scale(1)");
  setZoomPercent(100);
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
