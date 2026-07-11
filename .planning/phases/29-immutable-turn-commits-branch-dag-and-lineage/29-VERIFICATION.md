---
phase: 29-immutable-turn-commits-branch-dag-and-lineage
status: passed
verified: 2026-07-11
score: 7/7 requirements complete
---

# Phase 29 Verification

The immutable DAG, exact canonical checkout, save bookmarks, lineage schema,
and transactional/idempotent guards are implemented on `homelab-main`.
BRANCH-03 is now closed. Phase 35 added branch-local visual overrides,
selection history, sibling isolation, and stale-worker publication guards.
Phase 37 added branch-scoped audio assets/jobs and inactive-branch mutation
guards. Canonical state, messages, chapters, RAG, visuals, and audio all have
direct isolation coverage.

Evidence:

- `go test ./... -count=1`: 543 tests passed across 25 packages.
- `go vet ./...`: no issues.
- `cargo fmt --check && cargo test`: 48/48 gateway tests passed.
- `bun run test && bun run build`: 103/103 browser tests and production build passed.
- `playwright test`: 12/12 desktop/mobile browser scenarios passed.
- Visual selection tests cover inherited history, sibling isolation, undo/redo,
  inactive jobs, and stale publication.
- Audio service tests reject inactive-branch retry/cancel mutations and keep
  assets/jobs branch-scoped.
- Live-database migration rehearsal passed FK and integrity checks.
- Forced failure leaves the old state and head unchanged.
- Sibling checkout restores exact state without destroying descendants.

Deployment backup: `/opt/lab/backups/oneday/phase29-20260711/oneday.db`.
The active compose was rebuilt; port 8788 and domain HTTP 200 are healthy.
