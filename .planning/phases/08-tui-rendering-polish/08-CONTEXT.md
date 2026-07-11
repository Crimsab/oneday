# Phase 8: TUI Rendering Polish - Context

**Gathered:** 2026-04-08
**Status:** Ready for planning
**Source:** Manual GSD discuss-phase equivalent based on workspace analysis + planning notes

<domain>
## Phase Boundary

Phase 8 is a narrative-view-only TUI polish pass. It improves readability, semantic clarity, and player decision support in the main narrative loop without expanding scope into combat telemetry, challenge-specific rendering, inspect mode, plugin work, or a wholesale visual redesign.

The phase should build on the reliability fixes already landed in Phase 6.1 and the achievement/mood polish from Phase 7. The outcome is a stronger narrative renderer driven by structured data and known game state, not brittle prose parsing or decorative styling.

</domain>

<decisions>
## Implementation Decisions

### Scope and sequencing
- **D-01:** Phase 8 is narrative-view only. Combat telemetry, challenge renderer, inspect mode, and plugin work are explicitly deferred.
- **D-02:** This phase assumes the Phase 6.1 reliability fixes are the baseline and should not reopen them unless a rendering dependency is still demonstrably broken.
- **D-03:** Follow the implementation order from `TUI_RENDERING_NOTES`: quick wins first (speaker styling, known-entity highlighting, event callouts), then semantic choice rendering and renderer verification.

### Rendering philosophy
- **D-04:** Do not use ad-hoc regex coloring as the primary mechanism for semantic rendering.
- **D-05:** Rendering should prefer structured AI output, persisted entities/state, engine-tracked `state_changes`, and story `stats_schema`.
- **D-06:** When structured metadata is absent or partial, the narrative view must fall back safely to the current plain rendering behavior without blocking gameplay.
- **D-07:** Mood theming remains soft and supportive; the main prose body should stay stable and readable during long sessions.

### Data contract
- **D-08:** Introduce an explicit UI-facing rendering contract between AI/runtime and the narrative renderer instead of scattering heuristics across `narrative.go`.
- **D-09:** The contract must stay story-agnostic: no hardcoded assumptions like STR/DEX/CHA or fixed genre vocabularies.
- **D-10:** Dynamic stat badges and choice annotations must resolve labels from the active story `stats_schema`.

### UX priorities
- **D-11:** Speaker styling must clearly differentiate narrator prose, NPC speech, player/meta voice, and system-level content when metadata exists.
- **D-12:** Known-entity highlighting should apply only to entities the game actually knows about: NPCs, locations, factions, items, learned skills, titles, and chapter names.
- **D-13:** Event callouts should surface meaningful state transitions separately from prose, especially for items, traits, skills, titles, recipes, NPC/location discovery, and world changes.
- **D-14:** Choice rendering should use generic semantic axes (`intent`, `risk`, `scope`, `certainty`) plus optional `related_stats`, and remain fully usable without them.

### Verification
- **D-15:** Phase 8 should add renderer-focused tests for dialogue rendering, entity highlighting, callout ordering, semantic choice fallback, and metadata omission behavior.

### the agent's Discretion
- Exact metadata field names, as long as they remain consistent across prompt, AI response parsing, engine types, and renderer input
- Exact glyphs, badge wording, and layout density, as long as they preserve readability and do not require color alone
- Whether the semantic renderer lives in a dedicated `internal/tui/rendering` package or an equivalently clean subsystem

</decisions>

<specifics>
## Specific Ideas

- Treat this as the missing "narrative rendering polish" phase that was prepared in notes but not yet implemented.
- The current achievement popup and mood theming are the visual ceiling for this pass: functional first, expressive second.
- The new renderer should make the main loop easier to read and act on, not busier.
- Event callouts should prefer engine-tracked facts over AI prose interpretation whenever possible.
- Semantic choice rendering should communicate meaning with labels/badges first and color second.

</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning basis
- `.planning/TUI_RENDERING_NOTES_2026-04-08.md` — Primary design notes for Phase 8 scope, ordering, guardrails, and future renderer contract
- `.planning/CODE_REVIEW_2026-04-08.md` — Reliability prerequisites and testing gaps that must remain in scope during planning
- `.planning/phases/06.1-bugfix-and-stabilization-fix-resume-load-save-restoration-in/06.1-01-SUMMARY.md` — Confirms baseline state/reliability fixes expected by rendering work
- `.planning/phases/07-polish-achievements-and-mood-theming/07-01-SUMMARY.md` — Existing achievement popup and mood theming capabilities
- `.planning/phases/07-polish-achievements-and-mood-theming/07-02-SUMMARY.md` — Current Phase 7 polish/fallback behavior already delivered

### Runtime contracts
- `internal/engine/types.go` — Narrative response, choice, challenge, combat, and AI-facing runtime types
- `internal/engine/narrator.go` — Main turn loop, AI parsing, state application, achievement/challenge wiring
- `internal/engine/context.go` — Current AI context assembly and state summary injection
- `internal/ai/response.go` — AI package narrative response contract and parsing
- `internal/ai/prompts/narrator.go` — Narrator JSON schema and output expectations

### Narrative UI surface
- `internal/tui/views/narrative.go` — Main narrative view and integration hub for rendering, overlays, and subviews
- `internal/tui/components/choicelist.go` — Current simple choice rendering component
- `internal/tui/components/markdown.go` — Current prose rendering path
- `internal/tui/components/statusbar.go` — Status bar behavior and mood tint support
- `internal/tui/theme/theme.go` — Base styles and existing visual language
- `internal/tui/theme/mood.go` — Current mood palette system

### State and entity sources
- `internal/storage/models.go` — Story, character, NPC, world, achievement models
- `internal/engine/state.go` — Structured state changes and `StateChange` descriptions suitable for callouts

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `NarrativeModel` already orchestrates history, typewriter, choices, overlays, achievements, combat/crafting/challenge subviews
- `ApplyStateChanges()` already returns structured `[]StateChange` with human-readable descriptions that can feed event callouts
- `theme/mood.go` and `statusbar.go` already provide subtle mood-aware styling primitives
- `ChoiceListModel` is isolated and can be upgraded without rewriting the whole view
- `RenderMarkdown()` provides a safe baseline fallback for prose when semantic metadata is missing

### Established Patterns
- AI contracts are defined first in prompts/types, then parsed in the engine, then consumed in TUI views
- Persistent world knowledge lives in storage models and should be preferred over speculative text parsing
- The narrative view already handles queued transient UI elements (overlays, achievement popups, challenges), so callouts should fit this compositional style

### Integration Points
- `internal/ai/prompts/narrator.go` and `internal/engine/types.go` must evolve together for any new metadata
- `internal/engine/narrator.go` is the handoff point between AI output, applied state changes, and renderer-facing data
- `internal/tui/views/narrative.go` and `internal/tui/components/choicelist.go` are the main user-facing integration surfaces

</code_context>

<deferred>
## Deferred Ideas

- Combat telemetry polish
- Challenge-specific renderer improvements
- Inspect mode
- Chapter/area/boss cards beyond the existing achievement popup
- Plugin system work

</deferred>

---

*Phase: 08-tui-rendering-polish*
*Context gathered: 2026-04-08*
