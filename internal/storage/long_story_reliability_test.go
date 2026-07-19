package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

const defaultLongStoryTurns = 250

// TestLongStoryReliability is provider-free and deterministic. CI exercises
// 250 turns; scheduled verification can set ONEDAY_LONG_STORY_TURNS=1000.
func TestLongStoryReliability(t *testing.T) {
	turns := longStoryTurns(t)
	db, err := Open(filepath.Join(t.TempDir(), "long-story.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	storyID := "reliability-story"
	now := time.Unix(0, 0).UTC()
	story := &Story{ID: storyID, Name: "Reliability story", SettingJSON: `{}`, StatsSchemaJSON: `{}`, CreatedAt: now, UpdatedAt: now}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("create story: %v", err)
	}
	if err := db.CreateCharacter(&Character{ID: "reliability-hero", StoryID: storyID, Name: "Mara", StatsJSON: `{}`, TraitsJSON: `[]`, SkillsJSON: `{}`, InventoryJSON: `[]`, KnownRecipesJSON: `[]`, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("create protagonist: %v", err)
	}
	if err := db.CreateWorldState(&WorldState{ID: "reliability-world", StoryID: storyID, CurrentLocation: "Harbor", CurrentChapter: 1, CurrentTurn: 0, UpdatedAt: now}); err != nil {
		t.Fatalf("create world state: %v", err)
	}
	head, err := db.GetActiveTimeline(storyID)
	if err != nil {
		t.Fatalf("root timeline: %v", err)
	}
	if err := db.WithTx(func(tx *sql.Tx) error {
		return db.EnsureTurnSnapshotTx(tx, head.Commit.ID, storyID, head.Branch.ID)
	}); err != nil {
		t.Fatalf("seal root snapshot: %v", err)
	}
	if err := db.CreateSession(&Session{ID: "reliability-session", StoryID: storyID, StartedAt: now, Summary: "deterministic long-story session"}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	commits := make([]string, 0, turns)
	for turn := 1; turn <= turns; turn++ {
		if turn == 128 {
			rewindAt := commits[74] // 75th committed turn, a real rewind point.
			revision, err := db.GetStoryRevision(storyID)
			if err != nil {
				t.Fatalf("revision before rewind: %v", err)
			}
			if _, err := db.ForkAndCheckoutStoryBranch(storyID, rewindAt, "rewind-at-75", revision); err != nil {
				t.Fatalf("rewind at turn %d: %v", turn, err)
			}
		}
		if turn == 192 {
			head, err := db.GetActiveTimeline(storyID)
			if err != nil {
				t.Fatalf("head before alternate branch: %v", err)
			}
			revision, err := db.GetStoryRevision(storyID)
			if err != nil {
				t.Fatalf("revision before alternate branch: %v", err)
			}
			alternate, err := db.ForkStoryBranch(storyID, head.Commit.ID, "alternate-ending", revision)
			if err != nil {
				t.Fatalf("fork alternate at turn %d: %v", turn, err)
			}
			revision, err = db.GetStoryRevision(storyID)
			if err != nil {
				t.Fatalf("revision before alternate checkout: %v", err)
			}
			if _, err := db.CheckoutStoryBranch(storyID, alternate.ID, revision); err != nil {
				t.Fatalf("checkout alternate at turn %d: %v", turn, err)
			}
		}

		commitID := fmt.Sprintf("reliability-commit-%04d", turn)
		head, err = db.GetActiveTimeline(storyID)
		if err != nil {
			t.Fatalf("head at turn %d: %v", turn, err)
		}
		if err := appendReliabilityTurn(db, storyID, "reliability-session", head, commitID, turn, now); err != nil {
			t.Fatalf("append turn %d: %v", turn, err)
		}
		commits = append(commits, commitID)
	}

	assertSQLiteHealthy(t, db.Conn())
	assertLongStoryLineage(t, db, storyID, turns)
}

func longStoryTurns(t *testing.T) int {
	t.Helper()
	raw := os.Getenv("ONEDAY_LONG_STORY_TURNS")
	if raw == "" {
		return defaultLongStoryTurns
	}
	turns, err := strconv.Atoi(raw)
	if err != nil || turns < 192 || turns > 1000 {
		t.Fatalf("ONEDAY_LONG_STORY_TURNS must be an integer from 192 through 1000, got %q", raw)
	}
	return turns
}

func appendReliabilityTurn(db *DB, storyID, sessionID string, head *TimelineHead, commitID string, turn int, now time.Time) error {
	return db.WithTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`UPDATE world_state SET current_turn=?,updated_at=? WHERE story_id=?`, turn, now, storyID); err != nil {
			return err
		}
		message := &ChatMessage{
			SessionID: sessionID, StoryID: storyID, Turn: turn, Role: "assistant",
			Content: fmt.Sprintf("Turn %d: the harbor remembers.", turn), MessageType: "narrative", MetadataJSON: `{}`,
			CreatedAt: now, BranchID: head.Branch.ID, SourceCommitID: commitID,
		}
		if err := db.AppendChatMessageTx(tx, message); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO rag_chunks(story_id,text,chunk_type,turn_start,turn_end,embedding,branch_id,source_commit_id) VALUES(?,?,?,?,?,?,?,?)`, storyID, fmt.Sprintf("Memory %d", turn), "summary", turn, turn, []byte{byte(turn % 251)}, head.Branch.ID, commitID); err != nil {
			return err
		}
		if turn%50 == 0 {
			assetID := fmt.Sprintf("reliability-asset-%04d", turn)
			if _, err := tx.Exec(`INSERT INTO visual_assets(id,story_id,kind,subject,lineage_key,appearance_fingerprint,branch_id,source_commit_id) VALUES(?,?,?,?,?,?,?,?)`, assetID, storyID, "scene", fmt.Sprintf("Harbor %d", turn), assetID, assetID, head.Branch.ID, commitID); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO visual_asset_versions(asset_id,story_id,kind,subject,branch_id,source_commit_id) VALUES(?,?,?,?,?,?)`, assetID, storyID, "scene", fmt.Sprintf("Harbor %d", turn), head.Branch.ID, commitID); err != nil {
				return err
			}
		}
		payload, err := db.CaptureTimelineMaterializationTx(tx, storyID, head.Branch.ID)
		if err != nil {
			return err
		}
		_, err = db.AppendTurnCommitTx(tx, AppendTurnCommitParams{
			CommitID: commitID, StoryID: storyID, BranchID: head.Branch.ID, ExpectedHeadID: head.Commit.ID,
			CanonicalTurn: turn, Kind: "reliability", Message: fmt.Sprintf("turn %d", turn), PayloadJSON: payload,
			Events: []CanonicalEventInput{{Type: "canon.memory", PayloadJSON: fmt.Sprintf(`{"turn":%d}`, turn)}},
		})
		if err != nil {
			return err
		}
		if turn%25 != 0 {
			return nil
		}
		save := &SaveSnapshot{
			ID: fmt.Sprintf("reliability-save-%04d", turn), StoryID: storyID, Name: fmt.Sprintf("Turn %d", turn), Turn: turn, Chapter: 1, Location: "Harbor",
			CharacterJSON:  fmt.Sprintf(`{"id":"reliability-hero","story_id":%q,"name":"Mara"}`, storyID),
			WorldStateJSON: fmt.Sprintf(`{"id":"reliability-world","story_id":%q,"current_turn":%d,"current_chapter":1}`, storyID, turn),
			SessionID:      sessionID, MetadataJSON: fmt.Sprintf(`{"kind":"autosave","branch_id":%q,"commit_id":%q}`, head.Branch.ID, commitID), CreatedAt: now,
			BranchID: head.Branch.ID, SourceCommitID: commitID,
		}
		return db.CreateSaveTx(tx, save)
	})
}

func assertLongStoryLineage(t *testing.T, db *DB, storyID string, turns int) {
	t.Helper()
	for _, check := range []struct {
		name  string
		query string
	}{
		{"canonical events", `SELECT COUNT(*) FROM canonical_events WHERE story_id=?`},
		{"committed memories", `SELECT COUNT(*) FROM rag_chunks r WHERE r.story_id=? AND r.branch_id!='' AND r.source_commit_id IN (SELECT id FROM turn_commits WHERE story_id=r.story_id)`},
		{"committed saves", `SELECT COUNT(*) FROM saves s WHERE s.story_id=? AND s.branch_id!='' AND s.source_commit_id IN (SELECT id FROM turn_commits WHERE story_id=s.story_id)`},
		{"committed visual assets", `SELECT COUNT(*) FROM visual_assets a WHERE a.story_id=? AND a.branch_id!='' AND a.source_commit_id IN (SELECT id FROM turn_commits WHERE story_id=a.story_id)`},
		{"committed visual versions", `SELECT COUNT(*) FROM visual_asset_versions v WHERE v.story_id=? AND v.branch_id!='' AND v.source_commit_id IN (SELECT id FROM turn_commits WHERE story_id=v.story_id)`},
	} {
		var count int
		if err := db.Conn().QueryRow(check.query, storyID).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", check.name, err)
		}
		if count == 0 {
			t.Fatalf("%s were lost", check.name)
		}
		if check.name == "canonical events" && count != turns {
			t.Fatalf("canonical events=%d, want %d", count, turns)
		}
	}
	var branches, currentTurn int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM story_branches WHERE story_id=?`, storyID).Scan(&branches); err != nil {
		t.Fatal(err)
	}
	if branches < 3 {
		t.Fatalf("branches=%d, want main, rewind, and alternate", branches)
	}
	if err := db.Conn().QueryRow(`SELECT current_turn FROM world_state WHERE story_id=?`, storyID).Scan(&currentTurn); err != nil {
		t.Fatal(err)
	}
	if currentTurn != turns {
		t.Fatalf("current turn=%d, want %d", currentTurn, turns)
	}
}
