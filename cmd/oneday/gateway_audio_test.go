package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

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
	if _, err := db.EnsureStoryTimeline("story-audio"); err != nil {
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
	saved := run(gatewayAudioRequest{Operation: "settings-put", StoryID: "story-audio", ClientRevision: revision, Settings: &settings})
	if saved.Error != "" || saved.Settings == nil || saved.Settings.Mode != "all" || !saved.Settings.Autoplay {
		t.Fatalf("saved response=%+v", saved)
	}
	catalog := run(gatewayAudioRequest{Operation: "catalog", Language: "it-IT"})
	if catalog.Error != "" || len(catalog.Statuses) != 2 || catalog.Statuses[0].Available || catalog.Statuses[1].Available {
		t.Fatalf("disabled catalog=%+v", catalog)
	}
}
