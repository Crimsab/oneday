---
phase: 3
plan: 3.1
status: complete
commit: see git log
---

# Summary: Plan 3.1 — Session Management, JSONL Chat Persistence, and Context Builder

## What Was Built

### Task 1: `internal/storage/sessions.go`
Five methods on `*DB`: `CreateSession`, `GetSession`, `GetActiveSession`, `CloseSession`, `ListSessions`. `GetActiveSession` queries `ended_at IS NULL ORDER BY started_at DESC LIMIT 1` to find the most recent open session.

### Task 2: `internal/storage/chat.go`
Four methods: `AppendChatMessage`, `GetRecentMessages`, `GetSessionMessages`, `CountSessionMessages`. `GetRecentMessages` fetches last N in DESC order then reverses the slice to return chronological order.

### Task 3: `internal/engine/session.go`
`GameSession` struct manages session lifecycle:
- `NewGameSession`: checks for active session in DB (resume) or creates a new one; makes directory `{dataDir}/stories/{storyID}/sessions/{sessionID}/`; opens `main.jsonl` in append mode; counts existing lines to restore turn counter on resume.
- `AppendTurn`: marshals `ChatEntry` to JSONL line, then splits into user+assistant `ChatMessage` rows in DB.
- `Close`: closes the file handle and calls `CloseSession` on the DB.

### Task 4: `internal/engine/context.go`
`BuildContext(story, char, world, recentMessages, ragChunks, currentInput) []ai.Message`:
1. System prompt via `prompts.NarratorSystem`
2. Live state summary (chapter, turn, location, stats, traits)
3. RAG slot (injects `[]string` as a system message when non-empty; no-op when nil)
4. Recent DB messages mapped to `ai.RoleUser`/`ai.RoleAssistant`
5. Current user input as final user message

### Task 5: `internal/engine/narrator.go` (refactored)
- Removed in-memory `messages []ai.Message` field entirely
- Added `session *GameSession` and `contextCfg ContextConfig` fields
- `NewNarrator` now accepts `*GameSession` and `ContextConfig`
- `sendTurn` rebuilds context from DB via `BuildContext` on every call, then persists the turn via `session.AppendTurn`
- Added `CloseSession()` method

### Task 6: `internal/tui/app.go` + `internal/tui/views/narrative.go`
- `NarrativeModel.CloseSession()` delegates to `narrator.CloseSession()`
- `App.cleanup()` calls `narrative.CloseSession()` if a narrative is active
- Esc from narrative view: calls `a.narrative.CloseSession()` before switching to menu
- Ctrl+C: calls `a.cleanup()` before `tea.Quit`
- Menu quit (ActionQuit): calls `a.cleanup()` before `tea.Quit`

## Key Design Decisions
- `message_type` in DB uses `'narrative'` for all main chat messages (matches DB CHECK constraint: `narrative|combat|crafting|dialogue`)
- `countJSONLLines` restores turn counter on session resume without re-parsing JSON
- `AppendTurn` errors are swallowed (non-fatal) so a persistence hiccup doesn't break gameplay
- RAG slot is wired in as `[]string` with a placeholder; Phase 5 fills it

## Verification
- `go build ./cmd/oneday` passes
- `go vet ./...` passes
- 7 files changed, 615 insertions(+), 40 deletions(-)
