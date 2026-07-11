---
phase: 28-versioned-snapshot-integrity-and-safe-restore
plan: 28-01
status: complete
completed: 2026-07-11
requirements: [SNAP-01, SNAP-02, SNAP-03]
---

# Plan 28.1 Summary

Implemented a versioned full-rollback snapshot envelope with a required
collection manifest, canonical identity checks, SHA-256 payload checksum, and
explicit full/legacy/incomplete/incompatible/corrupt validation states.
`SaveGame` now seals the complete payload before atomic publication, and the
round-trip test compares every persisted NPC field including discovery state.

Verification: storage snapshot tests plus engine save/load regressions passed.

