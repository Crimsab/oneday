package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

func TestGatewayMiniGamePersistsAuthoritativeStateAndReturnsPlayerView(t *testing.T) {
	db, err := storage.Open(t.TempDir() + "/gateway-minigame.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Conn().Exec(`INSERT INTO stories (id,name) VALUES ('story-mini','Minigame')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureStoryTimeline("story-mini"); err != nil {
		t.Fatal(err)
	}
	start := gatewayMiniGameRequest{StoryID: "story-mini", Definition: engine.MiniGameDefinition{
		ID: "deduction", Kind: engine.MiniGameDeduction, Difficulty: 50,
		Answers: []string{"curator"}, Rules: map[string]string{"required_evidence": "2", "secret_note": "hidden"},
	}}
	var startOut bytes.Buffer
	startJSON, _ := json.Marshal(start)
	if err := runGatewayMiniGame(db, "start", bytes.NewReader(startJSON), &startOut); err != nil {
		t.Fatal(err)
	}
	var started gatewayMiniGameResponse
	if err := json.Unmarshal(startOut.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	if started.Error != "" || started.Instance == nil {
		t.Fatalf("start response = %+v", started)
	}
	if len(started.Instance.Definition.Answers) != 0 || started.Instance.Definition.Rules["secret_note"] != "" {
		t.Fatalf("player response leaked answers/rules: %+v", started.Instance.Definition)
	}
	record, err := db.GetMiniGameInstance("story-mini", started.Instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	authoritative, err := engine.NewMiniGameHost().Restore(record.Instance)
	if err != nil || len(authoritative.Definition.Answers) != 1 {
		t.Fatalf("authoritative answers lost: %+v err=%v", authoritative, err)
	}
	input := gatewayMiniGameRequest{StoryID: "story-mini", InstanceID: started.Instance.ID, Input: engine.MiniGameInput{Action: "submit", Value: "curator", Values: []string{"seal", "ink"}}}
	var inputOut bytes.Buffer
	inputJSON, _ := json.Marshal(input)
	if err := runGatewayMiniGame(db, "input", bytes.NewReader(inputJSON), &inputOut); err != nil {
		t.Fatal(err)
	}
	var resolved gatewayMiniGameResponse
	if err := json.Unmarshal(inputOut.Bytes(), &resolved); err != nil {
		t.Fatal(err)
	}
	if resolved.Instance == nil || resolved.Instance.Runtime.Result == nil || resolved.Instance.Runtime.Result.Outcome.Degree != contracts.OutcomeFullSuccess {
		t.Fatalf("resolved response = %+v", resolved)
	}
	auto := gatewayMiniGameRequest{StoryID: "story-mini", Selection: engine.MiniGameSelectionContext{NarrativeTags: []string{"social", "comedy", "zero-combat"}, TimingFreeOnly: true, PreferredKinds: []engine.MiniGameType{engine.MiniGameComedy}}}
	var autoOut bytes.Buffer
	autoJSON, _ := json.Marshal(auto)
	if err := runGatewayMiniGame(db, "start", bytes.NewReader(autoJSON), &autoOut); err != nil {
		t.Fatal(err)
	}
	var selected gatewayMiniGameResponse
	if err := json.Unmarshal(autoOut.Bytes(), &selected); err != nil {
		t.Fatal(err)
	}
	if selected.Instance == nil || selected.Instance.Definition.Kind != engine.MiniGameComedy || selected.Instance.Definition.Rules["selection_reason"] == "" {
		t.Fatalf("automatic selection = %+v", selected)
	}
}
