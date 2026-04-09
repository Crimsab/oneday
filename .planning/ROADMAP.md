# Roadmap: OneDay

**Milestone:** v1.1 — Reliability and Systemic World Depth
**Phases:** 21
**Requirements:** 99 mapped

> Note: the original `v1.0` milestone was completed through Phase 14. Phases 15-21 define the next roadmap extension, with Phases 15-19 recommended as the first execution batch and Phases 20-21 intentionally deferred until the new systemic world-state layers stabilize.

---

## Phase 1: Foundation

**Goal:** Establish the project skeleton, configuration system, storage layer, and AI provider router so all subsequent phases can build on a stable, tested core.
**Requirements:** CONF-01, CONF-02, AI-01, STOR-01
**Depends on:** —
**UI hint:** no

### Success Criteria

1. Running `oneday` binary starts without error on both Linux and Windows (cross-compile).
2. `config.yaml` controls AI provider order; switching the chain changes which provider is called.
3. SQLite database is created automatically on first run with all v1 schema tables.
4. AI provider router falls back across the configured chain (currently LiteLLM → OpenRouter → Claude Code) on failure, logged to console.

### Plans

- [ ] Plan 1.1: Initialize Go module, project layout, `Makefile`/`just` build targets, cross-compile targets (Linux + Windows)
- [ ] Plan 1.2: Implement `config/` package — `config.yaml` schema, load/save, defaults, env override
- [ ] Plan 1.3: Implement `storage/` package — SQLite connection, migration runner, CRUD helpers for all v1 entities
- [ ] Plan 1.4: Implement `ai/` package — provider interface, Claude Code CLI adapter, LiteLLM adapter, OpenRouter adapter, fallback chain router

---

## Phase 2: TUI Shell and Story Bootstrap

**Goal:** The player can launch the game, see a working main menu, create a new story through AI-guided conversation, and choose their protagonist name — all rendered in a functional Bubbletea TUI.
**Requirements:** TUI-01, TUI-02, TUI-03, CORE-01, CORE-02, AI-02, AI-03, AI-05
**Depends on:** Phase 1
**UI hint:** no

### Success Criteria

1. Player sees a styled main menu with New Story, Load Story, Settings, Quit options.
2. Selecting "New Story" starts an AI conversation that asks questions and produces a valid `story.json` with `setting`, `stats_schema`, and world rules.
3. Player can enter protagonist name and optional background before the story begins.
4. AI responses stream to the TUI with a visible typewriter effect.
5. Status bar shows current AI model name and last response latency in milliseconds.

### Plans

- [ ] Plan 2.1: Implement Bubbletea app skeleton — model, views router, key bindings, Lipgloss base theme
- [ ] Plan 2.2: Implement main menu view (`TUI-01`)
- [ ] Plan 2.3: Implement story creation flow — AI conversation that builds and writes `story.json` (`CORE-01`, `CORE-02`)
- [ ] Plan 2.4: Implement AI response streaming with typewriter render and status bar model/latency display (`AI-02`, `AI-03`, `AI-05`)

---

## Phase 3: Core Narrative Loop

**Goal:** The player can enter a loaded story and play turns — type a free action or pick a suggested choice, receive a streamed AI narrative response, have game state updated, and save/load at any time.
**Requirements:** CORE-03, CORE-04, CORE-05, CORE-06, AI-04, STOR-02, STOR-03, CMD-01
**Depends on:** Phase 2
**UI hint:** no

### Success Criteria

1. Player types a free action or selects a numbered choice; AI responds with narrative + new choices within the same session.
2. Game state (protagonist, world) is written to disk after every turn and can be loaded from the main menu.
3. Autosave fires every N turns (configurable); `/save` and `/load` work as commands.
4. Context sent to AI includes system prompt, recent chat turns, player state, and NPC state.
5. `/inventory`, `/stats`, `/help`, `/quit` commands produce correct output or actions.

### Plans

- [ ] Plan 3.1: Implement narrative view (`TUI-02`) — chapter/location header, narrative scroll area, choices list, free-action input
- [ ] Plan 3.2: Implement context builder (`AI-04`) — assemble system prompt + recent chat + protagonist state + NPC state
- [ ] Plan 3.3: Implement game loop — player input → context build → AI call → parse JSON response → apply `state_changes` → render
- [ ] Plan 3.4: Implement session management — auto-create on story open, close on exit (`STOR-03`); JSONL chat persistence (`STOR-02`)
- [ ] Plan 3.5: Implement autosave + manual save/load (`CORE-05`, `CORE-06`) and core chat commands (`CMD-01`)

---

## Phase 4: Character and NPC Systems

**Goal:** Characters grow organically through play (attributes, skills, traits, titles) and NPCs are generated at runtime with personality, private thoughts, desires, and disposition tracking.
**Requirements:** CHAR-01, CHAR-02, CHAR-03, CHAR-04, CHAR-05, CHAR-06, NPC-01, NPC-02, NPC-03, NPC-04, NPC-05, NPC-06, TUI-06, TUI-07
**Depends on:** Phase 3
**UI hint:** no

### Success Criteria

1. New character starts with near-zero stats defined by the story's `stats_schema`; no pre-set traits or skills.
2. After repeated physical actions, AI suggests a +1 attribute increase; it applies and persists.
3. AI assigns a trait after observing behavioral patterns (e.g. aggressive choices); trait is visible in `/stats`.
4. A skill (e.g. "Scassinare") appears with XP and level after the player attempts and learns it.
5. First encounter with an NPC generates and persists their full profile (personality, speech, quirks, values, fears, desires, disposition); private thoughts are never displayed to the player.
6. `/stats` shows full character sheet; `/inventory` shows backpack, equipped, quest items.

### Plans

- [ ] Plan 4.1: Implement `character/` package — stat schema loader, attribute growth handler, skill XP/level system, trait/title assignment from AI `state_changes`
- [ ] Plan 4.2: Implement character sheet (`TUI-06`) and inventory view (`TUI-07`)
- [ ] Plan 4.3: Implement `npc/` package — AI-driven NPC generation on first encounter, persistence, disposition updates, private thought/desire management (`NPC-01`–`NPC-06`)

---

## Phase 5: RAG, Chapters, and Dynamic World

**Goal:** The game maintains long-term narrative coherence through periodic summarization and vector retrieval, chapters are tracked with AI-generated summaries, and the world state (story.json, NPCs, world_state) evolves dynamically through gameplay and `/narrator`.
**Requirements:** RAG-01, RAG-02, RAG-03, RAG-04, STOR-04, STOR-05, WORLD-01, WORLD-02, WORLD-03, CMD-02, CMD-03, CMD-04
**Depends on:** Phase 4
**UI hint:** no

### Success Criteria

1. Every N turns, the last N turns are summarized and the summary is embedded and stored in SQLite as BLOB vectors.
2. Context builder retrieves top-K relevant chunks through cosine similarity in Go and includes them in AI prompts.
3. Combat and crafting sessions have separate `.jsonl` files from the main narrative.
4. When a chapter ends, the AI generates and saves a chapter summary; `/journal` shows chapters.
5. `/n Aggiungi una fazione segreta` updates `story.json` factions and the AI weaves it into future turns.
6. `/map` and `/achievements` commands render their respective views.

### Plans

- [ ] Plan 5.1: Implement `rag/` package — periodic summarization trigger, embedding via LiteLLM text-embedding-3-small, SQLite BLOB vector store/retrieve
- [ ] Plan 5.2: Wire RAG retrieval into context builder (`RAG-04`); implement separate chat logs for combat/crafting/dialogue (`STOR-04`)
- [ ] Plan 5.3: Implement chapter system — AI-driven chapter boundary detection, summary generation, JSONL organization by chapter (`STOR-05`)
- [ ] Plan 5.4: Implement `/narrator` (`CMD-03`, `CMD-04`) — meta-input tagging, AI system instruction path, auto-update to `story.json`/NPC files/`world_state`
- [ ] Plan 5.5: Implement dynamic world tracking (`WORLD-01`–`WORLD-03`) and `/map`, `/journal`, `/achievements` commands (`CMD-02`)

---

## Phase 6: Combat, Crafting, and Challenges

**Goal:** All major gameplay systems are playable — turn-based combat with engine-side damage resolution, AI-driven crafting through conversation, and the full challenge suite (stat checks, dice rolls, mini-games).
**Requirements:** COMBAT-01, COMBAT-02, COMBAT-03, COMBAT-04, COMBAT-05, COMBAT-06, CRAFT-01, CRAFT-02, CRAFT-03, CRAFT-04, CHAL-01, CHAL-02, CHAL-03, CHAL-04, STOR-04, TUI-04, TUI-05
**Depends on:** Phase 5
**UI hint:** no

### Success Criteria

1. Entering combat opens a separate chat session; the TUI switches to the combat view with HP bars and turn info.
2. Player damage and enemy damage are calculated by the engine (stats + dice + modifiers), not AI; the AI only narrates the result.
3. Typing a free action (e.g. "Fuggo!") works mid-combat and the AI resolves it narratively.
4. On player defeat, AI decides outcome from context (death/capture/rescue/unconscious/retreat); a combat summary appears in the main narrative.
5. Crafting opens a conversation view; the AI evaluates feasibility and creates items with narrative effects; discovered recipes persist.
6. A dice roll challenge shows an animated d100 roll in the TUI with visible modifiers before the result.
7. All four mini-games (rock-paper-scissors, memory sequence, quick-time, riddle) are playable.

### Plans

- [ ] Plan 6.1: Implement `combat/` package — enemy generation, damage formula engine, turn loop, defeat handler, post-combat summary injection
- [ ] Plan 6.2: Implement combat TUI view (`TUI-04`) — HP bars, turn indicator, choices + free input
- [ ] Plan 6.3: Implement `crafting/` package — conversation session, feasibility evaluation, item creation with narrative effects, recipe persistence (`CRAFT-01`–`CRAFT-04`)
- [ ] Plan 6.4: Implement crafting TUI view (`TUI-05`) — inventory panel + conversation area
- [ ] Plan 6.5: Implement `challenges/` package — stat check engine, dice roll engine with TUI animation, item/skill/relationship checks, four mini-game implementations (`CHAL-01`–`CHAL-04`)

---

### Phase 06.1: Bugfix and Stabilization — fix resume/load, save restoration, inventory contract, config alignment, story metadata, runtime contract (INSERTED)

**Goal:** [Urgent work - to be planned]
**Requirements**: TBD
**Depends on:** Phase 6
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 06.1 to break down)

## Phase 7: Polish, Achievements, and Mood Theming

**Goal:** The game feels complete and cohesive — achievements fire dynamically, the TUI reacts to narrative mood with themed styles, and all remaining quality-of-life features are in place.
**Requirements:** ACH-01, ACH-02, ACH-03, ACH-04, TUI-08, TUI-09
**Depends on:** Phase 6
**UI hint:** no

### Success Criteria

1. After a noteworthy moment, an achievement notification popup appears in the TUI with name, rarity badge, and category.
2. Achievements are per-story, persist correctly, and appear in `/achievements`.
3. Narrative mood changes (e.g. `"mood": "tense"`) visibly shift the TUI color scheme via Lipgloss.
4. Epic or legendary achievements use a distinct visual treatment from common ones.
5. The full game loop — create story → play turns → combat → craft → narrator command → achievements — works end-to-end without errors.

### Plans

- [ ] Plan 7.1: Implement `achievements/` package — AI `achievement_earned` output parser, per-story storage, rarity/category model, `achievement_rules.md` injection into AI context (`ACH-01`–`ACH-04`)
- [ ] Plan 7.2: Implement achievement notification popup (`TUI-08`) — overlay with name, description, rarity badge, category icon
- [ ] Plan 7.3: Implement mood-based theming (`TUI-09`) — Lipgloss style sets per mood (tense, peaceful, dark, epic, etc.), reactive to `mood` field in AI response
- [ ] Plan 7.4: End-to-end integration pass — verify all 63 requirements are exercised, fix edge cases, stabilize for v1.0 release

Note for future planning:
- See `.planning/TUI_RENDERING_NOTES_2026-04-08.md` for functional TUI expansion ideas:
  speaker styling, known-entity highlighting, event callouts, semantic choice rendering based on
  generic intent/risk metadata with optional dynamic stat badges, combat/challenge telemetry, and
  achievement/chapter moment cards.
- See `.planning/CODE_REVIEW_2026-04-08.md` for prerequisite reliability fixes that should be
  considered before or alongside major TUI polish.

---

## Phase 8: TUI Rendering Polish

**Goal:** The narrative TUI becomes more readable, semantically expressive, and decision-friendly without relying on brittle prose parsing or decorative-only styling. This phase focuses on the main narrative view only.
**Requirements:** AI-06, TUI-10, TUI-11, TUI-12, TUI-13, TUI-14
**Depends on:** Phase 7
**UI hint:** no
**Status:** Complete (2026-04-08)

### Success Criteria

1. Narrative rendering can distinguish narrator prose, NPC speech, player/meta voice, and structured dialogue blocks when metadata is available.
2. Known entities (NPCs, locations, factions, items, skills, titles, chapter names) can be highlighted using persisted state and/or structured metadata rather than ad-hoc keyword coloring.
3. Important state changes surface as compact event callouts separate from the main prose.
4. Suggested choices can render semantic intent/risk metadata and optional story-schema stat badges, while remaining fully usable when metadata is absent.
5. The renderer degrades gracefully to safe plain rendering when metadata is partial or missing, and this behavior is covered by focused renderer tests.

### Plans

- [x] Plan 8.1: Define the rendering data contract and build the narrative semantic renderer foundation — speaker styling, entity highlighting, event callout pipeline, and strong fallback behavior
- [x] Plan 8.2: Implement semantic choice rendering and integration polish — intent/risk metadata, dynamic stat badges, renderer verification, and narrative-view integration pass

## Phase 9: Narrative UX and Input Polish

**Goal:** Tighten the live play experience around the narrative view: make dialogue and relationship updates readable, fix stale/duplicated choice behavior, improve resume/load/session UX, and add keyboard-first + telemetry polish without changing the core game loop.
**Requirements**: AI-07, STOR-06, STOR-07, STOR-08, TUI-15, TUI-16, TUI-17, TUI-18, TUI-19, TUI-20, TUI-21, TUI-22
**Depends on:** Phase 8
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. Resume/load restores the last local turn cleanly without synthetic fallback narration, stale choices, or a fresh AI bill.
2. Structured dialogue renders with stronger speaker formatting and quoted speech treatment; relationship/NPC/skill/item updates are readable as separate callout blocks, not markdown blobs.
3. Choice lists sanitize duplicate/malformed IDs, clear immediately after selection, and do not leak stale rows into the next turn.
4. `Space` behaves like `Enter` anywhere the player is selecting or confirming an action, and `Esc` from the narrative view no longer throws the player straight back to the main menu without an in-game decision.
5. Quick save is available from a convenient hotkey, manual saves remain as explicit snapshots, and save deletion is available from the save-management UI.
6. Stories can be archived or deleted from the load/manage flow, and archived stories are kept out of the active play list by default.
7. `/stats` and other overlays wrap long content correctly, vitals stay clamped within valid ranges, escaped newlines are normalized, and optional ASCII art can appear safely when provided.
8. The footer/status line shows clearer response timing and token/cost telemetry, including cached prompt usage when the provider exposes it.

Plans:
- [x] Plan 9.1: Fix turn-flow reliability — resume/load restoration, choice sanitization/lifecycle, escaped-newline cleanup, and state clamping
- [x] Plan 9.2: Polish narrative presentation — dialogue quotes, relationship/system callout cards, optional ASCII art surfacing, overlay wrapping, and stronger trusted highlights
- [x] Plan 9.3: Improve keyboard/session UX, save/story management, and footer telemetry — Space/Enter parity, safer Esc flow, quick save, delete/archive flows, and cached-token/status metrics

---
## Phase 10: Ambient ASCII Art and Model Benchmarking

**Goal:** Add scene-aware ambient ASCII art as an optional same-turn enhancement, close the remaining narrative UX follow-ups, and benchmark dedicated ASCII-capable models so the feature can use the right model/cost profile instead of overloading the main narrator.
**Requirements**: AI-08, AI-09, BENCH-01, BENCH-02, TUI-23, TUI-24, TUI-25
**Depends on:** Phase 9
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. Narrative responses can request optional ambient ASCII art through structured cue metadata instead of embedding large art blobs directly in the main narrator payload.
2. When a valid ASCII cue is present, the game can trigger a second, specialized AI call and render the resulting art into the same scene without advancing the story to a new turn.
3. ASCII art appears only for curated ambient scenarios such as first location reveals, chapter openers, signage, terminals, ritual diagrams, maps, or iconic objects, and degrades cleanly when unavailable.
4. Choice-related stat badges have an in-context keyboard inspect/help flow so the player can understand referenced stats without leaving the narrative view blindly.
5. Local developer builds refresh the root `./oneday` binary in addition to cross-platform outputs, so testing `./oneday` from the repo root runs the latest code.
6. The repository includes a dedicated ASCII-art benchmark with latency, throughput, cost, and OpenRouter model catalog metadata for the candidate models under evaluation.
7. OneDay's LiteLLM/OpenRouter setup is aligned with the chosen narrator, fallback, embedding, and ASCII-art models so the default local config no longer drifts from the actual proxy capabilities.

Plans:
- [x] Plan 10.1: Introduce structured `ascii_cue` metadata, same-turn ambient ASCII generation, renderer integration, and scene-safe fallback behavior
- [x] Plan 10.2: Close remaining narrative UX follow-ups — in-context stat inspect/help, clearer ASCII/telemetry docs, and a reliable local root-binary build flow
- [x] Plan 10.3: Extend the benchmark suite for ASCII-art generation, evaluate the candidate models with cost/speed/context metrics, and align provider/proxy config with the selected defaults

---

## Phase 11: Runtime Reliability and History UX

**Goal:** Close the most visible live-play rough edges by making narrative playback reliable, keeping interaction timing coherent, improving challenge/session affordances, and adding a searchable history flow without changing the overall game structure.
**Requirements**: AI-10, CMD-05, TUI-26, TUI-27, TUI-28, TUI-29, TUI-30, TUI-31, TUI-32, TUI-33, TUI-34, TUI-35
**Depends on:** Phase 10
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. Live narrative playback never leaks raw ANSI escape sequences, and choices/free input do not appear before the visible scene playback is complete.
2. Story bootstrap is more resilient: invalid structured story drafts are validated locally and retried/repaired before the user sees a hard failure.
3. Active dice rolls and mini-games show a lightweight confirmation/prelude before starting, so events never seem to fire “by themselves”.
4. Choice navigation no longer scrolls the narrative viewport, while mouse-wheel scrolling applies only to the narrative/chat area.
5. Trusted locations, world names, factions, and similar entities render with clearer role-aware emphasis; dialogue and relationship/system updates render as compact structured blocks rather than flat markdown blobs.
6. Players can inspect current/legacy choice metadata more usefully, see visible quick-save feedback, and read the footer/status area cleanly even on narrower terminals.
7. A `/history` flow lets the player review prior interactions with optional text search without leaving the current story context.

Plans:
- [x] Plan 11.1: Fix runtime reliability — ANSI-safe typewriter playback, synchronized choice reveal, story-definition validation/retry, and challenge confirmation flow
- [x] Plan 11.2: Polish live narrative interaction — scroll/input separation, mouse-wheel behavior, stronger entity/dialogue/callout rendering, inspect/help improvements, and visible save/status feedback
- [x] Plan 11.3: Add searchable interaction history and final runtime polish — history command/overlay, narrow-footer layout, help/discoverability, and verification updates

### Phase 12: State rollback integrity and narrative persistence

**Goal:** Make save/load, canonical history, and long-term memory deterministic again so a resumed story never leaks future knowledge and every system-side event that should shape continuity is actually persisted.
**Requirements**: TBD
**Depends on:** Phase 11
**Plans:** 3 plans

### Success Criteria

1. Loading an older save restores a coherent snapshot of the run, including canonical chat/history inputs used for resume and long-term memory, without replaying future turns or future state.
2. `/narrator` interactions and combat/crafting follow-up summaries are persisted in the canonical log and survive resume, history, chapter summaries, and RAG retrieval.
3. The command surface is internally consistent: documented commands such as `/craft` are actually parseable and routed.
4. RAG embedding setup is explicit and provider-agnostic enough that the feature does not silently disappear just because LiteLLM is disabled.
5. Autosave cleanup removes superseded snapshot files, and regression tests cover rollback, narrator persistence, and combat-summary persistence.

Plans:
- [x] Plan 12.1: Build true rollback snapshots and restore canonical story/session state
- [x] Plan 12.2: Persist narrator-meta and combat-summary events in canonical history
- [x] Plan 12.3: Fix `/craft` routing, autosave hygiene, embedding-provider fallback, and verification

### Phase 13: Living world feedback, NPC conversations, and rewind UX

**Goal:** Make the world easier to follow and more reactive without railroading the player: surface clear turn deltas, track unresolved hooks, deepen nearby NPC relationships and conversations, add world-reaction/fail-forward feedback, and support safer rewind exploration.
**Requirements**: TBD
**Depends on:** Phase 12
**Status:** Complete (2026-04-09)
**Plans:** 3/3 plans complete

### Success Criteria

1. After each turn, the player can see a structured “what changed” summary covering stat shifts, inventory changes, relationship movement, unlocked lore/hooks, and other meaningful consequences.
2. The game maintains a lightweight hook tracker for promises, debts, mysteries, timers, and unresolved story threads, and uses it to improve continuity without forcing explicit chapter objectives.
3. Nearby NPCs support a dedicated conversation flow with persistent multi-axis relationships (for example trust, fear, debt, respect, intimacy) instead of a single disposition number.
4. Failures tend to resolve through fail-forward consequences such as injuries, complications, lost reputation, heat, rumors, or delays rather than dead stops, and those downstream reactions become visible through a world-reaction feed.
5. Players can branch or rewind around pivotal moments more safely, and the phase also improves high-value interaction UX such as downtime scenes and practical crafting guidance.

Plans:
- [x] Plan 13.1: Add structured turn-delta summaries and a persistent hook tracker
- [x] Plan 13.2: Build nearby NPC conversations, richer relationship axes, and world reaction/fail-forward systems
- [x] Plan 13.3: Add branch-friendly rewind UX, downtime scenes, and practical crafting QoL

### Phase 14: Story Hub, Achievement Archive, and Codex Dossiers

**Goal:** Add browseable archives and knowledge surfaces so players can inspect story achievements, protagonist/NPC dossiers, and a descriptive codex from both the home surface and inside a run without flattening everything into static text overlays.
**Requirements**: TBD
**Depends on:** Phase 13
**Plans:** 3/3 plans complete

### Success Criteria

1. From the home surface, the player can browse stories and inspect each story's unlocked achievements without loading into the live narrative session first.
2. Story achievements are clearly story-scoped: the home/archive flow can browse all stories, while the in-story achievements view only shows the current story and supports opening a full description for a selected entry.
3. The protagonist sheet evolves beyond a long plain-text overlay into an inspectable dossier that surfaces stats, traits, titles, relationships, and other relevant run data more clearly.
4. A codex aggregates canonical story knowledge into descriptive entries for people, places, factions, mysteries, and active threads, while keeping hidden NPC-only information private.
5. Codex navigation uses click-through drill-down or stacked inspector instances rather than accordion-only expansion, so linked entries can be explored without losing context.
6. Narrative dialogue renders more reliably even when the prose contains direct speech in single quotes, and duplicated prose-vs-dialogue output is normalized into clearer speaker-aware presentation.

Plans:
- [x] Plan 14.1: Build a home-surface story archive and structured achievement browser
- [x] Plan 14.2: Upgrade character inspection into protagonist and character dossiers
- [x] Plan 14.3: Implement a descriptive codex, dialogue normalization, and multi-instance drill-down navigation

### Phase 15: Canonical Turn Commit and Persistence Integrity

**Goal:** Make turn application, persistence, and resume/load behavior canonically coherent so partial failures cannot desync world state, session turn numbering, or canonical history.
**Requirements**: TBD
**Depends on:** Phase 14
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. A turn either commits canonical state/history coherently or fails without advancing `session.turn`, `world.CurrentTurn`, or the canonical log.
2. Critical persistence failures are surfaced and handled; narrator/session code no longer ignores write failures that can corrupt continuity.
3. Resume/load after interrupted or partially failed writes reconstructs a deterministic canonical snapshot.

### Plans

- [x] Plan 15.1: Define a single canonical turn-commit contract and route narrator/session persistence through it
- [x] Plan 15.2: Make DB-backed canonical persistence authoritative and treat JSONL as a derived mirror/log, not the source of truth
- [x] Plan 15.3: Remove silent persistence-error swallowing and add degraded-mode/backfill behavior for non-canonical mirrors
- [x] Plan 15.4: Add failure-injection and rollback/resume regressions for narrator, combat, crafting, and save/load paths

### Phase 16: Streaming/Memory Reliability and Provider Capability Hardening

**Goal:** Bring `Stream()` and `Complete()` to feature parity, fix RAG turn-window semantics, and make embedding selection deterministic and capability-aware.
**Requirements**: TBD
**Depends on:** Phase 15
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. Structured-output guards, retry/fallback behavior, and provider quirks are parity-matched between sync and streaming AI paths.
2. RAG summarization includes the first committed turn correctly and never skips or duplicates summary windows across resume/rewind.
3. Embedding provider selection obeys explicit config or first-capable probing with clear logs when embeddings are unavailable.

### Plans

- [x] Plan 16.1: Unify OpenAI-compatible request building, structured-response guards, and retry/fallback logic across sync and streaming paths
- [x] Plan 16.2: Fix turn-index and summary-trigger math for committed turns, first-window handling, and rewind/resume safety
- [x] Plan 16.3: Make embedding-provider selection explicit and capability-aware, with deterministic fallback and diagnostics
- [x] Plan 16.4: Fold touched boot/resume wiring duplication into a single initialization path and extend provider/RAG regressions

### Phase 17: Faction Fronts and Regional Pressure

**Goal:** Introduce engine-owned faction fronts/clocks that drive world escalation, opportunities, and regional heat/reputation in a persistent, inspectable way.
**Requirements**: TBD
**Depends on:** Phase 16
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. Factions/fronts persist segmented progress, stakes, visibility, and regional pressure that can change over time.
2. Turns, failures, narrator actions, and downtime can advance or reveal fronts through validated engine-side events.
3. World reactions, codex, and journals surface discovered pressure and front outcomes without leaking hidden information.

### Plans

- [x] Plan 17.1: Add canonical front/pressure state and engine events for advance, reveal, resolve, and consequence application
- [x] Plan 17.2: Fold local heat/reputation into fronts as regional pressure rather than a separate system
- [x] Plan 17.3: Integrate fronts with hook tracking, world reactions, fail-forward outcomes, and context-building
- [x] Plan 17.4: Surface known fronts and their fallout in codex/journal/dossiers with hidden-vs-known information rules

### Phase 18: Social Duels and Leverage Battles

**Goal:** Turn high-stakes negotiations, interrogations, seductions, tribunals, and stand-offs into a dedicated engine-resolved encounter type parallel to combat.
**Requirements**: TBD
**Depends on:** Phase 17
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. High-stakes dialogue can enter a social-duel flow with explicit objective, stance, leverage, and round state.
2. Outcomes are resolved by the engine from stats, skills, relationship axes, leverage, and situational pressure; AI narrates but does not own the mechanics.
3. Social failure resolves through fail-forward outcomes such as suspicion, debt, exposure, concessions, or heat rather than dead-end rejection.

### Plans

- [x] Plan 18.1: Define the social-duel state machine, action set, and engine-side resolution formulas on top of challenge and relationship systems
- [x] Plan 18.2: Add narrator/context contracts so the AI can frame stakes and consequences without bypassing engine authority
- [x] Plan 18.3: Build the duel UI/command flow between free narrative and full combat
- [x] Plan 18.4: Persist leverage, concessions, and aftermath into relationships, fronts, and world-reaction feeds

### Phase 19: Recurring Nemeses and Rival Escalation

**Goal:** Promote surviving rivals into persistent nemeses that remember scars, humiliations, tactics, and alliances, and re-enter the story through world-state escalation rather than random repetition.
**Requirements**: TBD
**Depends on:** Phase 18
**UI hint:** no

### Success Criteria

1. Eligible NPCs can be promoted into nemeses based on repeated conflict, escape, humiliation, political stakes, or major harms.
2. Nemeses persist scars, vows, remembered player patterns, and escalation tier across saves/resume.
3. Future encounters reintroduce nemeses coherently through fronts, world reactions, and codex/dossier state, with meaningful resolution paths beyond simple death.

### Plans

- [x] Plan 19.1: Add nemesis profile/state and promotion rules tied to combat, social duels, fronts, and relationship history
- [x] Plan 19.2: Extend encounter generation and retrieval so active nemeses are reused with remembered patterns and stakes
- [x] Plan 19.3: Surface nemesis dossiers, last-seen state, and escalation traces without leaking hidden plans
- [x] Plan 19.4: Support multiple resolution paths such as escape, capture, truce, humiliation, alliance, or succession

### Phase 20: Investigation Board and Evidence Logic

**Goal:** Add a persistent investigation surface for mysteries, conspiracies, clues, suspects, contradictions, and theory-building without turning the game into a rigid quest log.
**Requirements**: TBD
**Depends on:** Phase 19
**UI hint:** no

### Success Criteria

1. Clues, suspects, claims, contradictions, and active theories can be persisted and inspected as a structured investigation board.
2. The engine can normalize AI-proposed evidence updates into canonical board entities and links.
3. Investigation progress improves context quality and codex discoverability without exposing hidden truths prematurely.

### Plans

- [x] Plan 20.1: Define canonical investigation entities and link types for clues, suspects, claims, contradictions, and threads
- [x] Plan 20.2: Add engine-side normalization/validation for AI-proposed evidence updates and theory movement
- [x] Plan 20.3: Build board/codex integration and filtering so mysteries can be explored without flattening them into checklist quests

### Phase 21: Downtime Projects and Long-Arc Progress Clocks

**Goal:** Upgrade lightweight downtime scenes into persistent player-owned projects such as training, rituals, crafting lines, relationship arcs, and safehouse/base improvements.
**Requirements**: TBD
**Depends on:** Phase 20
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. Players can start, advance, pause, and complete long-arc projects across multiple downtime opportunities.
2. Project progress competes meaningfully with faction/front pressure so time spent improving oneself or one’s base has opportunity cost.
3. Completed projects produce durable mechanical and narrative changes rather than one-off flavor scenes.

### Plans

- [x] Plan 21.1: Add canonical project-clock state for training, rituals, crafting chains, relationships, and home/base improvements
- [x] Plan 21.2: Integrate downtime advancement with fronts, regional pressure, resource costs, and fail-forward consequences
- [x] Plan 21.3: Surface project status and outcomes through dossiers, codex, and downtime interaction flows

### Phase 22: QA, E2E Verification, and Release Hardening

**Goal:** Prove the post-Phase-21 game is shippable by adding build provenance, critical-path smoke coverage, a reusable QA matrix, and stronger release gates.
**Requirements**: TBD
**Depends on:** Phase 21
**UI hint:** no
**Status:** Complete (2026-04-09)

### Success Criteria

1. Running binaries clearly expose build identity so developers can tell whether `./oneday` matches current source without relying on timestamps.
2. Critical player flows have automated smoke coverage that passes locally and in CI without live-provider dependence.
3. A reusable QA matrix exists for long-play and cross-system scenarios, and issues found during the sweep are fixed or explicitly triaged.
4. Local and CI release flows both enforce the same rebuild and verification expectations before shipping artifacts.

### Plans

- [x] Plan 22.1: Add build identity and runtime provenance for local and release binaries
- [x] Plan 22.2: Extend deterministic smoke coverage across critical player flows
- [x] Plan 22.3: Run a reusable cross-system QA matrix and triage/fix findings
- [x] Plan 22.4: Harden release gates, docs, and operator workflow around rebuild/verify/ship

### Phase 23: Active Systems UX and World-State Navigation

**Goal:** Promote projects, investigations, fronts, and their fallout from codex-adjacent data into first-class player-facing TUI surfaces.
**Requirements**: TBD
**Depends on:** Phase 22
**UI hint:** no
**Status:** Planned (2026-04-09)

### Success Criteria

1. Players can open dedicated views for projects, investigations, and discovered fronts without relying on generic codex navigation alone.
2. These views make progress, stakes, contradictions, pressure, rewards, and fallout easier to understand at a glance than the current dossier/codex fallback.
3. Important system changes such as project breakthroughs, investigation shifts, and front escalation surface as readable player-facing callouts during live play.
4. Navigation between codex, dossiers, and the new active-system views feels unified instead of fragmented into unrelated commands.

### Plans

- [ ] Plan 23.1: Build a dedicated project workspace for active, paused, and completed long-arc projects
- [ ] Plan 23.2: Build a dedicated investigations workspace for cases, clues, suspects, contradictions, and leads
- [ ] Plan 23.3: Build a front tracker for discovered fronts, regional pressure, and visible world fallout
- [ ] Plan 23.4: Unify active-system navigation and add clearer live callouts for systemic state changes

---

## Requirements Coverage

| Phase | Requirement Count | IDs |
|-------|-------------------|-----|
| 1 | 4 | CONF-01, CONF-02, AI-01, STOR-01 |
| 2 | 8 | TUI-01, TUI-02, TUI-03, CORE-01, CORE-02, AI-02, AI-03, AI-05 |
| 3 | 8 | CORE-03, CORE-04, CORE-05, CORE-06, AI-04, STOR-02, STOR-03, CMD-01 |
| 4 | 14 | CHAR-01–CHAR-06, NPC-01–NPC-06, TUI-06, TUI-07 |
| 5 | 12 | RAG-01–RAG-04, STOR-04, STOR-05, WORLD-01–WORLD-03, CMD-02–CMD-04 |
| 6 | 17 | COMBAT-01–COMBAT-06, CRAFT-01–CRAFT-04, CHAL-01–CHAL-04, STOR-04*, TUI-04, TUI-05 |
| 7 | 6 | ACH-01–ACH-04, TUI-08, TUI-09 |
| 8 | 6 | AI-06, TUI-10–TUI-14 |
| 9 | 12 | AI-07, STOR-06–STOR-08, TUI-15–TUI-22 |
| 10 | 7 | AI-08, AI-09, BENCH-01, BENCH-02, TUI-23–TUI-25 |
| 11 | 12 | AI-10, CMD-05, TUI-26–TUI-35 |
| **Total** | **99** | All mapped requirements through Phase 11 |

*STOR-04 (separate chat logs for combat/crafting) spans Phase 5 (design) and Phase 6 (usage); primary mapping is Phase 5.
Post-v1.0 expansion phases 17-21 are now implemented; their requirements remain `TBD` until the next roadmap pass formalizes requirement IDs for the expansion systems.
