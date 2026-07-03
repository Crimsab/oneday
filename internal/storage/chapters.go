package storage

import (
	"database/sql"
	"errors"
	"fmt"
)

// CreateChapter inserts a new chapter record.
func (db *DB) CreateChapter(ch *Chapter) error {
	return createChapterExec(db.conn, ch)
}

func (db *DB) CreateChapterTx(tx *sql.Tx, ch *Chapter) error {
	return createChapterExec(tx, ch)
}

func createChapterExec(exec sqlExecer, ch *Chapter) error {
	result, err := exec.Exec(
		`INSERT INTO chapters (story_id, chapter_number, title, summary, start_turn, end_turn, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ch.StoryID, ch.ChapterNumber, ch.Title, ch.Summary, ch.StartTurn, ch.EndTurn, ch.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting chapter: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting chapter id: %w", err)
	}
	ch.ID = id
	return nil
}

// GetChapter retrieves a specific chapter by story and chapter number.
func (db *DB) GetChapter(storyID string, chapterNumber int) (*Chapter, error) {
	ch := &Chapter{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, chapter_number, title, summary, start_turn, end_turn, created_at
         FROM chapters WHERE story_id = ? AND chapter_number = ?`,
		storyID, chapterNumber,
	).Scan(&ch.ID, &ch.StoryID, &ch.ChapterNumber, &ch.Title, &ch.Summary, &ch.StartTurn, &ch.EndTurn, &ch.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting chapter %d for story %s: %w", chapterNumber, storyID, err)
	}
	return ch, nil
}

// ListChapters returns all chapters for a story, ordered by chapter number.
func (db *DB) ListChapters(storyID string) ([]Chapter, error) {
	rows, err := db.conn.Query(
		`SELECT id, story_id, chapter_number, title, summary, start_turn, end_turn, created_at
         FROM chapters WHERE story_id = ? ORDER BY chapter_number ASC`,
		storyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing chapters for story %s: %w", storyID, err)
	}
	defer rows.Close()

	var chapters []Chapter
	for rows.Next() {
		var ch Chapter
		if err := rows.Scan(&ch.ID, &ch.StoryID, &ch.ChapterNumber, &ch.Title, &ch.Summary, &ch.StartTurn, &ch.EndTurn, &ch.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning chapter: %w", err)
		}
		chapters = append(chapters, ch)
	}
	return chapters, rows.Err()
}

// UpdateChapterEnd sets the end_turn and summary for a chapter.
func (db *DB) UpdateChapterEnd(storyID string, chapterNumber int, endTurn int, summary string) error {
	return updateChapterEndExec(db.conn, storyID, chapterNumber, endTurn, summary)
}

func (db *DB) UpdateChapterEndTx(tx *sql.Tx, storyID string, chapterNumber int, endTurn int, summary string) error {
	return updateChapterEndExec(tx, storyID, chapterNumber, endTurn, summary)
}

func updateChapterEndExec(exec sqlExecer, storyID string, chapterNumber int, endTurn int, summary string) error {
	_, err := exec.Exec(
		`UPDATE chapters SET end_turn = ?, summary = ? WHERE story_id = ? AND chapter_number = ?`,
		endTurn, summary, storyID, chapterNumber,
	)
	if err != nil {
		return fmt.Errorf("updating chapter end for chapter %d story %s: %w", chapterNumber, storyID, err)
	}
	return nil
}

// GetCurrentChapter returns the latest (open) chapter for a story (end_turn IS NULL).
// Returns nil, nil if no open chapter exists.
func (db *DB) GetCurrentChapter(storyID string) (*Chapter, error) {
	ch := &Chapter{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, chapter_number, title, summary, start_turn, end_turn, created_at
         FROM chapters WHERE story_id = ? AND end_turn IS NULL
         ORDER BY chapter_number DESC LIMIT 1`,
		storyID,
	).Scan(&ch.ID, &ch.StoryID, &ch.ChapterNumber, &ch.Title, &ch.Summary, &ch.StartTurn, &ch.EndTurn, &ch.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting current chapter for story %s: %w", storyID, err)
	}
	return ch, nil
}

// UpdateChapterTitle updates the title of a chapter.
func (db *DB) UpdateChapterTitle(storyID string, chapterNumber int, title string) error {
	return updateChapterTitleExec(db.conn, storyID, chapterNumber, title)
}

func (db *DB) UpdateChapterTitleTx(tx *sql.Tx, storyID string, chapterNumber int, title string) error {
	return updateChapterTitleExec(tx, storyID, chapterNumber, title)
}

func updateChapterTitleExec(exec sqlExecer, storyID string, chapterNumber int, title string) error {
	_, err := exec.Exec(
		`UPDATE chapters SET title = ? WHERE story_id = ? AND chapter_number = ?`,
		title, storyID, chapterNumber,
	)
	if err != nil {
		return fmt.Errorf("updating chapter title for chapter %d story %s: %w", chapterNumber, storyID, err)
	}
	return nil
}
