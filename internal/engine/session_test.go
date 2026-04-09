package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func createSessionTestStory(t *testing.T, db *storage.DB, storyID string, currentTurn int) {
	t.Helper()

	now := time.Now()
	story := &storage.Story{
		ID:              storyID,
		Name:            "Session Tale",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := &storage.Character{
		ID:               "char-" + storyID,
		StoryID:          storyID,
		Name:             "Hero",
		Background:       "Test",
		StatsJSON:        `{"vitals":{"hp":{"current":10,"max":10}}}`,
		TraitsJSON:       `[]`,
		SkillsJSON:       `{}`,
		InventoryJSON:    `[]`,
		KnownRecipesJSON: `[]`,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	world := &storage.WorldState{
		ID:                   "world-" + storyID,
		StoryID:              storyID,
		CurrentLocation:      "Village",
		KnownLocationsJSON:   `["Village"]`,
		GlobalEventsJSON:     `[]`,
		FactionStandingsJSON: `{}`,
		StoryHooksJSON:       `[]`,
		WorldReactionsJSON:   `[]`,
		PlayerGuidanceJSON:   `[]`,
		CurrentChapter:       1,
		CurrentTurn:          currentTurn,
		UpdatedAt:            now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}
}

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

func TestNewGameSessionUsesCanonicalTurnInsteadOfJSONLMirror(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	createSessionTestStory(t, db, "story-canonical-cursor", 2)
	sessionRow := &storage.Session{
		ID:        "stale-jsonl",
		StoryID:   "story-canonical-cursor",
		StartedAt: time.Now(),
	}
	if err := db.CreateSession(sessionRow); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessionDir := filepath.Join(root, "data", "stories", "story-canonical-cursor", "sessions", "stale-jsonl")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "main.jsonl"), []byte("{\"turn\":99}\n{\"turn\":100}\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	session, err := NewGameSession(db, "story-canonical-cursor", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	if got := session.Turn(); got != 2 {
		t.Fatalf("session turn = %d, want 2 from canonical DB state", got)
	}
}

func TestNewGameSessionIgnoresMetaOnlyHistoryWhenRecoveringTurnCursor(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	createSessionTestStory(t, db, "story-meta-cursor", 2)
	now := time.Now()
	sessionRow := &storage.Session{
		ID:        "session-meta",
		StoryID:   "story-meta-cursor",
		StartedAt: now,
	}
	if err := db.CreateSession(sessionRow); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := db.AppendChatMessage(&storage.ChatMessage{
		SessionID:    sessionRow.ID,
		StoryID:      "story-meta-cursor",
		Turn:         8,
		Role:         "assistant",
		Content:      "Meta answer",
		MessageType:  "narrator",
		MetadataJSON: "{}",
		CreatedAt:    now,
	}); err != nil {
		t.Fatalf("AppendChatMessage narrator: %v", err)
	}
	if err := db.AppendChatMessage(&storage.ChatMessage{
		SessionID:    sessionRow.ID,
		StoryID:      "story-meta-cursor",
		Turn:         9,
		Role:         "assistant",
		Content:      "Combat summary",
		MessageType:  "combat_summary",
		MetadataJSON: "{}",
		CreatedAt:    now,
	}); err != nil {
		t.Fatalf("AppendChatMessage combat_summary: %v", err)
	}

	session, err := NewGameSession(db, "story-meta-cursor", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	if got := session.Turn(); got != 2 {
		t.Fatalf("session turn = %d, want 2 with meta-only history ignored", got)
	}
}

func TestAppendTurnCanonicalDBFailureDoesNotAdvanceTurn(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}

	createSessionTestStory(t, db, "story-db-failure", 0)

	session, err := NewGameSession(db, "story-db-failure", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(nil)

	if err := db.Close(); err != nil {
		t.Fatalf("db.Close: %v", err)
	}

	err = session.AppendTurn(db, ChatEntry{
		Timestamp: time.Now(),
		Input: &ChatInput{
			Type: "free_action",
			Text: "Inspect the room",
		},
		Output: &ChatOutput{
			Narrative: "You inspect the room.",
		},
	})
	if err == nil {
		t.Fatal("AppendTurn error = nil, want canonical DB failure")
	}
	if got := session.Turn(); got != 0 {
		t.Fatalf("session turn = %d, want 0 after canonical DB failure", got)
	}
}

func TestAppendTurnMirrorFailureStillAdvancesCanonicalTurn(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	createSessionTestStory(t, db, "story-mirror-failure", 0)

	session, err := NewGameSession(db, "story-mirror-failure", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	if err := session.jsonlFile.Close(); err != nil {
		t.Fatalf("closing jsonl mirror: %v", err)
	}

	err = session.AppendTurn(db, ChatEntry{
		Timestamp: time.Now(),
		Input: &ChatInput{
			Type: "free_action",
			Text: "Open the gate",
		},
		Output: &ChatOutput{
			Narrative: "The gate groans open.",
			Location:  "Village",
		},
	})
	if err == nil {
		t.Fatal("AppendTurn error = nil, want mirror sync error")
	}
	if !IsMirrorSyncError(err) {
		t.Fatalf("AppendTurn error = %v, want mirror sync error", err)
	}
	if got := session.Turn(); got != 1 {
		t.Fatalf("session turn = %d, want 1 after canonical DB commit", got)
	}

	msgs, err := db.GetSessionMessages(session.SessionID())
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("message count = %d, want 2 canonical DB messages", len(msgs))
	}
}
