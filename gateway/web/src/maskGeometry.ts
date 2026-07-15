export interface MaskPoint {
  x: number;
  y: number;
  pressure?: number;
}

export interface MaskViewTransform {
  originX: number;
  originY: number;
  scale: number;
}

export interface MaskViewport {
  width: number;
  height: number;
}

export function fitMaskView(
  viewport: MaskViewport,
  source: MaskViewport,
  zoom = 1,
  panX = 0,
  panY = 0,
): MaskViewTransform {
  const fitScale = Math.min(
    viewport.width / Math.max(source.width, 1),
    viewport.height / Math.max(source.height, 1),
  );
  const scale = Math.max(fitScale * zoom, Number.EPSILON);
  return {
    originX: (viewport.width - source.width * scale) / 2 + panX,
    originY: (viewport.height - source.height * scale) / 2 + panY,
    scale,
  };
}

export function sourcePointFromClient(
  clientX: number,
  clientY: number,
  rect: Pick<DOMRect, "left" | "top">,
  transform: MaskViewTransform,
): MaskPoint {
  return {
    x: (clientX - rect.left - transform.originX) / transform.scale,
    y: (clientY - rect.top - transform.originY) / transform.scale,
  };
}

export function clampMaskPoint(point: MaskPoint, source: MaskViewport): MaskPoint {
  return {
    ...point,
    x: Math.min(Math.max(point.x, 0), source.width),
    y: Math.min(Math.max(point.y, 0), source.height),
  };
}

export function panForAnchoredZoom(
  viewport: MaskViewport,
  source: MaskViewport,
  sourceAnchor: MaskPoint,
  viewportAnchor: MaskPoint,
  zoom: number,
): { x: number; y: number } {
  const centered = fitMaskView(viewport, source, zoom);
  return {
    x: viewportAnchor.x - centered.originX - sourceAnchor.x * centered.scale,
    y: viewportAnchor.y - centered.originY - sourceAnchor.y * centered.scale,
  };
}

export function clampBrushSize(value: number, source: MaskViewport): number {
  return Math.min(Math.max(Math.round(value), 2), Math.max(2, Math.min(source.width, source.height)));
}

export function invertCoverage(values: Uint8ClampedArray): Uint8ClampedArray {
  const inverted = new Uint8ClampedArray(values.length);
  for (let index = 0; index < values.length; index += 1) {
    inverted[index] = 255 - values[index];
  }
  return inverted;
}
