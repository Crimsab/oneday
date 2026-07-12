# OneDay

An AI-driven text RPG with a terminal client and a full browser interface.

Stories are infinite, AI-generated, and deeply personalized. Every NPC has personality, desires, and opinions about you. Every choice matters. Every story is unique.

## Features

- **AI-Powered Narrative** — stories are generated and continued by AI models (Claude, GPT, Gemini, etc.) via configurable provider chain
- **Dynamic Everything** — stats, skills, NPCs, items, locations, objectives, achievements are all generated at runtime. Nothing is hardcoded
- **Deep Character System** — start from nothing, earn everything. Traits emerge from your actions, skills from your practice, titles from your deeds
- **Living NPCs** — each NPC has personality, speech patterns, quirks, private thoughts about you, and hidden desires that drive their behavior
- **Always Free Action** — you can always type your own action, not just pick from a list. Talk to the boss instead of fighting. Craft something absurd. Try anything
- **Turn-Based Combat** — narrative-driven with stat checks, dice rolls, and creative solutions. Chat is always present, even mid-fight
- **AI Crafting** — no hardcoded recipes. Describe what you want to build, the AI evaluates if it makes sense with your materials and skills
- **Challenge System** — dice rolls, stat checks, skill checks, rock-paper-scissors, and other mini-games. The engine rolls, not the AI
- **AI Achievements** — no predefined list. The AI recognizes noteworthy moments and awards unique achievements
- **Persistent Memory** — full chat history saved, RAG-powered long-term context. Stories can be infinite
- **Multiple Genres** — fantasy, cyberpunk, horror, slice-of-life, anything. Each genre defines its own stat system and rules
- **Per-Story Authoring Control** — each story can lock its own language, prose style, and reusable prompt directives so narration stays consistent across turns, combat, crafting, summaries, and GM meta commands
- **Guided Story Setup** — `New Story` now uses a review-first wizard with quick choices for world draft, rules, factions, dangers, and stats, while still allowing free text edits at every step

## Tech Stack

- **Go** + Bubbletea/Bubbles/Lipgloss (TUI)
- **Rust** + Axum/SQLx (browser gateway, typed Go bridge, SSE streaming)
- **React** + TypeScript + Vite, built and tested with **Bun** (browser UI)
- **SQLite** + embedding BLOBs + cosine similarity in Go (RAG, no `sqlite-vec`)
- AI via **Codex OAuth** / **LiteLLM** / **OpenRouter** / **Claude Code** (configurable fallback chain)

## Quick Start

```bash
# Copy local config
cp config.example.yaml config.yaml

# Configure provider keys / endpoints.
# Prefer env vars so secrets do not live in config.yaml.
export ONEDAY_LITELLM_API_KEY="..."
export ONEDAY_OPENROUTER_API_KEY="..."
$EDITOR config.yaml

# Or run the first-time setup helper
go run ./cmd/oneday setup

# Re-open setup when config.yaml already exists
go run ./cmd/oneday setup --reconfigure

# Check local tools, provider auth, model smoke, and RAG readiness
go run ./cmd/oneday doctor

# Machine-readable diagnostics for scripts/CI/support
go run ./cmd/oneday doctor --json

# Inspect effective config without secrets
go run ./cmd/oneday config show --safe

# Check local/remote embedding readiness and latency
go run ./cmd/oneday rag benchmark

# Clear stale embeddings after changing models/dimensions
go run ./cmd/oneday rag reindex --story <story-id>

# List available story pack files
go run ./cmd/oneday story-packs list

# Create a clean share/release handoff directory
go run ./cmd/oneday export

# Run tests
go test ./...

# Test and build the browser UI
cd gateway/web
bun install --frozen-lockfile
bun run test
bun run build
cd ../..

# Build and run the complete browser gateway stack
docker compose build oneday-gateway
docker compose up -d oneday-gateway

# Run the reusable verification sweep (tests + vet + QA matrix)
make verify

# Generate Go and browser coverage baselines
make coverage

# Run the game
go run ./cmd/oneday

# Refresh the repo-root binary used by ./oneday
make build

# Check which build you are actually running
./oneday --version

# Run the stricter pre-release gate from a clean worktree
make release-check

# Build the benchmark tool
make build-bench

# Build the ASCII benchmark tool
make build-ascii-bench

# Linux amd64
GOOS=linux GOARCH=amd64 go build -o build/oneday-linux-amd64 ./cmd/oneday

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o build/oneday-windows-amd64.exe ./cmd/oneday

# Or use the Makefile helper
make all
```

The production container builds the Go engine, Rust gateway, and React UI into
one image. It listens on port `8788`, exposes `/api/health`, returns an
`X-Request-Id` response header, and emits structured request status/latency
logs. The Compose service includes a runtime healthcheck.

## Configuration

Config lives in two places:

- **inside the code**: `internal/config/config.go` contains safe defaults via `config.Default()`
- **outside the code**: `config.yaml` is the local runtime override loaded by `cmd/oneday/main.go`

Practical rules:

- `config.example.yaml` is the tracked template for the repo
- release archives also include `config.example.yaml` next to the binaries
- `config.yaml` is ignored by git and is where local secrets / endpoints go
- `${ENV_VAR}` placeholders in `config.yaml` are expanded at load time, so prefer environment variables for API keys
- `.env` is loaded automatically when present and does not overwrite already-exported variables
- the binary looks for `config.yaml` in the current working directory first, then next to the executable itself
- if `config.yaml` is missing, the app falls back to the built-in defaults
- `./oneday --version` prints the binary's version, commit, build date, and dirty-state so you can confirm you rebuilt after source changes

Current default provider strategy:

- primary: `litellm` via `http://lite.homelab.local/v1` with `gpt-5.4-mini`
- `openrouter` is available but disabled by default until you provide a real API key
- optional experimental `codex` provider shells out to the local Codex CLI after `codex login`, defaulting to `gpt-5.4-mini` with reasoning `off`
- ancillary repair/validation work can use `ai.generation.utility_model`, defaulting to `gpt-5.4-mini` when no dedicated repair model is configured
- final fallback: `claude-code` if enabled
- optional ambient ASCII art uses `ai.ascii_art.*` and can target a different model from the main narrator

RAG / embeddings note:

- embeddings are stored in SQLite as raw BLOB vectors
- retrieval uses cosine similarity in Go
- the default embedding model is `text-embedding-3-small`
- Codex is generation-only, but it can be paired with local RAG embeddings
- remote RAG can use LiteLLM/OpenRouter with `text-embedding-3-small`
- local RAG can use Ollama or a custom local HTTP embedding server without API keys
- recommended local default: `bge-m3` for multilingual/Italian-friendly retrieval
- alternatives: `nomic-embed-text` for fast/light local use, `mxbai-embed-large` for English retrieval quality, `qwen3-embedding` for heavier quality-oriented setups where available
- this project does **not** use `sqlite-vec`

Setup behavior:

- `oneday setup` preserves an existing `config.yaml` and prints the reconfigure command
- `oneday setup --reconfigure` or `oneday setup --force` opens the wizard again and can rewrite local config
- Ollama is optional: choose it for the easiest local model download/run path, or choose a custom local endpoint if you already run Python, llama.cpp, ONNX, or another embedding service

Safe sharing:

- share `config.example.yaml`, `.env.example`, and source files, not local `config.yaml`, `.env`, `oneday_data/`, databases, or binaries
- run `oneday setup` on a friend's machine so provider choice and local auth are generated there
- run `oneday doctor` after setup; 401/403 errors should point at the exact env key to fix
- `oneday export` is always safe-by-default; it excludes local config, env files, story data, databases, generated binaries, and secrets

## CI / Release

GitHub Actions is configured to:

- run `go test ./...` and `go vet ./...`
- build `oneday` and `oneday-benchmark`
- cross-compile Linux amd64 and Windows amd64 artifacts
- open or update a release PR through `release-please`
- publish the GitHub Release after that release PR is merged

Workflow files:

- `.github/workflows/build-release.yml` for CI and tag-based release builds
- `.github/workflows/release-please.yml` for automated release PRs, tags, and release asset upload
- `Makefile` for local `./oneday`, benchmark, and cross-platform builds
- `docs/qa-matrix.md` plus `scripts/qa-matrix.sh` for the high-risk cross-system sweep

Recommended local ship flow:

1. `make release-check`
2. verify `./oneday --version` matches the commit you intend to ship
3. only then publish or upload artifacts

CI now uses the same `make verify` gate before normal builds, and release automation runs `make release-check` before packaging release assets.

## License

Personal project.
