---
phase: 2
plan: 2.3
title: "AI response JSON parsing, streaming support, and typewriter effect"
status: completed
---

# Summary: Plan 2.3 — AI Streaming, Response Parsing, Typewriter

## What Was Done

### Task 1 — StreamProvider interface (`internal/ai/provider.go`)
Added `StreamChunk` struct (`Content string`, `Done bool`, `Error error`) and `StreamProvider` interface that embeds `Provider` and adds `Stream(ctx, Request) (<-chan StreamChunk, error)`. The existing `Provider` interface is unchanged; streaming is opt-in.

### Task 2 — OpenAI-compatible streaming (`internal/ai/providers/openai_compat.go`)
Implemented `Stream` on `*OpenAICompat`. Uses `bufio.Scanner` to parse SSE lines (`data: {...}` / `data: [DONE]`). Sends content chunks to a buffered channel (cap 32); goroutine closes the channel after `[DONE]` or scanner error. Added `bufio` and `strings` imports.

### Task 3 — Router streaming (`internal/ai/router.go`)
Added `Router.Stream(ctx, Request) (<-chan StreamChunk, string, error)`. First pass: tries every `StreamProvider` in priority order. Second pass fallback: calls `Complete` on non-streaming providers and wraps the result in a 2-item channel (content + done). Returns the provider name as the second value for status-bar display.

### Task 4 — Response parser (`internal/ai/response.go`)
New file. `NarrativeResponse` struct with all fields: `narrative`, `choices`, `state_changes`, `mood`, `ascii_art`, `achievement_earned`, `challenge`. `ParseNarrativeJSON(text)` extracts and unmarshals the first ` ```json ``` ` block; returns `nil, nil` for pure prose. `ExtractNarrative(text)` strips JSON blocks and trims whitespace, leaving only prose.

### Task 5 — Typewriter component (`internal/tui/components/typewriter.go`)
Standalone Bubbletea model. Key API:
- `NewTypewriter(speed int) TypewriterModel` — speed in chars/sec (default 80)
- `SetText(text) tea.Cmd` — resets and starts animation
- `AppendText(text) tea.Cmd` — extends text for streaming use
- `Skip()` — instantly reveals all text
- `View() string` — currently visible rune slice
- `Update(msg) (TypewriterModel, tea.Cmd)` — handles `typewriterTickMsg`
- `TypewriterDoneMsg{}` sent when last character is revealed
- Unicode-safe (uses `[]rune` indexing throughout)

### Task 6 — Tests
- `internal/ai/streaming_test.go` — 6 tests for `Router.Stream`: prefers StreamProvider, multi-chunk assembly, fallback to Complete, priority ordering, error fallback, all-fail error.
- `internal/ai/response_test.go` — 5 tests for `ParseNarrativeJSON` and `ExtractNarrative`: block extraction, pure prose nil return, invalid JSON error, full field coverage, no-op on plain text.
- `internal/tui/components/typewriter_test.go` — 9 tests: defaults, SetText activation, tick advancement, completion, DoneMsg, Skip, AppendText, Unicode runes, speed config, non-tick message passthrough.
- `internal/ai/providers/openai_compat_test.go` — 3 new tests: SSE stream parsing end-to-end (httptest server), stream 503 error, compile-time StreamProvider interface assertion.

## Verification
- `go build ./...` — success
- `go test ./internal/ai/... ./internal/tui/components/...` — 33 passed, 0 failed

## Notes
- `ClaudeCode` provider intentionally does NOT implement `StreamProvider` (CLI doesn't support streaming); the Router's fallback path handles it transparently.
- `NarrativeResponse` in `internal/ai/response.go` is parallel to `engine/types.go` — the `ai` package stays import-free from `engine` to avoid cycles.
