package views

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildCommandSuggestionsFiltersTalk(t *testing.T) {
	items := buildCommandSuggestions("ta")
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Value != "/talk " {
		t.Fatalf("items[0].Value = %q, want /talk ", items[0].Value)
	}
}

func TestNearbyTalkNPCSuggestionItemsPreferRecentRoster(t *testing.T) {
	db := openAutocompleteTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:        "story-talk",
		Name:      "Talk Story",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	mustCreateNPC(t, db, story.ID, "Lyanna", "scout", 9, now)
	mustCreateNPC(t, db, story.ID, "Brother Alden", "healer", 8, now)
	mustCreateNPC(t, db, story.ID, "Distant Duke", "noble", 2, now)

	items := nearbyTalkNPCSuggestionItems(db, story.ID, 10, 6, "")
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 recent NPCs", len(items))
	}
	if items[0].Label != "Lyanna" {
		t.Fatalf("items[0].Label = %q, want Lyanna", items[0].Label)
	}
	if items[1].Label != "Brother Alden" {
		t.Fatalf("items[1].Label = %q, want Brother Alden", items[1].Label)
	}
}

func TestBuildTalkIntentSuggestionItemsFiltersIntentPrefix(t *testing.T) {
	items := buildTalkIntentSuggestionItems("Lyanna", "pr")
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Value != "/talk Lyanna probe" {
		t.Fatalf("items[0].Value = %q, want /talk Lyanna probe", items[0].Value)
	}
	if items[1].Value != "/talk Lyanna promise" {
		t.Fatalf("items[1].Value = %q, want /talk Lyanna promise", items[1].Value)
	}
}

func TestBuildCraftingChoiceItemsKeepsExitLast(t *testing.T) {
	items, exitChoiceID := buildCraftingChoiceItems([]engine.Choice{
		{ID: 1, Text: "Forge a knife"},
		{ID: 2, Text: "Go back"},
	})

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Text != "Forge a knife" {
		t.Fatalf("items[0].Text = %q, want Forge a knife", items[0].Text)
	}
	if items[1].Text != "Go back" {
		t.Fatalf("items[1].Text = %q, want Go back", items[1].Text)
	}
	if items[1].ID != exitChoiceID {
		t.Fatalf("items[1].ID = %d, want exitChoiceID %d", items[1].ID, exitChoiceID)
	}
}

func openAutocompleteTestDB(t *testing.T) *storage.DB {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "autocomplete.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func mustCreateNPC(t *testing.T, db *storage.DB, storyID, name, role string, lastSeenTurn int, now time.Time) {
	t.Helper()

	if err := db.CreateNPC(&storage.NPC{
		ID:                name + "-id",
		StoryID:           storyID,
		Name:              name,
		Role:              role,
		PersonalityJSON:   `{}`,
		RelationshipJSON:  `{}`,
		Disposition:       0,
		IsAlive:           true,
		FirstAppearedTurn: lastSeenTurn,
		LastSeenTurn:      lastSeenTurn,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC(%s): %v", name, err)
	}
}
