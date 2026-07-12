package engine

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestApplySocialDuelAftermathPersistsRelationshipAndRumor(t *testing.T) {
	db, world, npc := openSocialDuelAftermathFixture(t)

	state := &SocialDuelState{
		NPCName:   npc.Name,
		Objective: "Gain access to the harbor ledger",
		Stakes:    "If you fail, the ledger disappears by dawn",
		Winner:    "Aria",
		Status:    SocialDuelResolved,
	}
	result := &SocialRoundResult{
		PlayerAction: SocialActionAppeal,
		NPCAction:    SocialActionPressure,
		NPCDamage:    3,
		Resolved:     true,
		Winner:       "Aria",
		Outcome:      "objective_secured",
	}

	aftermath, err := ApplySocialDuelAftermath(db, world, npc, state, result, &SocialDuelCue{
		Mode:      SocialDuelCueOffer,
		NPCName:   npc.Name,
		Objective: state.Objective,
		Stakes:    state.Stakes,
	}, 12)
	if err != nil {
		t.Fatalf("ApplySocialDuelAftermath: %v", err)
	}
	if aftermath == nil {
		t.Fatal("ApplySocialDuelAftermath = nil")
	}
	if aftermath.ReactionTitle == "" {
		t.Fatal("ReactionTitle empty, want persisted rumor")
	}
	if !strings.Contains(npc.RelationshipJSON, `"trust":`) {
		t.Fatalf("relationship json = %s, want trust delta", npc.RelationshipJSON)
	}
	if !strings.Contains(npc.NotesOnProtagonist, "harbor ledger") {
		t.Fatalf("notes_on_protagonist = %s, want duel note", npc.NotesOnProtagonist)
	}
	if !strings.Contains(world.WorldReactionsJSON, aftermath.ReactionTitle) {
		t.Fatalf("world reactions = %s, want %q", world.WorldReactionsJSON, aftermath.ReactionTitle)
	}
}

func TestApplySocialDuelAftermathPersistsFailForwardAndFrontPressure(t *testing.T) {
	db, world, npc := openSocialDuelAftermathFixture(t)
	world.CurrentLocation = "Old Harbor"
	world.FrontsJSON = `[{"id":"front-harbor","faction":"Harbor Syndicate","title":"Harbor Syndicate Crackdown","stakes":"Control the harbor ledger and silence witnesses","visibility":"known","status":"active","segments":4,"progress":2}]`

	state := &SocialDuelState{
		NPCName:   npc.Name,
		Objective: "Keep the harbor ledger out of syndicate hands",
		Stakes:    "If you fail, the syndicate locks down Old Harbor",
		Winner:    npc.Name,
		Status:    SocialDuelResolved,
	}
	result := &SocialRoundResult{
		PlayerAction: SocialActionDeceive,
		NPCAction:    SocialActionExpose,
		PlayerDamage: 3,
		Resolved:     true,
		Winner:       npc.Name,
		Outcome:      "fail_forward",
		FailForward: &SocialFailForward{
			Kind:   "suspicion",
			Title:  "Harbor watch alerted",
			Detail: "The watch begins asking who touched the ledger.",
		},
	}

	aftermath, err := ApplySocialDuelAftermath(db, world, npc, state, result, &SocialDuelCue{
		Mode:      SocialDuelCueOffer,
		NPCName:   npc.Name,
		Objective: state.Objective,
		Stakes:    state.Stakes,
	}, 18)
	if err != nil {
		t.Fatalf("ApplySocialDuelAftermath: %v", err)
	}
	if aftermath == nil {
		t.Fatal("ApplySocialDuelAftermath = nil")
	}
	if aftermath.ReactionTitle != "Harbor watch alerted" {
		t.Fatalf("ReactionTitle = %q, want Harbor watch alerted", aftermath.ReactionTitle)
	}
	if aftermath.FrontPressureTitle == "" {
		t.Fatal("FrontPressureTitle empty, want front impact summary")
	}
	if !strings.Contains(world.FrontsJSON, `"kind":"heat"`) {
		t.Fatalf("fronts json = %s, want heat pressure", world.FrontsJSON)
	}
}

func TestApplySocialDuelAftermathDoesNotPublishFailedPersistence(t *testing.T) {
	db, world, npc := openSocialDuelAftermathFixture(t)
	originalWorld := world.WorldReactionsJSON
	originalRelationship := npc.RelationshipJSON
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture DB: %v", err)
	}

	aftermath, err := ApplySocialDuelAftermath(db, world, npc, &SocialDuelState{
		NPCName: npc.Name,
		Winner:  "Aria",
		Status:  SocialDuelResolved,
	}, &SocialRoundResult{
		PlayerAction: SocialActionAppeal,
		NPCDamage:    3,
		Resolved:     true,
		Winner:       "Aria",
	}, nil, 20)
	if err == nil || aftermath != nil {
		t.Fatalf("result = %#v, %v; want persistence error", aftermath, err)
	}
	if npc.RelationshipJSON != originalRelationship || world.WorldReactionsJSON != originalWorld {
		t.Fatal("failed persistence mutated published NPC or world state")
	}
}

func openSocialDuelAftermathFixture(t *testing.T) (*storage.DB, *storage.WorldState, *storage.NPC) {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "social-duel-aftermath.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now()
	story := &storage.Story{
		ID:        "story-social-aftermath",
		Name:      "Social Aftermath",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	world := &storage.WorldState{
		ID:              "world-social-aftermath",
		StoryID:         story.ID,
		CurrentChapter:  1,
		CurrentTurn:     10,
		CurrentLocation: "Old Harbor",
		FrontsJSON:      `[]`,
		UpdatedAt:       now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	npc := &storage.NPC{
		ID:                 "npc-lyanna",
		StoryID:            story.ID,
		Name:               "Lyanna",
		Role:               "harbor broker",
		RelationshipJSON:   `{"trust":10,"respect":5,"fear":0,"debt":0}`,
		PrivateThoughts:    `[]`,
		NotesOnProtagonist: `[]`,
		Disposition:        4,
		IsAlive:            true,
		FirstAppearedTurn:  2,
		LastSeenTurn:       10,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := db.CreateNPC(npc); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	return db, world, npc
}
