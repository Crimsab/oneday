# Code Review - 2026-04-08

## Context

Review requested after implementation up to Phase 6 state, but intentionally **excluding missing Phase 6.1+ systems as defects by themselves** unless they already leak into completed phases or break existing behavior.

Build/test status during review:

- `go test ./...` -> pass
- `go build ./cmd/oneday` -> pass
- `go vet ./...` -> pass
- `go test -cover ./...` -> partial failure in this environment (`go: no such tool "covdata"`)

## Executive Summary

The codebase is structurally aligned with the roadmap and design: package boundaries are coherent, the TUI shell exists, story creation works, the narrative loop is present, NPC/chapter/RAG scaffolding is in place, and the project is not randomly drifting into a different product.

However, the current project state is more fragile than `.planning/STATE.md` suggests. In particular:

- Phase 1-2 look solid.
- Phase 3 is **not really production-safe yet** because resume/load behavior can restart the story and corrupt global turn progression.
- Phase 4 is only partially solid because inventory persistence/UI contracts are inconsistent.
- Phase 5 exists, but session/turn inconsistencies reduce trust in chapters, RAG summarization, and NPC recency.

## Priority Findings

### 1. Critical - Resume/load restarts the story and can corrupt global turn progression

Symptoms:

- Opening an existing story or loading a save calls `StartNarration()` again instead of resuming the current scene.
- `GameSession` restores its local turn counter from the current session JSONL line count.
- `Narrator.sendTurn()` then writes `world.CurrentTurn = currentTurn + 1`.

Impact:

- Existing stories can receive a fresh "Begin the story" prompt.
- `world.CurrentTurn` can move backwards after reopening/loading.
- Chapter tracking, RAG summarization windows, and NPC recency all become unreliable.

Relevant files:

- `internal/tui/app.go`
- `internal/engine/session.go`
- `internal/engine/narrator.go`

### 2. High - Save/load does not fully restore the character state in DB

`SaveGame()` serializes the full `Character`, but `LoadGame()` only persists `stats_json` back to SQLite through `UpdateCharacterStats()`.

Impact:

- `traits_json`, `skills_json`, `inventory_json`, and `known_recipes_json` can diverge from the loaded snapshot.
- A loaded game may look correct in memory during the current run but be inconsistent after reopening.

Relevant files:

- `internal/engine/save.go`
- `internal/storage/stories.go`

### 3. High - Inventory contract is inconsistent between prompt, state application, and UI

Current mismatch:

- The narrator prompt instructs the AI to emit `inventory_add` as objects.
- `ApplyStateChanges()` only accepts string slices for inventory changes.
- `/inventory` prefers `Character.InventoryJSON`, while new items are effectively being tracked inside `stats_json`.

Impact:

- AI-generated item rewards may be silently dropped.
- Even when items exist in stats, they may not appear in the inventory overlay.

Relevant files:

- `internal/ai/prompts/narrator.go`
- `internal/engine/state.go`
- `internal/tui/views/narrative.go`
- `internal/engine/storycreator.go`

### 4. Medium - Config file shape does not match the loader for generation settings

`config.example.yaml` and `config.yaml` place `temperature`, `max_tokens`, and `timeout_seconds` directly under `ai:`, but the code expects them under `ai.generation`.

Separately, key AI calls still hardcode temperature/max tokens instead of consistently reading from config.

Impact:

- The documented config is misleading.
- Runtime behavior is less configurable than advertised.

Relevant files:

- `config.example.yaml`
- `config.yaml`
- `internal/config/config.go`
- `internal/engine/storycreator.go`
- `internal/engine/narrator.go`
- `internal/engine/narrator_cmd.go`
- `internal/rag/summarizer.go`

### 5. Medium - Story metadata from creation is not persisted cleanly

The story definition includes `description`, `genre`, and `tone`, but persistence only stores `name`, `setting_json`, and `stats_schema_json`.

Impact:

- Story metadata is lost after creation.
- The load screen tries to extract `genre` from `setting_json`, where it does not exist.
- Future prompt/context quality is lower than the design implies.

Relevant files:

- `internal/engine/types.go`
- `internal/storage/models.go`
- `internal/engine/storycreator.go`
- `internal/tui/views/loadstory.go`
- `internal/engine/context.go`

### 6. Medium - Prompt/runtime contract is ahead of actual engine support

The narrator is asked to output `challenges` and `achievements`, and `NarrativeResponse` already models them, but the runtime does not actually consume those fields in the main loop.

Impact:

- The AI is encouraged to produce structured output the engine ignores.
- Requirement coverage is overstated for challenge/achievement-related behavior.

Relevant files:

- `internal/ai/prompts/narrator.go`
- `internal/engine/types.go`
- `internal/engine/narrator.go`
- `internal/storage/achievements.go`

## Roadmap / Design Alignment

### What is aligned

- Project structure follows the intended modular architecture.
- AI router, storage, TUI shell, narrative loop, NPC support, chapters, narrator command, and RAG plumbing all map well to the roadmap.
- The project still reflects the design goal of "AI proposes, engine decides", even if some parts are incomplete.

### Where the implementation is overstating completeness

- `CORE-05`, `CORE-06`, `STOR-02`, `STOR-03`, and part of Phase 3 should be treated as **implemented but not yet reliable**.
- `TUI-07` exists, but inventory behavior is not trustworthy enough to call finished.
- `AI-02` is only partially operational because some advertised response fields are not yet consumed.
- `STOR-04` is scaffolded in session management, but actual combat/crafting/dialogue flows are not yet driving it in normal gameplay.

### Does the project seem to be "doing its own thing"?

Mostly no.

The main issue is not roadmap drift, but **prematurely claiming completion** for systems whose contracts are still inconsistent. The codebase is directionally correct; the execution quality around persistence/resume/state coherence is the weak point.

## Recommended Fix Order

1. Fix session/turn ownership and resume semantics.
2. Make load restore full character state, not just `stats_json`.
3. Unify inventory representation across prompt, engine, persistence, and UI.
4. Align config schema with the documented YAML and stop hardcoding generation parameters in core paths.
5. Persist story metadata (`description`, `genre`, `tone`) properly.
6. Either wire `challenges` / `achievements` into runtime or remove them from the active narrator contract until their phases are actually implemented.

## Testing Gaps

There are no meaningful tests yet for:

- resume/load lifecycle
- session turn restoration
- story creator persistence
- narrator main loop integration
- inventory end-to-end behavior
- TUI views beyond low-level components

That gap matches the kinds of bugs found in this review.
