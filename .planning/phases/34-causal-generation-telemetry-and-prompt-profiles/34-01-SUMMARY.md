# Summary 34.1 — Telemetry and prompt schema

Added migration V32 with versioned prompt profiles/revisions, causal generation
runs, ordered provider attempts, and append-only events. Runs retain trace,
parent, story, branch, source commit, message, stage, prompt hash, streaming,
usage, cost, timing, and safe error-class fields without storing prompt bodies.

Transactional storage APIs provide idempotent prompt revision identity, guarded
run/attempt lifecycle transitions, immutable events/revisions, and late binding
of the AI-authored message. Migration and lifecycle tests pass.
