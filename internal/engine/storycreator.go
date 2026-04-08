package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/storage"
)

// StoryCreationPhase tracks where we are in the creation flow.
type StoryCreationPhase int

const (
	PhaseConversation StoryCreationPhase = iota // AI-guided story building
	PhaseCharacter                              // Name and background
	PhaseDone                                   // Creation complete
)

// StoryCreator manages the AI-guided story creation conversation.
type StoryCreator struct {
	router      *ai.Router
	db          *storage.DB
	genCfg      config.GenerationConfig
	phase       StoryCreationPhase
	messages    []ai.Message     // conversation history
	charMsgs    []ai.Message     // character creation history
	definition  *StoryDefinition // parsed after AI confirms
	story       *storage.Story
	character   *storage.Character
	lastModel   string
	lastLatency int64
}

// NewStoryCreator initializes the story creation flow.
func NewStoryCreator(router *ai.Router, db *storage.DB, genCfg config.GenerationConfig) *StoryCreator {
	if genCfg.Temperature == 0 {
		genCfg.Temperature = 0.9
	}
	if genCfg.MaxTokens == 0 {
		genCfg.MaxTokens = 2048
	}
	return &StoryCreator{
		router: router,
		db:     db,
		genCfg: genCfg,
		phase:  PhaseConversation,
		messages: []ai.Message{
			{Role: ai.RoleSystem, Content: prompts.StoryCreationSystem},
		},
		charMsgs: []ai.Message{
			{Role: ai.RoleSystem, Content: prompts.CharacterCreationSystem},
		},
	}
}

// Phase returns the current creation phase.
func (sc *StoryCreator) Phase() StoryCreationPhase { return sc.phase }

// LastModel returns the model name from the last AI call.
func (sc *StoryCreator) LastModel() string { return sc.lastModel }

// LastLatency returns the latency in ms from the last AI call.
func (sc *StoryCreator) LastLatency() int64 { return sc.lastLatency }

// Story returns the created story (nil until PhaseDone).
func (sc *StoryCreator) Story() *storage.Story { return sc.story }

// Character returns the created character (nil until PhaseDone).
func (sc *StoryCreator) Character() *storage.Character { return sc.character }

// Definition returns the parsed story definition (nil until PhaseCharacter).
func (sc *StoryCreator) Definition() *StoryDefinition { return sc.definition }

// SendMessage sends a player message and gets the AI response.
// Returns the AI's text response.
func (sc *StoryCreator) SendMessage(ctx context.Context, playerInput string) (string, error) {
	switch sc.phase {
	case PhaseConversation:
		return sc.handleConversation(ctx, playerInput)
	case PhaseCharacter:
		return sc.handleCharacter(ctx, playerInput)
	default:
		return "", fmt.Errorf("story creation already complete")
	}
}

// StartConversation returns the hardcoded welcome message to start story creation instantly.
// No API call needed — the first AI call happens when the player responds.
func (sc *StoryCreator) StartConversation(ctx context.Context) (string, error) {
	welcome := `# Welcome to Story Creation!

Let's build your world together. I'll guide you through the process step by step.

**First, tell me about the kind of story you want:**

- What **genre** appeals to you? *(fantasy, sci-fi, cyberpunk, post-apocalyptic, noir, horror, historical, slice-of-life, or something else entirely)*
- What **tone** should it have? *(dark & gritty, epic & heroic, lighthearted, mysterious, comedic, philosophical...)*
- Any **specific themes** or ideas you already have in mind?

Feel free to be as vague or detailed as you want — we'll shape everything together.`

	// Add the welcome as an assistant message to history so the AI has context
	sc.messages = append(sc.messages, ai.Message{
		Role:    ai.RoleAssistant,
		Content: welcome,
	})
	sc.lastModel = "system"
	sc.lastLatency = 0
	return welcome, nil
}

func (sc *StoryCreator) handleConversation(ctx context.Context, input string) (string, error) {
	sc.messages = append(sc.messages, ai.Message{
		Role:    ai.RoleUser,
		Content: input,
	})

	resp, err := sc.callAI(ctx, sc.messages)
	if err != nil {
		// Remove the failed message so player can retry
		sc.messages = sc.messages[:len(sc.messages)-1]
		return "", err
	}

	// Check if the response contains the final JSON
	if def := extractStoryJSON(resp); def != nil {
		sc.definition = def
		sc.phase = PhaseCharacter
		// Start character creation phase
		sc.charMsgs = append(sc.charMsgs, ai.Message{
			Role:    ai.RoleUser,
			Content: fmt.Sprintf("The story \"%s\" has been created. Now I need to create my character.", def.Name),
		})
		charResp, err := sc.callAI(ctx, sc.charMsgs)
		if err != nil {
			return resp + "\n\n[Story created! Now let's create your character. What's your protagonist's name?]", nil
		}
		return resp + "\n\n---\n\n" + charResp, nil
	}

	return resp, nil
}

func (sc *StoryCreator) handleCharacter(ctx context.Context, input string) (string, error) {
	sc.charMsgs = append(sc.charMsgs, ai.Message{
		Role:    ai.RoleUser,
		Content: input,
	})

	resp, err := sc.callAI(ctx, sc.charMsgs)
	if err != nil {
		sc.charMsgs = sc.charMsgs[:len(sc.charMsgs)-1]
		return "", err
	}

	// Check if character JSON is in the response
	if name, bg := extractCharacterJSON(resp); name != "" {
		if err := sc.persistStory(name, bg); err != nil {
			return "", fmt.Errorf("saving story: %w", err)
		}
		sc.phase = PhaseDone
	}

	return resp, nil
}

func (sc *StoryCreator) callAI(ctx context.Context, msgs []ai.Message) (string, error) {
	req := ai.Request{
		Messages:    msgs,
		Temperature: sc.genCfg.Temperature,
		MaxTokens:   sc.genCfg.MaxTokens,
	}
	if sc.phase == PhaseCharacter {
		req.ResponseFormat = ai.CharacterCreationResponseFormat()
	}
	resp, err := sc.router.Complete(ctx, req)
	if err != nil {
		return "", err
	}
	sc.lastModel = resp.Model
	sc.lastLatency = resp.LatencyMs

	// Append assistant response to the active message list
	if sc.phase == PhaseCharacter {
		sc.charMsgs = append(sc.charMsgs, ai.Message{
			Role:    ai.RoleAssistant,
			Content: resp.Content,
		})
	} else {
		sc.messages = append(sc.messages, ai.Message{
			Role:    ai.RoleAssistant,
			Content: resp.Content,
		})
	}

	return resp.Content, nil
}

func (sc *StoryCreator) persistStory(charName, charBackground string) error {
	if sc.definition == nil {
		return fmt.Errorf("no story definition to persist")
	}

	storyID := uuid.New().String()
	charID := uuid.New().String()
	now := time.Now()

	settingJSON, _ := json.Marshal(sc.definition.Setting)
	schemaJSON, _ := json.Marshal(sc.definition.StatsSchema)
	initialStats := sc.definition.StatsSchema.InitialStats()
	statsJSON, _ := json.Marshal(initialStats)

	sc.story = &storage.Story{
		ID:              storyID,
		Name:            sc.definition.Name,
		SettingJSON:     string(settingJSON),
		StatsSchemaJSON: string(schemaJSON),
		Description:     sc.definition.Description,
		Genre:           sc.definition.Genre,
		Tone:            sc.definition.Tone,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	sc.character = &storage.Character{
		ID:               charID,
		StoryID:          storyID,
		Name:             charName,
		Background:       charBackground,
		StatsJSON:        string(statsJSON),
		TraitsJSON:       "[]",
		SkillsJSON:       "[]",
		InventoryJSON:    "[]",
		KnownRecipesJSON: "[]",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Persist to DB
	if err := sc.db.CreateStory(sc.story); err != nil {
		return fmt.Errorf("creating story: %w", err)
	}
	if err := sc.db.CreateCharacter(sc.character); err != nil {
		return fmt.Errorf("creating character: %w", err)
	}

	// Create initial world state
	worldState := &storage.WorldState{
		ID:                   uuid.New().String(),
		StoryID:              storyID,
		CurrentLocation:      "",
		KnownLocationsJSON:   "[]",
		GlobalEventsJSON:     "[]",
		FactionStandingsJSON: "{}",
		CurrentChapter:       1,
		CurrentTurn:          0,
		UpdatedAt:            now,
	}
	if err := sc.db.CreateWorldState(worldState); err != nil {
		return fmt.Errorf("creating world state: %w", err)
	}

	return nil
}

func extractStoryJSON(text string) *StoryDefinition {
	raw, err := ai.ExtractJSONPayload(text)
	if err != nil || raw == "" {
		return nil
	}
	var def StoryDefinition
	if err := json.Unmarshal([]byte(raw), &def); err != nil {
		return nil
	}
	// Basic validation
	if def.Name == "" || def.Genre == "" || len(def.Setting.Rules) == 0 {
		return nil
	}
	return &def
}

type charJSON struct {
	Name       string `json:"name"`
	Background string `json:"background"`
}

func extractCharacterJSON(text string) (string, string) {
	raw, err := ai.ExtractJSONPayload(text)
	if err != nil || raw == "" {
		return "", ""
	}
	var c charJSON
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return "", ""
	}
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return "", ""
	}
	return c.Name, c.Background
}
