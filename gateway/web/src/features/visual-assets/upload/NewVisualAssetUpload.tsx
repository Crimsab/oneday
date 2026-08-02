import { ImagePlus, X } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CustomSelect } from "../../../components/CustomSelect";
import { uploadNewVisualAsset } from "./uploadVisualAsset";

const MAX_BYTES = 10 * 1024 * 1024;
const accepted = new Set(["image/png", "image/jpeg", "image/webp"]);

export function NewVisualAssetUpload({ storyId, onUploaded }: { storyId: string; onUploaded: (assetId: string) => Promise<void> | void }) {
  const { t } = useTranslation("drawer");
  const input = useRef<HTMLInputElement>(null);
  const abort = useRef<AbortController | null>(null);
  const [open, setOpen] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [name, setName] = useState("");
  const [kind, setKind] = useState<"custom" | "world" | "location" | "character">("custom");
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState("");
  const choose = (candidate: File | null) => {
    setError("");
    if (!candidate) return setFile(null);
    if (!accepted.has(candidate.type)) return setError(t("visuals.upload.typeError"));
    if (candidate.size > MAX_BYTES) return setError(t("visuals.upload.sizeError"));
    setFile(candidate); if (!name.trim()) setName(candidate.name.replace(/\.[^.]+$/, ""));
  };
  const submit = async () => {
    if (!file || !name.trim() || progress !== null) return;
    const controller = new AbortController(); abort.current = controller; setProgress(0); setError("");
    try {
      const result = await uploadNewVisualAsset({ storyId, file, displayName: name.trim(), assetKind: kind, signal: controller.signal, onProgress: setProgress });
      await onUploaded(result.asset_id); setOpen(false); setFile(null); setName("");
      if (input.current) input.current.value = "";
    } catch (cause) {
      if (!(cause instanceof DOMException && cause.name === "AbortError")) setError(cause instanceof Error ? cause.message : t("visuals.upload.failed"));
    } finally { abort.current = null; setProgress(null); }
  };
  if (!open) return <button type="button" className="new-visual-asset-trigger" onClick={() => setOpen(true)}><ImagePlus size={15} />{t("visuals.upload.newAsset")}</button>;
  return <section className="new-visual-asset-upload">
    <header><strong>{t("visuals.upload.newAsset")}</strong><button type="button" onClick={() => setOpen(false)} aria-label={t("visuals.upload.cancel")}><X size={15} /></button></header>
    <div className="new-visual-asset-fields">
      <label><span>{t("visuals.upload.name")}</span><input value={name} maxLength={100} onChange={(event) => setName(event.target.value)} /></label>
      <label>
        <span>{t("visuals.upload.kind")}</span>
        <CustomSelect
          value={kind}
          ariaLabel={t("visuals.upload.kind")}
          onChange={(value) => setKind(value as typeof kind)}
          options={[
            { value: "custom", label: t("visuals.upload.kinds.custom") },
            { value: "world", label: t("visuals.upload.kinds.world") },
            { value: "location", label: t("visuals.upload.kinds.location") },
            { value: "character", label: t("visuals.upload.kinds.character") },
          ]}
        />
      </label>
    </div>
    <button type="button" onClick={() => input.current?.click()}>{file ? file.name : t("visuals.upload.choose")}</button>
    <input ref={input} hidden type="file" accept="image/png,image/jpeg,image/webp" onChange={(event) => choose(event.target.files?.item(0) ?? null)} />
    {progress !== null && <progress max={1} value={progress} />}{error && <p role="alert">{error}</p>}
    <div className="visual-asset-upload-actions">{progress !== null && <button type="button" onClick={() => abort.current?.abort()}>{t("visuals.upload.cancel")}</button>}<button type="button" className="primary-action" disabled={!file || !name.trim() || progress !== null} onClick={() => void submit()}>{t("visuals.upload.create")}</button></div>
  </section>;
}
