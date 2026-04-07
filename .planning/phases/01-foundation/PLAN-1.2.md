---
phase: 1
plan: 1.2
title: "SQLite storage layer with migrations and core models"
wave: 1
depends_on: []
files_modified:
  - internal/storage/db.go
  - internal/storage/migrations.go
  - internal/storage/models.go
  - internal/storage/db_test.go
  - go.mod
  - go.sum
requirements_addressed: [STOR-01]
autonomous: true
---

# Plan 1.2: SQLite Storage Layer

## Objective

Implement the SQLite storage layer using `modernc.org/sqlite` (pure Go, no CGO). Create the database connection manager, schema migrations for core game data tables (stories, characters, NPCs, world state, sessions, chat messages), and Go model structs. This layer is the persistence foundation for all game data.

## must_haves

- SQLite opens via modernc.org/sqlite driver (no CGO_ENABLED needed)
- Schema creates tables for: stories, characters, npcs, world_state, sessions, chat_messages
- Migrations run automatically on first open
- WAL mode and foreign keys enabled via PRAGMA
- Models match the design doc JSON structures
- Database closes cleanly

## Tasks

### Task 1: Add modernc.org/sqlite dependency

<read_first>
- go.mod
</read_first>

<action>
Run `go get modernc.org/sqlite` to add the pure-Go SQLite driver. This registers as database/sql driver "sqlite".
</action>

<acceptance_criteria>
- `grep "modernc.org/sqlite" go.mod` returns a match
</acceptance_criteria>

### Task 2: Create database connection manager

<read_first>
- docs/design.md (Section 8: Session and Chat Storage, and data directory structure lines 90-104)
- config.example.yaml (data_dir field)
</read_first>

<action>
Create `internal/storage/db.go`:

```go
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
	path string
}

// Open creates or opens a SQLite database at the given path.
// It creates parent directories, enables WAL mode, foreign keys,
// and runs migrations.
func Open(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory %s: %w", dir, err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database %s: %w", dbPath, err)
	}

	// Enable WAL mode for concurrent reads
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	db := &DB{conn: conn, path: dbPath}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying *sql.DB for direct queries.
func (db *DB) Conn() *sql.DB {
	return db.conn
}
```
</action>

<acceptance_criteria>
- `grep "func Open" internal/storage/db.go` matches
- `grep "func.*Close" internal/storage/db.go` matches
- `grep 'PRAGMA journal_mode=WAL' internal/storage/db.go` matches
- `grep 'PRAGMA foreign_keys=ON' internal/storage/db.go` matches
- `grep 'modernc.org/sqlite' internal/storage/db.go` matches
</acceptance_criteria>

### Task 3: Create schema migrations

<read_first>
- docs/design.md (story.json structure lines 10-90, character structure lines 100-130, NPC structure lines 130-190, session/chat lines 350-400)
</read_first>

<action>
Create `internal/storage/migrations.go`:

```go
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
```

Note: JSON fields (e.g., `stats_json`, `traits_json`, `personality_json`) store marshaled JSON. This keeps the schema flexible for AI-generated dynamic content while still having typed Go structs for access.
</action>

<acceptance_criteria>
- `grep "CREATE TABLE stories" internal/storage/migrations.go` matches
- `grep "CREATE TABLE characters" internal/storage/migrations.go` matches
- `grep "CREATE TABLE npcs" internal/storage/migrations.go` matches
- `grep "CREATE TABLE world_state" internal/storage/migrations.go` matches
- `grep "CREATE TABLE sessions" internal/storage/migrations.go` matches
- `grep "CREATE TABLE chat_messages" internal/storage/migrations.go` matches
- `grep "CREATE TABLE chapters" internal/storage/migrations.go` matches
- `grep "CREATE TABLE achievements" internal/storage/migrations.go` matches
- `grep "schema_version" internal/storage/migrations.go` matches
</acceptance_criteria>

### Task 4: Create Go model structs

<read_first>
- internal/storage/migrations.go (just created)
- docs/design.md (story.json, character, NPC structures)
</read_first>

<action>
Create `internal/storage/models.go`:

```go
package storage

import "time"

// Story represents a game story with its setting and rules.
type Story struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	SettingJSON     string    `json:"setting_json"`
	StatsSchemaJSON string    `json:"stats_schema_json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Character represents the player's protagonist.
type Character struct {
	ID               string    `json:"id"`
	StoryID          string    `json:"story_id"`
	Name             string    `json:"name"`
	Background       string    `json:"background"`
	StatsJSON        string    `json:"stats_json"`
	TraitsJSON       string    `json:"traits_json"`
	SkillsJSON       string    `json:"skills_json"`
	InventoryJSON    string    `json:"inventory_json"`
	KnownRecipesJSON string   `json:"known_recipes_json"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NPC represents an AI-generated non-player character.
type NPC struct {
	ID                string    `json:"id"`
	StoryID           string    `json:"story_id"`
	Name              string    `json:"name"`
	Role              string    `json:"role"`
	PersonalityJSON   string    `json:"personality_json"`
	PrivateThoughts   string    `json:"private_thoughts"`
	Desires           string    `json:"desires"`
	Disposition       int       `json:"disposition"`
	IsAlive           bool      `json:"is_alive"`
	FirstAppearedTurn int       `json:"first_appeared_turn"`
	CanHelp           bool      `json:"can_help"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// WorldState tracks the global state of a story.
type WorldState struct {
	ID                  string    `json:"id"`
	StoryID             string    `json:"story_id"`
	CurrentLocation     string    `json:"current_location"`
	KnownLocationsJSON  string    `json:"known_locations_json"`
	GlobalEventsJSON    string    `json:"global_events_json"`
	FactionStandingsJSON string   `json:"faction_standings_json"`
	CurrentChapter      int       `json:"current_chapter"`
	CurrentTurn         int       `json:"current_turn"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Session represents a play session within a story.
type Session struct {
	ID        string     `json:"id"`
	StoryID   string     `json:"story_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Summary   string     `json:"summary"`
}

// ChatMessage represents a single message in a session.
type ChatMessage struct {
	ID           int64     `json:"id"`
	SessionID    string    `json:"session_id"`
	StoryID      string    `json:"story_id"`
	Turn         int       `json:"turn"`
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	MessageType  string    `json:"message_type"`
	MetadataJSON string    `json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
}

// Chapter represents an AI-generated chapter summary.
type Chapter struct {
	ID            int64     `json:"id"`
	StoryID       string    `json:"story_id"`
	ChapterNumber int       `json:"chapter_number"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	StartTurn     int       `json:"start_turn"`
	EndTurn       *int      `json:"end_turn,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Achievement represents an earned achievement.
type Achievement struct {
	ID          int64     `json:"id"`
	StoryID     string    `json:"story_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Rarity      string    `json:"rarity"`
	Context     string    `json:"context"`
	EarnedAt    time.Time `json:"earned_at"`
}
```
</action>

<acceptance_criteria>
- `grep "type Story struct" internal/storage/models.go` matches
- `grep "type Character struct" internal/storage/models.go` matches
- `grep "type NPC struct" internal/storage/models.go` matches
- `grep "type WorldState struct" internal/storage/models.go` matches
- `grep "type Session struct" internal/storage/models.go` matches
- `grep "type ChatMessage struct" internal/storage/models.go` matches
- `grep "type Chapter struct" internal/storage/models.go` matches
- `grep "type Achievement struct" internal/storage/models.go` matches
</acceptance_criteria>

### Task 5: Create storage tests

<read_first>
- internal/storage/db.go
- internal/storage/migrations.go
- internal/storage/models.go
</read_first>

<action>
Create `internal/storage/db_test.go`:

```go
package storage

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesTables(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Verify all expected tables exist
	tables := []string{
		"stories", "characters", "npcs", "world_state",
		"sessions", "chat_messages", "chapters", "achievements",
		"schema_version",
	}
	for _, table := range tables {
		var name string
		err := db.Conn().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestOpenIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Open twice — migrations should be idempotent
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	db1.Close()

	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer db2.Close()

	// Verify schema_version has exactly 1 row
	var count int
	if err := db2.Conn().QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count); err != nil {
		t.Fatalf("counting schema_version: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_version rows = %d, want 1", count)
	}
}

func TestForeignKeysEnabled(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var fk int
	if err := db.Conn().QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestWALMode(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var mode string
	if err := db.Conn().QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

func TestInsertAndQueryStory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Insert a story
	_, err = db.Conn().Exec(
		`INSERT INTO stories (id, name, setting_json) VALUES (?, ?, ?)`,
		"story-1", "Test Story", `{"genre": "fantasy"}`,
	)
	if err != nil {
		t.Fatalf("INSERT story: %v", err)
	}

	// Query it back
	var name string
	err = db.Conn().QueryRow("SELECT name FROM stories WHERE id = ?", "story-1").Scan(&name)
	if err != nil {
		t.Fatalf("SELECT story: %v", err)
	}
	if name != "Test Story" {
		t.Errorf("name = %q, want %q", name, "Test Story")
	}
}

func TestForeignKeyConstraint(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Try to insert a character with a non-existent story_id
	_, err = db.Conn().Exec(
		`INSERT INTO characters (id, story_id, name) VALUES (?, ?, ?)`,
		"char-1", "nonexistent-story", "Test Character",
	)
	if err == nil {
		t.Error("expected foreign key constraint error, got nil")
	}
}

func TestChatMessageRoleConstraint(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Create prerequisite story and session
	db.Conn().Exec(`INSERT INTO stories (id, name) VALUES ('s1', 'Test')`)
	db.Conn().Exec(`INSERT INTO sessions (id, story_id) VALUES ('sess1', 's1')`)

	// Valid role
	_, err = db.Conn().Exec(
		`INSERT INTO chat_messages (session_id, story_id, role, content) VALUES (?, ?, ?, ?)`,
		"sess1", "s1", "user", "Hello",
	)
	if err != nil {
		t.Fatalf("valid role insert: %v", err)
	}

	// Invalid role
	_, err = db.Conn().Exec(
		`INSERT INTO chat_messages (session_id, story_id, role, content) VALUES (?, ?, ?, ?)`,
		"sess1", "s1", "invalid_role", "Hello",
	)
	if err == nil {
		t.Error("expected CHECK constraint error for invalid role, got nil")
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "dir", "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open with nested dirs: %v", err)
	}
	defer db.Close()
}
```
</action>

<acceptance_criteria>
- `grep "func TestOpenCreatesTables" internal/storage/db_test.go` matches
- `grep "func TestForeignKeysEnabled" internal/storage/db_test.go` matches
- `grep "func TestWALMode" internal/storage/db_test.go` matches
- `grep "func TestInsertAndQueryStory" internal/storage/db_test.go` matches
- `grep "func TestForeignKeyConstraint" internal/storage/db_test.go` matches
- `go test ./internal/storage/ -v` passes all tests
</acceptance_criteria>
