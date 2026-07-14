# Architecture

OneDay has one canonical game state and two clients.

```text
Terminal client ─┐
                 ├─ Go engine and provider router ─ SQLite
React browser ─ Rust/Axum gateway ─ JSON bridge ┘
                 └─ SSE events, generated media, HTTP API
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
