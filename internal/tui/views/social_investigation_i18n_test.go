package views

import (
	"os"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/storage"
)

func TestSocialDuelPresentationUsesItalianWithoutChangingStoryText(t *testing.T) {
	loc := appi18n.New(appi18n.Italian)
	state := &engine.SocialDuelState{
		NPCName: "Mara", Objective: "Secure passage", Stakes: "The last ferry",
		PlayerComposure: 8, PlayerPatience: 7, NPCComposure: 6, NPCPatience: 5,
		PlayerStance: engine.SocialStanceBold,
	}
	view := NewSocialDuelView(&engine.SocialDuelCue{Objective: "Secure passage", Pressure: "The bell is ringing"}, state, &storage.Character{Name: "Nico"}, &storage.NPC{Name: "Mara"}, 100, 40, loc).View()
	for _, want := range []string{"Obiettivo: Secure passage", "Posta in gioco: The last ferry", "Pressione: The bell is ringing", "La tua compostezza", "Nota sull'approccio", "Persuadi"} {
		if !strings.Contains(view, want) {
			t.Errorf("social duel missing %q:\n%s", want, view)
		}
	}

	note := renderSocialDuelRoundNote(socialDuelRoundResolvedMsg{
		Loc: loc, State: state, Round: &engine.SocialRoundResult{ExchangeLabel: "Measured exchange", NPCDamage: 2, PlayerDamage: 1}, PlayerNote: "Offer the map",
	}, nil)
	for _, want := range []string{"Duello sociale", "compostezza di Mara di 2", "Perdi 1 punti compostezza", "Offer the map", "Measured exchange"} {
		if !strings.Contains(note, want) {
			t.Errorf("round note missing %q:\n%s", want, note)
		}
	}
}

func TestInvestigationPresentationUsesItalianWithoutChangingCanon(t *testing.T) {
	loc := appi18n.New(appi18n.Italian)
	board := engine.InvestigationBoard{Cases: []engine.InvestigationCase{{
		Title: "Who sold you out?", Status: "open", UpdatedTurn: 4, Summary: "The ledger is incomplete.",
		Clues:    []engine.InvestigationClue{{Label: "Ledger ash", Source: "Bell Tower"}},
		Theories: []engine.InvestigationTheory{{Statement: "The captain took silver", Confidence: "likely"}},
		Links:    []engine.InvestigationLink{{Kind: "front", Label: "Whispers"}},
	}}}
	model := NewInvestigationBrowserModel("Indagini", board, 110, 36, loc)
	view := model.View()
	for _, want := range []string{"Casi aperti", "Aperti 1", "turno 4", "Indizi 1", "Who sold you out?", "Invio apre"} {
		if !strings.Contains(view, want) {
			t.Errorf("investigation view missing %q:\n%s", want, view)
		}
	}
	detail := formatInvestigationCaseDetail(board.Cases[0], loc)
	for _, want := range []string{"Stato: aperto", "Indizi", "fonte: Bell Tower", "Teorie", "probabile", "Fronte: Whispers", "The captain took silver"} {
		if !strings.Contains(detail, want) {
			t.Errorf("investigation detail missing %q:\n%s", want, detail)
		}
	}
}

func TestSocialAndInvestigationStaticPresentationUsesCatalogs(t *testing.T) {
	checks := map[string][]string{
		"social_duel.go":           {`"Objective: `, `"Approach note:"`, `"Could not resolve exchange:`},
		"investigation_browser.go": {`"Open Cases"`, `"Clues %d"`, `"↑↓ navigate · Enter open`},
	}
	for file, literals := range checks {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, literal := range literals {
			if strings.Contains(string(source), literal) {
				t.Errorf("%s contains untranslated presentation %s", file, literal)
			}
		}
	}
}
