---
phase: 4
plan: 4.1
title: "Character Growth Engine and NPC Generation/Persistence"
status: completed
commit: 2cda4f1
---

# Summary: Plan 4.1 — Character Growth Engine and NPC Generation/Persistence

## What Was Built

All 8 tasks completed. Build passes cleanly (`go build ./...`, `go vet ./...`, exit 0).

### Files Created

- **`internal/storage/npcs.go`** — NPC CRUD layer: `CreateNPC`, `GetNPC`, `GetNPCByName` (case-insensitive, nil/nil for not found), `ListNPCs`, `ListRecentNPCs` (last N turns window), `UpdateNPC`, `UpdateNPCDisposition`
- **`internal/engine/npc.go`** — NPC helper types and functions:
  - `NPCData`, `NPCPersonality`, `NPCDesire` — structured AI-generated NPC types
  - `ParseNPCData` — safe conversion from untyped AI map (no panics on missing fields)
  - `NPCToStorage` — converts NPCData + metadata to `storage.NPC` with new UUID
  - `FormatNPCForContext` — full AI-visible representation (private thoughts, desires, notes)
  - `FormatNPCForPlayer` — player-visible summary (no private info, only known desires)

### Files Modified

- **`internal/storage/models.go`** — Added `LastSeenTurn`, `Appearance`, `NotesOnProtagonist` fields to `NPC` struct
- **`internal/storage/migrations.go`** — Added migration v3: `ALTER TABLE npcs` to add `appearance`, `notes_on_protagonist`, `last_seen_turn` columns
- **`internal/storage/stories.go`** — Added `UpdateCharacterFull`: updates stats_json, traits_json, skills_json, inventory_json, known_recipes_json atomically
- **`internal/engine/state.go`** — Rewrote `ApplyStateChanges`:
  - New signature: `(changes, char, world, db, storyID, currentTurn)`
  - Added `trait_add`: deduplicates (case-insensitive), syncs `char.TraitsJSON`
  - Added `title_add`: deduplicates, appends to stats titles array
  - Added `skill_learn`: creates skill at level 1, 0 XP if new
  - Added `skill_xp`: adds XP, auto-levels at `level*100` threshold, syncs `char.SkillsJSON`
  - Added `new_npc`: parses via `ParseNPCData`, checks for existing NPC by name, creates or updates `last_seen_turn`
  - Added `npc_disposition`: adjusts or sets disposition, clamped to [-100, 100]
  - Added `npc_thoughts`: appends to NPC private_thoughts JSON array
  - Added `npc_notes`: appends to NPC notes_on_protagonist JSON array
  - Added helpers: `toInterfaceSlice`, `toSkillsMap`
- **`internal/ai/prompts/narrator.go`** — Updated `NarratorSystem`:
  - New `npcsContext string` parameter; injects "## Known NPCs" section when non-empty
  - Added `import "strings"` for TrimSpace
  - Full `## State Changes` documentation section covering all growth and NPC keys
  - Character growth rules (frequency, thresholds) and NPC generation rules
- **`internal/engine/narrator.go`** — Wired everything together:
  - `sendTurn` now captures `currentTurn`, builds NPC context via `buildNPCContext`
  - `ApplyStateChanges` called with `n.db`, `n.story.ID`, `currentTurn`
  - `UpdateCharacterStats` replaced with `UpdateCharacterFull`
  - `BuildContext` called with `npcsContext` parameter
  - Added `buildNPCContext(currentTurn int) string`: loads NPCs seen in last 20 turns, formats via `FormatNPCForContext`
- **`internal/engine/context.go`** — `BuildContext` signature updated to accept `npcsContext string`; passed through to `NarratorSystem`

## Design Decisions

- `ApplyStateChanges` is the single entry point for all game state mutations — NPC ops were added here rather than as a separate function to keep mutations cohesive
- NPC private_thoughts and desires are injected into the AI prompt but never exposed in `FormatNPCForPlayer`
- Skill leveling uses `level * 100` XP threshold (level 1 → level 2 at 100 XP, level 2 → level 3 at 200 XP, etc.)
- `GetNPCByName` filters `is_alive = 1` so dead NPCs are not reused
- All NPC ops are non-fatal: errors are skipped to avoid failing the turn for a non-critical side effect
- NPC context window defaults to 20 turns to keep AI prompt size bounded
