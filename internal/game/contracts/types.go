package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ActionKind string

const (
	ActionKindChoice   ActionKind = "choice"
	ActionKindFreeText ActionKind = "free_text"
	ActionKindCommand  ActionKind = "command"
)

type PlayerAction struct {
	Kind     ActionKind `json:"kind"`
	Text     string     `json:"text,omitempty"`
	ChoiceID int        `json:"choice_id,omitempty"`
}

type ClientCapabilities struct {
	Images  bool `json:"images,omitempty"`
	ASCII   bool `json:"ascii,omitempty"`
	RollLog bool `json:"roll_log,omitempty"`
}

const (
	ChallengeProtocolVersion       = 1
	MaxPortableChallengeSeed int64 = (1 << 53) - 1
)

type OutcomeDegree string

const (
	OutcomeCriticalSuccess     OutcomeDegree = "critical_success"
	OutcomeFullSuccess         OutcomeDegree = "full_success"
	OutcomeSuccessWithCost     OutcomeDegree = "success_with_cost"
	OutcomeFailureWithProgress OutcomeDegree = "failure_with_progress"
	OutcomeHardFailure         OutcomeDegree = "hard_failure"
	OutcomeCatastrophe         OutcomeDegree = "catastrophe"
)

func ValidOutcomeDegree(degree OutcomeDegree) bool {
	switch degree {
	case OutcomeCriticalSuccess, OutcomeFullSuccess, OutcomeSuccessWithCost,
		OutcomeFailureWithProgress, OutcomeHardFailure, OutcomeCatastrophe:
		return true
	default:
		return false
	}
}

type ChallengeModifier struct {
	Source string `json:"source"`
	Value  int    `json:"value"`
}

type TimingPolicy struct {
	Mode          string `json:"mode,omitempty"`
	LimitMillis   int64  `json:"limit_ms,omitempty"`
	StartedUnixMs int64  `json:"started_unix_ms,omitempty"`
}

type OutcomePolicy struct {
	ID                string `json:"id" yaml:"id"`
	Genre             string `json:"genre,omitempty" yaml:"genre,omitempty"`
	DifficultyProfile string `json:"difficulty_profile,omitempty" yaml:"difficulty_profile,omitempty"`
	ConsequenceBudget int    `json:"consequence_budget" yaml:"consequence_budget"`
	CriticalBand      int    `json:"critical_band" yaml:"critical_band"`
	Fairness          string `json:"fairness" yaml:"fairness"`
}

type ChallengeDefinition struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Description string            `json:"description,omitempty"`
	Difficulty  int               `json:"difficulty"`
	Rules       map[string]string `json:"rules,omitempty"`
}

type ChallengeInstance struct {
	ProtocolVersion int                 `json:"protocol_version"`
	ID              string              `json:"id"`
	StoryID         string              `json:"story_id,omitempty"`
	BranchID        string              `json:"branch_id,omitempty"`
	Turn            int                 `json:"turn"`
	Definition      ChallengeDefinition `json:"definition"`
	Seed            int64               `json:"seed"`
	Policy          OutcomePolicy       `json:"policy"`
	Timing          TimingPolicy        `json:"timing,omitempty"`
}

func (i ChallengeInstance) Validate() error {
	if i.ProtocolVersion != ChallengeProtocolVersion {
		return fmt.Errorf("unsupported challenge protocol version %d", i.ProtocolVersion)
	}
	if strings.TrimSpace(i.ID) == "" || strings.TrimSpace(i.Definition.ID) == "" || strings.TrimSpace(i.Definition.Kind) == "" {
		return errors.New("challenge instance id, definition id, and kind are required")
	}
	if i.Definition.Difficulty < 1 || i.Definition.Difficulty > 100 {
		return fmt.Errorf("challenge difficulty %d is outside 1..100", i.Definition.Difficulty)
	}
	if i.Seed < 0 || i.Seed > MaxPortableChallengeSeed {
		return fmt.Errorf("challenge seed %d is not portable across JSON hosts", i.Seed)
	}
	return nil
}

type ChallengeInput struct {
	ActorID   string              `json:"actor_id,omitempty"`
	Intent    string              `json:"intent"`
	ChoiceID  int                 `json:"choice_id,omitempty"`
	Modifiers []ChallengeModifier `json:"modifiers,omitempty"`
	Payload   json.RawMessage     `json:"payload,omitempty"`
	ElapsedMS int64               `json:"elapsed_ms,omitempty"`
}

type OutcomeEffect struct {
	Kind   string `json:"kind"`
	Target string `json:"target,omitempty"`
	Field  string `json:"field,omitempty"`
	Amount int    `json:"amount,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type OutcomeEnvelope struct {
	Version          int                 `json:"version"`
	Degree           OutcomeDegree       `json:"degree"`
	Difficulty       int                 `json:"difficulty"`
	Seed             int64               `json:"seed"`
	Roll             int                 `json:"roll"`
	Modifiers        []ChallengeModifier `json:"modifiers,omitempty"`
	Total            int                 `json:"total"`
	Margin           int                 `json:"margin"`
	Costs            []OutcomeEffect     `json:"costs,omitempty"`
	Consequences     []string            `json:"consequences,omitempty"`
	StateDeltas      []OutcomeEffect     `json:"state_deltas,omitempty"`
	RevealedFacts    []string            `json:"revealed_facts,omitempty"`
	FollowUpPressure int                 `json:"follow_up_pressure,omitempty"`
}

func (o OutcomeEnvelope) Succeeded() bool {
	return o.Degree == OutcomeCriticalSuccess || o.Degree == OutcomeFullSuccess || o.Degree == OutcomeSuccessWithCost
}

type ChallengeResolution struct {
	ProtocolVersion int             `json:"protocol_version"`
	InstanceID      string          `json:"instance_id"`
	Input           ChallengeInput  `json:"input"`
	Outcome         OutcomeEnvelope `json:"outcome"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
}

type SubmitActionRequest struct {
	StoryID        string             `json:"story_id"`
	SessionID      string             `json:"session_id"`
	ClientTurn     int                `json:"client_turn"`
	ClientRevision int64              `json:"client_revision"`
	IdempotencyKey string             `json:"idempotency_key"`
	Action         PlayerAction       `json:"action"`
	Stream         bool               `json:"stream,omitempty"`
	Capabilities   ClientCapabilities `json:"capabilities,omitempty"`
}

func (r SubmitActionRequest) Validate(currentTurn int, currentRevision int64) error {
	if strings.TrimSpace(r.StoryID) == "" {
		return errors.New("story_id is required")
	}
	if strings.TrimSpace(r.SessionID) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		return errors.New("idempotency_key is required")
	}
	if r.ClientTurn != currentTurn {
		return fmt.Errorf("stale client_turn %d, current turn is %d", r.ClientTurn, currentTurn)
	}
	if currentRevision >= 0 && r.ClientRevision != currentRevision {
		return fmt.Errorf("stale client_revision %d, current revision is %d", r.ClientRevision, currentRevision)
	}
	if strings.TrimSpace(r.Action.Text) == "" && r.Action.ChoiceID == 0 {
		return errors.New("action text or choice_id is required")
	}
	return nil
}

type BrowserMetaKind string

const (
	BrowserMetaKindBTW      BrowserMetaKind = "btw"
	BrowserMetaKindGuide    BrowserMetaKind = "guide"
	BrowserMetaKindNarrator BrowserMetaKind = "narrator"
)

type BrowserMetaRequest struct {
	StoryID        string          `json:"story_id"`
	SessionID      string          `json:"session_id"`
	ClientTurn     int             `json:"client_turn"`
	ClientRevision int64           `json:"client_revision"`
	Kind           BrowserMetaKind `json:"kind"`
	Text           string          `json:"text,omitempty"`
}

func (r BrowserMetaRequest) Validate(currentTurn int, currentRevision int64) error {
	if err := validateStorySessionTurnRevision(r.StoryID, r.SessionID, r.ClientTurn, r.ClientRevision, currentTurn, currentRevision); err != nil {
		return err
	}
	switch r.Kind {
	case BrowserMetaKindBTW, BrowserMetaKindGuide, BrowserMetaKindNarrator:
		return nil
	default:
		return fmt.Errorf("unsupported browser meta kind %q", r.Kind)
	}
}

type BrowserMetaResponse struct {
	Kind    BrowserMetaKind `json:"kind"`
	Title   string          `json:"title"`
	Message string          `json:"message"`
}

type BrowserSaveRequest struct {
	StoryID        string `json:"story_id"`
	SessionID      string `json:"session_id"`
	ClientTurn     int    `json:"client_turn"`
	ClientRevision int64  `json:"client_revision"`
	Name           string `json:"name,omitempty"`
	Kind           string `json:"kind,omitempty"`
}

func (r BrowserSaveRequest) Validate(currentTurn int, currentRevision int64) error {
	return validateStorySessionTurnRevision(r.StoryID, r.SessionID, r.ClientTurn, r.ClientRevision, currentTurn, currentRevision)
}

type BrowserLoadRequest struct {
	StoryID        string `json:"story_id"`
	SessionID      string `json:"session_id"`
	ClientTurn     int    `json:"client_turn"`
	ClientRevision int64  `json:"client_revision"`
	SaveID         string `json:"save_id"`
}

func (r BrowserLoadRequest) Validate(currentTurn int, currentRevision int64) error {
	if err := validateStorySessionTurnRevision(r.StoryID, r.SessionID, r.ClientTurn, r.ClientRevision, currentTurn, currentRevision); err != nil {
		return err
	}
	if strings.TrimSpace(r.SaveID) == "" {
		return errors.New("save_id is required")
	}
	return nil
}

type BrowserDeleteSaveRequest struct {
	StoryID        string `json:"story_id"`
	SessionID      string `json:"session_id"`
	ClientTurn     int    `json:"client_turn"`
	ClientRevision int64  `json:"client_revision"`
	SaveID         string `json:"save_id"`
}

func (r BrowserDeleteSaveRequest) Validate(currentTurn int, currentRevision int64) error {
	if err := validateStorySessionTurnRevision(r.StoryID, r.SessionID, r.ClientTurn, r.ClientRevision, currentTurn, currentRevision); err != nil {
		return err
	}
	if strings.TrimSpace(r.SaveID) == "" {
		return errors.New("save_id is required")
	}
	return nil
}

type BrowserSaveView struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Turn      int             `json:"turn"`
	Chapter   int             `json:"chapter"`
	Location  string          `json:"location,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type BrowserSaveResponse struct {
	Save BrowserSaveView `json:"save"`
}

type BrowserLoadResponse struct {
	Save           BrowserSaveView `json:"save"`
	Legacy         bool            `json:"legacy,omitempty"`
	SnapshotState  string          `json:"snapshot_state"`
	SnapshotDetail string          `json:"snapshot_detail,omitempty"`
}

type BrowserDeleteSaveResponse struct {
	Save BrowserSaveView `json:"save"`
}

type TimelineAction string

const (
	TimelineList     TimelineAction = "list"
	TimelineFork     TimelineAction = "fork"
	TimelineRename   TimelineAction = "rename"
	TimelineCheckout TimelineAction = "checkout"
)

type BrowserTimelineRequest struct {
	StoryID        string         `json:"story_id"`
	Action         TimelineAction `json:"action"`
	ClientRevision int64          `json:"client_revision"`
	BranchID       string         `json:"branch_id,omitempty"`
	FromCommitID   string         `json:"from_commit_id,omitempty"`
	Name           string         `json:"name,omitempty"`
}

func (r BrowserTimelineRequest) Validate() error {
	if strings.TrimSpace(r.StoryID) == "" {
		return errors.New("story_id is required")
	}
	switch r.Action {
	case TimelineList:
		return nil
	case TimelineFork:
		if strings.TrimSpace(r.FromCommitID) == "" || strings.TrimSpace(r.Name) == "" {
			return errors.New("fork requires from_commit_id and name")
		}
	case TimelineRename:
		if strings.TrimSpace(r.BranchID) == "" || strings.TrimSpace(r.Name) == "" {
			return errors.New("rename requires branch_id and name")
		}
	case TimelineCheckout:
		if strings.TrimSpace(r.BranchID) == "" {
			return errors.New("checkout requires branch_id")
		}
	default:
		return fmt.Errorf("unsupported timeline action %q", r.Action)
	}
	return nil
}

type BrowserTimelineResponse struct {
	ActiveBranchID string               `json:"active_branch_id"`
	Revision       int64                `json:"revision"`
	Branches       []TimelineBranchView `json:"branches"`
	Head           *TimelineCommitView  `json:"head,omitempty"`
	Commits        []TimelineCommitView `json:"commits"`
}

// Wire views keep the public contract independent of storage implementation.
type TimelineBranchView struct {
	ID           string    `json:"id"`
	StoryID      string    `json:"story_id"`
	Name         string    `json:"name"`
	ForkCommitID string    `json:"fork_commit_id,omitempty"`
	HeadCommitID string    `json:"head_commit_id"`
	HeadTurn     int       `json:"head_turn"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TimelineCommitView struct {
	ID             string    `json:"id"`
	BranchID       string    `json:"branch_id"`
	ParentCommitID string    `json:"parent_commit_id,omitempty"`
	CanonicalTurn  int       `json:"canonical_turn"`
	Kind           string    `json:"kind"`
	Message        string    `json:"message,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func validateStorySessionTurn(storyID, sessionID string, clientTurn, currentTurn int) error {
	return validateStorySessionTurnRevision(storyID, sessionID, clientTurn, 0, currentTurn, -1)
}

func validateStorySessionTurnRevision(storyID, sessionID string, clientTurn int, clientRevision int64, currentTurn int, currentRevision int64) error {
	if strings.TrimSpace(storyID) == "" {
		return errors.New("story_id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("session_id is required")
	}
	if clientTurn != currentTurn {
		return fmt.Errorf("stale client_turn %d, current turn is %d", clientTurn, currentTurn)
	}
	if currentRevision >= 0 && clientRevision != currentRevision {
		return fmt.Errorf("stale client_revision %d, current revision is %d", clientRevision, currentRevision)
	}
	return nil
}

type TurnEventType string

const (
	EventTurnStarted       TurnEventType = "turn.started"
	EventNarrativeDelta    TurnEventType = "narrative.delta"
	EventNarrativeFinal    TurnEventType = "narrative.final"
	EventChoicesUpdated    TurnEventType = "choices.updated"
	EventStateDelta        TurnEventType = "state.delta"
	EventRollResolved      TurnEventType = "roll.resolved"
	EventChallengeStarted  TurnEventType = "challenge.started"
	EventChallengeResolved TurnEventType = "challenge.resolved"
	EventCombatStarted     TurnEventType = "combat.started"
	EventCombatUpdated     TurnEventType = "combat.updated"
	EventSocialStarted     TurnEventType = "social.started"
	EventSocialUpdated     TurnEventType = "social.updated"
	EventAssetQueued       TurnEventType = "asset.queued"
	EventAssetReady        TurnEventType = "asset.ready"
	EventAssetFailed       TurnEventType = "asset.failed"
	EventTurnCommitted     TurnEventType = "turn.committed"
	EventError             TurnEventType = "error"
)

type TurnEvent struct {
	ID        string          `json:"id"`
	StoryID   string          `json:"story_id"`
	SessionID string          `json:"session_id"`
	Turn      int             `json:"turn"`
	Type      TurnEventType   `json:"type"`
	CreatedAt time.Time       `json:"created_at"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

func NewTurnEvent(id, storyID, sessionID string, turn int, eventType TurnEventType, payload any) (TurnEvent, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return TurnEvent{}, err
	}
	return TurnEvent{
		ID:        id,
		StoryID:   storyID,
		SessionID: sessionID,
		Turn:      turn,
		Type:      eventType,
		CreatedAt: time.Now().UTC(),
		Payload:   raw,
	}, nil
}

type ChoiceView struct {
	ID           int      `json:"id"`
	Text         string   `json:"text"`
	Intent       string   `json:"intent,omitempty"`
	Risk         string   `json:"risk,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	Certainty    string   `json:"certainty,omitempty"`
	RelatedStats []string `json:"related_stats,omitempty"`
}

type StateDelta struct {
	Target      string `json:"target"`
	Field       string `json:"field"`
	Description string `json:"description,omitempty"`
}

type GameSnapshot struct {
	StoryID   string       `json:"story_id"`
	SessionID string       `json:"session_id"`
	Turn      int          `json:"turn"`
	Revision  int64        `json:"revision"`
	Location  string       `json:"location,omitempty"`
	Choices   []ChoiceView `json:"choices,omitempty"`
}
