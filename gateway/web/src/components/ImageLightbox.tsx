import { useEffect, useRef, useState, type PointerEvent, type WheelEvent } from "react";
import { Maximize2, Minus, Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";

interface ImageLightboxProps {
  open: boolean;
  src: string;
  alt: string;
  onClose: () => void;
}

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 8;

export function ImageLightbox({ open, src, alt, onClose }: ImageLightboxProps) {
  const { t } = useTranslation("image_editing");
  const imageRef = useRef<HTMLImageElement>(null);
  const viewRef = useRef({ zoom: 1, x: 0, y: 0 });
  const dragRef = useRef<{ pointerId: number; x: number; y: number; originX: number; originY: number } | null>(null);
  const [zoomLabel, setZoomLabel] = useState(100);

  const render = () => {
    const image = imageRef.current;
    if (!image) return;
    const view = viewRef.current;
    image.style.transform = `translate3d(${view.x}px, ${view.y}px, 0) scale(${view.zoom})`;
    setZoomLabel(Math.round(view.zoom * 100));
  };

  const setZoom = (next: number) => {
    viewRef.current.zoom = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, next));
    if (viewRef.current.zoom <= 1) {
      viewRef.current.x = 0;
      viewRef.current.y = 0;
    }
    render();
  };

  const fit = () => {
    viewRef.current = { zoom: 1, x: 0, y: 0 };
    render();
  };

  useEffect(() => {
    if (!open) return;
    fit();
    const close = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        onClose();
      }
      if (event.key === "+" || event.key === "=") setZoom(viewRef.current.zoom * 1.2);
      if (event.key === "-") setZoom(viewRef.current.zoom / 1.2);
      if (event.key === "0") fit();
    };
    document.addEventListener("keydown", close, true);
    return () => document.removeEventListener("keydown", close, true);
  }, [open, onClose]);

  if (!open) return null;

  const beginDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (viewRef.current.zoom <= 1) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    dragRef.current = {
      pointerId: event.pointerId,
      x: event.clientX,
      y: event.clientY,
      originX: viewRef.current.x,
      originY: viewRef.current.y,
    };
  };
  const moveDrag = (event: PointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current;
    if (!drag || drag.pointerId !== event.pointerId) return;
    viewRef.current.x = drag.originX + event.clientX - drag.x;
    viewRef.current.y = drag.originY + event.clientY - drag.y;
    render();
  };
  const endDrag = (event: PointerEvent<HTMLDivElement>) => {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    if (dragRef.current?.pointerId === event.pointerId) dragRef.current = null;
  };
  const wheel = (event: WheelEvent<HTMLDivElement>) => {
    event.preventDefault();
    setZoom(viewRef.current.zoom * Math.exp(-event.deltaY * 0.0014));
  };

  return (
    <div className="image-lightbox" role="dialog" aria-modal="true" aria-label={t("viewer.title")}>
      <button type="button" className="image-lightbox-backdrop" onClick={onClose} aria-label={t("viewer.close")} />
      <div className="image-lightbox-toolbar">
        <strong>{alt}</strong>
        <div role="group" aria-label={t("viewer.controls")}>
          <button type="button" onClick={() => setZoom(viewRef.current.zoom / 1.2)} aria-label={t("zoomOut")}><Minus size={17} /></button>
          <span>{zoomLabel}%</span>
          <button type="button" onClick={() => setZoom(viewRef.current.zoom * 1.2)} aria-label={t("zoomIn")}><Plus size={17} /></button>
          <button type="button" onClick={fit} aria-label={t("fit")}><Maximize2 size={17} /></button>
          <button type="button" onClick={onClose} aria-label={t("viewer.close")}><X size={18} /></button>
        </div>
      </div>
      <div
        className={`image-lightbox-stage ${zoomLabel > 100 ? "is-zoomed" : ""}`}
        onPointerDown={beginDrag}
        onPointerMove={moveDrag}
        onPointerUp={endDrag}
        onPointerCancel={endDrag}
        onWheel={wheel}
      >
        <img ref={imageRef} src={src} alt={alt} draggable={false} />
      </div>
      <p>{t("viewer.help")}</p>
    </div>
  );
}
