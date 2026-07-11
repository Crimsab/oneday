---
phase: 29-immutable-turn-commits-branch-dag-and-lineage
milestone: v2.0
requirements: [BRANCH-01, BRANCH-02, BRANCH-03, BRANCH-04, BRANCH-05, BRANCH-06, BRANCH-07]
status: active
---

# Phase 29 Context

## Boundary

Add immutable timeline identity around the existing SQLite transaction kernel.
Do not replace `CommitTurnWithSideEffects`, the durable idempotency claim, or
`stories.revision`; extend the same transaction so a successful turn advances
exactly one branch head.

## Decisions

- `stories.revision` remains a monotonic concurrency ETag and is never used as a commit ID.
- Every story is lazily or migrationally bootstrapped with one named `main` branch and a root commit.
- Commit IDs and branch IDs are opaque UUIDs. Parent links form a DAG; branch heads are mutable pointers to immutable commits.
- Each commit owns a full canonical materialization snapshot sufficient for exact checkout.
- Checkout moves only the selected branch head/current materialization and never deletes descendants or siblings.
- Aggregate rows gain nullable branch/source-commit lineage for safe legacy migration.
- Manual saves remain file-compatible and additionally resolve to named commit bookmarks.
- Head advancement and checkout require expected head/revision checks and are transactionally idempotent.

## Canonical Snapshot Scope

Story metadata, protagonist, world state, NPCs, active session, chat messages,
chapters, RAG chunks, and save metadata. Visual/audio/telemetry tables receive
lineage columns now or in their owning later phase, but checkout filtering is
defined by branch/source commit rather than turn number.

## Deferred Surface

Browser/TUI controls are Phase 33 (BRANCH-08). Phase 29 exposes storage and
engine contracts only.

