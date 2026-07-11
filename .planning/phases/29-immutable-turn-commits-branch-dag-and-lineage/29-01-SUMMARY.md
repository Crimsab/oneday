---
phase: 29-immutable-turn-commits-branch-dag-and-lineage
plan: 29-01
status: complete
completed: 2026-07-11
requirements: [BRANCH-01, BRANCH-02, BRANCH-06]
---

# Plan 29.1 Summary

Added forward-only SQLite migrations for named branches, immutable turn commits,
full commit snapshots, canonical events, save bookmarks, generation traces, and
audio artifacts. Existing stories and dependent rows receive a `main` branch,
root commit, and compatible lineage without rewriting canonical content.

