# Plan 09-01 Summary: Turn-Flow Reliability + Resume/Choice Lifecycle Fixes

**Completed:** 2026-04-09
**Status:** Done

## Delivered

- Resume/load now reconstructs the last locally persisted turn from stored assistant metadata instead of trying to parse plain assistant prose as raw JSON.
- Assistant turn persistence now stores richer renderer-facing output metadata so future resumes can recover choices, dialogue blocks, callouts, ASCII art, and related narrative UI data without a fresh AI call.
- Choice flow is sanitized before rendering:
  - blank choices dropped
  - duplicate choice text collapsed
  - UI numbering renumbered sequentially
- Literal escaped newline artifacts are normalized before rendering.
- Vitals are clamped so invalid displays like `55/50` do not persist.

## Key Files

- `internal/engine/narrator.go`
- `internal/engine/session.go`
- `internal/engine/state.go`
- `internal/engine/types.go`
- `internal/tui/views/narrative.go`
- `internal/tui/views/narrative_choices.go`
- `internal/tui/components/choicelist.go`

## Notes

- The old synthetic "Welcome back..." fallback remains only as the final fallback when no meaningful persisted turn data exists.
- Resume rendering now restores instantly instead of feeling like a fresh, auto-playing typewriter turn.
