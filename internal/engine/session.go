package engine

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/game/contracts"
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
	AITTFT      int64       `json:"ai_ttft_ms,omitempty"`
	AIUsage     ai.Usage    `json:"ai_usage,omitempty"`
	AIStreamed  bool        `json:"ai_streamed,omitempty"`
	MessageType string      `json:"message_type,omitempty"` // "narrative", "combat", "crafting", "dialogue", "narrator", "combat_summary"
}

// ChatInput represents the player's input for a turn.
type ChatInput struct {
	Type string `json:"type"` // "free_action", "choice", "command"
	Text string `json:"text"`
}

// ChatOutput represents the AI's output for a turn.
type ChatOutput struct {
	Narrative           string                         `json:"narrative"`
	Choices             []string                       `json:"choices,omitempty"`
	ChoicesData         []Choice                       `json:"choices_data,omitempty"`
	TurnDelta           *TurnDelta                     `json:"turn_delta,omitempty"`
	Mood                string                         `json:"mood,omitempty"`
	Location            string                         `json:"location,omitempty"`
	SceneType           string                         `json:"scene_type,omitempty"`
	DialogueBlocks      []DialogueBlock                `json:"dialogue_blocks,omitempty"`
	EntitiesMentioned   []EntityMention                `json:"entities_mentioned,omitempty"`
	EventCallouts       []EventCallout                 `json:"event_callouts,omitempty"`
	ASCIICue            *ASCIIArtCue                   `json:"ascii_cue,omitempty"`
	ASCIIArt            string                         `json:"ascii_art,omitempty"`
	OpenHooks           []StoryHook                    `json:"open_hooks,omitempty"`
	WorldReactions      []WorldReaction                `json:"world_reactions,omitempty"`
	SocialDuel          *SocialDuelCue                 `json:"social_duel,omitempty"`
	RollLog             []RollRecord                   `json:"roll_log,omitempty"`
	StateChanges        map[string]interface{}         `json:"state_changes,omitempty"`
	ResolvedOutcome     *contracts.OutcomeEnvelope     `json:"resolved_outcome,omitempty"`
	ChallengeInstance   *contracts.ChallengeInstance   `json:"challenge_instance,omitempty"`
	ChallengeResolution *contracts.ChallengeResolution `json:"challenge_resolution,omitempty"`
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

type mirrorSyncError struct {
	path string
	err  error
}

func (e *mirrorSyncError) Error() string {
	return fmt.Sprintf("writing jsonl mirror %s: %v", e.path, e.err)
}

func (e *mirrorSyncError) Unwrap() error {
	return e.err
}

// IsMirrorSyncError reports whether err is a non-fatal JSONL mirror sync failure.
func IsMirrorSyncError(err error) bool {
	var target *mirrorSyncError
	return errors.As(err, &target)
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

	// Restore the turn counter from canonical DB state, not from the JSONL mirror.
	initialTurn, err := db.GetStoryTurnCursor(storyID)
	if err != nil {
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
	committed, err := gs.appendEntry(db, entry)
	if committed {
		gs.turn++
	}
	if err != nil {
		return err
	}
	return nil
}

// AppendHistoryEntry writes an auxiliary canonical entry to the main history
// without advancing the story turn counter. Use this for things like narrator
// meta turns or combat summaries that belong to the current turn.
func (gs *GameSession) AppendHistoryEntry(db *storage.DB, entry ChatEntry) error {
	if entry.Turn < 0 {
		entry.Turn = gs.turn
	}
	_, err := gs.appendEntry(db, entry)
	return err
}

// CommitTurn persists the canonical turn state in one DB transaction and then
// mirrors the result to JSONL. The world must already contain the next turn
// number that should become canonical on success.
func (gs *GameSession) CommitTurn(db *storage.DB, char *storage.Character, world *storage.WorldState, entry ChatEntry) error {
	return gs.CommitTurnWithSideEffects(db, char, world, entry, nil)
}

func (gs *GameSession) CommitTurnWithSideEffects(
	db *storage.DB,
	char *storage.Character,
	world *storage.WorldState,
	entry ChatEntry,
	beforeCommit func(*sql.Tx) error,
) error {
	if db == nil {
		return fmt.Errorf("committing turn: db is nil")
	}
	if char == nil {
		return fmt.Errorf("committing turn: character is nil")
	}
	if world == nil {
		return fmt.Errorf("committing turn: world is nil")
	}

	entry.Turn = gs.turn
	expectedNextTurn := entry.Turn + 1
	if world.CurrentTurn != expectedNextTurn {
		return fmt.Errorf("committing turn %d: world current turn %d does not match expected next turn %d", entry.Turn, world.CurrentTurn, expectedNextTurn)
	}

	committed, err := gs.commitTurn(db, char, world, entry, beforeCommit)
	if committed {
		gs.turn = expectedNextTurn
	}
	return err
}

// CloseMirrors flushes and closes the JSONL mirror files without ending the
// canonical DB session. This is useful for short-lived in-process clients that
// reopen the active session on each request.
func (gs *GameSession) CloseMirrors() error {
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

	return closeErr
}

// Close flushes and closes the JSONL file, all sub-session files, and marks the session as ended in DB.
func (gs *GameSession) Close(db *storage.DB) error {
	closeErr := gs.CloseMirrors()

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
	if turn >= 0 {
		gs.turn = turn
	}
}

func (gs *GameSession) appendEntry(db *storage.DB, entry ChatEntry) (bool, error) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	if err := db.WithTx(func(tx *sql.Tx) error {
		return gs.appendEntryToDB(tx, db, entry)
	}); err != nil {
		return false, err
	}

	if err := gs.writeJSONLEntry(entry); err != nil {
		return true, &mirrorSyncError{path: gs.mainJSONLPath(), err: err}
	}

	return true, nil
}

func (gs *GameSession) commitTurn(db *storage.DB, char *storage.Character, world *storage.WorldState, entry ChatEntry, beforeCommit func(*sql.Tx) error) (bool, error) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	if err := db.WithTx(func(tx *sql.Tx) error {
		head, err := db.EnsureStoryTimelineTx(tx, gs.storyID)
		if err != nil {
			return fmt.Errorf("loading active timeline head: %w", err)
		}
		parentPayload, err := db.CaptureTimelineMaterializationTx(tx, gs.storyID, head.Branch.ID)
		if err != nil {
			return fmt.Errorf("capturing parent timeline state: %w", err)
		}
		if err := db.SealTurnSnapshotTx(tx, head.Commit.ID, gs.storyID, parentPayload); err != nil {
			return fmt.Errorf("sealing parent timeline state: %w", err)
		}
		if beforeCommit != nil {
			if err := beforeCommit(tx); err != nil {
				return err
			}
		}
		if err := db.UpdateCharacterFullTx(tx, char); err != nil {
			return fmt.Errorf("saving character state: %w", err)
		}
		if err := db.UpdateWorldStateExpectedTurnTx(tx, world, entry.Turn); err != nil {
			return fmt.Errorf("saving world state: %w", err)
		}
		if entry.Input != nil {
			command := strings.ToLower(strings.TrimSpace(entry.Input.Text))
			delta := 0
			reason := ""
			if strings.HasPrefix(command, "/timeskip") {
				delta = 1440
				reason = "timeskip"
			} else if strings.HasPrefix(command, "/downtime") {
				delta = 480
				reason = "downtime"
			}
			if delta > 0 {
				if _, err := db.AdvanceWorldTimeTx(tx, gs.storyID, reason, delta, world.CurrentTurn); err != nil {
					return fmt.Errorf("advancing canonical world time: %w", err)
				}
			}
		}
		if err := gs.appendEntryToDB(tx, db, entry); err != nil {
			return err
		}
		commitID := uuid.NewString()
		if err := db.BindPendingLineageTx(tx, gs.storyID, head.Branch.ID, commitID); err != nil {
			return err
		}
		payload, err := db.CaptureTimelineMaterializationTx(tx, gs.storyID, head.Branch.ID)
		if err != nil {
			return fmt.Errorf("capturing committed timeline state: %w", err)
		}
		eventPayload, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("encoding committed turn event: %w", err)
		}
		_, err = db.AppendTurnCommitTx(tx, storage.AppendTurnCommitParams{
			CommitID:       commitID,
			StoryID:        gs.storyID,
			BranchID:       head.Branch.ID,
			ExpectedHeadID: head.Commit.ID,
			CanonicalTurn:  world.CurrentTurn,
			Kind:           "turn",
			Message:        entry.MessageType,
			PayloadJSON:    payload,
			Events: []storage.CanonicalEventInput{{
				Type:        "turn.committed",
				PayloadJSON: string(eventPayload),
			}},
		})
		return err
	}); err != nil {
		return false, err
	}

	if err := gs.writeJSONLEntry(entry); err != nil {
		return true, &mirrorSyncError{path: gs.mainJSONLPath(), err: err}
	}

	return true, nil
}

func (gs *GameSession) appendEntryToDB(tx *sql.Tx, db *storage.DB, entry ChatEntry) error {
	msgType := entry.MessageType
	if msgType == "" {
		msgType = "narrative"
	}

	now := entry.Timestamp
	if entry.Input != nil {
		userMsg := &storage.ChatMessage{
			SessionID:    gs.session.ID,
			StoryID:      gs.storyID,
			Turn:         entry.Turn,
			Role:         "user",
			Content:      entry.Input.Text,
			MessageType:  msgType,
			MetadataJSON: "{}",
			CreatedAt:    now,
		}
		if err := db.AppendChatMessageTx(tx, userMsg); err != nil {
			return fmt.Errorf("saving user message to db: %w", err)
		}
	}

	if entry.Output == nil {
		return nil
	}

	meta := map[string]interface{}{
		"model":                  entry.AIModel,
		"latency_ms":             entry.AILatency,
		"time_to_first_token_ms": entry.AITTFT,
		"usage":                  entry.AIUsage,
		"streamed":               entry.AIStreamed,
		"mood":                   entry.Output.Mood,
		"location":               entry.Output.Location,
		"choices":                entry.Output.Choices,
		"output":                 entry.Output,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		metaJSON = []byte("{}")
	}
	assistantMsg := &storage.ChatMessage{
		SessionID:    gs.session.ID,
		StoryID:      gs.storyID,
		Turn:         entry.Turn,
		Role:         "assistant",
		Content:      entry.Output.Narrative,
		MessageType:  msgType,
		MetadataJSON: string(metaJSON),
		CreatedAt:    now,
	}
	if err := db.AppendChatMessageTx(tx, assistantMsg); err != nil {
		return fmt.Errorf("saving assistant message to db: %w", err)
	}

	return nil
}

func (gs *GameSession) writeJSONLEntry(entry ChatEntry) error {
	if gs.jsonlFile == nil {
		return fmt.Errorf("main jsonl file is closed")
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling chat entry: %w", err)
	}
	if _, err := gs.jsonlFile.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing to jsonl: %w", err)
	}
	return nil
}

func (gs *GameSession) mainJSONLPath() string {
	if gs == nil || gs.jsonlFile == nil {
		return ""
	}
	return gs.jsonlFile.Name()
}
