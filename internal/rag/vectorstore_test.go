package rag

import (
	"context"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

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
