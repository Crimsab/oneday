# Plan 23.4 Summary

## Delivered

- Added shared active-system navigation shortcuts in the narrative runtime so players can jump between `projects`, `investigations`, `fronts`, and `codex` with `P / I / F / C` while those surfaces are open.
- Updated the dedicated browser views and codex hints so the cross-navigation flow is visible inside the UI instead of living only in slash-command memory.
- Extended turn-delta classification and rendering so front, project, and investigation changes emit player-facing navigation hints in the `What changed this turn?` section.
- Added lightweight live status callouts when active-system state changes, making fronts, investigations, and projects feel immediately inspectable after a turn resolves.
- Added integration coverage for switching between tracker, projects, investigations, and codex during live play.

## Verification

- `go test ./internal/engine ./internal/tui/views`
- `go test ./...`
- `go vet ./...`
