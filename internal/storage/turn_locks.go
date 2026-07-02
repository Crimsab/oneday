package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const storyTurnLockTimeFormat = time.RFC3339Nano

// StoryTurnLock is a cross-process lock for serializing turn commits per story.
type StoryTurnLock struct {
	db      *DB
	storyID string
	owner   string
}

// AcquireStoryTurnLock waits until the caller owns the story turn lock or the
// context is canceled. Locks are lease-based so crashed clients unblock.
func (db *DB) AcquireStoryTurnLock(ctx context.Context, storyID, owner string, ttl time.Duration) (*StoryTurnLock, error) {
	storyID = strings.TrimSpace(storyID)
	owner = strings.TrimSpace(owner)
	if storyID == "" {
		return nil, errors.New("story_id is required")
	}
	if owner == "" {
		return nil, errors.New("lock owner is required")
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}

	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	for {
		acquired, err := db.tryAcquireStoryTurnLock(storyID, owner, ttl)
		if err != nil {
			return nil, err
		}
		if acquired {
			return &StoryTurnLock{db: db, storyID: storyID, owner: owner}, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("acquiring story turn lock for %s: %w", storyID, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (db *DB) tryAcquireStoryTurnLock(storyID, owner string, ttl time.Duration) (bool, error) {
	now := time.Now().UTC()
	until := now.Add(ttl)
	nowText := now.Format(storyTurnLockTimeFormat)
	untilText := until.Format(storyTurnLockTimeFormat)

	var acquired bool
	err := db.WithTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(
			`DELETE FROM story_turn_locks WHERE story_id = ? AND locked_until <= ?`,
			storyID, nowText,
		); err != nil {
			return fmt.Errorf("pruning expired story turn lock: %w", err)
		}

		result, err := tx.Exec(
			`INSERT INTO story_turn_locks (story_id, owner, acquired_at, locked_until)
             VALUES (?, ?, ?, ?)
             ON CONFLICT(story_id) DO UPDATE SET
               owner = excluded.owner,
               acquired_at = excluded.acquired_at,
               locked_until = excluded.locked_until
             WHERE story_turn_locks.locked_until <= ? OR story_turn_locks.owner = ?`,
			storyID, owner, nowText, untilText, nowText, owner,
		)
		if err != nil {
			return fmt.Errorf("acquiring story turn lock: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("checking story turn lock acquisition: %w", err)
		}
		acquired = rows > 0
		return nil
	})
	if err != nil {
		return false, err
	}
	return acquired, nil
}

// Release drops the lock only if this owner still holds it.
func (l *StoryTurnLock) Release() error {
	if l == nil || l.db == nil {
		return nil
	}
	_, err := l.db.conn.Exec(
		`DELETE FROM story_turn_locks WHERE story_id = ? AND owner = ?`,
		l.storyID, l.owner,
	)
	if err != nil {
		return fmt.Errorf("releasing story turn lock for %s: %w", l.storyID, err)
	}
	return nil
}
