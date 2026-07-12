export interface MapLayoutLocation {
  id: string;
}

export interface MapLayoutEdge {
  from_location_id: string;
  to_location_id: string;
  direction?: string;
}

export interface MapLayoutPoint {
  x: number;
  y: number;
}

interface MutablePoint extends MapLayoutPoint {
  vx: number;
  vy: number;
}

const TAU = Math.PI * 2;

/**
 * Deterministic, topology-aware graph layout for the known-location map.
 *
 * Nodes start from stable ID-derived positions, then a bounded force pass keeps
 * connected locations near each other, separates labels, honours cardinal
 * route directions, and anchors the current location in the visual centre.
 */
export function layoutMapTopology(
  locations: MapLayoutLocation[],
  edges: MapLayoutEdge[],
  width: number,
  height: number,
): Map<string, MapLayoutPoint> {
  if (locations.length === 0) return new Map();

  const ids = new Set(locations.map((location) => location.id));
  const validEdges = edges.filter(
    (edge) => ids.has(edge.from_location_id) && ids.has(edge.to_location_id) && edge.from_location_id !== edge.to_location_id,
  );
  const degrees = new Map(locations.map((location) => [location.id, 0]));
  for (const edge of validEdges) {
    degrees.set(edge.from_location_id, (degrees.get(edge.from_location_id) ?? 0) + 1);
    degrees.set(edge.to_location_id, (degrees.get(edge.to_location_id) ?? 0) + 1);
  }
  // A stable graph hub preserves the player's spatial memory when the current
  // location changes. Input order breaks degree ties so newly discovered nodes
  // do not reshuffle an established map.
  const anchorId = locations.reduce((best, location) =>
    (degrees.get(location.id) ?? 0) > (degrees.get(best.id) ?? 0) ? location : best,
  ).id;
  const centre = { x: width / 2, y: height / 2 - Math.min(12, height * 0.025) };
  const marginX = clamp(width * 0.105, 58, 112);
  const marginTop = clamp(height * 0.13, 38, 72);
  const marginBottom = clamp(height * 0.2, 52, 96);
  const spanX = Math.max(1, width - marginX * 2);
  const spanY = Math.max(1, height - marginTop - marginBottom);
  const countScale = Math.sqrt(locations.length);
  const separationX = clamp(spanX / (countScale + 1.05), 105, 210);
  const separationY = clamp(spanY / (countScale + 0.45), 70, 126);
  const springLength = Math.min(190, Math.max(82, Math.hypot(separationX, separationY) * 0.76));

  const points = new Map<string, MutablePoint>();
  for (const location of locations) {
    if (location.id === anchorId) {
      points.set(location.id, { ...centre, vx: 0, vy: 0 });
      continue;
    }
    const angle = stableUnit(`${location.id}:angle`) * TAU;
    const radius = 0.28 + stableUnit(`${location.id}:radius`) * 0.57;
    points.set(location.id, {
      x: centre.x + Math.cos(angle) * spanX * 0.5 * radius,
      y: centre.y + Math.sin(angle) * spanY * 0.5 * radius,
      vx: 0,
      vy: 0,
    });
  }

  for (let iteration = 0; iteration < 180; iteration += 1) {
    const cooling = 0.92 - iteration * 0.0024;
    const displacement = new Map(locations.map((location) => [location.id, { x: 0, y: 0 }]));

    for (let leftIndex = 0; leftIndex < locations.length; leftIndex += 1) {
      for (let rightIndex = leftIndex + 1; rightIndex < locations.length; rightIndex += 1) {
        const left = points.get(locations[leftIndex].id)!;
        const right = points.get(locations[rightIndex].id)!;
        let dx = right.x - left.x;
        let dy = right.y - left.y;
        if (Math.abs(dx) + Math.abs(dy) < 0.001) {
          const angle = stableUnit(`${locations[leftIndex].id}:${locations[rightIndex].id}`) * TAU;
          dx = Math.cos(angle);
          dy = Math.sin(angle);
        }
        const normalizedDistance = Math.hypot(dx / separationX, dy / separationY);
        if (normalizedDistance >= 1.15) continue;
        const force = (1.15 - normalizedDistance) * 0.72;
        const distance = Math.max(1, Math.hypot(dx, dy));
        const fx = (dx / distance) * force * separationX;
        const fy = (dy / distance) * force * separationY;
        displacement.get(locations[leftIndex].id)!.x -= fx;
        displacement.get(locations[leftIndex].id)!.y -= fy;
        displacement.get(locations[rightIndex].id)!.x += fx;
        displacement.get(locations[rightIndex].id)!.y += fy;
      }
    }

    for (const edge of validEdges) {
      const from = points.get(edge.from_location_id)!;
      const to = points.get(edge.to_location_id)!;
      const direction = directionVector(edge.direction);
      if (direction) {
        const desiredX = direction.x * springLength;
        const desiredY = direction.y * springLength;
        const fx = (to.x - from.x - desiredX) * 0.075;
        const fy = (to.y - from.y - desiredY) * 0.075;
        displacement.get(edge.from_location_id)!.x += fx;
        displacement.get(edge.from_location_id)!.y += fy;
        displacement.get(edge.to_location_id)!.x -= fx;
        displacement.get(edge.to_location_id)!.y -= fy;
      } else {
        const dx = to.x - from.x;
        const dy = to.y - from.y;
        const distance = Math.max(1, Math.hypot(dx, dy));
        const force = (distance - springLength) * 0.052;
        const fx = (dx / distance) * force;
        const fy = (dy / distance) * force;
        displacement.get(edge.from_location_id)!.x += fx;
        displacement.get(edge.from_location_id)!.y += fy;
        displacement.get(edge.to_location_id)!.x -= fx;
        displacement.get(edge.to_location_id)!.y -= fy;
      }
    }

    for (const location of locations) {
      const point = points.get(location.id)!;
      if (location.id === anchorId) {
        point.x = centre.x;
        point.y = centre.y;
        point.vx = 0;
        point.vy = 0;
        continue;
      }
      const delta = displacement.get(location.id)!;
      delta.x += (centre.x - point.x) * 0.004;
      delta.y += (centre.y - point.y) * 0.004;
      point.vx = (point.vx + delta.x * 0.085) * cooling;
      point.vy = (point.vy + delta.y * 0.085) * cooling;
      point.x = clamp(point.x + point.vx, marginX, width - marginX);
      point.y = clamp(point.y + point.vy, marginTop, height - marginBottom);
    }
  }

  return new Map(locations.map((location) => {
    const point = points.get(location.id)!;
    return [location.id, { x: round(point.x), y: round(point.y) }];
  }));
}

function directionVector(value?: string): { x: number; y: number } | null {
  if (!value) return null;
  const direction = value.toLowerCase().replaceAll(/[^a-z]/g, "");
  let x = 0;
  let y = 0;
  if (direction.includes("east") || direction.includes("right")) x += 1;
  if (direction.includes("west") || direction.includes("left")) x -= 1;
  if (direction.includes("north") || direction === "up" || direction.includes("upward")) y -= 1;
  if (direction.includes("south") || direction === "down" || direction.includes("downward")) y += 1;
  if (x === 0 && y === 0) return null;
  const length = Math.hypot(x, y);
  return { x: x / length, y: y / length };
}

function stableUnit(value: string): number {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0) / 4294967295;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

function round(value: number): number {
  return Math.round(value * 100) / 100;
}
