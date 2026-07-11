package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	gameservice "github.com/crimsab/oneday/internal/game/service"
	"github.com/crimsab/oneday/internal/storage"
)

type gatewayTurnResponse struct {
	Events []contracts.TurnEvent `json:"events,omitempty"`
	Error  string                `json:"error,omitempty"`
}

type gatewayTurnStreamLine struct {
	Event *contracts.TurnEvent `json:"event,omitempty"`
	Phase string               `json:"phase,omitempty"`
	Error string               `json:"error,omitempty"`
	Done  bool                 `json:"done,omitempty"`
}

type gatewayMetaResponse struct {
	Meta  *contracts.BrowserMetaResponse `json:"meta,omitempty"`
	Error string                         `json:"error,omitempty"`
}

type gatewaySaveResponse struct {
	Save  *contracts.BrowserSaveView `json:"save,omitempty"`
	Error string                     `json:"error,omitempty"`
}

type gatewayLoadResponse struct {
	Save   *contracts.BrowserSaveView `json:"save,omitempty"`
	Legacy bool                       `json:"legacy,omitempty"`
	Error  string                     `json:"error,omitempty"`
}

type gatewayDeleteSaveResponse struct {
	Save  *contracts.BrowserSaveView `json:"save,omitempty"`
	Error string                     `json:"error,omitempty"`
}

type gatewayCommandDescriptorsResponse struct {
	Commands []contracts.CommandDescriptor `json:"commands,omitempty"`
	Error    string                        `json:"error,omitempty"`
}

type gatewayStoryCreateRequest struct {
	Brief               string `json:"brief"`
	CharacterName       string `json:"character_name"`
	CharacterBackground string `json:"character_background"`
	Start               bool   `json:"start"`
}

type gatewayStoryWizardRequest struct {
	State  *engine.StoryCreatorState `json:"state,omitempty"`
	Input  string                    `json:"input,omitempty"`
	Action string                    `json:"action,omitempty"`
	Start  bool                      `json:"start"`
}

type gatewayStoryEnhanceRequest struct {
	Stage   string                    `json:"stage,omitempty"`
	Text    string                    `json:"text,omitempty"`
	Context string                    `json:"context,omitempty"`
	State   *engine.StoryCreatorState `json:"state,omitempty"`
}

type gatewayStoryCreateResponse struct {
	StoryID     string `json:"story_id,omitempty"`
	CharacterID string `json:"character_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	Started     bool   `json:"started,omitempty"`
	StartError  string `json:"start_error,omitempty"`
	Error       string `json:"error,omitempty"`
}

type gatewayStoryWizardResponse struct {
	State       engine.StoryCreatorState `json:"state,omitempty"`
	Phase       string                   `json:"phase,omitempty"`
	Stage       string                   `json:"stage,omitempty"`
	StageLabel  string                   `json:"stage_label,omitempty"`
	Placeholder string                   `json:"placeholder,omitempty"`
	Message     string                   `json:"message,omitempty"`
	Actions     []engine.CreationAction  `json:"actions,omitempty"`
	Definition  *engine.StoryDefinition  `json:"definition,omitempty"`
	LastModel   string                   `json:"last_model,omitempty"`
	LastLatency int64                    `json:"last_latency_ms,omitempty"`
	StoryID     string                   `json:"story_id,omitempty"`
	CharacterID string                   `json:"character_id,omitempty"`
	SessionID   string                   `json:"session_id,omitempty"`
	Started     bool                     `json:"started,omitempty"`
	StartError  string                   `json:"start_error,omitempty"`
	Error       string                   `json:"error,omitempty"`
}

type gatewayStoryEnhanceResponse struct {
	Text      string `json:"text,omitempty"`
	Model     string `json:"model,omitempty"`
	Provider  string `json:"provider,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

type gatewayModelSettingsResponse struct {
	Settings  *config.ModelRoutingSettings `json:"settings,omitempty"`
	Error     string                       `json:"error,omitempty"`
	ErrorCode string                       `json:"error_code,omitempty"`
}

type gatewaySchemaPreflightResponse struct {
	Status string `json:"status"`
}

func runGatewaySchemaPreflight(out io.Writer) error {
	if err := json.NewEncoder(out).Encode(gatewaySchemaPreflightResponse{Status: "ok"}); err != nil {
		return fmt.Errorf("writing gateway-schema-preflight response: %w", err)
	}
	return nil
}

func runGatewayCommandDescriptors(out io.Writer) error {
	if err := json.NewEncoder(out).Encode(gatewayCommandDescriptorsResponse{Commands: contracts.CommandDescriptors()}); err != nil {
		return fmt.Errorf("writing gateway-command-descriptors response: %w", err)
	}
	return nil
}

func runGatewayStoryCreate(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req gatewayStoryCreateRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayStoryCreateError(out, fmt.Errorf("invalid gateway-story-create JSON: %w", err))
	}
	req.Brief = strings.TrimSpace(req.Brief)
	req.CharacterName = strings.TrimSpace(req.CharacterName)
	req.CharacterBackground = strings.TrimSpace(req.CharacterBackground)
	if req.Brief == "" {
		return writeGatewayStoryCreateError(out, fmt.Errorf("story brief is required"))
	}
	if req.CharacterName == "" {
		return writeGatewayStoryCreateError(out, fmt.Errorf("character name is required"))
	}

	creator := engine.NewStoryCreator(router, db, cfg.AI.Generation)
	if _, err := creator.SendMessage(ctx, req.Brief); err != nil {
		return writeGatewayStoryCreateError(out, fmt.Errorf("drafting story: %w", err))
	}
	for _, action := range []string{"accept_world", "accept_rules", "accept_stats", "create_story"} {
		if _, err := creator.ExecuteAction(ctx, action); err != nil {
			return writeGatewayStoryCreateError(out, fmt.Errorf("executing %s: %w", action, err))
		}
	}
	if _, err := creator.SendMessage(ctx, req.CharacterName); err != nil {
		return writeGatewayStoryCreateError(out, fmt.Errorf("setting protagonist name: %w", err))
	}
	if _, err := creator.SendMessage(ctx, req.CharacterBackground); err != nil {
		return writeGatewayStoryCreateError(out, fmt.Errorf("saving story: %w", err))
	}

	story := creator.Story()
	character := creator.Character()
	resp := gatewayStoryCreateResponse{
		StoryID:     story.ID,
		CharacterID: character.ID,
	}

	if req.Start {
		sessionID, err := startCreatedStory(ctx, cfg, db, router, story.ID)
		if err != nil {
			resp.StartError = fmt.Sprintf("starting story: %v", err)
		} else {
			resp.SessionID = sessionID
			resp.Started = true
		}
	} else {
		sessionID, err := ensureCreatedStorySession(cfg, db, story.ID)
		if err != nil {
			resp.StartError = fmt.Sprintf("creating session: %v", err)
		} else {
			resp.SessionID = sessionID
		}
	}

	if err := json.NewEncoder(out).Encode(resp); err != nil {
		return fmt.Errorf("writing gateway-story-create response: %w", err)
	}
	return nil
}

func runGatewayStoryWizard(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req gatewayStoryWizardRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayStoryWizardError(out, fmt.Errorf("invalid gateway-story-wizard JSON: %w", err))
	}

	creator := engine.NewStoryCreator(router, db, cfg.AI.Generation)
	if req.State != nil {
		if err := creator.RestoreState(*req.State); err != nil {
			return writeGatewayStoryWizardError(out, err)
		}
	}

	input := strings.TrimSpace(req.Input)
	action := strings.TrimSpace(req.Action)
	var message string
	var err error
	switch {
	case req.State == nil && input == "" && action == "":
		message, err = creator.StartConversation(ctx)
	case action != "":
		message, err = creator.ExecuteAction(ctx, action)
	case input != "":
		message, err = creator.SendMessage(ctx, input)
	default:
		message = "Wizard state restored. Continue with a quick choice or typed input."
	}
	if err != nil {
		return writeGatewayStoryWizardError(out, err)
	}

	resp := gatewayStoryWizardResponse{
		State:       creator.ExportState(),
		Phase:       storyCreatorPhaseKey(creator.Phase()),
		Stage:       creator.StageKey(),
		StageLabel:  creator.StageLabel(),
		Placeholder: creator.InputPlaceholder(),
		Message:     message,
		Actions:     creator.Actions(),
		Definition:  creator.Definition(),
		LastModel:   creator.LastModel(),
		LastLatency: creator.LastLatency(),
	}

	if creator.Phase() == engine.PhaseDone && creator.Story() != nil && creator.Character() != nil {
		resp.StoryID = creator.Story().ID
		resp.CharacterID = creator.Character().ID
		resp.State = creator.ExportState()
		if req.Start {
			sessionID, err := startCreatedStory(ctx, cfg, db, router, creator.Story().ID)
			if err != nil {
				resp.StartError = fmt.Sprintf("starting story: %v", err)
			} else {
				resp.SessionID = sessionID
				resp.Started = true
			}
		} else {
			sessionID, err := ensureCreatedStorySession(cfg, db, creator.Story().ID)
			if err != nil {
				resp.StartError = fmt.Sprintf("creating session: %v", err)
			} else {
				resp.SessionID = sessionID
			}
		}
	}

	if err := json.NewEncoder(out).Encode(resp); err != nil {
		return fmt.Errorf("writing gateway-story-wizard response: %w", err)
	}
	return nil
}

func runGatewayStoryEnhance(ctx context.Context, cfg config.Config, router *ai.Router, in io.Reader, out io.Writer) error {
	var req gatewayStoryEnhanceRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayStoryEnhanceError(out, fmt.Errorf("invalid gateway-story-enhance JSON: %w", err))
	}
	seed := strings.TrimSpace(req.Text)
	if seed == "" {
		seed = "Create a concise playable OneDay story setup entry for this stage."
	}
	stateJSON := ""
	if req.State != nil {
		if data, err := json.Marshal(req.State); err == nil {
			stateJSON = string(data)
			if len(stateJSON) > 5000 {
				stateJSON = stateJSON[:5000] + "..."
			}
		}
	}
	userPrompt := strings.Join([]string{
		"Stage: " + strings.TrimSpace(req.Stage),
		"Current text:",
		seed,
		"Context:",
		strings.TrimSpace(req.Context),
		"Wizard state JSON, possibly truncated:",
		stateJSON,
	}, "\n\n")

	maxTokens := cfg.AI.Generation.MaxTokens
	if maxTokens == 0 || maxTokens > 700 {
		maxTokens = 700
	}
	resp, err := router.Complete(ctx, ai.Request{
		Messages: []ai.Message{
			{
				Role:    ai.RoleSystem,
				Content: "You improve text for the OneDay guided story setup wizard. Return only the improved text. No markdown fences, no commentary, no image prompts. Preserve the user's intended language. Make it more playable, concrete, coherent, anti-loop, and not overpowered. Keep it concise enough to paste back into the current wizard field.",
			},
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Temperature: 0.45,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return writeGatewayStoryEnhanceError(out, fmt.Errorf("enhancing story text: %w", err))
	}
	text := cleanupEnhancedStoryText(resp.Content)
	if text == "" {
		return writeGatewayStoryEnhanceError(out, fmt.Errorf("enhance returned empty text"))
	}
	if err := json.NewEncoder(out).Encode(gatewayStoryEnhanceResponse{
		Text:      text,
		Model:     resp.Model,
		Provider:  resp.Provider,
		LatencyMs: resp.LatencyMs,
	}); err != nil {
		return fmt.Errorf("writing gateway-story-enhance response: %w", err)
	}
	return nil
}

func cleanupEnhancedStoryText(text string) string {
	clean := strings.TrimSpace(text)
	clean = strings.TrimPrefix(clean, "```text")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)
	if strings.HasPrefix(clean, "\"") && strings.HasSuffix(clean, "\"") {
		var decoded string
		if err := json.Unmarshal([]byte(clean), &decoded); err == nil {
			clean = strings.TrimSpace(decoded)
		}
	}
	return clean
}

func storyCreatorPhaseKey(phase engine.StoryCreationPhase) string {
	switch phase {
	case engine.PhaseConversation:
		return "conversation"
	case engine.PhaseCharacter:
		return "character"
	case engine.PhaseDone:
		return "done"
	default:
		return "conversation"
	}
}

func startCreatedStory(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, storyID string) (string, error) {
	story, err := db.GetStory(storyID)
	if err != nil {
		return "", err
	}
	character, err := db.GetCharacterByStory(storyID)
	if err != nil {
		return "", err
	}
	world, err := db.GetWorldState(storyID)
	if err != nil {
		return "", err
	}
	session, err := engine.NewGameSession(db, storyID, cfg.DataDir)
	if err != nil {
		return "", err
	}
	defer func() { _ = session.CloseMirrors() }()

	contextCfg := engine.DefaultContextConfig()
	contextCfg.RewardBudget = cfg.Game.RewardBudget
	narrator := engine.NewNarrator(
		router,
		db,
		story,
		character,
		world,
		session,
		contextCfg,
		cfg.AI.Generation,
		cfg.AI.ASCIIArt,
		cfg.DataDir,
		cfg.Game.AutosaveEvery,
	)
	stream, err := narrator.StartNarrationStream(ctx)
	if err != nil {
		return "", err
	}
	for chunk := range stream {
		if chunk.Err != nil {
			return "", chunk.Err
		}
	}
	return session.SessionID(), nil
}

func ensureCreatedStorySession(cfg config.Config, db *storage.DB, storyID string) (string, error) {
	session, err := engine.NewGameSession(db, storyID, cfg.DataDir)
	if err != nil {
		return "", err
	}
	defer func() { _ = session.CloseMirrors() }()
	return session.SessionID(), nil
}

func runGatewayModelSettings(configPath string, out io.Writer) error {
	settings, err := config.ReadModelRoutingSettings(configPath)
	if err != nil {
		return writeGatewayModelSettingsError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayModelSettingsResponse{Settings: &settings}); err != nil {
		return fmt.Errorf("writing gateway-model-settings response: %w", err)
	}
	return nil
}

func runGatewayModelSettingsUpdate(configPath string, in io.Reader, out io.Writer) error {
	var update config.ModelRoutingUpdate
	if err := json.NewDecoder(in).Decode(&update); err != nil {
		return writeGatewayModelSettingsError(out, fmt.Errorf("invalid gateway-model-settings-update JSON: %w", err))
	}
	settings, err := config.UpdateModelRoutingSettings(configPath, update)
	if err != nil {
		return writeGatewayModelSettingsError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayModelSettingsResponse{Settings: &settings}); err != nil {
		return fmt.Errorf("writing gateway-model-settings-update response: %w", err)
	}
	return nil
}

func runGatewayTurn(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.SubmitActionRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayTurnError(out, fmt.Errorf("invalid gateway-turn JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	if req.Stream {
		stream, err := turns.SubmitActionStream(ctx, req)
		if err != nil {
			return writeGatewayTurnStreamError(out, err)
		}
		encoder := json.NewEncoder(out)
		for event := range stream {
			eventCopy := event
			if err := encoder.Encode(gatewayTurnStreamLine{
				Event: &eventCopy,
				Phase: gatewayTurnEventPhase(eventCopy),
			}); err != nil {
				return fmt.Errorf("writing gateway-turn stream event: %w", err)
			}
		}
		if err := encoder.Encode(gatewayTurnStreamLine{Done: true}); err != nil {
			return fmt.Errorf("writing gateway-turn stream done: %w", err)
		}
		return nil
	}

	stream, err := turns.SubmitAction(ctx, req)
	if err != nil {
		return writeGatewayTurnError(out, err)
	}

	events := make([]contracts.TurnEvent, 0, 8)
	for event := range stream {
		events = append(events, event)
	}
	if err := json.NewEncoder(out).Encode(gatewayTurnResponse{Events: events}); err != nil {
		return fmt.Errorf("writing gateway-turn response: %w", err)
	}
	return nil
}

func runGatewayMeta(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.BrowserMetaRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayMetaError(out, fmt.Errorf("invalid gateway-meta JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	resp, err := turns.SubmitMeta(ctx, req)
	if err != nil {
		return writeGatewayMetaError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayMetaResponse{Meta: resp}); err != nil {
		return fmt.Errorf("writing gateway-meta response: %w", err)
	}
	return nil
}

func runGatewaySave(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.BrowserSaveRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewaySaveError(out, fmt.Errorf("invalid gateway-save JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	resp, err := turns.CreateSave(ctx, req)
	if err != nil {
		return writeGatewaySaveError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewaySaveResponse{Save: &resp.Save}); err != nil {
		return fmt.Errorf("writing gateway-save response: %w", err)
	}
	return nil
}

func runGatewayLoad(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.BrowserLoadRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayLoadError(out, fmt.Errorf("invalid gateway-load JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	resp, err := turns.LoadSave(ctx, req)
	if err != nil {
		return writeGatewayLoadError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayLoadResponse{Save: &resp.Save, Legacy: resp.Legacy}); err != nil {
		return fmt.Errorf("writing gateway-load response: %w", err)
	}
	return nil
}

func runGatewayDeleteSave(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.BrowserDeleteSaveRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayDeleteSaveError(out, fmt.Errorf("invalid gateway-delete-save JSON: %w", err))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	resp, err := turns.DeleteSave(ctx, req)
	if err != nil {
		return writeGatewayDeleteSaveError(out, err)
	}
	if err := json.NewEncoder(out).Encode(gatewayDeleteSaveResponse{Save: &resp.Save}); err != nil {
		return fmt.Errorf("writing gateway-delete-save response: %w", err)
	}
	return nil
}

func runGatewayTimeline(db *storage.DB, in io.Reader, out io.Writer) error {
	var req contracts.BrowserTimelineRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return fmt.Errorf("invalid gateway-timeline JSON: %w", err)
	}
	if err := req.Validate(); err != nil {
		return err
	}
	story, err := db.GetStory(req.StoryID)
	if err != nil {
		return err
	}
	switch req.Action {
	case contracts.TimelineFork:
		_, err = db.ForkStoryBranch(req.StoryID, req.FromCommitID, req.Name, req.ClientRevision)
	case contracts.TimelineRename:
		err = db.RenameStoryBranch(req.StoryID, req.BranchID, req.Name, req.ClientRevision)
	case contracts.TimelineCheckout:
		_, err = db.CheckoutStoryBranch(req.StoryID, req.BranchID, req.ClientRevision)
	}
	if err != nil {
		return err
	}
	story, err = db.GetStory(req.StoryID)
	if err != nil {
		return err
	}
	branches, err := db.ListStoryBranches(req.StoryID)
	if err != nil {
		return err
	}
	head, err := db.GetActiveTimeline(req.StoryID)
	if err != nil {
		return err
	}
	views := make([]contracts.TimelineBranchView, 0, len(branches))
	for _, branch := range branches {
		var headTurn int
		if err := db.Conn().QueryRow(`SELECT canonical_turn FROM turn_commits WHERE id=? AND story_id=?`, branch.HeadCommitID, req.StoryID).Scan(&headTurn); err != nil {
			return fmt.Errorf("loading branch head %s: %w", branch.ID, err)
		}
		views = append(views, contracts.TimelineBranchView{ID: branch.ID, StoryID: branch.StoryID, Name: branch.Name, ForkCommitID: branch.ForkCommitID, HeadCommitID: branch.HeadCommitID, HeadTurn: headTurn, CreatedAt: branch.CreatedAt, UpdatedAt: branch.UpdatedAt})
	}
	resp := contracts.BrowserTimelineResponse{ActiveBranchID: story.ActiveBranchID, Revision: story.Revision, Branches: views, Head: &contracts.TimelineCommitView{ID: head.Commit.ID, BranchID: head.Commit.BranchID, ParentCommitID: head.Commit.ParentCommitID, CanonicalTurn: head.Commit.CanonicalTurn, Kind: head.Commit.Kind, Message: head.Commit.Message, CreatedAt: head.Commit.CreatedAt}}
	return json.NewEncoder(out).Encode(resp)
}

func writeGatewayTurnError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayTurnResponse{Error: err.Error()})
	return err
}

func writeGatewayTurnStreamError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayTurnStreamLine{Error: err.Error()})
	return err
}

func gatewayTurnEventPhase(event contracts.TurnEvent) string {
	if strings.Contains(event.ID, ":live:") || event.Type == contracts.EventNarrativeDelta {
		return "live"
	}
	return "final"
}

func writeGatewayStoryCreateError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayStoryCreateResponse{Error: err.Error()})
	return err
}

func writeGatewayStoryWizardError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayStoryWizardResponse{Error: err.Error()})
	return err
}

func writeGatewayStoryEnhanceError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayStoryEnhanceResponse{Error: err.Error()})
	return err
}

func writeGatewayMetaError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayMetaResponse{Error: err.Error()})
	return err
}

func writeGatewaySaveError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewaySaveResponse{Error: err.Error()})
	return err
}

func writeGatewayLoadError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayLoadResponse{Error: err.Error()})
	return err
}

func writeGatewayDeleteSaveError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayDeleteSaveResponse{Error: err.Error()})
	return err
}

func writeGatewayModelSettingsError(out io.Writer, err error) error {
	code := config.ModelRoutingErrorWrite
	var routingErr config.ModelRoutingError
	if errors.As(err, &routingErr) && routingErr.Code != "" {
		code = routingErr.Code
	} else if err != nil && errors.Is(err, context.Canceled) {
		code = "cancelled"
	}
	_ = json.NewEncoder(out).Encode(gatewayModelSettingsResponse{Error: err.Error(), ErrorCode: code})
	return err
}
