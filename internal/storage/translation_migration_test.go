package storage

import (
	"path/filepath"
	"testing"
)

func TestMigrationV43CreatesPersistentTranslationSchema(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "translation.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, table := range []string{"translation_glossary_entries", "content_translations", "translation_jobs", "translation_job_items"} {
		var got string
		if err := db.Conn().QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&got); err != nil {
			t.Fatalf("missing %s: %v", table, err)
		}
	}
	if _, err := db.Conn().Exec(`INSERT INTO stories(id,name,language) VALUES('story-translation','Translation','it')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureStoryTimeline("story-translation"); err != nil {
		t.Fatal(err)
	}
	var branchID string
	if err := db.Conn().QueryRow(`SELECT active_branch_id FROM stories WHERE id='story-translation'`).Scan(&branchID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO translation_jobs(id,story_id,branch_id,scope_kind,source_language,target_language,engine,status) VALUES('job','story-translation',?,'story','it','en','browser','queued')`, branchID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO translation_job_items(id,job_id,ordinal,content_kind,content_id,content_hash,source_text,status) VALUES('item','job',0,'message','1','hash','Ciao','pending')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`DELETE FROM stories WHERE id='story-translation'`); err != nil {
		t.Fatal(err)
	}
	var jobs int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM translation_jobs WHERE id='job'`).Scan(&jobs); err != nil || jobs != 0 {
		t.Fatalf("jobs=%d err=%v", jobs, err)
	}
}
