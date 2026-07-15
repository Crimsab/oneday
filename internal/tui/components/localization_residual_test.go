package components

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
)

func TestResidualComponentsRenderItalianPresentation(t *testing.T) {
	loc := appi18n.New(appi18n.Italian)

	suggestions := NewSuggestionList(loc)
	suggestions.SetItems([]SuggestionItem{{Value: "/map", Label: "/map"}})
	for _, want := range []string{"Suggerimenti", "scrivi per filtrare"} {
		if view := suggestions.View(); !strings.Contains(view, want) {
			t.Fatalf("suggestions missing %q:\n%s", want, view)
		}
	}

	choices := NewChoiceList(loc)
	choices.SetChoices([]ChoiceItem{{ID: 1, Text: "Parla con Mara", Intent: "social", Risk: "medium", Scope: "npc", Certainty: "uncertain"}})
	for _, want := range []string{"intento:sociale", "rischio:medio", "ambito:PNG", "certezza:incerta"} {
		if view := choices.View(); !strings.Contains(view, want) {
			t.Fatalf("choices missing %q:\n%s", want, view)
		}
	}

	rps := NewRPSModel(80, 24, loc)
	for _, want := range []string{"SASSO CARTA FORBICI", "Sasso", "Carta", "Forbici"} {
		if view := rps.View(); !strings.Contains(view, want) {
			t.Fatalf("rps missing %q:\n%s", want, view)
		}

	}
	memory := NewMemoryModel(engine.NewMemoryChallenge([]string{"up"}, 1), 80, 24, loc)
	if view := memory.View(); !strings.Contains(view, "SEQUENZA DI MEMORIA") {
		t.Fatalf("memory title not localized:\n%s", view)
	}
	quick := NewQuickTimeModel(engine.NewQuickTimeChallenge(1), 80, 24, loc)
	if view := quick.View(); !strings.Contains(view, "Tempo limite") {
		t.Fatalf("quick-time limit not localized:\n%s", view)
	}
	dice := NewDiceModel("", 1, 1, 1, nil, true, 80, 24, loc)
	if view := dice.View(); !strings.Contains(view, "LANCIO DI DADI") {
		t.Fatalf("dice title not localized:\n%s", view)
	}
}
