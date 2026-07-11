package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const CurrentSnapshotFormatVersion = 1

const (
	SnapshotCollectionStory        = "story"
	SnapshotCollectionCharacter    = "character"
	SnapshotCollectionWorld        = "world"
	SnapshotCollectionNPCs         = "npcs"
	SnapshotCollectionAchievements = "achievements"
	SnapshotCollectionChapters     = "chapters"
	SnapshotCollectionSessions     = "sessions"
	SnapshotCollectionChatMessages = "chat_messages"
	SnapshotCollectionRAGChunks    = "rag_chunks"
	SnapshotCollectionCombatLogs   = "combat_logs"
	SnapshotCollectionSessionFiles = "session_files"
	SnapshotCollectionCanonical    = "canonical_state"
)

var requiredSnapshotCollections = []string{
	SnapshotCollectionStory,
	SnapshotCollectionCharacter,
	SnapshotCollectionWorld,
	SnapshotCollectionNPCs,
	SnapshotCollectionAchievements,
	SnapshotCollectionChapters,
	SnapshotCollectionSessions,
	SnapshotCollectionChatMessages,
	SnapshotCollectionRAGChunks,
	SnapshotCollectionCombatLogs,
	SnapshotCollectionSessionFiles,
}

type SnapshotState string

const (
	SnapshotStateFull         SnapshotState = "full"
	SnapshotStateLegacy       SnapshotState = "legacy_partial"
	SnapshotStateIncomplete   SnapshotState = "incomplete"
	SnapshotStateIncompatible SnapshotState = "incompatible"
	SnapshotStateCorrupt      SnapshotState = "corrupt"
)

// SnapshotManifest proves which canonical collections were captured and binds
// the envelope to one story/session/turn identity.
type SnapshotManifest struct {
	StoryID     string         `json:"story_id"`
	SessionID   string         `json:"session_id"`
	Turn        int            `json:"turn"`
	Chapter     int            `json:"chapter"`
	Collections map[string]int `json:"collections"`
}

// SnapshotValidation classifies whether a snapshot may safely drive a full
// rollback. Detail is intentionally safe to surface to callers.
type SnapshotValidation struct {
	State  SnapshotState `json:"state"`
	Detail string        `json:"detail,omitempty"`
}

// SaveSnapshot represents a complete game state snapshot stored in the DB.
type SaveSnapshot struct {
	FormatVersion      int                `json:"format_version,omitempty"`
	Manifest           SnapshotManifest   `json:"manifest,omitempty"`
	PayloadChecksum    string             `json:"payload_checksum,omitempty"`
	ID                 string             `json:"id"`
	StoryID            string             `json:"story_id"`
	Name               string             `json:"name"`
	Turn               int                `json:"turn"`
	Chapter            int                `json:"chapter"`
	Location           string             `json:"location"`
	CharacterJSON      string             `json:"character_json"`
	WorldStateJSON     string             `json:"world_state_json"`
	SessionID          string             `json:"session_id"`
	MetadataJSON       string             `json:"metadata_json,omitempty"`
	Story              *Story             `json:"story,omitempty"`
	NPCs               []NPC              `json:"npcs,omitempty"`
	Achievements       []Achievement      `json:"achievements,omitempty"`
	Chapters           []Chapter          `json:"chapters,omitempty"`
	Sessions           []Session          `json:"sessions,omitempty"`
	ChatMessages       []ChatMessage      `json:"chat_messages,omitempty"`
	RAGChunks          []RAGChunkSnapshot `json:"rag_chunks,omitempty"`
	CombatLogs         []CombatLog        `json:"combat_logs,omitempty"`
	SessionFiles       map[string]string  `json:"session_files,omitempty"`
	CanonicalStateJSON string             `json:"canonical_state_json,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	BranchID           string             `json:"branch_id,omitempty"`
	SourceCommitID     string             `json:"source_commit_id,omitempty"`
}

// RAGChunkSnapshot is a JSON-serializable copy of a persisted RAG chunk.
type RAGChunkSnapshot struct {
	ID             int64     `json:"id"`
	StoryID        string    `json:"story_id"`
	Text           string    `json:"text"`
	ChunkType      string    `json:"chunk_type"`
	TurnStart      int       `json:"turn_start"`
	TurnEnd        int       `json:"turn_end"`
	Embedding      []byte    `json:"embedding"`
	CreatedAt      time.Time `json:"created_at"`
	BranchID       string    `json:"branch_id,omitempty"`
	SourceCommitID string    `json:"source_commit_id,omitempty"`
}

// SaveMetadata captures branch/rewind context for a snapshot.
type SaveMetadata struct {
	Kind                  string   `json:"kind,omitempty"`
	LoadedFromSaveID      string   `json:"loaded_from_save_id,omitempty"`
	LoadedFromSaveName    string   `json:"loaded_from_save_name,omitempty"`
	BranchLabel           string   `json:"branch_label,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
	SnapshotFormatVersion int      `json:"snapshot_format_version,omitempty"`
	BranchID              string   `json:"branch_id,omitempty"`
	CommitID              string   `json:"commit_id,omitempty"`
}

// HasFullRollbackState reports whether the snapshot contains the richer
// canonical state needed for a true rollback instead of just char/world data.
func (s *SaveSnapshot) HasFullRollbackState() bool {
	return s.ValidateRollbackState().State == SnapshotStateFull
}

// SealFullRollback records the current manifest and checksum after verifying
// that the in-memory snapshot is structurally coherent.
func (s *SaveSnapshot) SealFullRollback() error {
	if err := s.validateCanonicalPayload(); err != nil {
		return err
	}

	s.FormatVersion = CurrentSnapshotFormatVersion
	s.Manifest = s.buildManifest()
	checksum, err := s.payloadChecksum()
	if err != nil {
		return fmt.Errorf("computing snapshot checksum: %w", err)
	}
	s.PayloadChecksum = checksum
	return nil
}

// ValidateRollbackState distinguishes verified full snapshots from legacy,
// incomplete, incompatible, and corrupt payloads.
func (s *SaveSnapshot) ValidateRollbackState() SnapshotValidation {
	if s == nil {
		return SnapshotValidation{State: SnapshotStateIncomplete, Detail: "snapshot is missing"}
	}
	if s.FormatVersion == 0 {
		return SnapshotValidation{State: SnapshotStateLegacy, Detail: "snapshot predates the versioned full-rollback format"}
	}
	if s.FormatVersion != CurrentSnapshotFormatVersion {
		return SnapshotValidation{State: SnapshotStateIncompatible, Detail: fmt.Sprintf("snapshot format %d is not supported", s.FormatVersion)}
	}
	if s.PayloadChecksum == "" {
		return SnapshotValidation{State: SnapshotStateIncomplete, Detail: "snapshot checksum is missing"}
	}

	checksum, err := s.payloadChecksum()
	if err != nil {
		return SnapshotValidation{State: SnapshotStateCorrupt, Detail: "snapshot payload cannot be checksummed"}
	}
	if checksum != s.PayloadChecksum {
		return SnapshotValidation{State: SnapshotStateCorrupt, Detail: "snapshot checksum does not match its payload"}
	}
	if err := s.validateCanonicalPayload(); err != nil {
		return SnapshotValidation{State: SnapshotStateIncomplete, Detail: err.Error()}
	}
	if err := s.validateManifest(); err != nil {
		return SnapshotValidation{State: SnapshotStateIncomplete, Detail: err.Error()}
	}
	return SnapshotValidation{State: SnapshotStateFull}
}

func (s *SaveSnapshot) buildManifest() SnapshotManifest {
	return SnapshotManifest{
		StoryID:   s.StoryID,
		SessionID: s.SessionID,
		Turn:      s.Turn,
		Chapter:   s.Chapter,
		Collections: map[string]int{
			SnapshotCollectionStory:        1,
			SnapshotCollectionCharacter:    1,
			SnapshotCollectionWorld:        1,
			SnapshotCollectionNPCs:         len(s.NPCs),
			SnapshotCollectionAchievements: len(s.Achievements),
			SnapshotCollectionChapters:     len(s.Chapters),
			SnapshotCollectionSessions:     len(s.Sessions),
			SnapshotCollectionChatMessages: len(s.ChatMessages),
			SnapshotCollectionRAGChunks:    len(s.RAGChunks),
			SnapshotCollectionCombatLogs:   len(s.CombatLogs),
			SnapshotCollectionSessionFiles: len(s.SessionFiles),
			SnapshotCollectionCanonical:    boolCount(s.CanonicalStateJSON != ""),
		},
	}
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *SaveSnapshot) validateManifest() error {
	want := s.buildManifest()
	if s.Manifest.StoryID != want.StoryID || s.Manifest.SessionID != want.SessionID ||
		s.Manifest.Turn != want.Turn || s.Manifest.Chapter != want.Chapter {
		return fmt.Errorf("snapshot manifest identity does not match its payload")
	}
	if s.Manifest.Collections == nil {
		return fmt.Errorf("snapshot collection manifest is missing")
	}
	for _, name := range requiredSnapshotCollections {
		got, ok := s.Manifest.Collections[name]
		if !ok {
			return fmt.Errorf("snapshot manifest is missing required collection %q", name)
		}
		if got != want.Collections[name] {
			return fmt.Errorf("snapshot manifest count for %q is %d, want %d", name, got, want.Collections[name])
		}
	}
	if s.CanonicalStateJSON != "" {
		got, ok := s.Manifest.Collections[SnapshotCollectionCanonical]
		if !ok || got != 1 {
			return fmt.Errorf("snapshot manifest is missing canonical state")
		}
	}
	return nil
}

func (s *SaveSnapshot) validateCanonicalPayload() error {
	if s.ID == "" || s.StoryID == "" || s.SessionID == "" {
		return fmt.Errorf("snapshot identity is incomplete")
	}
	if s.Story == nil {
		return fmt.Errorf("snapshot story payload is missing")
	}
	if s.Story.ID != s.StoryID {
		return fmt.Errorf("snapshot story identity does not match story payload")
	}

	var character Character
	if s.CharacterJSON == "" || json.Unmarshal([]byte(s.CharacterJSON), &character) != nil {
		return fmt.Errorf("snapshot character payload is missing or invalid")
	}
	if character.StoryID != s.StoryID {
		return fmt.Errorf("snapshot character belongs to a different story")
	}

	var world WorldState
	if s.WorldStateJSON == "" || json.Unmarshal([]byte(s.WorldStateJSON), &world) != nil {
		return fmt.Errorf("snapshot world payload is missing or invalid")
	}
	if world.StoryID != s.StoryID || world.CurrentTurn != s.Turn || world.CurrentChapter != s.Chapter {
		return fmt.Errorf("snapshot world identity or position does not match its envelope")
	}

	for _, npc := range s.NPCs {
		if npc.StoryID != s.StoryID {
			return fmt.Errorf("snapshot npc %q belongs to a different story", npc.ID)
		}
	}
	for _, session := range s.Sessions {
		if session.StoryID != s.StoryID {
			return fmt.Errorf("snapshot session %q belongs to a different story", session.ID)
		}
	}
	for _, message := range s.ChatMessages {
		if message.StoryID != s.StoryID {
			return fmt.Errorf("snapshot message %d belongs to a different story", message.ID)
		}
	}
	for _, chunk := range s.RAGChunks {
		if chunk.StoryID != s.StoryID {
			return fmt.Errorf("snapshot rag chunk %d belongs to a different story", chunk.ID)
		}
	}
	if s.CanonicalStateJSON != "" && !json.Valid([]byte(s.CanonicalStateJSON)) {
		return fmt.Errorf("snapshot canonical state is invalid")
	}
	return nil
}

func (s *SaveSnapshot) payloadChecksum() (string, error) {
	copy := *s
	copy.PayloadChecksum = ""
	payload, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("sha256:%x", sum[:]), nil
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
		`INSERT INTO saves (id, story_id, name, turn, chapter, location, character_json, world_state_json, session_id, metadata_json, created_at, branch_id, source_commit_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.StoryID, s.Name, s.Turn, s.Chapter, s.Location,
		s.CharacterJSON, s.WorldStateJSON, s.SessionID, s.MetadataJSON, s.CreatedAt, s.BranchID, s.SourceCommitID,
	)
	if err != nil {
		return fmt.Errorf("inserting save: %w", err)
	}
	if s.BranchID != "" && s.SourceCommitID != "" {
		if _, err := exec.Exec(`INSERT OR IGNORE INTO save_bookmarks (id,story_id,branch_id,commit_id,save_id,name,created_at) VALUES (?,?,?,?,?,?,?)`, "bookmark-"+s.ID, s.StoryID, s.BranchID, s.SourceCommitID, s.ID, s.Name, s.CreatedAt); err != nil {
			return fmt.Errorf("creating save bookmark: %w", err)
		}
	}
	return nil
}

// GetSave retrieves a save snapshot by ID.
func (db *DB) GetSave(id string) (*SaveSnapshot, error) {
	s := &SaveSnapshot{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, name, turn, chapter, location, character_json, world_state_json, session_id, metadata_json, created_at, branch_id, source_commit_id
         FROM saves WHERE id = ?`, id,
	).Scan(&s.ID, &s.StoryID, &s.Name, &s.Turn, &s.Chapter, &s.Location,
		&s.CharacterJSON, &s.WorldStateJSON, &s.SessionID, &s.MetadataJSON, &s.CreatedAt, &s.BranchID, &s.SourceCommitID)
	if err != nil {
		return nil, fmt.Errorf("getting save %s: %w", id, err)
	}
	return s, nil
}

// ListSaves returns all saves for a story, most recent first.
func (db *DB) ListSaves(storyID string) ([]SaveSnapshot, error) {
	rows, err := db.conn.Query(
		`SELECT id, story_id, name, turn, chapter, location, character_json, world_state_json, session_id, metadata_json, created_at, branch_id, source_commit_id
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
			&s.CharacterJSON, &s.WorldStateJSON, &s.SessionID, &s.MetadataJSON, &s.CreatedAt, &s.BranchID, &s.SourceCommitID); err != nil {
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
		`SELECT id, story_id, name, turn, chapter, location, character_json, world_state_json, session_id, metadata_json, created_at, branch_id, source_commit_id
         FROM saves WHERE story_id = ? AND name = 'autosave'
         ORDER BY created_at DESC LIMIT 1`, storyID,
	).Scan(&s.ID, &s.StoryID, &s.Name, &s.Turn, &s.Chapter, &s.Location,
		&s.CharacterJSON, &s.WorldStateJSON, &s.SessionID, &s.MetadataJSON, &s.CreatedAt, &s.BranchID, &s.SourceCommitID)
	if err != nil {
		return nil, fmt.Errorf("getting autosave for story %s: %w", storyID, err)
	}
	return s, nil
}
