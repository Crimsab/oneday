package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
)

// AutosaveCompleteMsg is sent via Bubbletea when an autosave finishes.
type AutosaveCompleteMsg struct {
	Err error
}

// NarrativeStreamChunk is a high-level streamed narrative update for the TUI.
type NarrativeStreamChunk struct {
	Delta    string
	Done     bool
	Err      error
	Response *NarrativeResponse
}

// Narrator manages the gameplay AI conversation.
type Narrator struct {
	router                 *ai.Router
	db                     *storage.DB
	story                  *storage.Story
	character              *storage.Character
	world                  *storage.WorldState
	session                *GameSession
	contextCfg             ContextConfig
	genCfg                 config.GenerationConfig // AI generation parameters (temperature, max_tokens, timeout)
	lastModel              string
	lastLatency            int64
	lastTTFT               int64
	lastUsage              ai.Usage
	lastStreamed           bool
	asciiCfg               config.ASCIIArtConfig
	dataDir                string
	autosaveEvery          int
	rag                    *rag.RAG         // optional — nil means RAG is disabled
	chapters               *ChapterManager  // optional — nil means chapter tracking is disabled
	narratorCmd            *NarratorCommand // optional — nil means /narrator command is disabled
	achievementCacheLoaded bool
	achievementCache       []storage.Achievement
	chapterSummaryCache    map[int]string
	loadedFromSaveID       string
	loadedFromSaveName     string
}

// NewNarrator creates a narrator for an existing story.
// session must be a valid GameSession created via NewGameSession.
func NewNarrator(
	router *ai.Router,
	db *storage.DB,
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	session *GameSession,
	contextCfg ContextConfig,
	genCfg config.GenerationConfig,
	asciiCfg config.ASCIIArtConfig,
	dataDir string,
	autosaveEvery int,
) *Narrator {
	if autosaveEvery <= 0 {
		autosaveEvery = 5
	}
	// Apply defaults for any zero-value generation config fields.
	if genCfg.Temperature == 0 {
		genCfg.Temperature = 0.8
	}
	if genCfg.MaxTokens == 0 {
		genCfg.MaxTokens = 2048
	}
	if genCfg.TimeoutSeconds == 0 {
		genCfg.TimeoutSeconds = 60
	}
	if asciiCfg.Enabled {
		if asciiCfg.Temperature == 0 {
			asciiCfg.Temperature = 0.4
		}
		if asciiCfg.MaxTokens == 0 {
			asciiCfg.MaxTokens = 400
		}
		if asciiCfg.TimeoutSeconds == 0 {
			asciiCfg.TimeoutSeconds = 25
		}
	}
	return &Narrator{
		router:              router,
		db:                  db,
		story:               story,
		character:           char,
		world:               world,
		session:             session,
		contextCfg:          contextCfg,
		genCfg:              genCfg,
		asciiCfg:            asciiCfg,
		dataDir:             dataDir,
		autosaveEvery:       autosaveEvery,
		chapterSummaryCache: make(map[int]string),
	}
}

// SetRAG wires a RAG pipeline into the narrator. Call after construction to enable
// long-term memory retrieval. Passing nil disables RAG gracefully.
func (n *Narrator) SetRAG(r *rag.RAG) {
	n.rag = r
	// Re-wire chapter manager and narrator command with the new RAG.
	if r != nil {
		if n.chapters == nil {
			n.chapters = NewChapterManager(n.db, n.story.ID, r, n.router)
			_ = n.chapters.EnsureCurrentChapter(0)
		}
		if n.narratorCmd == nil {
			n.narratorCmd = NewNarratorCommand(n.router, n.db, n.story, n.character, n.world, r, n.session)
		}
	}
}

// LastModel returns the AI model used for the last response.
func (n *Narrator) LastModel() string { return n.lastModel }

// LastLatency returns the latency in ms for the last response.
func (n *Narrator) LastLatency() int64 { return n.lastLatency }

// LastTimeToFirstToken returns the time to first streamed content in ms.
func (n *Narrator) LastTimeToFirstToken() int64 { return n.lastTTFT }

// LastUsage returns token and cost data for the last AI response.
func (n *Narrator) LastUsage() ai.Usage { return n.lastUsage }

// LastStreamed reports whether the last response used streaming.
func (n *Narrator) LastStreamed() bool { return n.lastStreamed }

// ASCIIArtEnabled reports whether ambient ASCII generation is enabled.
func (n *Narrator) ASCIIArtEnabled() bool { return n.asciiCfg.Enabled }

// Turn returns the current turn number.
func (n *Narrator) Turn() int { return n.session.Turn() }

// Story returns the story being narrated.
func (n *Narrator) Story() *storage.Story { return n.story }

// Character returns the protagonist.
func (n *Narrator) Character() *storage.Character { return n.character }

// World returns the current world state.
func (n *Narrator) World() *storage.WorldState { return n.world }

// DB returns the database connection.
func (n *Narrator) DB() *storage.DB { return n.db }

// DataDir returns the data directory path.
func (n *Narrator) DataDir() string { return n.dataDir }

// SetLoadedSaveContext marks future saves as a rewind branch from the given snapshot.
func (n *Narrator) SetLoadedSaveContext(save *storage.SaveSnapshot) {
	if n == nil || save == nil {
		return
	}
	n.loadedFromSaveID = strings.TrimSpace(save.ID)
	n.loadedFromSaveName = strings.TrimSpace(save.Name)
}

// BuildSaveMetadata returns branch-aware metadata for manual, quick, or auto saves.
func (n *Narrator) BuildSaveMetadata(kind string) *storage.SaveMetadata {
	if n == nil {
		return nil
	}
	meta := &storage.SaveMetadata{
		Kind: strings.TrimSpace(kind),
	}
	if meta.Kind == "" {
		meta.Kind = "manual"
	}
	if n.loadedFromSaveID != "" {
		meta.LoadedFromSaveID = n.loadedFromSaveID
		meta.LoadedFromSaveName = n.loadedFromSaveName
		label := "Rewind branch"
		if n.loadedFromSaveName != "" {
			label = "Rewind branch from " + n.loadedFromSaveName
		}
		meta.BranchLabel = label
		meta.Notes = []string{
			"Created after loading an earlier snapshot.",
			"This branch preserves alternate choices without overwriting the original line.",
		}
	}
	return meta
}

// SessionID returns the current session ID (empty string if no session).
func (n *Narrator) SessionID() string {
	if n.session == nil {
		return ""
	}
	return n.session.SessionID()
}

// CloseSession closes the active game session.
func (n *Narrator) CloseSession() {
	if n.session != nil {
		_ = n.session.Close(n.db)
		n.session = nil
	}
}

// StartNarration sends the first turn to begin the story.
// Only call this for brand-new stories. For existing stories use ResumeNarration.
// Returns the parsed narrative response.
func (n *Narrator) StartNarration(ctx context.Context) (*NarrativeResponse, error) {
	return n.sendTurn(ctx, prompts.FirstTurnUser)
}

// StartNarrationStream streams the first turn for a brand-new story.
func (n *Narrator) StartNarrationStream(ctx context.Context) (<-chan NarrativeStreamChunk, error) {
	return n.streamTurn(ctx, prompts.FirstTurnUser)
}

// ResumeNarration prepares the narrator to resume an existing story without
// triggering a new first-turn AI call. It restores world.CurrentTurn from the
// DB world state (already loaded by the caller) and returns a synthetic response
// so the narrative view shows the last known state while awaiting player input.
func (n *Narrator) ResumeNarration(ctx context.Context) (*NarrativeResponse, error) {
	// The world.CurrentTurn was loaded from DB by the caller — no reset needed.
	// Restore session turn counter to match DB world state so the next AppendTurn
	// uses the correct turn number.
	n.session.SetTurn(n.world.CurrentTurn)

	// Load the most recent messages by story (not by session) so we can replay
	// the last known narrative even when this is a brand-new session object.
	limit := n.contextCfg.RecentMessageCount
	if limit < 10 {
		limit = 10
	}
	recentMsgs, err := n.db.GetRecentMessagesByStory(n.story.ID, limit)
	if err != nil {
		recentMsgs = nil
	}

	for i := len(recentMsgs) - 1; i >= 0; i-- {
		if recentMsgs[i].Role == "assistant" && recentMsgs[i].MessageType != "narrator" {
			n.restoreTelemetryFromStoredMessage(recentMsgs[i])
			if nr := resumeNarrativeFromStoredMessage(recentMsgs[i], n.world.CurrentLocation); nr != nil {
				return nr, nil
			}
		}
	}

	return &NarrativeResponse{
		Narrative: fmt.Sprintf("Welcome back to %s. Your adventure continues...", n.story.Name),
		Location:  n.world.CurrentLocation,
		Mood:      "neutral",
	}, nil
}

// SendAction sends a player action (choice or free text) and returns the AI narrative.
func (n *Narrator) SendAction(ctx context.Context, action string) (*NarrativeResponse, error) {
	return n.sendTurn(ctx, action)
}

// StreamAction streams a player action and emits the final structured response
// once the upstream provider finishes.
func (n *Narrator) StreamAction(ctx context.Context, action string) (<-chan NarrativeStreamChunk, error) {
	return n.streamTurn(ctx, action)
}

func (n *Narrator) loadEarnedAchievements() []storage.Achievement {
	if n == nil || n.db == nil || n.story == nil {
		return nil
	}
	if n.achievementCacheLoaded {
		return n.achievementCache
	}

	achievements, err := n.db.ListAchievements(n.story.ID)
	if err != nil {
		return nil
	}

	n.achievementCache = achievements
	n.achievementCacheLoaded = true
	return n.achievementCache
}

func (n *Narrator) rememberAchievement(a *storage.Achievement) {
	if n == nil || a == nil || !n.achievementCacheLoaded {
		return
	}
	n.achievementCache = append(n.achievementCache, *a)
}

func (n *Narrator) loadPreviousChapterSummary() string {
	if n == nil || n.db == nil || n.story == nil || n.world == nil || n.chapters == nil || n.world.CurrentChapter <= 1 {
		return ""
	}

	prevChapter := n.world.CurrentChapter - 1
	if summary, ok := n.chapterSummaryCache[prevChapter]; ok {
		return summary
	}

	chapter, err := n.db.GetChapter(n.story.ID, prevChapter)
	if err != nil || chapter == nil {
		n.chapterSummaryCache[prevChapter] = ""
		return ""
	}

	n.chapterSummaryCache[prevChapter] = chapter.Summary
	return chapter.Summary
}

func (n *Narrator) sendTurn(ctx context.Context, input string) (*NarrativeResponse, error) {
	lock, err := n.acquireTurnLock(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.Release() }()

	prep, err := n.prepareTurn(ctx, input)
	if err != nil {
		return nil, err
	}

	resp, err := n.completeTurnResponse(ctx, prep)
	if err != nil {
		return nil, err
	}

	return n.finalizeTurn(ctx, prep, input, resp, 0, false)
}

func (n *Narrator) streamTurn(ctx context.Context, input string) (<-chan NarrativeStreamChunk, error) {
	lock, err := n.acquireTurnLock(ctx)
	if err != nil {
		return nil, err
	}

	prep, err := n.prepareTurn(ctx, input)
	if err != nil {
		_ = lock.Release()
		return nil, err
	}

	if prep.sceneProgression != nil {
		out := make(chan NarrativeStreamChunk, 4)
		go func() {
			defer close(out)
			defer func() { _ = lock.Release() }()

			resp, err := n.completeTurnResponse(ctx, prep)
			if err != nil {
				out <- NarrativeStreamChunk{Err: err}
				return
			}

			narrative, err := n.finalizeTurn(ctx, prep, input, resp, 0, false)
			if err != nil {
				out <- NarrativeStreamChunk{Err: err}
				return
			}

			if resp.Content != "" {
				out <- NarrativeStreamChunk{Delta: resp.Content}
			}
			out <- NarrativeStreamChunk{Done: true, Response: narrative}
		}()
		return out, nil
	}

	stream, providerName, err := n.router.Stream(ctx, prep.req)
	if err != nil {
		_ = lock.Release()
		return nil, err
	}

	out := make(chan NarrativeStreamChunk, 32)
	go func() {
		defer close(out)
		defer func() { _ = lock.Release() }()

		start := time.Now()
		var builder strings.Builder
		var model string
		var usage ai.Usage
		var firstTokenMs int64

		for chunk := range stream {
			if chunk.Error != nil {
				out <- NarrativeStreamChunk{Err: chunk.Error}
				return
			}
			if chunk.Model != "" {
				model = chunk.Model
			}
			if chunk.Usage.TotalTokens > 0 || chunk.Usage.CostUSD > 0 {
				usage = chunk.Usage
			}
			if chunk.Content != "" {
				if firstTokenMs == 0 {
					firstTokenMs = time.Since(start).Milliseconds()
				}
				builder.WriteString(chunk.Content)
				out <- NarrativeStreamChunk{Delta: chunk.Content}
			}
			if chunk.Done {
				resp := ai.Response{
					Content:   builder.String(),
					Model:     model,
					Provider:  providerName,
					LatencyMs: time.Since(start).Milliseconds(),
					Usage:     usage,
				}
				narrative, err := n.finalizeTurn(ctx, prep, input, resp, firstTokenMs, true)
				if err != nil {
					out <- NarrativeStreamChunk{Err: err}
					return
				}
				out <- NarrativeStreamChunk{
					Done:     true,
					Response: narrative,
				}
				return
			}
		}
	}()

	return out, nil
}

func (n *Narrator) acquireTurnLock(ctx context.Context) (*storage.StoryTurnLock, error) {
	if n == nil || n.db == nil || n.story == nil {
		return nil, fmt.Errorf("acquiring turn lock: narrator is not fully initialized")
	}
	owner := fmt.Sprintf("oneday:%d", os.Getpid())
	if sessionID := strings.TrimSpace(n.SessionID()); sessionID != "" {
		owner = fmt.Sprintf("oneday:%s:%d", sessionID, os.Getpid())
	}
	return n.db.AcquireStoryTurnLock(ctx, n.story.ID, owner, 3*time.Minute)
}

type preparedTurn struct {
	inputType        string
	currentTurn      int
	req              ai.Request
	preflightUsage   ai.Usage
	preflightLatency int64
	recentMessages   []storage.ChatMessage
	sceneProgression *SceneProgressionGuidance
}

func (n *Narrator) prepareTurn(ctx context.Context, input string) (*preparedTurn, error) {
	// Determine input type.
	inputType := "free_action"
	if strings.HasPrefix(input, "[Choice ") {
		inputType = "choice"
	}

	// Current turn number (used for NPC context and state changes).
	currentTurn := n.session.Turn()

	// Load recent messages from DB to build context.
	recentMsgs, err := n.db.GetRecentMessages(n.session.SessionID(), n.contextCfg.RecentMessageCount)
	if err != nil {
		return nil, fmt.Errorf("loading messages: %w", err)
	}

	// Load NPCs relevant to this turn (seen within the configured lookback window).
	lookback := n.contextCfg.NPCLookbackTurns
	if lookback <= 0 {
		lookback = 20
	}
	npcs, err := RelevantNPCs(n.db, n.story.ID, currentTurn, lookback, 8)
	if err != nil {
		npcs = nil // non-fatal: proceed without NPC context
	}

	// Retrieve RAG chunks for long-term memory context (non-fatal if unavailable).
	var ragChunks []string
	if n.rag != nil {
		ragChunks, _ = n.rag.Retrieve(ctx, input)
	}

	lastChapterSummary := n.loadPreviousChapterSummary()
	earnedAchievements := n.loadEarnedAchievements()
	var sceneProgression *SceneProgressionGuidance
	var preflightUsage ai.Usage
	var preflightLatency int64
	if signal := detectNarrativeMomentumSignal(n.world, recentMsgs); signal != nil {
		guidance, usage, latency, err := n.evaluateSceneProgression(ctx, recentMsgs, signal)
		preflightUsage = mergeUsage(preflightUsage, usage)
		preflightLatency += latency
		if err == nil {
			sceneProgression = guidance
		}
	}

	// Build the full context using the context builder.
	messages := BuildContext(
		n.story,
		n.character,
		n.world,
		npcs,
		recentMsgs,
		ragChunks,
		lastChapterSummary,
		input,
		earnedAchievements,
		sceneProgression,
	)
	messages = appendRewardBudgetGuidance(messages, n.contextCfg.RewardBudget)

	return &preparedTurn{
		inputType:        inputType,
		currentTurn:      currentTurn,
		preflightUsage:   preflightUsage,
		preflightLatency: preflightLatency,
		recentMessages:   recentMsgs,
		sceneProgression: sceneProgression,
		req: ai.Request{
			Messages:       messages,
			Temperature:    n.genCfg.Temperature,
			MaxTokens:      n.genCfg.MaxTokens,
			ResponseFormat: ai.NarrativeResponseFormat(),
		},
	}, nil
}

func (n *Narrator) completeTurnResponse(ctx context.Context, prep *preparedTurn) (ai.Response, error) {
	if prep == nil {
		return ai.Response{}, fmt.Errorf("missing prepared turn")
	}

	resp, err := n.router.Complete(ctx, prep.req)
	if err != nil {
		return ai.Response{}, err
	}

	if prep.sceneProgression == nil {
		return resp, nil
	}

	return n.rerollStalledNarrativeDraft(ctx, prep, resp)
}

func (n *Narrator) finalizeTurn(
	ctx context.Context,
	prep *preparedTurn,
	input string,
	resp ai.Response,
	firstTokenMs int64,
	streamed bool,
) (*NarrativeResponse, error) {
	n.lastModel = resp.Model
	n.lastLatency = prep.preflightLatency + resp.LatencyMs
	n.lastTTFT = firstTokenMs
	n.lastUsage = mergeUsage(prep.preflightUsage, resp.Usage)
	n.lastStreamed = streamed

	charSnapshot := *n.character
	worldSnapshot := *n.world
	storySnapshot := *n.story
	restoreState := func() {
		*n.character = charSnapshot
		*n.world = worldSnapshot
		*n.story = storySnapshot
	}

	// Parse the structured response.
	narrative, err := parseNarrativeFromAI(resp.Content)
	if err != nil {
		repaired, repairErr := n.repairNarrativeResponse(ctx, input, resp.Content, err)
		if repairErr == nil {
			narrative = repaired
		} else {
			// If repair also fails, wrap the raw text as a minimal narrative.
			narrative = &NarrativeResponse{
				Narrative: normalizeNarrativeText(resp.Content),
				Choices: []Choice{
					{ID: 1, Text: "Continue..."},
				},
				Mood:     "mysterious",
				Location: n.world.CurrentLocation,
			}
		}
	}
	normalizeNarrativeResponse(narrative)
	if continuityErr := detectNarrativeContinuityIssue(n.story, narrative); continuityErr != nil {
		repaired, repairErr := n.repairNarrativeResponse(ctx, input, resp.Content, continuityErr)
		if repairErr != nil {
			restoreState()
			return nil, fmt.Errorf("continuity guard blocked response: %w", continuityErr)
		}
		narrative = repaired
		normalizeNarrativeResponse(narrative)
		if remaining := detectNarrativeContinuityIssue(n.story, narrative); remaining != nil {
			restoreState()
			return nil, fmt.Errorf("continuity guard blocked response after repair: %w", remaining)
		}
	}

	// Apply state_changes from AI response, including NPC creation/updates.
	var appliedChanges []StateChange
	var npcRecorder *npcMutationRecorder
	if len(narrative.StateChanges) > 0 {
		var applyErr error
		var npcStore npcStateStore
		if n.db != nil {
			npcRecorder = newNPCMutationRecorder(directNPCStore{db: n.db})
			npcStore = npcRecorder
		}
		appliedChanges, applyErr = applyStateChangesWithNPCStore(
			narrative.StateChanges,
			n.character,
			n.world,
			n.db,
			npcStore,
			n.story.ID,
			prep.currentTurn,
		)
		if applyErr != nil {
			restoreState()
			return nil, fmt.Errorf("applying state changes: %w", applyErr)
		} else {
			narrative.AppliedStateChanges = appliedChanges
			narrative.EventCallouts = MergeEventCallouts(
				narrative.EventCallouts,
				StateChangesToEventCallouts(appliedChanges),
			)
		}

		// Apply narrator-managed world/setting changes in memory first so the
		// canonical turn commit remains the source of truth for world state.
		_ = ApplyNarratorStateChanges(ctx, narrative.StateChanges, nil, n.story, n.world, nil)
	}
	narrative.TurnDelta = mergeTurnDelta(buildTurnDelta(appliedChanges), narrative.TurnDelta)
	narrative.OpenHooks = activeStoryHooks(loadStoryHooks(n.world))
	narrative.WorldReactions = visibleWorldReactions(loadWorldReactions(n.world))
	normalizeNarrativeResponse(narrative)

	// If AI specified a location (and state_changes didn't already update it), sync it.
	if narrative.Location != "" && narrative.Location != n.world.CurrentLocation {
		n.world.CurrentLocation = narrative.Location
	}

	// Auto-add the current location to known locations if not already tracked.
	if n.world.CurrentLocation != "" {
		AddLocationToWorldState(n.world, n.world.CurrentLocation, prep.currentTurn)
	}

	// Handle chapter end if AI signalled one.
	if narrative.ChapterEnd && n.chapters != nil {
		title := narrative.ChapterTitle
		if title == "" {
			title = fmt.Sprintf("Chapter %d", n.world.CurrentChapter)
		}
		if err := n.chapters.HandleChapterEnd(ctx, prep.currentTurn, title); err != nil {
			_ = err // non-fatal: chapter management failure does not break gameplay
		} else {
			n.world.CurrentChapter++
		}
	}

	n.world.CurrentTurn = prep.currentTurn + 1

	// Extract choice texts for the JSONL entry.
	choiceTexts := make([]string, len(narrative.Choices))
	for i, c := range narrative.Choices {
		choiceTexts[i] = c.Text
	}

	// Persist the turn to JSONL file and DB.
	entry := ChatEntry{
		Timestamp: time.Now(),
		Chapter:   n.world.CurrentChapter,
		Location:  n.world.CurrentLocation,
		Input: &ChatInput{
			Type: prep.inputType,
			Text: input,
		},
		Output: &ChatOutput{
			Narrative:         narrative.Narrative,
			Choices:           choiceTexts,
			ChoicesData:       narrative.Choices,
			TurnDelta:         narrative.TurnDelta,
			Mood:              narrative.Mood,
			Location:          n.world.CurrentLocation,
			SceneType:         narrative.SceneType,
			DialogueBlocks:    narrative.DialogueBlocks,
			EntitiesMentioned: narrative.EntitiesMentioned,
			EventCallouts:     narrative.EventCallouts,
			ASCIICue:          narrative.ASCIICue,
			ASCIIArt:          narrative.ASCIIArt,
			OpenHooks:         narrative.OpenHooks,
			WorldReactions:    narrative.WorldReactions,
			SocialDuel:        narrative.SocialDuel,
			StateChanges:      narrative.StateChanges,
		},
		AIModel:    resp.Model,
		AILatency:  n.lastLatency,
		AITTFT:     n.lastTTFT,
		AIUsage:    n.lastUsage,
		AIStreamed: n.lastStreamed,
	}
	var beforeCommit func(*sql.Tx) error
	if npcRecorder != nil {
		beforeCommit = func(tx *sql.Tx) error {
			return npcRecorder.Commit(txNPCStore{db: n.db, tx: tx})
		}
	}
	if err := n.session.CommitTurnWithSideEffects(n.db, n.character, n.world, entry, beforeCommit); err != nil {
		if IsMirrorSyncError(err) {
			log.Printf("oneday: canonical turn %d committed but jsonl mirror failed: %v", prep.currentTurn, err)
		} else {
			restoreState()
			return nil, fmt.Errorf("committing canonical turn %d: %w", prep.currentTurn, err)
		}
	}

	if storySnapshot.SettingJSON != n.story.SettingJSON {
		if err := n.db.UpdateStorySetting(n.story.ID, n.story.SettingJSON); err != nil {
			log.Printf("oneday: story setting update failed for story %s after canonical turn %d: %v", n.story.ID, prep.currentTurn, err)
		}
	}

	// Process achievement_earned from AI only after the canonical turn is committed.
	if narrative.AchievementEarned != nil {
		if persisted := ValidateAndPersistAchievement(n.db, n.story.ID, narrative.AchievementEarned); persisted != nil {
			narrative.PersistedAchievement = persisted
			n.rememberAchievement(persisted)
		}
	}

	// Update last_seen_turn for any NPCs mentioned by name in the narrative text.
	if narrative.Narrative != "" {
		if err := UpdateNPCLastSeen(n.db, n.story.ID, narrative.Narrative, prep.currentTurn); err != nil {
			log.Printf("oneday: npc last_seen update failed for story %s turn %d: %v", n.story.ID, prep.currentTurn, err)
		}
	}

	// Async RAG summarization — fire-and-forget after turn is persisted.
	// If summarization fails, the next trigger will pick up the gap.
	if n.rag != nil {
		storyID := n.story.ID
		turn := n.world.CurrentTurn - 1
		ragPipeline := n.rag
		db := n.db
		go func() {
			bgCtx := context.Background()
			startTurn, endTurn, should, err := ragPipeline.PendingSummaryWindow(bgCtx, turn)
			if err != nil || !should {
				return
			}
			msgs, err := db.GetStoryMessagesByTurnRange(storyID, startTurn, endTurn)
			if err != nil {
				return
			}
			_, _ = ragPipeline.MaybeSummarize(bgCtx, msgs, endTurn)
		}()
	}

	return narrative, nil
}

// ShouldAutosave returns true if the current turn count triggers an autosave.
// Called after SendAction to let the TUI fire the autosave as a tea.Cmd.
func (n *Narrator) ShouldAutosave() bool {
	if n.autosaveEvery <= 0 {
		return false
	}
	currentTurn := n.session.Turn()
	return currentTurn > 0 && currentTurn%n.autosaveEvery == 0
}

// AutosaveCmd returns a tea.Cmd that runs an autosave in the background
// and emits AutosaveCompleteMsg when done.
func (n *Narrator) AutosaveCmd() tea.Cmd {
	db := n.db
	dataDir := n.dataDir
	story := n.story
	// Take snapshots of current state to avoid races.
	char := *n.character
	world := *n.world
	sessionID := n.session.SessionID()
	meta := n.BuildSaveMetadata("autosave")

	return func() tea.Msg {
		err := AutosaveWithMetadata(db, dataDir, story, &char, &world, sessionID, meta)
		return AutosaveCompleteMsg{Err: err}
	}
}

// ExecuteNarratorCommand processes a /narrator (/n) meta-command.
// The player speaks to the AI as a game master. Does NOT increment turn counter.
func (n *Narrator) ExecuteNarratorCommand(ctx context.Context, input string) (*NarratorMetaResponse, error) {
	if n.narratorCmd == nil {
		// Lazily initialize narrator command handler even without RAG.
		n.narratorCmd = NewNarratorCommand(n.router, n.db, n.story, n.character, n.world, n.rag, n.session)
	}
	return n.narratorCmd.Execute(ctx, input)
}

// ExecuteAsideQuestion asks a contextual out-of-band question without advancing
// the story turn or mutating state.
func (n *Narrator) ExecuteAsideQuestion(ctx context.Context, input string) (string, error) {
	if n.narratorCmd == nil {
		n.narratorCmd = NewNarratorCommand(n.router, n.db, n.story, n.character, n.world, n.rag, n.session)
	}
	return n.narratorCmd.ExecuteAside(ctx, input)
}

// ExecuteGuideCommand stores a soft future-facing directive without advancing
// the story turn.
func (n *Narrator) ExecuteGuideCommand(ctx context.Context, input string) (*GuideMetaResponse, error) {
	if n.narratorCmd == nil {
		n.narratorCmd = NewNarratorCommand(n.router, n.db, n.story, n.character, n.world, n.rag, n.session)
	}
	return n.narratorCmd.ExecuteGuide(ctx, input)
}

// GetChapterSummaries returns formatted chapter summaries for the /journal command.
func (n *Narrator) GetChapterSummaries() (string, error) {
	if n.chapters == nil {
		n.chapters = NewChapterManager(n.db, n.story.ID, n.rag, n.router)
	}
	return n.chapters.GetChapterSummaries()
}

// parseNarrativeFromAI extracts the JSON block from an AI response and
// unmarshals it directly into engine.NarrativeResponse (which includes Location).
func parseNarrativeFromAI(text string) (*NarrativeResponse, error) {
	raw, err := ai.ExtractJSONPayload(text)
	if err != nil {
		return nil, fmt.Errorf("extracting narrative JSON: %w", err)
	}
	if raw == "" {
		return nil, fmt.Errorf("no JSON block found in AI response")
	}

	var nr NarrativeResponse
	if err := json.Unmarshal([]byte(raw), &nr); err != nil {
		return nil, fmt.Errorf("unmarshaling narrative JSON: %w", err)
	}

	if nr.Narrative == "" {
		return nil, fmt.Errorf("empty narrative in response")
	}

	return &nr, nil
}

type persistedAssistantMeta struct {
	Model              string      `json:"model"`
	LatencyMS          int64       `json:"latency_ms"`
	TimeToFirstTokenMS int64       `json:"time_to_first_token_ms"`
	Usage              ai.Usage    `json:"usage"`
	Streamed           bool        `json:"streamed"`
	Mood               string      `json:"mood"`
	Location           string      `json:"location"`
	Choices            []string    `json:"choices"`
	Output             *ChatOutput `json:"output"`
}

func (n *Narrator) restoreTelemetryFromStoredMessage(msg storage.ChatMessage) {
	meta, ok := parsePersistedAssistantMeta(msg.MetadataJSON)
	if !ok {
		return
	}
	n.lastModel = strings.TrimSpace(meta.Model)
	n.lastLatency = meta.LatencyMS
	n.lastTTFT = meta.TimeToFirstTokenMS
	n.lastUsage = meta.Usage
	n.lastStreamed = meta.Streamed
}

func parsePersistedAssistantMeta(raw string) (*persistedAssistantMeta, bool) {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "{}" {
		return nil, false
	}

	var meta persistedAssistantMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, false
	}
	return &meta, true
}

func resumeNarrativeFromStoredMessage(msg storage.ChatMessage, defaultLocation string) *NarrativeResponse {
	narrative := normalizeNarrativeText(msg.Content)
	if narrative == "" {
		return nil
	}

	nr := &NarrativeResponse{
		Narrative: narrative,
		Location:  defaultLocation,
		Mood:      "neutral",
	}

	if strings.TrimSpace(msg.MetadataJSON) == "" || strings.TrimSpace(msg.MetadataJSON) == "{}" {
		return nr
	}

	meta, ok := parsePersistedAssistantMeta(msg.MetadataJSON)
	if !ok {
		return nr
	}

	if meta.Output != nil {
		if text := normalizeNarrativeText(meta.Output.Narrative); text != "" {
			nr.Narrative = text
		}
		if len(meta.Output.ChoicesData) > 0 {
			nr.Choices = sanitizeChoices(meta.Output.ChoicesData)
		} else if len(meta.Output.Choices) > 0 {
			nr.Choices = stringsToChoices(meta.Output.Choices)
		}
		nr.Mood = firstNonEmpty(meta.Output.Mood, meta.Mood, nr.Mood)
		nr.Location = firstNonEmpty(meta.Output.Location, meta.Location, nr.Location)
		nr.SceneType = strings.TrimSpace(meta.Output.SceneType)
		nr.TurnDelta = normalizeTurnDelta(meta.Output.TurnDelta)
		nr.DialogueBlocks = normalizeDialogueBlocks(meta.Output.DialogueBlocks)
		nr.EntitiesMentioned = meta.Output.EntitiesMentioned
		nr.EventCallouts = normalizeEventCallouts(meta.Output.EventCallouts)
		nr.ASCIICue = normalizeASCIICue(meta.Output.ASCIICue)
		nr.ASCIIArt = normalizeASCIIArt(meta.Output.ASCIIArt)
		nr.OpenHooks = activeStoryHooks(meta.Output.OpenHooks)
		nr.WorldReactions = visibleWorldReactions(meta.Output.WorldReactions)
		nr.SocialDuel = normalizeSocialDuelCue(meta.Output.SocialDuel)
	} else {
		nr.Mood = firstNonEmpty(meta.Mood, nr.Mood)
		nr.Location = firstNonEmpty(meta.Location, nr.Location)
		nr.Choices = stringsToChoices(meta.Choices)
	}

	normalizeNarrativeResponse(nr)
	return nr
}

func normalizeNarrativeResponse(nr *NarrativeResponse) {
	if nr == nil {
		return
	}
	nr.Narrative = normalizeNarrativeText(nr.Narrative)
	nr.Choices = sanitizeChoices(nr.Choices)
	nr.TurnDelta = normalizeTurnDelta(nr.TurnDelta)
	nr.DialogueBlocks = normalizeDialogueBlocks(nr.DialogueBlocks)
	nr.EventCallouts = normalizeEventCallouts(nr.EventCallouts)
	nr.ASCIICue = normalizeASCIICue(nr.ASCIICue)
	nr.ASCIIArt = normalizeASCIIArt(nr.ASCIIArt)
	nr.OpenHooks = activeStoryHooks(nr.OpenHooks)
	nr.WorldReactions = visibleWorldReactions(nr.WorldReactions)
	nr.SocialDuel = normalizeSocialDuelCue(nr.SocialDuel)
}

func normalizeDialogueBlocks(blocks []DialogueBlock) []DialogueBlock {
	if len(blocks) == 0 {
		return nil
	}
	out := make([]DialogueBlock, 0, len(blocks))
	for _, block := range blocks {
		text := normalizeNarrativeText(block.Text)
		speaker := strings.TrimSpace(block.Speaker)
		role := strings.TrimSpace(block.Role)
		if text == "" {
			continue
		}
		out = append(out, DialogueBlock{
			Speaker: speaker,
			Role:    role,
			Text:    text,
		})
	}
	return out
}

func normalizeEventCallouts(callouts []EventCallout) []EventCallout {
	if len(callouts) == 0 {
		return nil
	}
	out := make([]EventCallout, 0, len(callouts))
	for _, callout := range callouts {
		title := normalizeNarrativeText(callout.Title)
		detail := normalizeNarrativeText(callout.Detail)
		if title == "" && detail == "" {
			continue
		}
		out = append(out, EventCallout{
			Kind:   strings.TrimSpace(callout.Kind),
			Title:  title,
			Detail: detail,
		})
	}
	return out
}

func normalizeASCIICue(cue *ASCIIArtCue) *ASCIIArtCue {
	if cue == nil {
		return nil
	}
	normalized := &ASCIIArtCue{
		Kind:      strings.TrimSpace(cue.Kind),
		Subject:   normalizeNarrativeText(cue.Subject),
		Detail:    normalizeNarrativeText(cue.Detail),
		Placement: strings.TrimSpace(cue.Placement),
	}
	if normalized.Subject == "" {
		return nil
	}
	if normalized.Placement == "" {
		normalized.Placement = "scene_header"
	}
	return normalized
}

func normalizeASCIIArt(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Trim(text, "\n")
}

func normalizeNarrativeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if strings.Contains(text, `\n`) || strings.Contains(text, `\t`) || strings.Contains(text, `\"`) {
		replacer := strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\"`, `"`)
		text = replacer.Replace(text)
	}
	return strings.TrimSpace(text)
}

func sanitizeChoices(choices []Choice) []Choice {
	if len(choices) == 0 {
		return nil
	}

	out := make([]Choice, 0, len(choices))
	seen := make(map[string]bool, len(choices))
	for _, choice := range choices {
		text := strings.Join(strings.Fields(normalizeNarrativeText(choice.Text)), " ")
		if text == "" {
			continue
		}
		key := strings.ToLower(text)
		if seen[key] {
			continue
		}
		seen[key] = true
		choice.ID = len(out) + 1
		choice.Text = text
		out = append(out, choice)
	}
	return out
}

func stringsToChoices(items []string) []Choice {
	if len(items) == 0 {
		return nil
	}
	choices := make([]Choice, 0, len(items))
	for _, item := range items {
		choices = append(choices, Choice{Text: item})
	}
	return sanitizeChoices(choices)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// GenerateAmbientASCII uses the dedicated ASCII-art prompt/model for a single scene.
// The generated art is also persisted back into the latest assistant-turn metadata
// so resume/load can restore it later.
func (n *Narrator) GenerateAmbientASCII(ctx context.Context, turn int, base *NarrativeResponse) (string, string, error) {
	if !n.asciiCfg.Enabled || base == nil || base.ASCIICue == nil {
		return "", "", nil
	}

	timeout := time.Duration(n.asciiCfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 25 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cue := normalizeASCIICue(base.ASCIICue)
	if cue == nil {
		return "", "", nil
	}

	req := ai.Request{
		Messages: []ai.Message{
			{Role: ai.RoleSystem, Content: prompts.ASCIIArtSystem()},
			{Role: ai.RoleUser, Content: prompts.ASCIIArtUser(
				n.story.Name,
				firstNonEmpty(base.Location, n.world.CurrentLocation),
				base.SceneType,
				base.Mood,
				base.Narrative,
				cue.Kind,
				cue.Subject,
				cue.Detail,
				cue.Placement,
			)},
		},
		Model:          strings.TrimSpace(n.asciiCfg.Model),
		Temperature:    n.asciiCfg.Temperature,
		MaxTokens:      n.asciiCfg.MaxTokens,
		ResponseFormat: ai.ASCIIArtResponseFormat(),
	}

	resp, err := n.router.Complete(ctx, req)
	if err != nil {
		configuredModel := strings.TrimSpace(req.Model)
		if configuredModel != "" {
			fallbackReq := req
			fallbackReq.Model = ""
			resp, err = n.router.Complete(ctx, fallbackReq)
		}
		if err != nil {
			if configuredModel != "" {
				return "", "", fmt.Errorf("ascii art generation failed for %q and provider default: %w", configuredModel, err)
			}
			return "", "", err
		}
	}

	payload, err := ai.ParseASCIIArtJSON(resp.Content)
	if err != nil {
		return "", resp.Model, err
	}
	if payload == nil {
		return "", resp.Model, nil
	}

	art := normalizeASCIIArt(payload.ASCIIArt)
	if art == "" {
		return "", resp.Model, nil
	}

	if err := n.persistAmbientASCII(turn, base, art, resp.Model); err != nil {
		_ = err
	}

	return art, resp.Model, nil
}

func (n *Narrator) persistAmbientASCII(turn int, base *NarrativeResponse, art string, artModel string) error {
	if turn < 0 || base == nil {
		return nil
	}

	choiceTexts := make([]string, len(base.Choices))
	for i, choice := range base.Choices {
		choiceTexts[i] = choice.Text
	}

	output := &ChatOutput{
		Narrative:         base.Narrative,
		Choices:           choiceTexts,
		ChoicesData:       base.Choices,
		TurnDelta:         base.TurnDelta,
		Mood:              base.Mood,
		Location:          firstNonEmpty(base.Location, n.world.CurrentLocation),
		SceneType:         base.SceneType,
		DialogueBlocks:    base.DialogueBlocks,
		EntitiesMentioned: base.EntitiesMentioned,
		EventCallouts:     base.EventCallouts,
		ASCIICue:          base.ASCIICue,
		ASCIIArt:          art,
		OpenHooks:         base.OpenHooks,
		WorldReactions:    base.WorldReactions,
		SocialDuel:        base.SocialDuel,
		StateChanges:      base.StateChanges,
	}

	meta := map[string]any{
		"model":                  n.lastModel,
		"latency_ms":             n.lastLatency,
		"time_to_first_token_ms": n.lastTTFT,
		"usage":                  n.lastUsage,
		"streamed":               n.lastStreamed,
		"mood":                   base.Mood,
		"location":               output.Location,
		"choices":                choiceTexts,
		"output":                 output,
	}
	if strings.TrimSpace(artModel) != "" {
		meta["ascii_model"] = artModel
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return n.db.UpdateAssistantMessageMetadata(n.story.ID, turn, string(metaJSON))
}
