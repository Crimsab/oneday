package contracts

import "testing"

func TestSubmitActionRejectsStaleClientTurn(t *testing.T) {
	req := SubmitActionRequest{
		StoryID:        "story-1",
		SessionID:      "session-1",
		ClientTurn:     3,
		IdempotencyKey: "idem-1",
		Action:         PlayerAction{Kind: ActionKindFreeText, Text: "Open the door"},
	}
	if err := req.Validate(4); err == nil {
		t.Fatal("Validate error = nil, want stale client_turn error")
	}
}

func TestSubmitActionRequiresIdempotencyKey(t *testing.T) {
	req := SubmitActionRequest{
		StoryID:    "story-1",
		SessionID:  "session-1",
		ClientTurn: 4,
		Action:     PlayerAction{Kind: ActionKindFreeText, Text: "Open the door"},
	}
	if err := req.Validate(4); err == nil {
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
