package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestRecentMessagesForTurnFallsBackToStoryWhenSessionIsEmpty(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now()
	story := &storage.Story{
		ID:              "story-context",
		Name:            "Context Story",
		Description:     "A story with a closed prior session.",
		Genre:           "mystery",
		Tone:            "tense",
		Language:        "en",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	endedAt := now.Add(time.Minute)
	oldSession := &storage.Session{
		ID:        "old-session",
		StoryID:   story.ID,
		StartedAt: now,
		EndedAt:   &endedAt,
	}
	if err := db.CreateSession(oldSession); err != nil {
		t.Fatalf("CreateSession old: %v", err)
	}
	if err := db.AppendChatMessage(&storage.ChatMessage{
		SessionID:   oldSession.ID,
		StoryID:     story.ID,
		Turn:        1,
		Role:        "assistant",
		Content:     "The last clue was the blue ticket.",
		MessageType: "narrative",
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("AppendChatMessage old: %v", err)
	}

	session, err := NewGameSession(db, story.ID, filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	t.Cleanup(func() { _ = session.CloseMirrors() })

	narrator := &Narrator{
		db:         db,
		story:      story,
		session:    session,
		contextCfg: ContextConfig{RecentMessageCount: 5},
	}
	recent, err := narrator.recentMessagesForTurn(5)
	if err != nil {
		t.Fatalf("recentMessagesForTurn: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("recent message count = %d, want 1", len(recent))
	}
	if recent[0].SessionID != oldSession.ID || recent[0].Content != "The last clue was the blue ticket." {
		t.Fatalf("recent message = %+v", recent[0])
	}
}
