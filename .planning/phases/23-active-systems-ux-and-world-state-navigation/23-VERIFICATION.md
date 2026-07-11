# Phase 23 Verification

## Scope

Phase 23 promoted systemic world-state surfaces into first-class TUI workspaces and then unified navigation between them.

Completed plans:

- `23.1` dedicated projects workspace
- `23.2` dedicated investigations workspace
- `23.3` dedicated fronts and fallout tracker
- `23.4` active-system navigation shortcuts and live callouts

## Automated Checks

- `go test ./internal/engine ./internal/tui/views`
- `go test ./...`
- `go vet ./...`

## Verified Outcomes

- `/projects` opens a dedicated workspace instead of dropping players into generic codex browsing.
- `/investigations` opens a dedicated mystery workspace that keeps hidden truths out of the player-facing surface.
- `/hooks` and `/fronts` open a dedicated tracker for hooks, visible fronts, hotspots, and fallout.
- Players can jump between the active-system workspaces and codex with `P / I / F / C` while staying inside the play loop.
- Turn-delta rendering now points players at the right workspace when fronts, projects, or investigations changed on that turn.

## Residual Notes

- Cross-links are currently command/shortcut based, not clickable inline links inside the dedicated system detail overlays.
- The old text overlay formatter for tracker data still exists for compatibility, but the primary player-facing path is now the dedicated tracker workspace.
