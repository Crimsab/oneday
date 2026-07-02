package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func createStoryFixture(t *testing.T, db *DB, storyID string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	if err := db.CreateStory(&Story{
		ID:              storyID,
		Name:            "Lock Story",
		SettingJSON:     "{}",
		StatsSchemaJSON: "{}",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	if err := db.CreateWorldState(&WorldState{
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

func TestStoryTurnLockSerializesOwners(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "locks.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	createStoryFixture(t, db, "story-lock")

	first, err := db.AcquireStoryTurnLock(context.Background(), "story-lock", "owner-a", time.Minute)
	if err != nil {
		t.Fatalf("Acquire first: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if _, err := db.AcquireStoryTurnLock(ctx, "story-lock", "owner-b", time.Minute); err == nil {
		t.Fatal("Acquire second succeeded while first owner held the lock")
	}

	if err := first.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if second, err := db.AcquireStoryTurnLock(context.Background(), "story-lock", "owner-b", time.Minute); err != nil {
		t.Fatalf("Acquire second after release: %v", err)
	} else {
		_ = second.Release()
	}
}

func TestStoryTurnLockExpires(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "locks-expire.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	createStoryFixture(t, db, "story-lock-expire")

	first, err := db.AcquireStoryTurnLock(context.Background(), "story-lock-expire", "owner-a", time.Millisecond)
	if err != nil {
		t.Fatalf("Acquire first: %v", err)
	}
	defer first.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	second, err := db.AcquireStoryTurnLock(ctx, "story-lock-expire", "owner-b", time.Millisecond)
	if err != nil {
		t.Fatalf("Acquire after expiry: %v", err)
	}
	_ = second.Release()
}
