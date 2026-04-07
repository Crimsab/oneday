package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CreateSession inserts a new session.
func (db *DB) CreateSession(s *Session) error {
	_, err := db.conn.Exec(
		`INSERT INTO sessions (id, story_id, started_at, ended_at, summary)
         VALUES (?, ?, ?, ?, ?)`,
		s.ID, s.StoryID, s.StartedAt, s.EndedAt, s.Summary,
	)
	if err != nil {
		return fmt.Errorf("inserting session: %w", err)
	}
	return nil
}

// GetSession retrieves a session by ID.
func (db *DB) GetSession(id string) (*Session, error) {
	s := &Session{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, started_at, ended_at, summary
         FROM sessions WHERE id = ?`, id,
	).Scan(&s.ID, &s.StoryID, &s.StartedAt, &s.EndedAt, &s.Summary)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting session %s: %w", id, err)
	}
	return s, nil
}

// GetActiveSession returns the most recent open session (EndedAt IS NULL) for a story.
// Returns nil, nil if no active session exists.
func (db *DB) GetActiveSession(storyID string) (*Session, error) {
	s := &Session{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, started_at, ended_at, summary
         FROM sessions WHERE story_id = ? AND ended_at IS NULL
         ORDER BY started_at DESC LIMIT 1`, storyID,
	).Scan(&s.ID, &s.StoryID, &s.StartedAt, &s.EndedAt, &s.Summary)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting active session for story %s: %w", storyID, err)
	}
	return s, nil
}

// CloseSession sets the ended_at timestamp to now.
func (db *DB) CloseSession(id string) error {
	now := time.Now()
	_, err := db.conn.Exec(
		`UPDATE sessions SET ended_at = ? WHERE id = ?`, now, id,
	)
	if err != nil {
		return fmt.Errorf("closing session %s: %w", id, err)
	}
	return nil
}

// ListSessions returns all sessions for a story, most recent first.
func (db *DB) ListSessions(storyID string) ([]Session, error) {
	rows, err := db.conn.Query(
		`SELECT id, story_id, started_at, ended_at, summary
         FROM sessions WHERE story_id = ? ORDER BY started_at DESC`, storyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing sessions for story %s: %w", storyID, err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.StoryID, &s.StartedAt, &s.EndedAt, &s.Summary); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}
