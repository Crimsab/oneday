package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
	PhaseConversation StoryCreationPhase = iota // guided story building
	PhaseCharacter                              // local character setup
	PhaseDone                                   // creation complete
)

type storyCreationStage int

const (
	stageBrief storyCreationStage = iota
	stageReviewWorld
	stageReviewRules
	stageReviewStats
	stageConfirm
	stageCharacterName
	stageCharacterBackground
	stageDone
)

// CreationAction is a quick action shown in the story creation wizard.
type CreationAction struct {
	Key   string
	Label string
}

// StoryCreator manages the guided story creation wizard.
type StoryCreator struct {
	router       *ai.Router
	db           *storage.DB
	genCfg       config.GenerationConfig
	stage        storyCreationStage
	definition   *StoryDefinition
	story        *storage.Story
	character    *storage.Character
	lastModel    string
	lastLatency  int64
	initialBrief string
	charName     string
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
		stage:  stageBrief,
	}
}

// Phase returns the current creation phase.
func (sc *StoryCreator) Phase() StoryCreationPhase {
	switch sc.stage {
	case stageCharacterName, stageCharacterBackground:
		return PhaseCharacter
	case stageDone:
		return PhaseDone
	default:
		return PhaseConversation
	}
}

// StageLabel returns a compact label for the current wizard step.
func (sc *StoryCreator) StageLabel() string {
	switch sc.stage {
	case stageBrief:
		return "Choose the story brief"
	case stageReviewWorld:
		return "Review world draft"
	case stageReviewRules:
		return "Review rules and factions"
	case stageReviewStats:
		return "Review stats schema"
	case stageConfirm:
		return "Confirm story"
	case stageCharacterName:
		return "Name your protagonist"
	case stageCharacterBackground:
		return "Add a background"
	default:
		return "Finalizing story"
	}
}

// InputPlaceholder returns the most useful textarea hint for the current step.
func (sc *StoryCreator) InputPlaceholder() string {
	switch sc.stage {
	case stageBrief:
		return "Describe the story you want: genre, tone, language, style, extra direction..."
	case stageReviewWorld:
		return "Type how to change the world draft..."
	case stageReviewRules:
		return "Type how to change rules, factions, cultures, or dangers..."
	case stageReviewStats:
		return "Type how to change the stats schema..."
	case stageConfirm:
		return "Type final adjustments before creating the story..."
	case stageCharacterName:
		return "Type your protagonist's name..."
	case stageCharacterBackground:
		return "Type a short background, or use the quick choice to skip..."
	default:
		return "Type your response..."
	}
}

// Actions returns quick actions for the current wizard step.
func (sc *StoryCreator) Actions() []CreationAction {
	switch sc.stage {
	case stageBrief:
		return []CreationAction{
			{Key: "preset_dark_fantasy", Label: "Dark fantasy"},
			{Key: "preset_cyberpunk", Label: "Cyberpunk noir"},
			{Key: "preset_horror", Label: "Horror mystery"},
			{Key: "preset_cozy", Label: "Cozy slice-of-life"},
			{Key: "focus_input", Label: "Write my own"},
		}
	case stageReviewWorld:
		return []CreationAction{
			{Key: "accept_world", Label: "Accept world"},
			{Key: "regenerate_world", Label: "Regenerate world"},
			{Key: "make_darker", Label: "Make darker"},
			{Key: "make_lighter", Label: "Make lighter"},
			{Key: "edit_world", Label: "Edit manually"},
		}
	case stageReviewRules:
		return []CreationAction{
			{Key: "accept_rules", Label: "Accept rules"},
			{Key: "more_danger", Label: "More danger"},
			{Key: "fewer_factions", Label: "Fewer factions"},
			{Key: "regenerate_rules", Label: "Regenerate section"},
			{Key: "edit_rules", Label: "Edit manually"},
		}
	case stageReviewStats:
		return []CreationAction{
			{Key: "accept_stats", Label: "Accept stats"},
			{Key: "lighter_stats", Label: "Lighter rules"},
			{Key: "crunchier_stats", Label: "More crunchy"},
			{Key: "no_combat", Label: "No combat"},
			{Key: "edit_stats", Label: "Edit manually"},
		}
	case stageConfirm:
		return []CreationAction{
			{Key: "create_story", Label: "Create story"},
			{Key: "regenerate_all", Label: "Regenerate all"},
			{Key: "edit_final", Label: "Edit final details"},
		}
	case stageCharacterBackground:
		return []CreationAction{
			{Key: "skip_background", Label: "Skip background"},
		}
	default:
		return nil
	}
}

// LastModel returns the model name from the last AI call.
func (sc *StoryCreator) LastModel() string { return sc.lastModel }

// LastLatency returns the latency in ms from the last AI call.
func (sc *StoryCreator) LastLatency() int64 { return sc.lastLatency }

// Story returns the created story (nil until PhaseDone).
func (sc *StoryCreator) Story() *storage.Story { return sc.story }

// Character returns the created character (nil until PhaseDone).
func (sc *StoryCreator) Character() *storage.Character { return sc.character }

// Definition returns the current draft story definition.
func (sc *StoryCreator) Definition() *StoryDefinition { return sc.definition }

// StartConversation returns the local wizard intro instantly.
func (sc *StoryCreator) StartConversation(ctx context.Context) (string, error) {
	sc.lastModel = "system"
	sc.lastLatency = 0
	return `Story setup starts with one short brief.

Tell me the kind of story you want and include, if relevant:
- genre
- tone
- language
- prose style
- extra direction for every future prompt

Example:
"Italian cyberpunk noir, fast dialogue, a little darkly comic, avoid purple prose."

You can also use one of the quick choices below.`, nil
}

// SendMessage handles free-text input for the current wizard stage.
func (sc *StoryCreator) SendMessage(ctx context.Context, playerInput string) (string, error) {
	input := strings.TrimSpace(playerInput)
	if input == "" {
		return "", fmt.Errorf("empty input")
	}

	switch sc.stage {
	case stageBrief:
		return sc.handleBrief(ctx, input)
	case stageReviewWorld:
		return sc.handleRevision(ctx, "world", input)
	case stageReviewRules:
		return sc.handleRevision(ctx, "rules", input)
	case stageReviewStats:
		return sc.handleRevision(ctx, "stats", input)
	case stageConfirm:
		return sc.handleRevision(ctx, "final", input)
	case stageCharacterName:
		return sc.handleCharacterName(input)
	case stageCharacterBackground:
		return sc.handleCharacterBackground(input)
	default:
		return "", fmt.Errorf("story creation already complete")
	}
}

// ExecuteAction handles a quick choice for the current wizard stage.
func (sc *StoryCreator) ExecuteAction(ctx context.Context, actionKey string) (string, error) {
	switch actionKey {
	case "focus_input":
		sc.lastModel = "system"
		sc.lastLatency = 0
		return "Write your own brief in the input box. A single compact paragraph is enough.", nil
	case "preset_dark_fantasy":
		return sc.handleBrief(ctx, "Italian dark fantasy with melancholy ruins, dangerous magic, elegant prose, and terse dialogue.")
	case "preset_cyberpunk":
		return sc.handleBrief(ctx, "Italian cyberpunk noir with sharp dialogue, neon decay, corporate power, and a slightly darkly comic edge.")
	case "preset_horror":
		return sc.handleBrief(ctx, "Italian horror mystery with oppressive atmosphere, slow dread, and clear, controlled prose.")
	case "preset_cozy":
		return sc.handleBrief(ctx, "Italian cozy slice-of-life fantasy with gentle humor, warm relationships, and light, vivid prose.")
	}

	switch sc.stage {
	case stageReviewWorld:
		return sc.executeWorldAction(ctx, actionKey)
	case stageReviewRules:
		return sc.executeRulesAction(ctx, actionKey)
	case stageReviewStats:
		return sc.executeStatsAction(ctx, actionKey)
	case stageConfirm:
		return sc.executeConfirmAction(ctx, actionKey)
	case stageCharacterBackground:
		if actionKey == "skip_background" {
			return sc.handleCharacterBackground("")
		}
	}

	return "", fmt.Errorf("unsupported action: %s", actionKey)
}

func (sc *StoryCreator) handleBrief(ctx context.Context, input string) (string, error) {
	sc.initialBrief = input

	def, err := sc.generateDraft(ctx, input)
	if err != nil {
		return "", err
	}

	sc.definition = def
	sc.stage = stageReviewWorld
	return sc.renderCurrentStage(), nil
}

func (sc *StoryCreator) handleRevision(ctx context.Context, section, input string) (string, error) {
	if sc.definition == nil {
		return "", fmt.Errorf("no story draft to revise")
	}

	def, err := sc.reviseDraft(ctx, section, input)
	if err != nil {
		return "", err
	}

	sc.definition = def
	return sc.renderCurrentStage(), nil
}

func (sc *StoryCreator) handleCharacterName(input string) (string, error) {
	name := strings.TrimSpace(input)
	if name == "" {
		return "", fmt.Errorf("character name is required")
	}

	sc.charName = name
	sc.lastModel = "system"
	sc.lastLatency = 0
	sc.stage = stageCharacterBackground
	return fmt.Sprintf("Protagonist name locked in: %s\n\nNow add a brief background, or use the quick choice below to skip it.", sc.charName), nil
}

func (sc *StoryCreator) handleCharacterBackground(input string) (string, error) {
	if sc.charName == "" {
		return "", fmt.Errorf("character name is required before background")
	}

	if err := sc.persistStory(sc.charName, strings.TrimSpace(input)); err != nil {
		return "", fmt.Errorf("saving story: %w", err)
	}

	sc.stage = stageDone
	sc.lastModel = "system"
	sc.lastLatency = 0
	return fmt.Sprintf("Story created. %s is ready. Starting the adventure...", sc.charName), nil
}

func (sc *StoryCreator) executeWorldAction(ctx context.Context, actionKey string) (string, error) {
	switch actionKey {
	case "accept_world":
		sc.stage = stageReviewRules
		sc.lastModel = "system"
		sc.lastLatency = 0
		return sc.renderCurrentStage(), nil
	case "regenerate_world":
		return sc.handleRevision(ctx, "world", "Regenerate the world section. Keep the same core concept, language, style, and overall promise, but propose a clearly different world setup.")
	case "make_darker":
		return sc.handleRevision(ctx, "world", "Keep the same core concept, but make the world harsher, darker, and more dangerous.")
	case "make_lighter":
		return sc.handleRevision(ctx, "world", "Keep the same core concept, but make the world lighter, more adventurous, and less oppressive.")
	case "edit_world":
		sc.lastModel = "system"
		sc.lastLatency = 0
		return "Type the world changes you want in the input box, then press Enter.", nil
	default:
		return "", fmt.Errorf("unsupported world action: %s", actionKey)
	}
}

func (sc *StoryCreator) executeRulesAction(ctx context.Context, actionKey string) (string, error) {
	switch actionKey {
	case "accept_rules":
		sc.stage = stageReviewStats
		sc.lastModel = "system"
		sc.lastLatency = 0
		return sc.renderCurrentStage(), nil
	case "more_danger":
		return sc.handleRevision(ctx, "rules", "Keep the current direction, but make the dangers sharper and the world rules more consequential.")
	case "fewer_factions":
		return sc.handleRevision(ctx, "rules", "Reduce faction sprawl. Keep only the strongest factions and make them more distinct.")
	case "regenerate_rules":
		return sc.handleRevision(ctx, "rules", "Regenerate rules, factions, cultures, and dangers while preserving the world identity.")
	case "edit_rules":
		sc.lastModel = "system"
		sc.lastLatency = 0
		return "Type the rules, factions, cultures, or danger changes you want in the input box, then press Enter.", nil
	default:
		return "", fmt.Errorf("unsupported rules action: %s", actionKey)
	}
}

func (sc *StoryCreator) executeStatsAction(ctx context.Context, actionKey string) (string, error) {
	switch actionKey {
	case "accept_stats":
		sc.stage = stageConfirm
		sc.lastModel = "system"
		sc.lastLatency = 0
		return sc.renderCurrentStage(), nil
	case "lighter_stats":
		return sc.handleRevision(ctx, "stats", "Keep the same story identity, but make the stats schema lighter, simpler, and more narrative-first.")
	case "crunchier_stats":
		return sc.handleRevision(ctx, "stats", "Keep the same story identity, but make the stats schema more tactical, crunchy, and game-like.")
	case "no_combat":
		return sc.handleRevision(ctx, "stats", "Disable combat. Build a schema centered on narrative tension, social pressure, travel, investigation, or survival instead.")
	case "edit_stats":
		sc.lastModel = "system"
		sc.lastLatency = 0
		return "Type the stat changes you want in the input box, then press Enter.", nil
	default:
		return "", fmt.Errorf("unsupported stats action: %s", actionKey)
	}
}

func (sc *StoryCreator) executeConfirmAction(ctx context.Context, actionKey string) (string, error) {
	switch actionKey {
	case "create_story":
		sc.stage = stageCharacterName
		sc.lastModel = "system"
		sc.lastLatency = 0
		return "Story locked in.\n\nNow type your protagonist's name.", nil
	case "regenerate_all":
		if strings.TrimSpace(sc.initialBrief) == "" {
			return "", fmt.Errorf("missing initial brief")
		}
		return sc.handleBrief(ctx, sc.initialBrief)
	case "edit_final":
		sc.lastModel = "system"
		sc.lastLatency = 0
		return "Type any final story changes in the input box, then press Enter.", nil
	default:
		return "", fmt.Errorf("unsupported confirm action: %s", actionKey)
	}
}

func (sc *StoryCreator) generateDraft(ctx context.Context, brief string) (*StoryDefinition, error) {
	return sc.requestStoryDefinition(ctx, []ai.Message{
		{Role: ai.RoleSystem, Content: prompts.StoryDefinitionSystemPrompt()},
		{Role: ai.RoleUser, Content: prompts.StoryDefinitionUserPrompt(brief)},
	})
}

func (sc *StoryCreator) reviseDraft(ctx context.Context, section, feedback string) (*StoryDefinition, error) {
	if sc.definition == nil {
		return nil, fmt.Errorf("no story draft to revise")
	}

	draftJSON, err := json.MarshalIndent(sc.definition, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling story draft: %w", err)
	}

	return sc.requestStoryDefinition(ctx, []ai.Message{
		{Role: ai.RoleSystem, Content: prompts.StoryRevisionSystemPrompt()},
		{Role: ai.RoleUser, Content: prompts.StoryRevisionUserPrompt(section, string(draftJSON), feedback)},
	})
}

func (sc *StoryCreator) requestStoryDefinition(ctx context.Context, msgs []ai.Message) (*StoryDefinition, error) {
	req := ai.Request{
		Messages:       msgs,
		Temperature:    sc.genCfg.Temperature,
		MaxTokens:      sc.genCfg.MaxTokens,
		ResponseFormat: ai.StoryDefinitionResponseFormat(),
	}

	repairFormats := []*ai.ResponseFormat{
		ai.StoryDefinitionResponseFormat(),
		ai.NewJSONObjectResponseFormat(),
	}

	start := time.Now()
	resp, err := sc.router.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	sc.lastModel = resp.Model
	sc.lastLatency = time.Since(start).Milliseconds()

	def, parseErr := parseStoryDefinitionWithFallback(resp.Content, sc.initialBrief, sc.definition)
	if parseErr == nil {
		return def, nil
	}

	lastErr := error(parseErr)
	previousDraftJSON := ""
	if sc.definition != nil {
		if data, err := json.MarshalIndent(sc.definition, "", "  "); err == nil {
			previousDraftJSON = string(data)
		}
	}

	for attempt := 0; attempt < len(repairFormats); attempt++ {
		repairReq := ai.Request{
			Messages: []ai.Message{
				{Role: ai.RoleSystem, Content: prompts.StoryRepairSystemPrompt()},
				{Role: ai.RoleUser, Content: prompts.StoryRepairUserPrompt(resp.Content, lastErr.Error(), sc.initialBrief, previousDraftJSON)},
			},
			Temperature:    0.2,
			MaxTokens:      sc.genCfg.MaxTokens,
			ResponseFormat: repairFormats[attempt],
		}
		def, repairResp, repairErr := sc.runRepairModels(ctx, repairReq)
		if repairErr == nil {
			sc.lastModel = repairResp.Model
			sc.lastLatency += repairResp.LatencyMs
			return def, nil
		}
		lastErr = repairErr
	}

	return nil, fmt.Errorf("invalid story definition returned by AI: %w", lastErr)
}

func (sc *StoryCreator) runRepairModels(ctx context.Context, req ai.Request) (*StoryDefinition, ai.Response, error) {
	candidates := sc.genCfg.RepairModelCandidates()
	if len(candidates) == 0 {
		candidates = []string{""}
	}

	var errs []string
	for _, model := range candidates {
		candidateReq := req
		candidateReq.Model = model

		start := time.Now()
		resp, err := sc.router.Complete(ctx, candidateReq)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			label := strings.TrimSpace(model)
			if label == "" {
				label = "provider-default"
			}
			errs = append(errs, fmt.Sprintf("%s: %v", label, err))
			continue
		}
		resp.LatencyMs = latency

		def, parseErr := parseStoryDefinitionWithFallback(resp.Content, sc.initialBrief, sc.definition)
		if parseErr == nil {
			return def, resp, nil
		}

		label := resp.Model
		if strings.TrimSpace(label) == "" {
			label = strings.TrimSpace(model)
		}
		if label == "" {
			label = "provider-default"
		}
		errs = append(errs, fmt.Sprintf("%s: %v", label, parseErr))
	}

	return nil, ai.Response{}, fmt.Errorf("repair models failed: %s", strings.Join(errs, " | "))
}

func (sc *StoryCreator) renderCurrentStage() string {
	if sc.definition == nil {
		return ""
	}

	switch sc.stage {
	case stageReviewWorld:
		return sc.renderWorldReview()
	case stageReviewRules:
		return sc.renderRulesReview()
	case stageReviewStats:
		return sc.renderStatsReview()
	case stageConfirm:
		return sc.renderConfirmReview()
	default:
		return ""
	}
}

func (sc *StoryCreator) renderWorldReview() string {
	def := sc.definition

	lines := []string{
		"World Draft",
		"",
		fmt.Sprintf("Story: %s", def.Name),
		fmt.Sprintf("Genre: %s", def.Genre),
		fmt.Sprintf("Tone: %s", def.Tone),
		fmt.Sprintf("Language: %s", emptyFallback(def.Language, "inferred by AI")),
		fmt.Sprintf("Writing style: %s", emptyFallback(def.WritingStyle, "default")),
		fmt.Sprintf("Extra directives: %s", emptyFallback(def.PromptDirectives, "none")),
		"",
		fmt.Sprintf("World: %s", def.Setting.WorldName),
		fmt.Sprintf("Era: %s", def.Setting.Era),
		fmt.Sprintf("Geography: %s", def.Setting.Geography),
		fmt.Sprintf("Magic / unusual system: %s", def.Setting.MagicSystem),
		fmt.Sprintf("Technology level: %s", def.Setting.TechnologyLevel),
		fmt.Sprintf("Society: %s", def.Setting.Society),
		"",
		"Description:",
		def.Description,
		"",
		"Use a quick choice below or type changes manually.",
	}

	return strings.Join(lines, "\n")
}

func (sc *StoryCreator) renderRulesReview() string {
	def := sc.definition

	lines := []string{
		"Rules, Factions, And Dangers",
		"",
		"World rules:",
		formatBulletedList(def.Setting.Rules),
		"",
		"Factions:",
		formatBulletedList(def.Setting.Factions),
		"",
		"Cultures:",
		formatBulletedList(def.Setting.Cultures),
		"",
		"Dangers:",
		formatBulletedList(def.Setting.Dangers),
		"",
		"Use a quick choice below or type changes manually.",
	}

	return strings.Join(lines, "\n")
}

func (sc *StoryCreator) renderStatsReview() string {
	def := sc.definition
	combat := "No"
	if def.StatsSchema.HasCombat {
		combat = "Yes"
	}

	lines := []string{
		"Stats Schema",
		"",
		fmt.Sprintf("Combat enabled: %s", combat),
		"",
		"Vitals:",
		formatStatDefs(def.StatsSchema.Vitals),
		"",
		"Attributes:",
		formatStatDefs(def.StatsSchema.Attributes),
		"",
		"Secondary stats:",
		formatStatDefs(def.StatsSchema.Secondary),
		"",
		fmt.Sprintf("Currency: %s", formatCurrency(def.StatsSchema.Currency)),
		"",
		"Use a quick choice below or type changes manually.",
	}

	return strings.Join(lines, "\n")
}

func (sc *StoryCreator) renderConfirmReview() string {
	def := sc.definition

	lines := []string{
		"Final Story Draft",
		"",
		fmt.Sprintf("Story: %s", def.Name),
		fmt.Sprintf("Genre / Tone: %s / %s", def.Genre, def.Tone),
		fmt.Sprintf("Language / Style: %s / %s", def.Language, def.WritingStyle),
		fmt.Sprintf("World: %s", def.Setting.WorldName),
		fmt.Sprintf("Combat: %s", yesNo(def.StatsSchema.HasCombat)),
		"",
		"Description:",
		def.Description,
		"",
		"Quick review:",
		fmt.Sprintf("- %d world rules", len(def.Setting.Rules)),
		fmt.Sprintf("- %d factions", len(def.Setting.Factions)),
		fmt.Sprintf("- %d cultures", len(def.Setting.Cultures)),
		fmt.Sprintf("- %d dangers", len(def.Setting.Dangers)),
		fmt.Sprintf("- %d vitals, %d attributes, %d secondary stats",
			len(def.StatsSchema.Vitals),
			len(def.StatsSchema.Attributes),
			len(def.StatsSchema.Secondary),
		),
		"",
		"Use a quick choice below or type final changes manually.",
	}

	return strings.Join(lines, "\n")
}

func formatBulletedList(items []string) string {
	if len(items) == 0 {
		return "- none"
	}
	var lines []string
	for _, item := range items {
		lines = append(lines, "- "+item)
	}
	return strings.Join(lines, "\n")
}

func formatStatDefs(defs []StatDef) string {
	if len(defs) == 0 {
		return "- none"
	}
	var lines []string
	for _, def := range defs {
		line := fmt.Sprintf("- %s (%s)", def.Label, def.Key)
		if def.Starting != 0 {
			line += fmt.Sprintf(" start %d", def.Starting)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func formatCurrency(currency *CurrencyDef) string {
	if currency == nil {
		return "none"
	}
	return fmt.Sprintf("%s (start %d)", currency.Name, currency.Starting)
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
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
		ID:               storyID,
		Name:             sc.definition.Name,
		SettingJSON:      string(settingJSON),
		StatsSchemaJSON:  string(schemaJSON),
		Description:      sc.definition.Description,
		Genre:            sc.definition.Genre,
		Tone:             sc.definition.Tone,
		Language:         sc.definition.Language,
		WritingStyle:     sc.definition.WritingStyle,
		PromptDirectives: sc.definition.PromptDirectives,
		CreatedAt:        now,
		UpdatedAt:        now,
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

	if err := sc.db.CreateStory(sc.story); err != nil {
		return fmt.Errorf("creating story: %w", err)
	}
	if err := sc.db.CreateCharacter(sc.character); err != nil {
		return fmt.Errorf("creating character: %w", err)
	}

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
	def, err := parseStoryDefinition(text)
	if err != nil {
		return nil
	}
	return def
}

func parseStoryDefinition(text string) (*StoryDefinition, error) {
	raw, err := ai.ExtractJSONPayload(text)
	if err != nil {
		return nil, fmt.Errorf("extracting JSON payload: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("empty JSON payload")
	}
	var def StoryDefinition
	if err := decodeStoryDefinitionJSON(raw, &def); err != nil {
		return nil, err
	}
	if err := validateStoryDefinition(&def); err != nil {
		return nil, err
	}
	return &def, nil
}

func parseStoryDefinitionWithFallback(text, brief string, previous *StoryDefinition) (*StoryDefinition, error) {
	raw, err := ai.ExtractJSONPayload(text)
	if err != nil {
		return nil, fmt.Errorf("extracting JSON payload: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("empty JSON payload")
	}
	var def StoryDefinition
	if err := decodeStoryDefinitionJSON(raw, &def); err != nil {
		return nil, err
	}
	normalizeStoryDefinition(&def, brief, previous)
	if err := validateStoryDefinition(&def); err != nil {
		return nil, err
	}
	return &def, nil
}

func decodeStoryDefinitionJSON(raw string, out *StoryDefinition) error {
	if err := json.Unmarshal([]byte(raw), out); err == nil {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return fmt.Errorf("decoding story definition JSON: %w", err)
	}

	normalized := normalizeLooseStoryDefinitionPayload(payload)
	buf, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("normalizing story definition JSON: %w", err)
	}
	if err := json.Unmarshal(buf, out); err != nil {
		return fmt.Errorf("decoding story definition JSON: %w", err)
	}
	return nil
}

func normalizeLooseStoryDefinitionPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}

	if statsRaw, ok := payload["stats_schema"].(map[string]any); ok {
		statsRaw["vitals"] = coerceStatDefPayload(statsRaw["vitals"])
		statsRaw["attributes"] = coerceStatDefPayload(statsRaw["attributes"])
		statsRaw["secondary"] = coerceStatDefPayload(statsRaw["secondary"])
		if currency := coerceCurrencyPayload(statsRaw["currency"]); currency != nil {
			statsRaw["currency"] = currency
		}
		payload["stats_schema"] = statsRaw
	}
	if settingRaw, ok := payload["setting"].(map[string]any); ok {
		settingRaw["rules"] = coerceStringListPayload(settingRaw["rules"])
		settingRaw["factions"] = coerceStringListPayload(settingRaw["factions"])
		settingRaw["cultures"] = coerceStringListPayload(settingRaw["cultures"])
		settingRaw["dangers"] = coerceStringListPayload(settingRaw["dangers"])
		payload["setting"] = settingRaw
	}

	return payload
}

func coerceStatDefPayload(value any) any {
	switch typed := value.(type) {
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if stat, ok := coerceSingleStatDef("", item); ok {
				out = append(out, stat)
			}
		}
		return out
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			if stat, ok := coerceSingleStatDef(key, typed[key]); ok {
				out = append(out, stat)
			}
		}
		return out
	default:
		return value
	}
}

func coerceSingleStatDef(defaultKey string, value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		key := stringFromAny(typed["key"])
		if key == "" {
			key = defaultKey
		}
		label := stringFromAny(typed["label"])
		if label == "" {
			label = defaultKey
		}
		starting, ok := intFromAny(typed["starting"])
		if !ok {
			starting = 0
		}
		if strings.TrimSpace(key) == "" {
			return nil, false
		}
		if strings.TrimSpace(label) == "" {
			label = key
		}
		return map[string]any{
			"key":      key,
			"label":    label,
			"starting": starting,
		}, true
	case float64, int, int64, json.Number:
		if strings.TrimSpace(defaultKey) == "" {
			return nil, false
		}
		starting, _ := intFromAny(typed)
		return map[string]any{
			"key":      defaultKey,
			"label":    defaultKey,
			"starting": starting,
		}, true
	default:
		return nil, false
	}
}

func coerceCurrencyPayload(value any) any {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if currency := coerceCurrencyPayload(item); currency != nil {
				return currency
			}
		}
		return nil
	case map[string]any:
		name := stringFromAny(typed["name"])
		if name == "" {
			name = "Credits"
		}
		starting, ok := intFromAny(typed["starting"])
		if !ok {
			starting = 0
		}
		return map[string]any{
			"name":     name,
			"starting": starting,
		}
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return map[string]any{
			"name":     typed,
			"starting": 0,
		}
	default:
		return value
	}
}

func coerceStringListPayload(value any) any {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if entry := coerceStringListEntry("", item); entry != "" {
				out = append(out, entry)
			}
		}
		return out
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make([]string, 0, len(keys))
		for _, key := range keys {
			if entry := coerceStringListEntry(key, typed[key]); entry != "" {
				out = append(out, entry)
			}
		}
		return out
	case string:
		if trimmed := strings.TrimSpace(typed); trimmed != "" {
			return []string{trimmed}
		}
		return []string{}
	default:
		return value
	}
}

func coerceStringListEntry(defaultKey string, value any) string {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed != "" {
			return trimmed
		}
	case map[string]any:
		for _, field := range []string{"name", "title", "label"} {
			if text := stringFromAny(typed[field]); text != "" {
				return text
			}
		}
		if trimmed := strings.TrimSpace(defaultKey); trimmed != "" && !looksGenericListKey(trimmed) {
			return trimmed
		}
		if text := stringFromAny(typed["description"]); text != "" {
			return text
		}
	}
	return strings.TrimSpace(defaultKey)
}

func looksGenericListKey(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return true
	}
	allDigits := true
	for _, r := range value {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}
	for _, prefix := range []string{"r", "rule", "item", "entry", "key"} {
		if strings.HasPrefix(value, prefix) {
			suffix := strings.TrimPrefix(value, prefix)
			if suffix == "" {
				continue
			}
			onlyDigits := true
			for _, r := range suffix {
				if r < '0' || r > '9' {
					onlyDigits = false
					break
				}
			}
			if onlyDigits {
				return true
			}
		}
	}
	return false
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		v, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}

func normalizeStoryDefinition(def *StoryDefinition, _ string, previous *StoryDefinition) {
	if def == nil {
		return
	}

	applyStringFallback(&def.Name, previousValue(previous, func(d *StoryDefinition) string { return d.Name }))
	applyStringFallback(&def.Description, previousValue(previous, func(d *StoryDefinition) string { return d.Description }))
	applyStringFallback(&def.Genre, previousValue(previous, func(d *StoryDefinition) string { return d.Genre }))
	applyStringFallback(&def.Tone, previousValue(previous, func(d *StoryDefinition) string { return d.Tone }))
	applyStringFallback(&def.Language, previousValue(previous, func(d *StoryDefinition) string { return d.Language }))
	applyStringFallback(&def.WritingStyle, previousValue(previous, func(d *StoryDefinition) string { return d.WritingStyle }))
	applyStringFallback(&def.PromptDirectives, previousValue(previous, func(d *StoryDefinition) string { return d.PromptDirectives }))
	applyStringFallback(&def.Setting.WorldName, previousValue(previous, func(d *StoryDefinition) string { return d.Setting.WorldName }))
	applyStringFallback(&def.Setting.Era, previousValue(previous, func(d *StoryDefinition) string { return d.Setting.Era }))
	applyStringFallback(&def.Setting.Geography, previousValue(previous, func(d *StoryDefinition) string { return d.Setting.Geography }))
	applyStringFallback(&def.Setting.MagicSystem, previousValue(previous, func(d *StoryDefinition) string { return d.Setting.MagicSystem }))
	applyStringFallback(&def.Setting.TechnologyLevel, previousValue(previous, func(d *StoryDefinition) string { return d.Setting.TechnologyLevel }))
	applyStringFallback(&def.Setting.Society, previousValue(previous, func(d *StoryDefinition) string { return d.Setting.Society }))

	if len(def.Setting.Rules) == 0 && previous != nil {
		def.Setting.Rules = append([]string(nil), previous.Setting.Rules...)
	}
	if len(def.Setting.Factions) == 0 && previous != nil {
		def.Setting.Factions = append([]string(nil), previous.Setting.Factions...)
	}
	if len(def.Setting.Cultures) == 0 && previous != nil {
		def.Setting.Cultures = append([]string(nil), previous.Setting.Cultures...)
	}
	if len(def.Setting.Dangers) == 0 && previous != nil {
		def.Setting.Dangers = append([]string(nil), previous.Setting.Dangers...)
	}
	if len(def.StatsSchema.Vitals) == 0 && previous != nil {
		def.StatsSchema.Vitals = append([]StatDef(nil), previous.StatsSchema.Vitals...)
	}
	if len(def.StatsSchema.Attributes) == 0 && previous != nil {
		def.StatsSchema.Attributes = append([]StatDef(nil), previous.StatsSchema.Attributes...)
	}
	if len(def.StatsSchema.Secondary) == 0 && previous != nil {
		def.StatsSchema.Secondary = append([]StatDef(nil), previous.StatsSchema.Secondary...)
	}
	if def.StatsSchema.Currency == nil && previous != nil && previous.StatsSchema.Currency != nil {
		currency := *previous.StatsSchema.Currency
		def.StatsSchema.Currency = &currency
	}
	if previous != nil && !def.StatsSchema.HasCombat && previous.StatsSchema.HasCombat {
		def.StatsSchema.HasCombat = true
	}
}

func applyStringFallback(target *string, fallback string) {
	if strings.TrimSpace(*target) == "" && strings.TrimSpace(fallback) != "" {
		*target = fallback
	}
}

func previousValue(previous *StoryDefinition, fn func(*StoryDefinition) string) string {
	if previous == nil {
		return ""
	}
	return fn(previous)
}

func validateStoryDefinition(def *StoryDefinition) error {
	if def == nil {
		return errors.New("missing story definition")
	}
	if strings.TrimSpace(def.Name) == "" {
		return errors.New("missing story name")
	}
	if strings.TrimSpace(def.Description) == "" {
		return errors.New("missing story description")
	}
	if strings.TrimSpace(def.Genre) == "" {
		return errors.New("missing genre")
	}
	if strings.TrimSpace(def.Tone) == "" {
		return errors.New("missing tone")
	}
	if strings.TrimSpace(def.Language) == "" {
		return errors.New("missing language")
	}
	if strings.TrimSpace(def.WritingStyle) == "" {
		return errors.New("missing writing_style")
	}
	if strings.TrimSpace(def.Setting.WorldName) == "" {
		return errors.New("missing setting.world_name")
	}
	if len(def.Setting.Rules) == 0 || containsBlankString(def.Setting.Rules) {
		return errors.New("missing setting.rules")
	}
	if len(def.Setting.Factions) == 0 || containsBlankString(def.Setting.Factions) {
		return errors.New("missing setting.factions")
	}
	if len(def.Setting.Cultures) == 0 || containsBlankString(def.Setting.Cultures) {
		return errors.New("missing setting.cultures")
	}
	if len(def.Setting.Dangers) == 0 || containsBlankString(def.Setting.Dangers) {
		return errors.New("missing setting.dangers")
	}
	if len(def.StatsSchema.Vitals) == 0 {
		return errors.New("missing stats_schema.vitals")
	}
	if len(def.StatsSchema.Attributes) == 0 {
		return errors.New("missing stats_schema.attributes")
	}
	if err := validateStatDefs("stats_schema.vitals", def.StatsSchema.Vitals); err != nil {
		return err
	}
	if err := validateStatDefs("stats_schema.attributes", def.StatsSchema.Attributes); err != nil {
		return err
	}
	if err := validateStatDefs("stats_schema.secondary", def.StatsSchema.Secondary); err != nil {
		return err
	}
	if def.StatsSchema.Currency != nil && strings.TrimSpace(def.StatsSchema.Currency.Name) == "" {
		return errors.New("missing stats_schema.currency.name")
	}
	return nil
}

func containsBlankString(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func validateStatDefs(path string, defs []StatDef) error {
	for i, def := range defs {
		if strings.TrimSpace(def.Key) == "" {
			return fmt.Errorf("missing %s[%d].key", path, i)
		}
		if strings.TrimSpace(def.Label) == "" {
			return fmt.Errorf("missing %s[%d].label", path, i)
		}
	}
	return nil
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
