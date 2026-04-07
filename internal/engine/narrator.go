package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/storage"
)

// narratorJSONRe matches fenced ```json ... ``` blocks in AI output.
var narratorJSONRe = regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```")

// Narrator manages the gameplay AI conversation.
type Narrator struct {
	router      *ai.Router
	db          *storage.DB
	story       *storage.Story
	character   *storage.Character
	world       *storage.WorldState
	session     *GameSession
	contextCfg  ContextConfig
	lastModel   string
	lastLatency int64
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
) *Narrator {
	return &Narrator{
		router:     router,
		db:         db,
		story:      story,
		character:  char,
		world:      world,
		session:    session,
		contextCfg: contextCfg,
	}
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
	// Load recent messages from DB to build context.
	recentMsgs, err := n.db.GetRecentMessages(n.session.SessionID(), n.contextCfg.RecentMessageCount)
	if err != nil {
		return nil, fmt.Errorf("loading recent messages: %w", err)
	}

	// Build the full context using the context builder.
	messages := BuildContext(
		n.story,
		n.character,
		n.world,
		recentMsgs,
		n.contextCfg.RAGChunks,
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

	// Update world location if changed.
	if narrative.Location != "" && narrative.Location != n.world.CurrentLocation {
		n.world.CurrentLocation = narrative.Location
	}
	n.world.CurrentTurn = n.session.Turn() + 1

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
			Type: "free_action",
			Text: input,
		},
		Output: &ChatOutput{
			Narrative: narrative.Narrative,
			Choices:   choiceTexts,
			Mood:      narrative.Mood,
		},
		AIModel:   resp.Model,
		AILatency: n.lastLatency,
	}
	if err := n.session.AppendTurn(n.db, entry); err != nil {
		// Non-fatal: log but don't fail the turn.
		_ = err
	}

	return narrative, nil
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
