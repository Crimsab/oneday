---
phase: 29-immutable-turn-commits-branch-dag-and-lineage
status: passed-with-cross-phase-followup
verified: 2026-07-11
score: 6/7 requirements complete
---

# Phase 29 Verification

The immutable DAG, exact canonical checkout, save bookmarks, lineage schema,
and transactional/idempotent guards are implemented on `homelab-main`.
BRANCH-03 remains open only for final derived visual/audio selection isolation
in Phases 35 and 37; canonical state/messages/chapters/RAG isolation passes.

Evidence:

- `go test ./... -count=1`: every Go package passed.
- `cargo fmt --check && cargo test`: 32/32 gateway tests passed.
- `bun test && bun run build`: 82/82 browser tests and build passed.
- Live-database migration rehearsal passed FK and integrity checks.
- Forced failure leaves the old state and head unchanged.
- Sibling checkout restores exact state without destroying descendants.

Deployment backup: `/opt/lab/backups/oneday/phase29-20260711/oneday.db`.
The active compose was rebuilt; port 8788 and domain HTTP 200 are healthy.

