package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestFormatPrivateThoughtsViewShowsOnlyNPCsWithThoughts(t *testing.T) {
	db, err := storage.Open(t.TempDir() + "/thoughts.db")
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now()
	story := &storage.Story{ID: "story-thoughts", Name: "Thought Story", CreatedAt: now, UpdatedAt: now}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-lyanna",
		StoryID:           story.ID,
		Name:              "Lyanna",
		Role:              "scout",
		PersonalityJSON:   `{}`,
		PrivateThoughts:   `["He is more dangerous than he looks.","I should test his loyalty."]`,
		RelationshipJSON:  `{}`,
		Disposition:       10,
		IsAlive:           true,
		FirstAppearedTurn: 1,
		LastSeenTurn:      3,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC(Lyanna): %v", err)
	}

	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-thorne",
		StoryID:           story.ID,
		Name:              "Thorne",
		Role:              "mentor",
		PersonalityJSON:   `{}`,
		PrivateThoughts:   `[]`,
		RelationshipJSON:  `{}`,
		Disposition:       20,
		IsAlive:           true,
		FirstAppearedTurn: 1,
		LastSeenTurn:      3,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC(Thorne): %v", err)
	}

	view := FormatPrivateThoughtsView(db, story.ID)
	if !strings.Contains(view, "Lyanna (scout)") {
		t.Fatalf("view missing Lyanna header:\n%s", view)
	}
	if !strings.Contains(view, "He is more dangerous than he looks.") {
		t.Fatalf("view missing first Lyanna thought:\n%s", view)
	}
	if strings.Contains(view, "Thorne") {
		t.Fatalf("view should hide NPCs with no thoughts:\n%s", view)
	}
}

func TestFormatPrivateThoughtsViewHandlesEmptyState(t *testing.T) {
	db, err := storage.Open(t.TempDir() + "/thoughts-empty.db")
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now()
	story := &storage.Story{ID: "story-empty-thoughts", Name: "Thought Story", CreatedAt: now, UpdatedAt: now}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	view := FormatPrivateThoughtsView(db, story.ID)
	if view != "No private NPC thoughts recorded yet." {
		t.Fatalf("view = %q, want empty-state message", view)
	}
}
