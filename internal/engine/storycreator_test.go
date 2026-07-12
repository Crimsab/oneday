package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/storage"
)

type storyCreatorMockProvider struct {
	responses []string
	callCount int
}

func (m *storyCreatorMockProvider) Name() string { return "mock-story-provider" }

func (m *storyCreatorMockProvider) Complete(_ context.Context, _ ai.Request) (ai.Response, error) {
	if m.callCount >= len(m.responses) {
		return ai.Response{}, fmt.Errorf("unexpected extra AI call")
	}
	resp := ai.Response{
		Content:   m.responses[m.callCount],
		Model:     "mock-story-model",
		Provider:  "mock-story-provider",
		LatencyMs: 12,
	}
	m.callCount++
	return resp, nil
}

func newStoryCreatorTestDB(t *testing.T) *storage.DB {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func newStoryCreatorForTest(t *testing.T, responses ...string) (*StoryCreator, *storyCreatorMockProvider) {
	t.Helper()

	provider := &storyCreatorMockProvider{responses: responses}
	router, err := ai.NewRouter([]ai.Provider{provider})
	if err != nil {
		t.Fatalf("ai.NewRouter: %v", err)
	}

	creator := NewStoryCreator(router, newStoryCreatorTestDB(t), configTestGenCfg())
	return creator, provider
}

func configTestGenCfg() config.GenerationConfig {
	return config.GenerationConfig{
		Temperature:    0.7,
		MaxTokens:      2048,
		TimeoutSeconds: 30,
	}
}

const validStoryDefinitionJSON = `{
  "name": "Le Campane di Vespera",
  "description": "Una citta in rovina affacciata sul sale. Le campane sommerse chiamano i vivi e attirano i disperati.",
  "genre": "fantasy oscuro",
  "tone": "malinconico",
  "language": "italiano",
  "writing_style": "prosa elegante e inquieta",
  "prompt_directives": "Dialoghi asciutti.",
  "setting": {
    "world_name": "Vespera",
    "era": "Eta delle Maree Spezzate",
    "geography": "Laguna nera e saline in rovina",
    "magic_system": "Campane sommerse che chiedono sacrifici",
    "technology_level": "Rinascimento decadente",
    "society": "Casate mercantili, culti e scavatori",
    "rules": ["La magia chiede sempre un prezzo", "Le maree portano voci e debiti", "Le campane non mentono mai", "Il sale conserva e corrompe"],
    "factions": ["Casata Valcerra", "Custodi delle Campane"],
    "cultures": ["Scavatori di laguna", "Mercanti del sale"],
    "dangers": ["Nebbie senzienti", "Predoni delle saline", "Sovraccarichi rituali"]
  },
  "stats_schema": {
    "vitals": [{"key":"hp","label":"Salute","starting":10}],
    "attributes": [{"key":"agi","label":"Agilita","starting":3},{"key":"wit","label":"Acume","starting":3}],
    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
    "currency": {"name":"Corone","starting":5},
    "has_combat": true
  }
}`

func TestExtractStoryJSONRequiresAuthoringFields(t *testing.T) {
	def := extractStoryJSON(validStoryDefinitionJSON)
	if def == nil {
		t.Fatal("extractStoryJSON returned nil")
	}
	if def.Language != "italiano" {
		t.Fatalf("Language = %q, want italiano", def.Language)
	}
	if def.WritingStyle == "" {
		t.Fatal("WritingStyle unexpectedly empty")
	}
}

func TestStoryCreatorGeneratesDraftAndMovesToWorldReview(t *testing.T) {
	creator, provider := newStoryCreatorForTest(t, validStoryDefinitionJSON)

	resp, err := creator.SendMessage(context.Background(), "Italian dark fantasy with dangerous bells and dry dialogue.")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if creator.Phase() != PhaseConversation {
		t.Fatalf("Phase = %v, want PhaseConversation", creator.Phase())
	}
	if creator.StageLabel() != "Review world draft" {
		t.Fatalf("StageLabel = %q", creator.StageLabel())
	}
	if creator.Definition() == nil {
		t.Fatal("Definition should not be nil after first brief")
	}
	if provider.callCount != 1 {
		t.Fatalf("AI call count = %d, want 1", provider.callCount)
	}
	if creator.LastModel() != "mock-story-model" {
		t.Fatalf("LastModel = %q, want mock-story-model", creator.LastModel())
	}
	if len(creator.Actions()) == 0 {
		t.Fatal("expected quick actions in world review stage")
	}
	if got := resp; got == "" || !containsSubstring(got, "World Draft") {
		t.Fatalf("world review response missing expected content: %q", got)
	}
}

func TestStoryCreatorPresetExposesCanonicalBriefBeforeGeneration(t *testing.T) {
	creator, provider := newStoryCreatorForTest(t, validStoryDefinitionJSON)
	actions := creator.Actions()

	if len(actions) == 0 {
		t.Fatal("expected story brief presets")
	}
	if actions[0].Key != "preset_dark_fantasy" {
		t.Fatalf("first action key = %q, want preset_dark_fantasy", actions[0].Key)
	}
	if actions[0].Seed != darkFantasyPreset {
		t.Fatalf("preset seed = %q, want canonical dark fantasy preset", actions[0].Seed)
	}
	if provider.callCount != 0 {
		t.Fatalf("AI call count = %d, want 0 before preset confirmation", provider.callCount)
	}
}

func TestStoryCreatorCanFinishWithLocalCharacterSetup(t *testing.T) {
	creator, provider := newStoryCreatorForTest(t, validStoryDefinitionJSON)

	if _, err := creator.SendMessage(context.Background(), "Italian dark fantasy with bells."); err != nil {
		t.Fatalf("initial SendMessage: %v", err)
	}
	if _, err := creator.ExecuteAction(context.Background(), "accept_world"); err != nil {
		t.Fatalf("accept_world: %v", err)
	}
	if _, err := creator.ExecuteAction(context.Background(), "accept_rules"); err != nil {
		t.Fatalf("accept_rules: %v", err)
	}
	if _, err := creator.ExecuteAction(context.Background(), "accept_stats"); err != nil {
		t.Fatalf("accept_stats: %v", err)
	}
	if _, err := creator.ExecuteAction(context.Background(), "create_story"); err != nil {
		t.Fatalf("create_story: %v", err)
	}
	if creator.Phase() != PhaseCharacter {
		t.Fatalf("Phase = %v, want PhaseCharacter", creator.Phase())
	}
	if _, err := creator.SendMessage(context.Background(), "Mira"); err != nil {
		t.Fatalf("character name: %v", err)
	}
	if _, err := creator.ExecuteAction(context.Background(), "skip_background"); err != nil {
		t.Fatalf("skip_background: %v", err)
	}

	if creator.Phase() != PhaseDone {
		t.Fatalf("Phase = %v, want PhaseDone", creator.Phase())
	}
	if creator.Story() == nil {
		t.Fatal("Story should be persisted")
	}
	if creator.Character() == nil {
		t.Fatal("Character should be persisted")
	}
	if creator.Character().Name != "Mira" {
		t.Fatalf("Character.Name = %q, want Mira", creator.Character().Name)
	}
	if creator.Character().Background != "" {
		t.Fatalf("Character.Background = %q, want empty", creator.Character().Background)
	}
	if provider.callCount != 1 {
		t.Fatalf("AI call count = %d, want 1", provider.callCount)
	}
}

func TestStoryCreatorStateRoundTripPreservesReviewDraft(t *testing.T) {
	creator, _ := newStoryCreatorForTest(t, validStoryDefinitionJSON)

	if _, err := creator.SendMessage(context.Background(), "Italian dark fantasy with bells."); err != nil {
		t.Fatalf("initial SendMessage: %v", err)
	}

	state := creator.ExportState()
	if state.Stage != "review_world" {
		t.Fatalf("state.Stage = %q, want review_world", state.Stage)
	}
	if state.Definition == nil || state.Definition.Name == "" {
		t.Fatalf("state.Definition = %#v, want populated draft", state.Definition)
	}

	restored, _ := newStoryCreatorForTest(t)
	if err := restored.RestoreState(state); err != nil {
		t.Fatalf("RestoreState: %v", err)
	}
	if restored.StageKey() != "review_world" {
		t.Fatalf("StageKey = %q, want review_world", restored.StageKey())
	}
	if restored.Definition() == nil || restored.Definition().Name != creator.Definition().Name {
		t.Fatalf("restored definition = %#v, want %q", restored.Definition(), creator.Definition().Name)
	}
	if restored.InputPlaceholder() == "" || len(restored.Actions()) == 0 {
		t.Fatal("restored creator should expose browser wizard controls")
	}
}

func containsSubstring(s, want string) bool {
	return strings.Contains(s, want)
}

func TestExtractStoryJSONRejectsMissingLanguage(t *testing.T) {
	raw := `{
	  "name": "Le Campane di Vespera",
	  "description": "Una citta in rovina affacciata sul sale.",
	  "genre": "fantasy oscuro",
	  "tone": "malinconico",
	  "writing_style": "prosa elegante e inquieta",
	  "prompt_directives": "",
	  "setting": {
	    "world_name": "Vespera",
	    "era": "Eta delle Maree Spezzate",
	    "geography": "Laguna nera",
	    "magic_system": "Campane sommerse",
	    "technology_level": "Rinascimento decadente",
	    "society": "Casate e culti",
	    "rules": ["La magia ha un prezzo"],
	    "factions": ["Casata Valcerra"],
	    "cultures": ["Scavatori"],
	    "dangers": ["Nebbie senzienti"]
	  },
	  "stats_schema": {
	    "vitals": [{"key":"hp","label":"Salute","starting":10}],
	    "attributes": [{"key":"agi","label":"Agilita","starting":3}],
	    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
	    "currency": {"name":"Corone","starting":5},
	    "has_combat": true
	  }
	}`

	if def := extractStoryJSON(raw); def != nil {
		t.Fatal("extractStoryJSON accepted a story without language")
	}
}

func TestStoryCreatorRepairsInvalidStoryDefinitionBeforeFailing(t *testing.T) {
	invalid := `not json at all`
	creator, provider := newStoryCreatorForTest(t, invalid, validStoryDefinitionJSON)

	resp, err := creator.SendMessage(context.Background(), "Dark fantasy with bells and salt.")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if creator.Definition() == nil {
		t.Fatal("Definition should be repaired and set")
	}
	if provider.callCount != 2 {
		t.Fatalf("AI call count = %d, want 2 after repair retry", provider.callCount)
	}
	if !containsSubstring(resp, "World Draft") {
		t.Fatalf("unexpected response after repair: %q", resp)
	}
}

func TestStoryCreatorRepairsMissingAuthoringFieldsViaSecondPass(t *testing.T) {
	partial := `{
	  "name": "Le Ciminiere di Nerofumo",
	  "description": "Una capitale di ottone e fuliggine sospesa tra culto e repressione.",
	  "setting": {
	    "world_name": "Nerofumo",
	    "era": "Secolo delle Caldaie",
	    "geography": "Canali tossici e quartieri-fabbrica",
	    "magic_system": "Liturgie del vapore",
	    "technology_level": "Macchine a pressione e automi rituali",
	    "society": "Gilde industriali, clero del vapore e polizia segreta",
	    "rules": ["Il vapore consacrato alimenta la città", "Ogni inquisizione lascia un marchio"],
	    "factions": ["Conclave delle Caldaie", "Ispettorato Fuliggine"],
	    "cultures": ["Operai dei canali", "Nobiltà delle turbine"],
	    "dangers": ["Blackout rituali", "Sparizioni nelle condotte", "Sommosse di automi"]
	  },
	  "stats_schema": {
	    "vitals": [{"key":"hp","label":"Salute","starting":10}],
	    "attributes": [{"key":"wit","label":"Acume","starting":3}],
	    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
	    "currency": {"name":"Corone di Carbone","starting":8},
	    "has_combat": true
	  }
	}`

	creator, provider := newStoryCreatorForTest(t, partial, validStoryDefinitionJSON)

	_, err := creator.SendMessage(context.Background(), "Mondo steampunk in tono serio e tenebroso, lingua italiana, prosa elegante ma non prolissa.")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("AI call count = %d, want 2 because repair pass should complete missing authoring fields", provider.callCount)
	}
	if creator.Definition() == nil {
		t.Fatal("Definition should be available")
	}
	if creator.Definition().Tone == "" || creator.Definition().Language == "" || creator.Definition().WritingStyle == "" {
		t.Fatal("Repair pass should return authoring fields")
	}
}

func TestStoryCreatorCoercesObjectShapedStatsSchema(t *testing.T) {
	partial := `{
	  "name": "Le Ciminiere di Nerofumo",
	  "description": "Una capitale di ottone e fuliggine sospesa tra culto e repressione.",
	  "genre": "steampunk investigativo",
	  "tone": "serio e tenebroso",
	  "language": "italiano",
	  "writing_style": "prosa elegante ma non prolissa",
	  "prompt_directives": "keep dialogue sharp",
	  "setting": {
	    "world_name": "Nerofumo",
	    "era": "Secolo delle Caldaie",
	    "geography": "Canali tossici e quartieri-fabbrica",
	    "magic_system": "Liturgie del vapore",
	    "technology_level": "Macchine a pressione e automi rituali",
	    "society": "Gilde industriali, clero del vapore e polizia segreta",
	    "rules": ["Il vapore consacrato alimenta la città", "Ogni inquisizione lascia un marchio"],
	    "factions": ["Conclave delle Caldaie", "Ispettorato Fuliggine"],
	    "cultures": ["Operai dei canali", "Nobiltà delle turbine"],
	    "dangers": ["Blackout rituali", "Sparizioni nelle condotte", "Sommosse di automi"]
	  },
	  "stats_schema": {
	    "vitals": {
	      "hp": {"label":"Salute","starting":10},
	      "steam": {"label":"Pressione","starting":8}
	    },
	    "attributes": {
	      "wit": {"label":"Acume","starting":3}
	    },
	    "secondary": {
	      "rep": {"label":"Reputazione","starting":0}
	    },
	    "currency": "Corone di Carbone",
	    "has_combat": true
	  }
	}`

	creator, provider := newStoryCreatorForTest(t, partial)
	_, err := creator.SendMessage(context.Background(), "Mondo steampunk in tono serio e tenebroso, lingua italiana.")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if provider.callCount != 1 {
		t.Fatalf("AI call count = %d, want 1", provider.callCount)
	}
	if creator.Definition() == nil {
		t.Fatal("Definition should be available")
	}
	if len(creator.Definition().StatsSchema.Vitals) != 2 {
		t.Fatalf("vitals len = %d, want 2 after object coercion", len(creator.Definition().StatsSchema.Vitals))
	}
	if creator.Definition().StatsSchema.Currency == nil {
		t.Fatal("Currency should be coerced from string")
	}
}

func TestStoryCreatorCoercesCurrencyArray(t *testing.T) {
	partial := `{
	  "name": "Le Ciminiere di Nerofumo",
	  "description": "Una capitale di ottone e fuliggine sospesa tra culto e repressione.",
	  "genre": "steampunk investigativo",
	  "tone": "serio e tenebroso",
	  "language": "italiano",
	  "writing_style": "prosa elegante ma non prolissa",
	  "prompt_directives": "keep dialogue sharp",
	  "setting": {
	    "world_name": "Nerofumo",
	    "era": "Secolo delle Caldaie",
	    "geography": "Canali tossici e quartieri-fabbrica",
	    "magic_system": "Liturgie del vapore",
	    "technology_level": "Macchine a pressione e automi rituali",
	    "society": "Gilde industriali, clero del vapore e polizia segreta",
	    "rules": ["Il vapore consacrato alimenta la città"],
	    "factions": ["Conclave delle Caldaie"],
	    "cultures": ["Operai dei canali"],
	    "dangers": ["Blackout rituali"]
	  },
	  "stats_schema": {
	    "vitals": [{"key":"hp","label":"Salute","starting":10}],
	    "attributes": [{"key":"wit","label":"Acume","starting":3}],
	    "secondary": [],
	    "currency": [{"name":"Corone di Carbone","starting":8}],
	    "has_combat": true
	  }
	}`

	creator, _ := newStoryCreatorForTest(t, partial)
	_, err := creator.SendMessage(context.Background(), "Mondo steampunk in tono serio e tenebroso, lingua italiana.")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if creator.Definition() == nil || creator.Definition().StatsSchema.Currency == nil {
		t.Fatal("Currency should be coerced from array")
	}
	if creator.Definition().StatsSchema.Currency.Name != "Corone di Carbone" {
		t.Fatalf("Currency.Name = %q, want Corone di Carbone", creator.Definition().StatsSchema.Currency.Name)
	}
}

func TestExtractStoryJSONCoercesLegacyFlatDraft(t *testing.T) {
	raw := `{
	  "language":"it",
	  "title":"Mystery alla Stazione Centrale",
	  "description":"Un mystery urbano ambientato attorno a una grande stazione centrale, dove ogni indizio ha un costo e ogni scelta lascia conseguenze visibili.",
	  "writing_style":"Prosa secca e concreta, con dettagli urbani osservati da vicino.",
	  "prompt_directives":"Mantieni il tono realistico; non introdurre poteri o soluzioni facili.",
	  "rules":["Nessun potere soprannaturale o abilità gratuite","Ogni azione lascia tracce"],
	  "factions":["Polizia ferroviaria","Personale della stazione"],
	  "cultures":["Pendolari e viaggiatori di passaggio"],
	  "dangers":["Falsi indizi","Sorveglianza e controlli"],
	  "stats":["Osservazione","Ragionamento","Empatia","Furtività","Reattività","Tenuta"]
	}`

	def := extractStoryJSON(raw)
	if def == nil {
		t.Fatal("extractStoryJSON returned nil for legacy flat draft")
	}
	if def.Name != "Mystery alla Stazione Centrale" {
		t.Fatalf("Name = %q, want title alias", def.Name)
	}
	if def.Genre != "mystery urbano" {
		t.Fatalf("Genre = %q, want inferred mystery urbano", def.Genre)
	}
	if def.Setting.WorldName != "Mystery alla Stazione Centrale" {
		t.Fatalf("WorldName = %q, want title fallback", def.Setting.WorldName)
	}
	if len(def.StatsSchema.Attributes) != 6 {
		t.Fatalf("attributes len = %d, want legacy stats converted", len(def.StatsSchema.Attributes))
	}
	if def.StatsSchema.Attributes[0].Key != "osservazione" || def.StatsSchema.Attributes[0].Label != "Osservazione" {
		t.Fatalf("first legacy stat = %#v", def.StatsSchema.Attributes[0])
	}
}

func TestNoCombatQuickActionCanDisablePreviousCombat(t *testing.T) {
	previous := &StoryDefinition{
		StatsSchema: StatsSchema{
			Vitals:     []StatDef{{Key: "hp", Label: "HP", Starting: 10}},
			Attributes: []StatDef{{Key: "wit", Label: "Wit", Starting: 5}},
			HasCombat:  true,
		},
	}
	def := &StoryDefinition{
		StatsSchema: StatsSchema{
			Vitals:     []StatDef{{Key: "focus", Label: "Focus", Starting: 10}},
			Attributes: []StatDef{{Key: "heart", Label: "Heart", Starting: 5}},
			HasCombat:  false,
		},
	}

	normalizeStoryDefinitionWithOptions(def, "", previous, storyDefinitionParseOptions{ForceDisableCombat: true})

	if def.StatsSchema.HasCombat {
		t.Fatal("HasCombat = true, want false after explicit no_combat action")
	}
}

func TestStoryCreatorCoercesSettingObjectLists(t *testing.T) {
	partial := `{
	  "name": "Le Ciminiere di Nerofumo",
	  "description": "Una capitale di ottone e fuliggine sospesa tra culto e repressione.",
	  "genre": "steampunk investigativo",
	  "tone": "serio e tenebroso",
	  "language": "italiano",
	  "writing_style": "prosa elegante ma non prolissa",
	  "prompt_directives": "keep dialogue sharp",
	  "setting": {
	    "world_name": "Nerofumo",
	    "era": "Secolo delle Caldaie",
	    "geography": "Canali tossici e quartieri-fabbrica",
	    "magic_system": "Liturgie del vapore",
	    "technology_level": "Macchine a pressione e automi rituali",
	    "society": "Gilde industriali, clero del vapore e polizia segreta",
	    "rules": {"r1":"Il vapore consacrato alimenta la città"},
	    "factions": {"Conclave delle Caldaie":{"description":"oligarchi del vapore"}},
	    "cultures": {"Operai dei canali":"resistenti e superstiziosi"},
	    "dangers": {"Blackout rituali":"aprono varchi e panico"}
	  },
	  "stats_schema": {
	    "vitals": [{"key":"hp","label":"Salute","starting":10}],
	    "attributes": [{"key":"wit","label":"Acume","starting":3}],
	    "secondary": [],
	    "currency": {"name":"Corone di Carbone","starting":8},
	    "has_combat": true
	  }
	}`

	creator, _ := newStoryCreatorForTest(t, partial)
	_, err := creator.SendMessage(context.Background(), "Mondo steampunk in tono serio e tenebroso, lingua italiana.")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if len(creator.Definition().Setting.Factions) != 1 || creator.Definition().Setting.Factions[0] != "Conclave delle Caldaie" {
		t.Fatalf("Factions = %#v, want coerced faction name", creator.Definition().Setting.Factions)
	}
	if len(creator.Definition().Setting.Cultures) != 1 || creator.Definition().Setting.Cultures[0] != "resistenti e superstiziosi" {
		t.Fatalf("Cultures = %#v, want value-based coercion", creator.Definition().Setting.Cultures)
	}
}

func TestExtractStoryJSONRejectsBlankStatDefinitions(t *testing.T) {
	raw := `{
	  "name": "Ombre nel Vapore",
	  "description": "Una città di ottone, culto e nebbia.",
	  "genre": "steampunk",
	  "tone": "serio",
	  "language": "it",
	  "writing_style": "prosa tesa e concisa",
	  "prompt_directives": "",
	  "setting": {
	    "world_name": "Brumafum",
	    "era": "Età delle Caldaie",
	    "geography": "Ponti sospesi e canali tossici",
	    "magic_system": "vapore rituale",
	    "technology_level": "steampunk avanzato",
	    "society": "clero, gilde e operai",
	    "rules": ["Il vapore ha un prezzo"],
	    "factions": ["Culto del Vapore"],
	    "cultures": ["Operai della nebbia"],
	    "dangers": ["esplosioni di caldaia"]
	  },
	  "stats_schema": {
	    "vitals": [{"key":"","label":"","starting":10}],
	    "attributes": [{"key":"wit","label":"Acume","starting":3}],
	    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
	    "currency": {"name":"Ingranaggi","starting":5},
	    "has_combat": true
	  }
	}`

	if def := extractStoryJSON(raw); def != nil {
		t.Fatal("extractStoryJSON accepted blank stat definitions")
	}
}

func TestStoryCreatorRevisionFallsBackToPreviousDraftFields(t *testing.T) {
	revision := `{
	  "name": "Le Campane di Vespera",
	  "description": "Una citta in rovina affacciata sul sale.",
	  "genre": "fantasy oscuro",
	  "setting": {
	    "world_name": "Vespera",
	    "era": "Eta delle Maree Spezzate",
	    "geography": "Laguna nera e saline in rovina",
	    "magic_system": "Campane sommerse che chiedono sacrifici",
	    "technology_level": "Rinascimento decadente",
	    "society": "Casate mercantili, culti e scavatori",
	    "rules": ["La magia chiede sempre un prezzo", "Le maree portano voci e debiti", "Le campane non mentono mai", "Il sale conserva e corrompe"],
	    "factions": ["Casata Valcerra", "Custodi delle Campane"],
	    "cultures": ["Scavatori di laguna", "Mercanti del sale"],
	    "dangers": ["Nebbie senzienti", "Predoni delle saline", "Sovraccarichi rituali"]
	  },
	  "stats_schema": {
	    "vitals": [{"key":"hp","label":"Salute","starting":10}],
	    "attributes": [{"key":"agi","label":"Agilita","starting":3},{"key":"wit","label":"Acume","starting":3}],
	    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
	    "currency": {"name":"Corone","starting":5},
	    "has_combat": true
	  }
	}`

	creator, provider := newStoryCreatorForTest(t, validStoryDefinitionJSON, revision)

	if _, err := creator.SendMessage(context.Background(), "Italian dark fantasy with dangerous bells and dry dialogue."); err != nil {
		t.Fatalf("initial SendMessage: %v", err)
	}
	if _, err := creator.SendMessage(context.Background(), "Make the world rules a bit harsher."); err != nil {
		t.Fatalf("revision SendMessage: %v", err)
	}
	if provider.callCount != 2 {
		t.Fatalf("AI call count = %d, want 2", provider.callCount)
	}
	if creator.Definition() == nil {
		t.Fatal("Definition should still be present after revision")
	}
	if creator.Definition().Tone == "" {
		t.Fatal("Tone should fall back from previous draft on revision")
	}
	if creator.Definition().Language == "" {
		t.Fatal("Language should fall back from previous draft on revision")
	}
	if creator.Definition().WritingStyle == "" {
		t.Fatal("WritingStyle should fall back from previous draft on revision")
	}
}

func TestStoryCreatorFailsAfterExhaustingRepairAttempts(t *testing.T) {
	invalid := `not json at all`
	creator, provider := newStoryCreatorForTest(t, invalid, invalid, invalid)

	_, err := creator.SendMessage(context.Background(), "Dark fantasy with bells and salt.")
	if err == nil {
		t.Fatal("expected SendMessage to fail after exhausting repair attempts")
	}
	if provider.callCount != 3 {
		t.Fatalf("AI call count = %d, want 3 including repair attempts", provider.callCount)
	}
	if !containsSubstring(err.Error(), "invalid story definition returned by AI") {
		t.Fatalf("unexpected error: %v", err)
	}
}
