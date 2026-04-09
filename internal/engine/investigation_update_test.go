package engine

import "testing"

func TestApplyStateChangesInvestigationUpdateNormalizesTheoryMovement(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	storeInvestigationBoard(world, InvestigationBoard{
		Cases: []InvestigationCase{
			{
				ID:      "case-sold-out",
				Title:   "Who sold you out?",
				Summary: "Someone warned the checkpoint.",
				Claims: []InvestigationClaim{
					{ID: "claim-alone", Statement: "Lyanna acted alone", Status: "open", Confidence: "uncertain"},
				},
				Theories: []InvestigationTheory{
					{ID: "theory-bribe", Statement: "The guard captain was bribed", Status: "forming", Confidence: "fragile"},
				},
			},
		},
	})

	applied, err := ApplyStateChanges(map[string]interface{}{
		"investigation_update": map[string]interface{}{
			"case_id":        "case-sold-out",
			"summary":        "The leak now points back toward the checkpoint ledger.",
			"clues":          []interface{}{map[string]interface{}{"action": "add", "label": "Ledger ash", "detail": "The burned ledger page still smelled of sealing wax."}, map[string]interface{}{"action": "add", "label": "Ledger ash", "detail": "duplicate should merge"}},
			"claims":         []interface{}{map[string]interface{}{"action": "discredit", "statement": "Lyanna acted alone"}},
			"theories":       []interface{}{map[string]interface{}{"action": "strengthen", "statement": "The guard captain was bribed"}},
			"leads":          []interface{}{map[string]interface{}{"action": "progress", "title": "Question the quartermaster"}},
			"contradictions": []interface{}{map[string]interface{}{"action": "add", "label": "Two alibis overlap", "detail": "The quartermaster and captain both claim to have sealed the log."}},
		},
	}, char, world, nil, "test-story", 6)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected investigation state changes, got none")
	}

	board := loadInvestigationBoard(world)
	if len(board.Cases) != 1 {
		t.Fatalf("len(cases) = %d, want 1", len(board.Cases))
	}
	invCase := board.Cases[0]
	if invCase.Summary != "The leak now points back toward the checkpoint ledger." {
		t.Fatalf("summary = %q, want updated summary", invCase.Summary)
	}
	if len(invCase.Clues) != 1 {
		t.Fatalf("clues = %+v, want merged clue entry", invCase.Clues)
	}
	if invCase.Claims[0].Status != "disputed" {
		t.Fatalf("claim status = %q, want disputed", invCase.Claims[0].Status)
	}
	if invCase.Theories[0].Status != "likely" || invCase.Theories[0].Confidence != "supported" {
		t.Fatalf("theory = %+v, want likely/supported after strengthen", invCase.Theories[0])
	}
	if len(invCase.Leads) != 1 || invCase.Leads[0].Status != "pursued" {
		t.Fatalf("leads = %+v, want progressed lead", invCase.Leads)
	}
	if len(invCase.Contradictions) != 1 || invCase.Contradictions[0].Status != "open" {
		t.Fatalf("contradictions = %+v, want open contradiction", invCase.Contradictions)
	}
}

func TestApplyStateChangesInvestigationUpdateRevealsHiddenTruthAndSkipsMalformed(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	storeInvestigationBoard(world, InvestigationBoard{
		Cases: []InvestigationCase{
			{
				ID:    "case-sold-out",
				Title: "Who sold you out?",
				HiddenTruths: []InvestigationHiddenTruth{
					{ID: "truth-paid", Label: "The guard captain was paid", Detail: "Bell Choir silver changed hands."},
				},
			},
		},
	})

	applied, err := ApplyStateChanges(map[string]interface{}{
		"investigation_update": []interface{}{
			map[string]interface{}{
				"case_id": "case-sold-out",
				"claims": []interface{}{
					map[string]interface{}{"action": "reveal", "hidden_truth_id": "truth-paid"},
				},
			},
			map[string]interface{}{
				"claims": []interface{}{
					map[string]interface{}{"action": "add", "statement": "This should be ignored without a case"},
				},
			},
		},
	}, char, world, nil, "test-story", 7)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied state changes for hidden truth reveal")
	}

	board := loadInvestigationBoard(world)
	if len(board.Cases) != 1 {
		t.Fatalf("len(cases) = %d, want 1", len(board.Cases))
	}
	invCase := board.Cases[0]
	if len(invCase.Claims) != 1 || invCase.Claims[0].Statement != "The guard captain was paid" {
		t.Fatalf("claims = %+v, want revealed hidden truth as claim", invCase.Claims)
	}
	if invCase.Claims[0].Status != "supported" {
		t.Fatalf("claim status = %q, want supported after reveal", invCase.Claims[0].Status)
	}
	if len(invCase.HiddenTruths) != 1 || invCase.HiddenTruths[0].Status != "revealed" {
		t.Fatalf("hidden truths = %+v, want revealed status", invCase.HiddenTruths)
	}
}
