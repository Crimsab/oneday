# Add Choice-Stat Inspect Help

**Captured:** 2026-04-09
**Status:** Pending
**Type:** UX enhancement

## Problem

Choice badges can show story-specific stat labels, but there is no keyboard-friendly way to inspect what a stat means in the current story schema.

## Desired outcome

Add a lightweight inspect/help flow driven from the keyboard, likely via `?` or a focused overlay, instead of brittle mouse hover behavior.

## Notes

- TUI mouse hover is not the preferred solution
- This should integrate with the active story's `stats_schema`
- Keep it lightweight so it does not clutter the main narrative loop

