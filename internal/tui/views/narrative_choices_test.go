package views

import (
	"encoding/json"
	"testing"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildChoiceItemsResolvesStorySchemaStatLabels(t *testing.T) {
	schema := engine.StatsSchema{
		Attributes: []engine.StatDef{
			{Key: "cha", Label: "Charisma"},
			{Key: "wil", Label: "Willpower"},
		},
	}
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}

	story := &storage.Story{StatsSchemaJSON: string(schemaJSON)}
	narrator := engine.NewNarrator(nil, nil, story, nil, nil, nil, engine.ContextConfig{}, config.GenerationConfig{}, config.ASCIIArtConfig{}, "", 0)
	model := &NarrativeModel{narrator: narrator}

	items := model.buildChoiceItems([]engine.Choice{
		{
			ID:           1,
			Text:         "Talk your way through",
			Intent:       "social",
			Risk:         "medium",
			RelatedStats: []string{"cha", "unknown", "wil", "cha"},
		},
	})

	if len(items) != 1 {
		t.Fatalf("expected 1 choice item, got %d", len(items))
	}
	if len(items[0].RelatedStats) != 2 {
		t.Fatalf("expected 2 resolved stat labels, got %+v", items[0].RelatedStats)
	}
	if items[0].RelatedStats[0] != "Charisma" || items[0].RelatedStats[1] != "Willpower" {
		t.Fatalf("unexpected stat labels: %+v", items[0].RelatedStats)
	}
}

func TestResolveChoiceStatLabelsIgnoresUnknownKeys(t *testing.T) {
	labels := resolveChoiceStatLabels(
		[]string{"per", "missing"},
		map[string]string{"per": "Perception"},
	)

	if len(labels) != 1 || labels[0] != "Perception" {
		t.Fatalf("unexpected labels: %+v", labels)
	}
}
