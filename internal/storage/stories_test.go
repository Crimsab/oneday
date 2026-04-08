package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCreateAndGetStoryPreservesAuthoringFields(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "stories.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC().Truncate(time.Second)
	input := &Story{
		ID:               "story-1",
		Name:             "Test Story",
		SettingJSON:      `{"world_name":"Vespera"}`,
		StatsSchemaJSON:  `{"attributes":[]}`,
		Description:      "A test story.",
		Genre:            "fantasy",
		Tone:             "dark",
		Language:         "italiano",
		WritingStyle:     "prosa tesa e controllata",
		PromptDirectives: "Dialoghi asciutti.",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := db.CreateStory(input); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	got, err := db.GetStory(input.ID)
	if err != nil {
		t.Fatalf("GetStory: %v", err)
	}

	if got.Language != input.Language {
		t.Fatalf("Language = %q, want %q", got.Language, input.Language)
	}
	if got.WritingStyle != input.WritingStyle {
		t.Fatalf("WritingStyle = %q, want %q", got.WritingStyle, input.WritingStyle)
	}
	if got.PromptDirectives != input.PromptDirectives {
		t.Fatalf("PromptDirectives = %q, want %q", got.PromptDirectives, input.PromptDirectives)
	}
}
