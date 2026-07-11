package storage

import (
	"path/filepath"
	"testing"
)

func TestMigrationV27CreatesTimelineSchema(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "timeline.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"story_branches", "turn_commits", "turn_snapshots", "canonical_events", "save_bookmarks", "generation_traces", "audio_artifacts"} {
		var got string
		if err := db.Conn().QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&got); err != nil {
			t.Fatalf("timeline table %s: %v", table, err)
		}
	}
	for _, item := range []struct{ table, column string }{
		{"stories", "active_branch_id"},
		{"chat_messages", "branch_id"}, {"chat_messages", "source_commit_id"},
		{"chapters", "branch_id"}, {"rag_chunks", "source_commit_id"},
		{"saves", "branch_id"}, {"visual_assets", "source_commit_id"},
	} {
		ok, err := db.columnExists(item.table, item.column)
		if err != nil || !ok {
			t.Fatalf("missing %s.%s (exists=%v err=%v)", item.table, item.column, ok, err)
		}
	}
}

func TestMigrationV27BackfillsExistingStoryAndLineage(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "upgrade.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if _, err := db.Conn().Exec(`INSERT INTO stories (id, name) VALUES ('legacy-story', 'Legacy')`); err != nil {
		t.Fatalf("insert story: %v", err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO sessions (id, story_id) VALUES ('legacy-session', 'legacy-story')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO chat_messages (session_id, story_id, role, content) VALUES ('legacy-session', 'legacy-story', 'assistant', 'before branches')`); err != nil {
		t.Fatalf("insert message: %v", err)
	}

	if _, err := db.Conn().Exec(`DELETE FROM schema_version WHERE version = 27; DELETE FROM story_branches;`); err != nil {
		t.Fatalf("simulate pre-v27 database: %v", err)
	}
	if err := db.migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var branchID, activeBranchID, headCommitID, messageBranchID, messageCommitID string
	if err := db.Conn().QueryRow(`SELECT id, head_commit_id FROM story_branches WHERE story_id='legacy-story'`).Scan(&branchID, &headCommitID); err != nil {
		t.Fatalf("branch backfill: %v", err)
	}
	if err := db.Conn().QueryRow(`SELECT active_branch_id FROM stories WHERE id='legacy-story'`).Scan(&activeBranchID); err != nil {
		t.Fatalf("active branch: %v", err)
	}
	if err := db.Conn().QueryRow(`SELECT branch_id, source_commit_id FROM chat_messages WHERE story_id='legacy-story'`).Scan(&messageBranchID, &messageCommitID); err != nil {
		t.Fatalf("message lineage: %v", err)
	}
	if branchID == "" || headCommitID == "" || activeBranchID != branchID || messageBranchID != branchID || messageCommitID != headCommitID {
		t.Fatalf("invalid backfill branch=%q active=%q head=%q message=(%q,%q)", branchID, activeBranchID, headCommitID, messageBranchID, messageCommitID)
	}
}

func TestMigrationV29BackfillsNPCAndProtagonistCanon(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "canon-upgrade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Conn().Exec(`INSERT INTO stories (id,name) VALUES ('legacy-canon','Legacy Canon')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureStoryTimeline("legacy-canon"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO characters (id,story_id,name) VALUES ('legacy-hero','legacy-canon','Hero')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO npcs (id,story_id,name) VALUES ('legacy-npc','legacy-canon','Mara')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`DELETE FROM schema_version WHERE version=29; DELETE FROM entity_forms; DELETE FROM identity_claims; DELETE FROM entity_aliases; DELETE FROM canonical_entities; UPDATE npcs SET canonical_entity_id=''`); err != nil {
		t.Fatal(err)
	}
	if err := db.migrate(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM canonical_entities WHERE story_id='legacy-canon'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("entities=%d want 2", count)
	}
	var canonicalID string
	if err := db.Conn().QueryRow(`SELECT canonical_entity_id FROM npcs WHERE id='legacy-npc'`).Scan(&canonicalID); err != nil {
		t.Fatal(err)
	}
	if canonicalID != "legacy-npc" {
		t.Fatalf("canonical id=%q", canonicalID)
	}
}
