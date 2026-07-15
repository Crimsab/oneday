package storage

import (
	"database/sql"
	"fmt"
)

// migrate runs all schema migrations in order.
func (db *DB) migrate() error {
	// Create migrations tracking table
	if _, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("creating schema_version table: %w", err)
	}

	migrations := []struct {
		version int
		sql     string
	}{
		{1, migrationV1},
		{2, migrationV2},
		{3, migrationV3},
		{4, migrationV4},
		{5, migrationV5},
		{6, migrationV6},
		{7, migrationV7},
		{8, migrationV8},
		{9, migrationV9},
		{10, migrationV10},
		{11, migrationV11},
		{12, migrationV12},
		{13, migrationV13},
		{14, migrationV14},
		{15, migrationV15},
		{16, migrationV16},
		{17, migrationV17},
		{18, migrationV18},
		{19, migrationV19},
		{20, migrationV20},
		{21, migrationV21},
		{22, migrationV22},
		{23, migrationV23},
		{24, migrationV24},
		{25, migrationV25},
		{26, migrationV26},
		{27, migrationV27},
		{28, migrationV28},
		{29, migrationV29},
		{30, migrationV30},
		{31, migrationV31},
		{32, migrationV32},
		{33, migrationV33},
		{34, migrationV34},
		{35, migrationV35},
		{36, migrationV36},
		{37, migrationV37},
		{38, migrationV38},
		{39, migrationV39},
		{40, migrationV40},
		{41, migrationV41},
		{42, migrationV42},
	}

	for _, m := range migrations {
		var count int
		err := db.conn.QueryRow("SELECT COUNT(*) FROM schema_version WHERE version = ?", m.version).Scan(&count)
		if err != nil {
			return fmt.Errorf("checking migration %d: %w", m.version, err)
		}
		if count > 0 {
			continue
		}

		if err := db.applyMigration(m.version, m.sql); err != nil {
			return fmt.Errorf("applying migration %d: %w", m.version, err)
		}
		if _, err := db.conn.Exec("INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
			return fmt.Errorf("recording migration %d: %w", m.version, err)
		}
	}
	return nil
}

func (db *DB) applyMigration(version int, migrationSQL string) error {
	switch version {
	case 21:
		return db.applyMigrationV21()
	case 22:
		return db.applyMigrationV22()
	case 26:
		return db.applyMigrationV26()
	case 27:
		return db.applyMigrationV27()
	case 29:
		return db.applyMigrationV29()
	case 30:
		return db.applyMigrationV30()
	case 33:
		return db.applyMigrationV33()
	case 36:
		return db.applyMigrationV36()
	case 39:
		return db.applyMigrationV39()
	default:
		_, err := db.conn.Exec(migrationSQL)
		return err
	}
}

func (db *DB) applyMigrationV39() error {
	exists, err := db.columnExists("rag_chunks", "embedding_norm")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = db.conn.Exec(migrationV39)
	return err
}

func (db *DB) applyMigrationV36() error {
	for _, item := range []struct {
		table      string
		column     string
		definition string
	}{
		{"regions", "region_kind", "region_kind TEXT NOT NULL DEFAULT 'region'"},
		{"locations", "location_kind", "location_kind TEXT NOT NULL DEFAULT 'place'"},
		{"location_edges", "travel_mode", "travel_mode TEXT NOT NULL DEFAULT 'travel'"},
		{"location_edges", "bidirectional", "bidirectional INTEGER NOT NULL DEFAULT 0 CHECK(bidirectional IN (0,1))"},
		{"visual_assets", "map_scope_kind", "map_scope_kind TEXT NOT NULL DEFAULT ''"},
		{"visual_assets", "map_scope_id", "map_scope_id TEXT NOT NULL DEFAULT ''"},
		{"visual_asset_versions", "map_scope_kind", "map_scope_kind TEXT NOT NULL DEFAULT ''"},
		{"visual_asset_versions", "map_scope_id", "map_scope_id TEXT NOT NULL DEFAULT ''"},
		{"visual_generation_jobs", "map_scope_kind", "map_scope_kind TEXT NOT NULL DEFAULT ''"},
		{"visual_generation_jobs", "map_scope_id", "map_scope_id TEXT NOT NULL DEFAULT ''"},
	} {
		if err := db.addColumnIfMissing(item.table, item.column, item.definition); err != nil {
			return err
		}
	}
	_, err := db.conn.Exec(migrationV36)
	return err
}

func (db *DB) applyMigrationV33() (err error) {
	for _, item := range []struct {
		table      string
		column     string
		definition string
	}{
		{"visual_asset_versions", "canonical_entity_id", "canonical_entity_id TEXT NOT NULL DEFAULT ''"},
		{"visual_asset_versions", "canonical_location_id", "canonical_location_id TEXT NOT NULL DEFAULT ''"},
		{"visual_asset_versions", "form_id", "form_id TEXT NOT NULL DEFAULT ''"},
		{"visual_asset_versions", "appearance_fingerprint", "appearance_fingerprint TEXT NOT NULL DEFAULT ''"},
		{"visual_asset_versions", "profile_revision_id", "profile_revision_id TEXT"},
		{"visual_asset_versions", "canon_status", "canon_status TEXT NOT NULL DEFAULT 'draft'"},
		{"visual_generation_jobs", "canonical_entity_id", "canonical_entity_id TEXT NOT NULL DEFAULT ''"},
		{"visual_generation_jobs", "canonical_location_id", "canonical_location_id TEXT NOT NULL DEFAULT ''"},
		{"visual_generation_jobs", "form_id", "form_id TEXT NOT NULL DEFAULT ''"},
		{"visual_generation_jobs", "appearance_fingerprint", "appearance_fingerprint TEXT NOT NULL DEFAULT ''"},
		{"visual_generation_jobs", "profile_revision_id", "profile_revision_id TEXT"},
	} {
		if err := db.addColumnIfMissing(item.table, item.column, item.definition); err != nil {
			return err
		}
	}

	if _, err = db.conn.Exec(migrationV33Prelude); err != nil {
		return err
	}
	if _, err = db.conn.Exec("PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}
	defer func() {
		if _, enableErr := db.conn.Exec("PRAGMA foreign_keys=ON"); err == nil && enableErr != nil {
			err = enableErr
		}
	}()

	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(migrationV33AssetRebuild); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if _, err = db.conn.Exec(migrationV33Backfill); err != nil {
		return err
	}

	rows, err := db.conn.Query("PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		var table string
		var rowID sql.NullInt64
		var parent string
		var foreignKeyID int
		if scanErr := rows.Scan(&table, &rowID, &parent, &foreignKeyID); scanErr != nil {
			return scanErr
		}
		return fmt.Errorf("visual canon migration foreign key violation: table=%s row=%v parent=%s fk=%d", table, rowID, parent, foreignKeyID)
	}
	return rows.Err()
}

func (db *DB) applyMigrationV30() error {
	if err := db.addColumnIfMissing("world_state", "current_location_id", "current_location_id TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if _, err := db.conn.Exec(migrationV30); err != nil {
		return err
	}
	_, err := db.conn.Exec(`
	INSERT OR IGNORE INTO locations (id,story_id,canonical_name,discovery_state,discovered_turn,branch_id,source_commit_id)
	SELECT 'loc-'||lower(hex(randomblob(16))),w.story_id,w.current_location,'discovered',w.current_turn,s.active_branch_id,b.head_commit_id FROM world_state w JOIN stories s ON s.id=w.story_id JOIN story_branches b ON b.id=s.active_branch_id WHERE trim(w.current_location)!='';
	UPDATE world_state SET current_location_id=COALESCE((SELECT l.id FROM locations l WHERE l.story_id=world_state.story_id AND lower(l.canonical_name)=lower(world_state.current_location) LIMIT 1),'') WHERE current_location_id='';
	INSERT OR IGNORE INTO world_calendars (story_id,name,config_json) SELECT id,'Default calendar','{"hours_per_day":24,"minutes_per_hour":60}' FROM stories;
	INSERT OR IGNORE INTO world_clocks (story_id,calendar_story_id,day,minute_of_day,display_text,branch_id,source_commit_id) SELECT s.id,s.id,0,0,'Day 0, 00:00',s.active_branch_id,b.head_commit_id FROM stories s JOIN story_branches b ON b.id=s.active_branch_id;
	`)
	return err
}

func (db *DB) applyMigrationV29() error {
	if err := db.addColumnIfMissing("npcs", "canonical_entity_id", "canonical_entity_id TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if _, err := db.conn.Exec(migrationV29); err != nil {
		return err
	}
	_, err := db.conn.Exec(`
		INSERT OR IGNORE INTO canonical_entities (id,story_id,entity_kind,canonical_name,lifecycle_status,profile_json,branch_id,source_commit_id,created_at,updated_at)
		SELECT n.id,n.story_id,'character',n.name,CASE WHEN n.is_alive=1 THEN 'active' ELSE 'dead' END,
		       json_object('compatibility_projection','npcs'),s.active_branch_id,b.head_commit_id,n.created_at,n.updated_at
		FROM npcs n JOIN stories s ON s.id=n.story_id JOIN story_branches b ON b.id=s.active_branch_id;
		INSERT OR IGNORE INTO canonical_entities (id,story_id,entity_kind,canonical_name,lifecycle_status,profile_json,branch_id,source_commit_id,created_at,updated_at)
		SELECT c.id,c.story_id,'protagonist',c.name,'active',json_object('compatibility_projection','characters'),s.active_branch_id,b.head_commit_id,c.created_at,c.updated_at
		FROM characters c JOIN stories s ON s.id=c.story_id JOIN story_branches b ON b.id=s.active_branch_id;
		INSERT OR IGNORE INTO entity_aliases (id,story_id,entity_id,alias,alias_kind,visibility,branch_id,source_commit_id,created_at)
		SELECT 'alias-'||lower(hex(randomblob(16))),n.story_id,n.id,n.name,'display','player',s.active_branch_id,b.head_commit_id,n.created_at
		FROM npcs n JOIN stories s ON s.id=n.story_id JOIN story_branches b ON b.id=s.active_branch_id;
		INSERT OR IGNORE INTO identity_claims (id,story_id,subject_entity_id,claimed_entity_id,label,claim_kind,status,confidence,visibility,learned_turn,branch_id,source_commit_id,created_at)
		SELECT 'identity-'||lower(hex(randomblob(16))),n.story_id,n.id,n.id,n.name,'self','confirmed',1.0,'player',n.first_appeared_turn,s.active_branch_id,b.head_commit_id,n.created_at
		FROM npcs n JOIN stories s ON s.id=n.story_id JOIN story_branches b ON b.id=s.active_branch_id
		WHERE NOT EXISTS (SELECT 1 FROM identity_claims i WHERE i.subject_entity_id=n.id AND i.label=n.name);
		INSERT OR IGNORE INTO entity_forms (id,story_id,entity_id,name,form_kind,appearance_json,valid_from_turn,branch_id,source_commit_id,created_at)
		SELECT 'form-'||n.id,n.story_id,n.id,n.name||' — base form','base',json_object('description',n.appearance),n.first_appeared_turn,s.active_branch_id,b.head_commit_id,n.created_at
		FROM npcs n JOIN stories s ON s.id=n.story_id JOIN story_branches b ON b.id=s.active_branch_id;
		UPDATE npcs SET canonical_entity_id=id WHERE canonical_entity_id='';
	`)
	return err
}

func (db *DB) applyMigrationV27() error {
	lineageColumns := []struct {
		table      string
		column     string
		definition string
	}{
		{"stories", "active_branch_id", "active_branch_id TEXT NOT NULL DEFAULT ''"},
		{"sessions", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"sessions", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
		{"chat_messages", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"chat_messages", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
		{"chapters", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"chapters", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
		{"rag_chunks", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"rag_chunks", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
		{"saves", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"saves", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
		{"combat_log", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"combat_log", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
		{"visual_assets", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"visual_assets", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
		{"visual_asset_versions", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"visual_asset_versions", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
		{"visual_generation_jobs", "branch_id", "branch_id TEXT NOT NULL DEFAULT ''"},
		{"visual_generation_jobs", "source_commit_id", "source_commit_id TEXT NOT NULL DEFAULT ''"},
	}
	for _, item := range lineageColumns {
		if err := db.addColumnIfMissing(item.table, item.column, item.definition); err != nil {
			return err
		}
	}
	if _, err := db.conn.Exec(migrationV27); err != nil {
		return err
	}
	return db.backfillStoryTimelines()
}

func (db *DB) backfillStoryTimelines() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO story_branches (id, story_id, name, created_at, updated_at)
		SELECT 'branch-' || lower(hex(randomblob(16))), s.id, 'main', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		FROM stories s
		WHERE NOT EXISTS (SELECT 1 FROM story_branches b WHERE b.story_id = s.id)
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO turn_commits (id, story_id, branch_id, parent_commit_id, canonical_turn, story_revision, payload_hash, kind, created_at)
		SELECT 'commit-' || lower(hex(randomblob(16))), s.id, b.id, NULL,
		       COALESCE(w.current_turn, 0), s.revision, '', 'legacy_root', CURRENT_TIMESTAMP
		FROM stories s
		JOIN story_branches b ON b.story_id = s.id
		LEFT JOIN world_state w ON w.story_id = s.id
		WHERE b.head_commit_id IS NULL
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE story_branches
		SET head_commit_id = (SELECT c.id FROM turn_commits c WHERE c.branch_id = story_branches.id ORDER BY c.created_at, c.id LIMIT 1),
		    updated_at = CURRENT_TIMESTAMP
		WHERE head_commit_id IS NULL
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE stories
		SET active_branch_id = (SELECT b.id FROM story_branches b WHERE b.story_id = stories.id ORDER BY b.created_at, b.id LIMIT 1)
		WHERE active_branch_id = ''
	`); err != nil {
		return err
	}

	for _, table := range []string{"sessions", "chat_messages", "chapters", "rag_chunks", "saves", "combat_log", "visual_assets", "visual_asset_versions", "visual_generation_jobs"} {
		stmt := fmt.Sprintf(`UPDATE %s
			SET branch_id = COALESCE((SELECT active_branch_id FROM stories WHERE stories.id = %s.story_id), '')
			WHERE branch_id = ''`, table, table)
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
		stmt = fmt.Sprintf(`UPDATE %s
			SET source_commit_id = COALESCE((SELECT b.head_commit_id FROM story_branches b WHERE b.id = %s.branch_id), '')
			WHERE source_commit_id = ''`, table, table)
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO save_bookmarks (id, story_id, branch_id, commit_id, save_id, name, created_at)
		SELECT 'bookmark-' || id, story_id, branch_id, source_commit_id, id, name, created_at
		FROM saves WHERE branch_id != '' AND source_commit_id != ''
	`); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) applyMigrationV21() error {
	if err := db.addColumnIfMissing("turn_idempotency", "status", "status TEXT NOT NULL DEFAULT 'committed'"); err != nil {
		return err
	}
	if err := db.addColumnIfMissing("turn_idempotency", "owner", "owner TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := db.addColumnIfMissing("turn_idempotency", "locked_until", "locked_until TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := db.addColumnIfMissing("turn_idempotency", "updated_at", "updated_at TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := db.addColumnIfMissing("turn_idempotency", "error", "error TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	_, err := db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_turn_idempotency_status_locked_until
		ON turn_idempotency(status, locked_until)`)
	return err
}

func (db *DB) applyMigrationV22() error {
	if err := db.addColumnIfMissing("stories", "revision", "revision INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := db.addColumnIfMissing("turn_idempotency", "request_hash", "request_hash TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	_, err := db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_turn_idempotency_request_hash
		ON turn_idempotency(story_id, idempotency_key, request_hash)`)
	return err
}

func (db *DB) applyMigrationV26() error {
	if err := db.addColumnIfMissing("npcs", "discovery_json", "discovery_json TEXT NOT NULL DEFAULT '{}'"); err != nil {
		return err
	}
	_, err := db.conn.Exec(`
		UPDATE npcs
		SET discovery_json = CASE
			WHEN LOWER(role) = 'person of interest'
				OR appearance LIKE '%Unidentified figure%'
				OR personality_json LIKE '%unknown%'
			THEN '{"stage":"identified","source":"migration","confidence":"inferred","profile_completeness":25,"visual_completeness":0,"visual_readiness":"none"}'
			WHEN TRIM(appearance) != ''
			THEN '{"stage":"established","source":"migration","confidence":"confirmed","profile_completeness":80,"visual_completeness":70,"visual_readiness":"canonical"}'
			ELSE '{"stage":"identified","source":"migration","confidence":"confirmed","profile_completeness":60,"visual_completeness":20,"visual_readiness":"none"}'
		END
		WHERE discovery_json = '' OR discovery_json = '{}';
		CREATE INDEX IF NOT EXISTS idx_npcs_story_last_seen_alive
			ON npcs(story_id, is_alive, last_seen_turn DESC);
	`)
	return err
}

func (db *DB) addColumnIfMissing(table, column, definition string) error {
	exists, err := db.columnExists(table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = db.conn.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, definition))
	return err
}

func (db *DB) columnExists(table, column string) (bool, error) {
	rows, err := db.conn.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

const migrationV1 = `
-- Stories table: core story metadata and settings (maps to story.json)
CREATE TABLE stories (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	setting_json TEXT NOT NULL DEFAULT '{}',
	stats_schema_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Characters table: player protagonists
CREATE TABLE characters (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	background TEXT NOT NULL DEFAULT '',
	stats_json TEXT NOT NULL DEFAULT '{}',
	traits_json TEXT NOT NULL DEFAULT '[]',
	skills_json TEXT NOT NULL DEFAULT '[]',
	inventory_json TEXT NOT NULL DEFAULT '[]',
	known_recipes_json TEXT NOT NULL DEFAULT '[]',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- NPCs table: AI-generated non-player characters
CREATE TABLE npcs (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT '',
	personality_json TEXT NOT NULL DEFAULT '{}',
	private_thoughts TEXT NOT NULL DEFAULT '',
	desires TEXT NOT NULL DEFAULT '',
	disposition INTEGER NOT NULL DEFAULT 0,
	is_alive INTEGER NOT NULL DEFAULT 1,
	first_appeared_turn INTEGER NOT NULL DEFAULT 0,
	can_help INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- World state table: global state per story
CREATE TABLE world_state (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL UNIQUE REFERENCES stories(id) ON DELETE CASCADE,
	current_location TEXT NOT NULL DEFAULT '',
	known_locations_json TEXT NOT NULL DEFAULT '[]',
	global_events_json TEXT NOT NULL DEFAULT '[]',
	faction_standings_json TEXT NOT NULL DEFAULT '{}',
	current_chapter INTEGER NOT NULL DEFAULT 1,
	current_turn INTEGER NOT NULL DEFAULT 0,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table: play sessions
CREATE TABLE sessions (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	ended_at DATETIME,
	summary TEXT NOT NULL DEFAULT ''
);

-- Chat messages table: individual messages in sessions
CREATE TABLE chat_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	turn INTEGER NOT NULL DEFAULT 0,
	role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system', 'narrator')),
	content TEXT NOT NULL,
	message_type TEXT NOT NULL DEFAULT 'narrative' CHECK(message_type IN ('narrative', 'combat', 'crafting', 'dialogue')),
	metadata_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Chapters table: AI-generated chapter summaries
CREATE TABLE chapters (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	chapter_number INTEGER NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	summary TEXT NOT NULL DEFAULT '',
	start_turn INTEGER NOT NULL,
	end_turn INTEGER,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id, chapter_number)
);

-- Achievements table
CREATE TABLE achievements (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	category TEXT NOT NULL DEFAULT 'story',
	rarity TEXT NOT NULL DEFAULT 'common',
	context TEXT NOT NULL DEFAULT '',
	earned_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries
CREATE INDEX idx_characters_story ON characters(story_id);
CREATE INDEX idx_npcs_story ON npcs(story_id);
CREATE INDEX idx_sessions_story ON sessions(story_id);
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX idx_chat_messages_story ON chat_messages(story_id);
CREATE INDEX idx_chapters_story ON chapters(story_id);
CREATE INDEX idx_achievements_story ON achievements(story_id);
`

const migrationV2 = `
-- Saves table: full game state snapshots
CREATE TABLE IF NOT EXISTS saves (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL DEFAULT 'autosave',
	turn INTEGER NOT NULL,
	chapter INTEGER NOT NULL,
	location TEXT NOT NULL DEFAULT '',
	character_json TEXT NOT NULL,
	world_state_json TEXT NOT NULL,
	session_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_saves_story ON saves(story_id);
`

const migrationV3 = `
-- Add new NPC columns: appearance, notes_on_protagonist, last_seen_turn
ALTER TABLE npcs ADD COLUMN appearance TEXT NOT NULL DEFAULT '';
ALTER TABLE npcs ADD COLUMN notes_on_protagonist TEXT NOT NULL DEFAULT '[]';
ALTER TABLE npcs ADD COLUMN last_seen_turn INTEGER NOT NULL DEFAULT 0;
`

const migrationV4 = `
-- RAG chunks table: stores embedded text summaries for long-term memory retrieval
CREATE TABLE IF NOT EXISTS rag_chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    chunk_type TEXT NOT NULL DEFAULT 'summary',
    turn_start INTEGER NOT NULL DEFAULT 0,
    turn_end INTEGER NOT NULL DEFAULT 0,
    embedding BLOB,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rag_chunks_story ON rag_chunks(story_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_story_type ON rag_chunks(story_id, chunk_type);
`

const migrationV5 = `
-- Combat log table: records outcomes of combat encounters per story
CREATE TABLE IF NOT EXISTS combat_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    enemy_name TEXT NOT NULL,
    enemy_hp INTEGER NOT NULL,
    turns INTEGER NOT NULL,
    victory BOOLEAN NOT NULL,
    defeat_outcome TEXT NOT NULL DEFAULT '',
    player_hp_start INTEGER NOT NULL,
    player_hp_end INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_combat_log_story ON combat_log(story_id);
`

const migrationV6 = `
-- Add description, genre, and tone columns to stories for richer metadata display.
ALTER TABLE stories ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE stories ADD COLUMN genre TEXT NOT NULL DEFAULT '';
ALTER TABLE stories ADD COLUMN tone TEXT NOT NULL DEFAULT '';
`

const migrationV7 = `
-- Add per-story language and authoring controls for consistent prompt behavior.
ALTER TABLE stories ADD COLUMN language TEXT NOT NULL DEFAULT '';
ALTER TABLE stories ADD COLUMN writing_style TEXT NOT NULL DEFAULT '';
ALTER TABLE stories ADD COLUMN prompt_directives TEXT NOT NULL DEFAULT '';
`

const migrationV8 = `
-- Add story archive flag for management workflows.
ALTER TABLE stories ADD COLUMN is_archived INTEGER NOT NULL DEFAULT 0;
`

const migrationV9 = `
-- Expand chat message types so narrator meta turns and combat summaries persist canonically.
ALTER TABLE chat_messages RENAME TO chat_messages_old;

CREATE TABLE chat_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	turn INTEGER NOT NULL DEFAULT 0,
	role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system', 'narrator')),
	content TEXT NOT NULL,
	message_type TEXT NOT NULL DEFAULT 'narrative' CHECK(message_type IN ('narrative', 'combat', 'crafting', 'dialogue', 'narrator', 'combat_summary')),
	metadata_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO chat_messages (id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at)
SELECT id, session_id, story_id, turn, role, content, message_type, metadata_json, created_at
FROM chat_messages_old;

DROP TABLE chat_messages_old;

CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX idx_chat_messages_story ON chat_messages(story_id);
`

const migrationV10 = `
-- Composite indexes for the live-play hot path.
CREATE INDEX IF NOT EXISTS idx_chat_messages_session_turn_id
	ON chat_messages(session_id, turn, id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_story_turn_id
	ON chat_messages(story_id, turn, id);
CREATE INDEX IF NOT EXISTS idx_npcs_story_last_seen
	ON npcs(story_id, last_seen_turn DESC);
CREATE INDEX IF NOT EXISTS idx_achievements_story_name_ci
	ON achievements(story_id, name COLLATE NOCASE);
`

const migrationV11 = `
-- Richer social/world state and branch-aware save metadata.
ALTER TABLE npcs ADD COLUMN relationship_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE world_state ADD COLUMN story_hooks_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE world_state ADD COLUMN world_reactions_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE saves ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}';
`

const migrationV12 = `
-- Soft player-authored future guidance for upcoming chapter beats.
ALTER TABLE world_state ADD COLUMN player_guidance_json TEXT NOT NULL DEFAULT '[]';
`

const migrationV13 = `
-- Canonical faction fronts and regional pressure state.
ALTER TABLE world_state ADD COLUMN fronts_json TEXT NOT NULL DEFAULT '[]';
`

const migrationV14 = `
-- Canonical nemesis state for promoted rivals.
ALTER TABLE npcs ADD COLUMN nemesis_json TEXT NOT NULL DEFAULT '{}';
`

const migrationV15 = `
-- Canonical investigation board state for mysteries, clues, suspects, and theories.
ALTER TABLE world_state ADD COLUMN investigation_board_json TEXT NOT NULL DEFAULT '{}';
`

const migrationV16 = `
-- Canonical long-arc downtime project clocks.
ALTER TABLE world_state ADD COLUMN project_clocks_json TEXT NOT NULL DEFAULT '{}';
`

const migrationV17 = `
-- Canonical protagonist timeline for age, life stage, and major growth milestones.
ALTER TABLE world_state ADD COLUMN character_timeline_json TEXT NOT NULL DEFAULT '{}';
`

const migrationV18 = `
-- Browser/API idempotency cache for committed turn submissions.
CREATE TABLE IF NOT EXISTS turn_idempotency (
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	idempotency_key TEXT NOT NULL,
	events_json TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (story_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_turn_idempotency_created_at
	ON turn_idempotency(created_at);
`

const migrationV19 = `
-- Optional persisted scene progression contract for anti-loop/runtime guidance.
ALTER TABLE world_state ADD COLUMN scene_contract_json TEXT NOT NULL DEFAULT '{}';
`

const migrationV20 = `
-- Cross-process story turn lock shared by terminal and browser gateway clients.
CREATE TABLE IF NOT EXISTS story_turn_locks (
	story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,
	owner TEXT NOT NULL,
	acquired_at TEXT NOT NULL,
	locked_until TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_story_turn_locks_locked_until
	ON story_turn_locks(locked_until);
`

const migrationV21 = `
-- Durable idempotency claims so browser retries cannot double-commit after
-- process restarts or provider timeouts.
ALTER TABLE turn_idempotency ADD COLUMN status TEXT NOT NULL DEFAULT 'committed';
ALTER TABLE turn_idempotency ADD COLUMN owner TEXT NOT NULL DEFAULT '';
ALTER TABLE turn_idempotency ADD COLUMN locked_until TEXT NOT NULL DEFAULT '';
ALTER TABLE turn_idempotency ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';
ALTER TABLE turn_idempotency ADD COLUMN error TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_turn_idempotency_status_locked_until
	ON turn_idempotency(status, locked_until);
`

const migrationV22 = `
-- Monotonic branch revision for shared browser/terminal mutation safety.
ALTER TABLE stories ADD COLUMN revision INTEGER NOT NULL DEFAULT 0;

-- Bind idempotency rows to the exact request fingerprint that created them.
ALTER TABLE turn_idempotency ADD COLUMN request_hash TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_turn_idempotency_request_hash
	ON turn_idempotency(story_id, idempotency_key, request_hash);
`

const migrationV23 = `
-- Browser visual profile and async image asset registry.
-- Image generation must not block the canonical turn path: rows can be pending
-- while the browser keeps using the latest ready asset.
CREATE TABLE IF NOT EXISTS story_visual_profiles (
	story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,
	world_style_prompt TEXT NOT NULL DEFAULT '',
	character_style_prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	palette TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS visual_assets (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	kind TEXT NOT NULL,
	subject TEXT NOT NULL,
	entity_id TEXT NOT NULL DEFAULT '',
	prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'queued', 'running', 'ready', 'failed')),
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	source TEXT NOT NULL DEFAULT '',
	error TEXT NOT NULL DEFAULT '',
	turn INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id, kind, subject)
);

CREATE INDEX IF NOT EXISTS idx_visual_assets_story_kind
	ON visual_assets(story_id, kind, updated_at DESC);
`

const migrationV24 = `
-- Preserve every generated visual variant so the browser can offer image
-- history, prompt inspection, and previous/next selection without losing files.
CREATE TABLE IF NOT EXISTS visual_asset_versions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	kind TEXT NOT NULL,
	subject TEXT NOT NULL,
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	turn INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_visual_asset_versions_asset
	ON visual_asset_versions(asset_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_visual_asset_versions_story
	ON visual_asset_versions(story_id, kind, subject, created_at DESC);
`

const migrationV25 = `
-- Durable non-blocking image generation queue. Browser requests enqueue work and
-- the gateway worker claims jobs, so OpenClaw/LiteLLM image latency never blocks
-- canonical gameplay or the browser request lifecycle.
CREATE TABLE IF NOT EXISTS visual_generation_jobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
	attempts INTEGER NOT NULL DEFAULT 0,
	max_attempts INTEGER NOT NULL DEFAULT 3,
	locked_until TEXT NOT NULL DEFAULT '',
	request_payload_json TEXT NOT NULL DEFAULT '{}',
	error TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	started_at TEXT NOT NULL DEFAULT '',
	finished_at TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_visual_generation_jobs_status_lock
	ON visual_generation_jobs(status, locked_until, created_at);

CREATE INDEX IF NOT EXISTS idx_visual_generation_jobs_story
	ON visual_generation_jobs(story_id, status, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_visual_generation_jobs_active_asset
	ON visual_generation_jobs(asset_id)
	WHERE status IN ('queued', 'running');
`

const migrationV26 = `
-- Progressive NPC discovery metadata. The Go migrator applies this with
-- idempotent column creation and save-compatible backfill.
ALTER TABLE npcs ADD COLUMN discovery_json TEXT NOT NULL DEFAULT '{}';
CREATE INDEX IF NOT EXISTS idx_npcs_story_last_seen_alive
	ON npcs(story_id, is_alive, last_seen_turn DESC);
`

const migrationV27 = `
-- Immutable timeline identity around the existing transactional turn kernel.
CREATE TABLE IF NOT EXISTS story_branches (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	fork_commit_id TEXT,
	head_commit_id TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id, name)
);

CREATE TABLE IF NOT EXISTS turn_commits (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	parent_commit_id TEXT REFERENCES turn_commits(id),
	canonical_turn INTEGER NOT NULL,
	story_revision INTEGER NOT NULL,
	payload_hash TEXT NOT NULL,
	kind TEXT NOT NULL DEFAULT 'turn',
	message TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS turn_snapshots (
	commit_id TEXT PRIMARY KEY REFERENCES turn_commits(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	format_version INTEGER NOT NULL,
	payload_json TEXT NOT NULL,
	payload_hash TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS canonical_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	commit_id TEXT NOT NULL REFERENCES turn_commits(id) ON DELETE CASCADE,
	sequence INTEGER NOT NULL,
	event_type TEXT NOT NULL,
	payload_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(commit_id, sequence)
);

CREATE TABLE IF NOT EXISTS save_bookmarks (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	save_id TEXT REFERENCES saves(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(save_id)
);

CREATE TABLE IF NOT EXISTS generation_traces (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	kind TEXT NOT NULL,
	request_id TEXT NOT NULL DEFAULT '',
	payload_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audio_artifacts (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	kind TEXT NOT NULL DEFAULT 'narration',
	entity_id TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	metadata_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_story_branches_story ON story_branches(story_id, created_at);
CREATE INDEX IF NOT EXISTS idx_turn_commits_story_branch_turn ON turn_commits(story_id, branch_id, canonical_turn, created_at);
CREATE INDEX IF NOT EXISTS idx_turn_commits_parent ON turn_commits(parent_commit_id, created_at);
CREATE INDEX IF NOT EXISTS idx_turn_snapshots_story ON turn_snapshots(story_id, created_at);
CREATE INDEX IF NOT EXISTS idx_canonical_events_commit ON canonical_events(commit_id, sequence);
CREATE INDEX IF NOT EXISTS idx_save_bookmarks_story ON save_bookmarks(story_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_generation_traces_lineage ON generation_traces(story_id, branch_id, source_commit_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audio_artifacts_lineage ON audio_artifacts(story_id, branch_id, source_commit_id, created_at);
CREATE INDEX IF NOT EXISTS idx_chat_messages_branch_turn ON chat_messages(story_id, branch_id, turn, id);
CREATE INDEX IF NOT EXISTS idx_chapters_branch_number ON chapters(story_id, branch_id, chapter_number);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_branch_turn ON rag_chunks(story_id, branch_id, turn_end);
CREATE INDEX IF NOT EXISTS idx_saves_branch_created ON saves(story_id, branch_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_visual_assets_branch_kind ON visual_assets(story_id, branch_id, kind, updated_at DESC);

CREATE TRIGGER IF NOT EXISTS trg_turn_commits_immutable
BEFORE UPDATE ON turn_commits
WHEN NOT (
	OLD.payload_hash = '' AND NEW.payload_hash != ''
	AND OLD.id = NEW.id AND OLD.story_id = NEW.story_id AND OLD.branch_id = NEW.branch_id
	AND COALESCE(OLD.parent_commit_id,'') = COALESCE(NEW.parent_commit_id,'')
	AND OLD.canonical_turn = NEW.canonical_turn AND OLD.story_revision = NEW.story_revision
	AND OLD.kind = NEW.kind AND OLD.message = NEW.message AND OLD.created_at = NEW.created_at
)
BEGIN
	SELECT RAISE(ABORT, 'turn commits are immutable');
END;

CREATE TRIGGER IF NOT EXISTS trg_turn_snapshots_immutable
BEFORE UPDATE ON turn_snapshots
BEGIN
	SELECT RAISE(ABORT, 'turn snapshots are immutable');
END;

CREATE TRIGGER IF NOT EXISTS trg_canonical_events_immutable
BEFORE UPDATE ON canonical_events
BEGIN
	SELECT RAISE(ABORT, 'canonical events are immutable');
END;
`

const migrationV28 = `
-- Forward-only completion of the lineage registry for installations that
-- applied V27 before generation/audio tables were introduced.
CREATE TABLE IF NOT EXISTS generation_traces (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	kind TEXT NOT NULL,
	request_id TEXT NOT NULL DEFAULT '',
	payload_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS audio_artifacts (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	kind TEXT NOT NULL DEFAULT 'narration',
	entity_id TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	metadata_json TEXT NOT NULL DEFAULT '{}',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_generation_traces_lineage ON generation_traces(story_id, branch_id, source_commit_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audio_artifacts_lineage ON audio_artifacts(story_id, branch_id, source_commit_id, created_at);
`

const migrationV29 = `
CREATE TABLE IF NOT EXISTS canonical_entities (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_kind TEXT NOT NULL,
	canonical_name TEXT NOT NULL DEFAULT '',
	lifecycle_status TEXT NOT NULL DEFAULT 'active',
	profile_json TEXT NOT NULL DEFAULT '{}',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS entity_aliases (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	alias TEXT NOT NULL,
	alias_kind TEXT NOT NULL DEFAULT 'known',
	visibility TEXT NOT NULL DEFAULT 'private',
	valid_from_turn INTEGER NOT NULL DEFAULT 0,
	valid_to_turn INTEGER,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(entity_id,alias,alias_kind,branch_id)
);
CREATE TABLE IF NOT EXISTS identity_claims (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	subject_entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	claimed_entity_id TEXT REFERENCES canonical_entities(id),
	observer_entity_id TEXT REFERENCES canonical_entities(id),
	label TEXT NOT NULL,
	claim_kind TEXT NOT NULL,
	status TEXT NOT NULL CHECK(status IN ('rumor','observed','confirmed','contradicted','retracted','reverified')),
	confidence REAL NOT NULL CHECK(confidence>=0 AND confidence<=1),
	visibility TEXT NOT NULL DEFAULT 'private',
	evidence_json TEXT NOT NULL DEFAULT '[]',
	learned_turn INTEGER NOT NULL DEFAULT 0,
	valid_from_world_time TEXT NOT NULL DEFAULT '',
	valid_to_world_time TEXT NOT NULL DEFAULT '',
	supersedes_claim_id TEXT REFERENCES identity_claims(id),
	contradicts_claim_id TEXT REFERENCES identity_claims(id),
	retracts_claim_id TEXT REFERENCES identity_claims(id),
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS entity_forms (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	form_kind TEXT NOT NULL,
	body_entity_id TEXT REFERENCES canonical_entities(id),
	appearance_json TEXT NOT NULL DEFAULT '{}',
	valid_from_turn INTEGER NOT NULL DEFAULT 0,
	valid_to_turn INTEGER,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS entity_controller_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	form_id TEXT NOT NULL REFERENCES entity_forms(id) ON DELETE CASCADE,
	controller_entity_id TEXT NOT NULL REFERENCES canonical_entities(id),
	control_kind TEXT NOT NULL CHECK(control_kind IN ('self','possession','body_theft','puppetry','shared','unknown')),
	status TEXT NOT NULL CHECK(status IN ('started','ended','disputed')),
	turn INTEGER NOT NULL,
	world_time TEXT NOT NULL DEFAULT '',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS entity_lifecycle_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	status TEXT NOT NULL,
	turn INTEGER NOT NULL,
	world_time TEXT NOT NULL DEFAULT '',
	reason TEXT NOT NULL DEFAULT '',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS character_facts (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	subject_entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	predicate TEXT NOT NULL,
	object_json TEXT NOT NULL,
	source_entity_id TEXT REFERENCES canonical_entities(id),
	source_event_id TEXT NOT NULL DEFAULT '',
	observer_entity_id TEXT REFERENCES canonical_entities(id),
	learned_turn INTEGER NOT NULL,
	valid_from_world_time TEXT NOT NULL DEFAULT '',
	valid_to_world_time TEXT NOT NULL DEFAULT '',
	confidence REAL NOT NULL CHECK(confidence>=0 AND confidence<=1),
	visibility TEXT NOT NULL DEFAULT 'private',
	supersedes_fact_id TEXT REFERENCES character_facts(id),
	contradicts_fact_id TEXT REFERENCES character_facts(id),
	retracts_fact_id TEXT REFERENCES character_facts(id),
	evidence_json TEXT NOT NULL DEFAULT '[]',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS entity_field_locks (
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	field_path TEXT NOT NULL,
	lock_kind TEXT NOT NULL CHECK(lock_kind IN ('profile','visual')),
	locked_value_json TEXT NOT NULL,
	locked_by TEXT NOT NULL DEFAULT 'player',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(entity_id,field_path,lock_kind)
);
CREATE TABLE IF NOT EXISTS factions (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	profile_json TEXT NOT NULL DEFAULT '{}',
	visibility TEXT NOT NULL DEFAULT 'private',
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,name,branch_id)
);
CREATE TABLE IF NOT EXISTS faction_membership_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	faction_id TEXT NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id) ON DELETE CASCADE,
	status TEXT NOT NULL CHECK(status IN ('joined','left','expelled','rumored','confirmed')),
	role TEXT NOT NULL DEFAULT '',
	visibility TEXT NOT NULL DEFAULT 'private',
	turn INTEGER NOT NULL,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS faction_relationship_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	source_faction_id TEXT NOT NULL REFERENCES factions(id),
	target_faction_id TEXT NOT NULL REFERENCES factions(id),
	delta INTEGER NOT NULL CHECK(delta>=-100 AND delta<=100),
	reason TEXT NOT NULL,
	turn INTEGER NOT NULL,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS reputation_events (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	faction_id TEXT NOT NULL REFERENCES factions(id),
	entity_id TEXT NOT NULL REFERENCES canonical_entities(id),
	delta INTEGER NOT NULL CHECK(delta>=-100 AND delta<=100),
	reason TEXT NOT NULL,
	source_event_id TEXT NOT NULL DEFAULT '',
	visibility TEXT NOT NULL DEFAULT 'player',
	turn INTEGER NOT NULL,
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_identity_claims_observer ON identity_claims(story_id,subject_entity_id,observer_entity_id,status,learned_turn);
CREATE INDEX IF NOT EXISTS idx_character_facts_projection ON character_facts(story_id,subject_entity_id,observer_entity_id,visibility,learned_turn);
CREATE INDEX IF NOT EXISTS idx_entity_forms_history ON entity_forms(story_id,entity_id,valid_from_turn);
CREATE INDEX IF NOT EXISTS idx_reputation_ledger ON reputation_events(story_id,faction_id,entity_id,turn);
CREATE UNIQUE INDEX IF NOT EXISTS idx_npcs_canonical_entity ON npcs(story_id,canonical_entity_id) WHERE canonical_entity_id!='';
CREATE TRIGGER IF NOT EXISTS trg_identity_claims_immutable BEFORE UPDATE ON identity_claims BEGIN SELECT RAISE(ABORT,'identity claims are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_character_facts_immutable BEFORE UPDATE ON character_facts BEGIN SELECT RAISE(ABORT,'character facts are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_controller_events_immutable BEFORE UPDATE ON entity_controller_events BEGIN SELECT RAISE(ABORT,'controller events are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_lifecycle_events_immutable BEFORE UPDATE ON entity_lifecycle_events BEGIN SELECT RAISE(ABORT,'lifecycle events are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_reputation_events_immutable BEFORE UPDATE ON reputation_events BEGIN SELECT RAISE(ABORT,'reputation events are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_membership_events_immutable BEFORE UPDATE ON faction_membership_events BEGIN SELECT RAISE(ABORT,'membership events are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_faction_relationship_events_immutable BEFORE UPDATE ON faction_relationship_events BEGIN SELECT RAISE(ABORT,'faction relationship events are append-only'); END;
`

const migrationV30 = `
CREATE TABLE IF NOT EXISTS regions (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,name TEXT NOT NULL,parent_region_id TEXT REFERENCES regions(id),visibility TEXT NOT NULL DEFAULT 'private',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),UNIQUE(story_id,name,branch_id));
CREATE TABLE IF NOT EXISTS locations (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,canonical_name TEXT NOT NULL,region_id TEXT REFERENCES regions(id),parent_location_id TEXT REFERENCES locations(id),description TEXT NOT NULL DEFAULT '',discovery_state TEXT NOT NULL DEFAULT 'unknown',discovered_turn INTEGER NOT NULL DEFAULT 0,visibility TEXT NOT NULL DEFAULT 'player',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP,UNIQUE(story_id,canonical_name,branch_id));
CREATE TABLE IF NOT EXISTS location_aliases (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,location_id TEXT NOT NULL REFERENCES locations(id) ON DELETE CASCADE,alias TEXT NOT NULL,visibility TEXT NOT NULL DEFAULT 'player',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),UNIQUE(location_id,alias,branch_id));
CREATE TABLE IF NOT EXISTS location_edges (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,from_location_id TEXT NOT NULL REFERENCES locations(id),to_location_id TEXT NOT NULL REFERENCES locations(id),direction TEXT NOT NULL DEFAULT '',travel_minutes INTEGER NOT NULL DEFAULT 0,conditions_json TEXT NOT NULL DEFAULT '{}',valid_from_world_time TEXT NOT NULL DEFAULT '',valid_to_world_time TEXT NOT NULL DEFAULT '',visibility TEXT NOT NULL DEFAULT 'private',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id));
CREATE TABLE IF NOT EXISTS entity_position_events (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,entity_id TEXT NOT NULL,location_id TEXT NOT NULL REFERENCES locations(id),event_kind TEXT NOT NULL,turn INTEGER NOT NULL,world_time TEXT NOT NULL DEFAULT '',visibility TEXT NOT NULL DEFAULT 'player',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS world_calendars (story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,name TEXT NOT NULL,config_json TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS world_clocks (story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,calendar_story_id TEXT NOT NULL REFERENCES world_calendars(story_id),day INTEGER NOT NULL DEFAULT 0,minute_of_day INTEGER NOT NULL DEFAULT 0,display_text TEXT NOT NULL DEFAULT '',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS world_time_events (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,delta_minutes INTEGER NOT NULL,reason TEXT NOT NULL,turn INTEGER NOT NULL,from_day INTEGER NOT NULL,from_minute INTEGER NOT NULL,to_day INTEGER NOT NULL,to_minute INTEGER NOT NULL,branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS weather_states (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,region_id TEXT REFERENCES regions(id),location_id TEXT REFERENCES locations(id),weather_kind TEXT NOT NULL,intensity TEXT NOT NULL DEFAULT '',description TEXT NOT NULL DEFAULT '',valid_from_day INTEGER NOT NULL,valid_from_minute INTEGER NOT NULL,valid_to_day INTEGER,valid_to_minute INTEGER,visibility TEXT NOT NULL DEFAULT 'player',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS canonical_world_events (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,event_kind TEXT NOT NULL,title TEXT NOT NULL,details_json TEXT NOT NULL DEFAULT '{}',location_id TEXT REFERENCES locations(id),faction_id TEXT REFERENCES factions(id),entity_id TEXT REFERENCES canonical_entities(id),caused_by_event_id TEXT REFERENCES canonical_world_events(id),turn INTEGER NOT NULL,world_day INTEGER NOT NULL,world_minute INTEGER NOT NULL,visibility TEXT NOT NULL DEFAULT 'private',branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS world_thread_events (id TEXT PRIMARY KEY,story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,thread_id TEXT NOT NULL,title TEXT NOT NULL,status TEXT NOT NULL,pressure INTEGER NOT NULL DEFAULT 0,details_json TEXT NOT NULL DEFAULT '{}',visibility TEXT NOT NULL DEFAULT 'private',turn INTEGER NOT NULL,branch_id TEXT NOT NULL REFERENCES story_branches(id),source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE INDEX IF NOT EXISTS idx_locations_story_discovery ON locations(story_id,discovery_state,canonical_name);
CREATE INDEX IF NOT EXISTS idx_location_edges_from ON location_edges(story_id,from_location_id,visibility);
CREATE INDEX IF NOT EXISTS idx_world_time_events_story ON world_time_events(story_id,created_at);
CREATE INDEX IF NOT EXISTS idx_weather_current ON weather_states(story_id,location_id,valid_from_day,valid_from_minute);
CREATE TRIGGER IF NOT EXISTS trg_position_events_immutable BEFORE UPDATE ON entity_position_events BEGIN SELECT RAISE(ABORT,'position events are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_world_time_events_immutable BEFORE UPDATE ON world_time_events BEGIN SELECT RAISE(ABORT,'world time events are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_weather_states_immutable BEFORE UPDATE ON weather_states BEGIN SELECT RAISE(ABORT,'weather states are append-only'); END;
CREATE TRIGGER IF NOT EXISTS trg_canonical_world_events_immutable BEFORE UPDATE ON canonical_world_events BEGIN SELECT RAISE(ABORT,'world events are append-only'); END;
`

const migrationV31 = `
CREATE TABLE IF NOT EXISTS challenge_runs (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	session_id TEXT NOT NULL DEFAULT '',
	turn INTEGER NOT NULL,
	protocol_version INTEGER NOT NULL,
	definition_json TEXT NOT NULL CHECK(json_valid(definition_json)),
	instance_json TEXT NOT NULL CHECK(json_valid(instance_json)),
	input_json TEXT NOT NULL CHECK(json_valid(input_json)),
	resolution_json TEXT NOT NULL CHECK(json_valid(resolution_json)),
	outcome_json TEXT NOT NULL CHECK(json_valid(outcome_json)),
	degree TEXT NOT NULL CHECK(degree IN ('critical_success','full_success','success_with_cost','failure_with_progress','hard_failure','catastrophe')),
	difficulty INTEGER NOT NULL,
	seed INTEGER NOT NULL,
	roll INTEGER NOT NULL,
	total INTEGER NOT NULL,
	margin INTEGER NOT NULL,
	modifiers_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(modifiers_json)),
	timing_policy_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(timing_policy_json)),
	costs_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(costs_json)),
	state_deltas_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(state_deltas_json)),
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_challenge_runs_lineage ON challenge_runs(story_id,branch_id,source_commit_id,turn,created_at);
CREATE TRIGGER IF NOT EXISTS trg_challenge_runs_immutable
BEFORE UPDATE ON challenge_runs
WHEN NOT (OLD.source_commit_id='' AND NEW.source_commit_id!='' AND OLD.branch_id=NEW.branch_id)
BEGIN SELECT RAISE(ABORT,'challenge runs are immutable'); END;
`

const migrationV34 = `
CREATE TABLE IF NOT EXISTS minigame_instances (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	turn INTEGER NOT NULL DEFAULT 0,
	protocol_version INTEGER NOT NULL,
	kind TEXT NOT NULL,
	phase TEXT NOT NULL CHECK(phase IN ('ready','active','paused','resolved')),
	instance_json TEXT NOT NULL CHECK(json_valid(instance_json)),
	branch_id TEXT NOT NULL REFERENCES story_branches(id),
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_minigame_instances_branch_phase
	ON minigame_instances(story_id,branch_id,phase,updated_at DESC);
CREATE TRIGGER IF NOT EXISTS trg_minigame_instance_lineage_immutable
BEFORE UPDATE ON minigame_instances
WHEN OLD.story_id!=NEW.story_id OR OLD.branch_id!=NEW.branch_id OR OLD.source_commit_id!=NEW.source_commit_id OR OLD.kind!=NEW.kind OR OLD.protocol_version!=NEW.protocol_version
BEGIN SELECT RAISE(ABORT,'minigame lineage is immutable'); END;
`

const migrationV35 = `
CREATE TABLE IF NOT EXISTS story_tts_settings (
	story_id TEXT PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,
	mode TEXT NOT NULL DEFAULT 'off' CHECK(mode IN ('off','narrator','dialogue','all')),
	autoplay INTEGER NOT NULL DEFAULT 0 CHECK(autoplay IN (0,1)),
	default_language_tag TEXT NOT NULL DEFAULT '',
	provider_policy_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(provider_policy_json)),
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS voice_profiles (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	provider_voice_id TEXT NOT NULL,
	display_name TEXT NOT NULL,
	language_tags_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(language_tags_json)),
	traits_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(traits_json)),
	rights_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(rights_json)),
	version TEXT NOT NULL DEFAULT '',
	style_family TEXT NOT NULL DEFAULT 'neutral',
	enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(provider,model,provider_voice_id,version,style_family)
);

CREATE TABLE IF NOT EXISTS character_voice_assignments (
	id TEXT PRIMARY KEY,
	assignment_key TEXT NOT NULL,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	entity_id TEXT,
	identity_id TEXT,
	form_id TEXT,
	role TEXT NOT NULL CHECK(role IN ('narrator','protagonist','npc')),
	voice_profile_id TEXT NOT NULL REFERENCES voice_profiles(id),
	enabled_mode TEXT NOT NULL DEFAULT 'inherit' CHECK(enabled_mode IN ('inherit','on','off')),
	language_tag TEXT NOT NULL DEFAULT '',
	style_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(style_json)),
	locked INTEGER NOT NULL DEFAULT 0 CHECK(locked IN (0,1)),
	importance TEXT NOT NULL DEFAULT 'supporting' CHECK(importance IN ('major','supporting','minor')),
	allow_duplicate INTEGER NOT NULL DEFAULT 0 CHECK(allow_duplicate IN (0,1)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,assignment_key)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_major_voice_unique
	ON character_voice_assignments(story_id,voice_profile_id)
	WHERE importance='major' AND allow_duplicate=0 AND enabled_mode!='off';
CREATE INDEX IF NOT EXISTS idx_voice_assignments_story
	ON character_voice_assignments(story_id,role,entity_id,form_id);

CREATE TABLE IF NOT EXISTS pronunciation_lexicon (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	language_tag TEXT NOT NULL,
	source_text TEXT NOT NULL,
	pronunciation TEXT NOT NULL,
	alphabet TEXT NOT NULL DEFAULT 'ipa' CHECK(alphabet IN ('ipa','x-sampa','provider')),
	case_sensitive INTEGER NOT NULL DEFAULT 0 CHECK(case_sensitive IN (0,1)),
	revision INTEGER NOT NULL DEFAULT 1 CHECK(revision > 0),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,language_tag,source_text,case_sensitive)
);

CREATE TABLE IF NOT EXISTS tts_cache_entries (
	cache_key TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	provider_voice_id TEXT NOT NULL,
	voice_version TEXT NOT NULL DEFAULT '',
	language_tag TEXT NOT NULL,
	text_hash TEXT NOT NULL,
	style_hash TEXT NOT NULL,
	style_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(style_json)),
	speed REAL NOT NULL DEFAULT 1.0 CHECK(speed BETWEEN 0.25 AND 4.0),
	output_format TEXT NOT NULL CHECK(output_format IN ('mp3','opus','wav','aac','flac','pcm')),
	status TEXT NOT NULL CHECK(status IN ('pending','ready','failed','invalidated')),
	file_path TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0 CHECK(duration_ms >= 0),
	timings_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(timings_json)),
	error TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tts_cache_identity
	ON tts_cache_entries(provider,model,provider_voice_id,voice_version,language_tag,text_hash,style_hash,speed,output_format);

CREATE TABLE IF NOT EXISTS audio_assets (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	source_message_id INTEGER NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
	segment_index INTEGER NOT NULL CHECK(segment_index >= 0),
	segment_kind TEXT NOT NULL CHECK(segment_kind IN ('narrator','dialogue')),
	speaker_entity_id TEXT,
	identity_id TEXT,
	form_id TEXT,
	voice_profile_id TEXT NOT NULL REFERENCES voice_profiles(id),
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	provider_voice_id TEXT NOT NULL,
	voice_version TEXT NOT NULL DEFAULT '',
	language_tag TEXT NOT NULL,
	pronunciation_revision INTEGER NOT NULL DEFAULT 0 CHECK(pronunciation_revision >= 0),
	text TEXT NOT NULL,
	text_hash TEXT NOT NULL,
	cache_key TEXT NOT NULL,
	style_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(style_json)),
	speed REAL NOT NULL DEFAULT 1.0 CHECK(speed BETWEEN 0.25 AND 4.0),
	output_format TEXT NOT NULL CHECK(output_format IN ('mp3','opus','wav','aac','flac','pcm')),
	status TEXT NOT NULL CHECK(status IN ('pending','queued','running','ready','failed','cancelled','invalidated')),
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	duration_ms INTEGER NOT NULL DEFAULT 0 CHECK(duration_ms >= 0),
	timings_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(timings_json)),
	generation_run_id TEXT REFERENCES generation_runs(id),
	error TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,branch_id,source_message_id,segment_index,voice_profile_id,cache_key)
);
CREATE INDEX IF NOT EXISTS idx_audio_assets_lineage_v35
	ON audio_assets(story_id,branch_id,source_commit_id,source_message_id,segment_index);
CREATE INDEX IF NOT EXISTS idx_audio_assets_cache
	ON audio_assets(cache_key,status);

CREATE TABLE IF NOT EXISTS tts_jobs (
	id TEXT PRIMARY KEY,
	audio_asset_id TEXT NOT NULL UNIQUE REFERENCES audio_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL REFERENCES story_branches(id) ON DELETE CASCADE,
	source_commit_id TEXT NOT NULL REFERENCES turn_commits(id),
	status TEXT NOT NULL CHECK(status IN ('queued','running','succeeded','failed','cancelled')),
	provider TEXT NOT NULL,
	attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts >= 0),
	max_attempts INTEGER NOT NULL DEFAULT 3 CHECK(max_attempts BETWEEN 1 AND 10),
	next_attempt_at DATETIME,
	trace_id TEXT NOT NULL DEFAULT '',
	parent_run_id TEXT,
	generation_run_id TEXT REFERENCES generation_runs(id),
	error_class TEXT NOT NULL DEFAULT '',
	error TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_tts_jobs_queue
	ON tts_jobs(status,next_attempt_at,created_at);
CREATE INDEX IF NOT EXISTS idx_tts_jobs_lineage
	ON tts_jobs(story_id,branch_id,source_commit_id,created_at);

CREATE TRIGGER IF NOT EXISTS trg_audio_asset_lineage_immutable
BEFORE UPDATE ON audio_assets
WHEN OLD.story_id!=NEW.story_id OR OLD.branch_id!=NEW.branch_id OR OLD.source_commit_id!=NEW.source_commit_id OR OLD.source_message_id!=NEW.source_message_id OR OLD.segment_index!=NEW.segment_index
BEGIN SELECT RAISE(ABORT,'audio asset lineage is immutable'); END;
CREATE TRIGGER IF NOT EXISTS trg_tts_job_lineage_immutable
BEFORE UPDATE ON tts_jobs
WHEN OLD.story_id!=NEW.story_id OR OLD.branch_id!=NEW.branch_id OR OLD.source_commit_id!=NEW.source_commit_id OR OLD.audio_asset_id!=NEW.audio_asset_id
BEGIN SELECT RAISE(ABORT,'tts job lineage is immutable'); END;
`

const migrationV32 = `
CREATE TABLE IF NOT EXISTS prompt_profiles (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT NOT NULL DEFAULT '',
	redaction_policy TEXT NOT NULL DEFAULT 'secrets_and_reasoning',
	retention_days INTEGER NOT NULL DEFAULT 30 CHECK(retention_days BETWEEN 1 AND 3650),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prompt_profile_revisions (
	id TEXT PRIMARY KEY,
	profile_id TEXT NOT NULL REFERENCES prompt_profiles(id) ON DELETE CASCADE,
	version INTEGER NOT NULL CHECK(version > 0),
	template_version TEXT NOT NULL DEFAULT '',
	prompt_hash TEXT NOT NULL,
	response_schema_hash TEXT NOT NULL DEFAULT '',
	config_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(config_json)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(profile_id, version),
	UNIQUE(profile_id, prompt_hash, response_schema_hash)
);

CREATE TABLE IF NOT EXISTS generation_runs (
	id TEXT PRIMARY KEY,
	trace_id TEXT NOT NULL,
	parent_run_id TEXT REFERENCES generation_runs(id),
	story_id TEXT NOT NULL DEFAULT '',
	branch_id TEXT NOT NULL DEFAULT '',
	source_commit_id TEXT NOT NULL DEFAULT '',
	message_id INTEGER,
	stage TEXT NOT NULL,
	status TEXT NOT NULL CHECK(status IN ('running','succeeded','failed','cancelled')),
	prompt_revision_id TEXT REFERENCES prompt_profile_revisions(id),
	prompt_hash TEXT NOT NULL,
	request_config_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(request_config_json)),
	requested_streaming INTEGER NOT NULL DEFAULT 0 CHECK(requested_streaming IN (0,1)),
	observed_streaming INTEGER NOT NULL DEFAULT 0 CHECK(observed_streaming IN (0,1)),
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	reasoning_tokens INTEGER NOT NULL DEFAULT 0,
	cached_input_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	cost_usd REAL NOT NULL DEFAULT 0,
	ttft_ms INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	error_class TEXT NOT NULL DEFAULT '',
	metadata_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(metadata_json)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME
);

CREATE TABLE IF NOT EXISTS generation_attempts (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
	sequence INTEGER NOT NULL CHECK(sequence > 0),
	provider TEXT NOT NULL,
	requested_model TEXT NOT NULL DEFAULT '',
	resolved_model TEXT NOT NULL DEFAULT '',
	reasoning_config_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(reasoning_config_json)),
	requested_streaming INTEGER NOT NULL DEFAULT 0 CHECK(requested_streaming IN (0,1)),
	observed_streaming INTEGER NOT NULL DEFAULT 0 CHECK(observed_streaming IN (0,1)),
	status TEXT NOT NULL CHECK(status IN ('running','succeeded','failed','cancelled')),
	ttft_ms INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	input_tokens INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	reasoning_tokens INTEGER NOT NULL DEFAULT 0,
	cached_input_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	cost_usd REAL NOT NULL DEFAULT 0,
	retry_reason TEXT NOT NULL DEFAULT '',
	error_class TEXT NOT NULL DEFAULT '',
	error_summary TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME,
	UNIQUE(run_id, sequence)
);

CREATE TABLE IF NOT EXISTS generation_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_id TEXT NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
	attempt_id TEXT REFERENCES generation_attempts(id) ON DELETE CASCADE,
	event_type TEXT NOT NULL,
	payload_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(payload_json)),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_generation_runs_trace ON generation_runs(trace_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_generation_runs_story_lineage ON generation_runs(story_id, branch_id, source_commit_id, created_at);
CREATE INDEX IF NOT EXISTS idx_generation_runs_message ON generation_runs(message_id, created_at);
CREATE INDEX IF NOT EXISTS idx_generation_attempts_run ON generation_attempts(run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_generation_events_run ON generation_events(run_id, id);
CREATE INDEX IF NOT EXISTS idx_prompt_revisions_profile ON prompt_profile_revisions(profile_id, version DESC);

CREATE TRIGGER IF NOT EXISTS trg_prompt_revisions_immutable BEFORE UPDATE ON prompt_profile_revisions
BEGIN SELECT RAISE(ABORT,'prompt profile revisions are immutable'); END;
CREATE TRIGGER IF NOT EXISTS trg_generation_events_immutable BEFORE UPDATE ON generation_events
BEGIN SELECT RAISE(ABORT,'generation events are append-only'); END;
`

const migrationV33 = `-- Applied by applyMigrationV33 because visual_assets must be rebuilt safely.`

const migrationV33Prelude = `
CREATE TABLE IF NOT EXISTS visual_profile_revisions (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	revision INTEGER NOT NULL CHECK(revision > 0),
	world_style_prompt TEXT NOT NULL DEFAULT '',
	character_style_prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	palette TEXT NOT NULL DEFAULT '',
	fingerprint TEXT NOT NULL,
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id, branch_id, revision),
	UNIQUE(story_id, branch_id, fingerprint)
);

INSERT OR IGNORE INTO visual_profile_revisions
	(id,story_id,revision,world_style_prompt,character_style_prompt,negative_prompt,palette,fingerprint,branch_id,source_commit_id,created_at)
SELECT 'visual-profile-legacy-' || lower(hex(randomblob(16))), p.story_id, 1,
	p.world_style_prompt,p.character_style_prompt,p.negative_prompt,p.palette,
	'legacy:' || p.story_id,s.active_branch_id,COALESCE(b.head_commit_id,''),p.created_at
FROM story_visual_profiles p
JOIN stories s ON s.id=p.story_id
LEFT JOIN story_branches b ON b.id=s.active_branch_id;
`

const migrationV33AssetRebuild = `
CREATE TABLE visual_assets_v33 (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	kind TEXT NOT NULL,
	subject TEXT NOT NULL,
	entity_id TEXT NOT NULL DEFAULT '',
	canonical_entity_id TEXT NOT NULL DEFAULT '',
	canonical_location_id TEXT NOT NULL DEFAULT '',
	form_id TEXT NOT NULL DEFAULT '',
	lineage_key TEXT NOT NULL,
	appearance_fingerprint TEXT NOT NULL,
	profile_revision_id TEXT REFERENCES visual_profile_revisions(id),
	canon_status TEXT NOT NULL DEFAULT 'draft',
	gate_state TEXT NOT NULL DEFAULT 'eligible',
	gate_reason TEXT NOT NULL DEFAULT '',
	generation_eligible INTEGER NOT NULL DEFAULT 1 CHECK(generation_eligible IN (0,1)),
	prompt TEXT NOT NULL DEFAULT '',
	negative_prompt TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','queued','running','ready','failed')),
	url TEXT NOT NULL DEFAULT '',
	file_path TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	source TEXT NOT NULL DEFAULT '',
	error TEXT NOT NULL DEFAULT '',
	turn INTEGER NOT NULL DEFAULT 0,
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id, branch_id, lineage_key)
);

INSERT INTO visual_assets_v33
	(id,story_id,kind,subject,entity_id,canonical_entity_id,canonical_location_id,form_id,
	 lineage_key,appearance_fingerprint,profile_revision_id,canon_status,gate_state,gate_reason,
	 generation_eligible,prompt,negative_prompt,status,url,file_path,provider,source,error,turn,
	 branch_id,source_commit_id,created_at,updated_at)
SELECT a.id,a.story_id,a.kind,a.subject,a.entity_id,
	CASE WHEN a.kind='character' THEN a.entity_id ELSE '' END,
	CASE WHEN a.kind='location' THEN COALESCE((SELECT l.id FROM locations l WHERE l.story_id=a.story_id AND lower(l.canonical_name)=lower(a.subject) ORDER BY l.discovered_turn DESC LIMIT 1),'') ELSE '' END,
	CASE WHEN a.kind='character' THEN COALESCE((SELECT f.id FROM entity_forms f WHERE f.story_id=a.story_id AND f.entity_id=a.entity_id AND f.valid_from_turn<=a.turn AND (f.valid_to_turn IS NULL OR f.valid_to_turn>=a.turn) ORDER BY f.valid_from_turn DESC,f.created_at DESC LIMIT 1),'') ELSE '' END,
	'legacy:' || a.id,'legacy:' || a.id,
	(SELECT p.id FROM visual_profile_revisions p WHERE p.story_id=a.story_id AND p.branch_id=a.branch_id ORDER BY p.revision DESC LIMIT 1),
	CASE WHEN a.status='ready' THEN 'legacy' ELSE 'draft' END,'legacy','Migrated visual lineage',1,
	a.prompt,a.negative_prompt,a.status,a.url,a.file_path,a.provider,a.source,a.error,a.turn,
	a.branch_id,a.source_commit_id,a.created_at,a.updated_at
FROM visual_assets a;

DROP TABLE visual_assets;
ALTER TABLE visual_assets_v33 RENAME TO visual_assets;
CREATE INDEX idx_visual_assets_story_kind ON visual_assets(story_id,kind,updated_at DESC);
CREATE INDEX idx_visual_assets_branch_kind ON visual_assets(story_id,branch_id,kind,updated_at DESC);
CREATE INDEX idx_visual_assets_lineage ON visual_assets(story_id,lineage_key,branch_id,source_commit_id);
`

const migrationV33Backfill = `
UPDATE visual_asset_versions
SET canonical_entity_id=COALESCE((SELECT a.canonical_entity_id FROM visual_assets a WHERE a.id=visual_asset_versions.asset_id),''),
	canonical_location_id=COALESCE((SELECT a.canonical_location_id FROM visual_assets a WHERE a.id=visual_asset_versions.asset_id),''),
	form_id=COALESCE((SELECT a.form_id FROM visual_assets a WHERE a.id=visual_asset_versions.asset_id),''),
	appearance_fingerprint=COALESCE((SELECT a.appearance_fingerprint FROM visual_assets a WHERE a.id=visual_asset_versions.asset_id),'legacy:'||asset_id),
	profile_revision_id=(SELECT a.profile_revision_id FROM visual_assets a WHERE a.id=visual_asset_versions.asset_id),
	canon_status=COALESCE((SELECT a.canon_status FROM visual_assets a WHERE a.id=visual_asset_versions.asset_id),'legacy')
WHERE appearance_fingerprint='';

UPDATE visual_generation_jobs
SET canonical_entity_id=COALESCE((SELECT a.canonical_entity_id FROM visual_assets a WHERE a.id=visual_generation_jobs.asset_id),''),
	canonical_location_id=COALESCE((SELECT a.canonical_location_id FROM visual_assets a WHERE a.id=visual_generation_jobs.asset_id),''),
	form_id=COALESCE((SELECT a.form_id FROM visual_assets a WHERE a.id=visual_generation_jobs.asset_id),''),
	appearance_fingerprint=COALESCE((SELECT a.appearance_fingerprint FROM visual_assets a WHERE a.id=visual_generation_jobs.asset_id),'legacy:'||asset_id),
	profile_revision_id=(SELECT a.profile_revision_id FROM visual_assets a WHERE a.id=visual_generation_jobs.asset_id)
WHERE appearance_fingerprint='';

CREATE TABLE IF NOT EXISTS visual_asset_branch_overrides (
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	prompt_override TEXT NOT NULL DEFAULT '',
	negative_prompt_override TEXT NOT NULL DEFAULT '',
	gate_state TEXT NOT NULL DEFAULT '',
	gate_reason TEXT NOT NULL DEFAULT '',
	generation_eligible INTEGER CHECK(generation_eligible IN (0,1)),
	status_override TEXT NOT NULL DEFAULT '',
	error_override TEXT NOT NULL DEFAULT '',
	provider_override TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(asset_id,branch_id)
);

CREATE TABLE IF NOT EXISTS visual_asset_selection_states (
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	selected_version_id INTEGER REFERENCES visual_asset_versions(id),
	history_json TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(history_json)),
	cursor INTEGER NOT NULL DEFAULT -1,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(asset_id,branch_id)
);

CREATE INDEX IF NOT EXISTS idx_visual_profile_revisions_reachable ON visual_profile_revisions(story_id,source_commit_id,revision DESC);
CREATE INDEX IF NOT EXISTS idx_visual_asset_versions_lineage ON visual_asset_versions(story_id,branch_id,source_commit_id,appearance_fingerprint,id DESC);
CREATE INDEX IF NOT EXISTS idx_visual_jobs_lineage ON visual_generation_jobs(story_id,branch_id,source_commit_id,status,created_at);
CREATE INDEX IF NOT EXISTS idx_visual_overrides_lineage ON visual_asset_branch_overrides(story_id,asset_id,source_commit_id);
DROP INDEX IF EXISTS idx_visual_generation_jobs_active_asset;
CREATE UNIQUE INDEX idx_visual_generation_jobs_active_asset ON visual_generation_jobs(asset_id,branch_id) WHERE status IN ('queued','running');
`

const migrationV36 = `
CREATE INDEX IF NOT EXISTS idx_regions_story_parent
	ON regions(story_id,parent_region_id,visibility,name);
CREATE INDEX IF NOT EXISTS idx_locations_story_scope
	ON locations(story_id,region_id,parent_location_id,discovery_state,canonical_name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_location_edges_canonical_route
	ON location_edges(story_id,branch_id,from_location_id,to_location_id,direction,travel_mode);
CREATE INDEX IF NOT EXISTS idx_visual_assets_map_scope
	ON visual_assets(story_id,branch_id,kind,map_scope_kind,map_scope_id,updated_at DESC);
`

const migrationV37 = `
CREATE INDEX IF NOT EXISTS idx_character_facts_visible
	ON character_facts(story_id,branch_id,subject_entity_id,visibility,learned_turn)
	WHERE retracts_fact_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_character_facts_retracts
	ON character_facts(story_id,branch_id,retracts_fact_id)
	WHERE retracts_fact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_character_facts_supersedes
	ON character_facts(story_id,branch_id,supersedes_fact_id)
	WHERE supersedes_fact_id IS NOT NULL;
`

const migrationV38 = `
CREATE INDEX IF NOT EXISTS idx_turn_idempotency_retention
	ON turn_idempotency(story_id,status,updated_at DESC);
`

const migrationV39 = `
ALTER TABLE rag_chunks ADD COLUMN embedding_norm REAL NOT NULL DEFAULT 0;
`

const migrationV40 = `
CREATE VIRTUAL TABLE IF NOT EXISTS chat_messages_fts USING fts5(
	content,
	content='chat_messages',
	content_rowid='id',
	tokenize='trigram'
);
CREATE VIRTUAL TABLE IF NOT EXISTS chapters_fts USING fts5(
	title,
	summary,
	content='chapters',
	content_rowid='id',
	tokenize='trigram'
);

CREATE TRIGGER IF NOT EXISTS trg_chat_messages_fts_insert AFTER INSERT ON chat_messages BEGIN
	INSERT INTO chat_messages_fts(rowid,content) VALUES (new.id,new.content);
END;
CREATE TRIGGER IF NOT EXISTS trg_chat_messages_fts_delete AFTER DELETE ON chat_messages BEGIN
	INSERT INTO chat_messages_fts(chat_messages_fts,rowid,content) VALUES ('delete',old.id,old.content);
END;
CREATE TRIGGER IF NOT EXISTS trg_chat_messages_fts_update AFTER UPDATE OF content ON chat_messages BEGIN
	INSERT INTO chat_messages_fts(chat_messages_fts,rowid,content) VALUES ('delete',old.id,old.content);
	INSERT INTO chat_messages_fts(rowid,content) VALUES (new.id,new.content);
END;

CREATE TRIGGER IF NOT EXISTS trg_chapters_fts_insert AFTER INSERT ON chapters BEGIN
	INSERT INTO chapters_fts(rowid,title,summary) VALUES (new.id,new.title,new.summary);
END;
CREATE TRIGGER IF NOT EXISTS trg_chapters_fts_delete AFTER DELETE ON chapters BEGIN
	INSERT INTO chapters_fts(chapters_fts,rowid,title,summary) VALUES ('delete',old.id,old.title,old.summary);
END;
CREATE TRIGGER IF NOT EXISTS trg_chapters_fts_update AFTER UPDATE OF title,summary ON chapters BEGIN
	INSERT INTO chapters_fts(chapters_fts,rowid,title,summary) VALUES ('delete',old.id,old.title,old.summary);
	INSERT INTO chapters_fts(rowid,title,summary) VALUES (new.id,new.title,new.summary);
END;

INSERT INTO chat_messages_fts(chat_messages_fts) VALUES ('rebuild');
INSERT INTO chapters_fts(chapters_fts) VALUES ('rebuild');
`

const migrationV41 = `
ALTER TABLE visual_asset_versions ADD COLUMN parent_version_id INTEGER REFERENCES visual_asset_versions(id);
ALTER TABLE visual_asset_versions ADD COLUMN operation_id TEXT;
ALTER TABLE visual_asset_versions ADD COLUMN mask_id TEXT;

CREATE TABLE image_masks (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	source_version_id INTEGER NOT NULL REFERENCES visual_asset_versions(id),
	semantics TEXT NOT NULL CHECK(semantics='edit_coverage'),
	pixel_format TEXT NOT NULL CHECK(pixel_format='L8'),
	width INTEGER NOT NULL CHECK(width>0),
	height INTEGER NOT NULL CHECK(height>0),
	orientation INTEGER NOT NULL DEFAULT 1 CHECK(orientation=1),
	preserve_value INTEGER NOT NULL DEFAULT 0 CHECK(preserve_value=0),
	editable_value INTEGER NOT NULL DEFAULT 255 CHECK(editable_value=255),
	soft_edges INTEGER NOT NULL DEFAULT 0 CHECK(soft_edges IN (0,1)),
	mime_type TEXT NOT NULL CHECK(mime_type='image/png'),
	sha256 TEXT NOT NULL,
	file_path TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(story_id,source_version_id,sha256)
);

CREATE TABLE image_operations (
	id TEXT PRIMARY KEY,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	operation TEXT NOT NULL CHECK(operation IN ('generate','edit','inpaint','image_transform','variation','reference_generate','outpaint')),
	status TEXT NOT NULL CHECK(status IN ('queued','running','succeeded','failed','cancelled')),
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	endpoint_id TEXT NOT NULL DEFAULT '',
	model_version TEXT NOT NULL DEFAULT '',
	deployment TEXT NOT NULL DEFAULT '',
	source_version_id INTEGER REFERENCES visual_asset_versions(id),
	parent_version_id INTEGER REFERENCES visual_asset_versions(id),
	mask_id TEXT REFERENCES image_masks(id),
	branch_id TEXT NOT NULL,
	source_commit_id TEXT NOT NULL,
	prompt TEXT NOT NULL,
	negative_prompt TEXT NOT NULL DEFAULT '',
	requested_parameters_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(requested_parameters_json)),
	effective_parameters_json TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(effective_parameters_json)),
	idempotency_key TEXT NOT NULL,
	provider_request_id TEXT NOT NULL DEFAULT '',
	result_version_id INTEGER REFERENCES visual_asset_versions(id),
	error_code TEXT NOT NULL DEFAULT '',
	error_summary TEXT NOT NULL DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	finished_at DATETIME,
	UNIQUE(story_id,idempotency_key)
);

CREATE TABLE provider_capability_snapshots (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	endpoint_id TEXT NOT NULL,
	model TEXT NOT NULL,
	model_version TEXT NOT NULL DEFAULT '',
	credential_mode TEXT NOT NULL,
	api_version TEXT NOT NULL DEFAULT '',
	schema_revision TEXT NOT NULL,
	capabilities_json TEXT NOT NULL CHECK(json_valid(capabilities_json)),
	provenance TEXT NOT NULL CHECK(provenance IN ('static_verified','provider_schema','runtime_probe')),
	schema_hash TEXT NOT NULL DEFAULT '',
	verified_at DATETIME NOT NULL,
	UNIQUE(provider,endpoint_id,model,model_version,api_version,schema_revision)
);

CREATE INDEX idx_image_operations_queue ON image_operations(status,created_at);
CREATE INDEX idx_image_operations_asset ON image_operations(story_id,asset_id,created_at DESC);
CREATE INDEX idx_image_masks_source ON image_masks(story_id,source_version_id);
CREATE INDEX idx_visual_versions_parent ON visual_asset_versions(asset_id,parent_version_id,id DESC);
`

const migrationV42 = `
ALTER TABLE visual_asset_versions ADD COLUMN source_kind TEXT NOT NULL DEFAULT 'generated'
	CHECK(source_kind IN ('generated','upload','imported'));

CREATE TABLE visual_asset_uploads (
	version_id INTEGER PRIMARY KEY REFERENCES visual_asset_versions(id) ON DELETE CASCADE,
	story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
	asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
	branch_id TEXT NOT NULL,
	original_filename_display TEXT NOT NULL DEFAULT '',
	declared_mime TEXT NOT NULL DEFAULT '',
	detected_mime TEXT NOT NULL CHECK(detected_mime IN ('image/png','image/jpeg','image/webp')),
	byte_size INTEGER NOT NULL CHECK(byte_size > 0),
	width INTEGER NOT NULL CHECK(width > 0),
	height INTEGER NOT NULL CHECK(height > 0),
	sha256 TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_visual_asset_uploads_asset
	ON visual_asset_uploads(story_id,asset_id,branch_id,created_at DESC);
`
