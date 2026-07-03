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
	Save   BrowserSaveView `json:"save"`
	Legacy bool            `json:"legacy,omitempty"`
}

type BrowserDeleteSaveResponse struct {
	Save BrowserSaveView `json:"save"`
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
