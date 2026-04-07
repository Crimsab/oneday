---
phase: 4
plan: 4.2
title: "Enhanced character sheet, inventory views, and NPC context injection"
status: completed
commit: 8cb8a56
tests: 10 new unit tests, 58 total passing
---

# Plan 4.2 Summary

## What Was Built

### Task 1 — /stats rewrite (`internal/tui/views/narrative.go`)
- **Vitals** rendered with `█/░` progress bars: `HP: ████████░░  62/100`
- **Attributes** use schema labels in schema-defined order (3 per row), falling back to uppercase key if schema missing
- **Secondary stats** use schema labels with schema-defined order
- **Skills** show level + XP bar + numbers: `Lockpicking  Lv.3  ████████░░  245/300 XP`
- **Traits** merged from `char.TraitsJSON` and `stats["traits"]` (deduplicated, case-insensitive)
- Empty states: `(none yet — try new things to learn!)` / `(none yet — great deeds earn great titles)`
- **NPC Relationships** loaded from DB via `ListNPCs`, shown with disposition bar and word label (Hostile/Unfriendly/Neutral/Friendly/Allied)
- Deaths counter shown in red when > 0
- All section headers styled with `theme.Title`, empty states with `theme.MutedText`

### Task 2 — /inventory rewrite (`internal/tui/views/narrative.go`)
- Type icons: `⚔` weapon, `◇` armor, `⚒` tool, `◈` consumable, `◆` quest/key_item, `•` misc/string
- Equipped section shows slot label (`Main Hand: ⚔ Iron Sword`) when slot info present
- Backpack section header shows item count; descriptions shown indented in muted text; weight total if any items carry weight field
- Currency name comes from stats schema
- Completely empty inventory (0 items, 0 currency) shows single friendly line: `You carry nothing. The journey begins light.`
- Backward-compatible: plain string items still display correctly

### Task 3+4 — NPC context injection (`internal/engine/context.go`, `narrator.go`)
- `BuildContext` signature changed: takes `[]storage.NPC` instead of `npcsContext string`; builds the context string internally using `FormatNPCForContext`
- `ContextConfig` gains `NPCLookbackTurns int` (default 20)
- `buildStateSummary` now includes `- Known NPCs: N (Name1, Name2)` line
- `sendTurn` loads `ListRecentNPCs` with the configured lookback before every AI call; NPC load failure is non-fatal (nil slice used)
- Redundant `buildNPCContext` method removed from narrator

### Task 5 — Auto-update NPC last_seen_turn (`internal/engine/npc.go`)
- `UpdateNPCLastSeen(db, storyID, narrativeText, currentTurn)` scans all story NPCs
- Matches full name and first-name-only (case-insensitive substring)
- Called after every AI response in `sendTurn` — best-effort, errors swallowed

### Task 6 — Unit tests (`internal/engine/state_test.go`)
10 tests, all passing:
- `TestApplyStateChanges_TraitAdd` — trait added, TraitsJSON synced
- `TestApplyStateChanges_DuplicateTrait` — second add produces 0 changes
- `TestApplyStateChanges_DuplicateTrait_CaseInsensitive` — "Bold" vs "bold" deduped
- `TestApplyStateChanges_TitleAdd` — title added to stats["titles"]
- `TestApplyStateChanges_TitleAdd_NoDuplicate` — duplicate title skipped
- `TestApplyStateChanges_SkillLearn` — skill created at level=1 xp=0, SkillsJSON synced
- `TestApplyStateChanges_SkillXP` — XP gained, no level-up
- `TestApplyStateChanges_SkillLevelUp` — 100 XP → level 2, 0 XP remaining
- `TestApplyStateChanges_SkillLevelUp_MultiLevel` — 350 XP → level 3, 50 XP remaining
- `TestApplyStateChanges_SkillXP_AutoLearn` — unknown skill auto-created on XP grant

## Requirements Addressed
- CHAR-06: Character sheet displays traits, skills with progression, titles
- TUI-06: /stats overlay with full character info and visual bars
- TUI-07: /inventory with grouped sections and type icons
- NPC-05: NPC context injected into AI prompts each turn
- NPC-06: last_seen_turn updated when NPC mentioned in narrative
