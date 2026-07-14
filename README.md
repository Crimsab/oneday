<div align="center">

# OneDay

<p><strong>Persistent AI stories that remember, branch, and evolve</strong></p>

[![CI](https://github.com/Crimsab/oneday/actions/workflows/ci.yml/badge.svg)](https://github.com/Crimsab/oneday/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Crimsab/oneday?display_name=tag&sort=semver)](https://github.com/Crimsab/oneday/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/Crimsab/oneday)](go.mod)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](docs/docker.md)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

<!-- Logo slot: add docs/assets/oneday-logo.svg above the title. -->
<!-- Hero slot: add docs/assets/oneday-hero.webp here when the final artwork is ready. -->

OneDay is a local-first narrative RPG with a browser interface and terminal
client. The AI writes the story; the engine preserves causality, resolves
mechanics, tracks the world, and keeps every branch coherent in SQLite.

[Get started](#quick-start) · [Documentation](docs/README.md) ·
[Releases](https://github.com/Crimsab/oneday/releases) ·
[Report a bug](https://github.com/Crimsab/oneday/issues/new/choose)

</div>

## Why OneDay

- **One world, not a disposable chat.** Characters, locations, factions,
  investigations, projects, achievements, inventory, and consequences persist.
- **Real player agency.** Pick a suggested choice or write any action in your
  own words; the story is not limited to a dialogue tree.
- **Branch without losing canon.** Fork decisions, explore alternate paths,
  restore snapshots, and navigate history with branch-aware world state.
- **Deterministic mechanics.** The engine owns checks, challenges, combat,
  crafting outcomes, and atomic state changes instead of trusting free-form prose.
- **Long-term memory.** RAG summaries and embeddings keep long-running stories
  grounded without treating the entire transcript as one prompt.
- **Optional generated media.** Scene art, character portraits, transparent map
  symbols, ambient ASCII, and spoken audio remain subordinate to canonical text.

## A story system, not only a narrator

| Narrative layer | Canonical engine |
| --- | --- |
| Any genre, tone, language, and prose style | Versioned SQLite state shared by browser and TUI |
| Free actions, dialogue, choices, and GM guidance | Atomic turn commits, idempotency, saves, and branches |
| Persistent NPC voice, motives, relationships, and reputation | Deterministic challenges, combat, crafting, and rewards |
| Model routing with fallbacks and repair passes | Typed Go ↔ Rust contracts and migration gates |
| Story-specific visual direction and generated assets | Branch-aware asset lineage and failure-safe background jobs |

## Interfaces

### Browser

The React interface is served by a Rust/Axum gateway over the same Go engine and
database as the terminal client. It includes the transcript, story library,
free-action composer, command palette, history and branches, canonical state
inspectors, maps, challenges, model settings, audio, and visual asset workflows.

### Terminal

The Bubble Tea client provides the complete narrative loop, guided story
creation, slash commands, choices and free actions, combat/crafting surfaces,
history, saves, diagnostics, and local CLI provider integrations.

## Quick start

### Browser with Docker

```bash
git clone https://github.com/Crimsab/oneday.git
cd oneday
cp config.example.yaml config.yaml
cp .env.example .env
```

Enable a provider in `config.yaml`, add its key to `.env`, then start the stack:

```bash
docker compose up -d --build
curl -fsS http://localhost:8788/api/health
```

Open [http://localhost:8788](http://localhost:8788). The first start creates and
migrates the persistent database automatically.

Docker does not bundle host Codex or Claude CLI credentials. The standard
container path is LiteLLM/OpenRouter; advanced users can add private CLI mounts
through a Compose override.

### Terminal from source

Requires Go 1.25 or newer:

```bash
git clone https://github.com/Crimsab/oneday.git
cd oneday
go run ./cmd/oneday setup
go run ./cmd/oneday doctor
go run ./cmd/oneday
```

Release archives with Linux and Windows binaries are available on the
[Releases page](https://github.com/Crimsab/oneday/releases).

Read the full [getting-started guide](docs/getting-started.md) for provider,
Docker-host networking, RAG, and verification details.

## AI providers and media

| Integration | Use |
| --- | --- |
| Codex CLI | Local generation through an existing `codex login` |
| Claude Code | Optional local CLI fallback |
| LiteLLM / OpenAI-compatible | Self-hosted or managed model gateway |
| OpenRouter | Direct hosted model routing |
| Ollama / custom HTTP | Local RAG embeddings |
| OpenClaw bridge / OpenAI-compatible image API | Non-blocking story visuals |

Narrative, utility, repair, embedding, ASCII, image, map-icon, and TTS paths can
use separate models. General visuals default to `openai/gpt-image-2`; transparent
map symbols can use `openai/gpt-image-1` independently.

During story creation, visual direction can be Auto, Photorealistic, Cinematic
Fantasy, Illustrated Fantasy, Anime, or a custom prompt. The selected direction
is persisted with the story for consistent later assets.

See [Configuration](docs/configuration.md) for the complete model, RAG, media,
storage, and secret-handling reference.

## How it works

```text
Terminal client ─┐
                 ├─ Go story engine + provider router ─ SQLite
React browser ─ Rust gateway ─ typed JSON bridge ┘
                 └─ SSE events + generated media
```

The Go engine owns narrative prompts, mechanics, persistence, migrations, and
canonical mutations. The Rust gateway owns HTTP, SSE, media jobs, and the typed
bridge. React renders canonical state but never invents a second game state.

Read [Architecture](docs/architecture.md) and the
[browser gateway contract](docs/browser-gateway.md) for the full turn flow and
component boundaries.

## Documentation

| Guide | Covers |
| --- | --- |
| [Getting started](docs/getting-started.md) | Native and Docker installation |
| [Configuration](docs/configuration.md) | Providers, RAG, visuals, game settings, and secrets |
| [Docker](docs/docker.md) | Networking, persistence, updates, backups, and operations |
| [Architecture](docs/architecture.md) | Components, contracts, turn flow, and canonical state |
| [Troubleshooting](docs/troubleshooting.md) | Provider, RAG, media, browser, and CI failures |
| [Development](docs/development.md) | Toolchains, layout, tests, and generated contracts |
| [Testing](docs/testing.md) | Automated gates, browser coverage, and manual release checks |
| [Benchmarks](docs/benchmarks/README.md) | Reproducible model, ASCII, and schema reliability comparisons |
| [Roadmap](docs/project-roadmap.md) | Current priorities and contribution directions |
| [Releases](docs/releases.md) | Changelog automation, versioning, tags, and artifacts |

## Development

The repository uses Go 1.25, Rust 1.97, React 19, TypeScript, Vite, and Bun.

```bash
make verify
cargo test --manifest-path gateway/Cargo.toml
cd gateway/web && bun install --frozen-lockfile && bun run test && bun run build
```

CI runs Go verification, reachable vulnerability scanning, and
cross-compilation; Rust tests and Clippy; web unit and Playwright gates; a
complete Docker build; workflow linting; and a Gitleaks source scan. Public
pull requests run only on GitHub-hosted runners.

See [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

## Releases and project status

OneDay is under active development. Back up persistent story data before an
upgrade and review the [changelog](CHANGELOG.md) for migration-sensitive changes.

Release Please turns Conventional Commits into a release PR, updates the
changelog, creates the semantic version tag, and publishes Linux/Windows
archives after the release gates pass.

## Community and security

- [Support](SUPPORT.md)
- [Security policy](SECURITY.md)
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Contributing](CONTRIBUTING.md)

OneDay is local/self-hosted software. If you expose it to an untrusted network,
you are responsible for authentication, TLS, network policy, backups, and
protecting provider credentials.

## License

OneDay is available under the [MIT License](LICENSE).
