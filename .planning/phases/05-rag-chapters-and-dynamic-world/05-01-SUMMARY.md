---
phase: 5
plan: 5.1
title: "Embedding client, vector storage, and periodic summarization engine"
status: completed
---

# Plan 5.1 — Execution Summary

## What Was Built

### Wave 1 — Foundation

**`internal/ai/provider.go`** — Added `EmbeddingRequest` and `EmbeddingResponse` types at the `ai` package level.

**`internal/ai/providers/openai_compat.go`** — Added `Embed()` method to `OpenAICompat` that POSTs to `/v1/embeddings` and parses the `data[0].embedding` float array.

**`internal/rag/embeddings.go`** — `Embedder` struct wrapping an `EmbeddingProvider` interface (mockable). `Generate(ctx, text) ([]float32, error)`.

**`internal/rag/vectorstore.go`** — `VectorStore` backed by the existing SQLite database. Pure-Go brute-force cosine similarity. Embedding BLOBs serialized as little-endian IEEE 754 float32. Functions: `Insert`, `Search` (topK), `CountByStory`, `LastSummarizedTurn`.

**`internal/storage/migrations.go`** — Migration V4 adds `rag_chunks` table with `story_id`, `text`, `chunk_type`, `turn_start`, `turn_end`, `embedding BLOB`, indexes on `story_id` and `(story_id, chunk_type)`.

### Wave 2 — Summarization

**`internal/ai/prompts/summarizer.go`** — `SummarizerSystem` constant: system prompt instructing the AI to produce factual, third-person, 200–500 word summaries for long-term memory retrieval.

**`internal/storage/chat.go`** — Added `GetStoryMessagesByTurnRange(storyID, turnStart, turnEnd)` for the summarizer to fetch unsummarized messages.

**`internal/rag/summarizer.go`** — `Summarizer` checks `LastSummarizedTurn` vs `currentTurn`. When gap ≥ interval, calls AI to summarize unsummarized messages, generates embedding, stores a `Chunk{ChunkType:"summary"}`. `AICompleter` interface satisfied by `*ai.Router`.

**`internal/rag/rag.go`** — Top-level `RAG` orchestrator. `Retrieve(ctx, query)` embeds query → vector search → `[]string` of chunk texts (returns empty on error, not hard failure). `MaybeSummarize(ctx, msgs, turn)` delegates to Summarizer.

### Wave 3 — Integration

**`internal/config/config.go`** — Added `Enabled bool` field to `RAGConfig` (default `true`). When false, narrator receives nil RAG.

**`internal/engine/narrator.go`** — Added `rag *rag.RAG` field. `SetRAG(*rag.RAG)` wires it post-construction. In `sendTurn`: retrieves RAG chunks before building context (replaces nil `n.contextCfg.RAGChunks`). After `AppendTurn`: fires async goroutine calling `MaybeSummarize` (fire-and-forget).

**`internal/tui/app.go`** — Added `buildRAG(storyID)` helper that constructs `Embedder → VectorStore → Summarizer → RAG` using the LiteLLM provider config. Called in both `enterNarrativeView` and `loadSaveAndResume` via `narrator.SetRAG(...)`.

## Tests

**`internal/rag/rag_test.go`** — 18 tests covering:
- `Embedder.Generate` (happy path + empty input)
- `cosineSimilarity` (identical, orthogonal, zero, mismatched lengths)
- Serialization roundtrip + nil handling
- `VectorStore` insert, search ranking (identical vector ranks first ~1.0), empty store, count, `LastSummarizedTurn`
- `Summarizer.ShouldSummarize` (boundary conditions), `Summarize` (produces chunk with correct fields), skip-already-summarized
- `RAG.Retrieve` (top-K ordering), `MaybeSummarize` (fires at interval, not before), empty store, embedder failure (non-fatal)

## Key Design Decisions

- **No sqlite-vec**: Pure-Go cosine similarity; project uses `modernc.org/sqlite` (pure Go, no CGO). Brute-force is adequate for <10K vectors.
- **RAG is non-fatal**: All errors in `Retrieve` are swallowed and logged; gameplay continues without long-term memory if embedding API is down.
- **Async summarization**: Goroutine after each turn so it never blocks the narrative response.
- **Same DB**: `rag_chunks` lives in the main oneday.db — no separate embeddings.db file.

## Verification

- `go build ./...` — passes
- `go test ./internal/rag/... -v` — 18/18 tests pass
