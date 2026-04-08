---
phase: 5
plan: 5.3
title: "/map, /journal, /achievements + dynamic world + context builder"
status: completed
commit: 91b92bc
---

# Plan 5.3 Execution Summary

## What Was Done

### Task 1: Command Registry
- Added `map`/`m` and `achievements`/`a` to `CommandRegistry` in `internal/engine/commands.go`.
- `journal`/`j` was already registered from Phase 5.2.

### Task 2: /map Command
- Created `internal/engine/world.go`:
  - `KnownLocation` struct (name, description, region, discovered_turn)
  - `parseKnownLocations` — handles both plain string arrays and object arrays in KnownLocationsJSON
  - `FormatMapView` — renders discovered locations grouped by region with current location highlighted
  - `AddLocationToWorldState` — adds a location to world state without duplicates, preserving existing format
- Wired `showMap()` into `NarrativeModel.handleCommand` in `narrative.go`

### Task 3: /journal Command
- Created `internal/engine/journal.go`:
  - `FormatJournalView` — reads chapters from DB, renders completed chapters with summaries and in-progress chapter status
- Replaced the previous `GetChapterSummaries`-based `showJournal` with the richer `FormatJournalView` approach

### Task 4: /achievements Command
- Created `internal/storage/achievements.go`: `CreateAchievement`, `ListAchievements` CRUD
- Added `FormatAchievementsView` to `journal.go` — displays achievements with rarity labels and totals by rarity
- Wired `showAchievements()` into `NarrativeModel.handleCommand`
- Shows "No achievements yet" on empty (earning logic is Phase 7)

### Task 5: Dynamic World Updates through Gameplay
- In `sendTurn` (narrator.go): after `ApplyStateChanges`, now also calls `ApplyNarratorStateChanges` so that `world_location_add`, `world_event_add`, `world_faction_standing`, `setting_factions_add/dangers_add/rules_add` all work during regular gameplay turns
- Auto-tracks current location: after every turn, `AddLocationToWorldState` is called with the current location so the map fills naturally as the protagonist moves

### Task 6: NPC Evolution
- Added `npc_desire_update` (and `npc_desires` alias) to `ApplyStateChanges` in `state.go` — works during regular gameplay
- Updated narrator system prompt (`ai/prompts/narrator.go`):
  - Added **Dynamic World Updates** section with all world mutation keys and usage guidance
  - Strengthened NPC evolution guidance: rule #4 now explicitly asks the AI to add `npc_thoughts` and `npc_notes` after every meaningful NPC interaction

### Task 7: Context Builder Finalization
- `BuildContext` signature extended with `lastChapterSummary string` parameter
- `buildStateSummary` now includes:
  - Known locations (KnownLocationsJSON)
  - Faction standings (FactionStandingsJSON)
  - Global events (GlobalEventsJSON)
  - Previous chapter summary (for narrative continuity)
- `sendTurn` fetches the previous chapter summary from DB and passes it to `BuildContext`

### Task 8: Help Text
- Updated `showHelp()` in `narrative.go` with full command list including `/map (/m)`, `/achievements (/a)`, and updated `/narrator` examples

## Files Created
- `internal/engine/world.go`
- `internal/engine/journal.go`
- `internal/storage/achievements.go`

## Files Modified
- `internal/engine/commands.go` — new aliases
- `internal/engine/context.go` — lastChapterSummary param + world state in summary
- `internal/engine/narrator.go` — dynamic world updates + chapter summary fetch
- `internal/engine/state.go` — npc_desire_update handler
- `internal/ai/prompts/narrator.go` — dynamic world + NPC evolution guidance
- `internal/tui/views/narrative.go` — /map, /journal, /achievements, updated help

## Verification
- `go build ./...` passes cleanly
- All 8 tasks implemented and committed (91b92bc)
