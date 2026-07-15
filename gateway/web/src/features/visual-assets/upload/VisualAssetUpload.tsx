import { Check, Upload, X } from "lucide-react";
import { useRef, useState, type DragEvent } from "react";
import { useTranslation } from "react-i18next";
import { uploadVisualAssetVersion } from "./uploadVisualAsset";

const ACCEPTED_TYPES = new Set(["image/png", "image/jpeg", "image/webp"]);
const MAX_BYTES = 10 * 1024 * 1024;

export function VisualAssetUpload({ storyId, assetId, onUploaded }: {
  storyId: string;
  assetId: string;
  onUploaded: () => Promise<void> | void;
}) {
  const { t } = useTranslation("drawer");
  const inputRef = useRef<HTMLInputElement>(null);
  const abortRef = useRef<AbortController | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [selectAfterUpload, setSelectAfterUpload] = useState(false);
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);

  const choose = (candidate: File | null) => {
    setDone(false);
    setError("");
    if (!candidate) return setFile(null);
    if (!ACCEPTED_TYPES.has(candidate.type)) return setError(t("visuals.upload.typeError"));
    if (candidate.size > MAX_BYTES) return setError(t("visuals.upload.sizeError"));
    setFile(candidate);
  };
  const drop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    choose(event.dataTransfer.files.item(0));
  };
  const upload = async () => {
    if (!file || progress !== null) return;
    const controller = new AbortController();
    abortRef.current = controller;
    setProgress(0);
    setError("");
    setDone(false);
    try {
      await uploadVisualAssetVersion({ storyId, assetId, file, selectAfterUpload, signal: controller.signal, onProgress: setProgress });
      setDone(true);
      setFile(null);
      if (inputRef.current) inputRef.current.value = "";
      await onUploaded();
    } catch (cause) {
      if (!(cause instanceof DOMException && cause.name === "AbortError")) setError(cause instanceof Error ? cause.message : t("visuals.upload.failed"));
    } finally {
      if (abortRef.current === controller) abortRef.current = null;
      setProgress(null);
    }
  };

  return (
    <section className="visual-asset-upload" aria-label={t("visuals.upload.region")}>
      <div className="visual-asset-dropzone" onDragOver={(event) => event.preventDefault()} onDrop={drop}>
        <Upload size={18} aria-hidden="true" />
        <div><strong>{t("visuals.upload.title")}</strong><small>{t("visuals.upload.hint")}</small></div>
        <button type="button" onClick={() => inputRef.current?.click()} disabled={progress !== null}>{t("visuals.upload.choose")}</button>
        <input ref={inputRef} type="file" accept="image/png,image/jpeg,image/webp" onChange={(event) => choose(event.target.files?.item(0) ?? null)} hidden />
      </div>
      {file && <div className="visual-asset-upload-file"><span>{file.name}</span><small>{formatBytes(file.size)}</small></div>}
      <label className="visual-asset-upload-select"><input type="checkbox" checked={selectAfterUpload} onChange={(event) => setSelectAfterUpload(event.target.checked)} disabled={progress !== null} />{t("visuals.upload.select")}</label>
      {progress !== null && <div className="visual-asset-upload-progress" role="status"><progress max={1} value={progress} /><span>{Math.round(progress * 100)}%</span></div>}
      {error && <p className="visual-asset-upload-error" role="alert">{error}</p>}
      {done && <p className="visual-asset-upload-success" role="status"><Check size={14} />{t("visuals.upload.success")}</p>}
      <div className="visual-asset-upload-actions">
        {progress !== null && <button type="button" onClick={() => abortRef.current?.abort()}><X size={14} />{t("visuals.upload.cancel")}</button>}
        <button type="button" className="primary-action" onClick={() => void upload()} disabled={!file || progress !== null}>{t("visuals.upload.action")}</button>
      </div>
    </section>
  );
}

function formatBytes(bytes: number): string {
  return bytes < 1024 * 1024 ? `${Math.ceil(bytes / 1024)} KiB` : `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}
