# OneDay

## What This Is

A personal AI-driven interactive-story engine playable through both a terminal TUI and a browser. Stories are open-ended, AI-generated, and deeply personalized, while a deterministic engine owns canonical state, outcomes, persistence, branching, and cross-surface contracts. NPCs have evolving identities, forms, knowledge, motivations, and relationships; generated content remains constrained by durable world truth rather than prompt memory alone.

## Core Value

The player can start a story and have an engaging, coherent, infinite narrative experience driven by AI — where every action matters and the world responds dynamically.

## Requirements

### Validated

- [x] Canonical transactional turn commit with SQLite-backed history and state
- [x] Terminal and browser play surfaces backed by the same story engine
- [x] Persistent NPC/world systems, RAG memory, chapters, saves, challenges, and visual assets
- [x] Provider-flexible AI setup with Codex OAuth, LiteLLM, OpenRouter, and local/remote embeddings

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
- [ ] Immutable branching timeline with safe rewind and alternative navigation
- [ ] Universal entity/identity/form/fact model for disguise, transformation, possession, and contradictions
- [ ] Canonical locations, world clock, weather, events, and spatial graph
- [ ] Engine-owned graded outcomes and portable minigames shared by browser and TUI
- [ ] Per-generation causal telemetry, branch-aware media, and configurable TTS

### Out of Scope

- Multiplayer — personal single-player experience only
- Pre-generated content — everything is AI-generated at runtime
- Hardcoded recipes/achievements/stats — all dynamic
- Psyche integration — removed from scope
- Companion combat system — NPCs can help narratively, not mechanically
- Multiplayer/cloud sync — defer until single-player branch and canonical-state invariants are proven
- PostgreSQL/microservices — SQLite and the current modular monolith remain the right deployment model
- Voice cloning — rights, consent, and operational complexity are outside v2.0

## Current Milestone: v2.0 Universal Canon & Multimodal Story Engine

**Goal:** Turn OneDay's strong transactional narrative kernel into a universal, branch-safe story engine whose identities, world facts, outcomes, media, and diagnostics remain canonical across TUI and browser play.

**Target features:**

- Versioned, validated snapshots and immutable branch/commit lineage
- Canonical entity, identity, form, fact, location, clock, weather, and event models
- Engine-owned graded outcomes plus a portable, extensible minigame protocol
- Browser/TUI parity, branch navigation, readable history/export, and player-safe projections
- Causal AI telemetry, branch-aware visual assets, configurable TTS, and advanced authoring/export tooling

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
| Preserve the Go/Rust kernel | The audit found the canonical turn transaction, locking, idempotency, discovery, anti-loop, and asset queue worth extending rather than replacing | ✓ Required for v2.0 |
| Engine resolves outcomes before narration | Prevents the model from retroactively deciding success and makes difficulty/fairness portable across genres and surfaces | — v2.0 |
| Derived artifacts carry branch/source lineage | Prevents future images, RAG, summaries, and audio from contaminating rewound branches | — v2.0 |
| P3 platform expansion deferred | Cloud sync, multiplayer, PostgreSQL, marketplace, fine-tuning, and voice cloning wait for canonical/eval gates | ✓ Scoped out |

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
*Last updated: 2026-07-11 for milestone v2.0 architecture-audit implementation*
