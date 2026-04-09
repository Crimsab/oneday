# Verification 14

## Automated Checks

- `go test ./...`
- `go build ./cmd/oneday`
- `go vet ./...`

## Verified Outcomes

- The main menu now exposes a home-surface achievement archive that lets the player browse stories and inspect each story's unlocked achievements without loading the run.
- Home/archive achievements stay story-scoped through an accordion browser, and in-story `/achievements` reuses the same interaction model while remaining filtered to the active story only.
- `/stats` now opens a protagonist dossier instead of a raw text dump, and `/characters` opens a people-focused browser for protagonist plus known NPC dossiers.
- `/codex` builds descriptive canonical entries for people, places, factions, mysteries, and threads with stacked drill-down navigation instead of accordion-only expansion.
- Dialogue rendering no longer drops or flattens direct speech just because the prose used single quotes; duplicated quoted prose is removed before rendering structured dialogue, and a fallback extractor promotes recognizable quoted speech when dialogue metadata is missing.
