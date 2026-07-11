---
phase: 29-immutable-turn-commits-branch-dag-and-lineage
plan: 29-02
status: complete
completed: 2026-07-11
requirements: [BRANCH-01, BRANCH-02, BRANCH-07]
---

# Plan 29.2 Summary

Extended `CommitTurnWithSideEffects` so character/world updates, messages,
lineage binding, canonical event insertion, immutable snapshot creation, and
branch-head advancement share the existing transaction. Failed side effects or
stale heads roll back gameplay state and timeline identity together.

