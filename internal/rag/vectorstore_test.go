package rag

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestVectorStorePersistsAndBackfillsEmbeddingNorms(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now()
	if err := db.CreateStory(&storage.Story{ID: "story-norm", Name: "Story", SettingJSON: "{}", StatsSchemaJSON: "{}", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	store := NewVectorStore(db.Conn())
	chunk := &Chunk{StoryID: "story-norm", Text: "norm", ChunkType: "summary", Embedding: []float32{1, 2, 3}}
	if err := store.Insert(context.Background(), chunk); err != nil {
		t.Fatal(err)
	}
	var norm float64
	if err := db.Conn().QueryRow(`SELECT embedding_norm FROM rag_chunks WHERE id=?`, chunk.ID).Scan(&norm); err != nil {
		t.Fatal(err)
	}
	if math.Abs(norm-14) > 1e-9 {
		t.Fatalf("inserted norm=%f want=14", norm)
	}
	if _, err := db.Conn().Exec(`UPDATE rag_chunks SET embedding_norm=0 WHERE id=?`, chunk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Search(context.Background(), "story-norm", []float32{1, 2, 3}, 1); err != nil {
		t.Fatal(err)
	}
	if err := db.Conn().QueryRow(`SELECT embedding_norm FROM rag_chunks WHERE id=?`, chunk.ID).Scan(&norm); err != nil {
		t.Fatal(err)
	}
	if math.Abs(norm-14) > 1e-9 {
		t.Fatalf("backfilled norm=%f want=14", norm)
	}
}

func TestVectorStorePruneDimensionMismatches(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewVectorStore(db.Conn())
	ctx := context.Background()
	now := time.Now()
	if err := db.CreateStory(&storage.Story{
		ID:              "story-1",
		Name:            "Story",
		SettingJSON:     "{}",
		StatsSchemaJSON: "{}",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Insert(ctx, &Chunk{StoryID: "story-1", Text: "old", ChunkType: "summary", Embedding: []float32{1, 2}}); err != nil {
		t.Fatal(err)
	}
	if err := store.Insert(ctx, &Chunk{StoryID: "story-1", Text: "new", ChunkType: "summary", Embedding: []float32{1, 2, 3}}); err != nil {
		t.Fatal(err)
	}

	removed, err := store.PruneDimensionMismatches(ctx, "story-1", 3)
	if err != nil {
		t.Fatalf("PruneDimensionMismatches: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	count, err := store.CountByStory(ctx, "story-1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestVectorStoreDeleteByStory(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now()
	if err := db.CreateStory(&storage.Story{ID: "story-1", Name: "Story", SettingJSON: "{}", StatsSchemaJSON: "{}", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	store := NewVectorStore(db.Conn())
	ctx := context.Background()
	if err := store.Insert(ctx, &Chunk{StoryID: "story-1", Text: "x", ChunkType: "summary", Embedding: []float32{1}}); err != nil {
		t.Fatal(err)
	}
	removed, err := store.DeleteByStory(ctx, "story-1")
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
}
