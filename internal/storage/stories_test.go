package storage

import (
	"database/sql"
	"errors"
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

func TestUpdateWorldStateExpectedTurnTxRejectsStaleWriter(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "world-cas.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC().Truncate(time.Second)
	story := &Story{ID: "story-world-cas", Name: "World CAS", CreatedAt: now, UpdatedAt: now}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	world := &WorldState{
		ID:              "world-cas",
		StoryID:         story.ID,
		CurrentLocation: "Harbor",
		CurrentChapter:  1,
		CurrentTurn:     2,
		UpdatedAt:       now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	world.CurrentLocation = "Market"
	world.CurrentTurn = 3
	if err := db.WithTx(func(tx *sql.Tx) error {
		return db.UpdateWorldStateExpectedTurnTx(tx, world, 2)
	}); err != nil {
		t.Fatalf("UpdateWorldStateExpectedTurnTx success: %v", err)
	}
	world.CurrentLocation = "Overwritten Road"
	world.CurrentTurn = 4
	err = db.WithTx(func(tx *sql.Tx) error {
		return db.UpdateWorldStateExpectedTurnTx(tx, world, 2)
	})
	if !errors.Is(err, ErrStaleWorldTurn) {
		t.Fatalf("stale update error = %v, want ErrStaleWorldTurn", err)
	}
	got, err := db.GetWorldState(story.ID)
	if err != nil {
		t.Fatalf("GetWorldState: %v", err)
	}
	if got.CurrentTurn != 3 || got.CurrentLocation != "Market" {
		t.Fatalf("world after stale update = turn %d location %q, want turn 3 Market", got.CurrentTurn, got.CurrentLocation)
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
