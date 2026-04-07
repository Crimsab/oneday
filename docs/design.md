# OneDay — Design Document

## Overview

OneDay is a personal TUI-based text RPG where every aspect of the game is driven by AI. Stories are infinite, characters start from nothing and grow through their actions, and the game world responds dynamically to player choices.

The core philosophy: **nothing is hardcoded**. The AI generates everything — stats, NPCs, items, locations, objectives, achievements — guided by the story's setting and rules. The game engine handles mechanics (dice rolls, damage calculation, stat checks) to keep things fair and deterministic.

---

## 1. Story Structure

A story is the top-level container. It defines the **rules of the world**, not the content. Content is generated at runtime by the AI.

### Story JSON (story.json)

```jsonc
{
  "id": "uuid",
  "name": "La Caduta di Eldrath",
  "description": "Un regno corrotto dalla magia proibita. Nessuno ricorda come è iniziato, ma tutti sanno come finirà.",
  "genre": "fantasy",
  "tone": "dark",

  "setting": {
    "world_name": "Eldrath",
    "era": "Terza Era — il Declino",
    "geography": "Continente circondato da mari velenosi. Tre regni in guerra fredda.",
    "magic_system": "La magia esiste ma ha un costo fisico. Ogni incantesimo invecchia il mago.",
    "technology_level": "Medioevo avanzato. Niente polvere da sparo.",
    "society": "Feudale. Nobili, mercanti, servi. La chiesa ha potere reale.",
    "rules": [
      "La magia ha sempre un costo — niente è gratis",
      "I morti non tornano, mai, in nessuna circostanza",
      "Il ferro puro blocca e indebolisce la magia",
      "I draghi sono estinti da 200 anni — o così si crede",
      "La menzogna è un crimine punibile con il marchio"
    ],
    "factions": [
      "La Corona di Eldrath — potere in declino, re malato",
      "La Gilda dei Tessitori — maghi organizzati, segreti",
      "I Senza Nome — ribelli, vogliono abolire la nobiltà"
    ],
    "cultures": [
      "Eldrani — orgogliosi, formali, legati alla tradizione",
      "Ghaaliti — nomadi del deserto, pragmatici, mercanti"
    ],
    "dangers": [
      "Bestie corrotte dalla magia selvaggia",
      "Banditi sulle strade tra le città",
      "La Nebbia — zone dove la magia è impazzita"
    ],
    "tone_guidelines": [
      "Le vittorie hanno sempre un costo",
      "Gli NPC non sono buoni o cattivi, hanno motivazioni",
      "La violenza ha conseguenze reali e durature",
      "L'umorismo è raro ma presente — mai comico"
    ]
  },

  "stats_schema": {
    "vitals": [
      { "key": "hp", "name": "Punti Vita", "min": 0, "starting": 20 },
      { "key": "mana", "name": "Mana", "min": 0, "starting": 5 },
      { "key": "stamina", "name": "Stamina", "min": 0, "starting": 30 }
    ],
    "attributes": [
      { "key": "str", "name": "Forza", "starting": 3 },
      { "key": "dex", "name": "Destrezza", "starting": 3 },
      { "key": "end", "name": "Resistenza", "starting": 3 },
      { "key": "int", "name": "Intelligenza", "starting": 3 },
      { "key": "wis", "name": "Saggezza", "starting": 3 },
      { "key": "per", "name": "Percezione", "starting": 3 },
      { "key": "cha", "name": "Carisma", "starting": 3 },
      { "key": "wil", "name": "Volontà", "starting": 3 },
      { "key": "lck", "name": "Fortuna", "starting": 3 }
    ],
    "secondary": [
      { "key": "reputation", "name": "Reputazione", "starting": 0 },
      { "key": "morality", "name": "Moralità", "starting": 0 }
    ],
    "currency": { "name": "Monete d'Oro", "starting": 0 },
    "has_combat": true,
    "has_crafting": true,
    "has_inventory": true
  },

  "chapter": 1,
  "total_turns": 0,
  "created_at": "2026-04-07T00:00:00Z",
  "last_played": null
}
```

The `setting` section is the **DNA of the story**. The AI reads this before every response to stay consistent. The more detailed the setting, the better the AI performs.

When creating a new story, the AI asks the player questions to build this structure collaboratively.

---

## 2. Character (Protagonist)

Characters start with almost nothing. Growth is organic and earned.

```jsonc
{
  "id": "uuid",
  "story_id": "uuid",
  "name": "Kael",
  "background": null,
  "titles": [],
  "traits": [],
  "vitals": {
    "hp": { "current": 20, "max": 20 },
    "mana": { "current": 5, "max": 5 },
    "stamina": { "current": 30, "max": 30 }
  },
  "attributes": {
    "str": 3, "dex": 3, "end": 3, "int": 3,
    "wis": 3, "per": 3, "cha": 3, "wil": 3, "lck": 3
  },
  "secondary": {
    "reputation": 0,
    "morality": 0
  },
  "skills": [],
  "abilities": [],
  "inventory": {
    "equipped": {},
    "backpack": [],
    "quest_items": []
  },
  "currency": 0,
  "relationships": {},
  "conditions": [],
  "known_recipes": [],
  "journal": [],
  "achievements": [],
  "deaths": 0
}
```

### How growth works

- **Traits**: The AI observes patterns in player choices. After repeated aggressive choices, the AI might assign `"Aggressivo"` as a trait. Traits affect how NPCs react.
- **Skills**: Learned by doing. Try to pick a lock → if you succeed (or fail interestingly), you gain XP in "Lockpicking". Skills have levels and can branch into specializations — all defined by AI based on context.
- **Attributes**: Grow slowly through use. Heavy physical actions slowly raise STR. Casting spells raises INT. The AI suggests +1 increases at natural moments.
- **Titles**: Earned through notable deeds. The AI awards them narratively. "Lo Sterminatore di Wyrm" after killing a dragon. "Il Tradito" after being betrayed.

---

## 3. NPCs

NPCs are generated by the AI when they first appear in the narrative. They are not pre-defined.

```jsonc
{
  "id": "npc_uuid",
  "story_id": "uuid",
  "name": "Lyanna Ashford",
  "role": "merchant",
  "personality": {
    "traits": ["manipolativa", "intelligente"],
    "speech_style": "Formale. Usa metafore. Chiama tutti per cognome.",
    "quirks": ["Si tocca l'anello quando mente"],
    "values": ["potere", "sopravvivenza"],
    "fears": ["essere dimenticata"]
  },
  "appearance": "Capelli corvini, cicatrice sul sopracciglio sinistro",
  "private_thoughts": [
    "Questo straniero potrebbe essere utile..."
  ],
  "notes_on_protagonist": [],
  "desires": [
    {
      "desire": "Controllare la rotta commerciale del nord",
      "priority": "high",
      "known_to_player": false
    }
  ],
  "disposition": 0,
  "is_alive": true,
  "first_appeared_turn": 12,
  "last_seen_turn": 12,
  "can_help": true
}
```

### NPC behavior

- `private_thoughts` and `desires` are **never shown to the player**. The AI uses them to maintain consistent NPC behavior.
- `notes_on_protagonist` evolve as the NPC interacts with the player.
- `disposition` (-100 to +100) tracks how the NPC feels. Affects dialogue tone, willingness to help, prices, etc.
- `can_help`: NPCs can be allies/helpers narratively (the AI describes them helping) but are not mechanically controlled like companions. No companion combat system.

---

## 4. Combat System

Turn-based, narrative-driven. Chat is always present.

### Flow

1. AI describes the encounter and sets up the enemy (stats, behavior pattern)
2. Player gets choices + free action input
3. Player acts → engine resolves mechanically:
   - Damage = weapon_base + relevant_attribute_bonus + dice_roll
   - Enemy damage = enemy_power + dice_roll - player_defense
   - Stat checks resolved by engine (dice + modifier vs difficulty)
4. AI narrates the result
5. Repeat until resolved

### Enemy intelligence

Enemies have behavior patterns the AI follows:
- **Aggressive**: attacks every turn, targets weakest point
- **Defensive**: waits for openings, counters
- **Tactical**: uses abilities strategically, retreats if losing
- **Beast**: predictable patterns, telegraphs big attacks
- **Intelligent**: adapts to player strategy, targets resources (mana drain, disarm)

The AI picks the pattern based on the enemy type and adjusts difficulty to the player's level. Early game enemies are simple and forgiving.

### Defeat (HP → 0)

What happens when you lose is **not hardcoded**. The AI decides based on context:
- **Death** → reload from last save (autosave exists)
- **Capture** → wake up in a cell, new narrative branch
- **Rescue** → an NPC intervenes, but at a cost
- **Unconscious** → wake up later, robbed, story continues
- **Retreat** → forced to flee, lose items/reputation

The consequence should match the story's tone and the narrative moment.

### Combat sessions

Combat has its own chat log (`combat_*.jsonl`), separate from the main narrative. When combat starts, the engine loads only combat-relevant context (player stats, enemy stats, inventory, active skills) into the AI prompt. This keeps the AI focused and efficient.

After combat ends, a summary is written to the main narrative chat.

---

## 5. Crafting System

No hardcoded recipes. AI-driven crafting through conversation.

### Flow

1. Player enters crafting mode (at a workbench, campfire, etc.)
2. Separate crafting chat session opens (`crafting_*.jsonl`)
3. Player describes what they want to create
4. AI evaluates:
   - Does the player have the materials?
   - Does it make sense in this world? (no laser guns in fantasy)
   - Does the player have the skill?
5. If feasible → AI creates the item with narrative description and effects
6. If not feasible → AI explains what's missing and suggests alternatives
7. Discovered recipes saved to `known_recipes` for future reference

### Items

Items have **narrative effects**, not numerical stats. The AI uses the item's description and effects to determine what the player can do.

```jsonc
{
  "id": "item_uuid",
  "name": "Piccone Rinforzato",
  "description": "Un piccone rattoppato con ingranaggi. Fa il suo lavoro, a malapena.",
  "type": "tool",
  "rarity": "common",
  "effects": ["Può rompere rocce medie", "Utilizzabile come arma improvvisata"],
  "weight": 3,
  "crafted_from": ["piccone_rotto", "ingranaggio"],
  "notes": "Craftato al capitolo 2"
}
```

The AI decides what the item can and can't do based on `effects` and `description`. This keeps things flexible — a "Piccone Rinforzato" with "utilizzabile come arma improvvisata" can be used in combat if the player asks.

---

## 6. Challenge and Randomness System

Three types of challenges, all resolved by the game engine (not the AI):

### Stat Check

Passive check against a threshold. The AI includes the check in its response:

```jsonc
{ "check": { "type": "stat", "stat": "per", "difficulty": 8 } }
```

Engine compares player's PER vs 8. Pass/fail determines which narrative branch the AI uses.

### Dice Roll

Active roll with modifiers. Creates tension — the player sees the roll happening.

```jsonc
{
  "check": {
    "type": "dice",
    "description": "Scassinare la serratura",
    "base_difficulty": 60,
    "modifiers": [
      { "source": "skill:lockpicking:2", "bonus": 15 },
      { "source": "item:grimaldello", "bonus": 10 }
    ]
  }
}
```

Engine: roll d100, add modifiers, compare vs difficulty. Result shown with animation in TUI.

### Mini-games

- **Rock-Paper-Scissors**: for duels of wit, gambling, negotiations
- **Memory sequence**: for spells, rituals, puzzles
- **Quick-time**: press key within N seconds (reactions, dodging)
- **Riddles**: AI generates contextual riddles

Mini-games are optional per story. A slice-of-life story won't have combat dice rolls but might have card games with friends.

---

## 7. Achievement System

No predefined achievements. The AI recognizes noteworthy moments.

The AI reads `docs/achievement_rules.md` as part of its context. This document defines **categories and criteria**, not specific achievements.

When the AI determines something is achievement-worthy, it includes in its output:

```jsonc
{
  "achievement_earned": {
    "name": "Il Diplomatico",
    "description": "Hai convinto un boss a diventare tuo alleato",
    "rarity": "epic",
    "category": "social",
    "context": "Boss: Lupo delle Ombre → alleato"
  }
}
```

Achievements are per-story and displayed with a notification in the TUI.

---

## 8. Session and Chat Management

### Hierarchy

```
Story
└── Session (one per play sitting)
    ├── Main chat (narrative)
    ├── Combat chats (0..N per session)
    ├── Crafting chats (0..N per session)
    └── Dialogue chats (0..N per session, deep NPC conversations)
```

A **session** is created automatically when the player opens a story. It records start time, end time, which chapter it covers, and a summary.

A **chapter** is an AI-determined narrative arc. The AI decides when a chapter ends (major event, objective complete, location change) and generates a summary. Chapters are used for:
- Organizing the narrative
- RAG summarization boundaries
- Achievement context

### Chat entry format (JSONL)

```jsonc
{
  "turn": 42,
  "timestamp": "2026-04-07T15:30:00Z",
  "chapter": 2,
  "location": "Ghaal marketplace",
  "input": {
    "type": "free_action",
    "text": "Guardo Lyanna negli occhi e le chiedo della Fonte"
  },
  "output": {
    "narrative": "Lyanna alza lo sguardo dal calice...",
    "choices": [
      "Insisti con tono minaccioso",
      "Cambia argomento",
      "Mostra la lettera"
    ],
    "state_changes": {},
    "mood": "tense",
    "ascii_art": null,
    "achievement_earned": null
  },
  "ai_model": "claude-sonnet-4-6",
  "ai_latency_ms": 1847
}
```

---

## 9. RAG (Retrieval-Augmented Generation)

### Purpose

Stories can be infinite. Context windows are not. RAG bridges the gap by retrieving only the relevant parts of history.

### Flow

```
Every N turns (configurable, default 10):
  1. AI summarizes the last N turns → "summary chunk"
  2. Generate embedding via text-embedding-3-small (through LiteLLM)
  3. Store embedding + text in sqlite-vec

When building AI context:
  1. Take current situation as query
  2. Vector search → top K relevant chunks
  3. Inject into prompt alongside recent chat history
```

The full chat is ALWAYS saved (never lost). RAG is only used to select what goes into the AI's context window.

---

## 10. ASCII Art

Strategic use only, when it serves the narrative:

- Entering a major new area
- First appearance of a significant boss/creature
- Finding a legendary item
- Epic story moments
- World map / dungeon layout

ASCII art is generated inline by the AI as part of its narrative response. No pre-generated files, no image conversion. The AI creates simple but evocative ASCII art using standard characters + Lipgloss styling (colors, borders).

---

## 11. TUI Layout

### Main narrative view

```
┌─ OneDay ──────────────────────────────────────────────┐
│                                                        │
│  [Chapter 3 — La Verità sulla Fonte]                  │
│  [Ghaal, Il Mercato delle Ombre]                      │
│                                                        │
│  Lyanna alza lo sguardo dal calice di vino. I suoi    │
│  occhi, freddi come sempre, tradiscono qualcosa di    │
│  nuovo. Paura, forse. O calcolo.                      │
│                                                        │
│  "Vuoi sapere della Fonte?" sussurra, avvicinandosi.  │
│  "Allora siediti. E ordina qualcosa. Sarà una lunga   │
│  notte."                                              │
│                                                        │
│  ─────────────────────────────────────────────────     │
│                                                        │
│  1. Insisti con tono minaccioso                       │
│  2. Cambia argomento con nonchalance                  │
│  3. Mostra la lettera che hai trovato                 │
│  > Scrivi la tua azione...                            │
│                                                        │
├────────────────────────────────────────────────────────┤
│ HP: 62/100 │ Mana: 15/50 │ claude-sonnet-4-6 · 1.2s  │
└────────────────────────────────────────────────────────┘
```

### Status bar shows

- Vitals (from story's stats schema)
- Current AI model
- Last response latency

---

## 12. Chat Commands

The player can type special commands during gameplay. Commands start with `/`.

### Core Commands

| Command | Short | Description |
|---------|-------|-------------|
| `/inventory` | `/i` | Show inventory |
| `/stats` | `/s` | Show character sheet (stats, attributes, skills, titles, traits) |
| `/map` | `/m` | Show discovered world map |
| `/journal` | `/j` | Show journal (auto + manual entries) |
| `/achievements` | `/a` | Show unlocked achievements |
| `/save` | | Manual save |
| `/load` | | Load a save |
| `/settings` | | Open settings |
| `/help` | `/h` | List all commands |
| `/quit` | `/q` | Save and quit |

### The Narrator Command — `/n`

`/n` (or `/narrator`) lets the player **speak directly to the AI as a narrator/game master**, outside of the story. This is a meta-level command that adds depth without breaking immersion.

#### What `/n` can do

**Inject lore and world-building:**
```
/n Aggiungi una fazione segreta: l'Ordine della Cenere,
   cultisti che credono che il mondo debba essere purificato
   dal fuoco. Operano nell'ombra da secoli.
```
→ The AI acknowledges, updates `story.json` (adds to factions/cultures/dangers), and weaves it into the world. The next time factions are relevant, l'Ordine della Cenere might appear organically.

**Reveal hidden NPC layers:**
```
/n Lyanna ha un figlio segreto che nessuno conosce.
   Questo è il vero motivo per cui vuole il potere.
```
→ AI updates Lyanna's `desires` and `private_thoughts`. Her behavior subtly shifts. The player might discover this in-story later, or it might simply color her actions.

**Steer the narrative:**
```
/n Vorrei che la prossima area che esploro sia una
   civiltà sotterranea dimenticata con la propria cultura.
```
→ AI takes note and introduces it at the next natural opportunity. Not forced, but guided.

**Ask the narrator questions:**
```
/n Quanto è pericoloso il territorio a nord?
/n Ci sono altre fazioni che non ho ancora incontrato?
```
→ AI answers from the narrator's perspective (may be vague or cryptic depending on the story's tone).

**Correct or adjust:**
```
/n L'ultimo NPC era troppo amichevole. Rendi il mondo
   più ostile e diffidente verso gli stranieri.
```
→ AI adjusts the tone and NPC generation going forward.

#### How it works technically

1. `/n` input is tagged as `type: "narrator"` in the chat log
2. The AI receives it as a system-level instruction, not as player action
3. State changes from `/n` are applied to `story.json`, `npcs/*.json`, `world_state.json` etc.
4. The narrative response acknowledges the change subtly (or explicitly, depending on what was asked)
5. Narrator interactions are logged but don't count as story turns

#### What `/n` is NOT

- Not a cheat command (can't give yourself items or stats)
- Not a rewrite (can't undo past events, use `/undo` for that)
- It's a **collaboration tool** between you and the AI narrator

### Dynamic World Updates

Whether through `/n` or through organic gameplay, the world evolves. When the player discovers something new:

- **New faction found** → added to `story.json` setting.factions
- **New culture encountered** → added to setting.cultures
- **New location discovered** → added to `world_state.json`
- **NPC secret revealed** → updated in `npcs/{id}.json` (desires, thoughts)
- **New world rule learned** → added to setting.rules
- **New danger identified** → added to setting.dangers

The AI updates these files automatically as part of its `state_changes` output. The story.json is a **living document** that grows richer as you play.

---

## 13. Plugin System (v2+)

### Phase 1: JSON/YAML mods

Custom stories with detailed settings, custom achievement rules.

### Phase 2: Lua scripting

Plugins can hook into game events:

```lua
-- on_combat_start: modify enemy stats based on player level
function on_combat_start(combat)
    if combat.enemy.type == "boss" and player.level < 5 then
        combat.enemy.hp = combat.enemy.hp * 0.7
    end
end
```

Hooks: `on_story_start`, `on_chapter_end`, `on_combat_start`, `on_combat_end`, `on_npc_interact`, `on_item_craft`, `on_achievement_check`, `on_dice_roll`, `on_player_death`, `on_save`, `on_load`

Using `gopher-lua` (pure Go Lua VM).
