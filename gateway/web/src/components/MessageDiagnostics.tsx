import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { getMessageDiagnostics } from "../api";
import type { GenerationDiagnostics, JsonValue, MessageView } from "../types";

export function MessageDiagnostics({ message }: { message: MessageView }) {
  const summary = generationSummary(message.metadata);
  const generation = objectValue(objectValue(message.metadata).generation);
  const hasRun = stringValue(generation.run_id) !== "";
  const [diagnostics, setDiagnostics] = useState<GenerationDiagnostics | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  if (message.role !== "assistant" || (!summary && !hasRun)) return null;

  if (!hasRun) {
    return <div className="generation-diagnostics-inline" aria-label="Generation debug summary">{summary}</div>;
  }

  const load = async () => {
    if (!hasRun || loading || diagnostics) return;
    setLoading(true);
    setError("");
    try {
      setDiagnostics(await getMessageDiagnostics(message.story_id, message.id));
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setLoading(false);
    }
  };

  return (
    <details className="generation-diagnostics" onToggle={(event) => {
      if (event.currentTarget.open) void load();
    }}>
      <summary>
        <span>{summary || "Generation trace"}</span>
        <ChevronDown size={13} aria-hidden="true" />
      </summary>
      {loading && <p className="diagnostics-state">Loading diagnostics…</p>}
      {error && <p className="inline-error" role="alert">Diagnostics unavailable: {error}</p>}
      {diagnostics && <DiagnosticsBody diagnostics={diagnostics} />}
    </details>
  );
}

function DiagnosticsBody({ diagnostics }: { diagnostics: GenerationDiagnostics }) {
  return (
    <div className="diagnostics-body">
      <div className="diagnostics-run">
        <span><strong>{diagnostics.stage}</strong> · {diagnostics.status}</span>
        <span>{diagnostics.prompt_profile || "unprofiled"} rev {diagnostics.prompt_revision}</span>
        <span>Total {formatDuration(diagnostics.duration_ms)}</span>
        <span>TTFT {formatDuration(diagnostics.ttft_ms)}</span>
        <span>{formatTokens(diagnostics.usage.total_tokens)}</span>
        <span>{diagnostics.observed_streaming ? "streamed" : "not streamed"}</span>
      </div>
      <ol className="diagnostics-attempts" aria-label="Provider attempts">
        {diagnostics.attempts.map((attempt) => (
          <li key={`${diagnostics.run_id}-${attempt.sequence}`}>
            <header>
              <strong>{attempt.provider || "unknown provider"}</strong>
              <span>{attempt.status}</span>
            </header>
            <p>{attempt.resolved_model || attempt.requested_model || "default model"}</p>
            <small>
              {attempt.observed_streaming ? `streamed · TTFT ${formatDuration(attempt.ttft_ms)}` : "not streamed"}
              {` · ${formatDuration(attempt.duration_ms)} · ${formatTokens(attempt.usage.total_tokens)}`}
              {attempt.usage.cost_usd > 0 ? ` · $${attempt.usage.cost_usd.toFixed(4)}` : ""}
            </small>
            {(attempt.retry_reason || attempt.error_class) && (
              <small className="diagnostics-cause">{[attempt.retry_reason, attempt.error_class].filter(Boolean).join(" · ")}</small>
            )}
          </li>
        ))}
      </ol>
    </div>
  );
}

export function generationSummary(metadata: JsonValue): string {
  const value = objectValue(metadata);
  const usage = objectValue(value.usage);
  const parts = [stringValue(value.provider), displayModelName(stringValue(value.model))].filter(Boolean);
  const latency = numberValue(value.latency_ms);
  const tokens = numberValue(usage.total_tokens);
  if (latency > 0) parts.push(formatDuration(latency));
  if (tokens > 0) parts.push(formatTokens(tokens));
  if (value.streamed === true) parts.push("streamed");
  return parts.join(" · ");
}

export function displayModelName(model: string): string {
  return model
    .replace(/^chatgpt-/, "")
    .replace(/-20\d{2}-\d{2}-\d{2}$/, "");
}

function objectValue(value: JsonValue | undefined): Record<string, JsonValue> {
  return value && typeof value === "object" && !Array.isArray(value) ? value : {};
}

function stringValue(value: JsonValue | undefined): string {
  return typeof value === "string" ? value.trim() : "";
}

function numberValue(value: JsonValue | undefined): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function formatDuration(value: number): string {
  if (value <= 0) return "0 ms";
  return value < 1000 ? `${Math.round(value)} ms` : `${(value / 1000).toFixed(value < 10000 ? 1 : 0)} s`;
}

function formatTokens(value: number): string {
  return `${Math.max(0, Math.round(value)).toLocaleString()} tokens`;
}
