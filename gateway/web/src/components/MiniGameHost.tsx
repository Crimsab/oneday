import { useEffect, useState } from "react";
import type { MiniGameInput, MiniGameInstance, MiniGameKind } from "../types";

const playableKinds: Array<{ kind: MiniGameKind; label: string }> = [
  { kind: "deduction", label: "Deduction" },
  { kind: "negotiation", label: "Negotiation" },
  { kind: "pattern", label: "Pattern" },
  { kind: "bidding", label: "Bidding" },
  { kind: "courtroom", label: "Courtroom" },
  { kind: "comedy", label: "Comedy" },
];

export function MiniGameHost({
  instance,
  busy,
  error,
  onStart,
  onInput,
}: {
  instance: MiniGameInstance | null;
  busy: boolean;
  error: string;
  onStart: (kind: MiniGameKind | null) => Promise<void>;
  onInput: (input: MiniGameInput) => Promise<void>;
}) {
  const [value, setValue] = useState("");
  const [support, setSupport] = useState("");
  const phase = instance?.runtime.phase;

  useEffect(() => {
    setValue(instance?.definition.options?.[0] ?? "");
    setSupport("");
  }, [instance?.id]);

  const submit = () =>
    onInput({
      action: "submit",
      value: value.trim(),
      values: support.split(",").map((item) => item.trim()).filter(Boolean),
    });

  return (
    <section className="minigame-host" aria-label="Challenge host">
      <div className="minigame-host-head">
        <div>
          <span>Challenge Host</span>
          <strong>{instance ? instance.definition.kind : "Choose a timing-free challenge"}</strong>
        </div>
        {instance && <small>{phase} · rev {instance.runtime.revision} · turn {instance.turn}</small>}
      </div>

      {(!instance || phase === "resolved") && (
        <div className="minigame-launchers" aria-label="Start challenge family">
          <button type="button" className="primary-action" disabled={busy} onClick={() => void onStart(null)}>Auto-fit</button>
          {playableKinds.map(({ kind, label }) => (
            <button type="button" key={kind} disabled={busy} onClick={() => void onStart(kind)}>{label}</button>
          ))}
        </div>
      )}

      {instance && phase !== "resolved" && (
        <div className="minigame-play">
          <p>{instance.definition.prompt || "Resolve the challenge using the available information."}</p>
          <label>
            <span>{fieldLabel(instance.definition.kind)}</span>
            {instance.definition.options?.length ? (
              <select value={value} disabled={busy || phase === "paused"} onChange={(event) => setValue(event.target.value)}>
                {instance.definition.options.map((option) => <option key={option} value={option}>{option}</option>)}
              </select>
            ) : (
              <input type={instance.definition.kind === "bidding" ? "number" : "text"} value={value} disabled={busy || phase === "paused"} onChange={(event) => setValue(event.target.value)} />
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
              <button type="button" disabled={busy} onClick={() => void onInput({ action: "resume" })}>Resume</button>
            ) : (
              <button type="button" disabled={busy} onClick={() => void onInput({ action: "pause" })}>Pause</button>
            )}
            <button type="button" className="primary-action" disabled={busy || phase === "paused" || !value.trim()} onClick={() => void submit()}>Resolve challenge</button>
          </div>
        </div>
      )}

      {instance?.runtime.result && (
        <div className={`minigame-result ${instance.runtime.result.outcome?.degree ?? ""}`} aria-live="polite">
          <strong>{degreeLabel(instance.runtime.result.outcome?.degree)}</strong>
          <p>{instance.runtime.result.detail}</p>
        </div>
      )}
      {instance?.definition.rules?.selection_reason && (
        <small className="minigame-selection-reason">Selected because: {instance.definition.rules.selection_reason}</small>
      )}
      {error && <p className="model-error">{error}</p>}
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
