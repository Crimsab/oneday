---
phase: 2
plan: 2.4
title: "Narrative view with streaming typewriter, choices, free input, and status bar"
status: complete
---

# Plan 2.4 Execution Summary

## What Was Built

### `internal/ai/prompts/narrator.go`
- `NarratorSystem()` — builds the full system prompt injecting story name, setting JSON, stats schema, character name/background/stats
- `FirstTurnUser` constant — standard opening message to begin narration
- Prompt instructs the AI to respond only in the structured JSON format matching `engine.NarrativeResponse`

### `internal/engine/narrator.go`
- `Narrator` struct — manages the gameplay AI conversation, holds the full message history
- `NewNarrator()` — instantiates with story, character, world state; auto-builds the system prompt
- `StartNarration()` — sends `FirstTurnUser` to trigger the opening scene
- `SendAction()` — sends any player input (chosen option or free text) to the AI
- `LastModel()` / `LastLatency()` — expose AI metadata for the status bar
- `parseNarrativeFromAI()` — extracts the `json` block and unmarshals directly into `engine.NarrativeResponse` (preserving the `Location` field not present in `ai.NarrativeResponse`)
- Fallback: if JSON parsing fails, wraps raw AI text as a minimal narrative with a single "Continue…" choice
- Updates `world.CurrentLocation` and `world.CurrentTurn` when the narrative reports a location change

### `internal/tui/components/statusbar.go`
- `StatusBarModel` with `SetData()`, `SetWidth()`, `View()` methods
- `StatusBarData` holds `[]Vital`, `Model` string, `Latency` int64
- `Vital` holds `Label`, `Current`, `Max`
- Vital color: green (normal) → accent color (≤50%) → red `DangerText` (≤25%)
- Right-aligns the AI model name and latency; left-aligns vitals
- Uses `theme.StatusBar` style for the container

### `internal/tui/components/choicelist.go`
- `ChoiceListModel` with `SetChoices()`, `SetWidth()`, `HasChoices()`, `Update()`, `View()` methods
- `ChoiceItem` struct (`ID`, `Text`)
- `ChoiceSelectedMsg` carries selected choice back to the parent view
- Number keys `1`–`5` select choices directly without pressing Enter
- `↑`/`↓` / `k`/`j` navigate the cursor; `Enter` confirms highlighted choice

### `internal/tui/views/narrative.go`
- `NarrativeModel` — core gameplay screen embedding:
  - `viewport.Model` — scrollable narrative text area
  - `TypewriterModel` — character-by-character text reveal
  - `StatusBarModel` — bottom vital/AI bar
  - `ChoiceListModel` — numbered AI-suggested choices
  - `textarea.Model` — free-action input
- `StartNarration()` — kicks off the first AI turn (called from app after story creation)
- `Update()` handles:
  - `narrativeResponseMsg` — populates choices, starts typewriter, updates status bar
  - `TypewriterDoneMsg` — scrolls viewport to bottom when reveal finishes
  - `ChoiceSelectedMsg` — triggers next AI turn with choice text
  - `tea.KeyMsg`:
    - `tab` — toggles between choice list and free-input focus
    - `enter` (in input mode) — sends free action text to narrator
    - `1`–`4` — delegates to choice list for direct selection
    - `esc` — no-op (handled by parent app to return to menu)
- `updateStatusBar()` — reads `character.StatsJSON` vitals map (dynamic, not hardcoded HP/Mana)
- `relayout()` — recalculates viewport height on window resize (reserves ~16 rows for chrome)

### `internal/tui/app.go`
- Added `narrative *views.NarrativeModel` field to `App`
- `StoryCreatedMsg` handler now calls `enterNarrativeView(storyID)` instead of returning to menu
- `enterNarrativeView()` — loads `Story`, `Character`, `WorldState` from DB; creates `Narrator`; creates `NarrativeModel`; calls `StartNarration()`
- `Update` routes `ViewNarrative` messages to `narrative.Update()`; `esc` returns to menu
- `View` returns `narrative.View()` when `ViewNarrative` is active
- `WindowSizeMsg` propagated to `narrative.SetSize()` when the model is initialized

## Key Design Decisions

- **No type duplication**: `engine.NarrativeResponse` is used throughout (has `Location` field); `ai.NarrativeResponse` is bypassed in the narrator in favour of direct JSON unmarshal into the engine type
- **Vitals are dynamic**: status bar reads the `vitals` key from `character.StatsJSON` at runtime — no hardcoded stat names
- **Free action always available**: `TAB` toggles focus between the choice list and the textarea; both are rendered at all times after the typewriter finishes
- **Typewriter drives viewport updates**: viewport content is updated on every typewriter tick and on `TypewriterDoneMsg`

## Compilation

```
go build ./...  # exit 0
go vet ./...    # no issues
```

## Flow After This Plan

1. Player finishes story creation → `StoryCreatedMsg` fires
2. App loads story/character/world from DB → creates `Narrator` → creates `NarrativeModel`
3. `StartNarration()` sends first turn to AI
4. AI returns JSON → typewriter reveals narrative text character by character
5. On finish, choices appear; player picks a number or types a free action
6. Loop repeats for each turn
7. `Esc` returns to main menu (story persisted, resumable in Phase 3)
