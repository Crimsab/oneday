package storage

import (
	"path/filepath"
	"testing"
)

func TestMigrationV33CreatesBranchAwareVisualCanonSchema(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "visual-canon.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, table := range []string{
		"visual_profile_revisions",
		"visual_asset_branch_overrides",
		"visual_asset_selection_states",
	} {
		var name string
		if err := db.Conn().QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
			t.Fatalf("missing table %s: %v", table, err)
		}
	}

	for _, item := range []struct{ table, column string }{
		{"visual_assets", "lineage_key"},
		{"visual_assets", "canonical_entity_id"},
		{"visual_assets", "canonical_location_id"},
		{"visual_assets", "form_id"},
		{"visual_assets", "appearance_fingerprint"},
		{"visual_assets", "profile_revision_id"},
		{"visual_assets", "gate_state"},
		{"visual_asset_versions", "appearance_fingerprint"},
		{"visual_generation_jobs", "profile_revision_id"},
	} {
		exists, err := db.columnExists(item.table, item.column)
		if err != nil || !exists {
			t.Fatalf("missing %s.%s (exists=%v err=%v)", item.table, item.column, exists, err)
		}
	}
}

func TestMigrationV33AllowsSameSubjectOnDistinctVisualLineages(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "visual-lineages.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Conn().Exec(`INSERT INTO stories (id,name) VALUES ('story-visual','Visual')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureStoryTimeline("story-visual"); err != nil {
		t.Fatal(err)
	}
	var branchID, commitID string
	if err := db.Conn().QueryRow(`SELECT active_branch_id,(SELECT head_commit_id FROM story_branches WHERE id=stories.active_branch_id) FROM stories WHERE id='story-visual'`).Scan(&branchID, &commitID); err != nil {
		t.Fatal(err)
	}

	for _, values := range [][3]string{
		{"asset-base", "character:entity:form-base:fingerprint-a", "form-base"},
		{"asset-shifted", "character:entity:form-shifted:fingerprint-b", "form-shifted"},
	} {
		if _, err := db.Conn().Exec(`INSERT INTO visual_assets
			(id,story_id,kind,subject,entity_id,canonical_entity_id,form_id,lineage_key,appearance_fingerprint,branch_id,source_commit_id)
			VALUES (?,?,'character','Mara','entity','entity',?,?,?, ?,?)`, values[0], "story-visual", values[2], values[1], values[1], branchID, commitID); err != nil {
			t.Fatalf("insert %s: %v", values[0], err)
		}
	}
	var count int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM visual_assets WHERE story_id='story-visual' AND subject='Mara'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("same-subject lineages=%d want 2", count)
	}
	var foreignKeyViolation string
	if err := db.Conn().QueryRow(`SELECT COALESCE(group_concat("table"),'') FROM pragma_foreign_key_check`).Scan(&foreignKeyViolation); err != nil {
		t.Fatal(err)
	}
	if foreignKeyViolation != "" {
		t.Fatalf("foreign key violations: %s", foreignKeyViolation)
	}
}

func TestMigrationV33RebuildPreservesLegacyAssetsVersionsAndJobs(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "visual-upgrade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Conn().Exec(`INSERT INTO stories (id,name) VALUES ('story-upgrade','Upgrade')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureStoryTimeline("story-upgrade"); err != nil {
		t.Fatal(err)
	}
	var branchID, commitID string
	if err := db.Conn().QueryRow(`SELECT active_branch_id,(SELECT head_commit_id FROM story_branches WHERE id=stories.active_branch_id) FROM stories WHERE id='story-upgrade'`).Scan(&branchID, &commitID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO story_visual_profiles (story_id,world_style_prompt) VALUES ('story-upgrade','legacy style')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO visual_assets
		(id,story_id,kind,subject,entity_id,lineage_key,appearance_fingerprint,prompt,status,url,file_path,branch_id,source_commit_id)
		VALUES ('legacy-asset','story-upgrade','character','Mara','entity-mara','before-rebuild','before-rebuild','portrait','ready','/mara.png','/tmp/mara.png',?,?)`, branchID, commitID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO visual_asset_versions (asset_id,story_id,kind,subject,url,file_path,branch_id,source_commit_id) VALUES ('legacy-asset','story-upgrade','character','Mara','/mara.png','/tmp/mara.png',?,?)`, branchID, commitID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO visual_generation_jobs (asset_id,story_id,status,branch_id,source_commit_id) VALUES ('legacy-asset','story-upgrade','succeeded',?,?)`, branchID, commitID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`DELETE FROM schema_version WHERE version=33`); err != nil {
		t.Fatal(err)
	}
	if err := db.migrate(); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"visual_assets": 1, "visual_asset_versions": 1, "visual_generation_jobs": 1} {
		var got int
		if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE story_id='story-upgrade'`).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s rows=%d want %d", table, got, want)
		}
	}
	var violations int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil {
		t.Fatal(err)
	}
	if violations != 0 {
		t.Fatalf("foreign key violations=%d", violations)
	}
}
