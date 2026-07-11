# Codebase Structure

**Analysis Date:** 2026-04-09

## Directory Layout

```text
oneday/
├── cmd/                 # CLI entry points and benchmark utilities
├── internal/            # Application packages (ai, engine, storage, rag, tui)
├── docs/                # Product notes and benchmark documentation
├── .github/workflows/   # CI and release automation
├── .planning/           # Project planning artifacts
├── oneday_data/         # Local runtime data directory (ignored)
├── build/               # Cross-built artifacts (ignored)
├── plugins/             # Reserved plugin/examples area; empty in current repo
└── config.example.yaml  # Tracked config template
```

## Directory Purposes

**`cmd/`:**
- Purpose: keep each executable isolated under `cmd/<tool>/main.go`
- Contains: `cmd/oneday/main.go`, `cmd/oneday-benchmark/main.go`, `cmd/oneday-ascii-benchmark/main.go`, `cmd/oneday-schema-benchmark/main.go`
- Key files: `cmd/oneday/main.go`

**`internal/ai/`:**
- Purpose: provider-agnostic request/response types, router, response schemas, prompt builders, and provider adapters
- Contains: `internal/ai/*.go`, `internal/ai/providers/*.go`, `internal/ai/prompts/*.go`
- Key files: `internal/ai/router.go`, `internal/ai/response_formats.go`, `internal/ai/providers/openai_compat.go`

**`internal/engine/`:**
- Purpose: gameplay orchestration and domain logic
- Contains: story creation, narration, combat, crafting, saves, sessions, commands, state mutations
- Key files: `internal/engine/storycreator.go`, `internal/engine/narrator.go`, `internal/engine/save.go`, `internal/engine/types.go`

**`internal/storage/`:**
- Purpose: SQLite access split by aggregate/table group
- Contains: `stories.go`, `chat.go`, `npcs.go`, `sessions.go`, `saves.go`, `chapters.go`, `achievements.go`, plus schema bootstrap in `db.go` and `migrations.go`
- Key files: `internal/storage/db.go`, `internal/storage/migrations.go`

**`internal/rag/`:**
- Purpose: long-term memory retrieval and summarization
- Contains: `embeddings.go`, `vectorstore.go`, `summarizer.go`, `rag.go`
- Key files: `internal/rag/rag.go`

**`internal/tui/`:**
- Purpose: Bubble Tea app shell, screens, widgets, render helpers, and theme tokens
- Contains: root app in `internal/tui/app.go`, screens in `internal/tui/views/`, reusable widgets in `internal/tui/components/`, markdown helpers in `internal/tui/rendering/`, styles in `internal/tui/theme/`
- Key files: `internal/tui/app.go`, `internal/tui/views/narrative.go`, `internal/tui/views/newstory.go`

**`docs/`:**
- Purpose: human documentation and benchmark writeups
- Contains: `docs/design.md`, `docs/roadmap.md`, `docs/benchmarks/**/*.md`
- Key files: `docs/design.md`

## Key File Locations

**Entry Points:**
- `cmd/oneday/main.go`: main game binary
- `cmd/oneday-benchmark/main.go`: model benchmark CLI
- `cmd/oneday-ascii-benchmark/main.go`: ASCII benchmark CLI
- `cmd/oneday-schema-benchmark/main.go`: schema-repair benchmark CLI

**Configuration:**
- `config.example.yaml`: tracked runtime config template
- `internal/config/config.go`: defaults, YAML loading, validation
- `Makefile`: local build/test targets

**Core Logic:**
- `internal/engine/narrator.go`: turn orchestration
- `internal/engine/storycreator.go`: story-definition workflow
- `internal/storage/migrations.go`: schema definition
- `internal/tui/views/narrative.go`: main gameplay screen

**Testing:**
- `internal/**/*_test.go`: co-located package tests
- Notable packages with coverage: `internal/ai/`, `internal/config/`, `internal/engine/`, `internal/rag/`, `internal/storage/`, `internal/tui/components/`, `internal/tui/rendering/`, `internal/tui/views/`

## Naming Conventions

**Files:**
- Executables use `cmd/<tool>/main.go`
- Package files use lowercase domain names such as `internal/engine/narrator.go` and `internal/storage/stories.go`
- Tests stay next to source as `*_test.go`

**Directories:**
- Top-level app packages stay under `internal/<domain>/`
- TUI splits by concern: `views/` for screens, `components/` for reusable widgets, `rendering/` for text shaping, `theme/` for styles

## Where to Add New Code

**New Feature:**
- Gameplay rules/orchestration: `internal/engine/`
- Persistence: `internal/storage/`
- Tests: next to the implementation file as `*_test.go`

**New Component/Module:**
- New full-screen UI flow: `internal/tui/views/`
- Reusable TUI widget: `internal/tui/components/`
- Provider adapter or request contract: `internal/ai/providers/` or `internal/ai/`

**Utilities:**
- Shared prompt text: `internal/ai/prompts/`
- Rendering-only helpers: `internal/tui/rendering/`
- New CLI tool: `cmd/<new-tool>/main.go`

## Special Directories

**`.planning/`:**
- Purpose: roadmap, requirements, phase outputs, and codebase maps
- Generated: No
- Committed: Yes

**`oneday_data/`:**
- Purpose: local SQLite DB, save snapshots, and JSONL session logs
- Generated: Yes
- Committed: No (`.gitignore`)

**`build/`:**
- Purpose: cross-compiled build artifacts from `Makefile` and CI
- Generated: Yes
- Committed: No (`.gitignore`)

**`plugins/` and `internal/plugins/`:**
- Purpose: reserved extension points
- Generated: No
- Committed: Yes

---

*Structure analysis: 2026-04-09*
