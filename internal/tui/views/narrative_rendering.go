package views

import (
	"encoding/json"
	"strings"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/rendering"
)

type knownLocationRecord struct {
	Name string `json:"name"`
}

func (m *NarrativeModel) renderNarrativeResponse(nr *engine.NarrativeResponse) string {
	if nr == nil {
		return ""
	}

	renderedMarkdown := rendering.RenderNarrativeMarkdown(rendering.NarrativeInput{
		Narrative:         nr.Narrative,
		DialogueBlocks:    nr.DialogueBlocks,
		EntitiesMentioned: nr.EntitiesMentioned,
		EventCallouts:     nr.EventCallouts,
		KnownEntities:     m.collectKnownEntities(nr),
	})
	if strings.TrimSpace(renderedMarkdown) == "" {
		renderedMarkdown = strings.TrimSpace(nr.Narrative)
	}
	return components.RenderMarkdown(renderedMarkdown)
}

func (m *NarrativeModel) collectKnownEntities(nr *engine.NarrativeResponse) []rendering.KnownEntity {
	entities := make([]rendering.KnownEntity, 0, 32)
	seen := map[string]bool{}

	add := func(name, kind string) {
		name = strings.TrimSpace(name)
		if len([]rune(name)) < 3 {
			return
		}
		key := strings.ToLower(name)
		if seen[key] {
			return
		}
		seen[key] = true
		entities = append(entities, rendering.KnownEntity{
			Name: name,
			Kind: kind,
		})
	}

	if nr != nil {
		add(nr.Location, "location")
	}

	if m == nil || m.narrator == nil {
		return entities
	}

	if world := m.narrator.World(); world != nil {
		add(world.CurrentLocation, "location")
		for _, location := range parseKnownLocationNames(world.KnownLocationsJSON) {
			add(location, "location")
		}
	}

	if story := m.narrator.Story(); story != nil {
		var setting engine.Setting
		if story.SettingJSON != "" && story.SettingJSON != "null" {
			if err := json.Unmarshal([]byte(story.SettingJSON), &setting); err == nil {
				add(setting.WorldName, "world")
				for _, faction := range setting.Factions {
					add(faction, "faction")
				}
			}
		}

		if db := m.narrator.DB(); db != nil {
			if npcs, err := db.ListNPCs(story.ID); err == nil {
				for _, npc := range npcs {
					add(npc.Name, "npc")
				}
			}
			if chapters, err := db.ListChapters(story.ID); err == nil {
				for _, chapter := range chapters {
					add(chapter.Title, "chapter")
				}
			}
		}
	}

	if char := m.narrator.Character(); char != nil {
		add(char.Name, "player")
		for _, skill := range parseSkillNames(char.SkillsJSON) {
			add(skill, "skill")
		}
		for _, title := range parseStatTitles(char.StatsJSON) {
			add(title, "title")
		}
		for _, skill := range parseStatSkillNames(char.StatsJSON) {
			add(skill, "skill")
		}
		for _, item := range parseInventoryNames(char.InventoryJSON) {
			add(item, "item")
		}
	}

	return entities
}

func parseKnownLocationNames(raw string) []string {
	if raw == "" || raw == "null" || raw == "[]" {
		return nil
	}

	var objects []knownLocationRecord
	if err := json.Unmarshal([]byte(raw), &objects); err == nil {
		names := make([]string, 0, len(objects))
		for _, obj := range objects {
			if strings.TrimSpace(obj.Name) != "" {
				names = append(names, obj.Name)
			}
		}
		if len(names) > 0 {
			return names
		}
	}

	var names []string
	if err := json.Unmarshal([]byte(raw), &names); err == nil {
		return names
	}

	return nil
}

func parseSkillNames(raw string) []string {
	if raw == "" || raw == "null" || raw == "{}" {
		return nil
	}

	var skills map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &skills); err != nil {
		return nil
	}

	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}
	return names
}

func parseStatTitles(raw string) []string {
	stats := parseJSONMap(raw)
	if len(stats) == 0 {
		return nil
	}

	return toStringList(stats["titles"])
}

func parseStatSkillNames(raw string) []string {
	stats := parseJSONMap(raw)
	if len(stats) == 0 {
		return nil
	}

	skills, ok := stats["skills"].(map[string]interface{})
	if !ok {
		return nil
	}

	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}
	return names
}

func parseInventoryNames(raw string) []string {
	if raw == "" || raw == "null" || raw == "[]" {
		return nil
	}

	var inventoryRaw interface{}
	if err := json.Unmarshal([]byte(raw), &inventoryRaw); err != nil {
		return nil
	}

	var names []string
	switch inv := inventoryRaw.(type) {
	case []interface{}:
		for _, item := range inv {
			if name := strings.TrimSpace(parseInventoryItem(item).name); name != "" {
				names = append(names, name)
			}
		}
	case map[string]interface{}:
		for _, section := range []string{"backpack", "equipped", "quest"} {
			items, _ := inv[section].([]interface{})
			for _, item := range items {
				if name := strings.TrimSpace(parseInventoryItem(item).name); name != "" {
					names = append(names, name)
				}
			}
		}
	}

	return names
}

func parseJSONMap(raw string) map[string]interface{} {
	if raw == "" || raw == "null" || raw == "{}" {
		return nil
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}
	return decoded
}

func toStringList(value interface{}) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}

	values := make([]string, 0, len(items))
	for _, item := range items {
		if name, ok := item.(string); ok {
			values = append(values, name)
		}
	}
	return values
}
