package contracts

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCanonicalChallengeFixtureUsesCurrentProtocol(t *testing.T) {
	raw, err := os.ReadFile("../../../contracts/challenge-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Instance   ChallengeInstance   `json:"instance"`
		Input      ChallengeInput      `json:"input"`
		Resolution ChallengeResolution `json:"resolution"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Instance.Validate(); err != nil {
		t.Fatal(err)
	}
	if fixture.Resolution.InstanceID != fixture.Instance.ID || fixture.Resolution.Outcome.Seed != fixture.Instance.Seed {
		t.Fatalf("fixture lineage mismatch: %+v", fixture)
	}
}

func TestOutcomeEnvelopeRoundTripPreservesAuthoritativeFields(t *testing.T) {
	outcome := OutcomeEnvelope{
		Version:          ChallengeProtocolVersion,
		Degree:           OutcomeSuccessWithCost,
		Difficulty:       62,
		Seed:             44,
		Roll:             57,
		Margin:           -5,
		Costs:            []OutcomeEffect{{Kind: "time", Amount: 10, Detail: "A slower route"}},
		Consequences:     []string{"The patrol notices the delay"},
		StateDeltas:      []OutcomeEffect{{Kind: "progress", Amount: 1}},
		RevealedFacts:    []string{"The side gate is watched"},
		FollowUpPressure: 2,
	}
	raw, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	var decoded OutcomeEnvelope
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Degree != OutcomeSuccessWithCost || decoded.Margin != -5 || decoded.Seed != 44 || len(decoded.Costs) != 1 {
		t.Fatalf("round trip lost authoritative fields: %+v", decoded)
	}
}

func TestChallengeInstanceValidateRejectsUnsupportedProtocol(t *testing.T) {
	instance := ChallengeInstance{ProtocolVersion: 99, ID: "challenge-1", Definition: ChallengeDefinition{ID: "ordinary-action", Kind: "action"}}
	if err := instance.Validate(); err == nil {
		t.Fatal("Validate error = nil, want unsupported protocol error")
	}
}

func TestSubmitActionRejectsStaleClientTurn(t *testing.T) {
	req := SubmitActionRequest{
		StoryID:        "story-1",
		SessionID:      "session-1",
		ClientTurn:     3,
		ClientRevision: 7,
		IdempotencyKey: "idem-1",
		Action:         PlayerAction{Kind: ActionKindFreeText, Text: "Open the door"},
	}
	if err := req.Validate(4, 7); err == nil {
		t.Fatal("Validate error = nil, want stale client_turn error")
	}
}

func TestSubmitActionRejectsStaleClientRevision(t *testing.T) {
	req := SubmitActionRequest{
		StoryID:        "story-1",
		SessionID:      "session-1",
		ClientTurn:     4,
		ClientRevision: 6,
		IdempotencyKey: "idem-1",
		Action:         PlayerAction{Kind: ActionKindFreeText, Text: "Open the door"},
	}
	if err := req.Validate(4, 7); err == nil {
		t.Fatal("Validate error = nil, want stale client_revision error")
	}
}

func TestSubmitActionRequiresIdempotencyKey(t *testing.T) {
	req := SubmitActionRequest{
		StoryID:        "story-1",
		SessionID:      "session-1",
		ClientTurn:     4,
		ClientRevision: 7,
		Action:         PlayerAction{Kind: ActionKindFreeText, Text: "Open the door"},
	}
	if err := req.Validate(4, 7); err == nil {
		t.Fatal("Validate error = nil, want idempotency_key error")
	}
}

func TestTurnEventEnvelopeCarriesPayload(t *testing.T) {
	evt, err := NewTurnEvent("evt-1", "story-1", "session-1", 4, EventNarrativeFinal, map[string]string{"text": "Done"})
	if err != nil {
		t.Fatalf("NewTurnEvent: %v", err)
	}
	if evt.ID != "evt-1" || evt.Type != EventNarrativeFinal || len(evt.Payload) == 0 {
		t.Fatalf("event envelope incomplete: %+v", evt)
	}
}

func TestCommandDescriptorsCoverBrowserCriticalCommands(t *testing.T) {
	descriptors := CommandDescriptors()
	byID := map[string]CommandDescriptor{}
	for _, descriptor := range descriptors {
		if descriptor.ID == "" || descriptor.Canonical == "" || descriptor.Title == "" {
			t.Fatalf("incomplete command descriptor: %+v", descriptor)
		}
		byID[descriptor.ID] = descriptor
	}

	for _, id := range []string{"talk", "btw", "guide", "narrator", "advance", "timeskip", "save", "load", "delete-save"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing command descriptor %q", id)
		}
	}
	if got := byID["delete-save"].Behavior; got != CommandBehaviorSaveDelete {
		t.Fatalf("delete-save behavior = %q, want %q", got, CommandBehaviorSaveDelete)
	}
	if got := byID["delete-save"].Parity; got != CommandParityBrowserOnly {
		t.Fatalf("delete-save parity = %q, want %q", got, CommandParityBrowserOnly)
	}
}

func TestCommandAliasRegistryExcludesBrowserOnlyCommands(t *testing.T) {
	registry := CommandAliasRegistry()
	if got := registry["fronts"]; got != "hooks" {
		t.Fatalf("fronts alias = %q, want hooks", got)
	}
	if got := registry["saves"]; got != "load" {
		t.Fatalf("saves alias = %q, want load", got)
	}
	if got := registry["delete-save"]; got != "" {
		t.Fatalf("delete-save alias = %q, want excluded browser-only command", got)
	}
}
