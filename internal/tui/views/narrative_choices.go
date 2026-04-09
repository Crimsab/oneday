package views

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
)

type storyStatInfo struct {
	Key      string
	Label    string
	Category string
}

func (m *NarrativeModel) buildChoiceItems(choices []engine.Choice) []components.ChoiceItem {
	if len(choices) == 0 {
		m.choiceHelp = map[int]string{}
		return nil
	}

	statInfo := resolveStoryStatInfo(m.narrator.Story())
	statLabels := labelsFromStatInfo(statInfo)
	currentStats := resolveCharacterStatValues(m.narrator.Character())

	m.choiceHelp = make(map[int]string, len(choices))
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
		m.choiceHelp[i+1] = buildChoiceHelp(choice, statInfo, currentStats)
	}
	return items
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

func buildChoiceHelp(choice engine.Choice, statInfo map[string]storyStatInfo, currentStats map[string]string) string {
	lines := []string{
		fmt.Sprintf("Choice: %s", strings.TrimSpace(choice.Text)),
		"",
		"This choice signals:",
	}
	if choice.Intent != "" {
		lines = append(lines, fmt.Sprintf("- intent: %s — %s", strings.ToLower(choice.Intent), semanticChoiceHint("intent", choice.Intent)))
	}
	if choice.Risk != "" {
		lines = append(lines, fmt.Sprintf("- risk: %s — %s", strings.ToLower(choice.Risk), semanticChoiceHint("risk", choice.Risk)))
	}
	if choice.Scope != "" {
		lines = append(lines, fmt.Sprintf("- scope: %s — %s", strings.ToLower(choice.Scope), semanticChoiceHint("scope", choice.Scope)))
	}
	if choice.Certainty != "" {
		lines = append(lines, fmt.Sprintf("- certainty: %s — %s", strings.ToLower(choice.Certainty), semanticChoiceHint("certainty", choice.Certainty)))
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

		value := "unknown"
		if currentStats != nil && currentStats[key] != "" {
			value = currentStats[key]
		}
		statLines = append(statLines, fmt.Sprintf("- %s [%s]: current %s. %s", info.Label, info.Category, value, statCategoryHint(info.Category)))
	}

	if len(statLines) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Related stats:")
		lines = append(lines, statLines...)
	}

	if len(lines) == 3 && len(statLines) == 0 {
		lines = append(lines,
			"- no structured metadata was provided for this choice",
			"",
			"Treat it as a freeform narrative action: judge risk and likely stat influence from the scene text.",
		)
	}
	return strings.Join(lines, "\n")
}

func statCategoryHint(category string) string {
	switch category {
	case "vital":
		return "A core resource; low values usually mean immediate danger or exhaustion."
	case "attribute":
		return "A core capability used in actions, checks, and narrative judgment."
	case "secondary":
		return "A progression or world-facing metric that tracks longer-term standing."
	default:
		return "This is a story-defined stat used by the narrator and game systems."
	}
}

func semanticChoiceHint(kind, value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch kind {
	case "intent":
		switch value {
		case "social":
			return "leans on rapport, deception, persuasion, or etiquette"
		case "explore", "observe", "lore":
			return "focuses on information, discovery, or environmental reading"
		case "combat", "aggressive":
			return "pushes the scene toward confrontation or force"
		case "stealth":
			return "prioritizes subtlety, concealment, or low attention"
		default:
			return "describes the narrative purpose of the action"
		}
	case "risk":
		switch value {
		case "low":
			return "safer play with fewer likely downsides"
		case "medium":
			return "balanced risk with meaningful upside and downside"
		case "high":
			return "big swing: stronger payoff, stronger danger"
		default:
			return "describes how dangerous or costly the action may be"
		}
	case "scope":
		return "shows what part of the scene this choice mainly affects"
	case "certainty":
		return "shows how predictable the likely outcome is"
	default:
		return "structured hint provided by the narrator"
	}
}
