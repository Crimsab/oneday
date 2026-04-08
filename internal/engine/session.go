package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/storage"
)

// ChatEntry is the JSONL format for a single turn on disk.
type ChatEntry struct {
	Turn        int         `json:"turn"`
	Timestamp   time.Time   `json:"timestamp"`
	Chapter     int         `json:"chapter"`
	Location    string      `json:"location"`
	Input       *ChatInput  `json:"input,omitempty"`
	Output      *ChatOutput `json:"output,omitempty"`
	AIModel     string      `json:"ai_model,omitempty"`
	AILatency   int64       `json:"ai_latency_ms,omitempty"`
	MessageType string      `json:"message_type,omitempty"` // "narrative", "combat", "crafting", "dialogue", "narrator"
}

// ChatInput represents the player's input for a turn.
type ChatInput struct {
	Type string `json:"type"` // "free_action", "choice", "command"
	Text string `json:"text"`
}

// ChatOutput represents the AI's output for a turn.
type ChatOutput struct {
	Narrative    string                 `json:"narrative"`
	Choices      []string               `json:"choices,omitempty"`
	Mood         string                 `json:"mood,omitempty"`
	StateChanges map[string]interface{} `json:"state_changes,omitempty"`
}

// GameSession manages a play session's lifecycle and JSONL persistence.
type GameSession struct {
	session     *storage.Session
	storyID     string
	dataDir     string
	jsonlFile   *os.File
	subFiles    map[string]*os.File // "combat_1" -> file handle
	subCounters map[string]int      // "combat" -> counter for naming
	turn        int
}

// NewGameSession creates or resumes a session.
// If an active session exists in DB, it resumes it. Otherwise creates a new one.
func NewGameSession(db *storage.DB, storyID, dataDir string) (*GameSession, error) {
	// Try to resume an existing open session.
	existing, err := db.GetActiveSession(storyID)
	if err != nil {
		return nil, fmt.Errorf("checking active session: %w", err)
	}

	var sess *storage.Session
	if existing != nil {
		sess = existing
	} else {
		// Create a new session.
		now := time.Now()
		sess = &storage.Session{
			ID:        uuid.New().String(),
			StoryID:   storyID,
			StartedAt: now,
		}
		if err := db.CreateSession(sess); err != nil {
			return nil, fmt.Errorf("creating session: %w", err)
		}
	}

	// Create the session directory.
	sessionDir := filepath.Join(dataDir, "stories", storyID, "sessions", sess.ID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("creating session directory: %w", err)
	}

	// Open (or create) the JSONL file in append mode.
	jsonlPath := filepath.Join(sessionDir, "main.jsonl")
	f, err := os.OpenFile(jsonlPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening jsonl file: %w", err)
	}

	// Count existing lines to restore turn counter for resumed sessions.
	initialTurn, err := countJSONLLines(jsonlPath)
	if err != nil {
		// Non-fatal: start from 0 if we can't read.
		initialTurn = 0
	}

	gs := &GameSession{
		session:     sess,
		storyID:     storyID,
		dataDir:     dataDir,
		jsonlFile:   f,
		subFiles:    make(map[string]*os.File),
		subCounters: make(map[string]int),
		turn:        initialTurn,
	}
	return gs, nil
}

// OpenSubSession creates a new sub-session JSONL file for combat, crafting, or dialogue.
// sessionType is "combat", "crafting", or "dialogue".
// Returns the sub-session ID (e.g., "combat_1").
func (gs *GameSession) OpenSubSession(sessionType string) (string, error) {
	gs.subCounters[sessionType]++
	subID := fmt.Sprintf("%s_%d", sessionType, gs.subCounters[sessionType])

	sessionDir := filepath.Join(gs.dataDir, "stories", gs.storyID, "sessions", gs.session.ID)
	subPath := filepath.Join(sessionDir, subID+".jsonl")

	f, err := os.OpenFile(subPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("opening sub-session file %s: %w", subPath, err)
	}
	gs.subFiles[subID] = f
	return subID, nil
}

// AppendSubTurn writes a ChatEntry to a specific sub-session JSONL file.
func (gs *GameSession) AppendSubTurn(subSessionID string, entry ChatEntry) error {
	f, ok := gs.subFiles[subSessionID]
	if !ok {
		return fmt.Errorf("sub-session %s not open", subSessionID)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling sub-session entry: %w", err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing to sub-session %s: %w", subSessionID, err)
	}
	return nil
}

// CloseSubSession closes a specific sub-session file.
func (gs *GameSession) CloseSubSession(subSessionID string) error {
	f, ok := gs.subFiles[subSessionID]
	if !ok {
		return nil // already closed or never opened
	}
	delete(gs.subFiles, subSessionID)
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing sub-session %s: %w", subSessionID, err)
	}
	return nil
}

// AppendTurn writes a ChatEntry to the JSONL file AND inserts into DB.
// The entry's Turn field is set automatically from the internal counter.
func (gs *GameSession) AppendTurn(db *storage.DB, entry ChatEntry) error {
	entry.Turn = gs.turn
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// 1. Write to JSONL file.
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling chat entry: %w", err)
	}
	if _, err := gs.jsonlFile.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing to jsonl: %w", err)
	}

	// Determine message type (default to "narrative").
	msgType := entry.MessageType
	if msgType == "" {
		msgType = "narrative"
	}

	// 2. Persist to DB — split into user and assistant messages.
	now := entry.Timestamp
	if entry.Input != nil {
		userMsg := &storage.ChatMessage{
			SessionID:    gs.session.ID,
			StoryID:      gs.storyID,
			Turn:         gs.turn,
			Role:         "user",
			Content:      entry.Input.Text,
			MessageType:  msgType,
			MetadataJSON: "{}",
			CreatedAt:    now,
		}
		if err := db.AppendChatMessage(userMsg); err != nil {
			return fmt.Errorf("saving user message to db: %w", err)
		}
	}

	if entry.Output != nil {
		// Build metadata JSON for the assistant message.
		meta := map[string]interface{}{
			"model":      entry.AIModel,
			"latency_ms": entry.AILatency,
			"mood":       entry.Output.Mood,
			"location":   entry.Location,
			"choices":    entry.Output.Choices,
		}
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			metaJSON = []byte("{}")
		}
		assistantMsg := &storage.ChatMessage{
			SessionID:    gs.session.ID,
			StoryID:      gs.storyID,
			Turn:         gs.turn,
			Role:         "assistant",
			Content:      entry.Output.Narrative,
			MessageType:  msgType,
			MetadataJSON: string(metaJSON),
			CreatedAt:    now,
		}
		if err := db.AppendChatMessage(assistantMsg); err != nil {
			return fmt.Errorf("saving assistant message to db: %w", err)
		}
	}

	gs.turn++
	return nil
}

// Close flushes and closes the JSONL file, all sub-session files, and marks the session as ended in DB.
func (gs *GameSession) Close(db *storage.DB) error {
	var closeErr error

	// Close all open sub-session files.
	for subID, f := range gs.subFiles {
		if err := f.Close(); err != nil && closeErr == nil {
			closeErr = fmt.Errorf("closing sub-session %s: %w", subID, err)
		}
	}
	gs.subFiles = make(map[string]*os.File)

	if gs.jsonlFile != nil {
		if err := gs.jsonlFile.Close(); err != nil {
			closeErr = fmt.Errorf("closing jsonl file: %w", err)
		}
		gs.jsonlFile = nil
	}

	if db != nil {
		if err := db.CloseSession(gs.session.ID); err != nil {
			// Return JSONL close error first if both fail, otherwise return this one.
			if closeErr != nil {
				return closeErr
			}
			return fmt.Errorf("closing session in db: %w", err)
		}
	}

	return closeErr
}

// SessionID returns the session ID.
func (gs *GameSession) SessionID() string {
	return gs.session.ID
}

// Turn returns the current turn number.
func (gs *GameSession) Turn() int {
	return gs.turn
}

// SetTurn overrides the internal turn counter. Used when resuming a story to
// align the session counter with the persisted world.CurrentTurn from the DB.
func (gs *GameSession) SetTurn(turn int) {
	if turn > 0 {
		gs.turn = turn
	}
}

// countJSONLLines counts non-empty lines in a file.
// Used to restore the turn counter when resuming a session.
func countJSONLLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("opening jsonl for counting: %w", err)
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			count++
		}
	}
	return count, scanner.Err()
}
