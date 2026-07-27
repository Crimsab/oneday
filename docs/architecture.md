# Architecture

OneDay has one canonical game state and several ways to reach it.

```text
Terminal client ─┐
                 ├─ Go engine and provider router ─ SQLite
React browser ─ Rust/Axum gateway ─ JSON bridge ┘
                 └─ SSE events, generated media, HTTP API
Desktop ── remote gateway or isolated local sidecars
```

## Components

### Go engine

`cmd/oneday` owns setup, diagnostics, terminal interaction, narrator prompts,
provider fallback, deterministic mechanics, save/load, migrations, RAG, and
all canonical turn mutations. Browser bridge commands use JSON on stdin/stdout
so they execute the same domain logic as the terminal client.

### SQLite

SQLite is the source of truth for stories, sessions, messages, characters,
world state, branches, saves, achievements, investigations, projects, generated
asset metadata, and idempotency. The gateway runs the Go schema preflight before
opening its SQLx pool, so both languages observe the same migrations.

### Rust gateway

`gateway/` serves the HTTP API and static web build, translates browser requests
into typed Go bridge commands, streams turn/asset events through SSE, protects
story mutations with locks and idempotency keys, and stores generated media under
the configured data directory.

### React client

`gateway/web/` renders the story transcript, choices/free actions, history and
branch navigation, canonical inspectors, map, challenges, settings, audio, and
visual asset workflows. It does not own an independent game state.

### Desktop profiles

The desktop client is a shell around the browser experience, not a second story
engine. On first use, choose one profile:

- **Remote** opens one configured HTTPS gateway origin (plain HTTP is accepted
  only for loopback development). It stores the selected server address in
  desktop settings, but no SQLite database, assets, queued turns, or offline
  edits on the device.
- **Standalone** starts the version-matched engine, gateway, and web UI bundled
  with the desktop application. It assigns an opaque local profile ID, creates
  a separate configuration and data directory for that profile, and listens on
  a fresh loopback port for that launch.

These are deliberately separate stores. Switching profiles is not a transfer,
backup, restore, or synchronization operation. Move a story explicitly through
the server's supported import/export workflow, and make backups before doing so.
See [Desktop](desktop.md) for storage locations and sidecar requirements.

## Gateway access boundary

The gateway binds to loopback by default. Its browser path exchanges a bootstrap
credential for a short-lived, signed browser session. An interactive credential
generated at startup is one-shot. An explicitly configured bootstrap credential
remains available for later reauthentication, and its derived session signature
key makes existing sessions survive gateway restarts. A separately configured
direct bearer credential is for non-browser callers such as a local desktop
launch; it must not be reused as a bootstrap credential.

The gateway validates `Host` against its listener and any explicitly configured
allowed hosts. Authenticated mutations also require a same-origin `Origin` when
present, reject cross-site fetch metadata, and do not accept cookie-authenticated
mutations without an origin. These defenses are part of the application
boundary, but do not turn a public listener into a complete internet-facing
service: place remote access behind TLS and an authentication-aware reverse
proxy, and configure the proxy host explicitly.

## Turn flow

1. A client submits a choice, command, or free action with an idempotency key.
2. The Go service claims the turn and reads the canonical snapshot.
3. The provider router builds narrative context and calls the first healthy configured provider.
4. Structured output is validated; deterministic mechanics and state changes are applied atomically.
5. Messages, world changes, branch commits, and follow-up jobs are committed to SQLite.
6. The gateway emits SSE updates; both clients reload the same canonical revision.

## Generated media

Image and TTS work is asynchronous and non-blocking. Jobs are branch-aware,
bounded, retry-safe, and retain the previous ready asset if regeneration fails.
Text and structured state remain authoritative when media is unavailable.

## Contracts

- `contracts/gateway-v1.schema.json` describes the generated Go/Rust bridge protocol.
- `contracts/challenge-v1.json` and `contracts/minigame-v1.json` define portable challenge payloads.
- `internal/game/contracts` owns browser-facing command descriptors.
- `docs/browser-gateway.md` records parity and non-goal decisions.

## Availability and optional services

Canonical text turns use the Go engine and SQLite transaction boundary. RAG,
images, speech, observability export, and provider calls are optional
integrations. Generated media runs asynchronously and text-only image mode is
supported; a missing media sidecar or provider must not block a committed text
turn. Narrative generation still needs a configured narrative provider, so
"offline" means local data and local services—not an automatic promise that a
remote model is available.
