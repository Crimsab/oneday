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

	"github.com/crimsab/oneday/internal/storage"
)

// LoadResult is the outcome of loading a save snapshot.
type LoadResult struct {
	Character      *storage.Character
	World          *storage.WorldState
	Save           *storage.SaveSnapshot
	Legacy         bool
	SnapshotState  storage.SnapshotState
	SnapshotDetail string
}

// SnapshotLoadError reports a safe, typed reason why loading was refused
// before any canonical state mutation occurred.
type SnapshotLoadError struct {
	State  storage.SnapshotState
	Detail string
}

func (e *SnapshotLoadError) Error() string {
	if e == nil {
		return "save snapshot load failed"
	}
	if e.Detail == "" {
		return fmt.Sprintf("save snapshot is %s", e.State)
	}
	return fmt.Sprintf("save snapshot is %s: %s", e.State, e.Detail)
}

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
	return SaveGameWithMetadata(db, dataDir, story, char, world, sessionID, saveName, nil)
}

// SaveGameWithMetadata creates a full state snapshot and persists optional rewind metadata.
func SaveGameWithMetadata(
	db *storage.DB,
	dataDir string,
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	sessionID string,
	saveName string,
	meta *storage.SaveMetadata,
) (*storage.SaveSnapshot, error) {
	if saveName == "" {
		saveName = "save"
	}
	head, err := db.GetActiveTimeline(story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading active timeline for save: %w", err)
	}
	canonicalStateJSON, err := db.CaptureCanonicalState(story.ID, head.Branch.ID)
	if err != nil {
		return nil, fmt.Errorf("capturing canonical entity state: %w", err)
	}

	charJSON, err := json.Marshal(char)
	if err != nil {
		return nil, fmt.Errorf("serializing character: %w", err)
	}

	worldJSON, err := json.Marshal(world)
	if err != nil {
		return nil, fmt.Errorf("serializing world state: %w", err)
	}

	npcs, err := db.ListNPCs(story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading npcs for save: %w", err)
	}

	achievements, err := db.ListAchievements(story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading achievements for save: %w", err)
	}

	chapters, err := db.ListChapters(story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading chapters for save: %w", err)
	}

	sessions, err := db.ListSessions(story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading sessions for save: %w", err)
	}

	chatMessages, err := db.GetStoryMessages(story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading chat messages for save: %w", err)
	}

	ragChunks, err := listRAGChunks(db, story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading rag chunks for save: %w", err)
	}

	combatLogs, err := listCombatLogs(db, story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading combat logs for save: %w", err)
	}

	sessionFiles, err := snapshotSessionFiles(dataDir, story.ID)
	if err != nil {
		return nil, fmt.Errorf("capturing session files for save: %w", err)
	}

	storyCopy := *story
	saveID := uuid.New().String()
	now := time.Now()
	saveMeta := storage.SaveMetadata{SnapshotFormatVersion: storage.CurrentSnapshotFormatVersion}
	if meta != nil {
		saveMeta = *meta
		saveMeta.SnapshotFormatVersion = storage.CurrentSnapshotFormatVersion
	}
	saveMeta.BranchID = head.Branch.ID
	saveMeta.CommitID = head.Commit.ID
	metaJSON := "{}"
	if payload, err := json.Marshal(&saveMeta); err == nil {
		metaJSON = string(payload)
	}

	snap := &storage.SaveSnapshot{
		ID:                 saveID,
		StoryID:            story.ID,
		Name:               saveName,
		Turn:               world.CurrentTurn,
		Chapter:            world.CurrentChapter,
		Location:           world.CurrentLocation,
		CharacterJSON:      string(charJSON),
		WorldStateJSON:     string(worldJSON),
		SessionID:          sessionID,
		MetadataJSON:       metaJSON,
		Story:              &storyCopy,
		NPCs:               npcs,
		Achievements:       achievements,
		Chapters:           chapters,
		Sessions:           sessions,
		ChatMessages:       chatMessages,
		RAGChunks:          ragChunks,
		CombatLogs:         combatLogs,
		SessionFiles:       sessionFiles,
		CanonicalStateJSON: canonicalStateJSON,
		CreatedAt:          now,
		BranchID:           head.Branch.ID,
		SourceCommitID:     head.Commit.ID,
	}
	if err := snap.SealFullRollback(); err != nil {
		return nil, fmt.Errorf("sealing full rollback snapshot: %w", err)
	}

	snapPath := saveFilePath(dataDir, story.ID, saveID)
	if err := os.MkdirAll(filepath.Dir(snapPath), 0755); err != nil {
		return nil, fmt.Errorf("creating saves directory: %w", err)
	}

	snapBytes, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("marshaling snapshot: %w", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(snapPath), "."+saveID+"-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("creating temporary snapshot file: %w", err)
	}
	tmpPath := tmpFile.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmpFile.Write(snapBytes); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("writing temporary snapshot file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("closing temporary snapshot file: %w", err)
	}
	if err := os.Rename(tmpPath, snapPath); err != nil {
		return nil, fmt.Errorf("publishing snapshot file: %w", err)
	}
	cleanupTmp = false

	var committedRevision int64
	if err := db.WithTx(func(tx *sql.Tx) error {
		currentHead, err := db.EnsureStoryTimelineTx(tx, story.ID)
		if err != nil {
			return err
		}
		if currentHead.Branch.ID != head.Branch.ID || currentHead.Commit.ID != head.Commit.ID {
			return storage.ErrStaleBranchHead
		}
		if err := db.EnsureTurnSnapshotTx(tx, head.Commit.ID, story.ID, head.Branch.ID); err != nil {
			return err
		}
		if err := db.CreateSaveTx(tx, snap); err != nil {
			return err
		}
		nextRevision, err := db.BumpStoryRevisionTx(tx, story.ID)
		if err != nil {
			return err
		}
		committedRevision = nextRevision
		return nil
	}); err != nil {
		_ = os.Remove(snapPath)
		return nil, fmt.Errorf("saving to database: %w", err)
	}
	if committedRevision > 0 {
		story.Revision = committedRevision
	}

	return snap, nil
}

// LoadGame restores game state from a save snapshot.
func LoadGame(
	db *storage.DB,
	dataDir string,
	saveID string,
) (*LoadResult, error) {
	snap, err := loadSaveSnapshot(db, dataDir, saveID)
	if err != nil {
		return nil, err
	}

	var char storage.Character
	if err := json.Unmarshal([]byte(snap.CharacterJSON), &char); err != nil {
		return nil, fmt.Errorf("deserializing character: %w", err)
	}

	var world storage.WorldState
	if err := json.Unmarshal([]byte(snap.WorldStateJSON), &world); err != nil {
		return nil, fmt.Errorf("deserializing world state: %w", err)
	}

	validation := snap.ValidateRollbackState()
	if validation.State != storage.SnapshotStateFull && validation.State != storage.SnapshotStateLegacy {
		return nil, &SnapshotLoadError{State: validation.State, Detail: validation.Detail}
	}
	result := &LoadResult{
		Character:      &char,
		World:          &world,
		Save:           snap,
		Legacy:         validation.State == storage.SnapshotStateLegacy,
		SnapshotState:  validation.State,
		SnapshotDetail: validation.Detail,
	}

	if result.Legacy {
		if err := db.WithTx(func(tx *sql.Tx) error {
			if err := db.UpdateCharacterFullTx(tx, &char); err != nil {
				return fmt.Errorf("restoring character state: %w", err)
			}
			if err := db.UpdateWorldStateTx(tx, &world); err != nil {
				return fmt.Errorf("restoring world state: %w", err)
			}
			if err := db.ClearTurnIdempotencyTx(tx, snap.StoryID); err != nil {
				return err
			}
			_, err := db.BumpStoryRevisionTx(tx, snap.StoryID)
			return err
		}); err != nil {
			return nil, err
		}
		return result, nil
	}

	if err := restoreFullRollback(db, dataDir, snap, &char, &world); err != nil {
		return nil, err
	}

	return result, nil
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
	return AutosaveWithMetadata(db, dataDir, story, char, world, sessionID, nil)
}

func AutosaveWithMetadata(
	db *storage.DB,
	dataDir string,
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	sessionID string,
	meta *storage.SaveMetadata,
) error {
	existing, err := db.GetAutosave(story.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		existing = nil
	}

	_, err = SaveGameWithMetadata(db, dataDir, story, char, world, sessionID, "autosave", meta)
	if err != nil {
		return err
	}
	if existing != nil {
		return DeleteSave(db, dataDir, existing.ID)
	}
	return nil
}

// DeleteSave removes a save snapshot from both the DB and the on-disk save directory.
func DeleteSave(db *storage.DB, dataDir, saveID string) error {
	snap, err := db.GetSave(saveID)
	if err != nil {
		return fmt.Errorf("getting save %s: %w", saveID, err)
	}

	if err := db.WithTx(func(tx *sql.Tx) error {
		if err := db.DeleteSaveTx(tx, saveID); err != nil {
			return err
		}
		_, err := db.BumpStoryRevisionTx(tx, snap.StoryID)
		return err
	}); err != nil {
		return err
	}

	snapPath := saveFilePath(dataDir, snap.StoryID, snap.ID)
	if err := os.Remove(snapPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing save file %s: %w", snapPath, err)
	}
	return nil
}

// SetStoryArchived toggles whether a story appears in the active story list.
func SetStoryArchived(db *storage.DB, storyID string, archived bool) error {
	return db.SetStoryArchived(storyID, archived)
}

// DeleteStory removes a story from the DB and deletes its local data directory.
func DeleteStory(db *storage.DB, dataDir, storyID string) error {
	if err := db.DeleteStory(storyID); err != nil {
		return err
	}

	storyDir := filepath.Join(dataDir, "stories", storyID)
	if err := os.RemoveAll(storyDir); err != nil {
		return fmt.Errorf("removing story directory %s: %w", storyDir, err)
	}
	return nil
}

func loadSaveSnapshot(db *storage.DB, dataDir, saveID string) (*storage.SaveSnapshot, error) {
	snap, err := db.GetSave(saveID)
	if err != nil {
		return nil, fmt.Errorf("retrieving save %s: %w", saveID, err)
	}

	snapPath := saveFilePath(dataDir, snap.StoryID, snap.ID)
	diskBytes, err := os.ReadFile(snapPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if meta := snap.Metadata(); meta != nil && meta.SnapshotFormatVersion > 0 {
				return nil, &SnapshotLoadError{
					State:  storage.SnapshotStateIncomplete,
					Detail: "the versioned snapshot file is missing",
				}
			}
			return snap, nil
		}
		return nil, &SnapshotLoadError{State: storage.SnapshotStateIncomplete, Detail: "the snapshot file cannot be read"}
	}

	var diskSnap storage.SaveSnapshot
	if err := json.Unmarshal(diskBytes, &diskSnap); err != nil {
		return nil, &SnapshotLoadError{State: storage.SnapshotStateCorrupt, Detail: "the snapshot file is not valid JSON"}
	}

	if diskSnap.FormatVersion > 0 {
		validation := diskSnap.ValidateRollbackState()
		if validation.State != storage.SnapshotStateFull {
			return nil, &SnapshotLoadError{State: validation.State, Detail: validation.Detail}
		}
		if diskSnap.ID != snap.ID || diskSnap.StoryID != snap.StoryID {
			return nil, &SnapshotLoadError{State: storage.SnapshotStateCorrupt, Detail: "the snapshot file identity does not match the selected save"}
		}
		return &diskSnap, nil
	}

	if diskSnap.ID == "" {
		diskSnap.ID = snap.ID
	}
	if diskSnap.StoryID == "" {
		diskSnap.StoryID = snap.StoryID
	}
	if diskSnap.Name == "" {
		diskSnap.Name = snap.Name
	}
	if diskSnap.CharacterJSON == "" {
		diskSnap.CharacterJSON = snap.CharacterJSON
	}
	if diskSnap.WorldStateJSON == "" {
		diskSnap.WorldStateJSON = snap.WorldStateJSON
	}
	if diskSnap.SessionID == "" {
		diskSnap.SessionID = snap.SessionID
	}
	if diskSnap.MetadataJSON == "" {
		diskSnap.MetadataJSON = snap.MetadataJSON
	}
	if diskSnap.CreatedAt.IsZero() {
		diskSnap.CreatedAt = snap.CreatedAt
	}
	if diskSnap.Turn == 0 {
		diskSnap.Turn = snap.Turn
	}
	if diskSnap.Chapter == 0 {
		diskSnap.Chapter = snap.Chapter
	}
	if diskSnap.Location == "" {
		diskSnap.Location = snap.Location
	}

	return &diskSnap, nil
}

func restoreFullRollback(
	db *storage.DB,
	dataDir string,
	snap *storage.SaveSnapshot,
	char *storage.Character,
	world *storage.WorldState,
) error {
	fileStage, err := prepareSessionRestore(dataDir, snap.StoryID, snap.SessionFiles)
	if err != nil {
		return fmt.Errorf("staging session files: %w", err)
	}
	defer fileStage.cleanup()

	tx, err := db.Conn().Begin()
	if err != nil {
		return fmt.Errorf("starting rollback transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := upsertStoryRow(tx, snap.Story); err != nil {
		return fmt.Errorf("restoring story state: %w", err)
	}
	if err := db.ClearTurnIdempotencyTx(tx, snap.StoryID); err != nil {
		return err
	}

	deleteStatements := []string{
		`DELETE FROM chat_messages WHERE story_id = ?`,
		`DELETE FROM combat_log WHERE story_id = ?`,
		`DELETE FROM rag_chunks WHERE story_id = ?`,
		`DELETE FROM achievements WHERE story_id = ?`,
		`DELETE FROM chapters WHERE story_id = ?`,
		`DELETE FROM npcs WHERE story_id = ?`,
		`DELETE FROM sessions WHERE story_id = ?`,
		`DELETE FROM world_state WHERE story_id = ?`,
		`DELETE FROM characters WHERE story_id = ?`,
	}
	for _, stmt := range deleteStatements {
		if _, err := tx.Exec(stmt, snap.StoryID); err != nil {
			return fmt.Errorf("clearing story state: %w", err)
		}
	}

	if err := insertCharacterRow(tx, char); err != nil {
		return fmt.Errorf("restoring character row: %w", err)
	}
	if err := insertWorldStateRow(tx, world); err != nil {
		return fmt.Errorf("restoring world state row: %w", err)
	}
	if err := insertNPCRows(tx, snap.NPCs); err != nil {
		return fmt.Errorf("restoring npcs: %w", err)
	}
	if err := insertChapterRows(tx, snap.Chapters); err != nil {
		return fmt.Errorf("restoring chapters: %w", err)
	}
	if err := insertAchievementRows(tx, snap.Achievements); err != nil {
		return fmt.Errorf("restoring achievements: %w", err)
	}
	if err := insertSessionRows(tx, snap.Sessions); err != nil {
		return fmt.Errorf("restoring sessions: %w", err)
	}
	if err := insertChatMessageRows(tx, snap.ChatMessages); err != nil {
		return fmt.Errorf("restoring chat messages: %w", err)
	}
	if err := insertRAGChunkRows(tx, snap.RAGChunks); err != nil {
		return fmt.Errorf("restoring rag chunks: %w", err)
	}
	if err := insertCombatLogRows(tx, snap.CombatLogs); err != nil {
		return fmt.Errorf("restoring combat logs: %w", err)
	}
	if snap.CanonicalStateJSON != "" {
		branchID := snap.BranchID
		if branchID == "" && snap.Story != nil {
			branchID = snap.Story.ActiveBranchID
		}
		if err := db.RestoreCanonicalStateTx(tx, snap.StoryID, branchID, snap.CanonicalStateJSON); err != nil {
			return fmt.Errorf("restoring canonical entity state: %w", err)
		}
	} else if err := db.RebuildCompatibilityCanonTx(tx, snap.StoryID, snap.NPCs, char); err != nil {
		return fmt.Errorf("rebuilding legacy canonical entity state: %w", err)
	}
	if _, err := db.BumpStoryRevisionTx(tx, snap.StoryID); err != nil {
		return err
	}

	if err := fileStage.activate(); err != nil {
		return fmt.Errorf("activating staged session files: %w", err)
	}

	if err := tx.Commit(); err != nil {
		if restoreErr := fileStage.rollback(); restoreErr != nil {
			return fmt.Errorf("committing rollback transaction: %w (session-file compensation also failed: %v)", err, restoreErr)
		}
		return fmt.Errorf("committing rollback transaction: %w", err)
	}
	fileStage.finalize()

	return nil
}

func upsertStoryRow(tx *sql.Tx, story *storage.Story) error {
	if story == nil {
		return fmt.Errorf("snapshot missing story payload")
	}

	_, err := tx.Exec(
		`INSERT INTO stories (
			id, name, setting_json, stats_schema_json, description, genre, tone,
			language, writing_style, prompt_directives, revision, is_archived, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			setting_json = excluded.setting_json,
			stats_schema_json = excluded.stats_schema_json,
			description = excluded.description,
			genre = excluded.genre,
			tone = excluded.tone,
			language = excluded.language,
			writing_style = excluded.writing_style,
			prompt_directives = excluded.prompt_directives,
			revision = stories.revision,
			is_archived = excluded.is_archived,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at`,
		story.ID, story.Name, story.SettingJSON, story.StatsSchemaJSON,
		story.Description, story.Genre, story.Tone, story.Language,
		story.WritingStyle, story.PromptDirectives, story.Revision, story.IsArchived,
		story.CreatedAt, story.UpdatedAt,
	)
	return err
}

func insertCharacterRow(tx *sql.Tx, char *storage.Character) error {
	_, err := tx.Exec(
		`INSERT INTO characters (
			id, story_id, name, background, stats_json, traits_json,
			skills_json, inventory_json, known_recipes_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		char.ID, char.StoryID, char.Name, char.Background, char.StatsJSON,
		char.TraitsJSON, char.SkillsJSON, char.InventoryJSON, char.KnownRecipesJSON,
		char.CreatedAt, char.UpdatedAt,
	)
	return err
}

func insertWorldStateRow(tx *sql.Tx, world *storage.WorldState) error {
	_, err := tx.Exec(
		`INSERT INTO world_state (
			id, story_id, current_location, current_location_id, known_locations_json,
			global_events_json, faction_standings_json, story_hooks_json, world_reactions_json,
			investigation_board_json, project_clocks_json, player_guidance_json, fronts_json, character_timeline_json, scene_contract_json, current_chapter, current_turn, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		world.ID, world.StoryID, world.CurrentLocation, world.CurrentLocationID, world.KnownLocationsJSON,
		world.GlobalEventsJSON, world.FactionStandingsJSON, world.StoryHooksJSON,
		world.WorldReactionsJSON, world.InvestigationBoardJSON, world.ProjectClocksJSON, world.PlayerGuidanceJSON, world.FrontsJSON, world.CharacterTimelineJSON, world.SceneContractJSON, world.CurrentChapter, world.CurrentTurn, world.UpdatedAt,
	)
	return err
}

func insertNPCRows(tx *sql.Tx, npcs []storage.NPC) error {
	for _, npc := range npcs {
		if _, err := tx.Exec(
			`INSERT INTO npcs (
				id, story_id, canonical_entity_id, name, role, appearance, personality_json, private_thoughts,
				relationship_json, nemesis_json, discovery_json, notes_on_protagonist, desires, disposition, is_alive,
				first_appeared_turn, last_seen_turn, can_help, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			npc.ID, npc.StoryID, npc.CanonicalEntityID, npc.Name, npc.Role, npc.Appearance, npc.PersonalityJSON,
			npc.PrivateThoughts, npc.RelationshipJSON, npc.NemesisJSON, npc.DiscoveryJSON, npc.NotesOnProtagonist, npc.Desires, npc.Disposition,
			npc.IsAlive, npc.FirstAppearedTurn, npc.LastSeenTurn, npc.CanHelp,
			npc.CreatedAt, npc.UpdatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertChapterRows(tx *sql.Tx, chapters []storage.Chapter) error {
	var maxID int64
	for _, chapter := range chapters {
		if chapter.ID > maxID {
			maxID = chapter.ID
		}
		if _, err := tx.Exec(
			`INSERT INTO chapters (
				id, story_id, chapter_number, title, summary, start_turn, end_turn, created_at, branch_id, source_commit_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			chapter.ID, chapter.StoryID, chapter.ChapterNumber, chapter.Title,
			chapter.Summary, chapter.StartTurn, chapter.EndTurn, chapter.CreatedAt, chapter.BranchID, chapter.SourceCommitID,
		); err != nil {
			return err
		}
	}
	return setSQLiteSequence(tx, "chapters", maxID)
}

func insertAchievementRows(tx *sql.Tx, achievements []storage.Achievement) error {
	var maxID int64
	for _, achievement := range achievements {
		if achievement.ID > maxID {
			maxID = achievement.ID
		}
		if _, err := tx.Exec(
			`INSERT INTO achievements (
				id, story_id, name, description, category, rarity, context, earned_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			achievement.ID, achievement.StoryID, achievement.Name, achievement.Description,
			achievement.Category, achievement.Rarity, achievement.Context, achievement.EarnedAt,
		); err != nil {
			return err
		}
	}
	return setSQLiteSequence(tx, "achievements", maxID)
}

func insertSessionRows(tx *sql.Tx, sessions []storage.Session) error {
	for _, session := range sessions {
		if _, err := tx.Exec(
			`INSERT INTO sessions (id, story_id, started_at, ended_at, summary, branch_id, source_commit_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			session.ID, session.StoryID, session.StartedAt, session.EndedAt, session.Summary, session.BranchID, session.SourceCommitID,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertChatMessageRows(tx *sql.Tx, messages []storage.ChatMessage) error {
	var maxID int64
	for _, message := range messages {
		if message.ID > maxID {
			maxID = message.ID
		}
		if _, err := tx.Exec(
			`INSERT INTO chat_messages (
				id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at, branch_id, source_commit_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			message.ID, message.SessionID, message.StoryID, message.Turn, message.Role,
			message.Content, message.MessageType, message.MetadataJSON, message.CreatedAt, message.BranchID, message.SourceCommitID,
		); err != nil {
			return err
		}
	}
	return setSQLiteSequence(tx, "chat_messages", maxID)
}

func insertRAGChunkRows(tx *sql.Tx, chunks []storage.RAGChunkSnapshot) error {
	var maxID int64
	for _, chunk := range chunks {
		if chunk.ID > maxID {
			maxID = chunk.ID
		}
		if _, err := tx.Exec(
			`INSERT INTO rag_chunks (
				id, story_id, text, chunk_type, turn_start, turn_end, embedding, created_at, branch_id, source_commit_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			chunk.ID, chunk.StoryID, chunk.Text, chunk.ChunkType,
			chunk.TurnStart, chunk.TurnEnd, chunk.Embedding, chunk.CreatedAt, chunk.BranchID, chunk.SourceCommitID,
		); err != nil {
			return err
		}
	}
	return setSQLiteSequence(tx, "rag_chunks", maxID)
}

func insertCombatLogRows(tx *sql.Tx, logs []storage.CombatLog) error {
	var maxID int64
	for _, logRow := range logs {
		if logRow.ID > maxID {
			maxID = logRow.ID
		}
		if _, err := tx.Exec(
			`INSERT INTO combat_log (
				id, story_id, session_id, enemy_name, enemy_hp, turns, victory,
				defeat_outcome, player_hp_start, player_hp_end, created_at, branch_id, source_commit_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			logRow.ID, logRow.StoryID, logRow.SessionID, logRow.EnemyName,
			logRow.EnemyHP, logRow.Turns, logRow.Victory, logRow.DefeatOutcome,
			logRow.PlayerHPStart, logRow.PlayerHPEnd, logRow.CreatedAt, logRow.BranchID, logRow.SourceCommitID,
		); err != nil {
			return err
		}
	}
	return setSQLiteSequence(tx, "combat_log", maxID)
}

func setSQLiteSequence(tx *sql.Tx, table string, seq int64) error {
	if _, err := tx.Exec(`DELETE FROM sqlite_sequence WHERE name = ?`, table); err != nil {
		return err
	}
	if seq == 0 {
		return nil
	}
	_, err := tx.Exec(`INSERT INTO sqlite_sequence(name, seq) VALUES (?, ?)`, table, seq)
	return err
}

func listRAGChunks(db *storage.DB, storyID string) ([]storage.RAGChunkSnapshot, error) {
	rows, err := db.Conn().Query(
		`SELECT id, story_id, text, chunk_type, turn_start, turn_end, embedding, created_at, branch_id, source_commit_id
		 FROM rag_chunks WHERE story_id = ? ORDER BY id ASC`,
		storyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []storage.RAGChunkSnapshot
	for rows.Next() {
		var chunk storage.RAGChunkSnapshot
		if err := rows.Scan(
			&chunk.ID, &chunk.StoryID, &chunk.Text, &chunk.ChunkType,
			&chunk.TurnStart, &chunk.TurnEnd, &chunk.Embedding, &chunk.CreatedAt,
			&chunk.BranchID, &chunk.SourceCommitID,
		); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, rows.Err()
}

func listCombatLogs(db *storage.DB, storyID string) ([]storage.CombatLog, error) {
	rows, err := db.Conn().Query(
		`SELECT id, story_id, session_id, enemy_name, enemy_hp, turns, victory,
		        defeat_outcome, player_hp_start, player_hp_end, created_at, branch_id, source_commit_id
		 FROM combat_log WHERE story_id = ? ORDER BY id ASC`,
		storyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []storage.CombatLog
	for rows.Next() {
		var logRow storage.CombatLog
		if err := rows.Scan(
			&logRow.ID, &logRow.StoryID, &logRow.SessionID, &logRow.EnemyName,
			&logRow.EnemyHP, &logRow.Turns, &logRow.Victory, &logRow.DefeatOutcome,
			&logRow.PlayerHPStart, &logRow.PlayerHPEnd, &logRow.CreatedAt,
			&logRow.BranchID, &logRow.SourceCommitID,
		); err != nil {
			return nil, err
		}
		logs = append(logs, logRow)
	}
	return logs, rows.Err()
}

func snapshotSessionFiles(dataDir, storyID string) (map[string]string, error) {
	root := filepath.Join(dataDir, "stories", storyID, "sessions")
	files := map[string]string{}

	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return files, nil
		}
		return nil, err
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(content)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

type sessionRestoreStage struct {
	root        string
	staged      string
	backup      string
	hadOriginal bool
	active      bool
}

func prepareSessionRestore(dataDir, storyID string, files map[string]string) (*sessionRestoreStage, error) {
	root := filepath.Join(dataDir, "stories", storyID, "sessions")
	for rel := range files {
		cleanRel := filepath.Clean(filepath.FromSlash(rel))
		if cleanRel == "." || filepath.IsAbs(cleanRel) || cleanRel == ".." ||
			strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("invalid session file path %q in snapshot", rel)
		}
	}

	storyRoot := filepath.Dir(root)
	if err := os.MkdirAll(storyRoot, 0755); err != nil {
		return nil, err
	}
	staged, err := os.MkdirTemp(storyRoot, ".sessions-restore-*")
	if err != nil {
		return nil, err
	}
	stage := &sessionRestoreStage{root: root, staged: staged}
	cleanup := true
	defer func() {
		if cleanup {
			stage.cleanup()
		}
	}()

	for rel, content := range files {
		cleanRel := filepath.Clean(filepath.FromSlash(rel))
		fullPath := filepath.Join(staged, cleanRel)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}
	cleanup = false
	return stage, nil
}

func (s *sessionRestoreStage) activate() error {
	if s == nil || s.staged == "" {
		return fmt.Errorf("session restore stage is not prepared")
	}
	if _, err := os.Stat(s.root); err == nil {
		s.hadOriginal = true
		s.backup = s.root + ".backup-" + uuid.NewString()
		if err := os.Rename(s.root, s.backup); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.Rename(s.staged, s.root); err != nil {
		if s.hadOriginal {
			_ = os.Rename(s.backup, s.root)
		}
		return err
	}
	s.staged = ""
	s.active = true
	return nil
}

func (s *sessionRestoreStage) rollback() error {
	if s == nil || !s.active {
		return nil
	}
	if err := os.RemoveAll(s.root); err != nil {
		return err
	}
	if s.hadOriginal {
		if err := os.Rename(s.backup, s.root); err != nil {
			return err
		}
	}
	s.active = false
	s.backup = ""
	return nil
}

func (s *sessionRestoreStage) finalize() {
	if s == nil {
		return
	}
	if s.backup != "" {
		_ = os.RemoveAll(s.backup)
		s.backup = ""
	}
	s.active = false
}

func (s *sessionRestoreStage) cleanup() {
	if s == nil {
		return
	}
	if s.staged != "" {
		_ = os.RemoveAll(s.staged)
		s.staged = ""
	}
}

func saveFilePath(dataDir, storyID, saveID string) string {
	return filepath.Join(dataDir, "stories", storyID, "saves", saveID+".json")
}
