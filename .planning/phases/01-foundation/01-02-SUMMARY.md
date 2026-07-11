---
phase: 1
plan: 1.2
title: "SQLite storage layer with migrations and core models"
status: completed
completed_at: "2026-04-07"
---

# Plan 1.2 Summary: SQLite Storage Layer

## What Was Built

Implemented the full SQLite storage layer for OneDay using `modernc.org/sqlite` (pure Go, no CGO).

## Files Created / Modified

| File | Action | Description |
|------|--------|-------------|
| `go.mod` | Modified | Added `modernc.org/sqlite v1.48.1` and transitive deps; upgraded to go 1.25.0 |
| `go.sum` | Modified | Updated checksums |
| `internal/storage/db.go` | Created | DB struct, Open/Close/Conn, WAL + FK PRAGMAs, migration runner |
| `internal/storage/migrations.go` | Created | Schema version tracking + V1 migration with all tables |
| `internal/storage/models.go` | Created | Go structs for all 8 entities |
| `internal/storage/db_test.go` | Created | 8 tests covering tables, idempotency, WAL, FK, constraints |

## Schema Tables Created (V1 Migration)

- `stories` — core story metadata, setting JSON, stats schema JSON
- `characters` — player protagonists with flexible JSON columns for stats, traits, skills, inventory, known_recipes
- `npcs` — AI-generated NPCs with personality JSON, private_thoughts, desires, disposition
- `world_state` — per-story global state (location, factions, events, chapter/turn tracking)
- `sessions` — play session records with start/end timestamps
- `chat_messages` — full message log with role CHECK constraint and message_type CHECK constraint
- `chapters` — AI-determined narrative arc summaries
- `achievements` — earned achievements with category and rarity
- `schema_version` — migration tracking table

## Design Decisions

- **JSON columns** for flexible AI-generated content (stats, traits, personality, etc.) — no rigid schema for story-defined attributes
- **WAL mode** enabled for concurrent reads during AI streaming
- **Foreign keys** enforced with `ON DELETE CASCADE` so deleting a story cleans up all related data
- **CHECK constraints** on `chat_messages.role` and `message_type` for data integrity
- **Indexes** on all `story_id`, `session_id` foreign keys for query performance

## Test Results

All 8 tests pass:
- `TestOpenCreatesTables` — verifies all 9 tables exist after Open
- `TestOpenIdempotent` — migrations run exactly once on repeated Open
- `TestForeignKeysEnabled` — PRAGMA foreign_keys=1 confirmed
- `TestWALMode` — PRAGMA journal_mode=wal confirmed
- `TestInsertAndQueryStory` — basic CRUD round-trip
- `TestForeignKeyConstraint` — FK violation correctly rejected
- `TestChatMessageRoleConstraint` — CHECK constraint correctly enforced
- `TestOpenCreatesDirectory` — nested directories created automatically

## Verification

```
go build ./...   ✓
go vet ./...     ✓
go test ./internal/storage/ -v   8/8 passed
```
