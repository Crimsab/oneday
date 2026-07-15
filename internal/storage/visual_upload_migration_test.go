package storage

import (
	"path/filepath"
	"testing"
)

func TestMigrationV42CreatesVisualUploadSchema(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "visual-upload.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exists, err := db.columnExists("visual_asset_versions", "source_kind")
	if err != nil || !exists {
		t.Fatalf("missing visual_asset_versions.source_kind (exists=%v err=%v)", exists, err)
	}

	var table string
	if err := db.Conn().QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='visual_asset_uploads'`,
	).Scan(&table); err != nil {
		t.Fatalf("missing visual_asset_uploads: %v", err)
	}

	if _, err := db.Conn().Exec(`INSERT INTO stories (id,name) VALUES ('story-upload','Upload')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureStoryTimeline("story-upload"); err != nil {
		t.Fatal(err)
	}
	var branchID, commitID string
	if err := db.Conn().QueryRow(`SELECT active_branch_id,(SELECT head_commit_id FROM story_branches WHERE id=stories.active_branch_id) FROM stories WHERE id='story-upload'`).Scan(&branchID, &commitID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO visual_assets
		(id,story_id,kind,subject,lineage_key,appearance_fingerprint,branch_id,source_commit_id)
		VALUES ('asset-upload','story-upload','location','Dock','location:dock','dock',?,?)`, branchID, commitID); err != nil {
		t.Fatal(err)
	}
	result, err := db.Conn().Exec(`INSERT INTO visual_asset_versions
		(asset_id,story_id,kind,subject,url,file_path,branch_id,source_commit_id)
		VALUES ('asset-upload','story-upload','location','Dock','/generated.png','/tmp/generated.png',?,?)`, branchID, commitID)
	if err != nil {
		t.Fatal(err)
	}
	versionID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	var sourceKind string
	if err := db.Conn().QueryRow(`SELECT source_kind FROM visual_asset_versions WHERE id=?`, versionID).Scan(&sourceKind); err != nil {
		t.Fatal(err)
	}
	if sourceKind != "generated" {
		t.Fatalf("source_kind=%q want generated", sourceKind)
	}

	if _, err := db.Conn().Exec(`INSERT INTO visual_asset_uploads
		(version_id,story_id,asset_id,branch_id,detected_mime,byte_size,width,height,sha256)
		VALUES (?,'story-upload','asset-upload',?,'image/png',123,4,5,'sha256:test')`, versionID, branchID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`DELETE FROM visual_asset_versions WHERE id=?`, versionID); err != nil {
		t.Fatal(err)
	}
	var uploadCount int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM visual_asset_uploads WHERE version_id=?`, versionID).Scan(&uploadCount); err != nil {
		t.Fatal(err)
	}
	if uploadCount != 0 {
		t.Fatalf("visual_asset_uploads rows=%d want 0 after version delete", uploadCount)
	}
}
