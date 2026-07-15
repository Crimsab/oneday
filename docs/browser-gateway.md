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
- The Go game contract owns browser-facing slash command descriptors. The Rust
  gateway exposes them at `/api/contracts/commands`; React may present them, but
  should not maintain a separate command truth.
- The Go CLI owns model routing semantics. Rust proxies `GET /api/config/models`
  to `oneday gateway-model-settings` and `PUT /api/config/models` to
  `oneday gateway-model-settings-update`. The update command writes the shared
  `config.yaml`, preserving secret placeholders by using an edit-safe config
  load path that does not expand environment variables.
- The Rust gateway serves HTTP, JSON APIs, the browser UI, and SSE realtime
  streams.
- Both clients must observe the same active session and turn cursor.
- Repeated browser submissions use `idempotency_key` and return the same stored
  event list.
- A cross-process story lock serializes AI/gameplay turn commits.

## Runtime Requirements

- The public Docker service does not mount host CLI binaries or authentication
  directories. Use an HTTP provider in the standard container setup. Advanced
  operators may add a private Compose override for a local CLI integration,
  but credentials must remain read-only and outside the repository.
- Browser visual asset generation prefers
  `ONEDAY_IMAGEGEN_PROVIDER=imagegen-bridge`, with native provider/fallback
  routing through `ONEDAY_IMAGEGEN_BRIDGE_URL`. The legacy `openclaw-bridge`
  route and generic OpenAI-compatible image endpoints remain available.
  The gateway forwards prompt, output format, size, optional resolution/aspect
  ratio/background, and stores the actual provider/model plus provider
  `revised_prompt` values on asset versions for audit.
- To send the saved asset prompt exactly as-is, set
  `ONEDAY_IMAGEGEN_APPEND_NEGATIVE_PROMPT=false`; otherwise the gateway appends
  the asset negative prompt as an `Avoid:` line for legacy providers; the native
  bridge receives it as a separate policy-aware field.

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
- Shared model/provider editor for Options, including provider priority,
  provider enablement, narrative/provider models, utility, repair, image/ascii,
  embedding, Codex reasoning, and planned TTS status.
- Large searchable Options workspace with category sidebar for interface,
  gameplay policy, spoken audio, visual/map generation, models, and runtime.
- Automatic browser challenge host: the narrator signals a situation, the Go
  engine selects a timing-free mechanic from narrative semantics and recent
  branch history, and the resolved result returns through the canonical turn
  bridge. The player never chooses the challenge family.
- Canonical known-location map with an optional generated background and
  generated location symbols. SVG topology, labels, current location, and
  accessibility remain authoritative when art is missing or pending.

## Non-Goals

- The Rust gateway does not reimplement narrator prompts, combat math, crafting,
  or persistence logic already owned by the Go engine.
- The browser UI does not invent or manually launch challenge state. It renders
  branch-scoped minigames selected by the Go engine and routes their result
  through the shared turn bridge.
