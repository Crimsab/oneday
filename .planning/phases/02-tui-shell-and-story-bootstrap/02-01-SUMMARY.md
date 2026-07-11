---
phase: 2
plan: 2.1
title: "Bubbletea app skeleton, main menu, and theme system"
status: completed
commit: a1a2ee9
---

# Summary: Plan 2.1 — Bubbletea App Skeleton, Main Menu, and Theme System

## What Was Built

### Dependencies Added
- `github.com/charmbracelet/bubbletea v1.3.10` — TUI framework
- `github.com/charmbracelet/bubbles v1.0.0` — reusable TUI components
- `github.com/charmbracelet/lipgloss v1.1.0` — styling and layout

### Files Created
| File | Description |
|------|-------------|
| `internal/tui/theme/theme.go` | Centralized Lipgloss style definitions — warm gold/brown RPG palette with Primary, Secondary, Accent, Danger, Success, Muted, Background, Text, Highlight colors and 11 pre-built styles |
| `internal/tui/keys.go` | Global `KeyMap` struct with Up/Down/Enter/Back/Quit bindings (arrow keys + vim hjkl) |
| `internal/tui/components/logo.go` | ASCII art `OneDay` logo rendered with theme.Logo style |
| `internal/tui/views/menu.go` | `MenuModel` — navigable main menu with four options (New Story, Load Story, Settings, Quit), centered in terminal via `lipgloss.Place` |
| `internal/tui/app.go` | Top-level `App` Bubbletea model: holds `config.Config`, `*storage.DB`, `*ai.Router`; routes `ViewMenu`/`ViewNewStory`/`ViewNarrative`; propagates `WindowSizeMsg` to child views |

### Files Modified
| File | Change |
|------|--------|
| `cmd/oneday/main.go` | Replaced stub with full wiring: loads config, opens DB, creates AI router, launches `tea.NewProgram` with `tea.WithAltScreen()` |
| `go.mod` / `go.sum` | Added 17 new dependencies (bubbletea + transitive) |

## Decisions Made

- **View routing**: enum-based `View` int in `app.go`; child views bubble up `MenuSelectedMsg` rather than directly mutating app state — clean separation of concerns.
- **Key handling**: global `ctrl+c` quit handled in `App.Update` before routing to child views; `q` quit handled inside `MenuModel.Update` via `MenuSelectedMsg`.
- **Window sizing**: `App.Update` intercepts `tea.WindowSizeMsg` and calls `menu.SetSize()` before routing — ensures menu has correct dimensions on first render.
- **Logo**: backtick escaping used to embed the ASCII art inline in Go source without a separate file.

## Acceptance Criteria — All Passed

- [x] `grep "charmbracelet/bubbletea" go.mod` matches
- [x] `grep "charmbracelet/bubbles" go.mod` matches
- [x] `grep "charmbracelet/lipgloss" go.mod` matches
- [x] `internal/tui/theme/theme.go` has `Primary`, `Title`, `StatusBar`, `SelectedItem`, `Border`
- [x] `internal/tui/keys.go` has `KeyMap` with all five bindings
- [x] `internal/tui/components/logo.go` has `Logo()` function
- [x] `internal/tui/views/menu.go` has `MenuModel` with `Init`/`Update`/`View` + four items
- [x] `internal/tui/app.go` has `App` with `cfg`, `db`, `router`, `view`, `menu` + `ViewMenu`/`ViewNewStory`/`ViewNarrative`
- [x] `cmd/oneday/main.go` uses `tea.WithAltScreen()`
- [x] `go build ./cmd/oneday/` exits 0
- [x] `go build ./...` exits 0

## Next Steps

Plan 2.2 will wire "New Story" to a story-creation flow (genre/title/character setup).
