package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

func TestServerHealthAndStories(t *testing.T) {
	db := newServerTestDB(t)
	srv := New(db)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	httpSrv := httptest.NewServer(mux)
	defer httpSrv.Close()

	resp, err := http.Get(httpSrv.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}

	resp, err = http.Get(httpSrv.URL + "/api/stories")
	if err != nil {
		t.Fatalf("GET stories: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stories status = %d", resp.StatusCode)
	}
}

func TestServerSnapshot(t *testing.T) {
	db := newServerTestDB(t)
	now := time.Now()
	story := &storage.Story{ID: "story-1", Name: "Browser Story", CreatedAt: now, UpdatedAt: now}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	if err := db.CreateCharacter(&storage.Character{ID: "char-1", StoryID: story.ID, Name: "Mira", StatsJSON: "{}", TraitsJSON: "[]", SkillsJSON: "{}", InventoryJSON: "[]", KnownRecipesJSON: "[]", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	if err := db.CreateWorldState(&storage.WorldState{ID: "world-1", StoryID: story.ID, CurrentLocation: "Harbor", CurrentTurn: 3, CurrentChapter: 1, UpdatedAt: now}); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	srv := New(db)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	httpSrv := httptest.NewServer(mux)
	defer httpSrv.Close()

	resp, err := http.Get(httpSrv.URL + "/api/stories/story-1/snapshot")
	if err != nil {
		t.Fatalf("GET snapshot: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("snapshot status = %d", resp.StatusCode)
	}
}

func TestServerSubmitTurn(t *testing.T) {
	db := newServerTestDB(t)
	turns := &fakeTurnService{}
	srv := NewWithTurnService(db, turns)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	httpSrv := httptest.NewServer(mux)
	defer httpSrv.Close()

	body := bytes.NewBufferString(`{
		"session_id": "session-1",
		"client_turn": 2,
		"idempotency_key": "request-1",
		"action": {"kind": "free_text", "text": "Open the gate"}
	}`)
	resp, err := http.Post(httpSrv.URL+"/api/stories/story-1/turns", "application/json", body)
	if err != nil {
		t.Fatalf("POST turn: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("turn status = %d", resp.StatusCode)
	}

	var payload struct {
		Events []contracts.TurnEvent `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Events) != 1 || payload.Events[0].Type != contracts.EventTurnCommitted {
		t.Fatalf("events = %#v, want one turn.committed event", payload.Events)
	}
	if turns.lastRequest.StoryID != "story-1" {
		t.Fatalf("story id = %q, want route story id", turns.lastRequest.StoryID)
	}
}

func newServerTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "server-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type fakeTurnService struct {
	lastRequest contracts.SubmitActionRequest
}

func (f *fakeTurnService) Snapshot(_ context.Context, storyID string) (*contracts.GameSnapshot, error) {
	return &contracts.GameSnapshot{StoryID: storyID, SessionID: "session-1", Turn: 2, Location: "Harbor"}, nil
}

func (f *fakeTurnService) SubmitAction(_ context.Context, req contracts.SubmitActionRequest) (<-chan contracts.TurnEvent, error) {
	f.lastRequest = req
	event, err := contracts.NewTurnEvent("event-1", req.StoryID, req.SessionID, req.ClientTurn, contracts.EventTurnCommitted, map[string]string{"ok": "true"})
	if err != nil {
		return nil, err
	}
	out := make(chan contracts.TurnEvent, 1)
	out <- event
	close(out)
	return out, nil
}
