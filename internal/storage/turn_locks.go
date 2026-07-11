package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const storyTurnLockTimeFormat = time.RFC3339Nano

var ErrStoryTurnLockLost = errors.New("story turn lock lost")

// StoryTurnLock is a cross-process lock for serializing turn commits per story.
type StoryTurnLock struct {
	db      *DB
	storyID string
	owner   string
}

type StoryTurnLockHeartbeat struct {
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
	err    error
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
			if isSQLiteBusy(err) {
				select {
				case <-ctx.Done():
					return nil, fmt.Errorf("acquiring story turn lock for %s: %w", storyID, ctx.Err())
				case <-ticker.C:
					continue
				}
			}
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

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLITE_BUSY") || strings.Contains(msg, "database is locked")
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

// StoryID returns the story protected by this lock.
func (l *StoryTurnLock) StoryID() string {
	if l == nil {
		return ""
	}
	return l.storyID
}

// Renew extends the lock lease only if the same owner still holds it.
func (l *StoryTurnLock) Renew(ttl time.Duration) error {
	if l == nil || l.db == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	until := time.Now().UTC().Add(ttl).Format(storyTurnLockTimeFormat)
	var result sql.Result
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		result, err = l.db.conn.Exec(
			`UPDATE story_turn_locks SET locked_until = ? WHERE story_id = ? AND owner = ?`,
			until, l.storyID, l.owner,
		)
		if err == nil {
			break
		}
		if !isSQLiteBusy(err) {
			return fmt.Errorf("renewing story turn lock for %s: %w", l.storyID, err)
		}
		time.Sleep(time.Duration(attempt+1) * 50 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("renewing story turn lock for %s after busy retries: %w", l.storyID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking story turn lock renewal for %s: %w", l.storyID, err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: story %s owner %s", ErrStoryTurnLockLost, l.storyID, l.owner)
	}
	return nil
}

// StartHeartbeat renews the lease periodically until Stop is called or renewal
// fails. Stop returns the first renewal error, if any.
func (l *StoryTurnLock) StartHeartbeat(ctx context.Context, interval, ttl time.Duration) *StoryTurnLockHeartbeat {
	if interval <= 0 {
		interval = ttl / 3
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	heartbeatCtx, cancel := context.WithCancel(ctx)
	hb := &StoryTurnLockHeartbeat{
		cancel: cancel,
		done:   make(chan struct{}),
	}
	go func() {
		defer close(hb.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				if err := l.Renew(ttl); err != nil {
					hb.setErr(err)
					cancel()
					return
				}
			}
		}
	}()
	return hb
}

func (h *StoryTurnLockHeartbeat) setErr(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.err == nil {
		h.err = err
	}
}

func (h *StoryTurnLockHeartbeat) Err() error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

func (h *StoryTurnLockHeartbeat) Stop() error {
	if h == nil {
		return nil
	}
	h.cancel()
	<-h.done
	return h.Err()
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
