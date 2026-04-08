# Phase 9: Narrative UX and Input Polish - Context

**Gathered:** 2026-04-09
**Status:** Ready for planning
**Source:** User-driven UX review + workspace analysis of TUI/runtime pain points after Phase 8

<domain>
## Phase Boundary

Phase 9 is a post-v1 polish and reliability phase focused on how the live narrative experience feels moment-to-moment. The scope is not "new gameplay systems"; it is the layer where AI/runtime data becomes usable terminal UX.

This phase covers:
- stale or malformed choices during live play and resume/load
- dialogue and relationship/event formatting quality
- overlay readability for `/stats`, save/load, and help-like panels
- keyboard-first behavior for common selection/confirmation flows
- quick save and explicit save/story management
- footer/status telemetry clarity
- optional ASCII art surfacing when the model emits it

This phase does not expand the core rules, combat engine, crafting rules, or plugin system. It should build on Phase 8's rendering contract instead of replacing it.

</domain>

<decisions>
## Implementation Decisions

### Scope and sequencing
- **D-01:** Phase 9 is a runtime UX/stability pass for the existing narrative loop, not a feature-expansion phase.
- **D-02:** Follow the order: turn-flow reliability first, presentation second, keyboard/telemetry integration third.
- **D-03:** External infrastructure issues that cannot be solved by repo code alone must still be captured in `.planning/todos/` so they are not lost.

### Turn-flow reliability
- **D-04:** Resume/load must rebuild the last visible turn from persisted local data, not from a synthetic "welcome back" fallback unless no better data exists.
- **D-05:** Choice lists must be sanitized before rendering: blank choices removed, duplicate text collapsed, IDs renumbered sequentially for the UI, and stale lists cleared immediately on selection.
- **D-06:** Resume should not feel like the story is "continuing by itself"; it should restore stable text, then present the current interaction state.
- **D-07:** Escaped newline artifacts such as literal `\\n\\n` should be normalized after parsing and before rendering.
- **D-08:** Character vitals must stay within valid bounds (`0 <= current <= max`) after state changes.

### Narrative presentation
- **D-09:** Relationship/NPC/skill/item/system updates should render as readable callout blocks, not a single markdown blockquote blob.
- **D-10:** Dialogue formatting should prefer `Speaker: "Quoted speech"` treatment when structured dialogue metadata exists, with distinct speaker/body styling that stays readable in long sessions.
- **D-11:** Stronger highlighting is allowed, but only for trusted entities from persisted state or structured metadata.
- **D-12:** Optional ASCII art should be surfaced only when present and reasonably bounded so it does not destroy the layout.

### Input and discoverability
- **D-13:** `Space` should mirror `Enter` in selection/confirmation UIs where that behavior is intuitive: menu, story/load pickers, save picker, overlays, and narrative choices.
- **D-14:** `Esc` from the narrative view should open a safer in-session decision point (resume/save/load/return) instead of immediately dumping the player to the main menu.
- **D-15:** Save/load/autosave should stay command-based under the hood, but the UI must make these paths more discoverable.
- **D-16:** Manual/quicksaves are explicit snapshots and should never be silently overwritten; only the rotating autosave may replace its previous version.
- **D-17:** Save deletion and story archive/delete are part of the management UX for this phase.

### Telemetry
- **D-18:** The status/footer should explain live response metrics better and surface cached prompt usage when the provider returns it.
- **D-19:** Footer telemetry must degrade gracefully when a provider omits usage, cost, or cache fields.

### the agent's Discretion
- Exact callout box glyphs, spacing, and accent colors, as long as they remain readable and do not rely on color alone
- Whether resume uses stored DB metadata, JSONL recovery, or both, as long as it avoids unnecessary AI calls and preserves correctness
- The exact shape of the in-session Esc menu, as long as it avoids accidental session loss and keeps the flow keyboard-first

</decisions>

<specifics>
## Specific Ideas

- Fix the specific "duplicate choice 1" class of issues by sanitizing UI numbering instead of trusting raw AI IDs.
- Quote-aware dialogue styling should make NPC speech visually distinct from narration without turning the whole scene into a chat log.
- Relationship callouts should feel like compact game-state updates, not debug output.
- `/stats` should stop truncating bios and relationships mid-line.
- Cached prompt usage is useful because OneDay has no local cache of its own; the provider/proxy may still be serving cached reads and the user should be able to see that.
- Quick save needs a fast keyboard path, and story/save cleanup should be possible without spelunking the filesystem.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Prior polish and known issues
- `.planning/TUI_RENDERING_NOTES_2026-04-08.md` — phase-8 rendering ideas and follow-up UI polish direction
- `.planning/CODE_REVIEW_2026-04-08.md` — reliability caveats that still matter for resume/load and render safety
- `.planning/phases/08-tui-rendering-polish/08-CONTEXT.md` — existing rendering contract assumptions
- `.planning/phases/08-tui-rendering-polish/08-01-SUMMARY.md` — what Phase 8 actually delivered
- `.planning/phases/08-tui-rendering-polish/08-02-SUMMARY.md` — choice rendering and integration baseline

### Narrative runtime and persistence
- `internal/engine/narrator.go` — turn prep/finalization, streaming, resume/load behavior
- `internal/engine/session.go` — JSONL persistence and DB chat message metadata
- `internal/storage/chat.go` — recent-message retrieval used by resume/load
- `internal/engine/state.go` — state_changes application and vital clamping behavior
- `internal/engine/rendering.go` — callout derivation from applied state changes
- `internal/engine/types.go` — narrative response, dialogue blocks, callouts, choices

### Narrative UI and selection flow
- `internal/tui/views/narrative.go` — live narrative loop, typewriter, choice integration, overlays, commands
- `internal/tui/views/narrative_rendering.go` — semantic rendering bridge
- `internal/tui/views/narrative_choices.go` — choice item building and stat-label resolution
- `internal/tui/components/choicelist.go` — choice selection behavior and rendering
- `internal/tui/components/overlay.go` — overlay wrapping/scrolling behavior
- `internal/tui/components/statusbar.go` — footer metrics and vital display
- `internal/tui/views/menu.go` — main menu keyboard flow
- `internal/tui/views/loadstory.go` — load story picker keyboard flow
- `internal/tui/views/saveload.go` — save picker keyboard flow
- `internal/tui/app.go` — global Esc/view routing and narrative session transitions

### Provider telemetry
- `internal/ai/provider.go` — provider usage model and stream chunk contract
- `internal/ai/providers/openai_compat.go` — usage/cost parsing and OpenAI-compatible streaming path

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Phase 8 already introduced structured dialogue blocks, trusted known-entity collection, and event callouts; Phase 9 should refine the presentation rather than invent a second renderer.
- The status bar already tracks total duration, first-token timing, token usage, throughput, and cost.
- `session.go` already stores assistant metadata JSON alongside narrative text, which can be used to improve resume without fresh AI calls.
- The app already has separate picker views for load/save, so keyboard parity can be improved without introducing a new framework.

### Gaps to close
- Overlay rendering currently truncates lines instead of wrapping them.
- Resume/load still tries to reconstruct the last turn from plain assistant content instead of using the richer local metadata it already stores.
- The provider usage model does not yet expose cached prompt usage.
- Choice rendering trusts AI-provided IDs too much and does not sanitize malformed/duplicate IDs before showing them.

</code_context>

<deferred>
## Deferred Ideas

- Keyboard-driven inspect/help for stat badges in the choice list
- Richer settings editor beyond the current read-only settings screen
- Provider/key management UI
- Broader context-window optimization work beyond surfacing cache stats

</deferred>

---

*Phase: 09-narrative-ux-and-input-polish*
*Context gathered: 2026-04-09*
