package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const CurrentTurnSnapshotFormatVersion = 1

var (
	ErrStaleBranchHead       = errors.New("stale branch head")
	ErrBranchNotFound        = errors.New("story branch not found")
	ErrCommitNotFound        = errors.New("turn commit not found")
	ErrCommitSnapshotMissing = errors.New("turn commit snapshot is missing")
)

type TimelineHead struct {
	Branch StoryBranch `json:"branch"`
	Commit TurnCommit  `json:"commit"`
}

type CanonicalEventInput struct {
	Type        string
	PayloadJSON string
}

type AppendTurnCommitParams struct {
	CommitID       string
	StoryID        string
	BranchID       string
	ExpectedHeadID string
	CanonicalTurn  int
	Kind           string
	Message        string
	PayloadJSON    string
	Events         []CanonicalEventInput
}

type sqlQueryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

func (db *DB) EnsureStoryTimeline(storyID string) (*TimelineHead, error) {
	var head *TimelineHead
	err := db.WithTx(func(tx *sql.Tx) error {
		var err error
		head, err = db.EnsureStoryTimelineTx(tx, storyID)
		return err
	})
	return head, err
}

func (db *DB) EnsureStoryTimelineTx(tx *sql.Tx, storyID string) (*TimelineHead, error) {
	if tx == nil {
		return nil, errors.New("transaction is required")
	}
	if head, err := getActiveTimelineExec(tx, storyID); err == nil {
		return head, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var revision int64
	var turn int
	if err := tx.QueryRow(`SELECT revision FROM stories WHERE id = ?`, storyID).Scan(&revision); err != nil {
		return nil, fmt.Errorf("loading story for timeline bootstrap: %w", err)
	}
	if err := tx.QueryRow(`SELECT COALESCE((SELECT current_turn FROM world_state WHERE story_id = ?), 0)`, storyID).Scan(&turn); err != nil {
		return nil, fmt.Errorf("loading story turn for timeline bootstrap: %w", err)
	}
	now := time.Now().UTC()
	branch := StoryBranch{ID: uuid.NewString(), StoryID: storyID, Name: "main", CreatedAt: now, UpdatedAt: now}
	commit := TurnCommit{ID: uuid.NewString(), StoryID: storyID, BranchID: branch.ID, CanonicalTurn: turn, StoryRevision: revision, Kind: "root", CreatedAt: now}
	if _, err := tx.Exec(`INSERT INTO story_branches (id, story_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, branch.ID, storyID, branch.Name, now, now); err != nil {
		// A concurrent/bootstrap migration may already have won. Re-read it.
		if head, readErr := getActiveTimelineExec(tx, storyID); readErr == nil {
			return head, nil
		}
		return nil, fmt.Errorf("creating main story branch: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO turn_commits (id, story_id, branch_id, canonical_turn, story_revision, payload_hash, kind, created_at) VALUES (?, ?, ?, ?, ?, '', ?, ?)`, commit.ID, storyID, branch.ID, turn, revision, commit.Kind, now); err != nil {
		return nil, fmt.Errorf("creating root turn commit: %w", err)
	}
	if _, err := tx.Exec(`UPDATE story_branches SET head_commit_id = ? WHERE id = ?`, commit.ID, branch.ID); err != nil {
		return nil, fmt.Errorf("setting main branch head: %w", err)
	}
	if _, err := tx.Exec(`UPDATE stories SET active_branch_id = ? WHERE id = ?`, branch.ID, storyID); err != nil {
		return nil, fmt.Errorf("activating main story branch: %w", err)
	}
	branch.HeadCommitID = commit.ID
	return &TimelineHead{Branch: branch, Commit: commit}, nil
}

func (db *DB) GetActiveTimeline(storyID string) (*TimelineHead, error) {
	head, err := getActiveTimelineExec(db.conn, storyID)
	if errors.Is(err, sql.ErrNoRows) {
		return db.EnsureStoryTimeline(storyID)
	}
	return head, err
}

func getActiveTimelineExec(q sqlQueryRower, storyID string) (*TimelineHead, error) {
	var h TimelineHead
	err := q.QueryRow(`
		SELECT b.id, b.story_id, b.name, COALESCE(b.fork_commit_id,''), COALESCE(b.head_commit_id,''), b.created_at, b.updated_at,
		       c.id, c.story_id, c.branch_id, COALESCE(c.parent_commit_id,''), c.canonical_turn, c.story_revision,
		       c.payload_hash, c.kind, c.message, c.created_at
		FROM stories s
		JOIN story_branches b ON b.id = s.active_branch_id
		JOIN turn_commits c ON c.id = b.head_commit_id
		WHERE s.id = ?`, storyID).Scan(
		&h.Branch.ID, &h.Branch.StoryID, &h.Branch.Name, &h.Branch.ForkCommitID, &h.Branch.HeadCommitID, &h.Branch.CreatedAt, &h.Branch.UpdatedAt,
		&h.Commit.ID, &h.Commit.StoryID, &h.Commit.BranchID, &h.Commit.ParentCommitID, &h.Commit.CanonicalTurn, &h.Commit.StoryRevision,
		&h.Commit.PayloadHash, &h.Commit.Kind, &h.Commit.Message, &h.Commit.CreatedAt,
	)
	return &h, err
}

func (db *DB) AppendTurnCommitTx(tx *sql.Tx, p AppendTurnCommitParams) (*TurnCommit, error) {
	if tx == nil {
		return nil, errors.New("transaction is required")
	}
	if p.StoryID == "" || p.BranchID == "" || p.ExpectedHeadID == "" {
		return nil, errors.New("story, branch, and expected head are required")
	}
	if !json.Valid([]byte(p.PayloadJSON)) {
		return nil, errors.New("turn snapshot payload must be valid JSON")
	}
	var currentHead string
	if err := tx.QueryRow(`SELECT COALESCE(head_commit_id,'') FROM story_branches WHERE id=? AND story_id=?`, p.BranchID, p.StoryID).Scan(&currentHead); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBranchNotFound
		}
		return nil, err
	}
	if currentHead != p.ExpectedHeadID {
		return nil, fmt.Errorf("%w: expected %s, current %s", ErrStaleBranchHead, p.ExpectedHeadID, currentHead)
	}
	var revision int64
	if err := tx.QueryRow(`SELECT revision FROM stories WHERE id=? AND active_branch_id=?`, p.StoryID, p.BranchID).Scan(&revision); err != nil {
		return nil, fmt.Errorf("branch is not active for story: %w", err)
	}
	sum := sha256.Sum256([]byte(p.PayloadJSON))
	hash := fmt.Sprintf("sha256:%x", sum[:])
	now := time.Now().UTC()
	commitID := p.CommitID
	if commitID == "" {
		commitID = uuid.NewString()
	}
	commit := &TurnCommit{ID: commitID, StoryID: p.StoryID, BranchID: p.BranchID, ParentCommitID: currentHead, CanonicalTurn: p.CanonicalTurn, StoryRevision: revision, PayloadHash: hash, Kind: p.Kind, Message: p.Message, CreatedAt: now}
	if commit.Kind == "" {
		commit.Kind = "turn"
	}
	if _, err := tx.Exec(`INSERT INTO turn_commits (id, story_id, branch_id, parent_commit_id, canonical_turn, story_revision, payload_hash, kind, message, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, commit.ID, commit.StoryID, commit.BranchID, commit.ParentCommitID, commit.CanonicalTurn, commit.StoryRevision, commit.PayloadHash, commit.Kind, commit.Message, commit.CreatedAt); err != nil {
		return nil, fmt.Errorf("inserting turn commit: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO turn_snapshots (commit_id, story_id, format_version, payload_json, payload_hash, created_at) VALUES (?, ?, ?, ?, ?, ?)`, commit.ID, commit.StoryID, CurrentTurnSnapshotFormatVersion, p.PayloadJSON, hash, now); err != nil {
		return nil, fmt.Errorf("inserting turn snapshot: %w", err)
	}
	for i, event := range p.Events {
		payload := event.PayloadJSON
		if payload == "" {
			payload = "{}"
		}
		if event.Type == "" || !json.Valid([]byte(payload)) {
			return nil, fmt.Errorf("canonical event %d is invalid", i)
		}
		if _, err := tx.Exec(`INSERT INTO canonical_events (story_id, branch_id, commit_id, sequence, event_type, payload_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, p.StoryID, p.BranchID, commit.ID, i, event.Type, payload, now); err != nil {
			return nil, fmt.Errorf("inserting canonical event %d: %w", i, err)
		}
	}
	res, err := tx.Exec(`UPDATE story_branches SET head_commit_id=?, updated_at=? WHERE id=? AND story_id=? AND head_commit_id=?`, commit.ID, now, p.BranchID, p.StoryID, p.ExpectedHeadID)
	if err != nil {
		return nil, err
	}
	if rows, _ := res.RowsAffected(); rows != 1 {
		return nil, ErrStaleBranchHead
	}
	return commit, nil
}

func (db *DB) ListStoryBranches(storyID string) ([]StoryBranch, error) {
	rows, err := db.conn.Query(`SELECT id, story_id, name, COALESCE(fork_commit_id,''), COALESCE(head_commit_id,''), created_at, updated_at FROM story_branches WHERE story_id=? ORDER BY created_at,id`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoryBranch
	for rows.Next() {
		var b StoryBranch
		if err := rows.Scan(&b.ID, &b.StoryID, &b.Name, &b.ForkCommitID, &b.HeadCommitID, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (db *DB) ForkStoryBranch(storyID, fromCommitID, name string, expectedRevision int64) (*StoryBranch, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("branch name is required")
	}
	var branch *StoryBranch
	err := db.WithTx(func(tx *sql.Tx) error {
		var existing StoryBranch
		readErr := tx.QueryRow(`SELECT id,story_id,name,COALESCE(fork_commit_id,''),COALESCE(head_commit_id,''),created_at,updated_at FROM story_branches WHERE story_id=? AND name=?`, storyID, name).Scan(&existing.ID, &existing.StoryID, &existing.Name, &existing.ForkCommitID, &existing.HeadCommitID, &existing.CreatedAt, &existing.UpdatedAt)
		if readErr == nil && existing.ForkCommitID == fromCommitID {
			branch = &existing
			return nil
		}
		if err := db.RequireStoryRevisionTx(tx, storyID, expectedRevision); err != nil {
			return err
		}
		var turn int
		if err := tx.QueryRow(`SELECT canonical_turn FROM turn_commits WHERE id=? AND story_id=?`, fromCommitID, storyID).Scan(&turn); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrCommitNotFound
			}
			return err
		}
		branch = &StoryBranch{ID: uuid.NewString(), StoryID: storyID, Name: name, ForkCommitID: fromCommitID, HeadCommitID: fromCommitID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		_, err := tx.Exec(`INSERT INTO story_branches (id,story_id,name,fork_commit_id,head_commit_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`, branch.ID, branch.StoryID, branch.Name, branch.ForkCommitID, branch.HeadCommitID, branch.CreatedAt, branch.UpdatedAt)
		if err != nil {
			var existing StoryBranch
			readErr := tx.QueryRow(`SELECT id,story_id,name,COALESCE(fork_commit_id,''),COALESCE(head_commit_id,''),created_at,updated_at FROM story_branches WHERE story_id=? AND name=?`, storyID, name).Scan(&existing.ID, &existing.StoryID, &existing.Name, &existing.ForkCommitID, &existing.HeadCommitID, &existing.CreatedAt, &existing.UpdatedAt)
			if readErr == nil && existing.ForkCommitID == fromCommitID {
				branch = &existing
				return nil
			}
			return fmt.Errorf("creating branch: %w", err)
		}
		_, err = db.BumpStoryRevisionTx(tx, storyID)
		return err
	})
	return branch, err
}

func (db *DB) RenameStoryBranch(storyID, branchID, name string, expectedRevision int64) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("branch name is required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		var current string
		if err := tx.QueryRow(`SELECT name FROM story_branches WHERE id=? AND story_id=?`, branchID, storyID).Scan(&current); err != nil {
			return ErrBranchNotFound
		}
		if current == name {
			return nil
		}
		if err := db.RequireStoryRevisionTx(tx, storyID, expectedRevision); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE story_branches SET name=?,updated_at=? WHERE id=? AND story_id=?`, name, time.Now().UTC(), branchID, storyID); err != nil {
			return err
		}
		_, err := db.BumpStoryRevisionTx(tx, storyID)
		return err
	})
}

func (db *DB) GetTurnSnapshot(commitID string) (*TurnSnapshot, error) {
	var s TurnSnapshot
	err := db.conn.QueryRow(`SELECT commit_id,story_id,format_version,payload_json,payload_hash,created_at FROM turn_snapshots WHERE commit_id=?`, commitID).Scan(&s.CommitID, &s.StoryID, &s.FormatVersion, &s.PayloadJSON, &s.PayloadHash, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommitNotFound
	}
	return &s, err
}

// SealTurnSnapshotTx fills the one allowed bootstrap gap: migrated/new root
// commits are created before a complete story materialization exists. Once a
// snapshot is present, the commit and payload are immutable.
func (db *DB) SealTurnSnapshotTx(tx *sql.Tx, commitID, storyID, payloadJSON string) error {
	if !json.Valid([]byte(payloadJSON)) {
		return errors.New("turn snapshot payload must be valid JSON")
	}
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM turn_snapshots WHERE commit_id=?`, commitID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	sum := sha256.Sum256([]byte(payloadJSON))
	hash := fmt.Sprintf("sha256:%x", sum[:])
	now := time.Now().UTC()
	if _, err := tx.Exec(`INSERT INTO turn_snapshots (commit_id,story_id,format_version,payload_json,payload_hash,created_at) VALUES (?,?,?,?,?,?)`, commitID, storyID, CurrentTurnSnapshotFormatVersion, payloadJSON, hash, now); err != nil {
		return err
	}
	res, err := tx.Exec(`UPDATE turn_commits SET payload_hash=? WHERE id=? AND story_id=? AND payload_hash=''`, hash, commitID, storyID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows != 1 {
		return errors.New("root commit was already sealed inconsistently")
	}
	return nil
}

// BindPendingLineageTx assigns rows produced by the current transaction to the
// commit that will own them. Existing rows already bound to older commits are
// never rewritten.
func (db *DB) BindPendingLineageTx(tx *sql.Tx, storyID, branchID, commitID string) error {
	for _, table := range []string{"sessions", "chat_messages", "chapters", "rag_chunks", "saves", "combat_log", "visual_assets", "visual_asset_versions", "visual_generation_jobs", "challenge_runs"} {
		stmt := fmt.Sprintf(`UPDATE %s SET branch_id=?, source_commit_id=? WHERE story_id=? AND source_commit_id='' AND (branch_id='' OR branch_id=?)`, table)
		if _, err := tx.Exec(stmt, branchID, commitID, storyID, branchID); err != nil {
			return fmt.Errorf("binding %s lineage: %w", table, err)
		}
	}
	return nil
}
