# Achievement Rules — AI Reference

This document is loaded into the AI's context when evaluating whether an achievement should be awarded.

## When to award

Award an achievement when the player does something **genuinely noteworthy**. Not every action deserves one. Reserve for moments that feel special, unexpected, or represent significant progress.

Good triggers:
- Solving a problem in a creative or unexpected way
- Completing a major story arc or chapter
- Making a choice with meaningful consequences
- Reaching a milestone (first kill, first craft, first ally)
- Discovering something hidden or secret
- Surviving something that should have killed them
- Choosing a path that most wouldn't (mercy, betrayal, sacrifice)
- Mastering a skill or reaching a notable attribute level

Bad triggers:
- Routine actions (walking, talking, picking up items)
- Every combat victory
- Minor dialogue choices
- Things that happen automatically

## Rules

1. Maximum ONE achievement per turn
2. The achievement name must be evocative and unique to the moment
3. The description should be 1-2 sentences explaining what the player did
4. Never repeat an achievement (check existing achievements before awarding)
5. The achievement should make the player feel recognized

## Categories

- **Story**: completing chapters, narrative arcs, alternate endings, major plot points
- **Combat**: creative victories, pacifist solutions, defeating powerful enemies, surviving impossible odds
- **Social**: unlikely alliances, betrayals, maximum reputation, converting enemies to allies, memorable negotiations
- **Exploration**: secret areas, easter eggs, completing world discovery, finding hidden lore
- **Skill**: reaching mastery, unique skill combinations, first use of a new ability
- **Creative**: out-of-the-box solutions, inventive item use, unexpected crafting, breaking conventions
- **Meta**: total deaths, playtime milestones, story completions, unusual patterns (never lying, always stealing, etc.)

## Rarity

- **common** (~30%): expected milestones — "first combat win", "chapter complete"
- **uncommon** (~25%): notable moments — "survived with 1 HP", "convinced a guard"
- **rare** (~20%): impressive feats — "cleared a dungeon without fighting"
- **epic** (~15%): truly exceptional — "turned a boss into an ally", "discovered the hidden ending"
- **legendary** (~10%): once-in-a-story moments — "defeated the final boss with words alone"

## Output format

When awarding, include in your JSON response:

```json
{
  "achievement_earned": {
    "name": "Il Diplomatico",
    "description": "Hai convinto il Lupo delle Ombre a diventare tuo alleato invece di combatterlo",
    "rarity": "epic",
    "category": "social",
    "context": "Turn 147, Chapter 3 — Boss encounter resolved peacefully"
  }
}
```

Set `"achievement_earned": null` when no achievement is warranted (most turns).
