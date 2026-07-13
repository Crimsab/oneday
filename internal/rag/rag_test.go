package rag

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/storage"
	_ "modernc.org/sqlite"
)

// --- Mock embedding provider ---

type mockEmbedder struct {
	vec []float32
	err error
}

func (m *mockEmbedder) Embed(_ context.Context, _ ai.EmbeddingRequest) (ai.EmbeddingResponse, error) {
	if m.err != nil {
		return ai.EmbeddingResponse{}, m.err
	}
	return ai.EmbeddingResponse{Embedding: m.vec, Model: "mock"}, nil
}

// --- Mock AI completer ---

type mockAI struct {
	text string
	err  error
}

func (m *mockAI) Complete(_ context.Context, _ ai.Request) (ai.Response, error) {
	if m.err != nil {
		return ai.Response{}, m.err
	}
	return ai.Response{Content: m.text}, nil
}

// --- Test helpers ---

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	// Create the rag_chunks table directly (migration not available here).
	_, err = db.Exec(`
		CREATE TABLE rag_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			story_id TEXT NOT NULL,
			text TEXT NOT NULL,
			chunk_type TEXT NOT NULL DEFAULT 'summary',
			turn_start INTEGER NOT NULL DEFAULT 0,
			turn_end INTEGER NOT NULL DEFAULT 0,
			embedding BLOB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		t.Fatalf("create rag_chunks: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func vec(vals ...float32) []float32 { return vals }

// --- Embedder tests ---

func TestEmbedderGenerate(t *testing.T) {
	expected := vec(0.1, 0.2, 0.3)
	e := NewEmbedder(&mockEmbedder{vec: expected}, "mock-model", 3)

	got, err := e.Generate(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected len %d got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("dim %d: expected %f got %f", i, expected[i], got[i])
		}
	}
}

func TestEmbedderEmptyInput(t *testing.T) {
	e := NewEmbedder(&mockEmbedder{vec: vec(0.1)}, "mock-model", 1)
	_, err := e.Generate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

// --- Cosine similarity tests ---

func TestCosineSimilarityIdentical(t *testing.T) {
	a := vec(1, 0, 0)
	sim := cosineSimilarity(a, a)
	if sim < 0.9999 {
		t.Errorf("identical vectors should have similarity ~1.0, got %f", sim)
	}
}

func TestCosineSimilarityOrthogonal(t *testing.T) {
	a := vec(1, 0, 0)
	b := vec(0, 1, 0)
	sim := cosineSimilarity(a, b)
	if sim > 0.001 {
		t.Errorf("orthogonal vectors should have similarity ~0, got %f", sim)
	}
}

func TestCosineSimilarityZeroVector(t *testing.T) {
	a := vec(0, 0, 0)
	b := vec(1, 2, 3)
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("zero vector should give similarity 0, got %f", sim)
	}
}

func TestCosineSimilarityMismatchedLengths(t *testing.T) {
	a := vec(1, 2)
	b := vec(1, 2, 3)
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("mismatched lengths should give 0, got %f", sim)
	}
}

// --- Serialization tests ---

func TestSerializeDeserializeRoundtrip(t *testing.T) {
	original := vec(1.5, -2.3, 0.0, 100.7, -0.001)
	serialized := serializeEmbedding(original)
	restored := deserializeEmbedding(serialized)

	if len(restored) != len(original) {
		t.Fatalf("length mismatch: %d vs %d", len(restored), len(original))
	}
	for i := range original {
		if restored[i] != original[i] {
			t.Errorf("dim %d: %f != %f", i, restored[i], original[i])
		}
	}
}

func TestSerializeEmpty(t *testing.T) {
	if b := serializeEmbedding(nil); b != nil {
		t.Errorf("expected nil for nil input, got %v", b)
	}
	if v := deserializeEmbedding(nil); v != nil {
		t.Errorf("expected nil for nil input, got %v", v)
	}
}

// --- VectorStore tests ---

func TestVectorStoreInsertAndSearch(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	ctx := context.Background()
	storyID := "story-1"

	// Insert 5 chunks with known embeddings.
	chunks := []struct {
		text string
		emb  []float32
	}{
		{"The hero entered the dark forest", vec(1, 0, 0)},
		{"A merchant sold a magic sword", vec(0, 1, 0)},
		{"The dragon attacked the village", vec(0, 0, 1)},
		{"The hero found a hidden treasure", vec(0.9, 0.1, 0)},
		{"An old wizard gave advice", vec(0, 0.9, 0.1)},
	}

	for i, c := range chunks {
		chunk := &Chunk{
			StoryID:   storyID,
			Text:      c.text,
			ChunkType: "summary",
			TurnStart: i * 10,
			TurnEnd:   i*10 + 9,
			Embedding: c.emb,
		}
		if err := vs.Insert(ctx, chunk); err != nil {
			t.Fatalf("insert chunk %d: %v", i, err)
		}
		if chunk.ID == 0 {
			t.Errorf("expected non-zero ID after insert")
		}
	}

	// Query identical to chunk[0] — should rank first with similarity ~1.0.
	query := vec(1, 0, 0)
	results, err := vs.Search(ctx, storyID, query, 3)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Chunk.Text != "The hero entered the dark forest" {
		t.Errorf("expected top result to be chunk[0], got: %q", results[0].Chunk.Text)
	}
	if results[0].Similarity < 0.999 {
		t.Errorf("expected similarity ~1.0 for identical vector, got %f", results[0].Similarity)
	}
}

func TestVectorStoreEmptyStore(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	ctx := context.Background()

	results, err := vs.Search(ctx, "nonexistent", vec(1, 0, 0), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results for empty store, got %d", len(results))
	}
}

func TestVectorStoreCountAndLastTurn(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	ctx := context.Background()
	storyID := "story-count"

	count, err := vs.CountByStory(ctx, storyID)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	lastTurn, err := vs.LastSummarizedTurn(ctx, storyID)
	if err != nil {
		t.Fatalf("last turn: %v", err)
	}
	if lastTurn != -1 {
		t.Errorf("expected -1 for empty store, got %d", lastTurn)
	}

	_ = vs.Insert(ctx, &Chunk{StoryID: storyID, Text: "t", ChunkType: "summary", TurnStart: 1, TurnEnd: 10, Embedding: vec(1)})
	_ = vs.Insert(ctx, &Chunk{StoryID: storyID, Text: "t", ChunkType: "summary", TurnStart: 11, TurnEnd: 20, Embedding: vec(1)})

	count, _ = vs.CountByStory(ctx, storyID)
	if count != 2 {
		t.Errorf("expected 2 chunks, got %d", count)
	}

	lastTurn, _ = vs.LastSummarizedTurn(ctx, storyID)
	if lastTurn != 20 {
		t.Errorf("expected last turn 20, got %d", lastTurn)
	}
}

func TestVectorStoreReadsNewInsertsWithoutProcessCache(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	ctx := context.Background()
	storyID := "story-cache"

	if err := vs.Insert(ctx, &Chunk{
		StoryID:   storyID,
		Text:      "older chunk",
		ChunkType: "summary",
		TurnStart: 1,
		TurnEnd:   5,
		Embedding: vec(1, 0, 0),
	}); err != nil {
		t.Fatalf("insert first chunk: %v", err)
	}

	results, err := vs.Search(ctx, storyID, vec(1, 0, 0), 1)
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	if len(results) != 1 || results[0].Chunk.Text != "older chunk" {
		t.Fatalf("unexpected first result: %+v", results)
	}

	if err := vs.Insert(ctx, &Chunk{
		StoryID:   storyID,
		Text:      "new hot chunk",
		ChunkType: "summary",
		TurnStart: 6,
		TurnEnd:   10,
		Embedding: vec(0, 1, 0),
	}); err != nil {
		t.Fatalf("insert second chunk: %v", err)
	}

	results, err = vs.Search(ctx, storyID, vec(0, 1, 0), 1)
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	if len(results) != 1 || results[0].Chunk.Text != "new hot chunk" {
		t.Fatalf("new insert was not visible: %+v", results)
	}
}

// --- Summarizer tests ---

func TestSummarizerShouldSummarize(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	embedder := NewEmbedder(&mockEmbedder{vec: vec(0.1, 0.2)}, "mock", 2)
	s := NewSummarizer(embedder, vs, &mockAI{text: "summary"}, "story-s", 10)
	ctx := context.Background()

	// No summaries yet — committed turns 0..8 count as 9 turns, not enough.
	should, err := s.ShouldSummarize(ctx, 8)
	if err != nil || should {
		t.Errorf("should be false at turn 8: %v %v", should, err)
	}

	// First summary window covers committed turns 0..9.
	should, err = s.ShouldSummarize(ctx, 9)
	if err != nil || !should {
		t.Errorf("should be true at turn 9: %v %v", should, err)
	}

	// After storing a summary up to turn 14, committed turns 15..23 count as 9.
	_ = vs.Insert(ctx, &Chunk{StoryID: "story-s", Text: "x", ChunkType: "summary", TurnStart: 0, TurnEnd: 14, Embedding: vec(0.1, 0.2)})
	should, _ = s.ShouldSummarize(ctx, 23)
	if should {
		t.Errorf("gap 9 after stored summary: should be false")
	}
	should, _ = s.ShouldSummarize(ctx, 24)
	if !should {
		t.Errorf("gap 10 after stored summary: should be true")
	}
}

func TestSummarizerPendingWindow(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	embedder := NewEmbedder(&mockEmbedder{vec: vec(0.1, 0.2)}, "mock", 2)
	s := NewSummarizer(embedder, vs, &mockAI{text: "summary"}, "story-window", 10)
	ctx := context.Background()

	start, end, should, err := s.PendingWindow(ctx, 8)
	if err != nil {
		t.Fatalf("PendingWindow turn 8: %v", err)
	}
	if should || start != 0 || end != 0 {
		t.Fatalf("unexpected pending window at turn 8: start=%d end=%d should=%v", start, end, should)
	}

	start, end, should, err = s.PendingWindow(ctx, 9)
	if err != nil {
		t.Fatalf("PendingWindow turn 9: %v", err)
	}
	if !should || start != 0 || end != 9 {
		t.Fatalf("unexpected pending window at turn 9: start=%d end=%d should=%v", start, end, should)
	}

	if err := vs.Insert(ctx, &Chunk{
		StoryID:   "story-window",
		Text:      "existing summary",
		ChunkType: "summary",
		TurnStart: 0,
		TurnEnd:   14,
		Embedding: vec(0.1, 0.2),
	}); err != nil {
		t.Fatalf("insert existing summary: %v", err)
	}

	start, end, should, err = s.PendingWindow(ctx, 24)
	if err != nil {
		t.Fatalf("PendingWindow turn 24: %v", err)
	}
	if !should || start != 15 || end != 24 {
		t.Fatalf("unexpected pending window after summary: start=%d end=%d should=%v", start, end, should)
	}
}

func TestSummarizerSummarize(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	embedder := NewEmbedder(&mockEmbedder{vec: vec(0.5, 0.5)}, "mock", 2)
	aiMock := &mockAI{text: "The hero fought bravely and won the battle."}
	s := NewSummarizer(embedder, vs, aiMock, "story-sum", 10)
	ctx := context.Background()

	msgs := []storage.ChatMessage{
		{Turn: 0, Role: "user", Content: "I attack the goblin"},
		{Turn: 0, Role: "assistant", Content: "You strike the goblin down."},
		{Turn: 1, Role: "user", Content: "I loot the body"},
		{Turn: 1, Role: "assistant", Content: "You find 5 gold coins."},
	}

	err := s.Summarize(ctx, msgs, 9)
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}

	count, _ := vs.CountByStory(ctx, "story-sum")
	if count != 1 {
		t.Errorf("expected 1 chunk stored, got %d", count)
	}

	results, _ := vs.Search(ctx, "story-sum", vec(0.5, 0.5), 1)
	if len(results) == 0 {
		t.Fatal("no results after summarize")
	}
	if results[0].Chunk.ChunkType != "summary" {
		t.Errorf("expected chunk_type 'summary', got %q", results[0].Chunk.ChunkType)
	}
	if results[0].Chunk.Text != aiMock.text {
		t.Errorf("expected summary text %q, got %q", aiMock.text, results[0].Chunk.Text)
	}
	if results[0].Chunk.TurnStart != 0 {
		t.Errorf("expected turn_start 0, got %d", results[0].Chunk.TurnStart)
	}
}

func TestSummarizerSkipsAlreadySummarized(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	embedder := NewEmbedder(&mockEmbedder{vec: vec(0.5, 0.5)}, "mock", 2)
	s := NewSummarizer(embedder, vs, &mockAI{text: "summary"}, "story-skip", 10)
	ctx := context.Background()

	// Store a summary up to turn 5.
	_ = vs.Insert(ctx, &Chunk{StoryID: "story-skip", Text: "old", ChunkType: "summary", TurnStart: 0, TurnEnd: 4, Embedding: vec(0.5, 0.5)})

	// Messages only for turns <= 4 — nothing new to summarize.
	msgs := []storage.ChatMessage{
		{Turn: 3, Role: "user", Content: "action"},
		{Turn: 4, Role: "assistant", Content: "response"},
	}

	err := s.Summarize(ctx, msgs, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, _ := vs.CountByStory(ctx, "story-skip")
	if count != 1 {
		t.Errorf("expected count to remain 1 (no new chunk), got %d", count)
	}
}

// --- RAG orchestrator tests ---

func TestRAGRetrieve(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	ctx := context.Background()
	storyID := "story-rag"

	// Pre-populate store.
	chunks := []struct {
		text string
		emb  []float32
	}{
		{"The hero fought the dragon", vec(1, 0, 0)},
		{"A merchant appeared in the market", vec(0, 1, 0)},
		{"Rain fell on the dark city", vec(0, 0, 1)},
	}
	for i, c := range chunks {
		_ = vs.Insert(ctx, &Chunk{
			StoryID:   storyID,
			Text:      c.text,
			ChunkType: "summary",
			TurnStart: i * 5,
			TurnEnd:   i*5 + 4,
			Embedding: c.emb,
		})
	}

	// Embedder returns vec(1,0,0) — should retrieve dragon chunk first.
	embedder := NewEmbedder(&mockEmbedder{vec: vec(1, 0, 0)}, "mock", 3)
	summarizer := NewSummarizer(embedder, vs, &mockAI{text: "x"}, storyID, 10)
	r := NewRAG(embedder, vs, summarizer, storyID, 2)

	texts, err := r.Retrieve(ctx, "fighting monsters")
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(texts) != 2 {
		t.Fatalf("expected 2 results, got %d", len(texts))
	}
	if texts[0] != "The hero fought the dragon" {
		t.Errorf("expected dragon chunk first, got %q", texts[0])
	}
}

func TestRAGMaybeSummarize(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	ctx := context.Background()
	storyID := "story-maybe"

	embedder := NewEmbedder(&mockEmbedder{vec: vec(0.5, 0.5)}, "mock", 2)
	summarizer := NewSummarizer(embedder, vs, &mockAI{text: "summary text"}, storyID, 5)
	r := NewRAG(embedder, vs, summarizer, storyID, 3)

	msgs := []storage.ChatMessage{
		{Turn: 0, Role: "user", Content: "action 0"},
		{Turn: 1, Role: "assistant", Content: "response 1"},
		{Turn: 2, Role: "user", Content: "action 2"},
		{Turn: 3, Role: "assistant", Content: "response 3"},
		{Turn: 4, Role: "user", Content: "action 4"},
	}

	// Turn 3: committed turns 0..3 count as 4, still below interval.
	fired, err := r.MaybeSummarize(ctx, msgs, 3)
	if err != nil || fired {
		t.Errorf("should not have fired at turn 3: fired=%v err=%v", fired, err)
	}

	// Turn 4: committed turns 0..4 count as 5 → should summarize.
	fired, err = r.MaybeSummarize(ctx, msgs, 4)
	if err != nil {
		t.Fatalf("unexpected error at turn 4: %v", err)
	}
	if !fired {
		t.Error("expected summarization to fire at turn 4")
	}

	count, _ := vs.CountByStory(ctx, storyID)
	if count != 1 {
		t.Errorf("expected 1 chunk after summarization, got %d", count)
	}
}

func TestRAGRetrieveEmptyStore(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	embedder := NewEmbedder(&mockEmbedder{vec: vec(1, 0)}, "mock", 2)
	summarizer := NewSummarizer(embedder, vs, &mockAI{text: "x"}, "empty", 10)
	r := NewRAG(embedder, vs, summarizer, "empty", 5)

	texts, err := r.Retrieve(context.Background(), "query")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(texts) != 0 {
		t.Errorf("expected empty results, got %d", len(texts))
	}
}

func TestRAGRetrieveEmbedderFailure(t *testing.T) {
	db := openTestDB(t)
	vs := NewVectorStore(db)
	// Embedder that always fails.
	embedder := NewEmbedder(&mockEmbedder{err: &mockError{"embed failed"}}, "mock", 2)
	summarizer := NewSummarizer(embedder, vs, &mockAI{text: "x"}, "fail", 10)
	r := NewRAG(embedder, vs, summarizer, "fail", 5)

	// Should return empty slice, not error (RAG is non-fatal).
	texts, err := r.Retrieve(context.Background(), "query")
	if err != nil {
		t.Fatalf("retrieve should not return error on embedder failure, got: %v", err)
	}
	if texts != nil {
		t.Errorf("expected nil texts on embedder failure, got %v", texts)
	}
}

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

// Ensure time import is used (for storage.ChatMessage.CreatedAt).
var _ = time.Now
