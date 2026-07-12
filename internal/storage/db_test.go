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

	// Verify schema_version has the expected number of migrations
	var count int
	if err := db2.Conn().QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count); err != nil {
		t.Fatalf("counting schema_version: %v", err)
	}
	if count < 1 {
		t.Errorf("schema_version rows = %d, want >= 1", count)
	}
}

func TestCharacterFactResolutionIndexes(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "facts.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	for _, index := range []string{
		"idx_character_facts_visible",
		"idx_character_facts_retracts",
		"idx_character_facts_supersedes",
	} {
		var name string
		if err := db.Conn().QueryRow(
			`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, index,
		).Scan(&name); err != nil {
			t.Fatalf("index %q not created: %v", index, err)
		}
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

func TestChatMessageMessageTypeConstraint(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	db.Conn().Exec(`INSERT INTO stories (id, name) VALUES ('s1', 'Test')`)
	db.Conn().Exec(`INSERT INTO sessions (id, story_id) VALUES ('sess1', 's1')`)

	validTypes := []string{"narrative", "combat", "crafting", "dialogue", "narrator", "combat_summary"}
	for _, messageType := range validTypes {
		if _, err := db.Conn().Exec(
			`INSERT INTO chat_messages (session_id, story_id, role, content, message_type) VALUES (?, ?, ?, ?, ?)`,
			"sess1", "s1", "assistant", "Hello", messageType,
		); err != nil {
			t.Fatalf("valid message_type %q insert: %v", messageType, err)
		}
	}

	if _, err := db.Conn().Exec(
		`INSERT INTO chat_messages (session_id, story_id, role, content, message_type) VALUES (?, ?, ?, ?, ?)`,
		"sess1", "s1", "assistant", "Hello", "unsupported_type",
	); err == nil {
		t.Error("expected CHECK constraint error for invalid message_type, got nil")
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
