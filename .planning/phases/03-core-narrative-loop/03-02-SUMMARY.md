---
phase: 3
plan: 3.2
title: "Full game loop, state changes, save/load system, and load story menu"
status: complete
completed_at: "2026-04-07"
---

# Plan 3.2 Execution Summary

## What Was Built

### Task 1 — State Change Applicator (`internal/engine/state.go`)
- `ApplyStateChanges()`: parses AI `state_changes` map, applies to `Character.StatsJSON` and `WorldState`
- Handles: `vitals` (nested current/max), `attributes`, `secondary`, `location`, `currency`, `inventory_add`, `inventory_remove`
- HP and attribute values clamped to >= 0
- Returns `[]StateChange` describing each applied change
- `internal/storage/stories.go` extended with: `UpdateCharacterStats()`, `UpdateWorldState()`, `UpdateStoryTimestamp()`

### Task 2 — Full Game Loop (`internal/engine/narrator.go`)
- `sendTurn` now implements the complete loop:
  1. Detect input type (`choice` vs `free_action`)
  2. Build context via `BuildContext()`
  3. Call AI via `router.Complete()`
  4. Parse response with `parseNarrativeFromAI()`
  5. Apply `state_changes` via `ApplyStateChanges()`
  6. Sync location from AI response
  7. Persist character stats + world state to DB
  8. Append turn to JSONL + DB via `session.AppendTurn()`
  9. Expose `ShouldAutosave()` and `AutosaveCmd()` for TUI-driven autosave
- `NewNarrator()` signature extended: `dataDir string`, `autosaveEvery int`
- `AutosaveCompleteMsg` struct defined for Bubbletea message passing

### Task 3 — Save/Load System
- `internal/engine/save.go`: `SaveGame()`, `LoadGame()`, `ListSaves()`, `Autosave()`
  - `SaveGame`: serializes char + world to JSON, writes `{dataDir}/stories/{id}/saves/{saveID}.json`, inserts DB row
  - `LoadGame`: reads save from DB, deserializes, restores char + world in DB
  - `Autosave`: deletes previous autosave, creates new one named "autosave"
- `internal/storage/saves.go`: `CreateSave()`, `GetSave()`, `ListSaves()`, `DeleteSave()`, `GetAutosave()`
- `internal/storage/migrations.go`: migration v2 creates `saves` table + index

### Task 4 — Load Story TUI View (`internal/tui/views/loadstory.go`)
- `LoadStoryModel`: lists stories with name, genre (extracted from JSON), last-played relative time
- Navigation: up/down, Enter to select, Esc to go back
- Empty state: "No stories found. Create a new one!"
- Messages: `StorySelectedMsg`, `LoadStoryBackMsg`
- `internal/tui/app.go`: added `ViewLoadStory`, handles `ActionLoadStory`, `StorySelectedMsg`, `LoadStoryBackMsg`

### Task 5 — Autosave Wiring (`internal/tui/views/narrative.go`)
- `NarrativeModel` gains `statusMsg` + `statusExpiry` fields
- After each successful AI response, calls `maybeAutosaveCmd()` → fires `narrator.AutosaveCmd()` as a `tea.Cmd`
- Handles `AutosaveCompleteMsg`: shows "Autosaved" status for 2 seconds, then clears via `clearStatusMsg`
- `clearStatusMsg` internal type for deferred clear

### Task 6 — Story Timestamp Update
- `enterNarrativeView` in `app.go` calls `db.UpdateStoryTimestamp(storyID)` when opening a story
- Ensures `ListStories()` returns stories in most-recently-played order

## Files Changed

| File | Status |
|------|--------|
| `internal/engine/state.go` | Created |
| `internal/engine/save.go` | Created |
| `internal/engine/narrator.go` | Rewritten |
| `internal/engine/session.go` | Updated (ChatOutput.StateChanges) |
| `internal/storage/stories.go` | Extended (3 new methods) |
| `internal/storage/saves.go` | Created |
| `internal/storage/migrations.go` | Extended (migrationV2) |
| `internal/tui/app.go` | Rewritten (ViewLoadStory, new routing) |
| `internal/tui/views/loadstory.go` | Created |
| `internal/tui/views/narrative.go` | Updated (autosave, statusMsg) |

## Verification

- `go build ./...` — exit 0, no errors
- `go vet ./...` — no issues
- All 6 tasks from the plan implemented
