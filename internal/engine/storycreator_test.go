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
	invalid := `{"name":"Broken draft","genre":"fantasy","tone":"dark"}`
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

func TestStoryCreatorFailsAfterExhaustingRepairAttempts(t *testing.T) {
	invalid := `{"name":"Broken draft","genre":"fantasy","tone":"dark"}`
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
