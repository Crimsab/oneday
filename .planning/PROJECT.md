# OneDay

## What This Is

A personal AI-driven text RPG played entirely in the terminal (TUI). Stories are infinite, AI-generated, and deeply personalized. Every NPC has personality, desires, and opinions about you. Every choice matters. Nothing is hardcoded — stats, skills, items, locations, objectives, achievements are all generated at runtime by AI.

## Core Value

The player can start a story and have an engaging, coherent, infinite narrative experience driven by AI — where every action matters and the world responds dynamically.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] AI-powered narrative loop with free action always available
- [ ] Dynamic story creation through AI conversation
- [ ] Character system starting from nothing, growing through play
- [ ] Living NPCs with personality, private thoughts, desires
- [ ] Turn-based combat with narrative and creative solutions
- [ ] AI-driven crafting (no hardcoded recipes)
- [ ] Challenge system (dice rolls, stat checks, mini-games)
- [ ] AI-driven achievement system (no predefined list)
- [ ] Persistent chat history with RAG for long-term memory
- [ ] /narrator command for meta-level world-building
- [ ] Chat commands (/inventory, /stats, /map, /journal, etc.)
- [ ] Dynamic world updates (story.json evolves as you play)

### Out of Scope

- Multiplayer — personal single-player experience only
- Web UI — TUI only, no browser interface
- Pre-generated content — everything is AI-generated at runtime
- Hardcoded recipes/achievements/stats — all dynamic
- Psyche integration — removed from scope
- Companion combat system — NPCs can help narratively, not mechanically

## Context

- Built for personal use on Windows (primary) and Linux
- AI backend: LiteLLM proxy → OpenRouter → Claude Code CLI (configurable fallback chain)
- Runs on homelab (Debian 13, AMD Ryzen 7) for development, Windows for playing
- Inspired by psychology discussions about personal growth through narrative
- Embedding: text-embedding-3-small generates vectors, SQLite stores them as BLOBs, retrieval uses cosine similarity in Go
- The game engine handles mechanics (dice, damage, checks); the AI handles narrative

## Constraints

- **Stack**: Go + Bubbletea/Bubbles/Lipgloss + SQLite + embedding BLOB RAG
- **AI**: Must work with multiple providers (LiteLLM, OpenRouter, Claude Code) via fallback chain
- **No hardcoding**: Stats, skills, NPCs, items, locations, objectives, achievements all AI-generated
- **Modularity**: Every system is a separate package with clean interfaces
- **Cross-platform**: Must compile for Windows (primary target) and Linux

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go over Python/Rust | Best TUI ecosystem (Bubbletea), good performance, easy cross-compile | ✓ Good |
| SQLite BLOB vectors over external vector DBs | Zero infrastructure, single file, simple with `modernc.org/sqlite`, fast enough for personal use | ✓ Good |
| No stat templates | Each story defines its own stats schema — more flexible, more creative | ✓ Good |
| Items have narrative effects not numerical stats | AI uses description/effects to determine capabilities, more flexible | ✓ Good |
| LiteLLM primary, OpenRouter fallback, Claude Code optional | Best balance of routing flexibility, local homelab integration, and fallback safety | ✓ Good |
| Lua for future plugin system | Standard for game modding (WoW, Factorio), gopher-lua is pure Go | — Pending |
| Separate chat sessions for combat/crafting | Keeps AI context focused and efficient | ✓ Good |
| /narrator command for meta world-building | Adds depth without breaking immersion, evolves story.json dynamically | ✓ Good |
| GSD workflow for development | Prevents context rot in long AI-assisted sessions, wave execution | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition:**
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone:**
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-07 after initialization*
