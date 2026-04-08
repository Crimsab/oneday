package engine

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/storage"
)

// SaveGame creates a full state snapshot to disk and DB.
// saveName should be "autosave" for automatic saves or a user-provided name.
func SaveGame(
	db *storage.DB,
	dataDir string,
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	sessionID string,
	saveName string,
) (*storage.SaveSnapshot, error) {
	if saveName == "" {
		saveName = "save"
	}

	// Serialize character and world state to JSON.
	charJSON, err := json.Marshal(char)
	if err != nil {
		return nil, fmt.Errorf("serializing character: %w", err)
	}

	worldJSON, err := json.Marshal(world)
	if err != nil {
		return nil, fmt.Errorf("serializing world state: %w", err)
	}

	saveID := uuid.New().String()
	now := time.Now()

	snap := &storage.SaveSnapshot{
		ID:             saveID,
		StoryID:        story.ID,
		Name:           saveName,
		Turn:           world.CurrentTurn,
		Chapter:        world.CurrentChapter,
		Location:       world.CurrentLocation,
		CharacterJSON:  string(charJSON),
		WorldStateJSON: string(worldJSON),
		SessionID:      sessionID,
		CreatedAt:      now,
	}

	// Write snapshot to disk.
	saveDir := filepath.Join(dataDir, "stories", story.ID, "saves")
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, fmt.Errorf("creating saves directory: %w", err)
	}

	snapPath := filepath.Join(saveDir, saveID+".json")
	snapBytes, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("marshaling snapshot: %w", err)
	}
	if err := os.WriteFile(snapPath, snapBytes, 0644); err != nil {
		return nil, fmt.Errorf("writing snapshot file: %w", err)
	}

	// Insert into DB.
	if err := db.CreateSave(snap); err != nil {
		// Best-effort: remove the file if DB insert fails.
		_ = os.Remove(snapPath)
		return nil, fmt.Errorf("saving to database: %w", err)
	}

	return snap, nil
}

// LoadGame restores game state from a save snapshot.
// Returns the restored character and world state.
// The caller is responsible for updating the narrator with the restored objects.
func LoadGame(
	db *storage.DB,
	saveID string,
) (*storage.Character, *storage.WorldState, error) {
	snap, err := db.GetSave(saveID)
	if err != nil {
		return nil, nil, fmt.Errorf("retrieving save %s: %w", saveID, err)
	}

	var char storage.Character
	if err := json.Unmarshal([]byte(snap.CharacterJSON), &char); err != nil {
		return nil, nil, fmt.Errorf("deserializing character: %w", err)
	}

	var world storage.WorldState
	if err := json.Unmarshal([]byte(snap.WorldStateJSON), &world); err != nil {
		return nil, nil, fmt.Errorf("deserializing world state: %w", err)
	}

	// Persist the fully restored state to DB — use UpdateCharacterFull so
	// traits_json, skills_json, inventory_json, and known_recipes_json are all
	// written back, not just stats_json.
	if err := db.UpdateCharacterFull(&char); err != nil {
		return nil, nil, fmt.Errorf("restoring character state: %w", err)
	}
	if err := db.UpdateWorldState(&world); err != nil {
		return nil, nil, fmt.Errorf("restoring world state: %w", err)
	}

	return &char, &world, nil
}

// ListSaves returns all saves for a story, most recent first.
func ListSaves(db *storage.DB, storyID string) ([]storage.SaveSnapshot, error) {
	return db.ListSaves(storyID)
}

// Autosave creates or overwrites the autosave for a story.
// It deletes any previous autosave before creating the new one.
func Autosave(
	db *storage.DB,
	dataDir string,
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	sessionID string,
) error {
	// Check if an autosave already exists for this story.
	existing, err := db.GetAutosave(story.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// If the error is not "no rows", it's unexpected — log and continue.
		existing = nil
	}

	if existing != nil {
		// Delete the old autosave file and DB row.
		_ = db.DeleteSave(existing.ID)
	}

	_, err = SaveGame(db, dataDir, story, char, world, sessionID, "autosave")
	return err
}
