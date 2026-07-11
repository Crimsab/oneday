package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type MiniGameInstanceRecord struct {
	ID              string          `json:"id"`
	StoryID         string          `json:"story_id"`
	Turn            int             `json:"turn"`
	ProtocolVersion int             `json:"protocol_version"`
	Kind            string          `json:"kind"`
	Phase           string          `json:"phase"`
	Instance        json.RawMessage `json:"instance"`
	BranchID        string          `json:"branch_id"`
	SourceCommitID  string          `json:"source_commit_id"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

func (db *DB) SaveMiniGameInstance(record MiniGameInstanceRecord) (*MiniGameInstanceRecord, error) {
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.StoryID) == "" || strings.TrimSpace(record.Kind) == "" {
		return nil, errors.New("minigame id, story, and kind are required")
	}
	if !json.Valid(record.Instance) {
		return nil, errors.New("minigame instance JSON is invalid")
	}
	if record.Phase != "ready" && record.Phase != "active" && record.Phase != "paused" && record.Phase != "resolved" {
		return nil, fmt.Errorf("invalid minigame phase %q", record.Phase)
	}
	head, err := db.GetActiveTimeline(record.StoryID)
	if err != nil {
		return nil, err
	}
	var saved *MiniGameInstanceRecord
	err = db.WithTx(func(tx *sql.Tx) error {
		var saveErr error
		saved, saveErr = db.SaveMiniGameInstanceTx(tx, record, head.Branch.ID, head.Commit.ID)
		return saveErr
	})
	return saved, err
}

func (db *DB) SaveMiniGameInstanceTx(tx *sql.Tx, record MiniGameInstanceRecord, branchID, sourceCommitID string) (*MiniGameInstanceRecord, error) {
	if tx == nil {
		return nil, errors.New("transaction is required")
	}
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.StoryID) == "" || strings.TrimSpace(record.Kind) == "" {
		return nil, errors.New("minigame id, story, and kind are required")
	}
	if !json.Valid(record.Instance) {
		return nil, errors.New("minigame instance JSON is invalid")
	}
	if record.Phase != "ready" && record.Phase != "active" && record.Phase != "paused" && record.Phase != "resolved" {
		return nil, fmt.Errorf("invalid minigame phase %q", record.Phase)
	}
	if strings.TrimSpace(branchID) == "" || strings.TrimSpace(sourceCommitID) == "" {
		return nil, errors.New("minigame branch and source commit are required")
	}
	result, err := tx.Exec(`INSERT INTO minigame_instances
		(id,story_id,turn,protocol_version,kind,phase,instance_json,branch_id,source_commit_id)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET turn=excluded.turn,phase=excluded.phase,instance_json=excluded.instance_json,updated_at=CURRENT_TIMESTAMP
		WHERE minigame_instances.story_id=excluded.story_id AND minigame_instances.branch_id=excluded.branch_id`,
		record.ID, record.StoryID, record.Turn, record.ProtocolVersion, record.Kind, record.Phase, record.Instance, branchID, sourceCommitID)
	if err != nil {
		return nil, fmt.Errorf("saving minigame instance: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, errors.New("minigame instance belongs to another branch")
	}
	return scanMiniGameInstance(tx.QueryRow(`SELECT id,story_id,turn,protocol_version,kind,phase,instance_json,branch_id,source_commit_id,created_at,updated_at FROM minigame_instances WHERE story_id=? AND id=? AND branch_id=?`, record.StoryID, record.ID, branchID))
}

func (db *DB) GetMiniGameInstance(storyID, instanceID string) (*MiniGameInstanceRecord, error) {
	return scanMiniGameInstance(db.conn.QueryRow(`SELECT m.id,m.story_id,m.turn,m.protocol_version,m.kind,m.phase,m.instance_json,m.branch_id,m.source_commit_id,m.created_at,m.updated_at
		FROM minigame_instances m JOIN stories s ON s.id=m.story_id
		WHERE m.story_id=? AND m.id=? AND m.branch_id=s.active_branch_id`, storyID, instanceID))
}

func (db *DB) GetActiveMiniGameInstance(storyID string) (*MiniGameInstanceRecord, error) {
	return scanMiniGameInstance(db.conn.QueryRow(`SELECT m.id,m.story_id,m.turn,m.protocol_version,m.kind,m.phase,m.instance_json,m.branch_id,m.source_commit_id,m.created_at,m.updated_at
		FROM minigame_instances m JOIN stories s ON s.id=m.story_id
		WHERE m.story_id=? AND m.branch_id=s.active_branch_id AND m.phase IN ('ready','active','paused')
		ORDER BY m.updated_at DESC,m.id DESC LIMIT 1`, storyID))
}

func (db *DB) ListRecentMiniGameInstances(storyID string, limit int) ([]MiniGameInstanceRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := db.conn.Query(`SELECT m.id,m.story_id,m.turn,m.protocol_version,m.kind,m.phase,m.instance_json,m.branch_id,m.source_commit_id,m.created_at,m.updated_at
		FROM minigame_instances m JOIN stories s ON s.id=m.story_id
		WHERE m.story_id=? AND m.branch_id=s.active_branch_id
		ORDER BY m.updated_at DESC,m.id DESC LIMIT ?`, storyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []MiniGameInstanceRecord{}
	for rows.Next() {
		var record MiniGameInstanceRecord
		if err := rows.Scan(&record.ID, &record.StoryID, &record.Turn, &record.ProtocolVersion, &record.Kind, &record.Phase, &record.Instance, &record.BranchID, &record.SourceCommitID, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func scanMiniGameInstance(row *sql.Row) (*MiniGameInstanceRecord, error) {
	var record MiniGameInstanceRecord
	if err := row.Scan(&record.ID, &record.StoryID, &record.Turn, &record.ProtocolVersion, &record.Kind, &record.Phase, &record.Instance, &record.BranchID, &record.SourceCommitID, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return nil, err
	}
	return &record, nil
}
