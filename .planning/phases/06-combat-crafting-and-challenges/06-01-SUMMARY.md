---
phase: 6
plan: 6.1
title: "Challenge engine + combat engine core"
status: complete
---

# Summary: Plan 6.1 — Challenge Engine + Combat Engine Core

## Files Created

| File | Description |
|------|-------------|
| `internal/engine/minigame.go` | RPS, Memory, QuickTime, Riddle mini-game resolvers |
| `internal/engine/minigame_test.go` | 14 tests covering all 4 mini-game types (all 9 RPS combos, memory edge cases, timer logic, riddle matching) |
| `internal/engine/challenge_test.go` | 13 tests: stat check boundary, dice roll with modifiers/crits, item check present/absent, skill check with level thresholds, mini-game error |
| `internal/engine/combat.go` | Turn-based combat engine: `CombatEngine`, `ValidateEnemy`, `PlayerAction`, damage formula, behavior patterns, flee/defeat/victory logic |
| `internal/engine/combat_test.go` | 23 tests: enemy validation clamping, all 5 behavior modifiers, HP tracking, victory/defeat conditions, attack detection, player vitals parsing, attribute bonus |
| `internal/ai/prompts/combat.go` | `CombatSystem`, `CombatDefeatPrompt`, `CombatVictoryPrompt` AI prompt builders |

## Files Modified

| File | Change |
|------|--------|
| `internal/engine/state.go` | Added `combat_start` and `crafting_start` cases to `ApplyStateChanges` switch |
| `internal/storage/models.go` | Added `CombatLog` struct |
| `internal/storage/migrations.go` | Added `migrationV5` — `combat_log` table + index; registered as version 5 |
| `internal/storage/stories.go` | Added `InsertCombatLog` and `GetCombatStats` DB methods |

## Key Decisions Made

- `challenge.go` was already implemented in a prior session — only tests were missing
- `combat.go` placed in `internal/engine/` (flat, not `internal/engine/combat/`) per CLAUDE.md naming convention
- `ValidateEnemy` clamps HP 1-999, Attack 0-50, Defense 0-30; defaults behavior to `aggressive`
- Player damage: `weapon_base + (str/3) + d20 - enemy.Defense` (min 0)
- Enemy damage: `enemy.Attack + behaviorModifier + d20 - playerDefense` (min 0)
- Flee via dice roll: d100 >= 50 succeeds; logs `DefeatOutcome = "retreat"`
- `syncPlayerHP` writes combat HP back to `character.StatsJSON` after combat resolves
- `combat_start`/`crafting_start` in `state.go` are signal-only — actual engine start happens in TUI layer

## Verification

- `go build ./...` — success
- `go test ./...` — 131 tests pass across 13 packages
- New tests: 65 in `internal/engine` package
