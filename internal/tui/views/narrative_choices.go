package views

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
)

type storyStatInfo struct {
	Key      string
	Label    string
	Category string
}

func (m *NarrativeModel) buildChoiceItems(choices []engine.Choice) []components.ChoiceItem {
	items, help := m.buildChoicePresentation(choices)
	m.choiceHelp = help
	return items
}

func (m *NarrativeModel) buildChoicePresentation(choices []engine.Choice) ([]components.ChoiceItem, map[int]string) {
	if len(choices) == 0 {
		return nil, map[int]string{}
	}

	statInfo := resolveStoryStatInfo(m.narrator.Story())
	statLabels := labelsFromStatInfo(statInfo)
	currentStats := resolveCharacterStatValues(m.narrator.Character())

	help := make(map[int]string, len(choices))
	items := make([]components.ChoiceItem, len(choices))
	for i, choice := range choices {
		relatedStats := resolveChoiceStatLabels(choice.RelatedStats, statLabels)
		items[i] = components.ChoiceItem{
			ID:           i + 1,
			Text:         choice.Text,
			Intent:       choice.Intent,
			Risk:         choice.Risk,
			Scope:        choice.Scope,
			Certainty:    choice.Certainty,
			RelatedStats: relatedStats,
		}
		help[i+1] = buildChoiceHelp(choice, statInfo, currentStats, m.loc)
	}
	return items, help
}

func resolveStoryStatInfo(story *storage.Story) map[string]storyStatInfo {
	if story == nil || story.StatsSchemaJSON == "" || story.StatsSchemaJSON == "null" {
		return nil
	}

	var schema engine.StatsSchema
	if err := json.Unmarshal([]byte(story.StatsSchemaJSON), &schema); err != nil {
		return nil
	}

	info := map[string]storyStatInfo{}
	addDefs := func(category string, defs []engine.StatDef) {
		for _, def := range defs {
			key := strings.ToLower(strings.TrimSpace(def.Key))
			if key == "" {
				continue
			}
			label := strings.TrimSpace(def.Label)
			if label == "" {
				label = strings.ToUpper(def.Key)
			}
			info[key] = storyStatInfo{
				Key:      key,
				Label:    label,
				Category: category,
			}
		}
	}

	addDefs("vital", schema.Vitals)
	addDefs("attribute", schema.Attributes)
	addDefs("secondary", schema.Secondary)
	return info
}

func labelsFromStatInfo(info map[string]storyStatInfo) map[string]string {
	if len(info) == 0 {
		return nil
	}
	labels := make(map[string]string, len(info))
	for key, item := range info {
		labels[key] = item.Label
	}
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

func resolveCharacterStatValues(char *storage.Character) map[string]string {
	if char == nil || char.StatsJSON == "" || char.StatsJSON == "null" {
		return nil
	}

	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(char.StatsJSON), &stats); err != nil {
		return nil
	}

	values := map[string]string{}
	if vitals, ok := stats["vitals"].(map[string]interface{}); ok {
		for key, raw := range vitals {
			if vitalMap, ok := raw.(map[string]interface{}); ok {
				current := toInt(vitalMap["current"])
				max := toInt(vitalMap["max"])
				values[strings.ToLower(strings.TrimSpace(key))] = fmt.Sprintf("%d/%d", current, max)
			}
		}
	}
	if attrs, ok := stats["attributes"].(map[string]interface{}); ok {
		for key, raw := range attrs {
			values[strings.ToLower(strings.TrimSpace(key))] = fmt.Sprintf("%d", toInt(raw))
		}
	}
	if secondary, ok := stats["secondary"].(map[string]interface{}); ok {
		for key, raw := range secondary {
			values[strings.ToLower(strings.TrimSpace(key))] = fmt.Sprintf("%d", toInt(raw))
		}
	}
	return values
}

func buildChoiceHelp(choice engine.Choice, statInfo map[string]storyStatInfo, currentStats map[string]string, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	lines := []string{
		loc.T("choice.help_choice", strings.TrimSpace(choice.Text)),
		"",
		loc.T("choice.help_signals"),
	}
	if choice.Intent != "" {
		lines = append(lines, loc.T("choice.help_metadata", loc.T("choice.intent"), strings.ToLower(choice.Intent), semanticChoiceHint("intent", choice.Intent, loc)))
	}
	if choice.Risk != "" {
		lines = append(lines, loc.T("choice.help_metadata", loc.T("choice.risk"), strings.ToLower(choice.Risk), semanticChoiceHint("risk", choice.Risk, loc)))
	}
	if choice.Scope != "" {
		lines = append(lines, loc.T("choice.help_metadata", loc.T("choice.scope"), strings.ToLower(choice.Scope), semanticChoiceHint("scope", choice.Scope, loc)))
	}
	if choice.Certainty != "" {
		lines = append(lines, loc.T("choice.help_metadata", loc.T("choice.certainty"), strings.ToLower(choice.Certainty), semanticChoiceHint("certainty", choice.Certainty, loc)))
	}

	statLines := make([]string, 0, len(choice.RelatedStats))
	seen := map[string]bool{}
	for _, rawKey := range choice.RelatedStats {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true

		info, ok := statInfo[key]
		if !ok {
			continue
		}

		value := loc.T("common.unknown_label")
		if currentStats != nil && currentStats[key] != "" {
			value = currentStats[key]
		}
		statLines = append(statLines, loc.T("choice.help_stat", info.Label, loc.T("choice.category."+info.Category), value, statCategoryHint(info.Category, loc)))
	}

	if len(statLines) > 0 {
		lines = append(lines, "")
		lines = append(lines, loc.T("choice.related_stats"))
		lines = append(lines, statLines...)
	}

	if len(lines) == 3 && len(statLines) == 0 {
		lines = append(lines,
			loc.T("choice.no_metadata"),
			"",
			loc.T("choice.freeform_hint"),
		)
	}
	return strings.Join(lines, "\n")
}

func statCategoryHint(category string, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	switch category {
	case "vital":
		return loc.T("choice.stat_hint.vital")
	case "attribute":
		return loc.T("choice.stat_hint.attribute")
	case "secondary":
		return loc.T("choice.stat_hint.secondary")
	default:
		return loc.T("choice.stat_hint.default")
	}
}

func semanticChoiceHint(kind, value string, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	value = strings.ToLower(strings.TrimSpace(value))
	switch kind {
	case "intent":
		switch value {
		case "social":
			return loc.T("choice.hint.intent_social")
		case "explore", "observe", "lore":
			return loc.T("choice.hint.intent_explore")
		case "combat", "aggressive":
			return loc.T("choice.hint.intent_combat")
		case "stealth":
			return loc.T("choice.hint.intent_stealth")
		default:
			return loc.T("choice.hint.intent_default")
		}
	case "risk":
		switch value {
		case "low":
			return loc.T("choice.hint.risk_low")
		case "medium":
			return loc.T("choice.hint.risk_medium")
		case "high":
			return loc.T("choice.hint.risk_high")
		default:
			return loc.T("choice.hint.risk_default")
		}
	case "scope":
		return loc.T("choice.hint.scope")
	case "certainty":
		return loc.T("choice.hint.certainty")
	default:
		return loc.T("choice.hint.default")
	}
}
