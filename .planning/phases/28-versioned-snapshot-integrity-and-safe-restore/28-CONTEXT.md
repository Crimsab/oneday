# Phase 28: Versioned Snapshot Integrity and Safe Restore - Context

**Gathered:** 2026-07-11
**Status:** Ready for planning
**Source:** Autonomous architecture-audit intake

<domain>
## Phase Boundary

This phase makes the existing snapshot save/load mechanism truthfully complete,
versioned, validated, and atomic across SQLite state and session-file mirrors.
It does not introduce the immutable branch DAG, which starts in Phase 29.

</domain>

<decisions>
## Implementation Decisions

### Snapshot contract
- New full snapshots carry a format version, deterministic checksum, collection manifest, story/session identity, and canonical turn metadata.
- Completeness is validated structurally and semantically; `Story != nil` is never sufficient evidence of a full rollback payload.
- Checksums cover the canonical serialized payload and exclude the checksum field itself.

### Compatibility policy
- Valid legacy partial snapshots remain loadable only through an explicit legacy result/policy path.
- Missing, corrupt, incompatible, and incomplete full snapshot files are distinct failure classes with actionable messages.
- Existing stories and saves are never silently deleted or rewritten during inspection.

### Atomic restore
- SQLite remains canonical, but session mirrors are staged before the destructive DB restore and atomically swapped only as part of a compensated flow.
- A failed file operation must leave either the pre-load state or the fully restored state recoverable.
- The NPC `discovery_json` regression fixed at phase intake remains permanently covered.

### the agent's Discretion
- Exact internal type names, checksum encoding, temporary-directory naming, and error wrapping may follow existing Go conventions.

</decisions>

<canonical_refs>
## Canonical References

- `.planning/AUDIT_INTAKE_2026-07-11.md` — audit decisions and P0 ordering.
- `.planning/REQUIREMENTS.md` — SNAP-01 through SNAP-05 acceptance contract.
- `internal/storage/saves.go` — current snapshot model and weak completeness check.
- `internal/engine/save.go` — save/load, full/legacy restore, and session-file flow.
- `internal/engine/save_test.go` — public round-trip and compatibility regressions.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `SaveSnapshot`, `SaveMetadata`, and the JSON snapshot file already provide a compatibility envelope to extend.
- `SaveGame` already collects canonical aggregates and `LoadGame` already distinguishes a legacy result.
- The exhaustive save/load integration fixture is the correct public interface for behavioral tests.

### Established Patterns
- SQLite mutations use explicit transactions with wrapped errors.
- Storage structs use JSON tags and zero-value defaults for backward compatibility.
- Focused Go package tests run quickly enough for red-green development.

### Integration Points
- `SaveSnapshot.HasFullRollbackState`, snapshot file serialization, `loadSaveSnapshot`, `LoadGame`, `restoreFullRollback`, and `restoreSessionFiles`.
- Browser/gateway/TUI load-result presentation where legacy/corruption status is surfaced.

</code_context>

<specifics>
## Specific Ideas

Use a small versioned envelope rather than an ontology-sized migration. Preserve
the existing kernel and make invalid states impossible to misclassify.

</specifics>

<deferred>
## Deferred Ideas

Branch commits, branch-scoped assets, commit bookmarks, and alternative
navigation belong to Phase 29 onward.

</deferred>

