package components

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/i18n"
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

func TestDiceViewUsesItalianPresentation(t *testing.T) {
	model := NewDiceModel("Una serratura arrugginita", 12, 14, 10, nil, true, 80, 24, i18n.New(i18n.Italian))
	model.done = true
	model.displayedNumber = model.FinalRoll
	view := model.View()
	for _, want := range []string{"Lancio", "Difficoltà", "SUPERATA", "Premi un tasto"} {
		if !strings.Contains(view, want) {
			t.Errorf("Italian dice view missing %q:\n%s", want, view)
		}
	}
}
