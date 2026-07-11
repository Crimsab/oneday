package service

import (
	"testing"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildTurnEventsPublishesPortableChallengeLifecycle(t *testing.T) {
	instance := &contracts.ChallengeInstance{ProtocolVersion: 1, ID: "challenge-event", StoryID: "story-1", BranchID: "branch-1", Turn: 2, Definition: contracts.ChallengeDefinition{ID: "action", Kind: "action", Difficulty: 50}, Seed: 3, Policy: contracts.OutcomePolicy{ID: "balanced"}}
	outcome := &contracts.OutcomeEnvelope{Version: 1, Degree: contracts.OutcomeFullSuccess, Difficulty: 50, Seed: 3, Roll: 70, Total: 70, Margin: 20}
	resolution := &contracts.ChallengeResolution{ProtocolVersion: 1, InstanceID: instance.ID, Input: contracts.ChallengeInput{Intent: "advance"}, Outcome: *outcome}
	resp := &engine.NarrativeResponse{Narrative: "Done", ChallengeInstance: instance, ChallengeResolution: resolution, ResolvedOutcome: outcome}
	events, err := buildTurnEvents(contracts.SubmitActionRequest{StoryID: "story-1", IdempotencyKey: "idem", Action: contracts.PlayerAction{Kind: contracts.ActionKindFreeText, Text: "advance"}}, &contracts.GameSnapshot{Turn: 2}, "session-1", resp, &storage.WorldState{CurrentTurn: 3}, 4)
	if err != nil {
		t.Fatal(err)
	}
	seenStarted, seenResolved := false, false
	for _, event := range events {
		seenStarted = seenStarted || event.Type == contracts.EventChallengeStarted
		seenResolved = seenResolved || event.Type == contracts.EventChallengeResolved
	}
	if !seenStarted || !seenResolved {
		t.Fatalf("missing challenge lifecycle events: %+v", events)
	}
}
