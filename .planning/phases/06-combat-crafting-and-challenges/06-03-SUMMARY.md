---
phase: 6
plan: 6.3
title: "Challenge TUI + Narrative Integration"
status: complete
commit: 14d299e
---

# Summary: Plan 6.3 — Challenge TUI + Narrative Integration

## What Was Built

### TUI Components (Wave 1)

**`internal/tui/components/dice.go`**
- Animated d100 dice roll component
- 30-frame animation at 50ms intervals, slows in last 5 frames for dramatic effect
- After animation: shows full breakdown (roll, each modifier with source/value, total, difficulty)
- Green PASSED / red FAILED display
- Emits `DiceResultMsg` on dismissal

**`internal/tui/components/rps.go`**
- Rock-Paper-Scissors 3-phase flow: choose → reveal → result
- Arrow key selection, Enter to confirm
- 3-frame "thinking" animation for AI reveal (300ms each)
- Emits `RPSResultMsg{Passed, Draw}`

**`internal/tui/components/memory.go`**
- Memory sequence show/input/result phases
- Show phase: displays symbols one at a time (800ms each) — up/down/left/right arrows
- Input phase: progress bar showing filled (green correct) vs pending slots
- Wrong key = immediate fail with correct sequence highlighted
- Emits `MemoryResultMsg`

**`internal/tui/components/quicktime.go`**
- Timed key-press challenge with 100ms tick resolution
- Ready phase (800ms) → active phase (countdown bar depletes) → result
- Countdown bar changes color: green > 60%, amber > 30%, red below
- Any key press during active phase = pass
- Emits `QuickTimeResultMsg`

**`internal/tui/components/riddle.go`**
- Text input riddle with fuzzy matching (case-insensitive, substring)
- Shows riddle text wrapped at 42 chars
- Esc to skip (treated as fail), Enter to submit
- Wrong answer reveals the correct answer
- Emits `RiddleResultMsg`

### Theme (`internal/tui/theme/theme.go`)
Added challenge-specific colors: `DiceGold`, `RPSPurple`, `MemoryTeal`, `QuickTimeOrange`, `RiddleCyan`
Added challenge styles: `ChallengeOverlay`, `DicePassed`, `DiceFailed`, `RPSHeader`, `MemorySymbol`, `MemoryCorrect`, `MemoryWrong`, `QuickTimePrompt`, `QuickTimeBar`, `RiddleText`

### Challenge View Coordinator (`internal/tui/views/challenge.go`)
- `ChallengeView` dispatches to the correct component based on `spec.Type`
- Passive challenges (stat/item/skill/relationship): resolved immediately via engine, shown as brief notification for 1.8s, then auto-emits `ChallengeResolvedMsg`
- `dice_roll`: pre-computes result via engine, passes values to `DiceModel` for pure visual animation
- `mini_game`: routes to rps/memory/quicktime/riddle by `spec.MiniGame` field
- All component result messages caught and converted to `ChallengeResolvedMsg{Spec, Result}`

### Narrative Integration (`internal/tui/views/narrative.go`)
- Added fields: `challengeView`, `inChallenge`, `pendingChallenges`
- Challenge delegation block (before main switch): routes all updates/keys to `ChallengeView` when active
- On `ChallengeResolvedMsg`: appends result note to history, sends `[Challenge Result: type PASSED/FAILED — detail]` to AI for narrative continuation
- Multiple challenges in one response queued and resolved sequentially via `startNextChallenge()`
- `View()` renders challenge overlay (full-screen, same as combat/crafting)
- Added `startNextChallenge()` helper method
- Updated `/help` text with challenge system description

### Narrator Prompt (`internal/ai/prompts/narrator.go`)
Added full `## Challenges` section documenting:
- All 6 challenge types with JSON examples
- Usage rules (sparingly, difficulty scale, modifiers)
- How to handle `[Challenge Result: ...]` feedback messages
- When to use each mini-game type

## Verification

- [x] `go build ./...` passes cleanly
- [x] Dice roll shows animated d100 cycling before revealing result
- [x] Dice display shows roll, modifiers, total, difficulty, PASS/FAIL
- [x] RPS shows player/AI choice and outcome
- [x] Memory sequence shows symbols then prompts player input
- [x] Quick-time shows countdown bar, accepts any key
- [x] Riddle shows text input, correct/wrong result with answer reveal
- [x] Passive challenges show brief notification and auto-continue
- [x] Challenge results sent to AI for narrative continuation
- [x] Narrator system prompt includes full challenge documentation
- [x] Multiple challenges queued and resolved sequentially

## Files Changed

| File | Change |
|------|--------|
| `internal/tui/components/dice.go` | Created |
| `internal/tui/components/rps.go` | Created |
| `internal/tui/components/memory.go` | Created |
| `internal/tui/components/quicktime.go` | Created |
| `internal/tui/components/riddle.go` | Created |
| `internal/tui/views/challenge.go` | Created |
| `internal/tui/theme/theme.go` | Modified — added challenge colors/styles |
| `internal/tui/views/narrative.go` | Modified — challenge integration |
| `internal/ai/prompts/narrator.go` | Modified — challenge documentation |
