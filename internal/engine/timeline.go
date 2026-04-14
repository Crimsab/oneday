package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/storage"
)

type CharacterTimeline struct {
	CurrentAge int                 `json:"current_age,omitempty"`
	LifeStage  string              `json:"life_stage,omitempty"`
	Milestones []TimelineMilestone `json:"milestones,omitempty"`
}

type TimelineMilestone struct {
	ID        string `json:"id,omitempty"`
	Kind      string `json:"kind,omitempty"` // origin, time_skip, growth, training, bond, trauma, season, custom
	Age       int    `json:"age,omitempty"`
	LifeStage string `json:"life_stage,omitempty"`
	Label     string `json:"label"`
	Detail    string `json:"detail,omitempty"`
	Turn      int    `json:"turn,omitempty"`
}

// LoadCharacterTimeline returns the normalized canonical timeline for a world state.
func LoadCharacterTimeline(world *storage.WorldState) CharacterTimeline {
	return loadCharacterTimeline(world)
}

// CharacterTimelineSummary returns a short header-friendly summary such as
// "Age 8 • childhood".
func CharacterTimelineSummary(timeline CharacterTimeline) string {
	return formatCharacterTimelineSummary(timeline)
}

// RecentTimelineMilestonesSummary returns the last N milestone summaries.
func RecentTimelineMilestonesSummary(timeline CharacterTimeline, limit int) string {
	return formatRecentTimelineMilestones(timeline, limit)
}

func loadCharacterTimeline(world *storage.WorldState) CharacterTimeline {
	if world == nil || strings.TrimSpace(world.CharacterTimelineJSON) == "" || strings.TrimSpace(world.CharacterTimelineJSON) == "{}" {
		return CharacterTimeline{}
	}
	var timeline CharacterTimeline
	if err := json.Unmarshal([]byte(world.CharacterTimelineJSON), &timeline); err != nil {
		return CharacterTimeline{}
	}
	return normalizeCharacterTimeline(timeline)
}

func storeCharacterTimeline(world *storage.WorldState, timeline CharacterTimeline) {
	if world == nil {
		return
	}
	normalized := normalizeCharacterTimeline(timeline)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return
	}
	world.CharacterTimelineJSON = string(payload)
}

func normalizeCharacterTimeline(timeline CharacterTimeline) CharacterTimeline {
	if timeline.CurrentAge < 0 {
		timeline.CurrentAge = 0
	}
	timeline.LifeStage = normalizeLifeStage(firstNonEmpty(timeline.LifeStage, inferLifeStageFromAge(timeline.CurrentAge)))

	if len(timeline.Milestones) == 0 {
		if timeline.CurrentAge == 0 && timeline.LifeStage == "" {
			return CharacterTimeline{}
		}
		return CharacterTimeline{
			CurrentAge: timeline.CurrentAge,
			LifeStage:  timeline.LifeStage,
		}
	}

	seen := map[string]bool{}
	out := make([]TimelineMilestone, 0, len(timeline.Milestones))
	for _, milestone := range timeline.Milestones {
		milestone.Kind = strings.TrimSpace(strings.ToLower(milestone.Kind))
		milestone.Label = strings.TrimSpace(milestone.Label)
		milestone.Detail = strings.TrimSpace(milestone.Detail)
		milestone.LifeStage = normalizeLifeStage(firstNonEmpty(milestone.LifeStage, inferLifeStageFromAge(milestone.Age)))
		if milestone.Age < 0 {
			milestone.Age = 0
		}
		if milestone.Label == "" {
			switch {
			case milestone.Age > 0 && milestone.LifeStage != "":
				milestone.Label = fmt.Sprintf("Age %d - %s", milestone.Age, humanizeLifeStage(milestone.LifeStage))
			case milestone.Age > 0:
				milestone.Label = fmt.Sprintf("Age %d", milestone.Age)
			case milestone.LifeStage != "":
				milestone.Label = humanizeLifeStage(milestone.LifeStage)
			default:
				continue
			}
		}
		if milestone.ID == "" {
			milestone.ID = "timeline:" + uuid.NewString()
		}
		key := strings.ToLower(firstNonEmpty(milestone.ID, fmt.Sprintf("%d|%s|%s", milestone.Age, milestone.LifeStage, milestone.Label)))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, milestone)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Age != out[j].Age {
			if out[i].Age == 0 {
				return false
			}
			if out[j].Age == 0 {
				return true
			}
			return out[i].Age < out[j].Age
		}
		if out[i].Turn != out[j].Turn {
			return out[i].Turn < out[j].Turn
		}
		return out[i].Label < out[j].Label
	})

	timeline.Milestones = out
	if timeline.CurrentAge == 0 {
		for i := len(out) - 1; i >= 0; i-- {
			if out[i].Age > 0 {
				timeline.CurrentAge = out[i].Age
				break
			}
		}
	}
	if timeline.LifeStage == "" {
		for i := len(out) - 1; i >= 0; i-- {
			if out[i].LifeStage != "" {
				timeline.LifeStage = out[i].LifeStage
				break
			}
		}
		timeline.LifeStage = normalizeLifeStage(firstNonEmpty(timeline.LifeStage, inferLifeStageFromAge(timeline.CurrentAge)))
	}

	if timeline.CurrentAge == 0 && timeline.LifeStage == "" && len(timeline.Milestones) == 0 {
		return CharacterTimeline{}
	}
	return timeline
}

func formatCharacterTimelineSummary(timeline CharacterTimeline) string {
	timeline = normalizeCharacterTimeline(timeline)
	if timeline.CurrentAge == 0 && timeline.LifeStage == "" && len(timeline.Milestones) == 0 {
		return ""
	}

	var parts []string
	if timeline.CurrentAge > 0 {
		parts = append(parts, fmt.Sprintf("Age %d", timeline.CurrentAge))
	}
	if timeline.LifeStage != "" {
		parts = append(parts, humanizeLifeStage(timeline.LifeStage))
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " • ")
}

func formatRecentTimelineMilestones(timeline CharacterTimeline, limit int) string {
	timeline = normalizeCharacterTimeline(timeline)
	if len(timeline.Milestones) == 0 {
		return ""
	}
	if limit <= 0 {
		limit = 2
	}
	start := len(timeline.Milestones) - limit
	if start < 0 {
		start = 0
	}
	parts := make([]string, 0, len(timeline.Milestones[start:]))
	for _, milestone := range timeline.Milestones[start:] {
		line := milestone.Label
		if milestone.Detail != "" {
			line += " — " + milestone.Detail
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, " | ")
}

func normalizeLifeStage(stage string) string {
	stage = strings.TrimSpace(strings.ToLower(stage))
	switch stage {
	case "", "unknown":
		return ""
	case "baby", "infant", "newborn", "toddler":
		return "early_childhood"
	case "child", "childhood", "kid":
		return "childhood"
	case "preteen":
		return "pre_adolescence"
	case "teen", "teenager", "adolescent", "adolescence":
		return "adolescence"
	case "young_adult", "young adult":
		return "young_adult"
	case "adult", "adulthood":
		return "adult"
	case "middle_age", "middle age":
		return "middle_age"
	case "elder", "old_age", "old age":
		return "elder"
	default:
		return stage
	}
}

func inferLifeStageFromAge(age int) string {
	switch {
	case age <= 0:
		return ""
	case age <= 4:
		return "early_childhood"
	case age <= 10:
		return "childhood"
	case age <= 12:
		return "pre_adolescence"
	case age <= 17:
		return "adolescence"
	case age <= 29:
		return "young_adult"
	case age <= 54:
		return "adult"
	case age <= 69:
		return "middle_age"
	default:
		return "elder"
	}
}

func humanizeLifeStage(stage string) string {
	switch normalizeLifeStage(stage) {
	case "early_childhood":
		return "early childhood"
	case "childhood":
		return "childhood"
	case "pre_adolescence":
		return "pre-adolescence"
	case "adolescence":
		return "adolescence"
	case "young_adult":
		return "young adult"
	case "adult":
		return "adult"
	case "middle_age":
		return "middle age"
	case "elder":
		return "elder"
	default:
		return strings.TrimSpace(stage)
	}
}

func ApplyTimelineUpdate(world *storage.WorldState, raw map[string]interface{}, currentTurn int) []StateChange {
	if world == nil || len(raw) == 0 {
		return nil
	}

	timeline := loadCharacterTimeline(world)
	oldAge := timeline.CurrentAge
	oldStage := timeline.LifeStage

	age := int(toFloat(raw["age"]))
	ageDelta := int(toFloat(raw["age_delta"]))
	if age <= 0 && ageDelta > 0 && timeline.CurrentAge > 0 {
		age = timeline.CurrentAge + ageDelta
	}
	if age > 0 {
		timeline.CurrentAge = age
	}

	stage := normalizeLifeStage(stringValue(raw["life_stage"]))
	if stage == "" && timeline.CurrentAge > 0 {
		stage = inferLifeStageFromAge(timeline.CurrentAge)
	}
	if stage != "" {
		timeline.LifeStage = stage
	}

	milestone := TimelineMilestone{
		ID:        strings.TrimSpace(stringValue(raw["id"])),
		Kind:      firstNonEmpty(strings.TrimSpace(stringValue(raw["kind"])), "growth"),
		Age:       timeline.CurrentAge,
		LifeStage: timeline.LifeStage,
		Label:     strings.TrimSpace(stringValue(raw["label"])),
		Detail:    strings.TrimSpace(stringValue(raw["detail"])),
		Turn:      currentTurn,
	}
	if milestone.Label == "" && (milestone.Age > 0 || milestone.LifeStage != "" || ageDelta > 0) {
		switch {
		case milestone.Age > 0 && strings.EqualFold(milestone.Kind, "time_skip"):
			milestone.Label = fmt.Sprintf("Age %d", milestone.Age)
		case milestone.Age > 0:
			milestone.Label = fmt.Sprintf("Age %d milestone", milestone.Age)
		case milestone.LifeStage != "":
			milestone.Label = humanizeLifeStage(milestone.LifeStage)
		}
	}

	if milestone.Label != "" || milestone.Detail != "" {
		timeline.Milestones = appendOrUpdateTimelineMilestone(timeline.Milestones, milestone)
	}

	storeCharacterTimeline(world, timeline)
	normalized := loadCharacterTimeline(world)
	if normalized.CurrentAge == oldAge && normalized.LifeStage == oldStage && milestone.Label == "" && milestone.Detail == "" {
		return nil
	}

	title := "Timeline advanced"
	switch {
	case normalized.CurrentAge > 0:
		title = fmt.Sprintf("Age %d", normalized.CurrentAge)
	case milestone.Label != "":
		title = milestone.Label
	case normalized.LifeStage != "":
		title = humanizeLifeStage(normalized.LifeStage)
	}
	if milestone.Label != "" && !strings.EqualFold(strings.TrimSpace(milestone.Label), title) {
		title += " — " + milestone.Label
	}
	detail := milestone.Detail
	if detail == "" && normalized.LifeStage != "" {
		detail = humanizeLifeStage(normalized.LifeStage)
	}

	return []StateChange{
		{
			Target:      "world",
			Field:       "timeline",
			Old:         map[string]interface{}{"age": oldAge, "life_stage": oldStage},
			New:         map[string]interface{}{"age": normalized.CurrentAge, "life_stage": normalized.LifeStage},
			Description: fmt.Sprintf("Timeline advanced: %s", strings.TrimSpace(title)),
		},
	}
}

func appendOrUpdateTimelineMilestone(items []TimelineMilestone, milestone TimelineMilestone) []TimelineMilestone {
	if milestone.ID != "" {
		for i := range items {
			if items[i].ID == milestone.ID {
				if milestone.Kind != "" {
					items[i].Kind = milestone.Kind
				}
				if milestone.Age > 0 {
					items[i].Age = milestone.Age
				}
				if milestone.LifeStage != "" {
					items[i].LifeStage = milestone.LifeStage
				}
				if milestone.Label != "" {
					items[i].Label = milestone.Label
				}
				if milestone.Detail != "" {
					items[i].Detail = milestone.Detail
				}
				if milestone.Turn > 0 {
					items[i].Turn = milestone.Turn
				}
				return items
			}
		}
	}

	for i := range items {
		if milestone.Label != "" && items[i].Age == milestone.Age && strings.EqualFold(items[i].Label, milestone.Label) {
			if milestone.Detail != "" {
				items[i].Detail = milestone.Detail
			}
			if milestone.LifeStage != "" {
				items[i].LifeStage = milestone.LifeStage
			}
			if milestone.Turn > 0 {
				items[i].Turn = milestone.Turn
			}
			return items
		}
	}

	if milestone.ID == "" {
		milestone.ID = "timeline:" + uuid.NewString()
	}
	return append(items, milestone)
}
