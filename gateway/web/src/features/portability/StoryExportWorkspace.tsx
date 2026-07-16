import { Copy, Download, FileArchive, FileText, PackageOpen } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { getStoryEpub, getStoryExport, getTelemetryExport } from "../../api";
import { CustomSelect } from "../../components/CustomSelect";
import { DialogDrawerShell } from "../../components/dialog/DialogDrawerShell";
import { exportArchive, exportTemplate, type ArchiveOptions, type ReadableFormat, type ReadingMode } from "./portabilityApi";
import { encodeTemplateCode } from "./templateCode";

const readableFormats: ReadableFormat[] = ["markdown", "html", "txt", "json", "epub"];
const defaultArchiveOptions: ArchiveOptions = { history: true, saves: true, visual_assets: true, audio: true, translations: true, world_detail: true };

interface StoryExportWorkspaceProps {
  storyId: string;
  includeTechnical?: boolean;
  compact?: boolean;
}

export function StoryExportWorkspace({ storyId, includeTechnical = true, compact = false }: StoryExportWorkspaceProps) {
  const { t } = useTranslation(["portability", "surfaces"]);
  const [format, setFormat] = useState<ReadableFormat>("markdown");
  const [readingMode, setReadingMode] = useState<ReadingMode>("original");
  const [targetLanguage, setTargetLanguage] = useState("en");
  const [archiveOptions, setArchiveOptions] = useState<ArchiveOptions>(defaultArchiveOptions);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const run = async (action: () => Promise<void>) => {
    setBusy(true);
    setError("");
    setNotice("");
    try { await action(); }
    catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)); }
    finally { setBusy(false); }
  };

  const exportReadable = () => run(async () => {
    if (format === "epub") {
      const result = await getStoryEpub(storyId, targetLanguage, readingMode);
      downloadBlob(result.blob, result.filename);
      return;
    }
    const result = await getStoryExport(storyId, format, targetLanguage, readingMode);
    const bytes = result.encoding === "base64" ? Uint8Array.from(atob(result.content), (character) => character.charCodeAt(0)) : result.content;
    downloadBlob(new Blob([bytes], { type: result.content_type || (format === "json" ? "application/json" : "text/markdown") }), result.filename);
  });

  const exportPortable = () => run(async () => {
    const result = await exportArchive(storyId, archiveOptions);
    downloadBlob(result.blob, result.filename);
  });

  const exportWorld = (asCode: boolean) => run(async () => {
    const result = await exportTemplate(storyId);
    if (asCode) {
      await navigator.clipboard.writeText(await encodeTemplateCode(result.text));
      setNotice(t("portability:codeCopied"));
      return;
    }
    downloadBlob(new Blob([result.text], { type: "application/json" }), result.filename);
  });

  const exportTechnical = (kind: "replay" | "telemetry") => run(async () => {
    if (kind === "telemetry") {
      const result = await getTelemetryExport(storyId);
      downloadBlob(new Blob([result.content], { type: "application/x-ndjson" }), result.filename);
      return;
    }
    const result = await getStoryExport(storyId, "replay");
    downloadBlob(new Blob([result.content], { type: result.content_type || "application/json" }), result.filename);
  });

  return (
    <div className={`story-export-workspace ${compact ? "compact" : ""}`}>
      <section className="story-export-section">
        <header><FileText size={17} /><div><strong>{t("portability:readableCopy")}</strong><small>{t("portability:readableCopyHelp")}</small></div></header>
        <div className="story-export-controls">
          <label><span>{t("portability:format")}</span><CustomSelect value={format} ariaLabel={t("portability:format")} onChange={(value) => setFormat(value as ReadableFormat)} options={readableFormats.map((value) => ({ value, label: value.toUpperCase() }))} /></label>
          <label><span>{t("portability:languageVersion")}</span><CustomSelect value={readingMode} ariaLabel={t("portability:languageVersion")} onChange={(value) => setReadingMode(value as ReadingMode)} options={[{ value: "original", label: t("portability:original") }, { value: "translated", label: t("portability:translated") }, { value: "bilingual", label: t("portability:bilingual") }]} /></label>
          {readingMode !== "original" && <label><span>{t("portability:targetLanguage")}</span><input value={targetLanguage} onChange={(event) => setTargetLanguage(event.target.value)} placeholder="en" /></label>}
          <button type="button" className="primary" disabled={busy || (readingMode !== "original" && !targetLanguage.trim())} onClick={() => void exportReadable()}><Download size={15} />{t("portability:download")}</button>
        </div>
      </section>

      <section className="story-export-section">
        <header><FileArchive size={17} /><div><strong>{t("portability:portableArchive")}</strong><small>{t("portability:portableArchiveHelp")}</small></div></header>
        <div className="story-export-options">
          {(Object.keys(archiveOptions) as Array<keyof ArchiveOptions>).map((key) => <label key={key}><input type="checkbox" checked={archiveOptions[key]} onChange={(event) => setArchiveOptions((value) => ({ ...value, [key]: event.target.checked }))} />{t(`portability:archiveOptions.${key}`)}</label>)}
        </div>
        <button type="button" disabled={busy} onClick={() => void exportPortable()}><PackageOpen size={15} />{t("portability:downloadArchive")}</button>
      </section>

      <section className="story-export-section">
        <header><Copy size={17} /><div><strong>{t("portability:worldTemplate")}</strong><small>{t("portability:worldTemplateHelp")}</small></div></header>
        <div className="story-export-actions"><button type="button" disabled={busy} onClick={() => void exportWorld(false)}><Download size={15} />{t("portability:downloadTemplate")}</button><button type="button" disabled={busy} onClick={() => void exportWorld(true)}><Copy size={15} />{t("portability:copyCode")}</button></div>
      </section>

      {includeTechnical && <details className="story-export-technical"><summary>{t("surfaces:history.technical")}</summary><div><button type="button" disabled={busy} onClick={() => void exportTechnical("replay")}>{t("surfaces:history.replay")}</button><button type="button" disabled={busy} onClick={() => void exportTechnical("telemetry")}>{t("surfaces:history.telemetry")}</button></div></details>}
      {notice && <p className="inline-notice" role="status">{notice}</p>}
      {error && <p className="inline-error" role="alert">{error}</p>}
    </div>
  );
}

export function StoryExportDialog({ storyId, storyName, onClose }: { storyId: string; storyName: string; onClose: () => void }) {
  const { t } = useTranslation("portability");
  return <DialogDrawerShell title={t("exportTitle", { name: storyName })} className="story-export-drawer" onClose={onClose}><StoryExportWorkspace storyId={storyId} /></DialogDrawerShell>;
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.hidden = true;
  document.body.append(link);
  link.click();
  link.remove();
  globalThis.setTimeout(() => URL.revokeObjectURL(url), 0);
}
