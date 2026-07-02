package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

type fakeTurnProvider struct {
	calls int
}

func (f *fakeTurnProvider) Name() string { return "fake-turn" }

func (f *fakeTurnProvider) Complete(_ context.Context, _ ai.Request) (ai.Response, error) {
	f.calls++
	return ai.Response{
		Model: "fake-model",
		Content: `{
			"narrative": "You step into the market and the rain thins.",
			"choices": [
				{"id": 1, "text": "Ask the lantern seller about the map."},
				{"id": 2, "text": "Follow the wet footprints."}
			],
			"mood": "curious",
			"location": "Market",
			"state_changes": {"location": "Market"}
		}`,
	}, nil
}

func TestInProcessTurnServiceSubmitActionProducesOrderedEvents(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-browser", 0)

	provider := &fakeTurnProvider{}
	router, err := ai.NewRouter([]ai.Provider{provider})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	cfg := config.Default()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.RAG.Enabled = false
	cfg.AI.ASCIIArt.Enabled = false

	svc := NewInProcessTurnService(cfg, db, router)
	snapshot, err := svc.Snapshot(context.Background(), "story-browser")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot.SessionID == "" {
		t.Fatal("Snapshot session id is empty")
	}

	stream, err := svc.SubmitAction(context.Background(), contracts.SubmitActionRequest{
		StoryID:        "story-browser",
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		IdempotencyKey: "request-1",
		Action: contracts.PlayerAction{
			Kind: contracts.ActionKindFreeText,
			Text: "I look around the market.",
		},
	})
	if err != nil {
		t.Fatalf("SubmitAction: %v", err)
	}

	events := collectTurnEvents(stream)
	wantTypes := []contracts.TurnEventType{
		contracts.EventTurnStarted,
		contracts.EventNarrativeFinal,
		contracts.EventStateDelta,
		contracts.EventChoicesUpdated,
		contracts.EventTurnCommitted,
	}
	if len(events) != len(wantTypes) {
		t.Fatalf("event count = %d, want %d: %#v", len(events), len(wantTypes), events)
	}
	for i, want := range wantTypes {
		if events[i].Type != want {
			t.Fatalf("event[%d] type = %q, want %q", i, events[i].Type, want)
		}
	}

	world, err := db.GetWorldState("story-browser")
	if err != nil {
		t.Fatalf("GetWorldState: %v", err)
	}
	if world.CurrentTurn != 1 {
		t.Fatalf("world current turn = %d, want 1", world.CurrentTurn)
	}
	if world.CurrentLocation != "Market" {
		t.Fatalf("world current location = %q, want Market", world.CurrentLocation)
	}

	replay, err := svc.SubmitAction(context.Background(), contracts.SubmitActionRequest{
		StoryID:        "story-browser",
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		IdempotencyKey: "request-1",
		Action: contracts.PlayerAction{
			Kind: contracts.ActionKindFreeText,
			Text: "I look around the market.",
		},
	})
	if err != nil {
		t.Fatalf("SubmitAction replay: %v", err)
	}
	if got := collectTurnEvents(replay); len(got) != len(events) {
		t.Fatalf("replay event count = %d, want %d", len(got), len(events))
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1 after idempotent replay", provider.calls)
	}

	providerAfterRestart := &fakeTurnProvider{}
	routerAfterRestart, err := ai.NewRouter([]ai.Provider{providerAfterRestart})
	if err != nil {
		t.Fatalf("NewRouter after restart: %v", err)
	}
	svcAfterRestart := NewInProcessTurnService(cfg, db, routerAfterRestart)
	persistentReplay, err := svcAfterRestart.SubmitAction(context.Background(), contracts.SubmitActionRequest{
		StoryID:        "story-browser",
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		IdempotencyKey: "request-1",
		Action: contracts.PlayerAction{
			Kind: contracts.ActionKindFreeText,
			Text: "I look around the market.",
		},
	})
	if err != nil {
		t.Fatalf("SubmitAction persistent replay: %v", err)
	}
	if got := collectTurnEvents(persistentReplay); len(got) != len(events) {
		t.Fatalf("persistent replay event count = %d, want %d", len(got), len(events))
	}
	if providerAfterRestart.calls != 0 {
		t.Fatalf("provider calls after restart = %d, want 0", providerAfterRestart.calls)
	}
}

func collectTurnEvents(stream <-chan contracts.TurnEvent) []contracts.TurnEvent {
	var events []contracts.TurnEvent
	for event := range stream {
		events = append(events, event)
	}
	return events
}

func newTurnServiceTestDB(t *testing.T, root string) *storage.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(root, "service-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createTurnServiceStory(t *testing.T, db *storage.DB, storyID string, currentTurn int) {
	t.Helper()
	now := time.Now()
	if err := db.CreateStory(&storage.Story{
		ID:              storyID,
		Name:            "Browser Story",
		Description:     "A browser-playable story",
		Genre:           "fantasy",
		Tone:            "mysterious",
		Language:        "en",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	if err := db.CreateCharacter(&storage.Character{
		ID:               "char-" + storyID,
		StoryID:          storyID,
		Name:             "Mira",
		Background:       "Test character",
		StatsJSON:        `{"vitals":{"hp":{"current":10,"max":10}},"skills":{}}`,
		TraitsJSON:       `[]`,
		SkillsJSON:       `{}`,
		InventoryJSON:    `[]`,
		KnownRecipesJSON: `[]`,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	if err := db.CreateWorldState(&storage.WorldState{
		ID:                   "world-" + storyID,
		StoryID:              storyID,
		CurrentLocation:      "Harbor",
		KnownLocationsJSON:   `["Harbor"]`,
		GlobalEventsJSON:     `[]`,
		FactionStandingsJSON: `{}`,
		StoryHooksJSON:       `[]`,
		WorldReactionsJSON:   `[]`,
		PlayerGuidanceJSON:   `[]`,
		CurrentChapter:       1,
		CurrentTurn:          currentTurn,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}
}

func decodeTurnPayload[T any](t *testing.T, event contracts.TurnEvent) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(event.Payload, &out); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return out
}
