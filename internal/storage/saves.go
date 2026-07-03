package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SaveSnapshot represents a complete game state snapshot stored in the DB.
type SaveSnapshot struct {
	ID             string             `json:"id"`
	StoryID        string             `json:"story_id"`
	Name           string             `json:"name"`
	Turn           int                `json:"turn"`
	Chapter        int                `json:"chapter"`
	Location       string             `json:"location"`
	CharacterJSON  string             `json:"character_json"`
	WorldStateJSON string             `json:"world_state_json"`
	SessionID      string             `json:"session_id"`
	MetadataJSON   string             `json:"metadata_json,omitempty"`
	Story          *Story             `json:"story,omitempty"`
	NPCs           []NPC              `json:"npcs,omitempty"`
	Achievements   []Achievement      `json:"achievements,omitempty"`
	Chapters       []Chapter          `json:"chapters,omitempty"`
	Sessions       []Session          `json:"sessions,omitempty"`
	ChatMessages   []ChatMessage      `json:"chat_messages,omitempty"`
	RAGChunks      []RAGChunkSnapshot `json:"rag_chunks,omitempty"`
	CombatLogs     []CombatLog        `json:"combat_logs,omitempty"`
	SessionFiles   map[string]string  `json:"session_files,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

// RAGChunkSnapshot is a JSON-serializable copy of a persisted RAG chunk.
type RAGChunkSnapshot struct {
	ID        int64     `json:"id"`
	StoryID   string    `json:"story_id"`
	Text      string    `json:"text"`
	ChunkType string    `json:"chunk_type"`
	TurnStart int       `json:"turn_start"`
	TurnEnd   int       `json:"turn_end"`
	Embedding []byte    `json:"embedding"`
	CreatedAt time.Time `json:"created_at"`
}

// SaveMetadata captures branch/rewind context for a snapshot.
type SaveMetadata struct {
	Kind               string   `json:"kind,omitempty"`
	LoadedFromSaveID   string   `json:"loaded_from_save_id,omitempty"`
	LoadedFromSaveName string   `json:"loaded_from_save_name,omitempty"`
	BranchLabel        string   `json:"branch_label,omitempty"`
	Notes              []string `json:"notes,omitempty"`
}

// HasFullRollbackState reports whether the snapshot contains the richer
// canonical state needed for a true rollback instead of just char/world data.
func (s *SaveSnapshot) HasFullRollbackState() bool {
	return s != nil && s.Story != nil
}

// Metadata parses the snapshot metadata payload. Nil means absent or invalid.
func (s SaveSnapshot) Metadata() *SaveMetadata {
	raw := s.MetadataJSON
	if raw == "" || raw == "{}" || raw == "null" {
		return nil
	}

	var meta SaveMetadata
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil
	}
	return &meta
}

// CreateSave inserts a new save snapshot into the DB.
func (db *DB) CreateSave(s *SaveSnapshot) error {
	return createSaveExec(db.conn, s)
}

func (db *DB) CreateSaveTx(tx *sql.Tx, s *SaveSnapshot) error {
	return createSaveExec(tx, s)
}

func createSaveExec(exec sqlExecer, s *SaveSnapshot) error {
	if s.MetadataJSON == "" {
		s.MetadataJSON = "{}"
	}
	_, err := exec.Exec(
		`INSERT INTO saves (id, story_id, name, turn, chapter, location, character_json, world_state_json, session_id, metadata_json, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.StoryID, s.Name, s.Turn, s.Chapter, s.Location,
		s.CharacterJSON, s.WorldStateJSON, s.SessionID, s.MetadataJSON, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting save: %w", err)
	}
	return nil
}

// GetSave retrieves a save snapshot by ID.
func (db *DB) GetSave(id string) (*SaveSnapshot, error) {
	s := &SaveSnapshot{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, name, turn, chapter, location, character_json, world_state_json, session_id, metadata_json, created_at
         FROM saves WHERE id = ?`, id,
	).Scan(&s.ID, &s.StoryID, &s.Name, &s.Turn, &s.Chapter, &s.Location,
		&s.CharacterJSON, &s.WorldStateJSON, &s.SessionID, &s.MetadataJSON, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting save %s: %w", id, err)
	}
	return s, nil
}

// ListSaves returns all saves for a story, most recent first.
func (db *DB) ListSaves(storyID string) ([]SaveSnapshot, error) {
	rows, err := db.conn.Query(
		`SELECT id, story_id, name, turn, chapter, location, character_json, world_state_json, session_id, metadata_json, created_at
         FROM saves WHERE story_id = ? ORDER BY created_at DESC`, storyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing saves for story %s: %w", storyID, err)
	}
	defer rows.Close()

	var saves []SaveSnapshot
	for rows.Next() {
		var s SaveSnapshot
		if err := rows.Scan(&s.ID, &s.StoryID, &s.Name, &s.Turn, &s.Chapter, &s.Location,
			&s.CharacterJSON, &s.WorldStateJSON, &s.SessionID, &s.MetadataJSON, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning save: %w", err)
		}
		saves = append(saves, s)
	}
	return saves, rows.Err()
}

// DeleteSave removes a save by ID.
func (db *DB) DeleteSave(id string) error {
	return deleteSaveExec(db.conn, id)
}

func (db *DB) DeleteSaveTx(tx *sql.Tx, id string) error {
	return deleteSaveExec(tx, id)
}

func deleteSaveExec(exec sqlExecer, id string) error {
	_, err := exec.Exec(`DELETE FROM saves WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting save %s: %w", id, err)
	}
	return nil
}

// GetAutosave retrieves the autosave for a story (if any).
func (db *DB) GetAutosave(storyID string) (*SaveSnapshot, error) {
	s := &SaveSnapshot{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, name, turn, chapter, location, character_json, world_state_json, session_id, metadata_json, created_at
         FROM saves WHERE story_id = ? AND name = 'autosave'
         ORDER BY created_at DESC LIMIT 1`, storyID,
	).Scan(&s.ID, &s.StoryID, &s.Name, &s.Turn, &s.Chapter, &s.Location,
		&s.CharacterJSON, &s.WorldStateJSON, &s.SessionID, &s.MetadataJSON, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting autosave for story %s: %w", storyID, err)
	}
	return s, nil
}
