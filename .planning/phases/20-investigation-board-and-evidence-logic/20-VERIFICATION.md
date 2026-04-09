# Phase 20 Verification

## Result

Phase 20 is complete.

## What Was Verified

- Investigation state is now canonical world data with explicit cases, clues, suspects, claims, contradictions, leads, theories, and hidden truths.
- Save/load and rollback restore preserve the full investigation board instead of losing mystery state between sessions.
- AI-authored investigation updates are normalized engine-side, with duplicate collapse, hidden-truth reveal support, theory movement, and graceful handling of malformed payloads.
- Open investigations feed back into narrator context, improving continuity without exposing hidden truths.
- Investigation cases are browseable through the codex, with backlinks from relevant dossiers/fronts/threads and a dedicated `/investigations` command.

## Commands

- `go test ./...`
- `go vet ./...`

## Notes

- The codex/browser integration currently uses codex-style navigation instead of a wholly separate investigation UI, which keeps the UX consistent while still giving cases dedicated space.
- Hidden truths remain stored canonically but intentionally do not render in player-facing codex sections; only engine-side reveal paths can move them into visible evidence.
