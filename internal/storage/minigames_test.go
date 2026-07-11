package storage

import (
	"database/sql"
	"testing"
)

func TestMiniGameInstancesAreMutableOnlyOnTheirActiveBranch(t *testing.T) {
	db, err := Open(t.TempDir() + "/minigames.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Conn().Exec(`INSERT INTO stories (id,name) VALUES ('story-mini','Minigames')`); err != nil {
		t.Fatal(err)
	}
	head, err := db.EnsureStoryTimeline("story-mini")
	if err != nil {
		t.Fatal(err)
	}
	record, err := db.SaveMiniGameInstance(MiniGameInstanceRecord{ID: "mini-1", StoryID: "story-mini", Turn: 1, ProtocolVersion: 1, Kind: "deduction", Phase: "active", Instance: []byte(`{"id":"mini-1"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if record.BranchID != head.Branch.ID || record.SourceCommitID != head.Commit.ID {
		t.Fatalf("lineage = %+v head=%+v", record, head)
	}
	record.Phase = "paused"
	record.Instance = []byte(`{"id":"mini-1","phase":"paused"}`)
	if _, err := db.SaveMiniGameInstance(*record); err != nil {
		t.Fatal(err)
	}
	story, err := db.GetStory("story-mini")
	if err != nil {
		t.Fatal(err)
	}
	fork, err := db.ForkStoryBranch("story-mini", head.Commit.ID, "fork", story.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`UPDATE stories SET active_branch_id=? WHERE id='story-mini'`, fork.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetMiniGameInstance("story-mini", "mini-1"); err != sql.ErrNoRows {
		t.Fatalf("fork exposed parent mutable minigame: %v", err)
	}
	if _, err := db.SaveMiniGameInstance(*record); err == nil {
		t.Fatal("sibling branch updated original minigame")
	}
}
