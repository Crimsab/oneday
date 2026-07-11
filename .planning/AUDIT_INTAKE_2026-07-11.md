# OneDay Architecture Audit Intake — 2026-07-11

## Source

This intake consolidates the external architecture audit produced after the
July 2026 Oracle bundle review. The full supplied HTML remains the detailed
reference; this file records the actionable project decisions and the findings
confirmed against the live source tree.

## Decision

Preserve the existing Go/Rust canonical-turn kernel. Do not rewrite the engine,
replace SQLite, split into microservices, or prioritize TTS/minigame expansion
before canonical integrity is established.

The implementation order is:

1. Timeline and rollback integrity.
2. Universal canonical truths for identity, forms, places, and world time.
3. Engine-owned outcome resolution and browser/TUI parity.
4. Observable, legible product UX.
5. Branch-aware images and TTS.

## P0 Workstreams

### P0.1 — Snapshot and rollback integrity

- Preserve every NPC field during full rollback, including `discovery_json`.
- Version and validate full snapshots with a checksum and required-collection manifest.
- Make missing/corrupt full snapshots an explicit policy decision; do not silently
  present partial character/world restoration as a full rollback.
- Stage session-file restoration so DB and file state cannot diverge after commit.
- Until branch lineage exists, prevent visual assets/jobs from leaking future state
  after loading an older save.

### P0.2 — Immutable timeline and branch DAG

- Add branch, immutable turn-commit, snapshot, and canonical-event identities.
- Keep `stories.revision` as a concurrency/ETag mechanism, not as timeline identity.
- Scope messages, chapters, RAG, images, and future audio by branch/source commit.
- Evolve manual saves toward named commit bookmarks.

### P0.3 — Character canon

- Separate canonical entity, perceived identity/alias, physical form/controller,
  lifecycle/presentation state, and append-only facts with provenance.
- Keep `npcs` as a compatibility projection while migrating consumers.

### P0.4 — World canon

- Introduce location IDs/aliases and a graph-ready location model.
- Track world clock and optional weather explicitly.
- Never display turn-derived clock/weather heuristics as canonical facts.

### P0.5 — Authoritative outcomes

- Introduce an engine-owned `OutcomeEnvelope` between player intent and narration.
- Use one serializable challenge protocol across TUI, browser, autoplay, and tests.
- Persist seed, difficulty, modifiers, degree, costs, and consequences.

### P0.6 — Causal generation telemetry

- Persist generation run, attempt, and event lineage for narrator, judge, repair,
  reroll, summaries, images, and future TTS.
- Store prompt/profile revisions and hashes without storing private reasoning.

### P0.7 — Browser correctness

- Clear the composer optimistically and restore safely on request failure.
- Normalize the relationship scale contract.
- Hide or rename debug-facing “Live engine” UI.
- Remove fake time/weather and add critical Playwright coverage.

## Confirmed Live Findings

- `internal/engine/save.go` omitted `discovery_json` in `insertNPCRows`, causing
  NPC discovery stage, confidence, aliases, and facts to reset during rollback.
  Fixed on 2026-07-11 with a public save/load regression in
  `internal/engine/save_test.go`.
- `internal/storage/saves.go:HasFullRollbackState` still treats `Story != nil` as
  sufficient proof of a complete snapshot. This is the next focused P0.1 slice.
- The current `.planning/STATE.md` marks milestone v1.1 complete while the roadmap
  still shows stale unchecked entries for phases 24–27. Reconcile roadmap truth
  before opening the next formal milestone; do not infer that those implementations
  are missing solely from the checkboxes because phase summaries exist.

## Next Slice

Define a versioned snapshot envelope and validation contract, then write failing
tests for incomplete and corrupt snapshots before changing load behavior.

