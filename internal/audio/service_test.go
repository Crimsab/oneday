package audio

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
