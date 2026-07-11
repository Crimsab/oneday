# Plan 23.2 Summary

## Delivered

- Added a dedicated `Investigations` workspace in `internal/tui/views/investigation_browser.go`.
- Routed `/investigations` to the new workspace instead of the generic codex browser.
- Grouped cases by status and added detail overlays for clues, suspects, contradictions, leads, theories, and linked entities.
- Kept hidden truths out of the player-facing investigation detail view and added focused coverage for that contract.

## Verification

- `go test ./internal/tui/views ./internal/engine`
- `go test ./...`
- `go vet ./...`
