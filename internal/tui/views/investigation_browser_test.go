package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/engine"
)

func TestInvestigationBrowserGroupsCasesByStatus(t *testing.T) {
	t.Parallel()

	model := NewInvestigationBrowserModel("Investigations", engine.InvestigationBoard{
		Cases: []engine.InvestigationCase{
			{ID: "case-solved", Title: "Where did the silver go?", Status: "solved", UpdatedTurn: 9},
			{ID: "case-open", Title: "Who sold you out?", Status: "open", UpdatedTurn: 10, Clues: []engine.InvestigationClue{{Label: "Ledger ash"}}},
		},
	}, 120, 40)

	if len(model.rows) != 4 {
		t.Fatalf("len(rows) = %d, want 4", len(model.rows))
	}
	if model.rows[0].kind != investigationBrowserRowSection || model.rows[0].label != "Open Cases" {
		t.Fatalf("rows[0] = %+v, want Open Cases section", model.rows[0])
	}
	if model.rows[1].kind != investigationBrowserRowCase || model.board.Cases[model.rows[1].caseIndex].Title != "Who sold you out?" {
		t.Fatalf("rows[1] = %+v, want open case row", model.rows[1])
	}
	if model.rows[2].kind != investigationBrowserRowSection || model.rows[2].label != "Resolved" {
		t.Fatalf("rows[2] = %+v, want Resolved section", model.rows[2])
	}
	if !strings.Contains(model.View(), "Who sold you out?") {
		t.Fatalf("view missing open case:\n%s", model.View())
	}
}

func TestInvestigationBrowserEnterShowsCaseDetailWithoutHiddenTruths(t *testing.T) {
	t.Parallel()

	model := NewInvestigationBrowserModel("Investigations", engine.InvestigationBoard{
		Cases: []engine.InvestigationCase{
			{
				ID:       "case-open",
				Title:    "Who sold you out?",
				Status:   "open",
				Summary:  "Someone warned the checkpoint ahead of time.",
				Clues:    []engine.InvestigationClue{{Label: "Ledger ash", Detail: "The page still smelled of sealing wax."}},
				Suspects: []engine.InvestigationSuspect{{Name: "Lyanna", Detail: "Broker with too many excuses."}},
				Contradictions: []engine.InvestigationContradiction{
					{Label: "Two alibis overlap", Detail: "The quartermaster and captain both claim to have sealed the log."},
				},
				Theories: []engine.InvestigationTheory{{Statement: "The guard captain was bribed", Confidence: "likely"}},
				Links:    []engine.InvestigationLink{{Kind: "front", Label: "Whispers Around the Bell Tower"}},
				HiddenTruths: []engine.InvestigationHiddenTruth{
					{Label: "Bell Choir silver changed hands", Detail: "This must stay hidden."},
				},
			},
		},
	}, 120, 40)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !updated.detail.Visible() {
		t.Fatal("detail overlay not visible after opening selected case")
	}
	if !strings.Contains(updated.detail.Content, "Ledger ash") {
		t.Fatalf("detail missing clue:\n%s", updated.detail.Content)
	}
	if !strings.Contains(updated.detail.Content, "Front: Whispers Around the Bell Tower") {
		t.Fatalf("detail missing linked front:\n%s", updated.detail.Content)
	}
	if strings.Contains(updated.detail.Content, "Bell Choir silver changed hands") {
		t.Fatalf("detail leaked hidden truth:\n%s", updated.detail.Content)
	}
}
