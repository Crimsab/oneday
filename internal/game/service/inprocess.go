package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/providers"
	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
)

// InProcessTurnService runs browser turns through the same Narrator pipeline as
// the terminal UI, while serializing mutations per story.
type InProcessTurnService struct {
	cfg    config.Config
	db     *storage.DB
	router *ai.Router

	mu          sync.Mutex
	storyLocks  map[string]*sync.Mutex
	idempotency map[string][]contracts.TurnEvent
}

func NewInProcessTurnService(cfg config.Config, db *storage.DB, router *ai.Router) *InProcessTurnService {
	return &InProcessTurnService{
		cfg:         cfg,
		db:          db,
		router:      router,
		storyLocks:  make(map[string]*sync.Mutex),
		idempotency: make(map[string][]contracts.TurnEvent),
	}
}

func (s *InProcessTurnService) Snapshot(_ context.Context, storyID string) (*contracts.GameSnapshot, error) {
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return nil, errors.New("story_id is required")
	}

	world, err := s.db.GetWorldState(storyID)
	if err != nil {
		return nil, err
	}
	story, err := s.db.GetStory(storyID)
	if err != nil {
		return nil, err
	}
	session, err := s.db.GetActiveSession(storyID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		gameSession, err := engine.NewGameSession(s.db, storyID, s.cfg.DataDir)
		if err != nil {
			return nil, fmt.Errorf("creating browser session: %w", err)
		}
		defer func() { _ = gameSession.CloseMirrors() }()
		session = &storage.Session{ID: gameSession.SessionID(), StoryID: storyID}
	}

	snapshot := &contracts.GameSnapshot{
		StoryID:  storyID,
		Turn:     world.CurrentTurn,
		Revision: story.Revision,
		Location: world.CurrentLocation,
	}
	if session != nil {
		snapshot.SessionID = session.ID
	}
	if choices := latestChoices(s.db, storyID); len(choices) > 0 {
		snapshot.Choices = choices
	}
	return snapshot, nil
}

func (s *InProcessTurnService) SubmitAction(ctx context.Context, req contracts.SubmitActionRequest) (<-chan contracts.TurnEvent, error) {
	if s.router == nil {
		return nil, errors.New("AI router is not configured")
	}
	if s.db == nil {
		return nil, errors.New("database is not configured")
	}

	requestHash, err := requestFingerprint(req)
	if err != nil {
		return nil, err
	}

	if events, ok, err := s.cachedEvents(req, requestHash); err != nil {
		return nil, err
	} else if ok {
		return eventsChannel(events), nil
	}

	lock := s.storyLock(req.StoryID)
	lock.Lock()
	defer lock.Unlock()

	lease, err := s.acquireStoryMutationLease(ctx, req.StoryID, "turn")
	if err != nil {
		return nil, err
	}
	defer func() { _ = lease.Release() }()

	if events, ok, err := s.cachedEvents(req, requestHash); err != nil {
		return nil, err
	} else if ok {
		return eventsChannel(events), nil
	}

	snapshot, err := s.Snapshot(ctx, req.StoryID)
	if err != nil {
		return nil, err
	}
	if err := req.Validate(snapshot.Turn, snapshot.Revision); err != nil {
		return nil, err
	}
	if snapshot.SessionID != "" && req.SessionID != snapshot.SessionID {
		return nil, fmt.Errorf("stale session_id %q, active session is %q", req.SessionID, snapshot.SessionID)
	}

	claim, events, ok, err := s.claimTurn(req, requestHash)
	if err != nil {
		return nil, err
	}
	if ok {
		return eventsChannel(events), nil
	}

	events, err = s.runTurn(ctx, req, snapshot, lease.Lock(), claim)
	if err != nil {
		if claim != nil {
			_ = claim.Fail(err)
		}
		return nil, err
	}
	s.cacheEvents(req, events)
	return eventsChannel(events), nil
}

// SubmitActionStream mirrors SubmitAction but emits turn.started and
// narrative.delta events while the provider is still streaming. The final
// canonical events are still cached through the same idempotency path.
func (s *InProcessTurnService) SubmitActionStream(ctx context.Context, req contracts.SubmitActionRequest) (<-chan contracts.TurnEvent, error) {
	if s.router == nil {
		return nil, errors.New("AI router is not configured")
	}
	if s.db == nil {
		return nil, errors.New("database is not configured")
	}

	requestHash, err := requestFingerprint(req)
	if err != nil {
		return nil, err
	}

	if events, ok, err := s.cachedEvents(req, requestHash); err != nil {
		return nil, err
	} else if ok {
		return eventsChannel(events), nil
	}

	out := make(chan contracts.TurnEvent, 32)
	go func() {
		defer close(out)
		if err := s.submitActionStreamLocked(ctx, req, requestHash, out); err != nil {
			out <- errorTurnEvent(req, err)
		}
	}()
	return out, nil
}

func (s *InProcessTurnService) submitActionStreamLocked(ctx context.Context, req contracts.SubmitActionRequest, requestHash string, out chan<- contracts.TurnEvent) error {
	lock := s.storyLock(req.StoryID)
	lock.Lock()
	defer lock.Unlock()

	lease, err := s.acquireStoryMutationLease(ctx, req.StoryID, "turn")
	if err != nil {
		return err
	}
	defer func() { _ = lease.Release() }()

	if events, ok, err := s.cachedEvents(req, requestHash); err != nil {
		return err
	} else if ok {
		sendTurnEvents(out, events)
		return nil
	}

	snapshot, err := s.Snapshot(ctx, req.StoryID)
	if err != nil {
		return err
	}
	if err := req.Validate(snapshot.Turn, snapshot.Revision); err != nil {
		return err
	}
	if snapshot.SessionID != "" && req.SessionID != snapshot.SessionID {
		return fmt.Errorf("stale session_id %q, active session is %q", req.SessionID, snapshot.SessionID)
	}

	claim, events, ok, err := s.claimTurn(req, requestHash)
	if err != nil {
		return err
	}
	if ok {
		sendTurnEvents(out, events)
		return nil
	}

	events, err = s.runTurnStream(ctx, req, snapshot, lease.Lock(), claim, out)
	if err != nil {
		if claim != nil {
			_ = claim.Fail(err)
		}
		return err
	}
	s.cacheEvents(req, events)
	return nil
}

func (s *InProcessTurnService) SubmitMeta(ctx context.Context, req contracts.BrowserMetaRequest) (*contracts.BrowserMetaResponse, error) {
	if s.router == nil {
		return nil, errors.New("AI router is not configured")
	}
	if s.db == nil {
		return nil, errors.New("database is not configured")
	}

	lock := s.storyLock(req.StoryID)
	lock.Lock()
	defer lock.Unlock()

	lease, err := s.acquireStoryMutationLease(ctx, req.StoryID, "meta")
	if err != nil {
		return nil, err
	}
	defer func() { _ = lease.Release() }()

	snapshot, err := s.Snapshot(ctx, req.StoryID)
	if err != nil {
		return nil, err
	}
	if err := req.Validate(snapshot.Turn, snapshot.Revision); err != nil {
		return nil, err
	}
	if snapshot.SessionID != "" && req.SessionID != snapshot.SessionID {
		return nil, fmt.Errorf("stale session_id %q, active session is %q", req.SessionID, snapshot.SessionID)
	}

	narrator, session, err := s.newNarrator(req.StoryID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.CloseMirrors() }()

	switch req.Kind {
	case contracts.BrowserMetaKindBTW:
		message, err := narrator.ExecuteAsideQuestion(ctx, req.Text)
		if err != nil {
			return nil, err
		}
		if err := lease.Renew(); err != nil {
			return nil, err
		}
		return &contracts.BrowserMetaResponse{Kind: req.Kind, Title: "By The Way", Message: message}, nil
	case contracts.BrowserMetaKindGuide:
		resp, err := narrator.ExecuteGuideCommand(ctx, req.Text)
		if err != nil {
			return nil, err
		}
		if err := lease.Renew(); err != nil {
			return nil, err
		}
		return &contracts.BrowserMetaResponse{Kind: req.Kind, Title: "Guide", Message: resp.Message}, nil
	case contracts.BrowserMetaKindNarrator:
		resp, err := narrator.ExecuteNarratorCommand(ctx, req.Text)
		if err != nil {
			return nil, err
		}
		if err := lease.Renew(); err != nil {
			return nil, err
		}
		return &contracts.BrowserMetaResponse{Kind: req.Kind, Title: "Narrator Control", Message: resp.Message}, nil
	default:
		return nil, fmt.Errorf("unsupported browser meta kind %q", req.Kind)
	}
}

func (s *InProcessTurnService) CreateSave(ctx context.Context, req contracts.BrowserSaveRequest) (*contracts.BrowserSaveResponse, error) {
	if s.db == nil {
		return nil, errors.New("database is not configured")
	}

	lock := s.storyLock(req.StoryID)
	lock.Lock()
	defer lock.Unlock()

	lease, err := s.acquireStoryMutationLease(ctx, req.StoryID, "save-create")
	if err != nil {
		return nil, err
	}
	defer func() { _ = lease.Release() }()

	snapshot, err := s.Snapshot(ctx, req.StoryID)
	if err != nil {
		return nil, err
	}
	if err := req.Validate(snapshot.Turn, snapshot.Revision); err != nil {
		return nil, err
	}
	if snapshot.SessionID != "" && req.SessionID != snapshot.SessionID {
		return nil, fmt.Errorf("stale session_id %q, active session is %q", req.SessionID, snapshot.SessionID)
	}

	story, err := s.db.GetStory(req.StoryID)
	if err != nil {
		return nil, err
	}
	character, err := s.db.GetCharacterByStory(req.StoryID)
	if err != nil {
		return nil, err
	}
	world, err := s.db.GetWorldState(req.StoryID)
	if err != nil {
		return nil, err
	}
	session, err := engine.NewGameSession(s.db, req.StoryID, s.cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("creating game session: %w", err)
	}
	defer func() { _ = session.CloseMirrors() }()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Manual Save"
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = "manual"
	}
	if err := lease.Renew(); err != nil {
		return nil, err
	}
	save, err := engine.SaveGameWithMetadata(
		s.db,
		s.cfg.DataDir,
		story,
		character,
		world,
		session.SessionID(),
		name,
		&storage.SaveMetadata{Kind: kind},
	)
	if err != nil {
		return nil, err
	}
	return &contracts.BrowserSaveResponse{Save: saveView(save)}, nil
}

func (s *InProcessTurnService) LoadSave(ctx context.Context, req contracts.BrowserLoadRequest) (*contracts.BrowserLoadResponse, error) {
	if s.db == nil {
		return nil, errors.New("database is not configured")
	}

	lock := s.storyLock(req.StoryID)
	lock.Lock()
	defer lock.Unlock()

	lease, err := s.acquireStoryMutationLease(ctx, req.StoryID, "save-load")
	if err != nil {
		return nil, err
	}
	defer func() { _ = lease.Release() }()

	snapshot, err := s.Snapshot(ctx, req.StoryID)
	if err != nil {
		return nil, err
	}
	if err := req.Validate(snapshot.Turn, snapshot.Revision); err != nil {
		return nil, err
	}
	if snapshot.SessionID != "" && req.SessionID != snapshot.SessionID {
		return nil, fmt.Errorf("stale session_id %q, active session is %q", req.SessionID, snapshot.SessionID)
	}
	save, err := s.db.GetSave(req.SaveID)
	if err != nil {
		return nil, err
	}
	if save.StoryID != req.StoryID {
		return nil, fmt.Errorf("save %s belongs to story %s, not %s", req.SaveID, save.StoryID, req.StoryID)
	}

	if err := lease.Renew(); err != nil {
		return nil, err
	}
	result, err := engine.LoadGame(s.db, s.cfg.DataDir, req.SaveID)
	if err != nil {
		return nil, err
	}
	s.clearCachedEvents(req.StoryID)
	return &contracts.BrowserLoadResponse{Save: saveView(result.Save), Legacy: result.Legacy}, nil
}

func (s *InProcessTurnService) DeleteSave(ctx context.Context, req contracts.BrowserDeleteSaveRequest) (*contracts.BrowserDeleteSaveResponse, error) {
	if s.db == nil {
		return nil, errors.New("database is not configured")
	}

	lock := s.storyLock(req.StoryID)
	lock.Lock()
	defer lock.Unlock()

	lease, err := s.acquireStoryMutationLease(ctx, req.StoryID, "save-delete")
	if err != nil {
		return nil, err
	}
	defer func() { _ = lease.Release() }()

	snapshot, err := s.Snapshot(ctx, req.StoryID)
	if err != nil {
		return nil, err
	}
	if err := req.Validate(snapshot.Turn, snapshot.Revision); err != nil {
		return nil, err
	}
	if snapshot.SessionID != "" && req.SessionID != snapshot.SessionID {
		return nil, fmt.Errorf("stale session_id %q, active session is %q", req.SessionID, snapshot.SessionID)
	}
	save, err := s.db.GetSave(req.SaveID)
	if err != nil {
		return nil, err
	}
	if save.StoryID != req.StoryID {
		return nil, fmt.Errorf("save %s belongs to story %s, not %s", req.SaveID, save.StoryID, req.StoryID)
	}
	view := saveView(save)
	if err := lease.Renew(); err != nil {
		return nil, err
	}
	if err := engine.DeleteSave(s.db, s.cfg.DataDir, req.SaveID); err != nil {
		return nil, err
	}
	return &contracts.BrowserDeleteSaveResponse{Save: view}, nil
}

func (s *InProcessTurnService) runTurn(ctx context.Context, req contracts.SubmitActionRequest, snapshot *contracts.GameSnapshot, storyLock *storage.StoryTurnLock, claim *storage.TurnIdempotencyClaim) ([]contracts.TurnEvent, error) {
	narrator, session, err := s.newNarrator(req.StoryID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.CloseMirrors() }()
	sessionID := session.SessionID()

	world := narrator.World()
	var committedEvents []contracts.TurnEvent
	var hook engine.TurnCommitHook
	if claim != nil {
		hook = func(tx *sql.Tx, result engine.TurnCommitResult) error {
			events, err := buildTurnEvents(req, snapshot, sessionID, result.Narrative, result.World, result.Revision)
			if err != nil {
				return err
			}
			data, err := json.Marshal(events)
			if err != nil {
				return fmt.Errorf("encoding idempotency events: %w", err)
			}
			if err := claim.CommitTx(tx, string(data)); err != nil {
				return err
			}
			committedEvents = cloneEvents(events)
			return nil
		}
	}

	resolvedActionText, err := actionText(req.Action, snapshot)
	if err != nil {
		return nil, err
	}
	resp, err := narrator.SendActionWithLeaseAndCommitHook(ctx, resolvedActionText, storyLock, hook)
	if err != nil {
		return nil, err
	}
	if len(committedEvents) > 0 {
		return cloneEvents(committedEvents), nil
	}
	return buildTurnEvents(req, snapshot, sessionID, resp, world, narrator.Story().Revision)
}

func (s *InProcessTurnService) runTurnStream(ctx context.Context, req contracts.SubmitActionRequest, snapshot *contracts.GameSnapshot, storyLock *storage.StoryTurnLock, claim *storage.TurnIdempotencyClaim, out chan<- contracts.TurnEvent) ([]contracts.TurnEvent, error) {
	narrator, session, err := s.newNarrator(req.StoryID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.CloseMirrors() }()
	sessionID := session.SessionID()

	world := narrator.World()
	seq := 1
	emit := func(eventType contracts.TurnEventType, payload any) (contracts.TurnEvent, error) {
		event, err := contracts.NewTurnEvent(
			fmt.Sprintf("%s:live:%d", req.IdempotencyKey, seq),
			req.StoryID,
			sessionID,
			snapshot.Turn,
			eventType,
			payload,
		)
		if err != nil {
			return contracts.TurnEvent{}, err
		}
		seq++
		out <- event
		return event, nil
	}

	if _, err := emit(contracts.EventTurnStarted, map[string]any{
		"client_turn":     req.ClientTurn,
		"client_revision": req.ClientRevision,
		"action":          req.Action,
	}); err != nil {
		return nil, err
	}

	var committedEvents []contracts.TurnEvent
	var hook engine.TurnCommitHook
	if claim != nil {
		hook = func(tx *sql.Tx, result engine.TurnCommitResult) error {
			events, err := buildTurnEvents(req, snapshot, sessionID, result.Narrative, result.World, result.Revision)
			if err != nil {
				return err
			}
			data, err := json.Marshal(events)
			if err != nil {
				return fmt.Errorf("encoding idempotency events: %w", err)
			}
			if err := claim.CommitTx(tx, string(data)); err != nil {
				return err
			}
			committedEvents = cloneEvents(events)
			return nil
		}
	}

	resolvedActionText, err := actionText(req.Action, snapshot)
	if err != nil {
		return nil, err
	}
	stream, err := narrator.StreamActionWithLeaseAndCommitHook(ctx, resolvedActionText, storyLock, hook)
	if err != nil {
		return nil, err
	}
	var resp *engine.NarrativeResponse
	for chunk := range stream {
		if chunk.Err != nil {
			return nil, chunk.Err
		}
		if chunk.Delta != "" {
			if _, err := emit(contracts.EventNarrativeDelta, map[string]any{
				"text": chunk.Delta,
			}); err != nil {
				return nil, err
			}
		}
		if chunk.Done {
			resp = chunk.Response
		}
	}
	if resp == nil {
		return nil, errors.New("streaming turn finished without a narrative response")
	}

	finalEvents := committedEvents
	if len(finalEvents) == 0 {
		finalEvents, err = buildTurnEvents(req, snapshot, sessionID, resp, world, narrator.Story().Revision)
		if err != nil {
			return nil, err
		}
	}
	for _, event := range finalEvents {
		out <- event
	}
	return cloneEvents(finalEvents), nil
}

func buildTurnEvents(req contracts.SubmitActionRequest, snapshot *contracts.GameSnapshot, sessionID string, resp *engine.NarrativeResponse, world *storage.WorldState, revision int64) ([]contracts.TurnEvent, error) {
	events := make([]contracts.TurnEvent, 0, 6)
	appendEvent := func(eventType contracts.TurnEventType, payload any) error {
		event, err := contracts.NewTurnEvent(
			fmt.Sprintf("%s:%d", req.IdempotencyKey, len(events)+1),
			req.StoryID,
			sessionID,
			snapshot.Turn,
			eventType,
			payload,
		)
		if err != nil {
			return err
		}
		events = append(events, event)
		return nil
	}

	if resp == nil {
		return nil, errors.New("missing narrative response")
	}
	if world == nil {
		return nil, errors.New("missing world state")
	}
	if err := appendEvent(contracts.EventTurnStarted, map[string]any{
		"client_turn":     req.ClientTurn,
		"client_revision": req.ClientRevision,
		"action":          req.Action,
	}); err != nil {
		return nil, err
	}

	if err := appendEvent(contracts.EventNarrativeFinal, map[string]any{
		"narrative": resp.Narrative,
		"mood":      resp.Mood,
		"location":  resp.Location,
		"scene":     resp.SceneType,
	}); err != nil {
		return nil, err
	}
	if len(resp.AppliedStateChanges) > 0 {
		if err := appendEvent(contracts.EventStateDelta, map[string]any{
			"changes": stateDeltaViews(resp.AppliedStateChanges),
		}); err != nil {
			return nil, err
		}
	}
	if resp.VisualCue != nil && req.Capabilities.Images {
		if err := appendEvent(contracts.EventAssetQueued, map[string]any{
			"visual_cue": resp.VisualCue,
		}); err != nil {
			return nil, err
		}
	}
	if len(resp.Choices) > 0 {
		if err := appendEvent(contracts.EventChoicesUpdated, map[string]any{
			"choices": choiceViews(resp.Choices),
		}); err != nil {
			return nil, err
		}
	}
	if err := appendEvent(contracts.EventTurnCommitted, map[string]any{
		"turn":       world.CurrentTurn,
		"session_id": sessionID,
		"snapshot": contracts.GameSnapshot{
			StoryID:   req.StoryID,
			SessionID: sessionID,
			Turn:      world.CurrentTurn,
			Revision:  revision,
			Location:  world.CurrentLocation,
			Choices:   choiceViews(resp.Choices),
		},
	}); err != nil {
		return nil, err
	}

	return events, nil
}

func (s *InProcessTurnService) newNarrator(storyID string) (*engine.Narrator, *engine.GameSession, error) {
	story, err := s.db.GetStory(storyID)
	if err != nil {
		return nil, nil, err
	}
	character, err := s.db.GetCharacterByStory(storyID)
	if err != nil {
		return nil, nil, err
	}
	world, err := s.db.GetWorldState(storyID)
	if err != nil {
		return nil, nil, err
	}
	session, err := engine.NewGameSession(s.db, storyID, s.cfg.DataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("creating game session: %w", err)
	}

	contextCfg := engine.DefaultContextConfig()
	contextCfg.RewardBudget = s.cfg.Game.RewardBudget
	narrator := engine.NewNarrator(
		s.router,
		s.db,
		story,
		character,
		world,
		session,
		contextCfg,
		s.cfg.AI.Generation,
		s.cfg.AI.ASCIIArt,
		s.cfg.DataDir,
		s.cfg.Game.AutosaveEvery,
	)
	narrator.SetRAG(s.buildRAG(storyID))
	return narrator, session, nil
}

func saveView(save *storage.SaveSnapshot) contracts.BrowserSaveView {
	if save == nil {
		return contracts.BrowserSaveView{}
	}
	metadata := json.RawMessage(save.MetadataJSON)
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	return contracts.BrowserSaveView{
		ID:        save.ID,
		Name:      save.Name,
		Turn:      save.Turn,
		Chapter:   save.Chapter,
		Location:  save.Location,
		SessionID: save.SessionID,
		Metadata:  metadata,
		CreatedAt: save.CreatedAt,
	}
}

func (s *InProcessTurnService) storyLock(storyID string) *sync.Mutex {
	storyID = strings.TrimSpace(storyID)
	s.mu.Lock()
	defer s.mu.Unlock()
	lock := s.storyLocks[storyID]
	if lock == nil {
		lock = &sync.Mutex{}
		s.storyLocks[storyID] = lock
	}
	return lock
}

func (s *InProcessTurnService) acquireStoryMutationLease(ctx context.Context, storyID, scope string) (*engine.StoryMutationLease, error) {
	return engine.AcquireStoryMutationLease(ctx, s.db, storyID, scope, "browser")
}

func (s *InProcessTurnService) cachedEvents(req contracts.SubmitActionRequest, requestHash string) ([]contracts.TurnEvent, bool, error) {
	key := idempotencyKey(req, requestHash)
	if key == "" {
		return nil, false, nil
	}
	s.mu.Lock()
	events, ok := s.idempotency[key]
	s.mu.Unlock()
	if ok {
		return cloneEvents(events), true, nil
	}

	if s.db == nil {
		return nil, false, nil
	}
	eventsJSON, found, err := s.db.GetTurnIdempotency(req.StoryID, req.IdempotencyKey, requestHash)
	if err != nil || !found {
		return nil, false, err
	}
	var stored []contracts.TurnEvent
	if err := json.Unmarshal([]byte(eventsJSON), &stored); err != nil {
		return nil, false, fmt.Errorf("decoding stored idempotency events: %w", err)
	}
	s.mu.Lock()
	s.idempotency[key] = cloneEvents(stored)
	s.mu.Unlock()
	return stored, true, nil
}

func (s *InProcessTurnService) claimTurn(req contracts.SubmitActionRequest, requestHash string) (*storage.TurnIdempotencyClaim, []contracts.TurnEvent, bool, error) {
	if s.db == nil {
		return nil, nil, false, nil
	}
	key := idempotencyKey(req, requestHash)
	if key == "" {
		return nil, nil, false, nil
	}
	result, err := s.db.ClaimTurnIdempotency(
		req.StoryID,
		req.IdempotencyKey,
		requestHash,
		fmt.Sprintf("browser:turn:%d", time.Now().UnixNano()),
		engine.StoryMutationLockTTL,
	)
	if err != nil {
		return nil, nil, false, err
	}
	if result == nil {
		return nil, nil, false, nil
	}
	if result.Committed {
		var stored []contracts.TurnEvent
		if err := json.Unmarshal([]byte(result.EventsJSON), &stored); err != nil {
			return nil, nil, false, fmt.Errorf("decoding committed idempotency events: %w", err)
		}
		s.cacheEvents(req, stored)
		return nil, stored, true, nil
	}
	return result.Claim, nil, false, nil
}

func (s *InProcessTurnService) cacheEvents(req contracts.SubmitActionRequest, events []contracts.TurnEvent) {
	requestHash, err := requestFingerprint(req)
	if err != nil {
		return
	}
	key := idempotencyKey(req, requestHash)
	if key == "" {
		return
	}
	s.mu.Lock()
	s.idempotency[key] = cloneEvents(events)
	s.mu.Unlock()
}

func (s *InProcessTurnService) clearCachedEvents(storyID string) {
	storyID = strings.TrimSpace(storyID)
	if storyID == "" {
		return
	}
	prefix := storyID + ":"
	s.mu.Lock()
	for key := range s.idempotency {
		if strings.HasPrefix(key, prefix) {
			delete(s.idempotency, key)
		}
	}
	s.mu.Unlock()
}

func (s *InProcessTurnService) buildRAG(storyID string) *rag.RAG {
	if !s.cfg.RAG.Enabled {
		return nil
	}
	spec, reason := aifactory.SelectEmbeddingProvider(s.cfg)
	if reason != "" {
		log.Printf("oneday: browser RAG disabled, reason: %s, story: %s", reason, storyID)
		return nil
	}

	timeout := time.Duration(s.cfg.AI.Generation.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	embProvider := buildEmbeddingProvider(spec, timeout)
	store := rag.NewVectorStore(s.db.Conn())
	if removed, err := store.PruneDimensionMismatches(context.Background(), storyID, spec.Dimensions); err != nil {
		log.Printf("oneday: browser RAG dimension migration failed, story: %s, err: %v", storyID, err)
	} else if removed > 0 {
		log.Printf("oneday: browser RAG removed %d stale embedding chunks for dimensions=%d, story: %s", removed, spec.Dimensions, storyID)
	}

	embedder := rag.NewEmbedder(embProvider, spec.Model, spec.Dimensions)
	summarizer := rag.NewSummarizer(embedder, store, s.router, storyID, s.cfg.RAG.SummarizeEvery)
	return rag.NewRAG(embedder, store, summarizer, storyID, s.cfg.RAG.TopK)
}

func buildEmbeddingProvider(spec aifactory.EmbeddingProviderSpec, timeout time.Duration) rag.EmbeddingProvider {
	switch spec.Kind {
	case "ollama":
		return providers.NewOllamaEmbedding(providers.OllamaEmbeddingConfig{
			BaseURL: spec.BaseURL,
			Model:   spec.Model,
			Timeout: timeout,
		})
	case "custom":
		return providers.NewLocalHTTPEmbedding(spec.BaseURL, spec.Model, timeout)
	default:
		return providers.NewOpenAICompat(providers.OpenAICompatConfig{
			Name:         spec.Name,
			BaseURL:      spec.BaseURL,
			APIKey:       spec.APIKey,
			DefaultModel: spec.Model,
			Timeout:      timeout,
		})
	}
}

func idempotencyKey(req contracts.SubmitActionRequest, requestHash string) string {
	if strings.TrimSpace(req.StoryID) == "" || strings.TrimSpace(req.IdempotencyKey) == "" || strings.TrimSpace(requestHash) == "" {
		return ""
	}
	return req.StoryID + ":" + req.IdempotencyKey + ":" + requestHash
}

func requestFingerprint(req contracts.SubmitActionRequest) (string, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return "", errors.New("idempotency_key is required")
	}
	if strings.TrimSpace(req.StoryID) == "" || strings.TrimSpace(req.SessionID) == "" {
		return "", errors.New("story_id and session_id are required")
	}
	payload := struct {
		Version        int                          `json:"version"`
		StoryID        string                       `json:"story_id"`
		SessionID      string                       `json:"session_id"`
		ClientTurn     int                          `json:"client_turn"`
		ClientRevision int64                        `json:"client_revision"`
		IdempotencyKey string                       `json:"idempotency_key"`
		Action         contracts.PlayerAction       `json:"action"`
		Capabilities   contracts.ClientCapabilities `json:"capabilities"`
	}{
		Version:        1,
		StoryID:        strings.TrimSpace(req.StoryID),
		SessionID:      strings.TrimSpace(req.SessionID),
		ClientTurn:     req.ClientTurn,
		ClientRevision: req.ClientRevision,
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		Action: contracts.PlayerAction{
			Kind:     req.Action.Kind,
			Text:     strings.TrimSpace(req.Action.Text),
			ChoiceID: req.Action.ChoiceID,
		},
		Capabilities: req.Capabilities,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encoding idempotency request fingerprint: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func eventsChannel(events []contracts.TurnEvent) <-chan contracts.TurnEvent {
	out := make(chan contracts.TurnEvent, len(events))
	for _, event := range events {
		out <- event
	}
	close(out)
	return out
}

func sendTurnEvents(out chan<- contracts.TurnEvent, events []contracts.TurnEvent) {
	for _, event := range events {
		out <- event
	}
}

func errorTurnEvent(req contracts.SubmitActionRequest, err error) contracts.TurnEvent {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" {
		key = "stream"
	}
	event, eventErr := contracts.NewTurnEvent(
		key+":error",
		req.StoryID,
		req.SessionID,
		req.ClientTurn,
		contracts.EventError,
		map[string]any{"message": err.Error()},
	)
	if eventErr != nil {
		return contracts.TurnEvent{
			ID:        key + ":error",
			StoryID:   req.StoryID,
			SessionID: req.SessionID,
			Turn:      req.ClientTurn,
			Type:      contracts.EventError,
			CreatedAt: time.Now().UTC(),
		}
	}
	return event
}

func reindexTurnEvents(events []contracts.TurnEvent, idempotencyKey string, startSeq int) []contracts.TurnEvent {
	out := cloneEvents(events)
	for i := range out {
		out[i].ID = fmt.Sprintf("%s:%d", idempotencyKey, startSeq+i)
	}
	return out
}

func cloneEvents(events []contracts.TurnEvent) []contracts.TurnEvent {
	if len(events) == 0 {
		return nil
	}
	cloned := make([]contracts.TurnEvent, len(events))
	copy(cloned, events)
	return cloned
}

func actionText(action contracts.PlayerAction, snapshot *contracts.GameSnapshot) (string, error) {
	text := strings.TrimSpace(action.Text)
	if action.ChoiceID > 0 {
		for _, choice := range snapshot.Choices {
			if choice.ID == action.ChoiceID {
				choiceText := strings.TrimSpace(choice.Text)
				if choiceText == "" {
					return fmt.Sprintf("[Choice %d]", action.ChoiceID), nil
				}
				return fmt.Sprintf("[Choice %d] %s", action.ChoiceID, choiceText), nil
			}
		}
		return "", fmt.Errorf("choice_id %d is not available for the current turn", action.ChoiceID)
	}
	if text != "" {
		return text, nil
	}
	return "", errors.New("action text or choice_id is required")
}

func choiceViews(choices []engine.Choice) []contracts.ChoiceView {
	out := make([]contracts.ChoiceView, 0, len(choices))
	for i, choice := range choices {
		id := choice.ID
		if id == 0 {
			id = i + 1
		}
		out = append(out, contracts.ChoiceView{
			ID:           id,
			Text:         choice.Text,
			Intent:       choice.Intent,
			Risk:         choice.Risk,
			Scope:        choice.Scope,
			Certainty:    choice.Certainty,
			RelatedStats: append([]string(nil), choice.RelatedStats...),
		})
	}
	return out
}

func stateDeltaViews(changes []engine.StateChange) []contracts.StateDelta {
	out := make([]contracts.StateDelta, 0, len(changes))
	for _, change := range changes {
		out = append(out, contracts.StateDelta{
			Target:      change.Target,
			Field:       change.Field,
			Description: change.Description,
		})
	}
	return out
}

func latestChoices(db *storage.DB, storyID string) []contracts.ChoiceView {
	messages, err := db.GetRecentMessagesByStory(storyID, 12)
	if err != nil {
		return nil
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "assistant" {
			continue
		}
		choices := choicesFromMetadata(messages[i].MetadataJSON)
		if len(choices) > 0 {
			return choices
		}
	}
	return nil
}

func choicesFromMetadata(metadataJSON string) []contracts.ChoiceView {
	var meta struct {
		Output struct {
			ChoicesData []engine.Choice `json:"choices_data"`
			Choices     []string        `json:"choices"`
		} `json:"output"`
		Choices []string `json:"choices"`
	}
	if err := decodeJSON(metadataJSON, &meta); err != nil {
		return nil
	}
	if len(meta.Output.ChoicesData) > 0 {
		return choiceViews(meta.Output.ChoicesData)
	}
	source := meta.Output.Choices
	if len(source) == 0 {
		source = meta.Choices
	}
	out := make([]contracts.ChoiceView, 0, len(source))
	for i, text := range source {
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, contracts.ChoiceView{ID: i + 1, Text: text})
	}
	return out
}

func decodeJSON(raw string, target any) error {
	if strings.TrimSpace(raw) == "" {
		return sql.ErrNoRows
	}
	return json.Unmarshal([]byte(raw), target)
}
