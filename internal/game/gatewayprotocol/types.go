package gatewayprotocol

import (
	"encoding/json"
	"github.com/crimsab/oneday/internal/ai"
	audioservice "github.com/crimsab/oneday/internal/audio"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

const Version = 1

type Error struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Retryable bool            `json:"retryable"`
	Details   json.RawMessage `json:"details,omitempty"`
}

type ResponseMeta struct {
	ProtocolVersion int    `json:"protocol_version,omitempty"`
	RequestID       string `json:"request_id,omitempty"`
	ErrorDetail     *Error `json:"error_detail,omitempty"`
}

func Failure(code, message string) ResponseMeta {
	return ResponseMeta{
		ProtocolVersion: Version,
		ErrorDetail: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// SchemaRoot keeps every public bridge shape reachable from one reflection
// root. It is not serialized at runtime; it is the canonical codegen input.
type SchemaRoot struct {
	TurnRequest                contracts.SubmitActionRequest      `json:"turn_request"`
	MetaRequest                contracts.BrowserMetaRequest       `json:"meta_request"`
	SaveRequest                contracts.BrowserSaveRequest       `json:"save_request"`
	LoadRequest                contracts.BrowserLoadRequest       `json:"load_request"`
	DeleteSaveRequest          contracts.BrowserDeleteSaveRequest `json:"delete_save_request"`
	TimelineRequest            contracts.BrowserTimelineRequest   `json:"timeline_request"`
	TimelineResponse           contracts.BrowserTimelineResponse  `json:"timeline_response"`
	TurnResponse               TurnResponse                       `json:"turn_response"`
	CraftRequest               CraftRequest                       `json:"craft_request"`
	CraftResponse              CraftResponse                      `json:"craft_response"`
	TurnStreamLine             TurnStreamLine                     `json:"turn_stream_line"`
	MetaResponse               MetaResponse                       `json:"meta_response"`
	SaveResponse               SaveResponse                       `json:"save_response"`
	LoadResponse               LoadResponse                       `json:"load_response"`
	DeleteSaveResponse         DeleteSaveResponse                 `json:"delete_save_response"`
	CommandDescriptorsResponse CommandDescriptorsResponse         `json:"command_descriptors_response"`
	StoryCreateRequest         StoryCreateRequest                 `json:"story_create_request"`
	StoryWizardRequest         StoryWizardRequest                 `json:"story_wizard_request"`
	StoryEnhanceRequest        StoryEnhanceRequest                `json:"story_enhance_request"`
	StoryCreateResponse        StoryCreateResponse                `json:"story_create_response"`
	StoryWizardResponse        StoryWizardResponse                `json:"story_wizard_response"`
	StoryEnhanceResponse       StoryEnhanceResponse               `json:"story_enhance_response"`
	ModelSettingsResponse      ModelSettingsResponse              `json:"model_settings_response"`
	ModelSettingsUpdate        config.ModelRoutingUpdate          `json:"model_settings_update"`
	SchemaPreflightResponse    SchemaPreflightResponse            `json:"schema_preflight_response"`
	MiniGameRequest            MiniGameRequest                    `json:"minigame_request"`
	MiniGameResponse           MiniGameResponse                   `json:"minigame_response"`
	AudioRequest               AudioRequest                       `json:"audio_request"`
	AudioResponse              AudioResponse                      `json:"audio_response"`
	Error                      Error                              `json:"error"`
}

type TurnResponse struct {
	ResponseMeta
	Events []contracts.TurnEvent `json:"events,omitempty"`
	Error  string                `json:"error,omitempty"`
}
type CraftRequest struct {
	StoryID string       `json:"story_id"`
	Message string       `json:"message"`
	History []ai.Message `json:"history,omitempty"`
}
type CraftResponse struct {
	ResponseMeta
	Crafting *engine.CraftingResponse `json:"crafting,omitempty"`
	Error    string                   `json:"error,omitempty"`
}
type TurnStreamLine struct {
	ResponseMeta
	Event    *contracts.TurnEvent `json:"event,omitempty"`
	Phase    string               `json:"phase,omitempty"`
	Error    string               `json:"error,omitempty"`
	Done     bool                 `json:"done,omitempty"`
	Sequence int64                `json:"sequence,omitempty"`
}
type MetaResponse struct {
	ResponseMeta
	Meta  *contracts.BrowserMetaResponse `json:"meta,omitempty"`
	Error string                         `json:"error,omitempty"`
}
type SaveResponse struct {
	ResponseMeta
	Save  *contracts.BrowserSaveView `json:"save,omitempty"`
	Error string                     `json:"error,omitempty"`
}
type LoadResponse struct {
	ResponseMeta
	Save           *contracts.BrowserSaveView `json:"save,omitempty"`
	Legacy         bool                       `json:"legacy,omitempty"`
	SnapshotState  string                     `json:"snapshot_state"`
	SnapshotDetail string                     `json:"snapshot_detail,omitempty"`
	Error          string                     `json:"error,omitempty"`
}

func LoadResponseFromContract(resp *contracts.BrowserLoadResponse) LoadResponse {
	return LoadResponse{Save: &resp.Save, Legacy: resp.Legacy, SnapshotState: resp.SnapshotState, SnapshotDetail: resp.SnapshotDetail}
}

type DeleteSaveResponse struct {
	ResponseMeta
	Save  *contracts.BrowserSaveView `json:"save,omitempty"`
	Error string                     `json:"error,omitempty"`
}
type CommandDescriptorsResponse struct {
	ResponseMeta
	Commands []contracts.CommandDescriptor `json:"commands,omitempty"`
	Error    string                        `json:"error,omitempty"`
}
type StoryCreateRequest struct {
	Brief               string `json:"brief"`
	CharacterName       string `json:"character_name"`
	CharacterBackground string `json:"character_background"`
	Start               bool   `json:"start"`
}
type StoryWizardRequest struct {
	State  *engine.StoryCreatorState `json:"state,omitempty"`
	Input  string                    `json:"input,omitempty"`
	Action string                    `json:"action,omitempty"`
	Start  bool                      `json:"start"`
}
type StoryEnhanceRequest struct {
	Stage   string                    `json:"stage,omitempty"`
	Text    string                    `json:"text,omitempty"`
	Context string                    `json:"context,omitempty"`
	State   *engine.StoryCreatorState `json:"state,omitempty"`
}
type StoryCreateResponse struct {
	ResponseMeta
	StoryID     string `json:"story_id,omitempty"`
	CharacterID string `json:"character_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	Started     bool   `json:"started,omitempty"`
	StartError  string `json:"start_error,omitempty"`
	Error       string `json:"error,omitempty"`
}
type StoryWizardResponse struct {
	ResponseMeta
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
type StoryEnhanceResponse struct {
	ResponseMeta
	Text      string `json:"text,omitempty"`
	Model     string `json:"model,omitempty"`
	Provider  string `json:"provider,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}
type ModelSettingsResponse struct {
	ResponseMeta
	Settings  *config.ModelRoutingSettings `json:"settings,omitempty"`
	Error     string                       `json:"error,omitempty"`
	ErrorCode string                       `json:"error_code,omitempty"`
}
type SchemaPreflightResponse struct {
	ResponseMeta
	Status string `json:"status"`
}
type MiniGameRequest struct {
	StoryID    string                          `json:"story_id"`
	InstanceID string                          `json:"instance_id,omitempty"`
	Kind       engine.MiniGameType             `json:"kind,omitempty"`
	Definition engine.MiniGameDefinition       `json:"definition,omitempty"`
	Input      engine.MiniGameInput            `json:"input,omitempty"`
	Selection  engine.MiniGameSelectionContext `json:"selection,omitempty"`
}
type MiniGameResponse struct {
	ResponseMeta
	Instance *engine.MiniGameInstance `json:"instance,omitempty"`
	Error    string                   `json:"error,omitempty"`
}
type AudioRequest struct {
	Operation       string                      `json:"operation"`
	StoryID         string                      `json:"story_id,omitempty"`
	MessageID       int64                       `json:"message_id,omitempty"`
	AssetID         string                      `json:"asset_id,omitempty"`
	AssignmentID    string                      `json:"assignment_id,omitempty"`
	PronunciationID string                      `json:"pronunciation_id,omitempty"`
	JobID           string                      `json:"job_id,omitempty"`
	Provider        string                      `json:"provider,omitempty"`
	Language        string                      `json:"language,omitempty"`
	ClientRevision  int64                       `json:"client_revision,omitempty"`
	Settings        *storage.StoryTTSSettings   `json:"settings,omitempty"`
	Assignment      *storage.VoiceAssignment    `json:"assignment,omitempty"`
	Pronunciation   *storage.PronunciationEntry `json:"pronunciation,omitempty"`
	DryRun          bool                        `json:"dry_run,omitempty"`
}
type AudioExport struct {
	Format         string                        `json:"format"`
	Filename       string                        `json:"filename"`
	GeneratedAt    string                        `json:"generated_at"`
	StoryID        string                        `json:"story_id"`
	Settings       *storage.StoryTTSSettings     `json:"settings"`
	Providers      []audioservice.ProviderStatus `json:"providers"`
	Voices         []storage.VoiceProfile        `json:"voices"`
	Assignments    []storage.VoiceAssignment     `json:"assignments"`
	Pronunciations []storage.PronunciationEntry  `json:"pronunciations"`
	Assets         []storage.AudioAsset          `json:"assets"`
	Jobs           []storage.TTSJob              `json:"jobs"`
}
type AudioResponse struct {
	ResponseMeta
	Statuses       []audioservice.ProviderStatus `json:"providers,omitempty"`
	Profiles       []storage.VoiceProfile        `json:"voices,omitempty"`
	Settings       *storage.StoryTTSSettings     `json:"settings,omitempty"`
	Assignments    []storage.VoiceAssignment     `json:"assignments,omitempty"`
	Assignment     *storage.VoiceAssignment      `json:"assignment,omitempty"`
	Pronunciations []storage.PronunciationEntry  `json:"pronunciations,omitempty"`
	Pronunciation  *storage.PronunciationEntry   `json:"pronunciation,omitempty"`
	Assets         []storage.AudioAsset          `json:"assets,omitempty"`
	Jobs           []storage.TTSJob              `json:"jobs,omitempty"`
	Asset          *storage.AudioAsset           `json:"asset,omitempty"`
	FilePath       string                        `json:"file_path,omitempty"`
	Format         string                        `json:"format,omitempty"`
	Cleanup        *audioservice.CleanupResult   `json:"cleanup,omitempty"`
	Export         *AudioExport                  `json:"export,omitempty"`
	Error          string                        `json:"error,omitempty"`
}
