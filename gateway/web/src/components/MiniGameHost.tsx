import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { MiniGameInput, MiniGameInstance, MiniGameKind } from "../types";
import { CustomSelect } from "./CustomSelect";

export function MiniGameHost({
  instance,
  busy,
  error,
  onInput,
}: {
  instance: MiniGameInstance;
  busy: boolean;
  error: string;
  onInput: (input: MiniGameInput) => Promise<void>;
}) {
  const { t } = useTranslation("flow");
  const [value, setValue] = useState("");
  const [support, setSupport] = useState("");
  const phase = instance.runtime.phase;
  const errorId = "minigame-error";

  useEffect(() => {
    setValue(instance.definition.options?.[0] ?? "");
    setSupport("");
  }, [instance.id]);

  const submit = () =>
    onInput({
      action: "submit",
      value: value.trim(),
      values: support.split(",").map((item) => item.trim()).filter(Boolean),
    });

  return (
    <section className="minigame-host" aria-label={t("challengeHost")}>
      <div className="minigame-host-head">
        <div>
          <span>{t("challengeHost")}</span>
          <strong>{instance.definition.kind}</strong>
        </div>
        <small>{phase} · rev {instance.runtime.revision} · turn {instance.turn}</small>
      </div>

      {phase !== "resolved" && (
        <div className="minigame-play">
          <p>{instance.definition.prompt || "Resolve the challenge using the available information."}</p>
          <label>
            <span>{fieldLabel(instance.definition.kind)}</span>
            {instance.definition.options?.length ? (
              <CustomSelect value={value} disabled={busy || phase === "paused"} ariaLabel={fieldLabel(instance.definition.kind)} onChange={setValue} options={instance.definition.options.map((option) => ({ value: option, label: option }))} />
            ) : (
              <input type={instance.definition.kind === "bidding" ? "number" : "text"} value={value} disabled={busy || phase === "paused"} aria-invalid={Boolean(error)} aria-describedby={error ? errorId : undefined} onChange={(event) => setValue(event.target.value)} />
            )}
          </label>
          {(["deduction", "negotiation", "courtroom", "comedy"] as MiniGameKind[]).includes(instance.definition.kind) && (
            <label>
              <span>{instance.definition.kind === "deduction" || instance.definition.kind === "courtroom" ? "Evidence (comma separated)" : "Leverage/callbacks (comma separated)"}</span>
              <input value={support} disabled={busy || phase === "paused"} onChange={(event) => setSupport(event.target.value)} />
            </label>
          )}
          <div className="minigame-actions">
            {phase === "paused" ? (
              <button type="button" disabled={busy} onClick={() => void onInput({ action: "resume" })}>{t("resume")}</button>
            ) : (
              <button type="button" disabled={busy} onClick={() => void onInput({ action: "pause" })}>{t("pause")}</button>
            )}
            <button type="button" className="primary-action" disabled={busy || phase === "paused" || !value.trim()} onClick={() => void submit()}>{t("resolveChallenge")}</button>
          </div>
        </div>
      )}

      {instance.runtime.result && (
        <div className={`minigame-result ${instance.runtime.result.outcome?.degree ?? ""}`} aria-live="polite">
          <strong>{degreeLabel(instance.runtime.result.outcome?.degree)}</strong>
          <p>{instance.runtime.result.detail}</p>
        </div>
      )}
      {instance.definition.rules?.selection_reason && (
        <small className="minigame-selection-reason">{t("selectedBecause", { reason: instance.definition.rules.selection_reason })}</small>
      )}
      {error && <p id={errorId} className="model-error" role="alert">{error}</p>}
    </section>
  );
}

function fieldLabel(kind: MiniGameKind): string {
  if (kind === "deduction") return "Conclusion";
  if (kind === "negotiation") return "Approach";
  if (kind === "pattern") return "Pattern answer";
  if (kind === "bidding") return "Offer";
  if (kind === "courtroom") return "Procedural move";
  if (kind === "comedy") return "Comedy move";
  return "Response";
}

function degreeLabel(degree?: string): string {
  return degree ? degree.replaceAll("_", " ") : "Resolved";
}
