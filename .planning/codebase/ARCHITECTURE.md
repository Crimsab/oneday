# Architecture

**Analysis Date:** 2026-04-09

## Pattern Overview

**Overall:** Layered single-process terminal application

**Key Characteristics:**
- `cmd/oneday/main.go` assembles config, storage, AI router, and Bubble Tea app in one process
- `internal/tui/` owns screen state and user interaction; `internal/engine/` owns gameplay rules and orchestration
- `internal/storage/` and `internal/rag/` persist canonical story state; AI providers are adapter-style clients behind `internal/ai/`

## Layers

**CLI Entry Points:**
- Purpose: start binaries and standalone tools
- Location: `cmd/oneday/`, `cmd/oneday-benchmark/`, `cmd/oneday-ascii-benchmark/`, `cmd/oneday-schema-benchmark/`
- Contains: `main.go` files only
- Depends on: `internal/config`, `internal/storage`, `internal/aifactory`, `internal/tui`, `internal/engine`
- Used by: end users, CI, and benchmark runs

**Provider / Prompt Layer:**
- Purpose: abstract AI backends and define structured response contracts
- Location: `internal/ai/`, `internal/ai/providers/`, `internal/ai/prompts/`, `internal/aifactory/`
- Contains: provider interfaces, router, OpenAI-compatible HTTP client, Claude CLI adapter, JSON schema builders, prompt text
- Depends on: `net/http`, `os/exec`, config structs
- Used by: `internal/engine/`, benchmark tools, `internal/tui/app.go` RAG setup

**Gameplay Engine Layer:**
- Purpose: coordinate story creation, narration, combat, crafting, saves, chapters, and state changes
- Location: `internal/engine/`
- Contains: `StoryCreator`, `Narrator`, combat/challenge/crafting engines, save/session helpers, data contracts in `types.go`
- Depends on: `internal/ai`, `internal/storage`, `internal/rag`, `internal/config`
- Used by: `internal/tui/views/` and CLI benchmark tools

**Persistence Layer:**
- Purpose: store canonical story, chat, save, NPC, achievement, and chapter data
- Location: `internal/storage/`
- Contains: SQLite connection/migrations in `db.go` and CRUD methods split by aggregate file (`stories.go`, `chat.go`, `npcs.go`, `sessions.go`, `saves.go`, `chapters.go`, `achievements.go`)
- Depends on: `database/sql`, `modernc.org/sqlite`
- Used by: `internal/engine/`, `internal/tui/app.go`, benchmark tools

**RAG Layer:**
- Purpose: embed summaries, store vectors, retrieve long-term memory, and summarize turn ranges
- Location: `internal/rag/`
- Contains: `Embedder`, `VectorStore`, `Summarizer`, `RAG`
- Depends on: `internal/ai`, `internal/storage`, SQLite connection from `storage.DB.Conn()`
- Used by: `internal/tui/app.go` and `internal/engine/narrator.go`

**TUI Layer:**
- Purpose: render the app, route Bubble Tea messages, and manage sub-views
- Location: `internal/tui/`, `internal/tui/views/`, `internal/tui/components/`, `internal/tui/rendering/`, `internal/tui/theme/`
- Contains: root `App`, screens, reusable widgets, markdown/narrative rendering helpers, theme tokens
- Depends on: Bubble Tea/Bubbles/Lipgloss plus `internal/engine`
- Used by: `cmd/oneday/main.go`

## Data Flow

**New Story Flow:**

1. `cmd/oneday/main.go` loads config and opens SQLite through `internal/storage.Open`
2. `internal/tui/app.go` creates `engine.NewStoryCreator` when the menu selects `New Story`
3. `internal/engine/storycreator.go` requests a schema-shaped story definition, validates/repairs it, then persists story, character, and world rows through `internal/storage/stories.go`
4. `internal/tui/app.go` enters narrative mode, creates `engine.Narrator`, and optionally wires `rag.NewRAG`

**Narrative Turn Flow:**

1. `internal/tui/views/narrative.go` sends free text, choices, or commands to `engine.Narrator`
2. `internal/engine/narrator.go` loads recent chat, NPCs, achievements, chapter summary, and RAG chunks, then builds prompts through `internal/engine/context.go` and `internal/ai/prompts/narrator.go`
3. `internal/ai.Router` calls the first working provider from the configured priority chain
4. `internal/engine/narrator.go` parses structured JSON, applies state changes, updates SQLite and JSONL session logs, then triggers autosave, chapter management, RAG summarization, and optional ASCII generation

**Save / Resume Flow:**

1. `internal/tui/views/narrative.go` invokes `engine.SaveGame`, `engine.LoadGame`, or `engine.Autosave`
2. `internal/engine/save.go` writes snapshot JSON files and mirrored DB rows
3. `internal/tui/app.go` recreates `engine.Narrator` and uses `ResumeNarration` so existing stories do not replay the first-turn prompt

**State Management:**
- Canonical long-lived state lives in SQLite tables created by `internal/storage/migrations.go`
- Per-session transcripts and save snapshots live on disk under `oneday_data/` through `internal/engine/session.go` and `internal/engine/save.go`
- Bubble Tea models in `internal/tui/views/` keep ephemeral UI state such as focus, streaming buffers, overlays, and animations

## Key Abstractions

**`ai.Router`:**
- Purpose: fallback-ordered completion and streaming across providers
- Examples: `internal/ai/router.go`, `internal/aifactory/factory.go`
- Pattern: adapter + chain-of-responsibility

**`engine.Narrator`:**
- Purpose: orchestrate gameplay turns, structured parsing, state mutation, persistence, and background tasks
- Examples: `internal/engine/narrator.go`, `internal/engine/context.go`
- Pattern: orchestration service over storage, AI, RAG, and session collaborators

**`storage.DB`:**
- Purpose: expose table-scoped CRUD on top of one SQLite connection
- Examples: `internal/storage/db.go`, `internal/storage/stories.go`, `internal/storage/chat.go`
- Pattern: lightweight repository wrapper

**`views.NarrativeModel`:**
- Purpose: main interactive screen that coordinates streaming text, commands, overlays, saves, combat, crafting, and challenges
- Examples: `internal/tui/views/narrative.go`
- Pattern: Bubble Tea state machine with delegated subviews

## Entry Points

**Main Game:**
- Location: `cmd/oneday/main.go`
- Triggers: local CLI execution
- Responsibilities: resolve config path, open DB, build AI router, start Bubble Tea program

**Model Benchmark:**
- Location: `cmd/oneday-benchmark/main.go`
- Triggers: manual benchmark runs and CI-built utility binary
- Responsibilities: call an OpenAI-compatible endpoint, score model outputs, write reports under `docs/benchmarks/runs/`

**ASCII Benchmark:**
- Location: `cmd/oneday-ascii-benchmark/main.go`
- Triggers: manual benchmark runs and CI-built utility binary
- Responsibilities: evaluate ambient ASCII-art generation quality and write reports

**Schema Reliability Benchmark:**
- Location: `cmd/oneday-schema-benchmark/main.go`
- Triggers: manual reliability benchmarking
- Responsibilities: load normal app config, seed invalid payloads, and measure story-definition repair success

## Error Handling

**Strategy:** Fail fast during startup; degrade gracefully during gameplay when optional subsystems fail

**Patterns:**
- Startup exits on config, DB, or router creation failures in `cmd/oneday/main.go`
- Provider chain fallback happens in `internal/ai/router.go`
- Structured response repair loops live in `internal/engine/storycreator.go` and `internal/engine/narrator.go`
- RAG, chapter, metadata, and some DB update failures are intentionally non-fatal in `internal/engine/narrator.go` and `internal/rag/rag.go`

## Cross-Cutting Concerns

**Logging:** stderr for startup paths; `log.Printf` for RAG failures in `internal/rag/rag.go`
**Validation:** config validation in `internal/config/config.go`, provider-side JSON schema requests in `internal/ai/response_formats.go`, engine-side parse/repair validation in `internal/engine/storycreator.go`
**Authentication:** no player auth; outbound provider auth comes from `config.yaml`, benchmark flags/env, or an existing Claude CLI login

---

*Architecture analysis: 2026-04-09*
