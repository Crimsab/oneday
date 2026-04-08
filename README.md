# OneDay

An AI-driven text RPG played entirely in the terminal.

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

## Tech Stack

- **Go** + Bubbletea/Bubbles/Lipgloss (TUI)
- **SQLite** + embedding BLOBs + cosine similarity in Go (RAG, no `sqlite-vec`)
- AI via **LiteLLM** / **OpenRouter** / **Claude Code** (configurable fallback chain)

## Quick Start

```bash
# Copy local config
cp config.example.yaml config.yaml

# Edit provider keys / endpoints
$EDITOR config.yaml

# Run tests
go test ./...

# Run the game
go run ./cmd/oneday

# Build the main binary
go build -o oneday ./cmd/oneday

# Build the benchmark tool
go build -o oneday-benchmark ./cmd/oneday-benchmark

# Linux amd64
GOOS=linux GOARCH=amd64 go build -o oneday-linux-amd64 ./cmd/oneday

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o oneday-windows-amd64.exe ./cmd/oneday
```

## Configuration

Config lives in two places:

- **inside the code**: `internal/config/config.go` contains safe defaults via `config.Default()`
- **outside the code**: `config.yaml` is the local runtime override loaded by `cmd/oneday/main.go`

Practical rules:

- `config.example.yaml` is the tracked template for the repo
- `config.yaml` is ignored by git and is where local secrets / endpoints go
- the binary looks for `config.yaml` in the current working directory first, then next to the executable itself
- if `config.yaml` is missing, the app falls back to the built-in defaults

Current default provider strategy:

- primary: `litellm` via `http://lite.homelab.local/v1` with `grok-4.1-fast`
- `openrouter` is available but disabled by default until you provide a real API key
- final fallback: `claude-code` if enabled

RAG / embeddings note:

- embeddings are stored in SQLite as raw BLOB vectors
- retrieval uses cosine similarity in Go
- this project does **not** use `sqlite-vec`

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

## License

Personal project.
