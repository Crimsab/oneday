package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/storage"
)

func TestGatewayAudioSettingsAreRevisionGuardedAndProvidersDegradeExplicitly(t *testing.T) {
	db, err := storage.Open(t.TempDir() + "/gateway-audio.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Conn().Exec(`INSERT INTO stories(id,name,language) VALUES('story-audio','Audio','it-IT')`); err != nil {
		t.Fatal(err)
	}
	head, err := db.EnsureStoryTimeline("story-audio")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO sessions(id,story_id,branch_id,source_commit_id) VALUES('session-audio','story-audio',?,?)`, head.Branch.ID, head.Commit.ID); err != nil {
		t.Fatal(err)
	}
	message := storage.ChatMessage{SessionID: "session-audio", StoryID: "story-audio", Turn: 1, Role: "assistant", Content: "Canonical audio export.", MessageType: "narrative", MetadataJSON: `{}`, BranchID: head.Branch.ID, SourceCommitID: head.Commit.ID, CreatedAt: time.Now().UTC()}
	if err := db.AppendChatMessage(&message); err != nil {
		t.Fatal(err)
	}
	voice, err := db.UpsertVoiceProfile(storage.VoiceProfile{Provider: "cloud", Model: "speech", ProviderVoiceID: "alloy", DisplayName: "Alloy", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.QueueAudioAsset(storage.AudioAsset{StoryID: "story-audio", SourceMessageID: message.ID, SegmentIndex: 0, SegmentKind: "narrator", VoiceProfileID: voice.ID, Provider: "cloud", Model: "speech", ProviderVoiceID: "alloy", LanguageTag: "it-IT", Text: message.Content, TextHash: "hash", CacheKey: "cache-export", Style: json.RawMessage(`{}`), Speed: 1, OutputFormat: "mp3", Status: "ready", FilePath: "/secret/internal/cache.mp3", Timings: json.RawMessage(`[]`)}, storage.TTSJob{}); err != nil {
		t.Fatal(err)
	}
	revision, err := db.GetStoryRevision("story-audio")
	if err != nil {
		t.Fatal(err)
	}
	run := func(request gatewayAudioRequest) gatewayAudioResponse {
		payload, _ := json.Marshal(request)
		var output bytes.Buffer
		if err := runGatewayAudio(context.Background(), config.Config{}, db, bytes.NewReader(payload), &output); err != nil {
			t.Fatal(err)
		}
		var response gatewayAudioResponse
		if err := json.Unmarshal(output.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	settings := storage.StoryTTSSettings{Mode: "all", Autoplay: true, DefaultLanguage: "it-IT", ProviderPolicy: json.RawMessage(`{}`)}
	stale := run(gatewayAudioRequest{Operation: "settings-put", StoryID: "story-audio", ClientRevision: revision - 1, Settings: &settings})
	if !strings.Contains(stale.Error, "stale story revision") {
		t.Fatalf("stale response=%+v", stale)
	}
	if stale.ErrorDetail == nil || stale.ErrorDetail.Code != gatewayCodeStaleRequest {
		t.Fatalf("stale error detail=%+v", stale.ErrorDetail)
	}
	saved := run(gatewayAudioRequest{Operation: "settings-put", StoryID: "story-audio", ClientRevision: revision, Settings: &settings})
	if saved.Error != "" || saved.Settings == nil || saved.Settings.Mode != "all" || !saved.Settings.Autoplay {
		t.Fatalf("saved response=%+v", saved)
	}
	catalog := run(gatewayAudioRequest{Operation: "catalog", Language: "it-IT"})
	if catalog.Error != "" || len(catalog.Statuses) != 2 || catalog.Statuses[0].Available || catalog.Statuses[1].Available {
		t.Fatalf("disabled catalog=%+v", catalog)
	}
	entry := storage.PronunciationEntry{LanguageTag: "it-IT", SourceText: "Lyanna", Pronunciation: "Lianna", Alphabet: "provider"}
	pronunciation := run(gatewayAudioRequest{Operation: "pronunciation-put", StoryID: "story-audio", ClientRevision: revision, Pronunciation: &entry})
	if pronunciation.Error != "" || pronunciation.Pronunciation == nil || pronunciation.Pronunciation.Revision != 1 {
		t.Fatalf("pronunciation response=%+v", pronunciation)
	}
	listed := run(gatewayAudioRequest{Operation: "pronunciations-get", StoryID: "story-audio", Language: "it-IT"})
	if listed.Error != "" || len(listed.Pronunciations) != 1 {
		t.Fatalf("pronunciation list=%+v", listed)
	}
	exported := run(gatewayAudioRequest{Operation: "export", StoryID: "story-audio"})
	if exported.Error != "" || exported.Export == nil || exported.Export.Format != "oneday-audio-manifest-v1" || len(exported.Export.Pronunciations) != 1 || len(exported.Export.Assets) != 1 {
		t.Fatalf("audio export=%+v", exported)
	}
	if exported.Export.Assets[0].FilePath != "" || exported.Export.Assets[0].URL == "" {
		t.Fatalf("audio export leaked internal path or missed media URL: %+v", exported.Export.Assets[0])
	}
}
