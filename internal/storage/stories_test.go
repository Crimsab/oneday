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

func TestWorldStatePreservesSceneContractJSON(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "world.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC().Truncate(time.Second)
	story := &Story{ID: "story-scene-contract", Name: "Scene Contract", CreatedAt: now, UpdatedAt: now}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	world := &WorldState{
		ID:                "world-scene-contract",
		StoryID:           story.ID,
		CurrentLocation:   "Harbor",
		CurrentChapter:    1,
		SceneContractJSON: `{"required_delta":"location_change"}`,
		UpdatedAt:         now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}
	got, err := db.GetWorldState(story.ID)
	if err != nil {
		t.Fatalf("GetWorldState: %v", err)
	}
	if got.SceneContractJSON != world.SceneContractJSON {
		t.Fatalf("SceneContractJSON = %q, want %q", got.SceneContractJSON, world.SceneContractJSON)
	}

	got.SceneContractJSON = `{"required_delta":"front_pressure"}`
	if err := db.UpdateWorldState(got); err != nil {
		t.Fatalf("UpdateWorldState: %v", err)
	}
	updated, err := db.GetWorldState(story.ID)
	if err != nil {
		t.Fatalf("GetWorldState updated: %v", err)
	}
	if updated.SceneContractJSON != got.SceneContractJSON {
		t.Fatalf("updated SceneContractJSON = %q, want %q", updated.SceneContractJSON, got.SceneContractJSON)
	}
}
