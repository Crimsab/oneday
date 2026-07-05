package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestInferNPCsFromNarrativeResponseTracksNamedSpeakerFromProseAndChoices(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:              "story-npc-infer",
		Name:            "NPC Infer Story",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	response := &NarrativeResponse{
		Narrative: `Marek deglutisce, poi abbassa il giornale. "Non mi hanno dato nomi," dice piano.`,
		Choices: []Choice{
			{ID: 1, Text: "Chiedere a Marek chi lo ha mandato e cosa sta aspettando."},
			{ID: 2, Text: "Avanzare verso il magazzino."},
		},
	}

	applied := inferNPCsFromNarrativeResponse(response, story.ID, 9, directNPCStore{db: db})
	if len(applied) != 1 {
		t.Fatalf("applied len = %d, want 1: %+v", len(applied), applied)
	}

	npc, err := db.GetNPCByName(story.ID, "Marek")
	if err != nil || npc == nil {
		t.Fatalf("GetNPCByName: %v, npc=%+v", err, npc)
	}
	if npc.Role != "person of interest" {
		t.Fatalf("Role = %q, want person of interest", npc.Role)
	}
	if npc.FirstAppearedTurn != 9 {
		t.Fatalf("FirstAppearedTurn = %d, want 9", npc.FirstAppearedTurn)
	}
	if !strings.Contains(npc.NotesOnProtagonist, "Named speaker") && !strings.Contains(npc.NotesOnProtagonist, "Referenced by") {
		t.Fatalf("NotesOnProtagonist = %q, want inference note", npc.NotesOnProtagonist)
	}
}

func TestInferNPCsFromNarrativeResponseIgnoresLocationsAndGenericGroups(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:              "story-npc-infer-ignore",
		Name:            "NPC Infer Ignore Story",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	response := &NarrativeResponse{
		Narrative: `Dock 7 resta chiuso. La Confraternita osserva da lontano.`,
		Choices: []Choice{
			{ID: 1, Text: "Avanzare verso il magazzino."},
			{ID: 2, Text: "Indagare sulla Confraternita."},
		},
		EntitiesMentioned: []EntityMention{
			{Name: "Dock 7", Type: "location"},
			{Name: "Confraternita", Type: "faction"},
		},
	}

	applied := inferNPCsFromNarrativeResponse(response, story.ID, 9, directNPCStore{db: db})
	if len(applied) != 0 {
		t.Fatalf("applied len = %d, want 0: %+v", len(applied), applied)
	}
}
