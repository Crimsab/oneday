import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
  type WheelEvent as ReactWheelEvent,
} from "react";
import { useTranslation } from "react-i18next";
import {
  Brush,
  CircleOff,
  Eraser,
  Hand,
  Maximize2,
  Redo2,
  RotateCcw,
  Undo2,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import {
  clampBrushSize,
  clampMaskPoint,
  fitMaskView,
  panForAnchoredZoom,
  sourcePointFromClient,
  type MaskPoint,
} from "../maskGeometry";
import {
  applyMaskCommand,
  drawMaskStroke,
  drawMaskStrokeSegment,
  exportCoveragePngBase64,
  maskHasCoverage,
  MaskCommandHistory,
  replayMaskHistory,
  type MaskCommand,
} from "../maskRaster";

type MaskTool = "brush" | "eraser" | "pan";

export interface MaskEditorHandle {
  exportCoveragePngBase64: () => string | null;
  hasCoverage: () => boolean;
}

interface MaskEditorProps {
  sourceUrl: string;
  sourceAlt: string;
  disabled?: boolean;
  onCoverageChange?: (hasCoverage: boolean) => void;
}

const MIN_ZOOM = 0.35;
const MAX_ZOOM = 8;

export const MaskEditor = forwardRef<MaskEditorHandle, MaskEditorProps>(function MaskEditor(
  { sourceUrl, sourceAlt, disabled = false, onCoverageChange },
  forwardedRef,
) {
  const { t } = useTranslation("image_editing");
  const viewportRef = useRef<HTMLDivElement>(null);
  const imageRef = useRef<HTMLImageElement>(null);
  const overlayRef = useRef<HTMLCanvasElement>(null);
  const coverageRef = useRef<HTMLCanvasElement | null>(null);
  const sourceSizeRef = useRef({ width: 1, height: 1 });
  const viewportSizeRef = useRef({ width: 1, height: 1 });
  const viewRef = useRef({ zoom: 1, panX: 0, panY: 0 });
  const historyRef = useRef(new MaskCommandHistory());
  const activeStrokeRef = useRef<Extract<MaskCommand, { kind: "stroke" }> | null>(null);
  const panGestureRef = useRef<{ pointerId: number; x: number; y: number; panX: number; panY: number } | null>(null);
  const spacePressedRef = useRef(false);
  const [tool, setTool] = useState<MaskTool>("brush");
  const toolRef = useRef<MaskTool>(tool);
  const [brushSize, setBrushSize] = useState(64);
  const brushSizeRef = useRef(brushSize);
  const [zoom, setZoom] = useState(1);
  const [ready, setReady] = useState(false);
  const [historyState, setHistoryState] = useState({ canUndo: false, canRedo: false });
  const [status, setStatus] = useState("");

  toolRef.current = tool;
  brushSizeRef.current = brushSize;

  const coverageContext = () => coverageRef.current?.getContext("2d", { willReadFrequently: true }) ?? null;

  const hasCoverage = () => {
    const context = coverageContext();
    return context ? maskHasCoverage(context) : false;
  };

  const publishCoverageState = () => onCoverageChange?.(hasCoverage());

  const redraw = () => {
    const overlay = overlayRef.current;
    const coverage = coverageRef.current;
    const image = imageRef.current;
    if (!overlay || !coverage || !image) return;
    const viewport = viewportSizeRef.current;
    const source = sourceSizeRef.current;
    const view = viewRef.current;
    const transform = fitMaskView(viewport, source, view.zoom, view.panX, view.panY);
    const dpr = Math.max(window.devicePixelRatio || 1, 1);
    const width = Math.max(1, Math.round(viewport.width * dpr));
    const height = Math.max(1, Math.round(viewport.height * dpr));
    if (overlay.width !== width || overlay.height !== height) {
      overlay.width = width;
      overlay.height = height;
    }
    image.style.width = `${source.width}px`;
    image.style.height = `${source.height}px`;
    image.style.transform = `translate3d(${transform.originX}px, ${transform.originY}px, 0) scale(${transform.scale})`;

    const context = overlay.getContext("2d");
    if (!context) return;
    context.setTransform(1, 0, 0, 1, 0, 0);
    context.clearRect(0, 0, width, height);
    context.save();
    context.setTransform(
      transform.scale * dpr,
      0,
      0,
      transform.scale * dpr,
      transform.originX * dpr,
      transform.originY * dpr,
    );
    context.imageSmoothingEnabled = true;
    context.drawImage(coverage, 0, 0);
    context.restore();
    context.globalCompositeOperation = "source-in";
    context.fillStyle = "#ef4458";
    context.fillRect(0, 0, width, height);
    context.globalCompositeOperation = "source-over";
  };

  const resetHistory = () => {
    historyRef.current.reset();
    setHistoryState({ canUndo: false, canRedo: false });
  };

  const replayHistory = () => {
    const context = coverageContext();
    if (!context) return;
    const history = historyRef.current;
    replayMaskHistory(history, context);
    setHistoryState({ canUndo: history.canUndo, canRedo: history.canRedo });
    redraw();
    publishCoverageState();
  };

  const commitCommand = (command: MaskCommand) => {
    const history = historyRef.current;
    const context = coverageContext();
    history.commit(command, context?.getImageData(0, 0, context.canvas.width, context.canvas.height));
    setHistoryState({ canUndo: true, canRedo: false });
    publishCoverageState();
  };

  const setZoomAround = (nextZoom: number, viewportAnchor?: MaskPoint) => {
    const viewport = viewportSizeRef.current;
    const source = sourceSizeRef.current;
    const currentView = viewRef.current;
    const boundedZoom = Math.min(Math.max(nextZoom, MIN_ZOOM), MAX_ZOOM);
    const anchor = viewportAnchor ?? { x: viewport.width / 2, y: viewport.height / 2 };
    const currentTransform = fitMaskView(viewport, source, currentView.zoom, currentView.panX, currentView.panY);
    const sourceAnchor = {
      x: (anchor.x - currentTransform.originX) / currentTransform.scale,
      y: (anchor.y - currentTransform.originY) / currentTransform.scale,
    };
    const nextPan = panForAnchoredZoom(viewport, source, sourceAnchor, anchor, boundedZoom);
    viewRef.current = { zoom: boundedZoom, panX: nextPan.x, panY: nextPan.y };
    setZoom(boundedZoom);
    redraw();
  };

  const fitToView = () => {
    viewRef.current = { zoom: 1, panX: 0, panY: 0 };
    setZoom(1);
    redraw();
    setStatus(t("status.fit"));
  };

  const undo = () => {
    const history = historyRef.current;
    if (!history.undo()) return;
    replayHistory();
    setStatus(t("status.undone"));
  };

  const redo = () => {
    const history = historyRef.current;
    if (!history.redo()) return;
    replayHistory();
    setStatus(t("status.redone"));
  };

  const clear = () => {
    const context = coverageContext();
    if (!context) return;
    context.clearRect(0, 0, context.canvas.width, context.canvas.height);
    commitCommand({ kind: "clear" });
    redraw();
    setStatus(t("status.cleared"));
  };

  const invert = () => {
    const context = coverageContext();
    if (!context) return;
    applyMaskCommand(context, { kind: "invert" });
    commitCommand({ kind: "invert" });
    redraw();
    setStatus(t("status.inverted"));
  };

  useImperativeHandle(forwardedRef, () => ({
    exportCoveragePngBase64: () => coverageRef.current ? exportCoveragePngBase64(coverageRef.current) : null,
    hasCoverage,
  }));

  useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    const resize = () => {
      const rect = viewport.getBoundingClientRect();
      viewportSizeRef.current = { width: Math.max(rect.width, 1), height: Math.max(rect.height, 1) };
      redraw();
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(viewport);
    return () => observer.disconnect();
  }, [ready]);

  const handleImageLoad = () => {
    const image = imageRef.current;
    if (!image) return;
    sourceSizeRef.current = { width: image.naturalWidth, height: image.naturalHeight };
    const coverage = document.createElement("canvas");
    coverage.width = image.naturalWidth;
    coverage.height = image.naturalHeight;
    coverageRef.current = coverage;
    resetHistory();
    viewRef.current = { zoom: 1, panX: 0, panY: 0 };
    setZoom(1);
    setBrushSize(clampBrushSize(Math.round(Math.min(image.naturalWidth, image.naturalHeight) * 0.06), sourceSizeRef.current));
    setReady(true);
    requestAnimationFrame(redraw);
    onCoverageChange?.(false);
  };

  const pointForEvent = (event: ReactPointerEvent<HTMLCanvasElement>) => {
    const overlay = overlayRef.current;
    if (!overlay) return { x: 0, y: 0 };
    const rect = overlay.getBoundingClientRect();
    const transform = fitMaskView(
      viewportSizeRef.current,
      sourceSizeRef.current,
      viewRef.current.zoom,
      viewRef.current.panX,
      viewRef.current.panY,
    );
    return clampMaskPoint(
      { ...sourcePointFromClient(event.clientX, event.clientY, rect, transform), pressure: event.pointerType === "pen" ? event.pressure : 1 },
      sourceSizeRef.current,
    );
  };

  const beginPointer = (event: ReactPointerEvent<HTMLCanvasElement>) => {
    if (disabled || !ready) return;
    const overlay = event.currentTarget;
    overlay.focus();
    overlay.setPointerCapture(event.pointerId);
    const shouldPan = toolRef.current === "pan" || spacePressedRef.current || event.button === 1;
    if (shouldPan) {
      panGestureRef.current = {
        pointerId: event.pointerId,
        x: event.clientX,
        y: event.clientY,
        panX: viewRef.current.panX,
        panY: viewRef.current.panY,
      };
      return;
    }
    if (event.button !== 0) return;
    const command: Extract<MaskCommand, { kind: "stroke" }> = {
      kind: "stroke",
      tool: toolRef.current === "eraser" ? "eraser" : "brush",
      size: brushSizeRef.current,
      points: [pointForEvent(event)],
    };
    activeStrokeRef.current = command;
    const context = coverageContext();
    if (context) drawMaskStroke(context, command);
    redraw();
  };

  const movePointer = (event: ReactPointerEvent<HTMLCanvasElement>) => {
    const pan = panGestureRef.current;
    if (pan?.pointerId === event.pointerId) {
      viewRef.current.panX = pan.panX + event.clientX - pan.x;
      viewRef.current.panY = pan.panY + event.clientY - pan.y;
      redraw();
      return;
    }
    const stroke = activeStrokeRef.current;
    if (!stroke) return;
    const nextPoint = pointForEvent(event);
    const previousPoint = stroke.points.at(-1)!;
    if (Math.hypot(nextPoint.x - previousPoint.x, nextPoint.y - previousPoint.y) < 0.25) return;
    stroke.points.push(nextPoint);
    const context = coverageContext();
    if (context) drawMaskStrokeSegment(context, stroke, previousPoint, nextPoint);
    redraw();
  };

  const endPointer = (event: ReactPointerEvent<HTMLCanvasElement>) => {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    if (panGestureRef.current?.pointerId === event.pointerId) {
      panGestureRef.current = null;
      return;
    }
    const stroke = activeStrokeRef.current;
    if (!stroke) return;
    activeStrokeRef.current = null;
    commitCommand(stroke);
    setStatus(t(stroke.tool === "brush" ? "status.painted" : "status.erased"));
  };

  const handleWheel = (event: ReactWheelEvent<HTMLCanvasElement>) => {
    event.preventDefault();
    const rect = event.currentTarget.getBoundingClientRect();
    const factor = Math.exp(-event.deltaY * 0.0015);
    setZoomAround(viewRef.current.zoom * factor, { x: event.clientX - rect.left, y: event.clientY - rect.top });
  };

  const handleKeyDown = (event: ReactKeyboardEvent<HTMLCanvasElement>) => {
    const key = event.key.toLowerCase();
    if (event.code === "Space") {
      event.preventDefault();
      spacePressedRef.current = true;
      return;
    }
    if ((event.metaKey || event.ctrlKey) && key === "z") {
      event.preventDefault();
      event.shiftKey ? redo() : undo();
      return;
    }
    const actions: Record<string, () => void> = {
      b: () => setTool("brush"),
      e: () => setTool("eraser"),
      h: () => setTool("pan"),
      "[": () => setBrushSize((value) => clampBrushSize(value - 4, sourceSizeRef.current)),
      "]": () => setBrushSize((value) => clampBrushSize(value + 4, sourceSizeRef.current)),
      "+": () => setZoomAround(viewRef.current.zoom * 1.2),
      "=": () => setZoomAround(viewRef.current.zoom * 1.2),
      "-": () => setZoomAround(viewRef.current.zoom / 1.2),
      "0": fitToView,
      c: clear,
    };
    const action = actions[key];
    if (action) {
      event.preventDefault();
      action();
    }
  };

  return (
    <section className="mask-editor" aria-label={t("title")}>
      <div className="mask-editor-toolbar" aria-label={t("tools")}>
        <div className="mask-editor-tool-group" role="group" aria-label={t("paintTools")}>
          <ToolButton active={tool === "brush"} label={t("brush")} shortcut="B" onClick={() => setTool("brush")}><Brush size={16} /></ToolButton>
          <ToolButton active={tool === "eraser"} label={t("eraser")} shortcut="E" onClick={() => setTool("eraser")}><Eraser size={16} /></ToolButton>
          <ToolButton active={tool === "pan"} label={t("pan")} shortcut="H" onClick={() => setTool("pan")}><Hand size={16} /></ToolButton>
        </div>
        <label className="mask-editor-size">
          <span>{t("size")}</span>
          <input
            type="range"
            min="2"
            max={Math.max(16, Math.min(sourceSizeRef.current.width, sourceSizeRef.current.height, 320))}
            value={brushSize}
            disabled={disabled || !ready}
            onChange={(event) => setBrushSize(clampBrushSize(Number(event.target.value), sourceSizeRef.current))}
          />
          <output>{brushSize}px</output>
        </label>
        <div className="mask-editor-tool-group" role="group" aria-label={t("historyTools")}>
          <ToolButton label={t("undoStroke")} shortcut="Ctrl+Z" disabled={!historyState.canUndo || disabled} onClick={undo}><Undo2 size={16} /></ToolButton>
          <ToolButton label={t("redoStroke")} shortcut="Ctrl+Shift+Z" disabled={!historyState.canRedo || disabled} onClick={redo}><Redo2 size={16} /></ToolButton>
          <ToolButton label={t("clear")} shortcut="C" disabled={disabled || !ready} onClick={clear}><CircleOff size={16} /></ToolButton>
          <ToolButton label={t("invert")} disabled={disabled || !ready} onClick={invert}><RotateCcw size={16} /></ToolButton>
        </div>
        <div className="mask-editor-tool-group mask-editor-zoom" role="group" aria-label={t("viewTools")}>
          <ToolButton label={t("zoomOut")} disabled={disabled || zoom <= MIN_ZOOM} onClick={() => setZoomAround(viewRef.current.zoom / 1.2)}><ZoomOut size={16} /></ToolButton>
          <span aria-label={t("zoomValue", { value: Math.round(zoom * 100) })}>{Math.round(zoom * 100)}%</span>
          <ToolButton label={t("zoomIn")} disabled={disabled || zoom >= MAX_ZOOM} onClick={() => setZoomAround(viewRef.current.zoom * 1.2)}><ZoomIn size={16} /></ToolButton>
          <ToolButton label={t("fit")} shortcut="0" disabled={disabled || !ready} onClick={fitToView}><Maximize2 size={16} /></ToolButton>
        </div>
      </div>
      <div ref={viewportRef} className="mask-editor-viewport">
        <img ref={imageRef} src={sourceUrl} alt={sourceAlt} draggable={false} onLoad={handleImageLoad} />
        <canvas
          ref={overlayRef}
          className={`mask-editor-overlay tool-${tool}`}
          tabIndex={disabled ? -1 : 0}
          role="application"
          aria-label={t("canvasLabel")}
          aria-describedby="mask-editor-help"
          onPointerDown={beginPointer}
          onPointerMove={movePointer}
          onPointerUp={endPointer}
          onPointerCancel={endPointer}
          onWheel={handleWheel}
          onKeyDown={handleKeyDown}
          onKeyUp={(event) => { if (event.code === "Space") spacePressedRef.current = false; }}
          onBlur={() => { spacePressedRef.current = false; }}
        />
        {!ready && <div className="mask-editor-loading" aria-live="polite">{t("loading")}</div>}
      </div>
      <p id="mask-editor-help" className="mask-editor-help">{t("help")}</p>
      <p className="sr-only" aria-live="polite">{status}</p>
    </section>
  );
});

function ToolButton({
  active = false,
  label,
  shortcut,
  disabled = false,
  onClick,
  children,
}: {
  active?: boolean;
  label: string;
  shortcut?: string;
  disabled?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  const title = shortcut ? `${label} (${shortcut})` : label;
  return (
    <button
      type="button"
      className={active ? "active" : ""}
      aria-label={label}
      aria-pressed={active || undefined}
      title={title}
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  );
}
