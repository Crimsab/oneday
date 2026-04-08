package storage

import "fmt"

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

		if _, err := db.conn.Exec(m.sql); err != nil {
			return fmt.Errorf("applying migration %d: %w", m.version, err)
		}
		if _, err := db.conn.Exec("INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
			return fmt.Errorf("recording migration %d: %w", m.version, err)
		}
	}
	return nil
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
