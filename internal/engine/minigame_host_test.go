package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/crimsab/oneday/internal/game/contracts"
)

func TestMiniGameHostPauseSerializeResumeResolveReplay(t *testing.T) {
	host := NewMiniGameHost()
	instance := NewMiniGameInstance("mini-1", "story", "branch", 4, 42, MiniGameDefinition{
		ID: "memory-basic", Kind: MiniGameMemory, Difficulty: 50,
		Sequence: []string{"up", "down", "left", "right"},
	})
	if err := host.Start(&instance); err != nil {
		t.Fatal(err)
	}
	if err := host.Apply(&instance, MiniGameInput{Action: "pause"}); err != nil {
		t.Fatal(err)
	}
	payload, err := host.Serialize(instance)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := host.Restore(payload)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Runtime.Phase != MiniGamePaused {
		t.Fatalf("restored phase = %s", restored.Runtime.Phase)
	}
	if err := host.Apply(restored, MiniGameInput{Action: "resume"}); err != nil {
		t.Fatal(err)
	}
	if err := host.Apply(restored, MiniGameInput{Action: "submit", Values: []string{"up", "down", "left", "up"}}); err != nil {
		t.Fatal(err)
	}
	if restored.Runtime.Phase != MiniGameResolved || restored.Runtime.Result == nil {
		t.Fatalf("instance did not resolve: %+v", restored.Runtime)
	}
	if restored.Runtime.Result.Outcome.Degree != contracts.OutcomeSuccessWithCost {
		t.Fatalf("degree = %s", restored.Runtime.Result.Outcome.Degree)
	}
	replayed, err := host.Replay(*restored)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(replayed.Runtime.Result, restored.Runtime.Result) {
		t.Fatalf("replay differs:\nrestored=%+v\nreplayed=%+v", restored.Runtime.Result, replayed.Runtime.Result)
	}
	encoded, _ := json.Marshal(replayed)
	if len(encoded) == 0 {
		t.Fatal("replayed instance was not serializable")
	}
}

func TestMiniGameHostRPSIsDeterministicAcrossHosts(t *testing.T) {
	definition := MiniGameDefinition{ID: "rps", Kind: MiniGameRPS, Difficulty: 50}
	resolve := func() *ChallengeResult {
		host := NewMiniGameHost()
		instance := NewMiniGameInstance("mini-rps", "story", "branch", 2, 99, definition)
		if err := host.Start(&instance); err != nil {
			t.Fatal(err)
		}
		if err := host.Apply(&instance, MiniGameInput{Action: "submit", Value: "rock"}); err != nil {
			t.Fatal(err)
		}
		return instance.Runtime.Result
	}
	first, second := resolve(), resolve()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same seed/input diverged: %+v vs %+v", first, second)
	}
}

func TestMiniGameHostRejectsInputWhilePaused(t *testing.T) {
	host := NewMiniGameHost()
	instance := NewMiniGameInstance("mini-riddle", "story", "branch", 1, 7, MiniGameDefinition{
		ID: "riddle", Kind: MiniGameRiddle, Difficulty: 40, Prompt: "What has roots?", Answers: []string{"mountain"},
	})
	if err := host.Start(&instance); err != nil {
		t.Fatal(err)
	}
	if err := host.Apply(&instance, MiniGameInput{Action: "pause"}); err != nil {
		t.Fatal(err)
	}
	if err := host.Apply(&instance, MiniGameInput{Action: "submit", Value: "mountain"}); err == nil {
		t.Fatal("paused host accepted resolution input")
	}
}
