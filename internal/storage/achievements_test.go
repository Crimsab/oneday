package storage

import (
	"path/filepath"
	"testing"
)

func TestAchievementExistsByNameCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "achievements.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if _, err := db.Conn().Exec(`INSERT INTO stories (id, name) VALUES (?, ?)`, "story-1", "Test Story"); err != nil {
		t.Fatalf("insert story: %v", err)
	}

	if err := db.CreateAchievement(&Achievement{
		StoryID:     "story-1",
		Name:        "First Step",
		Description: "You made it out alive.",
		Category:    "story",
		Rarity:      "common",
	}); err != nil {
		t.Fatalf("CreateAchievement: %v", err)
	}

	exists, err := db.AchievementExistsByName("story-1", "first step")
	if err != nil {
		t.Fatalf("AchievementExistsByName existing: %v", err)
	}
	if !exists {
		t.Fatal("expected achievement lookup to be case-insensitive")
	}

	exists, err = db.AchievementExistsByName("story-1", "missing")
	if err != nil {
		t.Fatalf("AchievementExistsByName missing: %v", err)
	}
	if exists {
		t.Fatal("unexpected match for missing achievement")
	}
}
