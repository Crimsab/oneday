package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func (db *DB) GetTurnIdempotency(storyID, key string) (string, bool, error) {
	storyID = strings.TrimSpace(storyID)
	key = strings.TrimSpace(key)
	if storyID == "" || key == "" {
		return "", false, nil
	}

	var eventsJSON string
	err := db.conn.QueryRow(
		`SELECT events_json FROM turn_idempotency WHERE story_id = ? AND idempotency_key = ?`,
		storyID, key,
	).Scan(&eventsJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("getting turn idempotency key: %w", err)
	}
	return eventsJSON, true, nil
}

func (db *DB) SaveTurnIdempotency(storyID, key, eventsJSON string) error {
	storyID = strings.TrimSpace(storyID)
	key = strings.TrimSpace(key)
	if storyID == "" || key == "" {
		return nil
	}
	_, err := db.conn.Exec(
		`INSERT INTO turn_idempotency (story_id, idempotency_key, events_json)
         VALUES (?, ?, ?)
         ON CONFLICT(story_id, idempotency_key) DO UPDATE SET events_json = excluded.events_json`,
		storyID, key, eventsJSON,
	)
	if err != nil {
		return fmt.Errorf("saving turn idempotency key: %w", err)
	}
	return nil
}
