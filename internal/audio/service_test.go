package audio

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/storage"
)

type fakeProvider struct {
	id        string
	available bool
	audio     []byte
	request   SynthesisRequest
}

func (provider *fakeProvider) ID() string { return provider.id }
func (provider *fakeProvider) Status(context.Context) ProviderStatus {
	return ProviderStatus{ID: provider.id, Available: provider.available, Reason: "test unavailable"}
}
func (provider *fakeProvider) Voices(context.Context, string) ([]Voice, error) { return nil, nil }
func (provider *fakeProvider) Synthesize(_ context.Context, request SynthesisRequest) (SynthesisResult, error) {
	provider.request = request
	return SynthesisResult{Audio: provider.audio, Format: "wav", DurationMS: 420, Timings: json.RawMessage(`[]`)}, nil
}

func audioServiceFixture(t *testing.T) (*storage.DB, *Service, int64) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "service.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO stories(id,name,language) VALUES('story-audio-service','Audio Service','it-IT')`); err != nil {
		t.Fatal(err)
	}
	head, err := db.EnsureStoryTimeline("story-audio-service")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO sessions(id,story_id,branch_id,source_commit_id) VALUES('session-audio','story-audio-service',?,?)`, head.Branch.ID, head.Commit.ID); err != nil {
		t.Fatal(err)
	}
	revision, err := db.EnsurePromptRevision(storage.PromptRevisionInput{ProfileName: "narrator-test", PromptHash: "parent-prompt", ConfigJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	parentRun := storage.GenerationRun{TraceID: "trace-parent", StoryID: "story-audio-service", BranchID: head.Branch.ID, SourceCommitID: head.Commit.ID, Stage: "narrator", PromptRevisionID: revision.ID, PromptHash: "parent-prompt", RequestConfigJSON: `{}`, MetadataJSON: `{}`}
	if err := db.StartGenerationRun(&parentRun); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishGenerationRun(parentRun.ID, storage.GenerationCompletion{Status: "succeeded"}); err != nil {
		t.Fatal(err)
	}
	metadata, _ := json.Marshal(map[string]any{"generation": map[string]string{"run_id": parentRun.ID, "trace_id": parentRun.TraceID}, "output": map[string]any{"dialogue_blocks": []map[string]string{{"speaker_id": "npc-lyanna", "speaker": "Lyanna", "role": "npc", "text": "Lyanna ascolta."}}}})
	message := storage.ChatMessage{SessionID: "session-audio", StoryID: "story-audio-service", Turn: 1, Role: "assistant", Content: "La pioggia cade. Lyanna ascolta.", MessageType: "narrative", MetadataJSON: string(metadata), BranchID: head.Branch.ID, SourceCommitID: head.Commit.ID, CreatedAt: time.Now().UTC()}
	if err := db.AppendChatMessage(&message); err != nil {
		t.Fatal(err)
	}
	voice, err := db.UpsertVoiceProfile(storage.VoiceProfile{Provider: "local", Model: "piper", ProviderVoiceID: "paola", DisplayName: "Paola", LanguageTags: json.RawMessage(`["it-IT"]`), Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, assignment := range []storage.VoiceAssignment{
		{StoryID: "story-audio-service", Role: "narrator", VoiceProfileID: voice.ID, EnabledMode: "on", Importance: "major"},
		{StoryID: "story-audio-service", Role: "npc", EntityID: "npc-lyanna", VoiceProfileID: voice.ID, EnabledMode: "on", Importance: "supporting"},
	} {
		if _, err := db.UpsertVoiceAssignment(assignment); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.UpsertStoryTTSSettings(storage.StoryTTSSettings{StoryID: "story-audio-service", Mode: "all", DefaultLanguage: "it-IT"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.UpsertPronunciation(storage.PronunciationEntry{StoryID: "story-audio-service", LanguageTag: "it-IT", SourceText: "Lyanna", Pronunciation: "Lianna", Alphabet: "provider"}); err != nil {
		t.Fatal(err)
	}
	cfg := config.TTSConfig{OutputDir: filepath.Join(t.TempDir(), "audio"), TimeoutSeconds: 2, ProviderOrder: []string{"local"}, Local: config.TTSEndpoint{Enabled: true, BaseURL: "http://unused", Model: "piper", Voice: "paola"}}
	return db, NewService(db, cfg), message.ID
}

func TestQueueCommittedMessageUsesPolicyLineageAndDedupe(t *testing.T) {
	db, service, messageID := audioServiceFixture(t)
	defer db.Close()
	assets, err := service.QueueCommittedMessage(context.Background(), "story-audio-service", messageID)
	if err != nil || len(assets) != 2 {
		t.Fatalf("assets=%+v err=%v", assets, err)
	}
	for _, asset := range assets {
		if asset.BranchID == "" || asset.SourceCommitID == "" || len(asset.CacheKey) != 64 || asset.PronunciationRevision != 1 {
			t.Fatalf("asset lacks canonical/cache lineage: %+v", asset)
		}
	}
	again, err := service.QueueCommittedMessage(context.Background(), "story-audio-service", messageID)
	if err != nil || len(again) != 2 {
		t.Fatalf("dedupe queue=%+v err=%v", again, err)
	}
	var count int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM audio_assets`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("asset rows=%d err=%v", count, err)
	}
}

func TestProcessJobWritesCacheAndCausalTTSRun(t *testing.T) {
	db, service, messageID := audioServiceFixture(t)
	defer db.Close()
	provider := &fakeProvider{id: "local", available: true, audio: []byte("RIFF-canonical-audio")}
	service.providers["local"] = provider
	if _, err := service.QueueCommittedMessage(context.Background(), "story-audio-service", messageID); err != nil {
		t.Fatal(err)
	}
	var jobID string
	if err := db.Conn().QueryRow(`SELECT id FROM tts_jobs ORDER BY created_at,id LIMIT 1`).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	if err := service.ProcessJob(context.Background(), jobID); err != nil {
		t.Fatal(err)
	}
	job, err := db.GetTTSJob(jobID)
	if err != nil || job.Status != "succeeded" || job.GenerationRunID == "" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	asset, err := db.GetAudioAsset(job.StoryID, job.AudioAssetID)
	if err != nil || asset.Status != "ready" || asset.DurationMS != 420 {
		t.Fatalf("asset=%+v err=%v", asset, err)
	}
	if _, err := os.Stat(asset.FilePath); err != nil {
		t.Fatalf("audio file missing: %v", err)
	}
	var stage, parent string
	if err := db.Conn().QueryRow(`SELECT stage,COALESCE(parent_run_id,'') FROM generation_runs WHERE id=?`, job.GenerationRunID).Scan(&stage, &parent); err != nil || stage != "tts" || parent == "" {
		t.Fatalf("tts trace stage=%q parent=%q err=%v", stage, parent, err)
	}
	if provider.request.Text != "La pioggia cade." && provider.request.Text != "Lianna ascolta." {
		t.Fatalf("pronunciation/request text=%q", provider.request.Text)
	}
}

func TestCacheIdentityInvalidatesOnVoiceStyleLanguageAndPronunciation(t *testing.T) {
	voice := storage.VoiceProfile{Provider: "cloud", Model: "speech", ProviderVoiceID: "alloy", Version: "1"}
	base, _, _, _ := CacheIdentity(voice, "it-IT", "Ciao", json.RawMessage(`{"tone":"calm"}`), 1, "mp3", 1)
	changed, _, _, _ := CacheIdentity(voice, "it-IT", "Ciao", json.RawMessage(`{"tone":"urgent"}`), 1, "mp3", 1)
	pronunciation, _, _, _ := CacheIdentity(voice, "it-IT", "Ciao", json.RawMessage(`{"tone":"calm"}`), 1, "mp3", 2)
	if base == changed || base == pronunciation {
		t.Fatalf("cache identities did not invalidate: %s %s %s", base, changed, pronunciation)
	}
}

func TestManualCancelRetryAndCompletionRaceAreSafe(t *testing.T) {
	db, service, messageID := audioServiceFixture(t)
	defer db.Close()
	provider := &fakeProvider{id: "local", available: true, audio: []byte("RIFF-retry")}
	service.providers["local"] = provider
	if _, err := service.QueueCommittedMessage(context.Background(), "story-audio-service", messageID); err != nil {
		t.Fatal(err)
	}
	jobs, err := db.ListMessageTTSJobs("story-audio-service", messageID)
	if err != nil || len(jobs) == 0 {
		t.Fatalf("jobs=%+v err=%v", jobs, err)
	}
	job := jobs[0]
	if _, err := db.CancelTTSJob(job.StoryID, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.ProcessJob(context.Background(), job.ID); err == nil {
		t.Fatal("canceled job must not be claimable")
	}
	if _, err := db.RetryTTSJob(job.StoryID, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.ProcessJob(context.Background(), job.ID); err != nil {
		t.Fatal(err)
	}
	completed, _ := db.GetTTSJob(job.ID)
	if completed.Status != "succeeded" || completed.Attempts != 1 {
		t.Fatalf("retried job=%+v", completed)
	}

	other := jobs[len(jobs)-1]
	if other.ID == job.ID {
		return
	}
	if _, err := db.ClaimTTSJob(other.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CancelTTSJob(other.StoryID, other.ID); err != nil {
		t.Fatal(err)
	}
	err = db.CompleteTTSJob(other.ID, storage.TTSCacheEntry{CacheKey: "race", Style: json.RawMessage(`{}`)}, "/tmp/race.wav", 1, json.RawMessage(`[]`), "")
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("canceled completion err=%v", err)
	}
	asset, _ := db.GetAudioAsset(other.StoryID, other.AudioAssetID)
	if asset.Status != "cancelled" {
		t.Fatalf("canceled asset=%+v", asset)
	}
}

func TestCleanupRetainsReferencedAudioAndRemovesOnlyOrphans(t *testing.T) {
	db, service, messageID := audioServiceFixture(t)
	defer db.Close()
	service.providers["local"] = &fakeProvider{id: "local", available: true, audio: []byte("RIFF-canonical")}
	if _, err := service.QueueCommittedMessage(context.Background(), "story-audio-service", messageID); err != nil {
		t.Fatal(err)
	}
	jobs, _ := db.ListMessageTTSJobs("story-audio-service", messageID)
	if err := service.ProcessJob(context.Background(), jobs[0].ID); err != nil {
		t.Fatal(err)
	}
	ready, _ := db.GetAudioAsset(jobs[0].StoryID, jobs[0].AudioAssetID)
	orphan := filepath.Join(service.cfg.OutputDir, "orphan.wav")
	if err := os.WriteFile(orphan, []byte("RIFF-orphan"), 0o640); err != nil {
		t.Fatal(err)
	}
	dry, err := service.CleanupAudioCache(true)
	if err != nil || dry.OrphanFiles != 1 || dry.FilesRemoved != 0 {
		t.Fatalf("dry cleanup=%+v err=%v", dry, err)
	}
	if _, err := os.Stat(orphan); err != nil {
		t.Fatalf("dry run removed orphan: %v", err)
	}
	cleaned, err := service.CleanupAudioCache(false)
	if err != nil || cleaned.FilesRemoved != 1 {
		t.Fatalf("cleanup=%+v err=%v", cleaned, err)
	}
	if _, err := os.Stat(orphan); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("orphan still exists: %v", err)
	}
	if _, err := os.Stat(ready.FilePath); err != nil {
		t.Fatalf("referenced cache file removed: %v", err)
	}

	outside := filepath.Join(t.TempDir(), "outside.wav")
	if err := os.WriteFile(outside, []byte("RIFF-outside"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`UPDATE tts_cache_entries SET file_path=? WHERE cache_key=?`, outside, ready.CacheKey); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`UPDATE audio_assets SET file_path=? WHERE id=?`, outside, ready.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.ResolveAudioFile(ready.ID); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("outside-root asset resolved: %v", err)
	}
	result, err := service.CleanupAudioCache(false)
	if err != nil || result.InvalidCacheRows != 1 {
		t.Fatalf("outside cleanup=%+v err=%v", result, err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("cleanup deleted outside-root file: %v", err)
	}
}

func TestAudioJobMutationRejectsInactiveBranch(t *testing.T) {
	db, service, messageID := audioServiceFixture(t)
	defer db.Close()
	if _, err := service.QueueCommittedMessage(context.Background(), "story-audio-service", messageID); err != nil {
		t.Fatal(err)
	}
	jobs, _ := db.ListMessageTTSJobs("story-audio-service", messageID)
	head, err := db.GetActiveTimeline("story-audio-service")
	if err != nil {
		t.Fatal(err)
	}
	story, _ := db.GetStory("story-audio-service")
	if _, err := db.Conn().Exec(`INSERT INTO story_branches(id,story_id,name,fork_commit_id,head_commit_id) VALUES('branch-audio-sibling',?,?,?,?)`, story.ID, "audio sibling", head.Commit.ID, head.Commit.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`UPDATE stories SET active_branch_id='branch-audio-sibling' WHERE id=?`, story.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CancelTTSJob(story.ID, jobs[0].ID); err == nil || !strings.Contains(err.Error(), "active branch") {
		t.Fatalf("inactive-branch job cancellation err=%v", err)
	}
}
