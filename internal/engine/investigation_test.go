package engine

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestStoreAndLoadInvestigationBoardPreservesCasesAndHiddenTruths(t *testing.T) {
	world := newTestWorld()

	board := InvestigationBoard{
		Cases: []InvestigationCase{
			{
				Title:       "Who sold you out?",
				Summary:     "Someone warned the checkpoint.",
				UpdatedTurn: 5,
				Links: []InvestigationLink{
					{Kind: "npc", RefID: "people:npc:lyanna", Label: "Lyanna"},
				},
				Clues: []InvestigationClue{
					{Label: "A missing seal", Detail: "The broken seal did not match the captain's archive."},
					{Label: "A missing seal", Detail: "duplicate should collapse"},
				},
				Suspects: []InvestigationSuspect{
					{Name: "Lyanna"},
				},
				Claims: []InvestigationClaim{
					{Statement: "The leak came from inside the watch."},
				},
				Contradictions: []InvestigationContradiction{
					{Label: "Lyanna was seen at both gates"},
				},
				Leads: []InvestigationLead{
					{Title: "Question the quartermaster"},
				},
				Theories: []InvestigationTheory{
					{Statement: "The guard captain was bribed."},
				},
				HiddenTruths: []InvestigationHiddenTruth{
					{Label: "The guard captain was paid"},
				},
			},
		},
	}

	storeInvestigationBoard(world, board)
	if !strings.Contains(world.InvestigationBoardJSON, "missing seal") {
		t.Fatalf("stored board = %s, want serialized clue payload", world.InvestigationBoardJSON)
	}

	loaded := loadInvestigationBoard(world)
	if len(loaded.Cases) != 1 {
		t.Fatalf("len(cases) = %d, want 1", len(loaded.Cases))
	}
	if len(loaded.Cases[0].Clues) != 1 {
		t.Fatalf("clues = %+v, want duplicate clue collapsed", loaded.Cases[0].Clues)
	}
	if len(loaded.Cases[0].HiddenTruths) != 1 || loaded.Cases[0].HiddenTruths[0].Status != "hidden" {
		t.Fatalf("hidden truths = %+v, want preserved hidden truth", loaded.Cases[0].HiddenTruths)
	}
	if loaded.Cases[0].Claims[0].Confidence != "uncertain" {
		t.Fatalf("claim confidence = %q, want default uncertain", loaded.Cases[0].Claims[0].Confidence)
	}
	if loaded.Cases[0].Theories[0].Confidence != "fragile" {
		t.Fatalf("theory confidence = %q, want default fragile", loaded.Cases[0].Theories[0].Confidence)
	}
}

func TestLoadInvestigationBoardHandlesInvalidPayloadGracefully(t *testing.T) {
	world := &storage.WorldState{InvestigationBoardJSON: `{"cases":[{"title":""}],"broken":`}

	loaded := loadInvestigationBoard(world)
	if len(loaded.Cases) != 0 {
		t.Fatalf("loaded cases = %+v, want empty board on invalid payload", loaded.Cases)
	}
}
