package service

import (
	"context"
	"database/sql"
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

	if events, ok, err := s.cachedEvents(req); err != nil {
		return nil, err
	} else if ok {
		return eventsChannel(events), nil
	}

	lock := s.storyLock(req.StoryID)
	lock.Lock()
	defer lock.Unlock()

	if events, ok, err := s.cachedEvents(req); err != nil {
		return nil, err
	} else if ok {
		return eventsChannel(events), nil
	}

	snapshot, err := s.Snapshot(ctx, req.StoryID)
	if err != nil {
		return nil, err
	}
	if err := req.Validate(snapshot.Turn); err != nil {
		return nil, err
	}
	if snapshot.SessionID != "" && req.SessionID != snapshot.SessionID {
		return nil, fmt.Errorf("stale session_id %q, active session is %q", req.SessionID, snapshot.SessionID)
	}

	events, err := s.runTurn(ctx, req, snapshot)
	if err != nil {
		return nil, err
	}
	s.storeEvents(req, events)
	return eventsChannel(events), nil
}

func (s *InProcessTurnService) runTurn(ctx context.Context, req contracts.SubmitActionRequest, snapshot *contracts.GameSnapshot) ([]contracts.TurnEvent, error) {
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
	narrator.SetRAG(s.buildRAG(req.StoryID))
	sessionID := session.SessionID()

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

	if err := appendEvent(contracts.EventTurnStarted, map[string]any{
		"client_turn": req.ClientTurn,
		"action":      req.Action,
	}); err != nil {
		return nil, err
	}

	resp, err := narrator.SendAction(ctx, actionText(req.Action))
	if err != nil {
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
			Location:  world.CurrentLocation,
			Choices:   choiceViews(resp.Choices),
		},
	}); err != nil {
		return nil, err
	}

	return events, nil
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

func (s *InProcessTurnService) cachedEvents(req contracts.SubmitActionRequest) ([]contracts.TurnEvent, bool, error) {
	key := idempotencyKey(req)
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
	eventsJSON, found, err := s.db.GetTurnIdempotency(req.StoryID, req.IdempotencyKey)
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

func (s *InProcessTurnService) storeEvents(req contracts.SubmitActionRequest, events []contracts.TurnEvent) {
	key := idempotencyKey(req)
	if key == "" {
		return
	}
	s.mu.Lock()
	s.idempotency[key] = cloneEvents(events)
	s.mu.Unlock()

	if s.db == nil {
		return
	}
	data, err := json.Marshal(events)
	if err != nil {
		log.Printf("oneday: browser idempotency encode failed for story %s: %v", req.StoryID, err)
		return
	}
	if err := s.db.SaveTurnIdempotency(req.StoryID, req.IdempotencyKey, string(data)); err != nil {
		log.Printf("oneday: browser idempotency persist failed for story %s: %v", req.StoryID, err)
	}
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

func idempotencyKey(req contracts.SubmitActionRequest) string {
	if strings.TrimSpace(req.StoryID) == "" || strings.TrimSpace(req.IdempotencyKey) == "" {
		return ""
	}
	return req.StoryID + ":" + req.IdempotencyKey
}

func eventsChannel(events []contracts.TurnEvent) <-chan contracts.TurnEvent {
	out := make(chan contracts.TurnEvent, len(events))
	for _, event := range events {
		out <- event
	}
	close(out)
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

func actionText(action contracts.PlayerAction) string {
	text := strings.TrimSpace(action.Text)
	if text != "" {
		return text
	}
	if action.ChoiceID > 0 {
		return fmt.Sprintf("[Choice %d]", action.ChoiceID)
	}
	return ""
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
