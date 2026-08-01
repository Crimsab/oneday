<div align="center">

# OneDay

<p><strong>Any story. Yours to live.</strong></p>

[![CI](https://github.com/Crimsab/oneday/actions/workflows/ci.yml/badge.svg)](https://github.com/Crimsab/oneday/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Crimsab/oneday?display_name=tag&sort=semver)](https://github.com/Crimsab/oneday/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/Crimsab/oneday)](go.mod)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](docs/docker.md)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

<img src="docs/assets/oneday-hero.webp" alt="OneDay: a luminous doorway and branching path beneath a starry sky, with persistent worlds, relationships, images, voice, crafting, minigames, and multiple AI providers" width="100%" />

OneDay turns any premise into a persistent, interactive world. Write any action,
follow or reject suggested choices, and explore stories that remember what
happened and evolve around the consequences. Combat is optional; the genre,
tone, language, rules, and play style belong to each story.

**Any genre · Free-form actions · Branching timelines · Minigames · Crafting ·
Investigations · Optional combat · Images and voices**

[Get started](#start-oneday) · [Documentation](docs/README.md) ·
[Releases](https://github.com/Crimsab/oneday/releases) ·
[Report a bug](https://github.com/Crimsab/oneday/issues/new/choose)

</div>

## Start OneDay

Choose one installation method.

| Method | Use it when | You need |
| --- | --- | --- |
| [Docker](#docker-recommended) | You want the browser or PWA on Windows, macOS, or Linux | Git and Docker Compose v2 |
| [Desktop](#desktop) | A release contains a package for your operating system | The matching package from GitHub Releases |
| [Terminal](#terminal-from-source) | You want the terminal client or you develop OneDay | Go 1.25.12 or newer |

At least one narrative AI provider is required. Images and speech are optional.

### Docker (recommended)

This method builds OneDay inside Docker. You do not need Go, Rust, Bun, or a
shell script on the host.

1. Clone the repository.

   ```bash
   git clone https://github.com/Crimsab/oneday.git
   cd oneday
   ```

2. Build the local image.

   ```bash
   docker compose -f compose.yaml -f compose.build.yaml build oneday-gateway
   ```

3. Create the local configuration and browser credential.

   ```bash
   docker compose -f compose.yaml -f compose.build.yaml run --rm oneday-tools docker init
   ```

4. Start OneDay.

   ```bash
   docker compose -f compose.yaml -f compose.build.yaml up -d
   ```

5. Show the browser credential when the login screen asks for it.

   ```bash
   docker compose -f compose.yaml -f compose.build.yaml run --rm oneday-tools docker token
   ```

6. Open [http://localhost:8788](http://localhost:8788).

7. Enter the credential. Then open **Setup** and configure a narrative
   provider.

The commands are the same in PowerShell, macOS, and Linux. The first start
creates the SQLite database in a Docker volume. A normal `docker compose down`
keeps this data.

The standard gateway image does not contain the Codex or Claude CLI, so it
cannot use their host logins for narrative generation. Use a LiteLLM-compatible
endpoint or OpenRouter for narrative generation in the standard container
setup. Codex OAuth image generation is available through the optional
`imagegen-bridge` profile: its helper copies only the host `auth.json` into a
dedicated Docker volume. See the [Docker guide](docs/docker.md) for that
profile, released images, provider networking, updates, and backups.

### Desktop

Open the [latest release](https://github.com/Crimsab/oneday/releases/latest).
Install a desktop package only when the release contains one for your operating
system and CPU.

On the first launch, choose one profile:

- **Connect to a server** opens an existing HTTPS OneDay server. It does not
  create a local story database.
- **Run on this device** creates an isolated local profile. It does not
  synchronize stories with a server.

In local mode, the desktop shows every narrative path instead of assuming
Codex. It can reuse an existing Codex CLI, install a pinned and
SHA-256-verified official Codex component on demand, detect Claude Code, or
install Claude Code through WinGet on Windows or Homebrew on macOS. OpenRouter
and LiteLLM-compatible endpoints are configured in the same protected model
screen. Nothing is downloaded until you choose an install action.

Public releases include Windows, Linux, Apple Silicon macOS, and Intel macOS
packages plus a signed update feed. The desktop checks that feed automatically,
but it downloads and installs an update only after you select **Install and
restart**. Source and pull-request builds keep the updater disabled. Read the
[desktop guide](docs/desktop.md) before you choose a profile.

### Terminal from source

Run these commands:

```bash
git clone https://github.com/Crimsab/oneday.git
cd oneday
go run ./cmd/oneday setup
go run ./cmd/oneday doctor
go run ./cmd/oneday
```

The setup command creates local configuration. The doctor command checks it
before OneDay starts. Continue with [Your first story](docs/first-story.md).

## More than generated prose

- **Any story you can describe.** Mystery, romance, political drama, comedy,
  horror, science fiction, slice of life, fantasy, or something entirely new.
- **One world, not a disposable chat.** Characters, locations, factions,
  investigations, projects, achievements, inventory, and consequences persist.
- **Real player agency.** Pick a suggested choice or write any action in your
  own words; the story is not limited to a dialogue tree.
- **Systems that fit the scene.** Challenges, contextual minigames, crafting,
  social confrontations, investigations, projects, and optional combat create
  interaction without forcing every story into an RPG template.
- **Branch without losing canon.** Fork decisions, explore alternate paths,
  restore snapshots, and navigate history with branch-aware world state.
- **Consequences the engine can trust.** Checks, rewards, inventory changes,
  relationships, crafting outcomes, and turn commits are resolved outside the
  model's free-form prose.
- **Long-term memory.** RAG summaries and embeddings keep long-running stories
  grounded without treating the entire transcript as one prompt.
- **Optional generated media.** Scene art, character portraits, transparent map
  symbols, native source-image edits, brush-mask inpainting, ambient ASCII, and
  spoken audio remain subordinate to canonical text.

## What lives inside a story

| Story experience | Persistent systems |
| --- | --- |
| Any genre, tone, language, and writing style | Characters, locations, factions, relationships, reputation, and world events |
| Free actions, dialogue, suggested choices, and story guidance | Atomic turns, saves, rewind, alternative branches, and searchable history |
| Scene-aware challenges and minigames | Stats, skills, items, relationships, dice, outcomes, rewards, and fail-forward consequences |
| Crafting, investigations, projects, social duels, and optional combat | Inventory, recipes, clues, suspects, leverage, fronts, achievements, and progression |
| Scene art, portraits, maps, ASCII, and spoken audio | Branch-aware media lineage, provider routing, retries, and non-blocking failure |

Automatic timing-free minigames include deduction, negotiation, pattern solving,
bidding, courtroom exchanges, and comedy. The extension engine also supports
riddles, memory, rock-paper-scissors, and quick-time definitions. Read the
[feature tour](docs/features.md) and [story systems guide](docs/story-systems.md)
for the complete behavior.

## Interfaces

### Browser

The React interface is served by a Rust/Axum gateway over the same Go engine and
database as the terminal client. It includes the transcript, story library,
free-action composer, command palette, history and branches, canonical state
inspectors, maps, challenges, model settings, audio, and a full-resolution
visual editor with brush/eraser masks, zoom, pan, and local stroke history.

### Desktop

The Windows, Linux, and macOS desktop client can either connect to one HTTPS OneDay server
or run a bundled local gateway and engine in an isolated standalone profile.
Remote mode creates no local story database; standalone mode is local-only.
They do not synchronize, merge, or share stories automatically. See [Desktop
client](docs/desktop.md) before choosing a profile.

### Terminal

The Bubble Tea client provides the complete narrative loop, guided story
creation, slash commands, choices and free actions, combat/crafting surfaces,
history, saves, diagnostics, and local CLI provider integrations.

## AI providers and media

| Integration | Use |
| --- | --- |
| Codex CLI | Local generation through an existing `codex login` |
| Claude Code | Optional local CLI fallback |
| LiteLLM / OpenAI-compatible | Self-hosted or managed model gateway |
| OpenRouter | Direct hosted model routing |
| Ollama / custom HTTP | Local RAG embeddings |
| Codex OAuth, OpenAI, Gemini, fal.ai, Replicate, Stability, Azure, or compatible image APIs | Explicit non-blocking story visual providers |

Narrative, utility, repair, embedding, ASCII, image, map-icon, and TTS paths can
use separate models. Codex OAuth is the default visual provider and uses the
Codex-only imagegen-bridge with `codex-responses` as its recommended route and
`codex-app-server` as its compatibility fallback. Vendor API-key providers use
separate direct OneDay adapters. General art and transparent map symbols can use
independent providers and models.

During story creation, visual direction can be Auto, Photorealistic, Cinematic
Fantasy, Illustrated Fantasy, Anime, or a custom prompt. The selected direction
is persisted with the story for consistent later assets.

See [Configuration](docs/configuration.md) for the complete model, RAG, media,
storage, and secret-handling reference.

## Frequently asked questions

### Can I use my Codex subscription?

Yes. In the terminal client, run `codex login` and choose **Codex OAuth** during
setup. In desktop standalone mode, OneDay detects an existing Codex CLI or can
install a verified private component on demand, then opens the Codex login.
A browser served by a native gateway also works when that gateway process can
reach the authenticated CLI; the browser never reads the credential itself.

For generated images in Docker, the optional `imagegen-bridge` profile can copy
only the host Codex `auth.json` into an isolated volume. This import is explicit,
not automatic.

The standard Docker gateway does not contain the Codex CLI, so host Codex OAuth
does not power narrative generation inside that container. This Docker boundary
does not apply to the native terminal, native gateway, or desktop standalone
runtime. Use LiteLLM/OpenRouter for the standard container narrative path.

### Can I use my Claude subscription?

Yes, for narrative generation when Claude Code is installed and authenticated
on the machine running OneDay. Desktop standalone detects it and offers a
WinGet install on Windows or a Homebrew install on macOS when available; Linux
links to Anthropic's current installation guide. Sign-in stays inside Claude
Code. Then enable **Claude Code** in the provider connections and priority.
Claude Code does not provide OneDay images or embeddings, and the standard
Docker image does not contain the Claude CLI.

### Do Codex and Claude work with long-term memory?

Yes. Normal structured context and recent messages work with every narrative
provider. Optional RAG also works with Codex or Claude as the narrator, but it
needs a separate embedding provider such as local Ollama, a custom endpoint,
LiteLLM, or OpenRouter. The retrieved memory is added to the narrative request;
Codex and Claude Code do not create the vectors themselves.

Read the [complete FAQ](docs/faq.md) for credential handling, automatic setup,
provider selection, context, embeddings, reindexing, and Docker boundaries.

## How it works

```text
Terminal client ─┐
                 ├─ Go story engine + provider router ─ SQLite
React browser ─ Rust gateway ─ typed JSON bridge ┘
                 └─ SSE events + generated media
Desktop ─ remote server or isolated loopback gateway ┘
```

The Go engine owns narrative prompts, mechanics, persistence, migrations, and
canonical mutations. The Rust gateway owns HTTP, SSE, media jobs, and the typed
bridge. React renders canonical story state but never invents a second source of
truth.

Read [Architecture](docs/architecture.md) and the
[browser gateway contract](docs/browser-gateway.md) for the full turn flow and
component boundaries.

## Documentation

| Guide | Covers |
| --- | --- |
| [FAQ](docs/faq.md) | Codex and Claude subscriptions, context, RAG, embeddings, and Docker boundaries |
| [Getting started](docs/getting-started.md) | Native and Docker installation |
| [Your first story](docs/first-story.md) | Configure a provider and create the first world |
| [Feature tour](docs/features.md) | Player-facing capabilities and interfaces |
| [Story systems](docs/story-systems.md) | Branches, challenges, minigames, crafting, investigations, projects, and conflict |
| [Generated media](docs/media.md) | Images, maps, ASCII, TTS, providers, and failure behavior |
| [Image providers](docs/image-providers.md) | Codex OAuth and direct vendor adapter coverage |
| [Observability](docs/observability.md) | Optional OpenTelemetry, Langfuse, local diagnostics, privacy, and verification |
| [Configuration](docs/configuration.md) | Providers, RAG, visuals, game settings, and secrets |
| [Docker](docs/docker.md) | Networking, persistence, updates, backups, and operations |
| [Desktop](docs/desktop.md) | Remote and standalone profiles, data isolation, and sidecar limits |
| [Extensions](docs/extensions.md) | Story packs, challenge pools, and minigame definitions |
| [Architecture](docs/architecture.md) | Components, contracts, turn flow, and canonical state |
| [Security threat model](docs/security-threat-model.md) | Trust boundaries, mitigations, and deployment responsibilities |
| [Troubleshooting](docs/troubleshooting.md) | Provider, RAG, media, browser, and CI failures |
| [Development](docs/development.md) | Toolchains, layout, tests, and generated contracts |
| [Testing](docs/testing.md) | Automated gates, browser coverage, and manual release checks |
| [Localization](docs/localization.md) | Interface catalogs, fallback rules, language boundaries, and contributor checks |
| [Documentation site](docs/wiki.md) | Material for MkDocs and GitHub Pages workflow |
| [Benchmarks](docs/benchmarks/README.md) | Reproducible model, ASCII, and schema reliability comparisons |
| [Roadmap](docs/project-roadmap.md) | Current priorities and contribution directions |
| [Releases](docs/releases.md) | Changelog automation, versioning, tags, and artifacts |

## Development

The repository uses Go 1.25.12, Rust 1.97, React 19, TypeScript, Vite, and Bun.

```bash
make verify
cargo test --manifest-path gateway/Cargo.toml
cd gateway/web && bun install --frozen-lockfile && bun run test && bun run build
```

One consolidated CI job runs Go verification, reachable vulnerability scanning,
and cross-compilation; Rust tests and Clippy; web unit and Playwright gates; a
complete Docker build with version smoke checks; workflow linting; and a
Gitleaks source scan. Public pull requests run only on GitHub-hosted runners.

See [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

## Releases and project status

OneDay is under active development. Back up persistent story data before an
upgrade and review the [changelog](CHANGELOG.md) for migration-sensitive changes.

Release automation manages release metadata and publication. The current
public release page is the source of truth for packages, checksums, signatures,
and the updater feed. Verify the publisher and the artifact for your platform
before installing it.

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
