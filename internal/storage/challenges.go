package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/crimsab/oneday/internal/game/contracts"
)

type ChallengeRun struct {
	ID             string
	StoryID        string
	SessionID      string
	Turn           int
	BranchID       string
	SourceCommitID string
	Instance       contracts.ChallengeInstance
	Resolution     contracts.ChallengeResolution
}

func (db *DB) RecordChallengeResolutionTx(tx *sql.Tx, storyID, sessionID, branchID string, turn int, instance contracts.ChallengeInstance, resolution contracts.ChallengeResolution) error {
	if tx == nil {
		return errors.New("transaction is required")
	}
	if storyID == "" || branchID == "" || instance.ID == "" {
		return errors.New("story, branch, and challenge instance are required")
	}
	if err := instance.Validate(); err != nil {
		return err
	}
	definitionJSON, err := json.Marshal(instance.Definition)
	if err != nil {
		return err
	}
	instanceJSON, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	inputJSON, err := json.Marshal(resolution.Input)
	if err != nil {
		return err
	}
	resolutionJSON, err := json.Marshal(resolution)
	if err != nil {
		return err
	}
	outcomeJSON, err := json.Marshal(resolution.Outcome)
	if err != nil {
		return err
	}
	modifiersJSON, _ := json.Marshal(resolution.Outcome.Modifiers)
	timingJSON, _ := json.Marshal(instance.Timing)
	costsJSON, _ := json.Marshal(resolution.Outcome.Costs)
	deltasJSON, _ := json.Marshal(resolution.Outcome.StateDeltas)
	_, err = tx.Exec(`INSERT INTO challenge_runs
		(id,story_id,session_id,turn,protocol_version,definition_json,instance_json,input_json,resolution_json,outcome_json,degree,difficulty,seed,roll,total,margin,modifiers_json,timing_policy_json,costs_json,state_deltas_json,branch_id,source_commit_id)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'')`,
		instance.ID, storyID, sessionID, turn, instance.ProtocolVersion, definitionJSON, instanceJSON, inputJSON, resolutionJSON, outcomeJSON,
		resolution.Outcome.Degree, resolution.Outcome.Difficulty, resolution.Outcome.Seed, resolution.Outcome.Roll, resolution.Outcome.Total, resolution.Outcome.Margin,
		modifiersJSON, timingJSON, costsJSON, deltasJSON, branchID)
	if err != nil {
		return fmt.Errorf("recording challenge resolution: %w", err)
	}
	return nil
}

func (db *DB) GetChallengeRun(id string) (*ChallengeRun, error) {
	var run ChallengeRun
	var instanceJSON, resolutionJSON string
	err := db.conn.QueryRow(`SELECT id,story_id,session_id,turn,branch_id,source_commit_id,instance_json,resolution_json FROM challenge_runs WHERE id=?`, id).Scan(&run.ID, &run.StoryID, &run.SessionID, &run.Turn, &run.BranchID, &run.SourceCommitID, &instanceJSON, &resolutionJSON)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(instanceJSON), &run.Instance); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(resolutionJSON), &run.Resolution); err != nil {
		return nil, err
	}
	return &run, nil
}

// RecordChallengeResolutionAtHead persists a sub-session challenge against the
// current immutable head (combat/crafting/social do not advance a story turn).
func (db *DB) RecordChallengeResolutionAtHead(storyID, sessionID string, turn int, instance contracts.ChallengeInstance, resolution contracts.ChallengeResolution) error {
	return db.WithTx(func(tx *sql.Tx) error {
		head, err := getActiveTimelineExec(tx, storyID)
		if err != nil {
			return err
		}
		instance.BranchID = head.Branch.ID
		instance.StoryID = storyID
		if err := db.RecordChallengeResolutionTx(tx, storyID, sessionID, head.Branch.ID, turn, instance, resolution); err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE challenge_runs SET source_commit_id=? WHERE id=? AND source_commit_id=''`, head.Commit.ID, instance.ID)
		return err
	})
}
