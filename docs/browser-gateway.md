# OneDay Browser Gateway

The browser surface is a Rust gateway over the same canonical SQLite database
and Go game engine used by the terminal UI. It must not maintain a separate
game state.

## Contract

- SQLite is the source of truth for stories, sessions, chat history, world
  state, character state, saves, achievements, codex data, projects, fronts,
  investigations, and idempotency.
- The Go engine remains the source of truth for turn advancement. Browser
  actions are submitted through a JSON bridge command so terminal and browser
  responses use the same narrator pipeline.
- The Rust gateway serves HTTP, JSON APIs, the browser UI, and SSE realtime
  streams.
- Both clients must observe the same active session and turn cursor.
- Repeated browser submissions use `idempotency_key` and return the same stored
  event list.
- A cross-process story lock serializes AI/gameplay turn commits.

## Runtime Requirements

- The Docker service mounts the host Codex CLI and Codex OAuth config read-only.
  This keeps browser gameplay compatible with terminal stories configured for
  the `codex` provider without baking auth material into the image.
- If the host Codex installation moves, update the compose mount for
  `/usr/local/bin/codex` and verify with `docker exec oneday-gateway codex login
  status`.

## Browser Feature Inventory

- Story list and active story/session selection.
- Narrative transcript from canonical `chat_messages`.
- Choice buttons from assistant metadata (`output.choices_data` or choices).
- Free action composer and command composer.
- Live status strip: turn, location, chapter, sync state.
- Canonical state inspector:
  - character stats, inventory, skills, traits, known recipes
  - world map, locations, events, factions
  - active hooks/fronts/fallout
  - investigations
  - downtime projects
  - character timeline
  - achievements
  - sessions and searchable history
- Realtime updates through SSE for terminal-to-browser and browser-to-browser.
- Browser-to-terminal coherence through terminal polling of canonical turn
  state.

## Non-Goals

- The Rust gateway does not reimplement narrator prompts, combat math, crafting,
  or persistence logic already owned by the Go engine.
- The browser UI does not fake terminal-only subgames. It shows their canonical
  state and routes gameplay inputs through the shared turn bridge.
