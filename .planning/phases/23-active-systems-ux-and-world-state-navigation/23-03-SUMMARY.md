# Plan 23.3 Summary

## Delivered

- Added a player-safe `FrontTrackerBoard` in [internal/engine/front_tracker.go](/opt/lab/docker/oneday/internal/engine/front_tracker.go) that builds one canonical tracker surface from active hooks, visible fronts, derived pressure hotspots, and visible world fallout.
- Kept hidden-front data protected by sanitizing rumored/known front output before it reaches the TUI, while still preserving resolved outcomes for known fronts.
- Added a dedicated `FrontTrackerModel` in [internal/tui/views/front_tracker.go](/opt/lab/docker/oneday/internal/tui/views/front_tracker.go) with sections for hooks, active fronts, resolved fronts, pressure hotspots, and recent fallout.
- Rerouted `/hooks` to the dedicated tracker workspace and added `/fronts` / `/front` aliases plus autocomplete support, so the living-world layer is discoverable without falling back to a generic overlay.
- Extended smoke and unit coverage for engine sanitization, tracker rendering, command aliases, and narrative command routing.

## Verification

- `go test ./internal/engine ./internal/tui/views`
- `go test ./...`
- `go vet ./...`
