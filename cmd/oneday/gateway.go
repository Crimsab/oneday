package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	audioservice "github.com/crimsab/oneday/internal/audio"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/game/gatewayprotocol"
	gameservice "github.com/crimsab/oneday/internal/game/service"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/google/uuid"
)

type gatewayTurnResponse = gatewayprotocol.TurnResponse
type gatewayCraftRequest = gatewayprotocol.CraftRequest
type gatewayCraftResponse = gatewayprotocol.CraftResponse
type gatewayTurnStreamLine = gatewayprotocol.TurnStreamLine
type gatewayMetaResponse = gatewayprotocol.MetaResponse
type gatewaySaveResponse = gatewayprotocol.SaveResponse
type gatewayLoadResponse = gatewayprotocol.LoadResponse
type gatewayDeleteSaveResponse = gatewayprotocol.DeleteSaveResponse
type gatewayCommandDescriptorsResponse = gatewayprotocol.CommandDescriptorsResponse

var gatewayLoadResponseFromContract = gatewayprotocol.LoadResponseFromContract

type gatewayStoryCreateRequest = gatewayprotocol.StoryCreateRequest
type gatewayStoryWizardRequest = gatewayprotocol.StoryWizardRequest
type gatewayStoryEnhanceRequest = gatewayprotocol.StoryEnhanceRequest
type gatewayStoryCreateResponse = gatewayprotocol.StoryCreateResponse
type gatewayStoryWizardResponse = gatewayprotocol.StoryWizardResponse
type gatewayStoryEnhanceResponse = gatewayprotocol.StoryEnhanceResponse
type gatewayModelSettingsResponse = gatewayprotocol.ModelSettingsResponse
type gatewaySchemaPreflightResponse = gatewayprotocol.SchemaPreflightResponse

type gatewayMiniGameRequest = gatewayprotocol.MiniGameRequest
type gatewayMiniGameResponse = gatewayprotocol.MiniGameResponse
type gatewayAudioRequest = gatewayprotocol.AudioRequest
type gatewayAudioExport = gatewayprotocol.AudioExport
type gatewayAudioResponse = gatewayprotocol.AudioResponse

func runGatewayAudio(ctx context.Context, cfg config.Config, db *storage.DB, in io.Reader, out io.Writer) error {
	var req gatewayAudioRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		message := fmt.Sprintf("invalid audio request: %v", err)
		return json.NewEncoder(out).Encode(gatewayAudioResponse{ResponseMeta: gatewayprotocol.Failure("invalid_audio_request", message), Error: message})
	}
	req.Operation, req.StoryID = strings.TrimSpace(req.Operation), strings.TrimSpace(req.StoryID)
	service := audioservice.NewService(db, cfg.AI.TTS)
	response := gatewayAudioResponse{}
	var err error
	requireRevision := func() error {
		current, revisionErr := db.GetStoryRevision(req.StoryID)
		if revisionErr != nil {
			return revisionErr
		}
		if current != req.ClientRevision {
			return fmt.Errorf("%w: expected revision %d, current revision is %d", storage.ErrStaleStoryRevision, req.ClientRevision, current)
		}
		return nil
	}
	switch req.Operation {
	case "catalog":
		response.Statuses, response.Profiles, err = service.DiscoverVoiceProfiles(ctx, strings.TrimSpace(req.Provider), strings.TrimSpace(req.Language))
	case "settings-get":
		response.Settings, err = db.GetStoryTTSSettings(req.StoryID)
	case "settings-put":
		if req.Settings == nil {
			err = errors.New("TTS settings are required")
		} else if err = requireRevision(); err == nil {
			req.Settings.StoryID = req.StoryID
			response.Settings, err = db.UpsertStoryTTSSettings(*req.Settings)
		}
	case "assignments-get":
		response.Assignments, err = db.ListVoiceAssignments(req.StoryID)
	case "assignment-put":
		if req.Assignment == nil {
			err = errors.New("voice assignment is required")
		} else if err = requireRevision(); err == nil {
			req.Assignment.StoryID = req.StoryID
			if req.AssignmentID != "" {
				req.Assignment.ID = req.AssignmentID
			}
			response.Assignment, err = db.UpsertVoiceAssignment(*req.Assignment)
		}
	case "pronunciations-get":
		response.Pronunciations, err = db.ListPronunciations(req.StoryID, req.Language)
	case "pronunciation-put":
		if req.Pronunciation == nil {
			err = errors.New("pronunciation entry is required")
		} else if err = requireRevision(); err == nil {
			req.Pronunciation.StoryID = req.StoryID
			if req.PronunciationID != "" {
				req.Pronunciation.ID = req.PronunciationID
			}
			response.Pronunciation, err = db.UpsertPronunciation(*req.Pronunciation)
		}
	case "pronunciation-delete":
		if err = requireRevision(); err == nil {
			err = db.DeletePronunciation(req.StoryID, req.PronunciationID)
		}
	case "message-get":
		response.Assets, err = db.ListMessageAudio(req.StoryID, req.MessageID)
		if err == nil {
			response.Jobs, err = db.ListMessageTTSJobs(req.StoryID, req.MessageID)
		}
	case "message-create":
		_, err = service.QueueCommittedMessage(ctx, req.StoryID, req.MessageID)
		if err == nil {
			response.Jobs, err = db.ListMessageTTSJobs(req.StoryID, req.MessageID)
		}
		if err == nil {
			for _, job := range response.Jobs {
				if job.Status == "queued" || job.Status == "failed" {
					_ = service.ProcessJob(ctx, job.ID)
				}
			}
			response.Assets, err = db.ListMessageAudio(req.StoryID, req.MessageID)
			if err == nil {
				response.Jobs, err = db.ListMessageTTSJobs(req.StoryID, req.MessageID)
			}
		}
	case "job-cancel":
		var job *storage.TTSJob
		job, err = db.GetTTSJob(req.JobID)
		if err == nil && job.StoryID != req.StoryID {
			err = errors.New("TTS job belongs to another story")
		}
		if err == nil {
			_, err = db.CancelTTSJob(req.StoryID, req.JobID)
		}
		if err == nil {
			var asset *storage.AudioAsset
			asset, err = db.GetAudioAsset(req.StoryID, job.AudioAssetID)
			if err == nil {
				req.MessageID = asset.SourceMessageID
			}
		}
		if err == nil {
			response.Assets, err = db.ListMessageAudio(req.StoryID, req.MessageID)
		}
		if err == nil {
			response.Jobs, err = db.ListMessageTTSJobs(req.StoryID, req.MessageID)
		}
	case "job-retry":
		var job *storage.TTSJob
		job, err = db.GetTTSJob(req.JobID)
		if err == nil && job.StoryID != req.StoryID {
			err = errors.New("TTS job belongs to another story")
		}
		if err == nil {
			_, err = db.RetryTTSJob(req.StoryID, req.JobID)
		}
		if err == nil {
			_ = service.ProcessJob(ctx, req.JobID)
			var asset *storage.AudioAsset
			asset, err = db.GetAudioAsset(req.StoryID, job.AudioAssetID)
			if err == nil {
				req.MessageID = asset.SourceMessageID
			}
		}
		if err == nil {
			response.Assets, err = db.ListMessageAudio(req.StoryID, req.MessageID)
		}
		if err == nil {
			response.Jobs, err = db.ListMessageTTSJobs(req.StoryID, req.MessageID)
		}
	case "cleanup":
		var cleanup audioservice.CleanupResult
		if _, err = db.GetStory(req.StoryID); err == nil {
			cleanup, err = service.CleanupAudioCache(req.DryRun)
		}
		response.Cleanup = &cleanup
	case "export":
		manifest := gatewayAudioExport{
			Format: "oneday-audio-manifest-v1", Filename: "oneday-audio-" + safeAudioFilename(req.StoryID) + ".json",
			GeneratedAt: time.Now().UTC().Format(time.RFC3339), StoryID: req.StoryID,
		}
		manifest.Settings, err = db.GetStoryTTSSettings(req.StoryID)
		if err == nil {
			manifest.Providers, manifest.Voices, err = service.DiscoverVoiceProfiles(ctx, "", "")
		}
		if err == nil {
			manifest.Assignments, err = db.ListVoiceAssignments(req.StoryID)
		}
		if err == nil {
			manifest.Pronunciations, err = db.ListPronunciations(req.StoryID, "")
		}
		if err == nil {
			manifest.Assets, err = db.ListStoryAudio(req.StoryID)
		}
		if err == nil {
			manifest.Jobs, err = db.ListStoryTTSJobs(req.StoryID)
		}
		for index := range manifest.Assets {
			manifest.Assets[index].FilePath = ""
			if manifest.Assets[index].Status == "ready" {
				manifest.Assets[index].URL = "/api/audio/" + manifest.Assets[index].ID
			}
		}
		response.Export = &manifest
	case "asset-path":
		response.Asset, response.FilePath, err = service.ResolveAudioFile(req.AssetID)
		if response.Asset != nil {
			response.Format = response.Asset.OutputFormat
		}
	default:
		err = fmt.Errorf("unknown audio operation %q", req.Operation)
	}
	if err != nil {
		response.Error = err.Error()
		response.ResponseMeta = gatewayprotocol.Failure("audio_operation_failed", err.Error())
	}
	return json.NewEncoder(out).Encode(response)
}

func safeAudioFilename(value string) string {
	value = strings.Map(func(char rune) rune {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' {
			return char
		}
		return '-'
	}, strings.TrimSpace(value))
	if value == "" {
		return "story"
	}
	return value
}

func runGatewayMiniGame(db *storage.DB, operation string, in io.Reader, out io.Writer) error {
	var req gatewayMiniGameRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		message := fmt.Sprintf("invalid minigame request: %v", err)
		return json.NewEncoder(out).Encode(gatewayMiniGameResponse{ResponseMeta: gatewayprotocol.Failure("invalid_minigame_request", message), Error: message})
	}
	req.StoryID = strings.TrimSpace(req.StoryID)
	if req.StoryID == "" {
		const message = "story id is required"
		return json.NewEncoder(out).Encode(gatewayMiniGameResponse{ResponseMeta: gatewayprotocol.Failure("invalid_minigame_request", message), Error: message})
	}
	host := engine.NewMiniGameHost()
	var instance *engine.MiniGameInstance
	var err error
	switch operation {
	case "start":
		head, headErr := db.GetActiveTimeline(req.StoryID)
		if headErr != nil {
			err = headErr
			break
		}
		definition := req.Definition
		if definition.ID == "" {
			kind := definition.Kind
			if kind == "" {
				kind = req.Kind
			}
			if kind == "" {
				recent, recentErr := db.ListRecentMiniGameInstances(req.StoryID, 20)
				if recentErr != nil {
					err = recentErr
					break
				}
				req.Selection.CurrentTurn = head.Commit.CanonicalTurn
				req.Selection.Recent = make([]engine.MiniGameUsage, 0, len(recent))
				for _, record := range recent {
					req.Selection.Recent = append(req.Selection.Recent, engine.MiniGameUsage{Kind: engine.MiniGameType(record.Kind), Turn: record.Turn})
				}
				selected, selectErr := engine.SelectMiniGame(engine.DefaultMiniGameCandidates(), req.Selection)
				if selectErr != nil {
					err = selectErr
					break
				}
				definition = selected.Definition
			} else {
				definition = engine.DefaultMiniGameDefinition(kind)
				if definition.Rules == nil {
					definition.Rules = map[string]string{}
				}
				definition.Rules["selection_reason"] = "player selected; timing-free"
			}
		}
		id := "mini-" + uuid.NewString()
		digest := sha256.Sum256([]byte(id + "\x00" + req.StoryID + "\x00" + head.Branch.ID))
		seed := int64(binary.BigEndian.Uint64(digest[:8]) & uint64(contracts.MaxPortableChallengeSeed))
		value := engine.NewMiniGameInstance(id, req.StoryID, head.Branch.ID, head.Commit.CanonicalTurn, seed, definition)
		if err = host.Start(&value); err == nil {
			instance, err = saveGatewayMiniGame(db, host, value)
		}
	case "get":
		var record *storage.MiniGameInstanceRecord
		record, err = db.GetActiveMiniGameInstance(req.StoryID)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
			break
		}
		if err == nil {
			instance, err = host.Restore(record.Instance)
		}
	case "input":
		var record *storage.MiniGameInstanceRecord
		record, err = db.GetMiniGameInstance(req.StoryID, strings.TrimSpace(req.InstanceID))
		if err == nil {
			instance, err = host.Restore(record.Instance)
		}
		if err == nil {
			err = host.Apply(instance, req.Input)
		}
		if err == nil {
			instance, err = saveGatewayMiniGame(db, host, *instance)
		}
	default:
		err = fmt.Errorf("unknown minigame operation %q", operation)
	}
	if err != nil {
		return json.NewEncoder(out).Encode(gatewayMiniGameResponse{ResponseMeta: gatewayprotocol.Failure("minigame_operation_failed", err.Error()), Error: err.Error()})
	}
	if instance == nil {
		return json.NewEncoder(out).Encode(gatewayMiniGameResponse{})
	}
	view := engine.PlayerMiniGameView(*instance)
	return json.NewEncoder(out).Encode(gatewayMiniGameResponse{Instance: &view})
}

func saveGatewayMiniGame(db *storage.DB, host *engine.MiniGameHost, instance engine.MiniGameInstance) (*engine.MiniGameInstance, error) {
	payload, err := host.Serialize(instance)
	if err != nil {
		return nil, err
	}
	_, err = db.SaveMiniGameInstance(storage.MiniGameInstanceRecord{
		ID: instance.ID, StoryID: instance.StoryID, Turn: instance.Turn,
		ProtocolVersion: instance.ProtocolVersion, Kind: string(instance.Definition.Kind),
		Phase: string(instance.Runtime.Phase), Instance: payload,
	})
	if err != nil {
		return nil, err
	}
	return &instance, nil
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
	enhanceCtx := ai.WithTelemetry(ctx, ai.TelemetryMetadata{Stage: "story_enhance", PromptProfile: "story_enhance", PromptTemplate: "v1", SafeMetadata: map[string]string{"wizard_stage": strings.TrimSpace(req.Stage)}})
	resp, err := router.Complete(enhanceCtx, ai.Request{
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
	audio := audioservice.NewService(db, cfg.AI.TTS)
	_, _ = audio.EnsureConfiguredVoiceProfiles()
	narrator.SetCommittedAudioQueue(func(ctx context.Context, storyID string, messageID int64) error {
		_, err := audio.QueueCommittedMessage(ctx, storyID, messageID)
		return err
	})
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
		var sequence int64
		for event := range stream {
			sequence++
			eventCopy := event
			if err := encoder.Encode(gatewayTurnStreamLine{
				Event:    &eventCopy,
				Phase:    gatewayTurnEventPhase(eventCopy),
				Sequence: sequence,
			}); err != nil {
				return fmt.Errorf("writing gateway-turn stream event: %w", err)
			}
		}
		sequence++
		if err := encoder.Encode(gatewayTurnStreamLine{Done: true, Sequence: sequence}); err != nil {
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

func runGatewayCraft(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req gatewayCraftRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayCraftError(out, fmt.Errorf("invalid gateway-craft JSON: %w", err))
	}
	req.StoryID = strings.TrimSpace(req.StoryID)
	req.Message = strings.TrimSpace(req.Message)
	if req.StoryID == "" || req.Message == "" {
		return writeGatewayCraftError(out, fmt.Errorf("story_id and message are required"))
	}
	story, err := db.GetStory(req.StoryID)
	if err != nil {
		return writeGatewayCraftError(out, err)
	}
	character, err := db.GetCharacterByStory(req.StoryID)
	if err != nil {
		return writeGatewayCraftError(out, err)
	}
	world, err := db.GetWorldState(req.StoryID)
	if err != nil {
		return writeGatewayCraftError(out, err)
	}
	session, err := engine.NewGameSession(db, req.StoryID, cfg.DataDir)
	if err != nil {
		return writeGatewayCraftError(out, err)
	}
	defer func() { _ = session.CloseMirrors() }()
	contextCfg := engine.DefaultContextConfig()
	contextCfg.RewardBudget = cfg.Game.RewardBudget
	narrator := engine.NewNarrator(router, db, story, character, world, session, contextCfg, cfg.AI.Generation, cfg.AI.ASCIIArt, cfg.DataDir, cfg.Game.AutosaveEvery)
	crafting, err := engine.NewCraftingEngine(narrator)
	if err != nil {
		return writeGatewayCraftError(out, err)
	}
	defer func() { _ = crafting.Close() }()
	crafting.RestoreConversation(req.History)
	response, err := crafting.SendMessage(ctx, req.Message)
	if err != nil {
		return writeGatewayCraftError(out, err)
	}
	return json.NewEncoder(out).Encode(gatewayCraftResponse{Crafting: response})
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
	if err := json.NewEncoder(out).Encode(gatewayLoadResponseFromContract(resp)); err != nil {
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
	rows, err := db.Conn().Query(`
		WITH RECURSIVE ancestors(id,branch_id,parent_commit_id,canonical_turn,kind,message,created_at) AS (
			SELECT c.id,c.branch_id,COALESCE(c.parent_commit_id,''),c.canonical_turn,c.kind,c.message,c.created_at
			FROM turn_commits c JOIN story_branches b ON b.head_commit_id=c.id
			WHERE b.story_id=? AND c.story_id=?
			UNION
			SELECT parent.id,parent.branch_id,COALESCE(parent.parent_commit_id,''),parent.canonical_turn,parent.kind,parent.message,parent.created_at
			FROM turn_commits parent JOIN ancestors child ON child.parent_commit_id=parent.id
			WHERE parent.story_id=?
		)
		SELECT id,branch_id,parent_commit_id,canonical_turn,kind,message,created_at
		FROM ancestors ORDER BY canonical_turn,created_at,id`, req.StoryID, req.StoryID, req.StoryID)
	if err != nil {
		return fmt.Errorf("loading active timeline ancestry: %w", err)
	}
	defer rows.Close()
	commits := make([]contracts.TimelineCommitView, 0)
	for rows.Next() {
		var commit contracts.TimelineCommitView
		if err := rows.Scan(&commit.ID, &commit.BranchID, &commit.ParentCommitID, &commit.CanonicalTurn, &commit.Kind, &commit.Message, &commit.CreatedAt); err != nil {
			return fmt.Errorf("scanning active timeline ancestry: %w", err)
		}
		commits = append(commits, commit)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating active timeline ancestry: %w", err)
	}
	resp := contracts.BrowserTimelineResponse{ActiveBranchID: story.ActiveBranchID, Revision: story.Revision, Branches: views, Head: &contracts.TimelineCommitView{ID: head.Commit.ID, BranchID: head.Commit.BranchID, ParentCommitID: head.Commit.ParentCommitID, CanonicalTurn: head.Commit.CanonicalTurn, Kind: head.Commit.Kind, Message: head.Commit.Message, CreatedAt: head.Commit.CreatedAt}, Commits: commits}
	return json.NewEncoder(out).Encode(resp)
}

func writeGatewayTurnError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayTurnResponse{ResponseMeta: gatewayprotocol.Failure("turn_failed", err.Error()), Error: err.Error()})
	return err
}

func writeGatewayCraftError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayCraftResponse{ResponseMeta: gatewayprotocol.Failure("craft_failed", err.Error()), Error: err.Error()})
	return err
}

func writeGatewayTurnStreamError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayTurnStreamLine{ResponseMeta: gatewayprotocol.Failure("turn_stream_failed", err.Error()), Error: err.Error(), Sequence: 1})
	return err
}

func gatewayTurnEventPhase(event contracts.TurnEvent) string {
	if strings.Contains(event.ID, ":live:") || event.Type == contracts.EventNarrativeDelta {
		return "live"
	}
	return "final"
}

func writeGatewayStoryCreateError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayStoryCreateResponse{ResponseMeta: gatewayprotocol.Failure("story_create_failed", err.Error()), Error: err.Error()})
	return err
}

func writeGatewayStoryWizardError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayStoryWizardResponse{ResponseMeta: gatewayprotocol.Failure("story_wizard_failed", err.Error()), Error: err.Error()})
	return err
}

func writeGatewayStoryEnhanceError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayStoryEnhanceResponse{ResponseMeta: gatewayprotocol.Failure("story_enhance_failed", err.Error()), Error: err.Error()})
	return err
}

func writeGatewayMetaError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayMetaResponse{ResponseMeta: gatewayprotocol.Failure("meta_failed", err.Error()), Error: err.Error()})
	return err
}

func writeGatewaySaveError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewaySaveResponse{ResponseMeta: gatewayprotocol.Failure("save_failed", err.Error()), Error: err.Error()})
	return err
}

func writeGatewayLoadError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayLoadResponse{ResponseMeta: gatewayprotocol.Failure("load_failed", err.Error()), Error: err.Error()})
	return err
}

func writeGatewayDeleteSaveError(out io.Writer, err error) error {
	_ = json.NewEncoder(out).Encode(gatewayDeleteSaveResponse{ResponseMeta: gatewayprotocol.Failure("delete_save_failed", err.Error()), Error: err.Error()})
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
	_ = json.NewEncoder(out).Encode(gatewayModelSettingsResponse{ResponseMeta: gatewayprotocol.Failure(code, err.Error()), Error: err.Error(), ErrorCode: code})
	return err
}
