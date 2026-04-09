# Phase 21 Verification

## Result

Phase 21 is complete.

## What Was Verified

- Project clocks are now canonical world data with persistent progress, status, rewards, links, and outcomes for downtime arcs.
- Save/load and rollback restore preserve project boards, so long-arc work survives snapshot operations.
- Downtime advancement now competes with fronts and resources through pressure, costs, and fail-forward fallout instead of feeling free or decorative.
- Active and completed projects feed back into narrator context, so future turns can reference ongoing work and resolved project fallout coherently.
- Projects are browseable through the codex with protagonist summaries, backlinks from related dossiers/fronts/places, and a dedicated `/projects` command.
- Recognized completion rewards now apply durable character or world changes instead of being stored as inert flavor metadata.

## Commands

- `go test ./...`
- `go vet ./...`

## Notes

- Project browsing currently reuses the codex browser rather than introducing a separate downtime-only UI, which keeps navigation consistent with investigations/fronts/dossiers.
- Reward kinds outside the normalized set remain persisted and visible in project entries, but only recognized reward kinds currently apply direct engine-side fallout.
