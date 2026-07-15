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
	"log"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	audioservice "github.com/crimsab/oneday/internal/audio"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/game/gatewayprotocol"
	gameservice "github.com/crimsab/oneday/internal/game/service"
	appi18n "github.com/crimsab/oneday/internal/i18n"
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

const (
	gatewayCodeInvalidRequest = "invalid_request"
	gatewayCodeStaleRequest   = "stale_request"
	gatewayCodeConflict       = "conflict"
	gatewayCodeNotFound       = "not_found"
	gatewayCodeInternal       = "internal_error"
)

type gatewayCauseError struct {
	code string
	err  error
}

func (e gatewayCauseError) Error() string { return e.err.Error() }
func (e gatewayCauseError) Unwrap() error { return e.err }

func gatewayCause(code string, err error) error {
	if err == nil {
		return nil
	}
	return gatewayCauseError{code: code, err: err}
}

func gatewayErrorCode(err error) string {
	var cause gatewayCauseError
	if errors.As(err, &cause) && cause.code != "" {
		return cause.code
	}
	switch {
	case errors.Is(err, sql.ErrNoRows),
		errors.Is(err, storage.ErrBranchNotFound),
		errors.Is(err, storage.ErrCommitNotFound):
		return gatewayCodeNotFound
	case errors.Is(err, storage.ErrStaleWorldTurn),
		errors.Is(err, storage.ErrStaleStoryRevision),
		errors.Is(err, storage.ErrStaleBranchHead),
		errors.Is(err, storage.ErrStoryTurnLockLost):
		return gatewayCodeStaleRequest
	case errors.Is(err, storage.ErrTurnIdempotencyConflict),
		errors.Is(err, storage.ErrTurnIdempotencyInProgress),
		errors.Is(err, storage.ErrTurnIdempotencyLost):
		return gatewayCodeConflict
	default:
		return gatewayCodeInternal
	}
}

func gatewayErrorPresentation(err error) (string, string) {
	code := gatewayErrorCode(err)
	if code == gatewayCodeInternal {
		log.Printf("oneday: gateway internal error: %v", err)
		return code, "An internal gateway error occurred."
	}
	return code, err.Error()
}

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
			err = gatewayCause(gatewayCodeInvalidRequest, errors.New("TTS settings are required"))
		} else if err = requireRevision(); err == nil {
			req.Settings.StoryID = req.StoryID
			response.Settings, err = db.UpsertStoryTTSSettings(*req.Settings)
		}
	case "assignments-get":
		response.Assignments, err = db.ListVoiceAssignments(req.StoryID)
	case "assignment-put":
		if req.Assignment == nil {
			err = gatewayCause(gatewayCodeInvalidRequest, errors.New("voice assignment is required"))
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
			err = gatewayCause(gatewayCodeInvalidRequest, errors.New("pronunciation entry is required"))
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
			err = gatewayCause(gatewayCodeConflict, errors.New("TTS job belongs to another story"))
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
			err = gatewayCause(gatewayCodeConflict, errors.New("TTS job belongs to another story"))
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
		err = gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("unknown audio operation %q", req.Operation))
	}
	if err != nil {
		code, message := gatewayErrorPresentation(err)
		response.Error = message
		response.ResponseMeta = gatewayprotocol.Failure(code, message)
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
		if startErr := host.Start(&value); startErr != nil {
			err = gatewayCause(gatewayCodeInvalidRequest, startErr)
		} else {
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
			if applyErr := host.Apply(instance, req.Input); applyErr != nil {
				err = gatewayCause(gatewayCodeInvalidRequest, applyErr)
			}
		}
		if err == nil {
			instance, err = saveGatewayMiniGame(db, host, *instance)
		}
	default:
		err = gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("unknown minigame operation %q", operation))
	}
	if err != nil {
		code, message := gatewayErrorPresentation(err)
		return json.NewEncoder(out).Encode(gatewayMiniGameResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
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

func runGatewayCommandDescriptors(out io.Writer, locale ...string) error {
	loc := appi18n.New(appi18n.English)
	if len(locale) > 0 {
		loc = appi18n.New(appi18n.Normalize(locale[0]))
	}
	if err := json.NewEncoder(out).Encode(gatewayCommandDescriptorsResponse{Commands: contracts.CommandDescriptors(loc)}); err != nil {
		return fmt.Errorf("writing gateway-command-descriptors response: %w", err)
	}
	return nil
}

func runGatewayStoryCreate(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req gatewayStoryCreateRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayStoryCreateError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-story-create JSON: %w", err)))
	}
	req.Brief = strings.TrimSpace(req.Brief)
	req.CharacterName = strings.TrimSpace(req.CharacterName)
	req.CharacterBackground = strings.TrimSpace(req.CharacterBackground)
	if req.Brief == "" {
		return writeGatewayStoryCreateError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("story brief is required")))
	}
	if req.CharacterName == "" {
		return writeGatewayStoryCreateError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("character name is required")))
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
		return writeGatewayStoryWizardError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-story-wizard JSON: %w", err)))
	}

	creator := engine.NewStoryCreator(router, db, cfg.AI.Generation)
	if req.State != nil {
		if err := creator.RestoreState(*req.State); err != nil {
			return writeGatewayStoryWizardError(out, gatewayCause(gatewayCodeInvalidRequest, err))
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
		return writeGatewayStoryEnhanceError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-story-enhance JSON: %w", err)))
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
		return writeGatewayModelSettingsError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-model-settings-update JSON: %w", err)))
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

func gatewayMutationPreflight(
	ctx context.Context,
	turns *gameservice.InProcessTurnService,
	storyID string,
	sessionID string,
	clientTurn int,
	clientRevision int64,
) error {
	if strings.TrimSpace(storyID) == "" {
		return gatewayCause(gatewayCodeInvalidRequest, errors.New("story_id is required"))
	}
	if strings.TrimSpace(sessionID) == "" {
		return gatewayCause(gatewayCodeInvalidRequest, errors.New("session_id is required"))
	}
	snapshot, err := turns.Snapshot(ctx, storyID)
	if err != nil {
		return err
	}
	if clientTurn != snapshot.Turn {
		return gatewayCause(gatewayCodeStaleRequest, fmt.Errorf("stale client_turn %d, current turn is %d", clientTurn, snapshot.Turn))
	}
	if clientRevision != snapshot.Revision {
		return gatewayCause(gatewayCodeStaleRequest, fmt.Errorf("stale client_revision %d, current revision is %d", clientRevision, snapshot.Revision))
	}
	if snapshot.SessionID != "" && sessionID != snapshot.SessionID {
		return gatewayCause(gatewayCodeStaleRequest, fmt.Errorf("stale session_id %q, active session is %q", sessionID, snapshot.SessionID))
	}
	return nil
}

func gatewayTurnPreflight(ctx context.Context, db *storage.DB, turns *gameservice.InProcessTurnService, req contracts.SubmitActionRequest) error {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return gatewayCause(gatewayCodeInvalidRequest, errors.New("idempotency_key is required"))
	}
	if strings.TrimSpace(req.Action.Text) == "" && req.Action.ChoiceID == 0 {
		return gatewayCause(gatewayCodeInvalidRequest, errors.New("action text or choice_id is required"))
	}
	var status string
	err := db.Conn().QueryRow(
		`SELECT status FROM turn_idempotency WHERE story_id=? AND idempotency_key=?`,
		strings.TrimSpace(req.StoryID), strings.TrimSpace(req.IdempotencyKey),
	).Scan(&status)
	if err == nil && status == "committed" {
		// Let the turn service replay the durable response or return its typed
		// idempotency conflict without rejecting an intentionally old retry.
		return nil
	}
	if err == nil && status == "running" {
		return gatewayCause(gatewayCodeConflict, storage.ErrTurnIdempotencyInProgress)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return gatewayMutationPreflight(ctx, turns, req.StoryID, req.SessionID, req.ClientTurn, req.ClientRevision)
}

func runGatewayTurn(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req contracts.SubmitActionRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayTurnError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-turn JSON: %w", err)))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	if err := gatewayTurnPreflight(ctx, db, turns, req); err != nil {
		if req.Stream {
			return writeGatewayTurnStreamError(out, err)
		}
		return writeGatewayTurnError(out, err)
	}
	if req.Stream {
		stream, err := turns.SubmitActionStream(ctx, req)
		if err != nil {
			return writeGatewayTurnStreamError(out, err)
		}
		encoder := json.NewEncoder(out)
		var sequence int64
		for event := range stream {
			sequence++
			eventCopy := gatewayStreamEvent(event)
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

func gatewayStreamEvent(event contracts.TurnEvent) contracts.TurnEvent {
	if event.Type != contracts.EventError {
		return event
	}
	event.Payload, _ = json.Marshal(map[string]string{
		"code":    gatewayCodeInternal,
		"message": "An internal gateway error occurred.",
	})
	return event
}

func runGatewayCraft(ctx context.Context, cfg config.Config, db *storage.DB, router *ai.Router, in io.Reader, out io.Writer) error {
	var req gatewayCraftRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return writeGatewayCraftError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-craft JSON: %w", err)))
	}
	req.StoryID = strings.TrimSpace(req.StoryID)
	req.Message = strings.TrimSpace(req.Message)
	if req.StoryID == "" || req.Message == "" {
		return writeGatewayCraftError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("story_id and message are required")))
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
		return writeGatewayMetaError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-meta JSON: %w", err)))
	}
	switch req.Kind {
	case contracts.BrowserMetaKindBTW, contracts.BrowserMetaKindGuide, contracts.BrowserMetaKindNarrator:
	default:
		return writeGatewayMetaError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("unsupported browser meta kind %q", req.Kind)))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	if err := gatewayMutationPreflight(ctx, turns, req.StoryID, req.SessionID, req.ClientTurn, req.ClientRevision); err != nil {
		return writeGatewayMetaError(out, err)
	}
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
		return writeGatewaySaveError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-save JSON: %w", err)))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	if err := gatewayMutationPreflight(ctx, turns, req.StoryID, req.SessionID, req.ClientTurn, req.ClientRevision); err != nil {
		return writeGatewaySaveError(out, err)
	}
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
		return writeGatewayLoadError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-load JSON: %w", err)))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	if strings.TrimSpace(req.SaveID) == "" {
		return writeGatewayLoadError(out, gatewayCause(gatewayCodeInvalidRequest, errors.New("save_id is required")))
	}
	if err := gatewayMutationPreflight(ctx, turns, req.StoryID, req.SessionID, req.ClientTurn, req.ClientRevision); err != nil {
		return writeGatewayLoadError(out, err)
	}
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
		return writeGatewayDeleteSaveError(out, gatewayCause(gatewayCodeInvalidRequest, fmt.Errorf("invalid gateway-delete-save JSON: %w", err)))
	}

	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	if strings.TrimSpace(req.SaveID) == "" {
		return writeGatewayDeleteSaveError(out, gatewayCause(gatewayCodeInvalidRequest, errors.New("save_id is required")))
	}
	if err := gatewayMutationPreflight(ctx, turns, req.StoryID, req.SessionID, req.ClientTurn, req.ClientRevision); err != nil {
		return writeGatewayDeleteSaveError(out, err)
	}
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
	case contracts.TimelineForkCheckout:
		_, err = db.ForkAndCheckoutStoryBranch(req.StoryID, req.FromCommitID, req.Name, req.ClientRevision)
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
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayTurnResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewayCraftError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayCraftResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewayTurnStreamError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayTurnStreamLine{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message, Sequence: 1})
	return err
}

func gatewayTurnEventPhase(event contracts.TurnEvent) string {
	if strings.Contains(event.ID, ":live:") || event.Type == contracts.EventNarrativeDelta {
		return "live"
	}
	return "final"
}

func writeGatewayStoryCreateError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayStoryCreateResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewayStoryWizardError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayStoryWizardResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewayStoryEnhanceError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayStoryEnhanceResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewayMetaError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayMetaResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewaySaveError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewaySaveResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewayLoadError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayLoadResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewayDeleteSaveError(out io.Writer, err error) error {
	code, message := gatewayErrorPresentation(err)
	_ = json.NewEncoder(out).Encode(gatewayDeleteSaveResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message})
	return err
}

func writeGatewayModelSettingsError(out io.Writer, err error) error {
	code := config.ModelRoutingErrorWrite
	var routingErr config.ModelRoutingError
	var cause gatewayCauseError
	if errors.As(err, &cause) && cause.code != "" {
		code = cause.code
	} else if errors.As(err, &routingErr) && routingErr.Code != "" {
		code = routingErr.Code
	} else if err != nil && errors.Is(err, context.Canceled) {
		code = "cancelled"
	}
	message := err.Error()
	if code == config.ModelRoutingErrorWrite {
		log.Printf("oneday: gateway model settings error: %v", err)
		message = "An internal gateway error occurred."
	}
	_ = json.NewEncoder(out).Encode(gatewayModelSettingsResponse{ResponseMeta: gatewayprotocol.Failure(code, message), Error: message, ErrorCode: code})
	return err
}
