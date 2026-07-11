---
phase: 3
plan: 3.3
title: "Chat commands: /inventory, /stats, /save, /load, /help, /quit"
status: completed
---

# Plan 3.3 Summary

## What Was Built

### New Files

- **`internal/engine/commands.go`** — Command parser and registry. `ParseCommand` strips the leading `/`, lowercases the name, looks it up in `CommandRegistry` (which maps aliases like `/i`→`inventory`, `/s`→`stats`, `/h`→`help`, `/q`→`quit`). Returns `nil` for non-commands. `IsCommand` checks for `/` prefix.

- **`internal/tui/components/overlay.go`** — Dismissable centered overlay component. Renders a rounded-border box at 60% width / 70% height, centered via `lipgloss.Place`. Supports scroll (up/down), dismissed by Esc/Enter/Space. Emits `OverlayDismissedMsg` on close.

- **`internal/tui/views/saveload.go`** — `SaveLoadModel` for the in-game save picker. Shows save name, turn, location, relative timestamp. Arrow key navigation, Enter to load (emits `SaveLoadSelectedMsg`), Esc to cancel (emits `SaveLoadCancelMsg`).

### Modified Files

- **`internal/engine/narrator.go`** — Added `DB()`, `DataDir()`, `SessionID()` accessors so command handlers can call `engine.SaveGame`/`engine.Autosave`/`engine.ListSaves` without accessing narrator internals.

- **`internal/tui/views/narrative.go`** — Major additions:
  - `overlay components.OverlayModel` field on `NarrativeModel`
  - Enter key handler now checks `engine.IsCommand(text)` before sending to AI
  - `handleCommand` dispatcher routes to per-command methods
  - `showHelp()` — static help text overlay listing all 6 commands with aliases
  - `showInventory()` — parses `InventoryJSON` (list or map format), extracts backpack/equipped/quest sections, reads currency from `StatsJSON` and currency name from story's `StatsSchemaJSON`
  - `showStats()` — parses `StatsJSON` for vitals (current/max), attributes (3-per-row grid), secondary stats; reads traits from `TraitsJSON`/stats, skills from `SkillsJSON`/stats, titles from stats
  - `doSave(args)` — async `engine.SaveGame`, emits `SaveCompleteMsg`
  - `doLoad()` — async `engine.ListSaves`, emits `ShowSaveListMsg`
  - `doQuit()` — async `engine.Autosave` + `CloseSession`, emits `QuitToMenuMsg`
  - `ShowSaveListMsg`, `SaveCompleteMsg`, `QuitToMenuMsg` message types
  - `StoryID()` accessor, `SetStatusMsg()` helper
  - Help line updated to include `/help commands` hint
  - Overlay routes all keys when visible, restores focus on dismiss

- **`internal/tui/app.go`** — Added:
  - `ViewSaveLoad` view constant
  - `saveLoad *views.SaveLoadModel` field
  - Handlers for `QuitToMenuMsg` (clear narrative, return to menu), `ShowSaveListMsg` (switch to save picker or show "No saves found"), `SaveLoadSelectedMsg` (call `loadSaveAndResume`), `SaveLoadCancelMsg` (back to game), `SaveCompleteMsg` (route to narrative)
  - `loadSaveAndResume(storyID, saveID)` helper — calls `engine.LoadGame`, closes current session, creates new session, rebuilds narrator, starts narration
  - `ViewSaveLoad` case in routing switch and `View()` method

## Verification

- `go build ./...` — clean
- `go vet ./...` — no issues
- Command parsing verified: `/i`→inventory, `/s`→stats, `/h`→help, `/q`→quit, `/save My Save`→save with args, non-commands→nil, unknown→`Command{Name:"unknown"}`

## Notes

- Commands never reach the AI or get appended to chat history
- `/quit` uses `Autosave` (not `SaveGame`) to avoid creating a named save on every quit
- `/load` after restore calls `StartNarration()` which gives the AI the restored world state as context — the AI will naturally continue from the loaded point
- The overlay renders full-screen (replaces normal view content) to keep the UI simple without needing lipgloss overlay compositing
