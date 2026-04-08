# 08-01 Summary: Rendering Contract + Narrative Semantic Renderer Foundation

**Phase:** 8 - TUI Rendering Polish
**Wave:** 1 - contract + narrative foundations
**Status:** Complete
**Commit:** not created (user did not request git)
**Requirements covered:** AI-06, TUI-10, TUI-11, TUI-12, TUI-14

---

## Changes Made

### Task 1: Extended the narrative response contract with optional rendering metadata
- `internal/engine/types.go`: added `DialogueBlock`, `EntityMention`, `EventCallout`; extended `NarrativeResponse` with `scene_type`, `dialogue_blocks`, `entities_mentioned`, `event_callouts`, and engine-only `AppliedStateChanges`; enriched `Choice` with optional semantic fields (`intent`, `risk`, `scope`, `certainty`, `related_stats`)
- `internal/ai/response.go`: mirrored the same optional renderer metadata in the AI parser contract, keeping parsing backwards-compatible with plain responses
- `internal/ai/prompts/narrator.go`: documented the new optional rendering metadata and clarified that semantic choice metadata is guidance, not a mandatory payload every turn

### Task 2: Introduced a dedicated narrative semantic renderer layer
- `internal/tui/rendering/types.go`: added `KnownEntity` and `NarrativeInput` as the renderer-facing contract
- `internal/tui/rendering/narrative.go`: added `RenderNarrativeMarkdown()` plus helpers for event callouts, dialogue blocks, trusted entity highlighting, and safe fallback behavior
- `internal/tui/views/narrative.go`: switched the main narrative loop from direct markdown rendering to the new renderer boundary

### Task 3: Implemented speaker-aware narrative rendering
- `internal/tui/rendering/narrative.go`: dialogue blocks now render speaker-aware markdown for NPC, player, narrator, and meta/system roles without over-styling the narrative body
- `internal/tui/views/narrative_rendering.go` (new): added `renderNarrativeResponse()` so the view consumes structured narrative data through a single safe entry point

### Task 4: Added trusted entity highlighting from persisted state
- `internal/tui/views/narrative_rendering.go` (new): collected trusted entities from persisted runtime state only:
  - NPCs from `ListNPCs()`
  - locations from `world_state`
  - factions and world name from `story.SettingJSON`
  - skills, titles, and inventory items from character JSON
  - chapter titles from `ListChapters()`
- Highlighting now prefers structured metadata and known state instead of freeform keyword guessing

### Task 5: Built the event callout pipeline from engine-tracked state changes
- `internal/engine/rendering.go` (new): added `StateChangesToEventCallouts()` and `MergeEventCallouts()`
- `internal/engine/narrator.go`: captured applied state changes from `ApplyStateChanges()` and merged deterministic engine-generated callouts with any AI-provided `event_callouts`
- Event callouts now surface traits, titles, skill progress, NPC encounters, location updates, inventory changes, combat start, and crafting start independently from prose

### Task 6: Codified fallback behavior for partial metadata
- `internal/tui/rendering/narrative.go`: no metadata still renders plain narrative text
- `internal/tui/views/narrative.go`: if semantic rendering returns empty content, the view falls back to the previous direct markdown path
- `internal/ai/prompts/narrator.go`: prompt now states clearly that metadata is optional, not required for gameplay

### Task 7: Added focused renderer coverage
- `internal/tui/rendering/narrative_test.go` (new): verifies event callouts, known-entity highlighting, dialogue block rendering, and plain narrative fallback
- `internal/engine/rendering_test.go` (new): verifies engine state changes are converted into stable callouts
- `internal/ai/response_test.go`: added structured metadata parsing coverage for dialogue blocks, entity mentions, event callouts, and enriched choices

---

## Verification

- `go build ./...` - passes
- `go test ./internal/tui/rendering ./internal/engine ./internal/ai ./internal/tui/views` - passes

---

## Notes

- A prompt syntax regression introduced during phase 8 was fixed in `internal/ai/prompts/narrator.go` before verification.
- This wave intentionally kept the renderer markdown-based so it integrates cleanly with the existing viewport and typewriter pipeline.
