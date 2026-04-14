package components

import (
	"strings"
	"testing"
)

func TestDiceViewShowsContextAndContinuationHint(t *testing.T) {
	t.Parallel()

	model := NewDiceModel("Pick the rusted lock before the guard returns.", 47, 57, 60, nil, false, 80, 24)
	model.done = true
	model.active = false
	model.displayedNumber = model.FinalRoll

	view := model.View()
	if !strings.Contains(view, "Pick the rusted lock") || !strings.Contains(view, "guard returns.") {
		t.Fatalf("dice view missing challenge context: %q", view)
	}
	if !strings.Contains(view, "continue the") || !strings.Contains(view, "story") {
		t.Fatalf("dice view missing continuation hint: %q", view)
	}
}
