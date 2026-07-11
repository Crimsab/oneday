package storage

import (
	"path/filepath"
	"testing"
)

func TestMigrationV32CreatesCausalTelemetrySchema(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "telemetry.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, table := range []string{"prompt_profiles", "prompt_profile_revisions", "generation_runs", "generation_attempts", "generation_events"} {
		var count int
		if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("table %s count=%d", table, count)
		}
	}
	var version int
	if err := db.Conn().QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version < 32 {
		t.Fatalf("schema version=%d, want at least 32", version)
	}
}

func TestGenerationTelemetryLifecycleAndPromptRevisionIdentity(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "telemetry.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	input := PromptRevisionInput{ProfileName: "narrator", TemplateVersion: "v1", PromptHash: "sha256:prompt-a", ResponseSchemaHash: "sha256:schema-a", ConfigJSON: `{"temperature":0.7}`}
	revision, err := db.EnsurePromptRevision(input)
	if err != nil {
		t.Fatal(err)
	}
	again, err := db.EnsurePromptRevision(input)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != revision.ID || again.Version != 1 {
		t.Fatalf("idempotent revision=%+v, want %+v", again, revision)
	}
	input.PromptHash = "sha256:prompt-b"
	second, err := db.EnsurePromptRevision(input)
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 || second.ID == revision.ID {
		t.Fatalf("second revision=%+v", second)
	}
	if _, err := db.Conn().Exec(`UPDATE prompt_profile_revisions SET prompt_hash='tampered' WHERE id=?`, revision.ID); err == nil {
		t.Fatal("immutable prompt revision accepted an update")
	}

	run := &GenerationRun{TraceID: "trace-1", StoryID: "story-1", BranchID: "branch-1", SourceCommitID: "commit-1", Stage: "narrator", PromptRevisionID: revision.ID, PromptHash: revision.PromptHash, RequestConfigJSON: `{"max_tokens":100}`, RequestedStreaming: true}
	if err := db.StartGenerationRun(run); err != nil {
		t.Fatal(err)
	}
	first := &GenerationAttempt{RunID: run.ID, Sequence: 1, Provider: "primary", RequestedModel: "model-a", RequestedStreaming: true, ReasoningConfigJSON: `{"effort":"high"}`}
	if err := db.StartGenerationAttempt(first); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishGenerationAttempt(first.ID, GenerationCompletion{Status: GenerationStatusFailed, DurationMs: 25, ErrorClass: "timeout", ErrorSummary: "provider timeout"}); err != nil {
		t.Fatal(err)
	}
	secondAttempt := &GenerationAttempt{RunID: run.ID, Sequence: 2, Provider: "fallback", RequestedModel: "model-b", RetryReason: "provider_fallback"}
	if err := db.StartGenerationAttempt(secondAttempt); err != nil {
		t.Fatal(err)
	}
	usage := GenerationUsage{InputTokens: 10, OutputTokens: 5, ReasoningTokens: 2, CachedInputTokens: 3, TotalTokens: 15, CostUSD: 0.004}
	if err := db.AppendGenerationEvent(run.ID, secondAttempt.ID, "first_token", `{"offset_ms":11}`); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishGenerationAttempt(secondAttempt.ID, GenerationCompletion{Status: GenerationStatusSucceeded, ResolvedModel: "model-b-2026", ObservedStreaming: true, TTFTMs: 11, DurationMs: 40, Usage: usage}); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishGenerationRun(run.ID, GenerationCompletion{Status: GenerationStatusSucceeded, ObservedStreaming: true, TTFTMs: 11, DurationMs: 65, Usage: usage}); err != nil {
		t.Fatal(err)
	}
	if err := db.BindGenerationRunMessage(run.ID, 42); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishGenerationRun(run.ID, GenerationCompletion{Status: GenerationStatusSucceeded}); err == nil {
		t.Fatal("finished run accepted a second completion")
	}

	var status string
	var attemptCount, eventCount, observed, totalTokens int
	var messageID int64
	if err := db.Conn().QueryRow(`SELECT status,message_id,observed_streaming,total_tokens FROM generation_runs WHERE id=?`, run.ID).Scan(&status, &messageID, &observed, &totalTokens); err != nil {
		t.Fatal(err)
	}
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM generation_attempts WHERE run_id=?`, run.ID).Scan(&attemptCount); err != nil {
		t.Fatal(err)
	}
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM generation_events WHERE run_id=?`, run.ID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if status != GenerationStatusSucceeded || messageID != 42 || observed != 1 || totalTokens != 15 || attemptCount != 2 || eventCount != 1 {
		t.Fatalf("run status=%s message=%d observed=%d tokens=%d attempts=%d events=%d", status, messageID, observed, totalTokens, attemptCount, eventCount)
	}
	if _, err := db.Conn().Exec(`UPDATE generation_events SET event_type='tampered' WHERE run_id=?`, run.ID); err == nil {
		t.Fatal("append-only generation event accepted an update")
	}
}
