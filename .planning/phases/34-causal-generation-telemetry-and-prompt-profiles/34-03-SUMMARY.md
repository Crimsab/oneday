# Summary 34.3 — Diagnostics, export, and retention

AI-authored browser messages now show a compact player-safe provider, model,
latency, token, and streaming summary. Messages with an authoring run expose a
lazy diagnostics drawer containing stage/profile revision, aggregate timing,
and ordered provider attempts without raw prompts, error bodies, secrets, or
reasoning text.

The gateway exposes message diagnostics and bounded story telemetry export.
The export is redacted JSONL suitable for eval tooling and omits internal
request configuration, hidden prompt bodies, metadata blobs, and private
reasoning. Prompt-profile retention policies are enforced at gateway startup;
tests cover redaction, bounded export, retention deletion, safe UI summaries,
API paths, and desktop/mobile rendering.

Deployment used a pre-migration database and binary backup, rebuilt the Go
runtime needed by schema preflight, applied schema V32, rebuilt the gateway
image, and verified the live service on the new `homelab-main` server.
