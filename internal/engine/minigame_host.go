package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const MiniGameHostProtocolVersion = 1

type MiniGamePhase string

const (
	MiniGameReady    MiniGamePhase = "ready"
	MiniGameActive   MiniGamePhase = "active"
	MiniGamePaused   MiniGamePhase = "paused"
	MiniGameResolved MiniGamePhase = "resolved"
)

// MiniGameDefinition is authorable data. Reducers own mechanics; story packs
// supply framing, difficulty, options, and explicit answer/sequence material.
type MiniGameDefinition struct {
	ID          string            `json:"id" yaml:"id"`
	Kind        MiniGameType      `json:"kind" yaml:"kind"`
	Prompt      string            `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Difficulty  int               `json:"difficulty" yaml:"difficulty"`
	Options     []string          `json:"options,omitempty" yaml:"options,omitempty"`
	Sequence    []string          `json:"sequence,omitempty" yaml:"sequence,omitempty"`
	Answers     []string          `json:"answers,omitempty" yaml:"answers,omitempty"`
	TimeLimitMS int64             `json:"time_limit_ms,omitempty" yaml:"time_limit_ms,omitempty"`
	Rules       map[string]string `json:"rules,omitempty" yaml:"rules,omitempty"`
}

type MiniGameInput struct {
	Action    string   `json:"action"`
	Value     string   `json:"value,omitempty"`
	Values    []string `json:"values,omitempty"`
	ElapsedMS int64    `json:"elapsed_ms,omitempty"`
}

type MiniGameRuntime struct {
	Phase    MiniGamePhase    `json:"phase"`
	Revision int              `json:"revision"`
	State    json.RawMessage  `json:"state,omitempty"`
	History  []MiniGameInput  `json:"history,omitempty"`
	Result   *ChallengeResult `json:"result,omitempty"`
}

type MiniGameInstance struct {
	ProtocolVersion int                `json:"protocol_version"`
	ID              string             `json:"id"`
	StoryID         string             `json:"story_id,omitempty"`
	BranchID        string             `json:"branch_id,omitempty"`
	Turn            int                `json:"turn"`
	Seed            int64              `json:"seed"`
	Definition      MiniGameDefinition `json:"definition"`
	Runtime         MiniGameRuntime    `json:"runtime"`
}

func (instance MiniGameInstance) Validate() error {
	if instance.ProtocolVersion != MiniGameHostProtocolVersion {
		return fmt.Errorf("unsupported minigame protocol version %d", instance.ProtocolVersion)
	}
	if strings.TrimSpace(instance.ID) == "" || strings.TrimSpace(instance.Definition.ID) == "" {
		return errors.New("minigame instance and definition ids are required")
	}
	if strings.TrimSpace(string(instance.Definition.Kind)) == "" {
		return errors.New("minigame kind is required")
	}
	if instance.Definition.Difficulty < 1 || instance.Definition.Difficulty > 100 {
		return fmt.Errorf("minigame difficulty %d is outside 1..100", instance.Definition.Difficulty)
	}
	if instance.Seed < 0 {
		return errors.New("minigame seed must not be negative")
	}
	return nil
}

type MiniGameReducer interface {
	Initialize(MiniGameDefinition, int64) (json.RawMessage, error)
	Reduce(MiniGameDefinition, int64, json.RawMessage, MiniGameInput) (json.RawMessage, *ChallengeResult, error)
}

type MiniGameHost struct {
	reducers map[MiniGameType]MiniGameReducer
}

func NewMiniGameHost() *MiniGameHost {
	host := &MiniGameHost{reducers: map[MiniGameType]MiniGameReducer{}}
	legacy := legacyMiniGameReducer{}
	for _, kind := range []MiniGameType{MiniGameRPS, MiniGameMemory, MiniGameQuickTime, MiniGameRiddle} {
		host.Register(kind, legacy)
	}
	families := genreNeutralMiniGameReducer{}
	for _, kind := range []MiniGameType{MiniGameDeduction, MiniGameNegotiation, MiniGamePattern, MiniGameBidding, MiniGameCourtroom, MiniGameComedy} {
		host.Register(kind, families)
	}
	return host
}

func (host *MiniGameHost) Register(kind MiniGameType, reducer MiniGameReducer) {
	if host.reducers == nil {
		host.reducers = map[MiniGameType]MiniGameReducer{}
	}
	host.reducers[kind] = reducer
}

func NewMiniGameInstance(id, storyID, branchID string, turn int, seed int64, definition MiniGameDefinition) MiniGameInstance {
	if definition.Difficulty == 0 {
		definition.Difficulty = 50
	}
	return MiniGameInstance{
		ProtocolVersion: MiniGameHostProtocolVersion,
		ID:              id, StoryID: storyID, BranchID: branchID, Turn: turn, Seed: seed,
		Definition: definition,
		Runtime:    MiniGameRuntime{Phase: MiniGameReady},
	}
}

func (host *MiniGameHost) Start(instance *MiniGameInstance) error {
	if instance == nil {
		return errors.New("minigame instance is required")
	}
	if err := instance.Validate(); err != nil {
		return err
	}
	if instance.Runtime.Phase != MiniGameReady {
		return fmt.Errorf("minigame cannot start from %s", instance.Runtime.Phase)
	}
	reducer, ok := host.reducers[instance.Definition.Kind]
	if !ok {
		return fmt.Errorf("minigame reducer %q is not registered", instance.Definition.Kind)
	}
	state, err := reducer.Initialize(instance.Definition, instance.Seed)
	if err != nil {
		return err
	}
	instance.Runtime = MiniGameRuntime{
		Phase: MiniGameActive, Revision: 1, State: state,
		History: []MiniGameInput{{Action: "start"}},
	}
	return nil
}

func (host *MiniGameHost) Apply(instance *MiniGameInstance, input MiniGameInput) error {
	if instance == nil {
		return errors.New("minigame instance is required")
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	input.Action = action
	switch action {
	case "pause":
		if instance.Runtime.Phase != MiniGameActive {
			return fmt.Errorf("minigame cannot pause from %s", instance.Runtime.Phase)
		}
		instance.Runtime.Phase = MiniGamePaused
	case "resume":
		if instance.Runtime.Phase != MiniGamePaused {
			return fmt.Errorf("minigame cannot resume from %s", instance.Runtime.Phase)
		}
		instance.Runtime.Phase = MiniGameActive
	default:
		if instance.Runtime.Phase != MiniGameActive {
			return fmt.Errorf("minigame input is not accepted in %s", instance.Runtime.Phase)
		}
		reducer, ok := host.reducers[instance.Definition.Kind]
		if !ok {
			return fmt.Errorf("minigame reducer %q is not registered", instance.Definition.Kind)
		}
		state, result, err := reducer.Reduce(instance.Definition, instance.Seed, instance.Runtime.State, input)
		if err != nil {
			return err
		}
		instance.Runtime.State = state
		if result != nil {
			instance.Runtime.Result = EnsureLegacyChallengeOutcome(result)
			instance.Runtime.Phase = MiniGameResolved
		}
	}
	instance.Runtime.Revision++
	instance.Runtime.History = append(instance.Runtime.History, input)
	return nil
}

func (host *MiniGameHost) Serialize(instance MiniGameInstance) ([]byte, error) {
	if err := instance.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(instance)
}

func (host *MiniGameHost) Restore(payload []byte) (*MiniGameInstance, error) {
	var instance MiniGameInstance
	if err := json.Unmarshal(payload, &instance); err != nil {
		return nil, fmt.Errorf("decoding minigame instance: %w", err)
	}
	if err := instance.Validate(); err != nil {
		return nil, err
	}
	if _, ok := host.reducers[instance.Definition.Kind]; !ok {
		return nil, fmt.Errorf("minigame reducer %q is not registered", instance.Definition.Kind)
	}
	return &instance, nil
}

func (host *MiniGameHost) Replay(source MiniGameInstance) (*MiniGameInstance, error) {
	history := append([]MiniGameInput(nil), source.Runtime.History...)
	replay := NewMiniGameInstance(source.ID, source.StoryID, source.BranchID, source.Turn, source.Seed, source.Definition)
	for _, input := range history {
		if input.Action == "start" {
			if err := host.Start(&replay); err != nil {
				return nil, err
			}
			continue
		}
		if err := host.Apply(&replay, input); err != nil {
			return nil, err
		}
	}
	return &replay, nil
}

func (host *MiniGameHost) Autoplay(instance *MiniGameInstance, input MiniGameInput) error {
	if instance.Runtime.Phase == MiniGameReady {
		if err := host.Start(instance); err != nil {
			return err
		}
	}
	if instance.Runtime.Phase == MiniGamePaused {
		if err := host.Apply(instance, MiniGameInput{Action: "resume"}); err != nil {
			return err
		}
	}
	input.Action = "submit"
	return host.Apply(instance, input)
}

// PlayerMiniGameView removes authoritative answers and hidden reducer state
// while retaining the same public instance/revision contract for renderers.
func PlayerMiniGameView(source MiniGameInstance) MiniGameInstance {
	view := source
	view.Definition.Answers = nil
	if len(view.Definition.Rules) > 0 {
		view.Definition.Rules = map[string]string{}
		for key, value := range source.Definition.Rules {
			if key != "reserve" && !strings.HasPrefix(key, "secret_") {
				view.Definition.Rules[key] = value
			}
		}
	}
	switch source.Definition.Kind {
	case MiniGameRiddle, MiniGameDeduction, MiniGamePattern, MiniGameBidding:
		view.Runtime.State = json.RawMessage(`{}`)
	}
	return view
}

type legacyMiniGameState struct {
	Sequence    []string `json:"sequence,omitempty"`
	Answers     []string `json:"answers,omitempty"`
	TimeLimitMS int64    `json:"time_limit_ms,omitempty"`
}

type legacyMiniGameReducer struct{}

func (legacyMiniGameReducer) Initialize(definition MiniGameDefinition, seed int64) (json.RawMessage, error) {
	state := legacyMiniGameState{}
	switch definition.Kind {
	case MiniGameRPS:
	case MiniGameMemory:
		state.Sequence = append([]string(nil), definition.Sequence...)
		if len(state.Sequence) == 0 {
			length := 4
			if definition.Difficulty >= 70 {
				length = 6
			}
			state.Sequence = NewMemoryChallengeWithRNG(nil, length, NewRNGService(seed)).Sequence
		}
	case MiniGameQuickTime:
		state.TimeLimitMS = definition.TimeLimitMS
		if state.TimeLimitMS <= 0 {
			state.TimeLimitMS = 3000
		}
	case MiniGameRiddle:
		state.Answers = append([]string(nil), definition.Answers...)
		if len(state.Answers) == 0 {
			return nil, errors.New("riddle minigame requires an explicit answer")
		}
	default:
		return nil, fmt.Errorf("legacy reducer cannot initialize %q", definition.Kind)
	}
	return json.Marshal(state)
}

func (legacyMiniGameReducer) Reduce(definition MiniGameDefinition, seed int64, payload json.RawMessage, input MiniGameInput) (json.RawMessage, *ChallengeResult, error) {
	if input.Action != "submit" {
		return payload, nil, fmt.Errorf("minigame action %q is not supported", input.Action)
	}
	var state legacyMiniGameState
	if err := json.Unmarshal(payload, &state); err != nil {
		return payload, nil, fmt.Errorf("decoding legacy minigame state: %w", err)
	}
	var result *ChallengeResult
	switch definition.Kind {
	case MiniGameRPS:
		choice := RPSChoice(strings.ToLower(strings.TrimSpace(input.Value)))
		if choice != RPSRock && choice != RPSPaper && choice != RPSScissors {
			return payload, nil, errors.New("rps choice must be rock, paper, or scissors")
		}
		resolved := ResolveRPSWithRNG(choice, NewRNGService(seed))
		result = RPSToChallengeResult(resolved)
	case MiniGameMemory:
		result = NewMemoryChallenge(state.Sequence, 0).CheckMemory(input.Values)
	case MiniGameQuickTime:
		challenge := NewQuickTimeChallenge(float64(state.TimeLimitMS) / 1000)
		result = challenge.CheckQuickTimeElapsed(time.Duration(input.ElapsedMS) * time.Millisecond)
	case MiniGameRiddle:
		challenge := NewRiddleChallenge(definition.Prompt, state.Answers[0], state.Answers[1:]...)
		result = challenge.CheckRiddle(input.Value)
	default:
		return payload, nil, fmt.Errorf("legacy reducer cannot resolve %q", definition.Kind)
	}
	return payload, result, nil
}
