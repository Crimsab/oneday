package views

import (
	"encoding/json"
	"strings"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
)

func (m *NarrativeModel) buildChoiceItems(choices []engine.Choice) []components.ChoiceItem {
	if len(choices) == 0 {
		return nil
	}

	statLabels := resolveStoryStatLabels(m.narrator.Story())
	items := make([]components.ChoiceItem, len(choices))
	for i, choice := range choices {
		items[i] = components.ChoiceItem{
			ID:           choice.ID,
			Text:         choice.Text,
			Intent:       choice.Intent,
			Risk:         choice.Risk,
			Scope:        choice.Scope,
			Certainty:    choice.Certainty,
			RelatedStats: resolveChoiceStatLabels(choice.RelatedStats, statLabels),
		}
	}
	return items
}

func resolveStoryStatLabels(story *storage.Story) map[string]string {
	if story == nil || story.StatsSchemaJSON == "" || story.StatsSchemaJSON == "null" {
		return nil
	}

	var schema engine.StatsSchema
	if err := json.Unmarshal([]byte(story.StatsSchemaJSON), &schema); err != nil {
		return nil
	}

	labels := map[string]string{}
	addDefs := func(defs []engine.StatDef) {
		for _, def := range defs {
			key := strings.ToLower(strings.TrimSpace(def.Key))
			if key == "" {
				continue
			}
			label := strings.TrimSpace(def.Label)
			if label == "" {
				label = strings.ToUpper(def.Key)
			}
			labels[key] = label
		}
	}

	addDefs(schema.Vitals)
	addDefs(schema.Attributes)
	addDefs(schema.Secondary)
	return labels
}

func resolveChoiceStatLabels(keys []string, statLabels map[string]string) []string {
	if len(keys) == 0 || len(statLabels) == 0 {
		return nil
	}

	labels := make([]string, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		label, ok := statLabels[strings.ToLower(strings.TrimSpace(key))]
		if !ok || label == "" {
			continue
		}
		normalized := strings.ToLower(label)
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		labels = append(labels, label)
	}
	return labels
}
