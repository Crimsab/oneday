package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

type fakeTurnProvider struct {
	mu    sync.Mutex
	calls int
}

func (f *fakeTurnProvider) Name() string { return "fake-turn" }

func (f *fakeTurnProvider) Complete(_ context.Context, _ ai.Request) (ai.Response, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	return ai.Response{
		Model: "fake-model",
		Content: `{
			"narrative": "You step into the market and the rain thins.",
			"choices": [
				{"id": 1, "text": "Ask the lantern seller about the map.", "intent": "social", "risk": "medium", "scope": "npc", "certainty": "uncertain", "related_stats": ["cha", "wis"]},
				{"id": 2, "text": "Follow the wet footprints.", "intent": "observe", "risk": "low", "scope": "environment", "certainty": "safe", "related_stats": ["wis"]}
			],
			"mood": "curious",
			"location": "Market",
			"state_changes": {"location": "Market"}
		}`,
	}, nil
}

func (f *fakeTurnProvider) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type blockingTurnProvider struct {
	mu      sync.Mutex
	calls   int
	entered chan struct{}
	release chan struct{}
}

func (f *blockingTurnProvider) Name() string { return "blocking-turn" }

func (f *blockingTurnProvider) Complete(ctx context.Context, _ ai.Request) (ai.Response, error) {
	f.mu.Lock()
	f.calls++
	calls := f.calls
	f.mu.Unlock()
	if calls == 1 {
		close(f.entered)
		select {
		case <-ctx.Done():
			return ai.Response{}, ctx.Err()
		case <-f.release:
		}
	}
	return ai.Response{
		Model: "blocking-model",
		Content: `{
			"narrative": "The lock resolves into one canonical turn.",
			"choices": [{"id": 1, "text": "Continue.", "intent": "act", "risk": "low", "scope": "scene", "certainty": "safe"}],
			"mood": "steady",
			"location": "Harbor"
		}`,
	}, nil
}

func (f *blockingTurnProvider) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
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
		ClientRevision: snapshot.Revision,
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
	choicesPayload := decodeTurnPayload[struct {
		Choices []contracts.ChoiceView `json:"choices"`
	}](t, events[3])
	if len(choicesPayload.Choices) != 2 {
		t.Fatalf("choices payload count = %d, want 2", len(choicesPayload.Choices))
	}
	firstChoice := choicesPayload.Choices[0]
	if firstChoice.Intent != "social" || firstChoice.Risk != "medium" || firstChoice.Scope != "npc" || firstChoice.Certainty != "uncertain" {
		t.Fatalf("choice metadata was not preserved: %+v", firstChoice)
	}
	if got := firstChoice.RelatedStats; len(got) != 2 || got[0] != "cha" || got[1] != "wis" {
		t.Fatalf("choice related stats = %#v, want [cha wis]", got)
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
		ClientRevision: snapshot.Revision,
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
	if provider.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1 after idempotent replay", provider.callCount())
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
		ClientRevision: snapshot.Revision,
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
	if providerAfterRestart.callCount() != 0 {
		t.Fatalf("provider calls after restart = %d, want 0", providerAfterRestart.callCount())
	}
}

func TestInProcessTurnServiceConcurrentSubmissionsSerializeAcrossInstances(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-race", 0)

	provider := &blockingTurnProvider{entered: make(chan struct{}), release: make(chan struct{})}
	router, err := ai.NewRouter([]ai.Provider{provider})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	cfg := config.Default()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.RAG.Enabled = false
	cfg.AI.ASCIIArt.Enabled = false

	svcA := NewInProcessTurnService(cfg, db, router)
	svcB := NewInProcessTurnService(cfg, db, router)
	snapshot, err := svcA.Snapshot(context.Background(), "story-race")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	req := func(key string) contracts.SubmitActionRequest {
		return contracts.SubmitActionRequest{
			StoryID:        "story-race",
			SessionID:      snapshot.SessionID,
			ClientTurn:     snapshot.Turn,
			ClientRevision: snapshot.Revision,
			IdempotencyKey: key,
			Action: contracts.PlayerAction{
				Kind: contracts.ActionKindFreeText,
				Text: "I test the lock.",
			},
		}
	}
	type result struct {
		events []contracts.TurnEvent
		err    error
	}
	first := make(chan result, 1)
	second := make(chan result, 1)
	go func() {
		stream, err := svcA.SubmitAction(context.Background(), req("race-a"))
		if err != nil {
			first <- result{err: err}
			return
		}
		first <- result{events: collectTurnEvents(stream)}
	}()
	select {
	case <-provider.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first provider call did not start")
	}
	go func() {
		stream, err := svcB.SubmitAction(context.Background(), req("race-b"))
		if err != nil {
			second <- result{err: err}
			return
		}
		second <- result{events: collectTurnEvents(stream)}
	}()
	close(provider.release)

	var results []result
	for i := 0; i < 2; i++ {
		select {
		case res := <-first:
			results = append(results, res)
			first = nil
		case res := <-second:
			results = append(results, res)
			second = nil
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent submissions did not finish")
		}
	}
	successes := 0
	stales := 0
	for _, res := range results {
		if res.err == nil {
			successes++
			continue
		}
		if strings.Contains(res.err.Error(), "stale client_turn") {
			stales++
			continue
		}
		t.Fatalf("unexpected concurrent submit error: %v", res.err)
	}
	if successes != 1 || stales != 1 {
		t.Fatalf("successes=%d stales=%d, want 1/1; results=%#v", successes, stales, results)
	}
	if provider.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.callCount())
	}
	world, err := db.GetWorldState("story-race")
	if err != nil {
		t.Fatalf("GetWorldState: %v", err)
	}
	if world.CurrentTurn != 1 {
		t.Fatalf("world current turn = %d, want 1", world.CurrentTurn)
	}
}

func TestInProcessTurnServiceConcurrentSameIdempotencyReplaysCommittedTurn(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-same-idem", 0)

	provider := &blockingTurnProvider{entered: make(chan struct{}), release: make(chan struct{})}
	router, err := ai.NewRouter([]ai.Provider{provider})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	cfg := config.Default()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.RAG.Enabled = false
	cfg.AI.ASCIIArt.Enabled = false

	svcA := NewInProcessTurnService(cfg, db, router)
	svcB := NewInProcessTurnService(cfg, db, router)
	snapshot, err := svcA.Snapshot(context.Background(), "story-same-idem")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	req := contracts.SubmitActionRequest{
		StoryID:        "story-same-idem",
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		ClientRevision: snapshot.Revision,
		IdempotencyKey: "same-key",
		Action: contracts.PlayerAction{
			Kind: contracts.ActionKindFreeText,
			Text: "I test the retry path.",
		},
	}

	type result struct {
		events []contracts.TurnEvent
		err    error
	}
	first := make(chan result, 1)
	second := make(chan result, 1)
	go func() {
		stream, err := svcA.SubmitAction(context.Background(), req)
		if err != nil {
			first <- result{err: err}
			return
		}
		first <- result{events: collectTurnEvents(stream)}
	}()
	select {
	case <-provider.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first provider call did not start")
	}
	go func() {
		stream, err := svcB.SubmitAction(context.Background(), req)
		if err != nil {
			second <- result{err: err}
			return
		}
		second <- result{events: collectTurnEvents(stream)}
	}()
	close(provider.release)

	var got []result
	for i := 0; i < 2; i++ {
		select {
		case res := <-first:
			got = append(got, res)
			first = nil
		case res := <-second:
			got = append(got, res)
			second = nil
		case <-time.After(5 * time.Second):
			t.Fatal("same-key submissions did not finish")
		}
	}
	for _, res := range got {
		if res.err != nil {
			t.Fatalf("same-key submit error: %v", res.err)
		}
		if len(res.events) == 0 {
			t.Fatalf("same-key submit returned no events")
		}
	}
	if provider.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.callCount())
	}
	if len(got[0].events) != len(got[1].events) {
		t.Fatalf("event counts = %d/%d, want equal", len(got[0].events), len(got[1].events))
	}
	for i := range got[0].events {
		if got[0].events[i].ID != got[1].events[i].ID || got[0].events[i].Type != got[1].events[i].Type {
			t.Fatalf("event[%d] mismatch: %+v vs %+v", i, got[0].events[i], got[1].events[i])
		}
	}

	restarted := NewInProcessTurnService(cfg, db, router)
	replay, err := restarted.SubmitAction(context.Background(), req)
	if err != nil {
		t.Fatalf("persistent same-key replay: %v", err)
	}
	if events := collectTurnEvents(replay); len(events) != len(got[0].events) {
		t.Fatalf("persistent replay event count = %d, want %d", len(events), len(got[0].events))
	}
	if provider.callCount() != 1 {
		t.Fatalf("provider calls after persistent replay = %d, want 1", provider.callCount())
	}
}

func TestInProcessTurnServiceMetaCommandsDoNotAdvanceTurnForUsagePrompts(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-meta", 7)

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

	snapshot, err := svc.Snapshot(context.Background(), "story-meta")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	tests := []struct {
		kind contracts.BrowserMetaKind
		want string
	}{
		{kind: contracts.BrowserMetaKindBTW, want: "Ask anything"},
		{kind: contracts.BrowserMetaKindGuide, want: "Tell me what you want"},
		{kind: contracts.BrowserMetaKindNarrator, want: "I'm listening"},
	}
	for _, tc := range tests {
		resp, err := svc.SubmitMeta(context.Background(), contracts.BrowserMetaRequest{
			StoryID:        "story-meta",
			SessionID:      snapshot.SessionID,
			ClientTurn:     snapshot.Turn,
			ClientRevision: snapshot.Revision,
			Kind:           tc.kind,
		})
		if err != nil {
			t.Fatalf("SubmitMeta(%s): %v", tc.kind, err)
		}
		if resp.Kind != tc.kind || resp.Message == "" {
			t.Fatalf("SubmitMeta(%s) response = %+v", tc.kind, resp)
		}
		if got := resp.Message; len(got) < len(tc.want) || got[:len(tc.want)] != tc.want {
			t.Fatalf("SubmitMeta(%s) message = %q, want prefix %q", tc.kind, got, tc.want)
		}
	}
	world, err := db.GetWorldState("story-meta")
	if err != nil {
		t.Fatalf("GetWorldState: %v", err)
	}
	if world.CurrentTurn != 7 {
		t.Fatalf("world turn = %d, want 7", world.CurrentTurn)
	}
	if provider.callCount() != 0 {
		t.Fatalf("provider calls = %d, want 0 for usage prompts", provider.callCount())
	}
}

func TestInProcessTurnServiceCreateAndLoadSave(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-save", 3)

	cfg := config.Default()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.RAG.Enabled = false
	svc := NewInProcessTurnService(cfg, db, nil)

	snapshot, err := svc.Snapshot(context.Background(), "story-save")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	saveResp, err := svc.CreateSave(context.Background(), contracts.BrowserSaveRequest{
		StoryID:        "story-save",
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		ClientRevision: snapshot.Revision,
		Name:           "Browser Slot",
		Kind:           "manual",
	})
	if err != nil {
		t.Fatalf("CreateSave: %v", err)
	}
	if saveResp.Save.Name != "Browser Slot" || saveResp.Save.Turn != 3 {
		t.Fatalf("save response = %+v, want Browser Slot at turn 3", saveResp.Save)
	}

	world, err := db.GetWorldState("story-save")
	if err != nil {
		t.Fatalf("GetWorldState before mutate: %v", err)
	}
	world.CurrentLocation = "Changed Road"
	world.CurrentTurn = 4
	if err := db.UpdateWorldState(world); err != nil {
		t.Fatalf("UpdateWorldState: %v", err)
	}
	latest, err := svc.Snapshot(context.Background(), "story-save")
	if err != nil {
		t.Fatalf("Snapshot after mutate: %v", err)
	}
	loadResp, err := svc.LoadSave(context.Background(), contracts.BrowserLoadRequest{
		StoryID:        "story-save",
		SessionID:      latest.SessionID,
		ClientTurn:     latest.Turn,
		ClientRevision: latest.Revision,
		SaveID:         saveResp.Save.ID,
	})
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if loadResp.Save.ID != saveResp.Save.ID {
		t.Fatalf("loaded save id = %q, want %q", loadResp.Save.ID, saveResp.Save.ID)
	}
	restored, err := db.GetWorldState("story-save")
	if err != nil {
		t.Fatalf("GetWorldState after load: %v", err)
	}
	if restored.CurrentTurn != 3 || restored.CurrentLocation != "Harbor" {
		t.Fatalf("restored world = turn %d location %q, want turn 3 Harbor", restored.CurrentTurn, restored.CurrentLocation)
	}
}

func TestInProcessTurnServiceDeleteSave(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-delete-save", 5)

	cfg := config.Default()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.RAG.Enabled = false
	svc := NewInProcessTurnService(cfg, db, nil)

	snapshot, err := svc.Snapshot(context.Background(), "story-delete-save")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	saveResp, err := svc.CreateSave(context.Background(), contracts.BrowserSaveRequest{
		StoryID:        "story-delete-save",
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		ClientRevision: snapshot.Revision,
		Name:           "Delete Me",
		Kind:           "manual",
	})
	if err != nil {
		t.Fatalf("CreateSave: %v", err)
	}

	latest, err := svc.Snapshot(context.Background(), "story-delete-save")
	if err != nil {
		t.Fatalf("Snapshot after create save: %v", err)
	}
	deleteResp, err := svc.DeleteSave(context.Background(), contracts.BrowserDeleteSaveRequest{
		StoryID:        "story-delete-save",
		SessionID:      latest.SessionID,
		ClientTurn:     latest.Turn,
		ClientRevision: latest.Revision,
		SaveID:         saveResp.Save.ID,
	})
	if err != nil {
		t.Fatalf("DeleteSave: %v", err)
	}
	if deleteResp.Save.ID != saveResp.Save.ID || deleteResp.Save.Name != "Delete Me" {
		t.Fatalf("delete response = %+v, want deleted save view", deleteResp.Save)
	}
	if _, err := db.GetSave(saveResp.Save.ID); err == nil {
		t.Fatal("GetSave after DeleteSave succeeded, want error")
	}
	saves, err := db.ListSaves("story-delete-save")
	if err != nil {
		t.Fatalf("ListSaves: %v", err)
	}
	if len(saves) != 0 {
		t.Fatalf("saves after delete = %d, want 0", len(saves))
	}
}

func TestInProcessTurnServiceRejectsStaleClientRevisionAtSameTurn(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-stale-revision", 2)

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
	snapshot, err := svc.Snapshot(context.Background(), "story-stale-revision")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := db.BumpStoryRevision("story-stale-revision"); err != nil {
		t.Fatalf("BumpStoryRevision: %v", err)
	}

	_, err = svc.SubmitAction(context.Background(), contracts.SubmitActionRequest{
		StoryID:        "story-stale-revision",
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		ClientRevision: snapshot.Revision,
		IdempotencyKey: "stale-revision-key",
		Action: contracts.PlayerAction{
			Kind: contracts.ActionKindFreeText,
			Text: "I act on the old branch.",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "stale client_revision") {
		t.Fatalf("SubmitAction err = %v, want stale client_revision", err)
	}
	if provider.callCount() != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.callCount())
	}
}

func TestInProcessTurnServiceRejectsSameIdempotencyKeyDifferentAction(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-idem-conflict", 0)

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
	snapshot, err := svc.Snapshot(context.Background(), "story-idem-conflict")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	first := contracts.SubmitActionRequest{
		StoryID:        "story-idem-conflict",
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		ClientRevision: snapshot.Revision,
		IdempotencyKey: "same-key-different-action",
		Action: contracts.PlayerAction{
			Kind: contracts.ActionKindFreeText,
			Text: "Open the door.",
		},
	}
	stream, err := svc.SubmitAction(context.Background(), first)
	if err != nil {
		t.Fatalf("SubmitAction first: %v", err)
	}
	_ = collectTurnEvents(stream)

	second := first
	second.Action.Text = "Attack the guard."
	_, err = svc.SubmitAction(context.Background(), second)
	if err == nil || !strings.Contains(err.Error(), "turn idempotency key belongs to a different request") {
		t.Fatalf("SubmitAction conflict err = %v, want idempotency conflict", err)
	}
	if provider.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.callCount())
	}
}

func TestInProcessTurnServiceLoadClearsFutureIdempotencyReplay(t *testing.T) {
	root := t.TempDir()
	db := newTurnServiceTestDB(t, root)
	createTurnServiceStory(t, db, "story-load-clears-idem", 0)

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
	initial, err := svc.Snapshot(context.Background(), "story-load-clears-idem")
	if err != nil {
		t.Fatalf("initial Snapshot: %v", err)
	}
	saveResp, err := svc.CreateSave(context.Background(), contracts.BrowserSaveRequest{
		StoryID:        "story-load-clears-idem",
		SessionID:      initial.SessionID,
		ClientTurn:     initial.Turn,
		ClientRevision: initial.Revision,
		Name:           "Before Future Turn",
		Kind:           "manual",
	})
	if err != nil {
		t.Fatalf("CreateSave: %v", err)
	}
	beforeTurn, err := svc.Snapshot(context.Background(), "story-load-clears-idem")
	if err != nil {
		t.Fatalf("Snapshot before turn: %v", err)
	}
	req := contracts.SubmitActionRequest{
		StoryID:        "story-load-clears-idem",
		SessionID:      beforeTurn.SessionID,
		ClientTurn:     beforeTurn.Turn,
		ClientRevision: beforeTurn.Revision,
		IdempotencyKey: "future-key",
		Action: contracts.PlayerAction{
			Kind: contracts.ActionKindFreeText,
			Text: "Create a future branch.",
		},
	}
	stream, err := svc.SubmitAction(context.Background(), req)
	if err != nil {
		t.Fatalf("SubmitAction future branch: %v", err)
	}
	_ = collectTurnEvents(stream)

	afterTurn, err := svc.Snapshot(context.Background(), "story-load-clears-idem")
	if err != nil {
		t.Fatalf("Snapshot after turn: %v", err)
	}
	if _, err := svc.LoadSave(context.Background(), contracts.BrowserLoadRequest{
		StoryID:        "story-load-clears-idem",
		SessionID:      afterTurn.SessionID,
		ClientTurn:     afterTurn.Turn,
		ClientRevision: afterTurn.Revision,
		SaveID:         saveResp.Save.ID,
	}); err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	_, err = svc.SubmitAction(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "stale client_revision") {
		t.Fatalf("SubmitAction after load err = %v, want stale client_revision without replay", err)
	}
	if provider.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.callCount())
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
