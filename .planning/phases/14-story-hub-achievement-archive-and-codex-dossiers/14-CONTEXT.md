# Phase 14 Context: Story Hub, Achievement Archive, and Codex Dossiers

## Objective

Make OneDay easier to browse and understand outside the raw narrative stream by adding a stronger story hub, per-story achievement archives, richer character dossiers, and a highly descriptive codex that stays grounded in canonical game state.

## Included Scope

- A home-surface archive flow for browsing stories and their unlocked achievements before entering a run
- Story-scoped achievement browsing inside the live narrative session
- A richer protagonist/character inspection flow with clearer stats, traits, relationships, and other relevant run data
- A codex for people, places, factions, mysteries, and active threads
- A narrative dialogue normalization pass so direct speech still renders as dialogue when prose uses single quotes or mixed quoting styles
- Keyboard-friendly drill-down navigation that can open linked entries in new inspector instances instead of collapsing everything into a single accordion
- Engine-first data shaping so codex and dossiers are grounded in stored story, character, NPC, chapter, hook, and achievement state

## Explicit Exclusions For Now

- Rewind/branch UX
  Rationale: already part of Phase 13 scope and intentionally left out here.
- Map v2 / graph map rendering
  Rationale: valuable, but separate from the archive/codex surfaces requested here.
- Ambient ASCII or home-surface art layers
  Rationale: no need to mix presentational art work into a data-heavy navigation phase.
- Large NPC schema redesign
  Rationale: Phase 13 is already reworking NPC and relationship behavior; Phase 14 should consume that state, not compete with it.
- Leaking private NPC thoughts or hidden motivations into player-visible codex entries
  Rationale: codex should be rich, not omniscient.

## Design Direction

### Homepage Interpretation

For this phase, "homepage" means the start-of-run surfaces reachable before loading into the live narrative loop: the main menu and/or load-story browser. The phase should not introduce a separate web UI or external dashboard.

### Achievement Browsing

Achievements remain story-scoped in storage and runtime behavior. The home surface should let the player browse stories first, then expand a story row accordion-style to reveal that story's unlocked achievements. Selecting an achievement should open its full description in a focused detail view rather than dumping every description inline at once.

### Character Dossiers

The current `/stats` overlay already exposes a lot of raw data, but it reads like a long text report. This phase should turn that information into a clearer dossier experience and make it easier to inspect protagonist and relevant character data without losing story context.

### Codex Interaction Model

The codex should not be an accordion. Selecting an entry should open a new inspect view or stacked instance so the player can move from person → faction → place → thread while preserving where they came from. The experience should feel like following links through a living case file, not toggling sections in one long list.

### Dialogue Reliability Follow-Up

The current runtime can already receive structured `dialogue_blocks`, but live scenes still degrade when the prose repeats those same lines with single quotes or mixed quoting styles. Phase 14 should explicitly absorb a dialogue-normalization follow-up now that the Phase 13 work is done: normalize quote styles, deduplicate prose-vs-structured dialogue more intelligently, and add a safe fallback that can still promote recognizable spoken lines into clearer dialogue rendering when structured blocks are missing or partial.

### Canonical Truth First

Codex entries and dossiers should be engine-first. Confirmed facts come from stored data such as:
- `stories`
- `characters`
- `npcs`
- `world_state`
- `chapters`
- `achievements`
- canonical message/session history

AI-authored prose can help summarize or polish an entry, but it must be optional, cacheable, and unable to invent facts or expose hidden NPC-only state.

## Open Planning Questions

1. Should the archive be a new top-level menu destination, a tab within load story, or both?
2. Which linked entities deserve first-class codex types in v1: people, places, factions, mysteries, threads only, or also items/chapters?
3. How much of the codex description should be fully deterministic versus synthesized from canonical state into a narrative summary?
