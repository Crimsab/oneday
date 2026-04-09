package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestAppendHistoryEntryDoesNotAdvanceTurn(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	story := &storage.Story{
		ID:              "story-session",
		Name:            "Session Tale",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	session, err := NewGameSession(db, story.ID, filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	session.SetTurn(3)
	if err := session.AppendHistoryEntry(db, ChatEntry{
		Turn:        3,
		Timestamp:   time.Now(),
		MessageType: "combat_summary",
		Output: &ChatOutput{
			Narrative: "Victory summary",
			Mood:      "neutral",
		},
	}); err != nil {
		t.Fatalf("AppendHistoryEntry: %v", err)
	}

	if got := session.Turn(); got != 3 {
		t.Fatalf("session turn = %d, want 3", got)
	}

	msgs, err := db.GetSessionMessages(session.SessionID())
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("message count = %d, want 1", len(msgs))
	}
	if msgs[0].MessageType != "combat_summary" {
		t.Fatalf("message_type = %q, want combat_summary", msgs[0].MessageType)
	}
	if msgs[0].Turn != 3 {
		t.Fatalf("message turn = %d, want 3", msgs[0].Turn)
	}
}
