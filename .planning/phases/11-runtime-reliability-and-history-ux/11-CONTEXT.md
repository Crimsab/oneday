# Phase 11 Context: Runtime Reliability and History UX

## Why This Phase Exists

Phase 10 improved ambient presentation and inspectability, but a real playtest surfaced the next layer of issues:

- live typewriter playback can reveal broken ANSI escape sequences because styled text is animated rune-by-rune
- choices can feel ahead of the visible narrative because the view is replaying accumulated history instead of only the new scene segment
- story bootstrap can still fail hard when the model returns structurally incomplete story JSON
- challenge/minigame transitions can appear abrupt or “self-starting”
- viewport scrolling, choice navigation, and footer density still feel rough in actual play
- players still need a real way to review/search prior interaction history inside the story

## User Direction

- include all playtest feedback in planning, not as loose TODOs
- prioritize `New Story` reliability first, then the runtime narrative UI
- keep choices in the worldbuilding/new-story flow and the live narrative flow
- improve relationship/dialogue formatting, challenge affordances, save feedback, and history review
- make zone/city/location names more visible with a practical, robust method
- support better keyboard-first interaction and saner scroll behavior
- use validators/retries for unreliable AI responses where it makes sense

## Scope

In scope:

- ANSI-safe typewriter behavior and scene-reveal timing fixes
- story-definition validation plus retry/repair for bootstrap failures
- challenge/minigame confirmation/prelude for active events
- clear separation between choice navigation and viewport scrolling
- stronger, role-aware rendering for trusted entities, dialogue, and relationship/system callouts
- searchable `/history` review flow for current-session interactions
- clearer quick-save/status feedback and narrow-terminal footer behavior

Out of scope:

- redesigning the entire combat/crafting system
- replacing the main narrative architecture with a different renderer stack
- full mouse-driven hover UX for stats/badges
- large schema changes to saved game data beyond what is needed for runtime polish
