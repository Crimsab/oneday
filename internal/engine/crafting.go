package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/storage"
)

// craftingJSONRe reuses narratorJSONRe from narrator.go to extract JSON blocks.

// CraftedItem represents an item created through crafting.
type CraftedItem struct {
	Name        string   `json:"name"`
	Description string   `json:"description"` // narrative description
	Effect      string   `json:"effect"`      // what it does (narrative, not numerical)
	Materials   []string `json:"materials"`   // what was consumed to make it
	CraftedAt   string   `json:"crafted_at"`  // timestamp
}

// CraftingResponse is the AI's evaluation of a crafting attempt.
type CraftingResponse struct {
	Feasible     bool         `json:"feasible"`
	Narrative    string       `json:"narrative"`              // AI's description of the attempt
	Item         *CraftedItem `json:"item,omitempty"`         // only if feasible
	Missing      []string     `json:"missing,omitempty"`      // what player lacks
	Alternatives []string     `json:"alternatives,omitempty"` // what they could make instead
	Choices      []Choice     `json:"choices,omitempty"`      // next options
}

// CraftingEngine manages a crafting conversation session.
type CraftingEngine struct {
	narrator     *Narrator
	session      *GameSession
	subSessionID string
	chatHistory  []ai.Message // conversation within this crafting session
	turnCount    int
}

// NewCraftingEngine starts a crafting session.
func NewCraftingEngine(narrator *Narrator) (*CraftingEngine, error) {
	// Open sub-session JSONL for this crafting session.
	subSessionID, err := narrator.session.OpenSubSession("crafting")
	if err != nil {
		return nil, fmt.Errorf("opening crafting sub-session: %w", err)
	}

	return &CraftingEngine{
		narrator:     narrator,
		session:      narrator.session,
		subSessionID: subSessionID,
		chatHistory:  []ai.Message{},
	}, nil
}

// SendMessage sends a player message in the crafting conversation.
// The player describes what they want to craft or asks about possibilities.
func (ce *CraftingEngine) SendMessage(ctx context.Context, message string) (*CraftingResponse, error) {
	// Append player message to chat history.
	ce.chatHistory = append(ce.chatHistory, ai.Message{
		Role:    "user",
		Content: message,
	})

	// Build the system prompt with current inventory, recipes, skills.
	char := ce.narrator.character
	story := ce.narrator.story

	inventoryJSON := buildInventoryContext(char)
	knownRecipesJSON := char.KnownRecipesJSON
	if knownRecipesJSON == "" || knownRecipesJSON == "null" {
		knownRecipesJSON = "[]"
	}
	skillsJSON := char.SkillsJSON
	if skillsJSON == "" || skillsJSON == "null" {
		skillsJSON = "{}"
	}

	systemPrompt := prompts.CraftingSystem(
		story.Name,
		story.SettingJSON,
		char.Name,
		inventoryJSON,
		knownRecipesJSON,
		skillsJSON,
	)

	// Build messages: system context + recent chat history.
	messages := []ai.Message{
		{Role: "user", Content: systemPrompt},
	}
	messages = append(messages, ce.chatHistory...)

	start := time.Now()
	req := ai.Request{
		Messages:    messages,
		Temperature: 0.80,
		MaxTokens:   1024,
	}

	resp, err := ce.narrator.router.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI crafting response: %w", err)
	}
	latency := time.Since(start).Milliseconds()

	// Parse crafting response from AI.
	craftResp, parseErr := parseCraftingResponse(resp.Content)
	if parseErr != nil {
		// Fallback: treat as infeasible with raw narrative.
		craftResp = &CraftingResponse{
			Feasible:  false,
			Narrative: resp.Content,
			Choices: []Choice{
				{ID: 1, Text: "Try something else"},
				{ID: 2, Text: "Leave crafting"},
			},
		}
	}

	// If feasible and item was created, apply state changes.
	if craftResp.Feasible && craftResp.Item != nil {
		craftResp.Item.CraftedAt = time.Now().Format(time.RFC3339)

		// Remove consumed materials from inventory.
		if len(craftResp.Item.Materials) > 0 {
			changes := map[string]interface{}{
				"inventory_remove": toInterfaceSliceFromStrings(craftResp.Item.Materials),
				"inventory_add":    []interface{}{craftResp.Item.Name},
			}
			_, _ = ApplyStateChanges(changes, char, ce.narrator.world, ce.narrator.db, story.ID, ce.narrator.session.Turn())
			// Persist updated character.
			_ = ce.narrator.db.UpdateCharacterFull(char)
		}

		// Save recipe to known_recipes.
		if err := ce.SaveRecipe(craftResp.Item); err != nil {
			// Non-fatal: recipe saving failure doesn't break crafting.
			_ = err
		}
	}

	// Append AI response to chat history.
	ce.chatHistory = append(ce.chatHistory, ai.Message{
		Role:    "assistant",
		Content: resp.Content,
	})

	ce.turnCount++

	// Log to sub-session JSONL.
	entry := ChatEntry{
		Timestamp:   time.Now(),
		MessageType: "crafting",
		Input: &ChatInput{
			Type: "crafting_request",
			Text: message,
		},
		Output: &ChatOutput{
			Narrative: craftResp.Narrative,
			Mood:      "focused",
		},
		AIModel:   resp.Model,
		AILatency: latency,
	}
	_ = ce.session.AppendSubTurn(ce.subSessionID, entry)

	return craftResp, nil
}

// Close ends the crafting session.
func (ce *CraftingEngine) Close() error {
	return ce.session.CloseSubSession(ce.subSessionID)
}

// SaveRecipe persists a discovered recipe to the character's known_recipes.
func (ce *CraftingEngine) SaveRecipe(item *CraftedItem) error {
	char := ce.narrator.character
	recipes, err := GetKnownRecipes(char)
	if err != nil {
		recipes = []CraftedItem{}
	}

	// Check for duplicate (same item name, case-insensitive).
	for _, r := range recipes {
		if strings.EqualFold(r.Name, item.Name) {
			return nil // already known
		}
	}

	recipes = append(recipes, *item)

	recipesBytes, err := json.Marshal(recipes)
	if err != nil {
		return fmt.Errorf("marshaling recipes: %w", err)
	}
	char.KnownRecipesJSON = string(recipesBytes)
	return ce.narrator.db.UpdateCharacterFull(char)
}

// GetKnownRecipes returns the player's discovered recipes.
func GetKnownRecipes(char *storage.Character) ([]CraftedItem, error) {
	if char.KnownRecipesJSON == "" || char.KnownRecipesJSON == "null" {
		return []CraftedItem{}, nil
	}
	var recipes []CraftedItem
	if err := json.Unmarshal([]byte(char.KnownRecipesJSON), &recipes); err != nil {
		return nil, fmt.Errorf("parsing known recipes: %w", err)
	}
	return recipes, nil
}

// parseCraftingResponse extracts the JSON block from an AI crafting response.
func parseCraftingResponse(text string) (*CraftingResponse, error) {
	matches := narratorJSONRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no JSON block found in crafting response")
	}

	var cr CraftingResponse
	if err := json.Unmarshal([]byte(matches[1]), &cr); err != nil {
		return nil, fmt.Errorf("unmarshaling crafting JSON: %w", err)
	}

	if cr.Narrative == "" {
		return nil, fmt.Errorf("empty narrative in crafting response")
	}

	return &cr, nil
}

// buildInventoryContext builds a JSON string of the player's current inventory
// suitable for inclusion in the crafting system prompt.
func buildInventoryContext(char *storage.Character) string {
	if char.StatsJSON == "" {
		return "{}"
	}

	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(char.StatsJSON), &stats); err != nil {
		return "{}"
	}

	inv, ok := stats["inventory"].(map[string]interface{})
	if !ok {
		// Try InventoryJSON field.
		if char.InventoryJSON != "" && char.InventoryJSON != "null" {
			return char.InventoryJSON
		}
		return "{}"
	}

	b, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

// toInterfaceSliceFromStrings converts []string to []interface{} for state changes.
func toInterfaceSliceFromStrings(ss []string) []interface{} {
	result := make([]interface{}, len(ss))
	for i, s := range ss {
		result[i] = s
	}
	return result
}
