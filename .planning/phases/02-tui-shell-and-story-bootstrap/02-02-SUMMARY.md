---
phase: 2
plan: 2.2
title: "Story creation flow with AI-guided conversation and character setup"
status: completed
---

# Summary: Plan 2.2 — Story Creation Flow

## What Was Built

### Task 1: Engine Types (`internal/engine/types.go`)
- `StoryDefinition` — top-level AI-generated story struct mirroring story.json design
- `Setting` — world details: name, era, geography, magic system, factions, rules, cultures, dangers
- `StatsSchema` with `StatDef` and `CurrencyDef` — fully dynamic per-story stat definitions
- `InitialStats()` method — generates starting character stats map from schema (vitals as current/max, attributes as flat ints, empty traits/skills/titles)
- `NarrativeResponse` and `Choice` — gameplay response types for future use (AI-02)

### Task 2: Story Creation Prompts (`internal/ai/prompts/storycreation.go`)
- `StoryCreationSystem` — 5-step conversational system prompt: Genre/Tone → World Building → Rules/Factions → Stats Schema → Confirmation. Enforces JSON-only output on confirmation, uses player language.
- `CharacterCreationSystem` — protagonist name + optional background prompt, outputs structured JSON

### Task 3: Story Creator Engine (`internal/engine/storycreator.go`)
- `StoryCreator` struct with `PhaseConversation` → `PhaseCharacter` → `PhaseDone` FSM
- `StartConversation()` — kicks off AI with initial greeting
- `SendMessage()` — routes to `handleConversation` or `handleCharacter` based on phase
- Automatic JSON extraction via `extractStoryJSON` (regex on ```json fences) and `extractCharacterJSON`
- Phase transition: when story JSON detected, auto-sends character creation prompt
- `persistStory()` — saves Story, Character, WorldState to SQLite in one flow
- Retry safety: removes failed user messages from history on AI error

### Task 4: Storage CRUD (`internal/storage/stories.go`)
- `CreateStory`, `GetStory`, `ListStories` on `*DB`
- `CreateCharacter`, `GetCharacterByStory` on `*DB`
- `CreateWorldState`, `GetWorldState` on `*DB`
- Uses correct table names matching migrations (`world_state` singular)

### Task 5: New Story TUI View (`internal/tui/views/newstory.go`)
- `NewStoryModel` — viewport (scrollable history) + textarea (input) + spinner (loading)
- `StoryCreatedMsg` — transition message sent to app when PhaseDone reached
- `StartCreation()` returns a `tea.Cmd` for async initial AI call
- Enter key sends message; spinner shown while waiting; error state with retry on any key
- Phase label updates: "Building your world..." → "Create your protagonist..."
- Status bar shows model name and latency from last AI call

### Task 6: App Wiring (`internal/tui/app.go`)
- Added `newStory *views.NewStoryModel` field
- `ActionNewStory` creates `StoryCreator`, initialises `NewStoryModel`, transitions to `ViewNewStory`, fires `StartCreation()`
- `WindowSizeMsg` propagates size to `newStory` when active
- `ViewNewStory` in Update: delegates to newStory, handles `esc` to return to menu
- `ViewNewStory` in View: renders newStory view
- `StoryCreatedMsg` handler: returns to menu (placeholder until Plan 2.4 wires narrative view)

## Key Design Decisions

- **No hardcoded templates**: the AI defines all stats, world details, and rules per story
- **JSON extraction via regex**: `(?s)```json\n(.*?)\n``` ` — robust against surrounding prose
- **Phase-based FSM**: single `StoryCreator` manages both story-building and character-naming conversations with separate message histories
- **Retry safety**: failed AI calls remove the user message so the player can try again
- **world_state table is singular** (matching migrations schema) — not `world_states`
- **SkillsJSON defaults to `[]`** (array) not `{}` (object) to match migration defaults

## Files Created/Modified

- `internal/engine/types.go` — new
- `internal/engine/storycreator.go` — new
- `internal/ai/prompts/storycreation.go` — new
- `internal/storage/stories.go` — new
- `internal/tui/views/newstory.go` — new
- `internal/tui/app.go` — modified
- `go.sum` — updated (textarea/bubbles dependency)

## Verification

- `go build ./internal/engine/` — pass
- `go build ./internal/ai/prompts/` — pass
- `go build ./internal/storage/` — pass
- `go build ./internal/tui/views/` — pass
- `go build ./internal/tui/` — pass
- `go build ./cmd/oneday/` — pass
