# Story systems

Story systems make consequences durable while leaving prose and player intent
open-ended. They are opt-in per story and should serve the scene rather than
force every world into the same ruleset.

## Canon, turns, saves, and branches

Each accepted action becomes one atomic turn commit. Narrative messages,
structured state changes, rolls, outcomes, and generated-asset lineage share
the same branch and source commit. A failed generation cannot leave a half-turn
in canonical state.

Saves capture restorable state. Branching creates a new line of history from a
known commit, so an alternate choice does not silently modify its parent.

## Challenges and outcomes

The narrator can signal an uncertain situation; the engine resolves it using
the story's rules and recorded state. A challenge may draw on attributes,
skills, inventory, relationships, resources, deterministic dice, or a
story-pack definition.

Resolution records stakes and degree of success. Outcome policies can budget
consequences, preserve partial progress, and fail forward instead of inventing
an unrelated punishment. Rolls and result metadata remain inspectable.

## Contextual minigames

The automatic host selects a fitting mechanic from narrative tags, recent
branch history, cooldowns, difficulty, and accessibility policy. The player
does not need to choose a minigame family before acting.

Timing-free narrative families are deduction, negotiation, pattern, bidding,
courtroom, and comedy. The reusable host also implements riddle, memory,
rock-paper-scissors, and quick-time protocols. A story or difficulty profile can
require timing-free mechanics. Results are returned through the same canonical
turn bridge as ordinary actions.

## Crafting and inventory

Inventory and currency are canonical state. Recipes can be learned, validated
against requirements, and resolved atomically so consumed ingredients and the
crafted result cannot drift apart. Crafting may be deterministic or may feed a
challenge outcome when the story calls for risk.

## Investigations, projects, and fronts

- Investigations organize clues, suspects, testimony, contradictions, and case
  progress without allowing the narrator to create evidence without provenance.
- Downtime projects use persistent progress and consequences across turns.
- Fronts, hooks, fallout, factions, and pressure clocks let the world evolve
  beyond the current scene.
- Achievements and character timelines provide durable milestones rather than
  relying on transcript recall.

## Social conflict and optional combat

Relationship changes, leverage, reputation, and social duels can resolve
conflict without physical combat. When combat is enabled by the story schema,
the engine owns damage, resources, victory, and defeat state. A story pack can
set `has_combat: false` and keep every other system.

## Presentation and accessibility

Browser and terminal render the same committed state. Visual assets never
replace map topology or labels, and generated speech never replaces text.
Timing-free minigame policy, keyboard operation, readable state, and reduced
motion remain presentation requirements rather than story mechanics.

See [Extensions](extensions.md) to define story-specific schemas and challenge
pools, and [Architecture](architecture.md) for ownership boundaries.
