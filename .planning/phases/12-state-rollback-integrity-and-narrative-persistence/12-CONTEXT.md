# Phase 12: state-rollback-integrity-and-narrative-persistence - Context

**Gathered:** 2026-04-09
**Status:** Ready for planning

<domain>
## Phase Boundary

Restore trust in the runtime state model. This phase fixes rollback semantics, canonical-history persistence, and long-memory continuity for existing systems without introducing large new gameplay features.

</domain>

<decisions>
## Implementation Decisions

### Snapshot Semantics
- **D-01:** Save/load must behave like a real rollback for all AI-visible continuity, not just `character` and `world_state`.
- **D-02:** Resume after loading a snapshot must reconstruct from snapshot-coherent history/session data, never from the latest story-wide messages.
- **D-03:** If full destructive rollback is risky, the implementation may introduce snapshot-scoped restore/branch metadata internally, but the player-facing outcome must still be deterministic and future-safe.

### Canonical History Contract
- **D-04:** `/narrator` interactions are part of canonical continuity and must be persisted in the main history path, not treated as best-effort side effects.
- **D-05:** Combat outcomes that affect future narration must land in the same canonical persistence layer used by resume, `/history`, chapter summaries, and RAG.
- **D-06:** Storage schema and runtime message types must be aligned; runtime code must not write unsupported `message_type` values and ignore the resulting insert errors.

### Command and UX Consistency
- **D-07:** The runtime command registry, help overlay, and command dispatcher must describe the same command surface. `/craft` is required to work if it is documented and routed by the view layer.
- **D-08:** Silent feature disappearance is unacceptable for core continuity systems. If long-memory support cannot initialize, the game should either use an explicit fallback path or surface that degradation clearly.

### Embedding and Memory Pipeline
- **D-09:** Embedding configuration must be decoupled from LiteLLM-only assumptions. The phase should move toward an explicit embedding provider contract or equivalent config that does not silently return `nil` when LiteLLM is off.
- **D-10:** RAG inputs must remain consistent with the canonical log so chapter summaries, narrator lore injections, and snapshot restore all reason over the same source of truth.

### Safety and Cleanup
- **D-11:** Autosave replacement must clean up superseded on-disk snapshot files, not only DB rows.
- **D-12:** Add regression coverage around rollback, narrator persistence, combat-summary persistence, and command parsing so these bugs stop reappearing after UX/runtime phases.

### the agent's Discretion
- Exact storage design for snapshot completeness: extra snapshot tables, snapshot manifests, snapshot-linked session/history restore, or another minimal design that preserves deterministic rollback.
- Whether combat summaries should reuse `narrative` message type with metadata or expand the schema with additional allowed message types.
- Whether degraded RAG initialization is surfaced through UI status text, logs, or both, as long as it is no longer silent.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project and roadmap
- `.planning/ROADMAP.md` — phase ordering, completed reliability work, and new Phase 12 boundary
- `.planning/PROJECT.md` — product goals and non-negotiable narrative/RAG expectations
- `.planning/REQUIREMENTS.md` — global requirement inventory for persistence, RAG, commands, and runtime behavior
- `.planning/CODE_REVIEW_2026-04-08.md` — prior reliability review and already-closed findings to avoid duplicating solved work

### Product behavior docs
- `README.md` — user-facing feature promises for saves, memory, combat, crafting, and narrator behavior
- `docs/design.md` — intended semantics for save/load, narrator command, combat summaries, session hierarchy, and RAG

### Runtime files
- `internal/tui/app.go` — story entry, RAG initialization, save load/resume flow
- `internal/tui/views/narrative.go` — command dispatch, combat/crafting return flow, local-only summary append behavior
- `internal/tui/views/combat.go` — combat end message emission to the narrative layer
- `internal/tui/views/crafting.go` — crafting session UX and return flow

### Engine and persistence files
- `internal/engine/save.go` — snapshot format, autosave replacement, restore path
- `internal/engine/session.go` — canonical JSONL append path, turn ownership, sub-session files
- `internal/engine/narrator.go` — resume behavior, turn finalize path, RAG summarization hook
- `internal/engine/narrator_cmd.go` — `/narrator` execution and logging contract
- `internal/engine/combat.go` — combat prompts, summary generation, and unused canonical-write helper
- `internal/engine/crafting.go` — crafting prompts and sub-session persistence
- `internal/engine/commands.go` — command registry/parser contract
- `internal/engine/state.go` — narrator/world mutations and lore-to-RAG persistence
- `internal/storage/migrations.go` — chat message schema and allowed roles/message types
- `internal/storage/chat.go` — story/session message retrieval behavior for resume and summarization
- `internal/storage/saves.go` — save snapshot DB schema
- `internal/rag/rag.go` — retrieval and summarization orchestration
- `internal/rag/vectorstore.go` — canonical chunk types and search behavior

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `GameSession.AppendTurn(...)` already centralizes canonical JSONL + DB persistence for user/assistant narrative turns.
- `resumeNarrativeFromStoredMessage(...)` in `internal/engine/narrator.go` already knows how to rebuild a `NarrativeResponse` from stored assistant metadata.
- `ApplyNarratorStateChanges(...)` in `internal/engine/state.go` already persists lore/world mutations and can push narrator lore into RAG chunks.
- `DeleteSave(...)` in `internal/engine/save.go` already shows the intended DB+filesystem delete pattern for snapshots.

### Established Patterns
- Main narrative persistence flows through DB `chat_messages` plus per-session `main.jsonl`.
- Combat and crafting currently use sub-session JSONL files first, with only selective reinjection into the main narrative layer.
- Resume logic assumes canonical continuity can be reconstructed from DB messages, so any feature not persisted there effectively does not exist for later turns.
- RAG is intentionally optional, but current initialization degrades silently and is tightly coupled to the LiteLLM config branch.

### Integration Points
- Save restore must coordinate `saves`, `chat_messages`, `sessions`, `chapters`, `achievements`, `npcs`, and RAG chunk sourcing.
- `/narrator` fix spans schema (`migrations.go`), storage writes (`chat.go`), engine logging (`narrator_cmd.go`), and downstream consumers (`ResumeNarration`, `/history`, summarizer).
- Combat-summary persistence spans `combat.go`, `views/combat.go`, `views/narrative.go`, and schema/runtime message-type alignment.
- Embedding-provider decoupling starts in `internal/tui/app.go` (`buildRAG`) and propagates through config and provider construction.

</code_context>

<specifics>
## Specific Ideas

- The user explicitly wants save/load to avoid "memorie del futuro" when restoring an older snapshot.
- Branch save / rewind is a desired direction, but it should come after or alongside fixing baseline deterministic rollback.
- Prompt role hygiene for combat/crafting (`system` prompt currently sent as `user`) is a low-severity cleanup worth folding in if the touched files are already being modified.

</specifics>

<deferred>
## Deferred Ideas

- Rich NPC relationship dimensions beyond disposition (`trust`, `fear`, `debt`, `respect`, romantic tension, broken promises)
- Dedicated smart NPC chat system with local nearby NPC discovery, directed questions, and conversational memory
- "What changed this turn?" panel with explicit deltas
- Automatic hook tracker for promises, mysteries, debts, timers, and active objectives
- Optional micro-objectives after chapter transitions
- Fail-forward expansion, crafting QoL surfacing, world reaction feed, downtime scenes, and broader terminal QoL features

These belong in later design/product phases once the state model is trustworthy again.

</deferred>

---

*Phase: 12-state-rollback-integrity-and-narrative-persistence*
*Context gathered: 2026-04-09*
