# Add Choice-Stat Inspect Help

**Captured:** 2026-04-09
**Status:** Completed
**Completed:** 2026-04-09
**Type:** UX enhancement

## Resolution

Phase 10 implemented a keyboard-first inspect/help flow for choice stat badges.

- Trigger: `?` on the selected choice
- Source: active story `stats_schema` plus current character values
- Surface: in-context overlay inside the narrative view

## Delivered In

- `internal/tui/views/narrative.go`
- `internal/tui/views/narrative_choices.go`
- `.planning/phases/10-ambient-ascii-art-and-model-benchmarking/10-02-SUMMARY.md`
