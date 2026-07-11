---
phase: 28-versioned-snapshot-integrity-and-safe-restore
plan: 28-03
status: complete
completed: 2026-07-11
requirements: [SNAP-03, SNAP-04]
---

# Plan 28.3 Summary

Replaced post-commit destructive session-file restoration with a staged,
story-local atomic swap. The old session tree remains as a backup until the
SQLite transaction commits; commit failure invokes filesystem compensation.
Invalid snapshot paths fail before DB mutation, and tests prove both the public
failure behavior and direct backup restoration.

Verification: the complete Go suite, Rust gateway suite, browser suite/build,
live image rebuild, port inspection, logs, and HTTP `200` check passed on
`homelab-main`.

