package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
)

// NarratorMetaResponse is the AI's structured response to a /narrator command.
type NarratorMetaResponse struct {
	Message      string                 `json:"message"`
	StateChanges map[string]interface{} `json:"state_changes,omitempty"`
}

// NarratorCommand processes /narrator meta-commands.
// These are out-of-story interactions where the player speaks to the AI as game master.
type NarratorCommand struct {
	router    *ai.Router
	db        *storage.DB
	story     *storage.Story
	character *storage.Character
	world     *storage.WorldState
	rag       *rag.RAG
	session   *GameSession
}

// NewNarratorCommand creates a new NarratorCommand handler.
func NewNarratorCommand(
	router *ai.Router,
	db *storage.DB,
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	ragPipeline *rag.RAG,
	session *GameSession,
) *NarratorCommand {
	return &NarratorCommand{
		router:    router,
		db:        db,
		story:     story,
		character: char,
		world:     world,
		rag:       ragPipeline,
		session:   session,
	}
}

// Execute processes a /narrator command and returns the AI response message.
// The input is the player's text after "/n " or "/narrator ".
// Does NOT increment the turn counter.
func (nc *NarratorCommand) Execute(ctx context.Context, input string) (*NarratorMetaResponse, error) {
	if strings.TrimSpace(input) == "" {
		return &NarratorMetaResponse{
			Message: "I'm listening. What would you like to add or change about the world?",
		}, nil
	}

	// Build NPC context for the prompt.
	npcsContext := nc.buildNPCContext(ctx)

	// Build world state JSON for the prompt.
	worldStateJSON := fmt.Sprintf(`{"location":"%s","chapter":%d,"turn":%d}`,
		nc.world.CurrentLocation, nc.world.CurrentChapter, nc.world.CurrentTurn)

	// Build the meta system prompt.
	systemPrompt := prompts.NarratorMetaSystem(
		nc.story.Name,
		nc.story.Language,
		nc.story.WritingStyle,
		nc.story.PromptDirectives,
		nc.story.SettingJSON,
		worldStateJSON,
		npcsContext,
	)

	// Call the AI.
	req := ai.Request{
		Messages: []ai.Message{
			{Role: ai.RoleSystem, Content: systemPrompt},
			{Role: ai.RoleUser, Content: input},
		},
		Temperature:    0.7,
		MaxTokens:      1024,
		ResponseFormat: ai.NarratorMetaResponseFormat(),
	}

	resp, err := nc.router.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("narrator meta AI call: %w", err)
	}

	// Parse the JSON response.
	metaResp, err := parseNarratorMetaResponse(resp.Content)
	if err != nil || metaResp == nil {
		// Fallback: treat full response as the message.
		metaResp = &NarratorMetaResponse{
			Message: resp.Content,
		}
	}

	// Apply state changes (extended types for world/story modifications).
	if len(metaResp.StateChanges) > 0 {
		if err := nc.applyNarratorStateChanges(ctx, metaResp.StateChanges); err != nil {
			// Non-fatal: log but continue showing the message.
			metaResp.Message += fmt.Sprintf("\n\n_(Note: some changes could not be applied: %v)_", err)
		}
	}

	// Persist the interaction canonically without advancing the story turn.
	if err := nc.logNarratorInteraction(input, metaResp.Message); err != nil {
		metaResp.Message += fmt.Sprintf("\n\n_(Note: this narrator exchange could not be saved cleanly: %v)_", err)
	}

	return metaResp, nil
}

// ExecuteAside answers a quick contextual question without mutating state or
// logging a new in-world interaction.
func (nc *NarratorCommand) ExecuteAside(ctx context.Context, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "Ask anything about the current story and I'll answer without advancing the scene.", nil
	}

	npcsContext := nc.buildNPCContext(ctx)
	worldStateJSON := fmt.Sprintf(`{"location":"%s","chapter":%d,"turn":%d}`,
		nc.world.CurrentLocation, nc.world.CurrentChapter, nc.world.CurrentTurn)

	systemPrompt := prompts.NarratorAsideSystem(
		nc.story.Name,
		nc.story.Language,
		nc.story.WritingStyle,
		nc.story.PromptDirectives,
		nc.story.SettingJSON,
		worldStateJSON,
		npcsContext,
	)

	req := ai.Request{
		Messages: []ai.Message{
			{Role: ai.RoleSystem, Content: systemPrompt},
			{Role: ai.RoleUser, Content: input},
		},
		Temperature: 0.4,
		MaxTokens:   512,
	}

	resp, err := nc.router.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("narrator aside AI call: %w", err)
	}
	return strings.TrimSpace(resp.Content), nil
}

// applyNarratorStateChanges applies the extended state changes from a /narrator response.
// These extend the normal ApplyStateChanges with story/world mutation operations.
func (nc *NarratorCommand) applyNarratorStateChanges(ctx context.Context, changes map[string]interface{}) error {
	// First apply standard state changes (npc_disposition, npc_thoughts, etc).
	// We pass a copy of the character and world since narrator shouldn't modify player stats.
	charCopy := *nc.character
	worldCopy := *nc.world

	// Separate narrator-specific changes from standard ones.
	narratorChanges := make(map[string]interface{})
	standardChanges := make(map[string]interface{})

	narratorKeys := map[string]bool{
		"setting_factions_add":   true,
		"setting_cultures_add":   true,
		"setting_dangers_add":    true,
		"setting_rules_add":      true,
		"setting_tone_add":       true,
		"world_location_add":     true,
		"world_event_add":        true,
		"world_faction_standing": true,
		"npc_desires":            true,
	}

	for k, v := range changes {
		if narratorKeys[k] {
			narratorChanges[k] = v
		} else {
			standardChanges[k] = v
		}
	}

	// Apply standard changes (NPC thoughts, notes, dispositions).
	if len(standardChanges) > 0 {
		_, err := ApplyStateChanges(standardChanges, &charCopy, &worldCopy, nc.db, nc.story.ID, nc.world.CurrentTurn)
		if err != nil {
			// Non-fatal.
			_ = err
		}
	}

	// Apply narrator-specific changes.
	return ApplyNarratorStateChanges(ctx, narratorChanges, nc.db, nc.story, nc.world, nc.rag)
}

// buildNPCContext builds a formatted NPC context string for the narrator meta prompt.
func (nc *NarratorCommand) buildNPCContext(_ context.Context) string {
	npcs, err := nc.db.ListNPCs(nc.story.ID)
	if err != nil || len(npcs) == 0 {
		return ""
	}

	var parts []string
	for i := range npcs {
		parts = append(parts, FormatNPCForContext(&npcs[i]))
	}
	return strings.Join(parts, "\n---\n")
}

// logNarratorInteraction saves the /narrator interaction to the main session
// history without incrementing the story turn.
func (nc *NarratorCommand) logNarratorInteraction(input, response string) error {
	if nc.db == nil || nc.session == nil {
		return nil
	}

	return nc.session.AppendHistoryEntry(nc.db, ChatEntry{
		Turn:        nc.world.CurrentTurn,
		Timestamp:   time.Now(),
		Chapter:     nc.world.CurrentChapter,
		Location:    nc.world.CurrentLocation,
		MessageType: "narrator",
		Input: &ChatInput{
			Type: "command",
			Text: "/narrator " + input,
		},
		Output: &ChatOutput{
			Narrative: response,
			Mood:      "neutral",
			Location:  nc.world.CurrentLocation,
		},
	})
}

// parseNarratorMetaResponse extracts JSON from the AI response and unmarshals it.
func parseNarratorMetaResponse(text string) (*NarratorMetaResponse, error) {
	raw := extractJSONFromResponse(text)
	if raw == "" {
		return nil, fmt.Errorf("no JSON block found in narrator meta response")
	}

	var r NarratorMetaResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return nil, fmt.Errorf("unmarshaling narrator meta response: %w", err)
	}
	if r.Message == "" {
		return nil, fmt.Errorf("empty message in narrator meta response")
	}
	return &r, nil
}
