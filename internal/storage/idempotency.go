package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTurnIdempotencyInProgress = errors.New("turn idempotency key is in progress")
	ErrTurnIdempotencyLost       = errors.New("turn idempotency claim lost")
	ErrTurnIdempotencyConflict   = errors.New("turn idempotency key belongs to a different request")
)

const turnIdempotencyTimeFormat = time.RFC3339Nano

type TurnIdempotencyClaim struct {
	db          *DB
	storyID     string
	key         string
	requestHash string
	owner       string
}

type TurnIdempotencyClaimResult struct {
	Claim      *TurnIdempotencyClaim
	EventsJSON string
	Committed  bool
}

func (db *DB) GetTurnIdempotency(storyID, key, requestHash string) (string, bool, error) {
	storyID = strings.TrimSpace(storyID)
	key = strings.TrimSpace(key)
	requestHash = strings.TrimSpace(requestHash)
	if storyID == "" || key == "" {
		return "", false, nil
	}
	if requestHash == "" {
		return "", false, errors.New("idempotency request hash is required")
	}

	var eventsJSON string
	var status string
	var existingHash string
	err := db.conn.QueryRow(
		`SELECT events_json, status, request_hash FROM turn_idempotency WHERE story_id = ? AND idempotency_key = ?`,
		storyID, key,
	).Scan(&eventsJSON, &status, &existingHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("getting turn idempotency key: %w", err)
	}
	if existingHash != requestHash {
		return "", false, fmt.Errorf("%w: story %s key %s", ErrTurnIdempotencyConflict, storyID, key)
	}
	if status != "committed" || strings.TrimSpace(eventsJSON) == "" {
		return "", false, nil
	}
	return eventsJSON, true, nil
}

func (db *DB) ClaimTurnIdempotency(storyID, key, requestHash, owner string, ttl time.Duration) (*TurnIdempotencyClaimResult, error) {
	storyID = strings.TrimSpace(storyID)
	key = strings.TrimSpace(key)
	requestHash = strings.TrimSpace(requestHash)
	owner = strings.TrimSpace(owner)
	if storyID == "" || key == "" {
		return &TurnIdempotencyClaimResult{}, nil
	}
	if requestHash == "" {
		return nil, errors.New("idempotency request hash is required")
	}
	if owner == "" {
		return nil, errors.New("idempotency owner is required")
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}

	now := time.Now().UTC()
	nowText := now.Format(turnIdempotencyTimeFormat)
	untilText := now.Add(ttl).Format(turnIdempotencyTimeFormat)
	result := &TurnIdempotencyClaimResult{}

	err := db.WithTx(func(tx *sql.Tx) error {
		inserted, err := insertTurnIdempotencyClaim(tx, storyID, key, requestHash, owner, nowText, untilText)
		if err != nil {
			return err
		}
		if inserted {
			result.Claim = &TurnIdempotencyClaim{db: db, storyID: storyID, key: key, requestHash: requestHash, owner: owner}
			return nil
		}

		var eventsJSON, status, existingHash, existingOwner, lockedUntil string
		if err := tx.QueryRow(
			`SELECT events_json, status, request_hash, owner, locked_until
             FROM turn_idempotency
             WHERE story_id = ? AND idempotency_key = ?`,
			storyID, key,
		).Scan(&eventsJSON, &status, &existingHash, &existingOwner, &lockedUntil); err != nil {
			return fmt.Errorf("loading turn idempotency claim: %w", err)
		}
		if existingHash != requestHash {
			return fmt.Errorf("%w: story %s key %s", ErrTurnIdempotencyConflict, storyID, key)
		}

		if status == "committed" && strings.TrimSpace(eventsJSON) != "" {
			result.EventsJSON = eventsJSON
			result.Committed = true
			return nil
		}

		if status == "running" && existingOwner != owner && !idempotencyExpired(lockedUntil, now) {
			return fmt.Errorf("%w: story %s key %s", ErrTurnIdempotencyInProgress, storyID, key)
		}

		claimed, err := reclaimTurnIdempotencyClaim(tx, storyID, key, requestHash, owner, nowText, untilText)
		if err != nil {
			return err
		}
		if !claimed {
			return fmt.Errorf("%w: story %s key %s", ErrTurnIdempotencyInProgress, storyID, key)
		}
		result.Claim = &TurnIdempotencyClaim{db: db, storyID: storyID, key: key, requestHash: requestHash, owner: owner}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func insertTurnIdempotencyClaim(tx *sql.Tx, storyID, key, requestHash, owner, nowText, untilText string) (bool, error) {
	res, err := tx.Exec(
		`INSERT OR IGNORE INTO turn_idempotency
           (story_id, idempotency_key, request_hash, events_json, status, owner, locked_until, created_at, updated_at, error)
         VALUES (?, ?, ?, '', 'running', ?, ?, ?, ?, '')`,
		storyID, key, requestHash, owner, untilText, nowText, nowText,
	)
	if err != nil {
		return false, fmt.Errorf("claiming turn idempotency key: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("checking turn idempotency insert: %w", err)
	}
	return rows > 0, nil
}

func reclaimTurnIdempotencyClaim(tx *sql.Tx, storyID, key, requestHash, owner, nowText, untilText string) (bool, error) {
	res, err := tx.Exec(
		`UPDATE turn_idempotency
         SET events_json = '', status = 'running', owner = ?, locked_until = ?, updated_at = ?, error = ''
         WHERE story_id = ? AND idempotency_key = ?
           AND request_hash = ?
           AND status != 'committed'
           AND (status = 'failed' OR owner = ? OR locked_until = '' OR locked_until <= ?)`,
		owner, untilText, nowText, storyID, key, requestHash, owner, nowText,
	)
	if err != nil {
		return false, fmt.Errorf("reclaiming turn idempotency key: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("checking turn idempotency reclaim: %w", err)
	}
	return rows > 0, nil
}

func idempotencyExpired(lockedUntil string, now time.Time) bool {
	lockedUntil = strings.TrimSpace(lockedUntil)
	if lockedUntil == "" {
		return true
	}
	parsed, err := time.Parse(turnIdempotencyTimeFormat, lockedUntil)
	if err != nil {
		return true
	}
	return !parsed.After(now)
}

func (c *TurnIdempotencyClaim) CommitTx(tx *sql.Tx, eventsJSON string) error {
	if c == nil || tx == nil {
		return nil
	}
	res, err := tx.Exec(
		`UPDATE turn_idempotency
         SET events_json = ?, status = 'committed', owner = '', locked_until = '', updated_at = CURRENT_TIMESTAMP, error = ''
         WHERE story_id = ? AND idempotency_key = ? AND request_hash = ? AND owner = ? AND status = 'running'`,
		eventsJSON, c.storyID, c.key, c.requestHash, c.owner,
	)
	if err != nil {
		return fmt.Errorf("committing turn idempotency key: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking turn idempotency commit: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: story %s key %s", ErrTurnIdempotencyLost, c.storyID, c.key)
	}
	return nil
}

func (c *TurnIdempotencyClaim) Renew(ttl time.Duration) error {
	if c == nil || c.db == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	untilText := time.Now().UTC().Add(ttl).Format(turnIdempotencyTimeFormat)
	res, err := c.db.conn.Exec(
		`UPDATE turn_idempotency
         SET locked_until = ?, updated_at = CURRENT_TIMESTAMP
         WHERE story_id = ? AND idempotency_key = ? AND request_hash = ? AND owner = ? AND status = 'running'`,
		untilText, c.storyID, c.key, c.requestHash, c.owner,
	)
	if err != nil {
		return fmt.Errorf("renewing turn idempotency key: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking turn idempotency renewal: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: story %s key %s", ErrTurnIdempotencyLost, c.storyID, c.key)
	}
	return nil
}

func (c *TurnIdempotencyClaim) Fail(cause error) error {
	if c == nil || c.db == nil {
		return nil
	}
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	_, err := c.db.conn.Exec(
		`UPDATE turn_idempotency
         SET status = 'failed', owner = '', locked_until = '', updated_at = CURRENT_TIMESTAMP, error = ?
         WHERE story_id = ? AND idempotency_key = ? AND request_hash = ? AND owner = ? AND status = 'running'`,
		msg, c.storyID, c.key, c.requestHash, c.owner,
	)
	if err != nil {
		return fmt.Errorf("failing turn idempotency key: %w", err)
	}
	return nil
}

// ClearTurnIdempotencyTx removes branch-local replay cache for a story. Load and
// rollback must do this atomically with the restored canonical state.
func (db *DB) ClearTurnIdempotencyTx(tx *sql.Tx, storyID string) error {
	storyID = strings.TrimSpace(storyID)
	if tx == nil || storyID == "" {
		return nil
	}
	if _, err := tx.Exec(`DELETE FROM turn_idempotency WHERE story_id = ?`, storyID); err != nil {
		return fmt.Errorf("clearing turn idempotency for %s: %w", storyID, err)
	}
	return nil
}

func (db *DB) ClearTurnIdempotency(storyID string) error {
	return db.WithTx(func(tx *sql.Tx) error {
		return db.ClearTurnIdempotencyTx(tx, storyID)
	})
}
