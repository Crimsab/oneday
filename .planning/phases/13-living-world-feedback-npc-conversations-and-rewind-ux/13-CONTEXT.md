# Phase 13 Context: Living world feedback, NPC conversations, and rewind UX

## Objective

Push OneDay toward a more legible and emotionally reactive narrative RPG without turning it into a railroad. This phase should make consequences easier to track, NPC interactions deeper, and alternate-choice exploration safer.

## Included Scope

- A `What changed this turn?` surface with structured deltas
- Automatic hook tracking for promises, debts, mysteries, timers, and unresolved threads
- Richer NPC relationship state beyond a single `disposition`
- A smart nearby-NPC conversation flow (`/talk` or equivalent) that is context-aware
- Fail-forward consequence design so failure changes the world instead of hard-blocking play
- World reaction feed for rumors, heat, faction stance, notoriety, and delayed fallout
- Branch save / rewind support around pivotal moments
- Downtime scenes with clear narrative purpose
- Practical crafting QoL such as “what can I make now?” and recipe/material guidance

## Explicit Exclusions For Now

- Chapter micro-objectives
  Rationale: they risk making the story feel guided or prescriptive before we know how to keep them optional and non-railroading.
- `compact mode`
  Rationale: low current value relative to higher-impact narrative systems.
- `pin scene`
  Rationale: weak payoff right now; not enough evidence that players need it.

## Design Direction

### Guidance Without Railroading

The game should surface context, consequences, and open hooks, but it should not tell the player what they “should” do next in a hard quest-log style. Hook tracking should be descriptive first, suggestive second.

### NPC Conversation Model

Conversation should be centered on NPCs who are nearby, recently relevant, or strongly connected to the current scene. The player interaction layer should express intent such as asking, probing, bonding, threatening, bargaining, lying, promising, or confessing. Relationship axes should evolve from those intents and from narrative outcomes.

### World Reactivity

The world should feel alive through deferred consequences: rumors spreading, faction temperatures shifting, local heat rising, favors/debts accumulating, and people reacting differently over time. The player should feel the world “remembering” what happened.

### Rewind Safety

Rewind should prefer branch-oriented saves over destructive overwrite semantics, so players can explore forks without feeling they are breaking the run or losing continuity.

## Open Planning Questions

1. How much of the turn-delta and hook-tracker data should be AI-authored versus engine-derived?
2. Should NPC conversation live as a dedicated mode, a slash command, or a choice-picker hybrid?
3. How should rewind branches be represented in the save UI so they remain understandable in a terminal?
4. Which parts of fail-forward should be hard simulation rules versus narrator guidance?
