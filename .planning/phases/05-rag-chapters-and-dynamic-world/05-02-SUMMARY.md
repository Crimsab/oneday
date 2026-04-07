---
phase: 5
plan: 5.2
status: complete
---

# Summary: Plan 5.2 — Chapter System, Separate Chat Sessions, and /narrator Command

## What Was Built

### Task 1: Chapter Storage CRUD (`internal/storage/chapters.go`)
- `CreateChapter`, `GetChapter`, `ListChapters`, `UpdateChapterEnd`, `GetCurrentChapter`, `UpdateChapterTitle`
- chapters table already existed in migrationV1 — no new migration needed

### Task 7: Chapter Summary Prompt (`internal/ai/prompts/chapter_summary.go`)
- `ChapterSummarySystem` constant — instructs AI to generate JSON `{title, summary}`
- `ChapterSummaryUser(transcript)` helper to build the user message

### Task 4: Sub-session JSONL Files (`internal/engine/session.go`)
- Added `subFiles map[string]*os.File` and `subCounters map[string]int` to `GameSession`
- `OpenSubSession(sessionType)` — creates `combat_1.jsonl`, `crafting_1.jsonl`, etc.
- `AppendSubTurn(subSessionID, entry)` — writes to specific sub-session file
- `CloseSubSession(subSessionID)` — closes specific sub-session file
- `Close()` updated to close all open sub-session files
- `ChatEntry` gets `MessageType` field; `AppendTurn` now passes it through to DB

### Task 2: Chapter Lifecycle Engine (`internal/engine/chapters.go`)
- `ChapterManager` struct with `NewChapterManager`, `EnsureCurrentChapter`, `HandleChapterEnd`, `GetChapterSummaries`
- `HandleChapterEnd`: fetches chapter messages, calls AI for summary, stores in RAG with chunk_type "chapter", closes chapter in DB, opens next chapter
- `RAG.StoreChunk` added to `internal/rag/rag.go` for direct chunk embedding

### Task 5: /narrator Command
- **commands.go**: registered `"narrator"` and `"n"` aliases; also added `"journal"` / `"j"`
- **narrator_cmd.go**: `NarratorCommand` struct with `Execute(ctx, input)` — builds NarratorMeta prompt, calls AI, applies extended state changes, logs to DB as `message_type="narrator"` (no turn increment)
- **narrator_meta.go**: `NarratorMetaSystem(...)` — meta-level GM prompt accepting lore injections, NPC modifications, world-building. JSON response format: `{message, state_changes}`
- **narrative.go (TUI)**: `narratorMetaResponseMsg` type, `sendNarratorCommand`, `showJournal`, updated `handleCommand` with "narrator" and "journal" cases, updated help text

### Task 6: State Applicator Extensions (`internal/engine/state.go`)
New change types in `ApplyNarratorStateChanges`:
- `setting_factions_add`, `setting_cultures_add`, `setting_dangers_add`, `setting_rules_add`, `setting_tone_add` — append to story.json setting arrays
- `world_location_add`, `world_event_add` — append to world state JSON arrays
- `world_faction_standing` — update faction standings map
- `npc_desires` — update NPC desires field
- `UpdateStorySetting` added to `internal/storage/stories.go`
- Narrator changes embed as RAG chunk type "narrator" for long-term memory

### Task 3: Narrator Integration (`internal/engine/narrator.go`)
- `Narrator` struct gets `chapters *ChapterManager` and `narratorCmd *NarratorCommand` fields
- `SetRAG` lazily initializes both; `EnsureCurrentChapter(0)` called on first RAG wire
- `sendTurn`: after state changes, checks `narrative.ChapterEnd` — calls `HandleChapterEnd`, increments `world.CurrentChapter`
- `ExecuteNarratorCommand(ctx, input)` — delegates to `narratorCmd.Execute`
- `GetChapterSummaries()` — delegates to `chapters.GetChapterSummaries`

### System Prompt Update (`internal/ai/prompts/narrator.go`)
- Added "Chapter Management" section instructing AI when to set `chapter_end: true` and `chapter_title`
- Added `chapter_end` and `chapter_title` to the JSON response format example

### NarrativeResponse (`internal/engine/types.go`)
- Added `ChapterEnd bool` and `ChapterTitle string` fields

## Verification
- `go build ./...` — success
- `go vet ./...` — no issues
- `go test ./...` — 76 passed in 13 packages
