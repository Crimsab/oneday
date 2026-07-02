package engine

import (
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestParseSceneProgressionGuidance(t *testing.T) {
	raw := `{
		"assessment": "stalled",
		"strategy": "time_skip",
		"reason": "The scene keeps repeating the same domestic magical beat.",
		"instruction": "Jump to the next meaningful age and show what has changed in the home.",
		"time_skip_label": "Age 5 - first controlled spark",
		"time_skip_detail": "Keep the cottage and family dynamic, but show a new tension around training."
	}`

	guidance, err := parseSceneProgressionGuidance(raw)
	if err != nil {
		t.Fatalf("parseSceneProgressionGuidance: %v", err)
	}
	if guidance.Strategy != sceneProgressionStrategyTimeSkip {
		t.Fatalf("strategy = %q, want time_skip", guidance.Strategy)
	}
	if guidance.TimeSkipLabel == "" || guidance.TimeSkipDetail == "" {
		t.Fatalf("expected time skip details, got %+v", guidance)
	}
}

func TestDetectStalledNarrativeDraftRequestsRerollForMissedTimeSkip(t *testing.T) {
	recent := []storage.ChatMessage{
		testAssistantTurnWithMeta(t, 3, "Nella casa la scintilla di magia torna a tremare sopra il latte mentre la madre osserva il focolare.", "Casa di famiglia",
			"Osserva ancora la scintilla di magia sopra il latte",
			"Chiedi alla madre cosa significhi quella scintilla di magia",
		),
		testAssistantTurnWithMeta(t, 4, "Nel pomeriggio la stessa scintilla di magia torna sul latte nella stessa casa.", "Casa di famiglia",
			"Segui ancora la scintilla di magia sul latte",
			"Fai un'altra domanda alla madre sulla magia della scintilla",
		),
		testAssistantTurnWithMeta(t, 5, "La sera nella casa di famiglia riporta la stessa scintilla di magia sul latte e la stessa attesa.", "Casa di famiglia",
			"Controlla ancora la scintilla di magia che vibra sul latte",
			"Parla di nuovo con tua madre della scintilla di magia",
		),
	}

	guidance := &SceneProgressionGuidance{
		Assessment:     sceneProgressionAssessmentStalled,
		Strategy:       sceneProgressionStrategyTimeSkip,
		Reason:         "The scene needs to jump to the next meaningful childhood beat.",
		Instruction:    "Jump to the next milestone instead of replaying the same house beat.",
		TimeSkipLabel:  "Age 8 - first stable magical habit",
		TimeSkipDetail: "Keep the same home, but show how practice changed the family routine.",
	}
	candidate := &NarrativeResponse{
		Narrative: "Resti nella casa di famiglia e guardi ancora la stessa scintilla di magia tremare sopra il latte vicino al focolare.",
		Choices: []Choice{
			{ID: 1, Text: "Osserva ancora la scintilla di magia sopra il latte"},
			{ID: 2, Text: "Parla di nuovo con tua madre della scintilla di magia"},
		},
		Location: "Casa di famiglia",
	}

	issue := detectStalledNarrativeDraft(recent, candidate, guidance)
	if issue == nil {
		t.Fatal("expected stalled draft issue for repeated house beat without time skip")
	}
}

func TestDetectStalledNarrativeDraftAcceptsMeaningfulTimeSkip(t *testing.T) {
	recent := []storage.ChatMessage{
		testAssistantTurnWithMeta(t, 3, "Nella casa la scintilla di magia torna a tremare sopra il latte mentre la madre osserva il focolare.", "Casa di famiglia",
			"Osserva ancora la scintilla di magia sopra il latte",
			"Chiedi alla madre cosa significhi quella scintilla di magia",
		),
		testAssistantTurnWithMeta(t, 4, "Nel pomeriggio la stessa scintilla di magia torna sul latte nella stessa casa.", "Casa di famiglia",
			"Segui ancora la scintilla di magia sul latte",
			"Fai un'altra domanda alla madre sulla magia della scintilla",
		),
		testAssistantTurnWithMeta(t, 5, "La sera nella casa di famiglia riporta la stessa scintilla di magia sul latte e la stessa attesa.", "Casa di famiglia",
			"Controlla ancora la scintilla di magia che vibra sul latte",
			"Parla di nuovo con tua madre della scintilla di magia",
		),
	}

	guidance := &SceneProgressionGuidance{
		Assessment:     sceneProgressionAssessmentStalled,
		Strategy:       sceneProgressionStrategyTimeSkip,
		Reason:         "The scene needs to jump to the next meaningful childhood beat.",
		Instruction:    "Jump to the next milestone instead of replaying the same house beat.",
		TimeSkipLabel:  "Age 8 - first stable magical habit",
		TimeSkipDetail: "Keep the same home, but show how practice changed the family routine.",
	}
	candidate := &NarrativeResponse{
		Narrative: "Tre anni dopo, nella stessa casa di famiglia, la scintilla non e piu un incidente: a otto anni riesci a farla danzare tra le dita, e questo cambia il modo in cui tua madre ti guarda.",
		Choices: []Choice{
			{ID: 1, Text: "Mostra il nuovo controllo a tua madre"},
			{ID: 2, Text: "Chiedi perche ora la magia la preoccupa davvero"},
		},
		Location: "Casa di famiglia",
		EventCallouts: []EventCallout{
			{Kind: "timeskip", Title: "Tre anni dopo"},
		},
	}

	issue := detectStalledNarrativeDraft(recent, candidate, guidance)
	if issue != nil {
		t.Fatalf("expected time skip draft to pass, got %+v", issue)
	}
}

func TestSceneContractIgnoresUnknownStateChangesAsMovement(t *testing.T) {
	narrative := &NarrativeResponse{
		Narrative: "La scena resta ferma, ma il modello emette una chiave sbagliata.",
		StateChanges: map[string]interface{}{
			"skil_xp": map[string]interface{}{"skill": "Stealth", "xp": 25},
		},
	}
	if narrativeHasStructuralMovement(narrative) {
		t.Fatal("unknown state_changes key should not count as validated scene movement")
	}
}

func TestSceneContractAcceptsKnownStateChangeAsMovement(t *testing.T) {
	narrative := &NarrativeResponse{
		Narrative: "La scena cambia: paghi un costo concreto.",
		StateChanges: map[string]interface{}{
			"currency": -2,
		},
	}
	if !narrativeHasStructuralMovement(narrative) {
		t.Fatal("known meaningful state change should count as scene movement")
	}
}
