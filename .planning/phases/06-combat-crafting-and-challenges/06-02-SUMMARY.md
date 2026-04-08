---
phase: 6
plan: 6.2
status: complete
commit: d29ad39
---

# Summary: Plan 6.2 — Combat TUI View + Crafting Engine + Crafting TUI View

## Files Created

| File | Description |
|------|-------------|
| `internal/tui/components/hpbar.go` | HP bar component with colored fill (green/yellow/red thresholds) |
| `internal/tui/views/combat.go` | Full combat TUI view (CombatModel) |
| `internal/engine/crafting.go` | AI-conversation crafting engine |
| `internal/tui/views/crafting.go` | Two-column crafting TUI view (CraftingModel) |
| `internal/ai/prompts/crafting.go` | Crafting system prompt (no hardcoded recipes) |

## Files Modified

| File | Changes |
|------|---------|
| `internal/tui/theme/theme.go` | Added CombatHeader, CombatTurn, HPBar styles, CraftingHeader, InventorySidebar, RecipeItem |
| `internal/tui/views/narrative.go` | Added combat/crafting sub-view fields, CombatStart detection, `/craft` command, startCrafting(), updated View() to delegate |

## What Was Built

### HP Bar Component
- `HPBar{Label, Current, Max, Width}` — renders `Label: [████░░░░] 15/20`
- Color: green (>50%), yellow/gold (25–50%), red (<=25%)
- Empty fill in muted grey

### Combat TUI View (CombatModel)
- Header: "COMBAT — Turn N" with danger/gold styling
- Side-by-side HP bars: player left, enemy right
- Scrollable typewriter-animated narrative viewport
- Choice list (AI-generated) + free input textarea (always available)
- Tab toggles focus between choices and free input
- Esc attempts flee action
- Emits `CombatEndMsg{Summary, Victory}` when combat resolves
- Auto-wired: NarrativeModel detects `nr.CombatStart != nil` after any AI response and transitions to CombatModel

### Crafting Engine (CraftingEngine)
- Opens a `crafting_*.jsonl` sub-session via `session.OpenSubSession("crafting")`
- `SendMessage(ctx, msg)` → builds crafting system prompt with current inventory/skills/recipes → calls AI → parses `CraftingResponse`
- If feasible: removes consumed materials, adds new item to inventory, saves recipe to `KnownRecipesJSON`
- Duplicate recipe detection (case-insensitive)
- `GetKnownRecipes(char)` helper for reading recipe list

### Crafting AI Prompt
- Zero hardcoded recipes — AI evaluates each request against inventory, skills, world rules
- Items get narrative effects only (no numerical stats)
- Response schema: `{feasible, narrative, item{name,description,effect,materials}, missing, alternatives, choices}`

### Crafting TUI View (CraftingModel)
- Two-column layout: 70% chat (left) + 30% inventory sidebar (right)
- Sidebar: current backpack items + known recipes (updated live after successful craft)
- Single-column fallback for narrow terminals (<80 cols)
- Esc or "exit" choice closes crafting and emits `CraftingEndMsg`
- Player-initiated via `/craft` command in narrative view

### Narrative Integration
- `NarrativeModel` gained `combatView *CombatModel`, `inCombat bool`, `craftingView *CraftingModel`, `inCrafting bool`
- When `inCombat`: all messages delegated to combat view; `CombatEndMsg` transitions back and appends summary
- When `inCrafting`: all messages delegated to crafting view; `CraftingEndMsg` transitions back and notes crafted item
- View() delegates rendering to the active sub-view
- `/craft` command added to help text

## Verification

- `go build ./...` — passes clean
- All 8 tasks completed across 3 waves
- Combat, crafting, and HP bar components are independent of each other (clean package boundaries)
