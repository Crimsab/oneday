import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Image, Paintbrush, ScanLine } from "lucide-react";
import {
  availableVisualOperations,
  operationAcceptsNegativePrompt,
  type EditableImageOperation,
} from "../imageOperations";
import type {
  ImageOperationCapability,
  ImageOperationView,
  VisualAsset,
  VisualAssetOperationRequest,
  VisualAssetsResponse,
} from "../types";
import { MaskEditor, type MaskEditorHandle } from "./MaskEditor";

interface VisualAssetOperationEditorProps {
  asset: VisualAsset;
  sourceVersionId: number;
  sourceUrl: string;
  prompt: string;
  negativePrompt: string;
  routeCapabilities: ImageOperationCapability[];
  operations: ImageOperationView[];
  disabled?: boolean;
  onRun: (payload: VisualAssetOperationRequest) => Promise<VisualAssetsResponse | void>;
}

export function VisualAssetOperationEditor({
  asset,
  sourceVersionId,
  sourceUrl,
  prompt,
  negativePrompt,
  routeCapabilities,
  operations,
  disabled = false,
  onRun,
}: VisualAssetOperationEditorProps) {
  const { t } = useTranslation("image_editing");
  const maskRef = useRef<MaskEditorHandle>(null);
  const capabilities = useMemo(
    () => availableVisualOperations(asset, routeCapabilities),
    [asset, routeCapabilities],
  );
  const [operation, setOperation] = useState<EditableImageOperation | null>(capabilities[0]?.operation ?? null);
  const [hasMask, setHasMask] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setOperation((current) => capabilities.some((capability) => capability.operation === current)
      ? current
      : capabilities[0]?.operation ?? null);
    setError("");
    setHasMask(false);
  }, [asset.id, sourceVersionId, capabilities]);

  if (capabilities.length === 0 || !operation) return null;
  const capability = capabilities.find((item) => item.operation === operation)!;
  const requiresMask = operation === "inpaint";
  const acceptsNegativePrompt = operationAcceptsNegativePrompt(capability);
  const recentOperations = operations.filter((item) =>
    item.asset_id ? item.asset_id === asset.id : item.source_version_id === sourceVersionId,
  ).slice(0, 4);

  const submit = async () => {
    const trimmedPrompt = prompt.trim();
    if (!trimmedPrompt) {
      setError(t("errors.promptRequired"));
      return;
    }
    const mask = requiresMask ? maskRef.current?.exportCoveragePngBase64() ?? null : null;
    if (requiresMask && !mask) {
      setError(t("errors.maskRequired"));
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await onRun({
        operation,
        source_version_id: sourceVersionId,
        prompt: trimmedPrompt,
        negative_prompt: acceptsNegativePrompt && negativePrompt.trim() ? negativePrompt.trim() : undefined,
        mask_png_base64: mask ?? undefined,
        fallback: { mode: "forbid" },
        idempotency_key: crypto.randomUUID(),
      });
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : String(failure));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="visual-operation-editor" aria-labelledby={`visual-operation-${asset.id}`}>
      <div className="visual-operation-heading">
        <div>
          <strong id={`visual-operation-${asset.id}`}>{t("heading")}</strong>
          <p>{t("description")}</p>
        </div>
        <span>{t("sourceVersion", { id: sourceVersionId })}</span>
      </div>
      <div className="visual-operation-tabs" role="radiogroup" aria-label={t("operationLabel")}>
        {capabilities.map((item) => (
          <button
            type="button"
            role="radio"
            aria-checked={operation === item.operation}
            className={operation === item.operation ? "active" : ""}
            key={item.operation}
            onClick={() => { setOperation(item.operation); setError(""); }}
            disabled={disabled || submitting}
          >
            <OperationIcon operation={item.operation} />
            <span><strong>{t(`operations.${item.operation}.label`)}</strong><small>{t(`operations.${item.operation}.description`)}</small></span>
          </button>
        ))}
      </div>
      {requiresMask && (
        <>
          <div className="visual-operation-notice">
            <strong>{t("maskNoticeTitle")}</strong>
            <span>{t(capability.mask?.adherence === "region_constrained" ? "maskRegionConstrained" : "maskBestEffort")}</span>
          </div>
          <MaskEditor
            key={`${asset.id}:${sourceVersionId}`}
            ref={maskRef}
            sourceUrl={sourceUrl}
            sourceAlt={t("sourceAlt", { subject: asset.subject })}
            disabled={disabled || submitting}
            onCoverageChange={setHasMask}
          />
        </>
      )}
      {!acceptsNegativePrompt && negativePrompt.trim() && (
        <p className="visual-operation-muted">{t("negativePromptIgnored")}</p>
      )}
      <div className="visual-operation-submit">
        <p>{t("fallbackForbidden")}</p>
        <button
          type="button"
          className="primary-action"
          disabled={disabled || submitting || !prompt.trim() || (requiresMask && !hasMask)}
          onClick={() => void submit()}
        >
          {submitting ? t("submitting") : t(`operations.${operation}.submit`)}
        </button>
      </div>
      {recentOperations.length > 0 && (
        <div className="visual-operation-jobs" aria-label={t("recentOperations")}>
          {recentOperations.map((item) => (
            <div className={`visual-operation-job ${item.status}`} key={item.id}>
              <span>{t(`statuses.${item.status}`, { defaultValue: item.status })}</span>
              <strong>{t(`operations.${item.operation}.label`, { defaultValue: item.operation })}</strong>
              <small>
                {item.status === "succeeded" && item.result_version_id
                  ? t("resultVersion", { id: item.result_version_id })
                  : item.provider && item.model
                    ? `${item.provider} · ${item.model}`
                    : t("accepted")}
              </small>
              {item.status === "failed" && item.error_summary && <p role="alert">{item.error_summary}</p>}
            </div>
          ))}
        </div>
      )}
      {error && <p className="model-error" role="alert">{error}</p>}
    </section>
  );
}

function OperationIcon({ operation }: { operation: EditableImageOperation }) {
  if (operation === "inpaint") return <Paintbrush size={17} aria-hidden="true" />;
  if (operation === "image_transform") return <ScanLine size={17} aria-hidden="true" />;
  return <Image size={17} aria-hidden="true" />;
}
