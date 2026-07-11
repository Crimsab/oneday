package storage

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newTimelineStory(t *testing.T) (*DB, *Story) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "timeline.db"))
	if err != nil {
		t.Fatal(err)
	}
	s := &Story{ID: "story-timeline", Name: "Timeline", SettingJSON: "{}", StatsSchemaJSON: "{}", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.CreateStory(s); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db, s
}

func TestCreateStoryBootstrapsMainBranchAndRootCommit(t *testing.T) {
	db, s := newTimelineStory(t)
	defer db.Close()
	head, err := db.GetActiveTimeline(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if head.Branch.Name != "main" || head.Branch.ID == "" || head.Commit.ID == "" || head.Branch.HeadCommitID != head.Commit.ID {
		t.Fatalf("bad head: %#v", head)
	}
	if s.ActiveBranchID != head.Branch.ID {
		t.Fatalf("story active branch=%q want %q", s.ActiveBranchID, head.Branch.ID)
	}
}

func TestChatRowsDefaultToActiveBranchAndBindToCommit(t *testing.T) {
	db, s := newTimelineStory(t)
	defer db.Close()
	head, err := db.GetActiveTimeline(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.CreateSession(&Session{ID: "legacy-session", StoryID: s.ID, StartedAt: now}); err != nil {
		t.Fatal(err)
	}
	message := &ChatMessage{SessionID: "legacy-session", StoryID: s.ID, Turn: 1, Role: "assistant", Content: "before timeline", MessageType: "narrative", MetadataJSON: "{}", CreatedAt: now}
	if err := db.AppendChatMessage(message); err != nil {
		t.Fatal(err)
	}
	if err := db.WithTx(func(tx *sql.Tx) error {
		return db.BindPendingLineageTx(tx, s.ID, head.Branch.ID, head.Commit.ID)
	}); err != nil {
		t.Fatal(err)
	}
	var branchID, commitID string
	if err := db.Conn().QueryRow(`SELECT branch_id,source_commit_id FROM chat_messages WHERE id=?`, message.ID).Scan(&branchID, &commitID); err != nil {
		t.Fatal(err)
	}
	if branchID != head.Branch.ID || commitID != head.Commit.ID {
		t.Fatalf("bootstrap lineage=(%q,%q), want (%q,%q)", branchID, commitID, head.Branch.ID, head.Commit.ID)
	}
}

func TestAppendTurnCommitUsesExpectedHeadAndImmutableSnapshot(t *testing.T) {
	db, s := newTimelineStory(t)
	defer db.Close()
	head, _ := db.GetActiveTimeline(s.ID)
	var committed *TurnCommit
	err := db.WithTx(func(tx *sql.Tx) error {
		var err error
		committed, err = db.AppendTurnCommitTx(tx, AppendTurnCommitParams{StoryID: s.ID, BranchID: head.Branch.ID, ExpectedHeadID: head.Commit.ID, CanonicalTurn: 1, PayloadJSON: `{"turn":1}`, Events: []CanonicalEventInput{{Type: "turn.committed", PayloadJSON: `{"choice":"north"}`}}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if committed.ParentCommitID != head.Commit.ID || committed.PayloadHash == "" {
		t.Fatalf("bad commit: %#v", committed)
	}
	snap, err := db.GetTurnSnapshot(committed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if snap.PayloadJSON != `{"turn":1}` || snap.PayloadHash != committed.PayloadHash {
		t.Fatalf("bad snapshot: %#v", snap)
	}
	if _, err := db.Conn().Exec(`UPDATE turn_commits SET message='tampered' WHERE id=?`, committed.ID); err == nil {
		t.Fatal("immutable commit accepted an update")
	}
	if _, err := db.Conn().Exec(`UPDATE turn_snapshots SET payload_json='{}' WHERE commit_id=?`, committed.ID); err == nil {
		t.Fatal("immutable snapshot accepted an update")
	}
	err = db.WithTx(func(tx *sql.Tx) error {
		_, err := db.AppendTurnCommitTx(tx, AppendTurnCommitParams{StoryID: s.ID, BranchID: head.Branch.ID, ExpectedHeadID: head.Commit.ID, CanonicalTurn: 2, PayloadJSON: `{"turn":2}`})
		return err
	})
	if !errors.Is(err, ErrStaleBranchHead) {
		t.Fatalf("stale append err=%v", err)
	}
}

func TestSiblingBranchesShareParentAndAreIdempotentByName(t *testing.T) {
	db, s := newTimelineStory(t)
	defer db.Close()
	head, _ := db.GetActiveTimeline(s.ID)
	rev, _ := db.GetStoryRevision(s.ID)
	a, err := db.ForkStoryBranch(s.ID, head.Commit.ID, "left", rev)
	if err != nil {
		t.Fatal(err)
	}
	rev, _ = db.GetStoryRevision(s.ID)
	b, err := db.ForkStoryBranch(s.ID, head.Commit.ID, "right", rev)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID || a.ForkCommitID != head.Commit.ID || b.ForkCommitID != head.Commit.ID {
		t.Fatalf("invalid siblings: %#v %#v", a, b)
	}
	again, err := db.ForkStoryBranch(s.ID, head.Commit.ID, "left", rev-1)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != a.ID {
		t.Fatalf("idempotent fork returned %q want %q", again.ID, a.ID)
	}
}

func TestSiblingCheckoutRestoresExactStateWithoutDeletingDescendants(t *testing.T) {
	db, s := newTimelineStory(t)
	defer db.Close()
	now := time.Now()
	if err := db.CreateCharacter(&Character{ID: "hero", StoryID: s.ID, Name: "Hero", StatsJSON: "{}", TraitsJSON: "[]", SkillsJSON: "{}", InventoryJSON: "[]", KnownRecipesJSON: "[]", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateWorldState(&WorldState{ID: "world", StoryID: s.ID, CurrentLocation: "Origin", KnownLocationsJSON: "[]", GlobalEventsJSON: "[]", FactionStandingsJSON: "{}", CurrentChapter: 1, CurrentTurn: 0, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	head, _ := db.GetActiveTimeline(s.ID)
	var mainCommit *TurnCommit
	err := db.WithTx(func(tx *sql.Tx) error {
		rootPayload, err := db.CaptureTimelineMaterializationTx(tx, s.ID, head.Branch.ID)
		if err != nil {
			return err
		}
		if err := db.SealTurnSnapshotTx(tx, head.Commit.ID, s.ID, rootPayload); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE world_state SET current_turn=1,current_location='Shared parent' WHERE story_id=?`, s.ID); err != nil {
			return err
		}
		payload, err := db.CaptureTimelineMaterializationTx(tx, s.ID, head.Branch.ID)
		if err != nil {
			return err
		}
		mainCommit, err = db.AppendTurnCommitTx(tx, AppendTurnCommitParams{StoryID: s.ID, BranchID: head.Branch.ID, ExpectedHeadID: head.Commit.ID, CanonicalTurn: 1, PayloadJSON: payload})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	rev, _ := db.GetStoryRevision(s.ID)
	left, err := db.ForkStoryBranch(s.ID, mainCommit.ID, "left", rev)
	if err != nil {
		t.Fatal(err)
	}
	rev, _ = db.GetStoryRevision(s.ID)
	right, err := db.ForkStoryBranch(s.ID, mainCommit.ID, "right", rev)
	if err != nil {
		t.Fatal(err)
	}
	rev, _ = db.GetStoryRevision(s.ID)
	if _, err := db.CheckoutStoryBranch(s.ID, left.ID, rev); err != nil {
		t.Fatal(err)
	}
	var leftCommit *TurnCommit
	err = db.WithTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE world_state SET current_turn=2,current_location='Left future' WHERE story_id=?`, s.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO sessions (id,story_id,branch_id,source_commit_id) VALUES ('left-session',?,?,?)`, s.ID, left.ID, mainCommit.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO chat_messages (session_id,story_id,turn,role,content,branch_id) VALUES ('left-session',?,1,'assistant','left-only message',?)`, s.ID, left.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO chapters (story_id,chapter_number,title,start_turn,branch_id) VALUES (?,2,'Left chapter',1,?)`, s.ID, left.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO rag_chunks (story_id,text,turn_start,turn_end,branch_id) VALUES (?,'left-only memory',1,2,?)`, s.ID, left.ID); err != nil {
			return err
		}
		commitID := "left-descendant"
		if err := db.BindPendingLineageTx(tx, s.ID, left.ID, commitID); err != nil {
			return err
		}
		payload, err := db.CaptureTimelineMaterializationTx(tx, s.ID, left.ID)
		if err != nil {
			return err
		}
		leftCommit, err = db.AppendTurnCommitTx(tx, AppendTurnCommitParams{CommitID: commitID, StoryID: s.ID, BranchID: left.ID, ExpectedHeadID: mainCommit.ID, CanonicalTurn: 2, PayloadJSON: payload})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	rev, _ = db.GetStoryRevision(s.ID)
	if _, err := db.CheckoutStoryBranch(s.ID, right.ID, rev); err != nil {
		t.Fatal(err)
	}
	world, err := db.GetWorldState(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if world.CurrentTurn != 1 || world.CurrentLocation != "Shared parent" {
		t.Fatalf("right checkout state=%#v", world)
	}
	for _, table := range []string{"chat_messages", "chapters", "rag_chunks"} {
		var count int
		if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE story_id=?`, s.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s leaked %d left-branch rows", table, count)
		}
	}
	var exists int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM turn_commits WHERE id=?`, leftCommit.ID).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists != 1 {
		t.Fatal("left descendant was destroyed by sibling checkout")
	}
	branches, _ := db.ListStoryBranches(s.ID)
	if len(branches) != 3 {
		t.Fatalf("branches=%d want 3", len(branches))
	}
}
