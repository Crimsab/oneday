import { forwardRef, useEffect, useImperativeHandle, useRef, useState, type PointerEvent } from "react";
import { Brush, MapPin, RotateCcw, Undo2 } from "lucide-react";
import { useTranslation } from "react-i18next";

type Point = { x: number; y: number };
type AnnotationCommand =
  | { kind: "stroke"; points: Point[] }
  | { kind: "note"; point: Point; text: string; number: number };

export interface AnnotationEditorHandle {
  exportAnnotatedPngBase64: () => string | null;
  promptSummary: () => string;
}

interface AnnotationEditorProps {
  sourceUrl: string;
  sourceAlt: string;
  disabled?: boolean;
  onAnnotationChange?: (hasAnnotations: boolean) => void;
}

export const AnnotationEditor = forwardRef<AnnotationEditorHandle, AnnotationEditorProps>(function AnnotationEditor(
  { sourceUrl, sourceAlt, disabled = false, onAnnotationChange },
  forwardedRef,
) {
  const { t } = useTranslation("image_editing");
  const imageRef = useRef<HTMLImageElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const commandsRef = useRef<AnnotationCommand[]>([]);
  const activeStrokeRef = useRef<Extract<AnnotationCommand, { kind: "stroke" }> | null>(null);
  const [tool, setTool] = useState<"draw" | "note">("draw");
  const [noteText, setNoteText] = useState("");
  const [commandCount, setCommandCount] = useState(0);
  const [aspectRatio, setAspectRatio] = useState("16 / 9");

  const resize = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const dpr = Math.max(window.devicePixelRatio || 1, 1);
    canvas.width = Math.max(1, Math.round(rect.width * dpr));
    canvas.height = Math.max(1, Math.round(rect.height * dpr));
    drawOverlay();
  };

  const drawCommands = (context: CanvasRenderingContext2D, width: number, height: number) => {
    context.lineCap = "round";
    context.lineJoin = "round";
    for (const command of commandsRef.current) {
      if (command.kind === "stroke") {
        if (command.points.length < 2) continue;
        context.beginPath();
        context.strokeStyle = "#ff4d4f";
        context.lineWidth = Math.max(3, width * 0.006);
        context.moveTo(command.points[0].x * width, command.points[0].y * height);
        for (const point of command.points.slice(1)) context.lineTo(point.x * width, point.y * height);
        context.stroke();
        continue;
      }
      const x = command.point.x * width;
      const y = command.point.y * height;
      const fontSize = Math.max(14, Math.round(width * 0.022));
      context.font = `600 ${fontSize}px IBM Plex Sans, sans-serif`;
      const label = `${command.number}. ${command.text}`;
      const boxWidth = Math.min(width * 0.58, context.measureText(label).width + fontSize * 1.6);
      const boxHeight = fontSize * 1.8;
      const boxX = Math.min(width - boxWidth - 4, Math.max(4, x + fontSize * 0.8));
      const boxY = Math.min(height - boxHeight - 4, Math.max(4, y - boxHeight / 2));
      context.fillStyle = "rgba(8, 12, 15, .88)";
      context.fillRect(boxX, boxY, boxWidth, boxHeight);
      context.strokeStyle = "#ff4d4f";
      context.lineWidth = Math.max(2, width * 0.002);
      context.strokeRect(boxX, boxY, boxWidth, boxHeight);
      context.fillStyle = "#ffffff";
      context.textBaseline = "middle";
      context.fillText(label, boxX + fontSize * 0.6, boxY + boxHeight / 2, boxWidth - fontSize);
      context.beginPath();
      context.fillStyle = "#ff4d4f";
      context.arc(x, y, Math.max(6, fontSize * 0.38), 0, Math.PI * 2);
      context.fill();
    }
  };

  const drawOverlay = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const context = canvas.getContext("2d");
    if (!context) return;
    context.clearRect(0, 0, canvas.width, canvas.height);
    drawCommands(context, canvas.width, canvas.height);
  };

  const publish = () => {
    const count = commandsRef.current.length;
    setCommandCount(count);
    onAnnotationChange?.(count > 0);
    drawOverlay();
  };

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const observer = new ResizeObserver(resize);
    observer.observe(canvas);
    resize();
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    commandsRef.current = [];
    setCommandCount(0);
    setAspectRatio("16 / 9");
    onAnnotationChange?.(false);
    requestAnimationFrame(resize);
  }, [sourceUrl]);

  useImperativeHandle(forwardedRef, () => ({
    exportAnnotatedPngBase64: () => {
      const image = imageRef.current;
      if (!image?.naturalWidth || commandsRef.current.length === 0) return null;
      const output = document.createElement("canvas");
      output.width = image.naturalWidth;
      output.height = image.naturalHeight;
      const context = output.getContext("2d");
      if (!context) return null;
      context.drawImage(image, 0, 0, output.width, output.height);
      drawCommands(context, output.width, output.height);
      return output.toDataURL("image/png").split(",", 2)[1] ?? null;
    },
    promptSummary: () => {
      const notes = commandsRef.current.filter((command): command is Extract<AnnotationCommand, { kind: "note" }> => command.kind === "note");
      const lines = notes.map((note) => `${note.number}. ${note.text}`);
      const hasDrawing = commandsRef.current.some((command) => command.kind === "stroke");
      if (hasDrawing) lines.unshift(t("annotations.drawnInstruction"));
      return lines.join("\n");
    },
  }));

  const point = (event: PointerEvent<HTMLCanvasElement>): Point => {
    const rect = event.currentTarget.getBoundingClientRect();
    return {
      x: Math.min(1, Math.max(0, (event.clientX - rect.left) / rect.width)),
      y: Math.min(1, Math.max(0, (event.clientY - rect.top) / rect.height)),
    };
  };
  const down = (event: PointerEvent<HTMLCanvasElement>) => {
    if (disabled || event.button !== 0) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    const location = point(event);
    if (tool === "note") {
      const text = noteText.trim();
      if (!text) return;
      const number = commandsRef.current.filter((command) => command.kind === "note").length + 1;
      commandsRef.current.push({ kind: "note", point: location, text, number });
      setNoteText("");
      publish();
      return;
    }
    const stroke: Extract<AnnotationCommand, { kind: "stroke" }> = { kind: "stroke", points: [location] };
    activeStrokeRef.current = stroke;
    commandsRef.current.push(stroke);
  };
  const move = (event: PointerEvent<HTMLCanvasElement>) => {
    const stroke = activeStrokeRef.current;
    if (!stroke) return;
    stroke.points.push(point(event));
    drawOverlay();
  };
  const up = (event: PointerEvent<HTMLCanvasElement>) => {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    if (!activeStrokeRef.current) return;
    activeStrokeRef.current = null;
    publish();
  };

  const undo = () => {
    commandsRef.current.pop();
    publish();
  };
  const clear = () => {
    commandsRef.current = [];
    publish();
  };

  return (
    <section className="annotation-editor" aria-label={t("annotations.title")}>
      <div className="annotation-editor-toolbar">
        <div role="group" aria-label={t("annotations.tools")}>
          <button type="button" className={tool === "draw" ? "active" : ""} aria-pressed={tool === "draw"} onClick={() => setTool("draw")}><Brush size={16} />{t("annotations.draw")}</button>
          <button type="button" className={tool === "note" ? "active" : ""} aria-pressed={tool === "note"} onClick={() => setTool("note")}><MapPin size={16} />{t("annotations.note")}</button>
        </div>
        <div>
          <button type="button" disabled={!commandCount || disabled} onClick={undo} aria-label={t("annotations.undo")}><Undo2 size={16} /></button>
          <button type="button" disabled={!commandCount || disabled} onClick={clear} aria-label={t("annotations.clear")}><RotateCcw size={16} /></button>
        </div>
      </div>
      {tool === "note" && (
        <label className="annotation-note-field">
          <span>{t("annotations.noteLabel")}</span>
          <input value={noteText} onChange={(event) => setNoteText(event.target.value)} placeholder={t("annotations.notePlaceholder")} maxLength={240} />
          <small>{t("annotations.noteHelp")}</small>
        </label>
      )}
      <div className="annotation-editor-stage" style={{ aspectRatio }}>
        <img
          ref={imageRef}
          src={sourceUrl}
          alt={sourceAlt}
          draggable={false}
          onLoad={(event) => {
            setAspectRatio(`${event.currentTarget.naturalWidth} / ${event.currentTarget.naturalHeight}`);
            requestAnimationFrame(resize);
          }}
        />
        <canvas
          ref={canvasRef}
          className={`tool-${tool}`}
          role="application"
          tabIndex={0}
          onPointerDown={down}
          onPointerMove={move}
          onPointerUp={up}
          onPointerCancel={up}
          aria-label={t("annotations.canvas")}
        />
      </div>
      <p>{commandCount ? t("annotations.count", { count: commandCount }) : t("annotations.empty")}</p>
    </section>
  );
});
