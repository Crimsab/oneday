package engine

import (
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestApplyStateChangesPersistsRelationshipsHooksAndReactions(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:         "story-social",
		Name:       "Social Story",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID

	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-lyanna",
		StoryID:           story.ID,
		Name:              "Lyanna",
		Role:              "scout",
		PersonalityJSON:   `{}`,
		RelationshipJSON:  `{}`,
		Disposition:       5,
		IsAlive:           true,
		FirstAppearedTurn: 1,
		LastSeenTurn:      1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	changes := map[string]interface{}{
		"npc_relationship": map[string]interface{}{
			"name":    "Lyanna",
			"trust":   map[string]interface{}{"change": 12},
			"fear":    map[string]interface{}{"change": -4},
			"respect": map[string]interface{}{"value": 9},
		},
		"hook_add": map[string]interface{}{
			"kind":   "mystery",
			"title":  "Who sold you out?",
			"detail": "Someone warned the guard before you arrived.",
		},
		"world_reaction_add": map[string]interface{}{
			"kind":   "rumor",
			"title":  "The market is whispering",
			"detail": "Your name is spreading through the stalls.",
		},
		"fail_forward": map[string]interface{}{
			"title":  "The guard is now suspicious",
			"detail": "The setback adds heat instead of ending the thread.",
		},
	}

	applied, err := ApplyStateChanges(changes, char, world, db, story.ID, 2)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied state changes, got none")
	}

	npc, err := db.GetNPCByName(story.ID, "Lyanna")
	if err != nil || npc == nil {
		t.Fatalf("GetNPCByName: %v", err)
	}
	axes := loadRelationshipAxes(npc)
	if axes.Trust != 12 || axes.Fear != -4 || axes.Respect != 9 {
		t.Fatalf("relationship axes = %+v, want trust=12 fear=-4 respect=9", axes)
	}

	hooks := activeStoryHooks(loadStoryHooks(world))
	if len(hooks) != 1 || hooks[0].Title != "Who sold you out?" {
		t.Fatalf("hooks = %+v, want single mystery hook", hooks)
	}

	reactions := visibleWorldReactions(loadWorldReactions(world))
	if len(reactions) != 2 {
		t.Fatalf("reactions = %+v, want 2 active reactions", reactions)
	}

	delta := buildTurnDelta(applied)
	if delta == nil || len(delta.Items) == 0 {
		t.Fatal("expected turn delta items from applied changes")
	}
}

