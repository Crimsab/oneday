# TUI Rendering Notes - 2026-04-08

## Purpose

Functional TUI/UX ideas to keep in mind for future GSD planning and implementation.

These notes are intentionally focused on **useful rendering**, not decoration for its own sake.
The goal is to improve readability, decision-making, and narrative clarity in a way that stays
compatible with OneDay's core rule: each story can define its own world, stats, and vocabulary.

## Prerequisite Review Notes

Before investing heavily in TUI polish, keep the technical blockers from
`.planning/CODE_REVIEW_2026-04-08.md` in scope.

Especially relevant prerequisites:

- fix resume/load turn corruption
- fix full save/load restoration for character state
- unify inventory representation across prompt, engine, persistence, and UI
- align config schema/runtime behavior

Reason:

many of the rendering ideas below become more useful only when the underlying state is reliable.
For example, entity highlighting, event callouts, choice badges, and future combat/challenge UI all
depend on trustworthy state, persistence, and structured data flow.

## Key Principle

Do not color arbitrary words with ad-hoc regex rules.

Prefer rendering that is driven by:

- structured AI output
- known entities already persisted in DB/state
- story `stats_schema`
- session/world/game events already tracked by the engine

This keeps the TUI expressive without becoming noisy or brittle.

## 1. Semantic Choice Rendering

### Important constraint

Each story is different. Stats and labels may be entirely custom.

Because of that, choice styling should **not** depend on hardcoded stat names such as STR/DEX/CHA.

### Recommended approach

Use a small set of story-agnostic semantic axes:

- `intent`: `attack`, `social`, `stealth`, `explore`, `observe`, `craft`, `survive`, `flee`, `use_item`, `lore`, `meta`
- `risk`: `low`, `medium`, `high`, `unknown`
- `scope`: `self`, `npc`, `world`, `party`, `environment`
- `certainty`: `safe`, `uncertain`, `desperate`

Optional dynamic link to custom stats:

- `related_stats`: array of stat keys from the current story schema

Example future choice shape:

```json
{
  "id": 2,
  "text": "Confronta la guardia e prova a convincerla",
  "intent": "social",
  "risk": "medium",
  "related_stats": ["cha", "wil"]
}
```

### UI behavior

- color should come mainly from `intent` + `risk`
- stat references should be rendered as badges using labels resolved from `stats_schema`
- if metadata is missing, fallback to current neutral rendering

This makes choice styling compatible with any story while still letting custom stats appear when relevant.

## 2. Speaker Styling

Add distinct rendering for dialogue structure:

- narrator prose
- NPC speech
- player speech/thought/action
- `/narrator` or meta/system voice

### Functional behavior

- render speaker names consistently
- keep speaker name visually separate from quoted speech
- allow NPC name styling to reflect relationship/disposition or faction affinity later

Example:

- `Lyanna:` visually distinct
- dialogue body remains readable and not over-styled
- narrator/meta responses should never be confused with in-world speech

## 3. Highlight Known Entities

Highlight only entities the system actually knows:

- NPCs
- locations
- factions
- items
- learned skills
- titles
- chapter names

### Source of truth

- DB entities
- world state
- story setting
- structured AI output

### Why this matters

This gives the player a light semantic map of the world without relying on arbitrary keyword coloring.

## 4. Event Callouts

Introduce compact callouts for meaningful state changes:

- new NPC encountered
- new location discovered
- world event recorded
- faction standing changed
- skill learned
- XP gained
- trait gained
- title earned
- item acquired/lost
- recipe discovered

### Rendering idea

Short single-line or boxed summaries separated from the main narrative prose.

These callouts should make the "state change layer" explicit instead of burying it inside paragraphs.

## 5. Soft Mood-Based Theming

Already aligned with Phase 7 / `TUI-09`.

Mood changes should affect:

- headers
- accents
- borders
- choice highlight
- status bar accents

Mood changes should **not** aggressively recolor every line of narrative text.

The body text should remain stable and readable.

Useful moods already suggested by the project:

- `tense`
- `peaceful`
- `dark`
- `mysterious`
- `epic`
- `lighthearted`

## 6. Combat Telemetry

When combat work becomes active:

- highlight incoming threats / telegraphed enemy actions
- clearly separate player damage vs enemy damage
- visually mark critical success / critical failure
- show status effects and temporary conditions distinctly
- make free-action options readable even during combat chaos

This should be functional first, dramatic second.

## 7. Challenge Rendering

When challenge systems land:

- clearly display check type
- clearly display relevant stat/skill/item/NPC relationship
- show modifier breakdown
- separate raw roll from final total
- make pass/fail/critical state obvious

For dynamic stats, render labels from `stats_schema`, never from hardcoded assumptions.

## 8. Achievement / Chapter / Big Moment Cards

When polish work becomes active:

- achievement popup by rarity
- chapter-end title card
- major area discovery card
- boss/legendary encounter card
- `/narrator` world update acknowledgment style

These should be used sparingly for major beats, not every turn.

## 9. Inspect Mode

Potential future UX expansion:

- inspect last-mentioned NPC
- inspect current location
- inspect faction
- inspect item referenced in latest turn
- inspect "what changed this turn"

This would pair especially well with entity highlighting and event callouts.

## 10. Suggested Implementation Order

### Quick wins

- speaker styling
- known-entity highlighting
- event callouts for existing `state_changes`

### Medium

- semantic choice metadata with graceful fallback
- soft mood-based theming
- richer status/event badges

### Later

- combat telemetry
- challenge renderer
- achievement/chapter cards
- inspect mode

## Guardrails

- Keep the default view readable on long play sessions.
- Never require color alone to convey meaning; pair color with labels, icons, wording, or layout.
- Use semantic metadata where possible instead of free-form keyword matching.
- Preserve compatibility with stories whose stats, genre, and tone differ radically.

## Recommended Future Planning Hook

When GSD reaches the relevant phases, use these notes for:

- Phase 6 combat/challenge UI planning
- Phase 7 achievement + mood/theming planning
- future TUI polish passes
- always review `.planning/CODE_REVIEW_2026-04-08.md` alongside this file before implementation planning

Especially important: semantic choice rendering should be designed around **generic intent/risk metadata**
plus **optional dynamic stat badges** resolved from story schema.

## Already Present Elsewhere In Project Plans

The following ideas are already present in roadmap/design/planning at least at a high level,
so they do not need a separate new planning artifact:

- conditions / status effects
- improved NPC dialogue / dedicated dialogue sessions
- journal and world map
- achievements UI direction
- mood-based theming
- chapter summaries and chapter system

This note should therefore focus on the missing or under-specified UX/rendering additions below.

## Additional Useful Ideas Not Explicitly Planned Elsewhere

### 11. Rendering Data Contract

Add a dedicated UI-facing contract between AI output and the renderer.

Potential future payloads:

- `entities_mentioned`
- `event_callouts`
- `dialogue_blocks`
- `scene_type`
- enriched choice metadata

Reason:

this avoids building visual behavior out of brittle prose parsing alone.

### 12. Semantic Renderer Layer

Introduce a renderer layer that consumes structured narrative/event/entity data and decides:

- which parts are prose
- which parts are dialogue
- which parts become badges/callouts
- which parts get highlight treatment

This should be an explicit subsystem, not ad-hoc styling scattered across views.

### 13. Renderer Test Strategy

Add golden/snapshot-style tests for rendering behavior.

Examples:

- dialogue block renders correctly
- known entity highlight appears for stored NPC/location/item
- event callouts render in stable order
- missing metadata falls back to plain safe output
- mood accent changes do not break readability

This will matter a lot once the TUI becomes more semantic and layered.

### 14. Strong Fallback Behavior

Define graceful fallback rules for partially-structured AI output.

Examples:

- no semantic metadata -> plain neutral choice rendering
- no known entity match -> leave text unhighlighted
- malformed event payload -> keep narrative visible and skip only the broken UI enhancement

The renderer should never make the core story unreadable.

### 15. Glossary / Codex Overlay

Potential overlay or command for discovered knowledge:

- NPCs
- factions
- locations
- world rules
- notable items
- learned skills

This is different from `/journal`: it is a semantic reference index, not a chronological log.

### 16. Turn Recap / "What Changed" View

Add a compact summary for the latest turn:

- new entities
- gained/lost items
- relationship shifts
- stat/skill/trait/title changes
- world changes

This could be an overlay, a collapsible section, or a lightweight command.

### 17. Pinned Entities / Focus Panel

Allow the UI to pin a few currently important entities:

- current NPC of interest
- current location
- current threat
- current objective lead

Useful for long sessions where context drift is a real problem.

### 18. Dialogue History View

Separate from full narrative history, provide a clean conversation-focused view:

- who spoke
- latest important exchanges
- current disposition context

Especially useful once dedicated dialogue sessions become important.

### 19. Recent World Changes Panel

Add a concise world-state digest:

- discovered locations
- new faction mentions
- standing changes
- global events

This would make the living-world systems feel more tangible.

### 20. Current Objective / Current Tension Summary

Even without hardcoded quests, the UI could surface a short current-focus line generated or derived from state:

- what the protagonist is currently trying to do
- what unresolved pressure is active

Examples:

- `Current lead: convince Lyanna to reveal the northern route`
- `Current tension: city guards are becoming suspicious`

This helps orientation during long-form sessions.

### 21. Faction and Relationship Badges

Potential compact visual language:

- faction markers
- relationship tendency markers
- ally / hostile / wary / trusted labels

Should complement, not replace, readable text.

### 22. Relationship Trend Rendering

Not just current disposition, but movement:

- improving
- worsening
- unstable
- recently changed

This would be more informative than a static number alone.

### 23. Scene-Type Rendering

Allow a scene to carry a lightweight classification:

- exploration
- dialogue
- danger
- combat-prelude
- respite
- travel
- mystery
- dream / flashback

This could drive headers, accents, and event emphasis without needing full genre-specific logic.
