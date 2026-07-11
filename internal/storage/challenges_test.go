package storage

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/crimsab/oneday/internal/game/contracts"
)

func TestChallengeResolutionPersistsAndBindsImmutableLineage(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "challenge.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Conn().Exec(`INSERT INTO stories (id,name) VALUES ('story-challenge','Challenge')`); err != nil {
		t.Fatal(err)
	}
	head, err := db.EnsureStoryTimeline("story-challenge")
	if err != nil {
		t.Fatal(err)
	}
	instance := contracts.ChallengeInstance{ProtocolVersion: 1, ID: "challenge-1", StoryID: "story-challenge", BranchID: head.Branch.ID, Turn: 1, Definition: contracts.ChallengeDefinition{ID: "door", Kind: "action", Difficulty: 50}, Seed: 42, Policy: contracts.OutcomePolicy{ID: "balanced"}}
	resolution := contracts.ChallengeResolution{ProtocolVersion: 1, InstanceID: instance.ID, Input: contracts.ChallengeInput{Intent: "open"}, Outcome: contracts.OutcomeEnvelope{Version: 1, Degree: contracts.OutcomeFullSuccess, Difficulty: 50, Seed: 42, Roll: 60, Total: 60, Margin: 10}}
	err = db.WithTx(func(tx *sql.Tx) error {
		if err := db.RecordChallengeResolutionTx(tx, instance.StoryID, "session-1", head.Branch.ID, 1, instance, resolution); err != nil {
			return err
		}
		return db.BindPendingLineageTx(tx, instance.StoryID, head.Branch.ID, head.Commit.ID)
	})
	if err != nil {
		t.Fatal(err)
	}
	run, err := db.GetChallengeRun(instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if run.SourceCommitID != head.Commit.ID || run.Resolution.Outcome.Degree != contracts.OutcomeFullSuccess {
		t.Fatalf("bad persisted run: %+v", run)
	}
	if _, err := db.Conn().Exec(`UPDATE challenge_runs SET degree='hard_failure' WHERE id=?`, instance.ID); err == nil {
		t.Fatal("immutable challenge run accepted update")
	}
}

func TestMigrationV31CreatesChallengeProtocolSchema(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "v31.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, column := range []string{"instance_json", "input_json", "resolution_json", "outcome_json", "seed", "modifiers_json", "timing_policy_json", "costs_json", "state_deltas_json", "branch_id", "source_commit_id"} {
		ok, err := db.columnExists("challenge_runs", column)
		if err != nil || !ok {
			t.Fatalf("missing challenge_runs.%s exists=%v err=%v", column, ok, err)
		}
	}
}
