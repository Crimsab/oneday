package storage

import "testing"

func TestTimelineSearchIndexesTrackContentChanges(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	head, err := db.GetActiveTimeline(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO sessions (id,story_id,branch_id,source_commit_id) VALUES ('fts-session',?,?,?)`, story.ID, head.Branch.ID, head.Commit.ID); err != nil {
		t.Fatal(err)
	}
	result, err := db.Conn().Exec(`INSERT INTO chat_messages (session_id,story_id,turn,role,content,branch_id,source_commit_id) VALUES ('fts-session',?,1,'assistant','a hidden needle',?,?)`, story.ID, head.Branch.ID, head.Commit.ID)
	if err != nil {
		t.Fatal(err)
	}
	messageID, _ := result.LastInsertId()
	if _, err := db.Conn().Exec(`INSERT INTO chapters (story_id,chapter_number,title,summary,start_turn,branch_id,source_commit_id) VALUES (?,1,'The Needle','archive summary',1,?,?)`, story.ID, head.Branch.ID, head.Commit.ID); err != nil {
		t.Fatal(err)
	}

	assertFTSCount(t, db, `SELECT COUNT(*) FROM chat_messages_fts WHERE content LIKE '%needle%'`, 1)
	assertFTSCount(t, db, `SELECT COUNT(*) FROM chapters_fts WHERE title LIKE '%Need%'`, 1)
	if _, err := db.Conn().Exec(`UPDATE chat_messages SET content='a replaced phrase' WHERE id=?`, messageID); err != nil {
		t.Fatal(err)
	}
	assertFTSCount(t, db, `SELECT COUNT(*) FROM chat_messages_fts WHERE content LIKE '%needle%'`, 0)
	assertFTSCount(t, db, `SELECT COUNT(*) FROM chat_messages_fts WHERE content LIKE '%placed%'`, 1)
	if _, err := db.Conn().Exec(`DELETE FROM chapters WHERE story_id=?`, story.ID); err != nil {
		t.Fatal(err)
	}
	assertFTSCount(t, db, `SELECT COUNT(*) FROM chapters_fts WHERE summary LIKE '%archive%'`, 0)
}

func assertFTSCount(t *testing.T, db *DB, query string, want int) {
	t.Helper()
	var count int
	if err := db.Conn().QueryRow(query).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("FTS count=%d want=%d for %s", count, want, query)
	}
}
