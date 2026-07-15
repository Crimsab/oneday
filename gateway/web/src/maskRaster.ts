import type { MaskPoint } from "./maskGeometry";

export type MaskPaintTool = "brush" | "eraser";
export type MaskCommand =
  | { kind: "stroke"; tool: MaskPaintTool; size: number; points: MaskPoint[] }
  | { kind: "clear" }
  | { kind: "invert" };

export interface MaskReplayPlan {
  checkpoint: ImageData | null;
  checkpointIndex: number;
  commands: MaskCommand[];
}

/** Keeps local paint history independent from OneDay's persisted version selection history. */
export class MaskCommandHistory {
  private commands: MaskCommand[] = [];
  private index = 0;
  private checkpoints = new Map<number, ImageData>();

  constructor(private readonly checkpointInterval = 12) {}

  get canUndo(): boolean { return this.index > 0; }
  get canRedo(): boolean { return this.index < this.commands.length; }

  reset(): void {
    this.commands = [];
    this.index = 0;
    this.checkpoints.clear();
  }

  commit(command: MaskCommand, checkpoint?: ImageData): void {
    this.commands.splice(this.index);
    for (const checkpointIndex of [...this.checkpoints.keys()]) {
      if (checkpointIndex > this.index) this.checkpoints.delete(checkpointIndex);
    }
    this.commands.push(command);
    this.index += 1;
    if (checkpoint && this.index % this.checkpointInterval === 0) {
      this.checkpoints.set(this.index, checkpoint);
    }
  }

  undo(): boolean {
    if (!this.canUndo) return false;
    this.index -= 1;
    return true;
  }

  redo(): boolean {
    if (!this.canRedo) return false;
    this.index += 1;
    return true;
  }

  replayPlan(): MaskReplayPlan {
    let checkpointIndex = 0;
    for (const candidate of this.checkpoints.keys()) {
      if (candidate <= this.index && candidate > checkpointIndex) checkpointIndex = candidate;
    }
    return {
      checkpoint: this.checkpoints.get(checkpointIndex) ?? null,
      checkpointIndex,
      commands: this.commands.slice(checkpointIndex, this.index),
    };
  }
}

export function applyMaskCommand(context: CanvasRenderingContext2D, command: MaskCommand): void {
  const canvas = context.canvas;
  if (command.kind === "clear") {
    context.clearRect(0, 0, canvas.width, canvas.height);
    return;
  }
  if (command.kind === "invert") {
    const image = context.getImageData(0, 0, canvas.width, canvas.height);
    for (let index = 0; index < image.data.length; index += 4) {
      image.data[index] = 255;
      image.data[index + 1] = 255;
      image.data[index + 2] = 255;
      image.data[index + 3] = 255 - image.data[index + 3];
    }
    context.putImageData(image, 0, 0);
    return;
  }
  drawMaskStroke(context, command);
}

export function replayMaskHistory(history: MaskCommandHistory, context: CanvasRenderingContext2D): void {
  const plan = history.replayPlan();
  context.clearRect(0, 0, context.canvas.width, context.canvas.height);
  if (plan.checkpoint) context.putImageData(plan.checkpoint, 0, 0);
  for (const command of plan.commands) applyMaskCommand(context, command);
}

export function drawMaskStroke(
  context: CanvasRenderingContext2D,
  command: Extract<MaskCommand, { kind: "stroke" }>,
): void {
  const [first, ...rest] = command.points;
  if (!first) return;
  drawMaskStrokeSegment(context, command, first, first);
  let previous = first;
  for (const point of rest) {
    drawMaskStrokeSegment(context, command, previous, point);
    previous = point;
  }
}

export function drawMaskStrokeSegment(
  context: CanvasRenderingContext2D,
  command: Extract<MaskCommand, { kind: "stroke" }>,
  from: MaskPoint,
  to: MaskPoint,
): void {
  const pressure = ((from.pressure ?? 1) + (to.pressure ?? 1)) / 2;
  context.save();
  context.globalCompositeOperation = command.tool === "brush" ? "source-over" : "destination-out";
  context.strokeStyle = "rgba(255,255,255,1)";
  context.fillStyle = "rgba(255,255,255,1)";
  context.lineWidth = command.size * (0.45 + pressure * 0.55);
  context.lineCap = "round";
  context.lineJoin = "round";
  context.beginPath();
  context.moveTo(from.x, from.y);
  context.lineTo(to.x, to.y);
  context.stroke();
  if (from.x === to.x && from.y === to.y) {
    context.beginPath();
    context.arc(from.x, from.y, context.lineWidth / 2, 0, Math.PI * 2);
    context.fill();
  }
  context.restore();
}

export function maskHasCoverage(context: CanvasRenderingContext2D): boolean {
  const values = context.getImageData(0, 0, context.canvas.width, context.canvas.height).data;
  for (let index = 3; index < values.length; index += 4) {
    if (values[index] > 0) return true;
  }
  return false;
}

/** Converts offscreen alpha coverage into an opaque L8-equivalent RGBA raster. */
export function coverageRgbaFromAlpha(sourceRgba: Uint8ClampedArray): Uint8ClampedArray {
  const normalized = new Uint8ClampedArray(sourceRgba.length);
  for (let index = 0; index < sourceRgba.length; index += 4) {
    const coverage = sourceRgba[index + 3];
    normalized[index] = coverage;
    normalized[index + 1] = coverage;
    normalized[index + 2] = coverage;
    normalized[index + 3] = 255;
  }
  return normalized;
}

export function exportCoveragePngBase64(coverage: HTMLCanvasElement): string | null {
  const source = coverage.getContext("2d", { willReadFrequently: true });
  if (!source || !maskHasCoverage(source)) return null;
  const output = document.createElement("canvas");
  output.width = coverage.width;
  output.height = coverage.height;
  const context = output.getContext("2d");
  if (!context) return null;
  const sourceImage = source.getImageData(0, 0, coverage.width, coverage.height);
  const normalized = context.createImageData(coverage.width, coverage.height);
  normalized.data.set(coverageRgbaFromAlpha(sourceImage.data));
  context.putImageData(normalized, 0, 0);
  return output.toDataURL("image/png").replace(/^data:image\/png;base64,/, "");
}
