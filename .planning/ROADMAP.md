# Roadmap: OneDay

**Milestone:** v1.0 — First Playable
**Phases:** 9
**Requirements:** 81 mapped

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

### Phase 9: Narrative UX and Input Polish

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
| **Total** | **81** | All mapped requirements through Phase 9 |

*STOR-04 (separate chat logs for combat/crafting) spans Phase 5 (design) and Phase 6 (usage); primary mapping is Phase 5.
