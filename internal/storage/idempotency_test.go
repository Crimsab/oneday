package storage

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestTurnIdempotencyClaimCommitAndReplay(t *testing.T) {
	db := openIdempotencyTestDB(t, "idempotency-commit.db")
	createStoryFixture(t, db, "idem-story")

	result, err := db.ClaimTurnIdempotency("idem-story", "key-1", "hash-a", "owner-a", time.Minute)
	if err != nil {
		t.Fatalf("ClaimTurnIdempotency: %v", err)
	}
	if result == nil || result.Claim == nil || result.Committed {
		t.Fatalf("claim result = %+v, want fresh claim", result)
	}

	if err := db.WithTx(func(tx *sql.Tx) error {
		return result.Claim.CommitTx(tx, `[{"type":"turn.committed"}]`)
	}); err != nil {
		t.Fatalf("CommitTx: %v", err)
	}

	replay, err := db.ClaimTurnIdempotency("idem-story", "key-1", "hash-a", "owner-b", time.Minute)
	if err != nil {
		t.Fatalf("ClaimTurnIdempotency replay: %v", err)
	}
	if replay == nil || !replay.Committed || replay.EventsJSON == "" || replay.Claim != nil {
		t.Fatalf("replay result = %+v, want committed events", replay)
	}

	eventsJSON, found, err := db.GetTurnIdempotency("idem-story", "key-1", "hash-a")
	if err != nil {
		t.Fatalf("GetTurnIdempotency: %v", err)
	}
	if !found || eventsJSON != replay.EventsJSON {
		t.Fatalf("GetTurnIdempotency found=%v json=%q, want replay json %q", found, eventsJSON, replay.EventsJSON)
	}
}

func TestTurnIdempotencyRejectsSameKeyDifferentRequestHash(t *testing.T) {
	db := openIdempotencyTestDB(t, "idempotency-conflict.db")
	createStoryFixture(t, db, "idem-conflict")

	result, err := db.ClaimTurnIdempotency("idem-conflict", "key-1", "hash-a", "owner-a", time.Minute)
	if err != nil {
		t.Fatalf("ClaimTurnIdempotency: %v", err)
	}
	if err := db.WithTx(func(tx *sql.Tx) error {
		return result.Claim.CommitTx(tx, `[{"type":"turn.committed"}]`)
	}); err != nil {
		t.Fatalf("CommitTx: %v", err)
	}

	if _, _, err := db.GetTurnIdempotency("idem-conflict", "key-1", "hash-b"); !errors.Is(err, ErrTurnIdempotencyConflict) {
		t.Fatalf("GetTurnIdempotency conflict err = %v, want ErrTurnIdempotencyConflict", err)
	}
	if _, err := db.ClaimTurnIdempotency("idem-conflict", "key-1", "hash-b", "owner-b", time.Minute); !errors.Is(err, ErrTurnIdempotencyConflict) {
		t.Fatalf("ClaimTurnIdempotency conflict err = %v, want ErrTurnIdempotencyConflict", err)
	}
}

func TestTurnIdempotencyClaimBlocksAndReclaimsExpiredOwner(t *testing.T) {
	db := openIdempotencyTestDB(t, "idempotency-reclaim.db")
	createStoryFixture(t, db, "idem-reclaim")

	first, err := db.ClaimTurnIdempotency("idem-reclaim", "key-1", "hash-a", "owner-a", 20*time.Millisecond)
	if err != nil {
		t.Fatalf("Claim first: %v", err)
	}
	if first == nil || first.Claim == nil {
		t.Fatalf("first claim = %+v, want claim", first)
	}

	if _, err := db.ClaimTurnIdempotency("idem-reclaim", "key-1", "hash-a", "owner-b", time.Minute); !errors.Is(err, ErrTurnIdempotencyInProgress) {
		t.Fatalf("second claim err = %v, want ErrTurnIdempotencyInProgress", err)
	}

	time.Sleep(35 * time.Millisecond)
	second, err := db.ClaimTurnIdempotency("idem-reclaim", "key-1", "hash-a", "owner-b", time.Minute)
	if err != nil {
		t.Fatalf("Claim after expiry: %v", err)
	}
	if second == nil || second.Claim == nil || second.Committed {
		t.Fatalf("second claim = %+v, want reclaimed claim", second)
	}

	if err := db.WithTx(func(tx *sql.Tx) error {
		return first.Claim.CommitTx(tx, `[]`)
	}); !errors.Is(err, ErrTurnIdempotencyLost) {
		t.Fatalf("stale claim CommitTx err = %v, want ErrTurnIdempotencyLost", err)
	}
}

func openIdempotencyTestDB(t *testing.T, name string) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
