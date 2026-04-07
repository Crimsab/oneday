package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
)

// narratorJSONRe matches fenced ```json ... ``` blocks in AI output.
var narratorJSONRe = regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```")

// AutosaveCompleteMsg is sent via Bubbletea when an autosave finishes.
type AutosaveCompleteMsg struct {
	Err error
}

// Narrator manages the gameplay AI conversation.
type Narrator struct {
	router        *ai.Router
	db            *storage.DB
	story         *storage.Story
	character     *storage.Character
	world         *storage.WorldState
	session       *GameSession
	contextCfg    ContextConfig
	lastModel     string
	lastLatency   int64
	dataDir       string
	autosaveEvery int
	rag           *rag.RAG // optional — nil means RAG is disabled
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
	dataDir string,
	autosaveEvery int,
) *Narrator {
	if autosaveEvery <= 0 {
		autosaveEvery = 5
	}
	return &Narrator{
		router:        router,
		db:            db,
		story:         story,
		character:     char,
		world:         world,
		session:       session,
		contextCfg:    contextCfg,
		dataDir:       dataDir,
		autosaveEvery: autosaveEvery,
	}
}

// SetRAG wires a RAG pipeline into the narrator. Call after construction to enable
// long-term memory retrieval. Passing nil disables RAG gracefully.
func (n *Narrator) SetRAG(r *rag.RAG) {
	n.rag = r
}

// LastModel returns the AI model used for the last response.
func (n *Narrator) LastModel() string { return n.lastModel }

// LastLatency returns the latency in ms for the last response.
func (n *Narrator) LastLatency() int64 { return n.lastLatency }

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
// Returns the parsed narrative response.
func (n *Narrator) StartNarration(ctx context.Context) (*NarrativeResponse, error) {
	return n.sendTurn(ctx, prompts.FirstTurnUser)
}

// SendAction sends a player action (choice or free text) and returns the AI narrative.
func (n *Narrator) SendAction(ctx context.Context, action string) (*NarrativeResponse, error) {
	return n.sendTurn(ctx, action)
}

func (n *Narrator) sendTurn(ctx context.Context, input string) (*NarrativeResponse, error) {
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
	npcs, err := n.db.ListRecentNPCs(n.story.ID, currentTurn, lookback)
	if err != nil {
		npcs = nil // non-fatal: proceed without NPC context
	}

	// Retrieve RAG chunks for long-term memory context (non-fatal if unavailable).
	var ragChunks []string
	if n.rag != nil {
		ragChunks, _ = n.rag.Retrieve(ctx, input)
	}

	// Build the full context using the context builder.
	messages := BuildContext(
		n.story,
		n.character,
		n.world,
		npcs,
		recentMsgs,
		ragChunks,
		input,
	)

	start := time.Now()
	req := ai.Request{
		Messages:    messages,
		Temperature: 0.85,
		MaxTokens:   2048,
	}

	resp, err := n.router.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	n.lastModel = resp.Model
	n.lastLatency = time.Since(start).Milliseconds()

	// Parse the structured response.
	narrative, err := parseNarrativeFromAI(resp.Content)
	if err != nil {
		// If parsing fails, wrap the raw text as a minimal narrative.
		narrative = &NarrativeResponse{
			Narrative: resp.Content,
			Choices: []Choice{
				{ID: 1, Text: "Continue..."},
			},
			Mood:     "mysterious",
			Location: n.world.CurrentLocation,
		}
	}

	// Apply state_changes from AI response, including NPC creation/updates.
	if len(narrative.StateChanges) > 0 {
		_, applyErr := ApplyStateChanges(
			narrative.StateChanges,
			n.character,
			n.world,
			n.db,
			n.story.ID,
			currentTurn,
		)
		if applyErr != nil {
			// Non-fatal: log but continue.
			_ = applyErr
		}
	}

	// Update last_seen_turn for any NPCs mentioned by name in the narrative text.
	if narrative.Narrative != "" {
		_ = UpdateNPCLastSeen(n.db, n.story.ID, narrative.Narrative, currentTurn)
	}

	// If AI specified a location (and state_changes didn't already update it), sync it.
	if narrative.Location != "" && narrative.Location != n.world.CurrentLocation {
		n.world.CurrentLocation = narrative.Location
	}

	// Persist full character state (stats, traits, skills, inventory) to DB.
	if err := n.db.UpdateCharacterFull(n.character); err != nil {
		_ = err // non-fatal
	}

	n.world.CurrentTurn = currentTurn + 1
	if err := n.db.UpdateWorldState(n.world); err != nil {
		_ = err // non-fatal
	}

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
			Type: inputType,
			Text: input,
		},
		Output: &ChatOutput{
			Narrative:    narrative.Narrative,
			Choices:      choiceTexts,
			Mood:         narrative.Mood,
			StateChanges: narrative.StateChanges,
		},
		AIModel:   resp.Model,
		AILatency: n.lastLatency,
	}
	if err := n.session.AppendTurn(n.db, entry); err != nil {
		// Non-fatal: log but don't fail the turn.
		_ = err
	}

	// Async RAG summarization — fire-and-forget after turn is persisted.
	// If summarization fails, the next trigger will pick up the gap.
	if n.rag != nil {
		storyID := n.story.ID
		turn := currentTurn
		ragPipeline := n.rag
		db := n.db
		go func() {
			bgCtx := context.Background()
			msgs, err := db.GetStoryMessagesByTurnRange(storyID, 0, turn)
			if err != nil {
				return
			}
			_, _ = ragPipeline.MaybeSummarize(bgCtx, msgs, turn)
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

	return func() tea.Msg {
		err := Autosave(db, dataDir, story, &char, &world, sessionID)
		return AutosaveCompleteMsg{Err: err}
	}
}

// parseNarrativeFromAI extracts the JSON block from an AI response and
// unmarshals it directly into engine.NarrativeResponse (which includes Location).
func parseNarrativeFromAI(text string) (*NarrativeResponse, error) {
	matches := narratorJSONRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no JSON block found in AI response")
	}

	var nr NarrativeResponse
	if err := json.Unmarshal([]byte(matches[1]), &nr); err != nil {
		return nil, fmt.Errorf("unmarshaling narrative JSON: %w", err)
	}

	if nr.Narrative == "" {
		return nil, fmt.Errorf("empty narrative in response")
	}

	return &nr, nil
}
