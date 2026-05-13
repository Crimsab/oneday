package engine

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildStateSummaryIncludesKnownFrontPressureAndHidesHiddenFronts(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	world.FrontsJSON = `[
		{
			"id":"front-known",
			"faction":"Bell Choir",
			"title":"The Silent Bell Choir is seeding sleepers across the district",
			"public_title":"Whispers Around the Bell Tower",
			"stakes":"Sleeper-priests will take the guard towers.",
			"visibility":"known",
			"segments":6,
			"progress":3,
			"pressures":[{"region":"Bell Quarter","kind":"suspicion","level":35}]
		},
		{
			"id":"front-hidden",
			"faction":"Ash Court",
			"title":"The Ash Court is buying judges in secret",
			"visibility":"hidden",
			"segments":4,
			"progress":1,
			"pressures":[{"region":"High Court","kind":"control","level":80}]
		}
	]`

	summary := buildStateSummary(char, world, nil, "")
	if !strings.Contains(summary, "Active Fronts: Whispers Around the Bell Tower {Bell Choir} 3/6") {
		t.Fatalf("summary missing known front headline:\n%s", summary)
	}
	if !strings.Contains(summary, "Bell Quarter [suspicion 35 rising]") {
		t.Fatalf("summary missing known front pressure:\n%s", summary)
	}
	if !strings.Contains(summary, "guards ask sharper questions") {
		t.Fatalf("summary missing derived systemic fallout:\n%s", summary)
	}
	if strings.Contains(summary, "Ash Court is buying judges in secret") {
		t.Fatalf("summary leaked hidden front title:\n%s", summary)
	}
	if strings.Contains(summary, "High Court [control 80 critical]") {
		t.Fatalf("summary leaked hidden front pressure:\n%s", summary)
	}
}

func TestBuildStateSummaryIncludesActiveNemesisRoster(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	npcs := []storage.NPC{
		{
			Name:        "Lyanna",
			Role:        "broker",
			NemesisJSON: `{"status":"active","escalation_tier":3,"threat_posture":"vengeful","rivalry_score":8}`,
		},
		{
			Name:        "Brother Alden",
			Role:        "healer",
			NemesisJSON: `{}`,
		},
	}

	summary := buildStateSummary(char, world, npcs, "")
	if !strings.Contains(summary, "Known NPCs: 2 (Lyanna, Brother Alden)") {
		t.Fatalf("summary missing npc roster:\n%s", summary)
	}
	if !strings.Contains(summary, "Active Nemeses: Lyanna(tier 3)") {
		t.Fatalf("summary missing active nemesis line:\n%s", summary)
	}
}

func TestBuildStateSummaryIncludesOpenInvestigations(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	storeInvestigationBoard(world, InvestigationBoard{
		Cases: []InvestigationCase{
			{
				Title: "Who sold you out?",
				Clues: []InvestigationClue{
					{Label: "Ledger ash"},
				},
				Contradictions: []InvestigationContradiction{
					{Label: "Two alibis overlap"},
				},
				Theories: []InvestigationTheory{
					{Statement: "The guard captain was bribed"},
				},
			},
			{
				Title:  "Closed Case",
				Status: "solved",
			},
		},
	})

	summary := buildStateSummary(char, world, nil, "")
	if !strings.Contains(summary, "Investigations: Who sold you out? [clues:1] [contradictions:1] [theories:1]") {
		t.Fatalf("summary missing investigation digest:\n%s", summary)
	}
	if strings.Contains(summary, "Closed Case") {
		t.Fatalf("summary leaked solved investigation into open digest:\n%s", summary)
	}
}

func TestBuildStateSummaryIncludesProjectProgressAndCompletedFallout(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	storeProjectBoard(world, ProjectBoard{
		Projects: []ProjectClock{
			{
				ID:       "project-training",
				Title:    "Train with Lyanna",
				Kind:     "training",
				Status:   "active",
				Progress: 2,
				Segments: 4,
			},
			{
				ID:            "project-safehouse",
				Title:         "Restore the Lantern Loft",
				Kind:          "base",
				Status:        "completed",
				Progress:      4,
				Segments:      4,
				Outcome:       "You now have a safe place to disappear for a night.",
				CompletedTurn: 9,
			},
		},
	})

	summary := buildStateSummary(char, world, nil, "")
	if !strings.Contains(summary, "Projects: Train with Lyanna 2/4 [training]") {
		t.Fatalf("summary missing active project digest:\n%s", summary)
	}
	if !strings.Contains(summary, "Completed Projects: Restore the Lantern Loft 4/4 [base] — You now have a safe place to disappear for a night.") {
		t.Fatalf("summary missing completed project fallout:\n%s", summary)
	}
}

func TestBuildContextAddsNarrativeMomentumWhenRecentTurnsStall(t *testing.T) {
	story := &storage.Story{
		ID:              "story-momentum",
		Name:            "Momentum Test",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		Language:        "it",
	}
	char := newTestChar()
	world := newTestWorld()
	world.CurrentLocation = "Capanna familiare"

	recent := []storage.ChatMessage{
		testAssistantTurnWithMeta(t, 7, "Una scintilla azzurra vibra sopra il latte nel pentolino.", "Capanna familiare",
			"Ascolta ancora la scintilla e cerca un ritmo",
			"Osserva il latte per capire se la magia cresce",
			"Chiama Thorne e chiedigli cosa ne pensa",
		),
		testAssistantTurnWithMeta(t, 8, "La scintilla torna a increspare il latte mentre l'aria resta ferma.", "Capanna familiare",
			"Studia la scintilla piu da vicino",
			"Controlla il latte e la sua aura",
			"Vai da Thorne per un consiglio",
		),
		testAssistantTurnWithMeta(t, 9, "La magia continua a fremere sul bordo del latte senza cambiare davvero scena.", "Capanna familiare",
			"Analizza la scintilla prima che svanisca",
			"Esamina il latte per leggere la magia",
			"Cerca Thorne e condividi il dubbio",
		),
	}

	msgs := BuildContext(story, char, world, nil, recent, nil, "", "Continuo.", nil, nil)
	momentum := findLiveMomentumSummary(msgs)
	if momentum == "" {
		t.Fatalf("expected Narrative Momentum system message, got %#v", msgs)
	}
	if !strings.Contains(momentum, "Do NOT offer near-identical choices") {
		t.Fatalf("momentum guidance missing anti-repeat instruction:\n%s", momentum)
	}
	if !strings.Contains(momentum, "The live world state currently lacks strong external pressure.") {
		t.Fatalf("momentum guidance missing low-pressure note:\n%s", momentum)
	}
}

func TestBuildContextSkipsNarrativeMomentumWhenTurnsAreHealthy(t *testing.T) {
	story := &storage.Story{
		ID:              "story-healthy",
		Name:            "Healthy Test",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		Language:        "it",
	}
	char := newTestChar()
	world := newTestWorld()
	world.CurrentLocation = "Villaggio"

	recent := []storage.ChatMessage{
		testAssistantTurnWithMeta(t, 4, "La piazza si apre davanti a te con il mercato in fermento.", "Villaggio",
			"Parla con il mercante del rame",
			"Vai verso il tempio in fondo alla strada",
			"Segui il corriere che corre verso il porto",
		),
		testAssistantTurnWithMeta(t, 5, "Al porto una sirena interrompe le contrattazioni e tutti si voltano verso l'acqua.", "Porto",
			"Raggiungi il molo in fiamme",
			"Interroga la guardia sul segnale",
			"Cerca subito una via di fuga",
		),
		testAssistantTurnWithMeta(t, 6, "Nel tempio abbandonato trovi un archivio spezzato e un diario ancora tiepido.", "Tempio abbandonato",
			"Apri il diario sul tavolo",
			"Ispeziona le impronte nella polvere",
			"Richiama la tua memoria sul simbolo inciso",
		),
	}

	msgs := BuildContext(story, char, world, nil, recent, nil, "", "Continuo.", nil, nil)
	if momentum := findLiveMomentumSummary(msgs); momentum != "" {
		t.Fatalf("did not expect Narrative Momentum guidance for varied turns:\n%s", momentum)
	}
}

func TestBuildContextUsesSceneProgressionJudgeGuidance(t *testing.T) {
	story := &storage.Story{
		ID:              "story-timeskip",
		Name:            "Timeskip Test",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		Language:        "it",
	}
	char := newTestChar()
	world := newTestWorld()
	world.CurrentLocation = "Casa di famiglia"

	recent := []storage.ChatMessage{
		testAssistantTurnWithMeta(t, 3, "Nella casa di famiglia la scintilla di magia torna a tremare sopra il latte mentre la madre osserva il focolare.", "Casa di famiglia",
			"Osserva ancora la scintilla di magia sopra il latte",
			"Chiedi alla madre cosa significhi quella scintilla di magia",
			"Resta accanto al focolare guardando latte e scintilla",
		),
		testAssistantTurnWithMeta(t, 4, "Nel pomeriggio la stessa scintilla di magia torna sul latte nella stessa casa mentre la madre resta al focolare.", "Casa di famiglia",
			"Segui ancora la scintilla di magia sul latte",
			"Fai un'altra domanda alla madre sulla magia della scintilla",
			"Rimani al focolare osservando latte e scintilla",
		),
		testAssistantTurnWithMeta(t, 5, "La sera nella casa di famiglia riporta la stessa scintilla di magia sul latte e la stessa attesa accanto al focolare.", "Casa di famiglia",
			"Controlla ancora la scintilla di magia che vibra sul latte",
			"Parla di nuovo con tua madre della scintilla di magia",
			"Aspetta al focolare guardando latte e scintilla",
		),
	}

	guidance := &SceneProgressionGuidance{
		Assessment:     sceneProgressionAssessmentStalled,
		Strategy:       sceneProgressionStrategyTimeSkip,
		Reason:         "The childhood beat is atmospheric but no new pressure is landing turn by turn.",
		Instruction:    "Jump to the next meaningful childhood milestone and show how the same home now feels different.",
		TimeSkipLabel:  "Age 8 - first stable magical habit",
		TimeSkipDetail: "Carry forward the family home, but reveal what changed in routine, confidence, and tension.",
	}

	msgs := BuildContext(story, char, world, nil, recent, nil, "", "Continuo.", nil, guidance)
	momentum := findLiveMomentumSummary(msgs)
	if momentum == "" {
		t.Fatalf("expected Narrative Momentum summary with scene judge guidance")
	}
	if !strings.Contains(momentum, "Preferred strategy: time_skip.") {
		t.Fatalf("momentum summary missing time_skip strategy:\n%s", momentum)
	}
	if !strings.Contains(momentum, "Time skip target: Age 8 - first stable magical habit") {
		t.Fatalf("momentum summary missing time skip target:\n%s", momentum)
	}
}

func TestBuildContextAddsFreeActionHandlingForMacroInput(t *testing.T) {
	story := &storage.Story{
		ID:              "story-free-action",
		Name:            "Lucernel",
		Genre:           "fantasy",
		SettingJSON:     `{"world_name":"Lucernel","technology_level":"mythic fantasy"}`,
		StatsSchemaJSON: `{}`,
		Language:        "it",
	}
	char := newTestChar()
	world := newTestWorld()
	world.CurrentLocation = "Capanna familiare"

	msgs := BuildContext(
		story,
		char,
		world,
		nil,
		nil,
		nil,
		"",
		"crollo a terra, sogno un animale alato, poi dormo due anni e mi risveglio cresciuto in un altro posto",
		nil,
		nil,
	)

	summary := findFreeActionSummary(msgs)
	if summary == "" {
		t.Fatalf("expected Free Action Handling system message, got %#v", msgs)
	}
	if !strings.Contains(summary, "Do NOT drift into an unrelated setting or genre") {
		t.Fatalf("free action guidance missing continuity guard:\n%s", summary)
	}
	if !strings.Contains(summary, "If you honor a time jump") {
		t.Fatalf("free action guidance missing time jump handling:\n%s", summary)
	}
}

func TestBuildContextSkipsFreeActionHandlingForSimpleInput(t *testing.T) {
	story := &storage.Story{
		ID:              "story-simple-free-action",
		Name:            "Lucernel",
		Genre:           "fantasy",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		Language:        "it",
	}
	char := newTestChar()
	world := newTestWorld()

	msgs := BuildContext(story, char, world, nil, nil, nil, "", "Busso alla porta.", nil, nil)
	if summary := findFreeActionSummary(msgs); summary != "" {
		t.Fatalf("did not expect Free Action Handling guidance for a simple local action:\n%s", summary)
	}
}

func testAssistantTurnWithMeta(t *testing.T, turn int, narrative, location string, choices ...string) storage.ChatMessage {
	t.Helper()

	metaJSON, err := json.Marshal(persistedAssistantMeta{
		Location: location,
		Choices:  choices,
		Output: &ChatOutput{
			Narrative: narrative,
			Choices:   choices,
			Location:  location,
		},
	})
	if err != nil {
		t.Fatalf("marshal test assistant meta: %v", err)
	}

	return storage.ChatMessage{
		Turn:         turn,
		Role:         "assistant",
		Content:      narrative,
		MetadataJSON: string(metaJSON),
	}
}

func findLiveMomentumSummary(msgs []ai.Message) string {
	for _, msg := range msgs {
		if msg.Role != "system" {
			continue
		}
		if strings.HasPrefix(msg.Content, "## Narrative Momentum\n") &&
			(strings.Contains(msg.Content, "Recent turns are circling the same micro-beat.") ||
				strings.Contains(msg.Content, "Recent turns may be drifting. Use this scene progression directive")) {
			return msg.Content
		}
	}
	return ""
}

func findFreeActionSummary(msgs []ai.Message) string {
	for _, msg := range msgs {
		if msg.Role != "system" {
			continue
		}
		if strings.HasPrefix(msg.Content, "## Free Action Handling\n") {
			return msg.Content
		}
	}
	return ""
}
