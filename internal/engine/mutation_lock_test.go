package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestStoryMutationLeaseSerializesOwners(t *testing.T) {
	db, err := storage.Open(filepath.Join(t.TempDir(), "mutation-lease.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	createMutationLeaseStory(t, db, "lease-story")

	first, err := AcquireStoryMutationLease(context.Background(), db, "lease-story", "turn", "test-a")
	if err != nil {
		t.Fatalf("Acquire first: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if _, err := AcquireStoryMutationLease(ctx, db, "lease-story", "save-load", "test-b"); err == nil {
		t.Fatal("Acquire second succeeded while first lease was held")
	}

	if err := first.Release(); err != nil {
		t.Fatalf("Release first: %v", err)
	}
	second, err := AcquireStoryMutationLease(context.Background(), db, "lease-story", "save-load", "test-b")
	if err != nil {
		t.Fatalf("Acquire second after release: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("Release second: %v", err)
	}
}

func createMutationLeaseStory(t *testing.T, db *storage.DB, storyID string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	if err := db.CreateStory(&storage.Story{
		ID:              storyID,
		Name:            "Lease Story",
		SettingJSON:     "{}",
		StatsSchemaJSON: "{}",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	if err := db.CreateWorldState(&storage.WorldState{
		ID:              storyID + "-world",
		StoryID:         storyID,
		CurrentLocation: "Harbor",
		CurrentChapter:  1,
		CurrentTurn:     0,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}
}
