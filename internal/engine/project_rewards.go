package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

func applyProjectCompletionRewards(char *storage.Character, stats map[string]interface{}, inventory *[]interface{}, world *storage.WorldState, db *storage.DB, storyID string, project *ProjectClock, currentTurn int) []StateChange {
	if project == nil {
		return nil
	}

	rewards := normalizeProjectRewards(project.Rewards)
	applied := make([]StateChange, 0, len(rewards)+1)
	addedReaction := false

	for _, reward := range rewards {
		switch strings.ToLower(strings.TrimSpace(reward.Kind)) {
		case "skill":
			if change, ok := grantProjectSkillReward(char, stats, reward); ok {
				applied = append(applied, change)
			}
		case "trait":
			if change, ok := grantProjectTraitReward(char, stats, reward); ok {
				applied = append(applied, change)
			}
		case "title":
			if change, ok := grantProjectTitleReward(stats, reward); ok {
				applied = append(applied, change)
			}
		case "item", "gear", "recipe":
			if change, ok := grantProjectItemReward(inventory, project, reward); ok {
				applied = append(applied, change)
			}
		case "relationship", "bond":
			applied = append(applied, grantProjectRelationshipReward(world, db, storyID, project, reward, currentTurn)...)
		case "reaction":
			if change, ok := grantProjectReactionReward(world, project, reward, currentTurn); ok {
				applied = append(applied, change)
				addedReaction = true
			}
		case "hook":
			if change, ok := grantProjectHookReward(world, project, reward, currentTurn); ok {
				applied = append(applied, change)
			}
		}
	}

	if !addedReaction {
		if change, ok := grantProjectCompletionEcho(world, project, currentTurn); ok {
			applied = append(applied, change)
		}
	}

	return applied
}

func grantProjectSkillReward(char *storage.Character, stats map[string]interface{}, reward ProjectReward) (StateChange, bool) {
	skillName := strings.TrimSpace(reward.Label)
	if skillName == "" {
		return StateChange{}, false
	}

	skills := toSkillsMap(stats["skills"])
	if _, exists := skills[skillName]; exists {
		return StateChange{}, false
	}

	skills[skillName] = map[string]interface{}{"level": 1, "xp": 0}
	stats["skills"] = skills
	if char != nil {
		if payload, err := json.Marshal(skills); err == nil {
			char.SkillsJSON = string(payload)
		}
	}

	return StateChange{
		Target:      "character",
		Field:       fmt.Sprintf("skills.%s", skillName),
		New:         skills[skillName],
		Description: fmt.Sprintf("Project reward learned: %s", skillName),
	}, true
}

func grantProjectTraitReward(char *storage.Character, stats map[string]interface{}, reward ProjectReward) (StateChange, bool) {
	traitName := strings.TrimSpace(reward.Label)
	if traitName == "" {
		return StateChange{}, false
	}

	traits := toStringSlice(stats["traits"])
	for _, existing := range traits {
		if strings.EqualFold(existing, traitName) {
			return StateChange{}, false
		}
	}

	traits = append(traits, traitName)
	stats["traits"] = toInterfaceSlice(traits)
	if char != nil {
		if payload, err := json.Marshal(traits); err == nil {
			char.TraitsJSON = string(payload)
		}
	}

	return StateChange{
		Target:      "character",
		Field:       "traits",
		New:         traitName,
		Description: fmt.Sprintf("Project reward gained trait: %s", traitName),
	}, true
}

func grantProjectTitleReward(stats map[string]interface{}, reward ProjectReward) (StateChange, bool) {
	titleName := strings.TrimSpace(reward.Label)
	if titleName == "" {
		return StateChange{}, false
	}

	titles := toStringSlice(stats["titles"])
	for _, existing := range titles {
		if strings.EqualFold(existing, titleName) {
			return StateChange{}, false
		}
	}

	titles = append(titles, titleName)
	stats["titles"] = toInterfaceSlice(titles)
	return StateChange{
		Target:      "character",
		Field:       "titles",
		New:         titleName,
		Description: fmt.Sprintf("Project reward earned title: %s", titleName),
	}, true
}

func grantProjectItemReward(inventory *[]interface{}, project *ProjectClock, reward ProjectReward) (StateChange, bool) {
	if inventory == nil {
		return StateChange{}, false
	}

	itemName := strings.TrimSpace(reward.Label)
	if itemName == "" {
		return StateChange{}, false
	}

	items := *inventory
	for _, raw := range items {
		switch item := raw.(type) {
		case string:
			if strings.EqualFold(strings.TrimSpace(item), itemName) {
				return StateChange{}, false
			}
		case map[string]interface{}:
			if strings.EqualFold(strings.TrimSpace(stringValue(item["name"])), itemName) {
				return StateChange{}, false
			}
		}
	}

	item := map[string]interface{}{
		"name":        itemName,
		"type":        firstNonEmpty(strings.TrimSpace(reward.Kind), cueProjectItemType(project)),
		"description": strings.TrimSpace(reward.Detail),
	}
	*inventory = append(items, item)

	return StateChange{
		Target:      "character",
		Field:       "inventory",
		New:         item,
		Description: fmt.Sprintf("Project reward gained item: %s", itemName),
	}, true
}

func grantProjectRelationshipReward(world *storage.WorldState, db *storage.DB, storyID string, project *ProjectClock, reward ProjectReward, currentTurn int) []StateChange {
	if db == nil || strings.TrimSpace(storyID) == "" || project == nil {
		return nil
	}

	npc := findProjectRewardNPC(db, storyID, project, reward)
	if npc == nil {
		return nil
	}

	axes := loadRelationshipAxes(npc)
	beforeAxes := axes
	beforeDisposition := npc.Disposition
	axes.Trust = clampRelationshipValue(axes.Trust + 8)
	axes.Respect = clampRelationshipValue(axes.Respect + 4)
	storeRelationshipAxes(npc, axes)
	npc.Disposition = clampRange(npc.Disposition+6, -100, 100)

	note := strings.TrimSpace(reward.Detail)
	if note == "" {
		note = fmt.Sprintf("Project completed together: %s", project.Title)
	}
	npc.NotesOnProtagonist = appendJSONStringUnique(npc.NotesOnProtagonist, fmt.Sprintf("Turn %d: %s", currentTurn, note))
	_ = db.UpdateNPC(npc)

	changes := []StateChange{}
	if axes.Trust != beforeAxes.Trust {
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("relationship.%s.trust", npc.Name),
			Old:         beforeAxes.Trust,
			New:         axes.Trust,
			Description: fmt.Sprintf("%s trust deepens after %s", npc.Name, project.Title),
		})
	}
	if axes.Respect != beforeAxes.Respect {
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("relationship.%s.respect", npc.Name),
			Old:         beforeAxes.Respect,
			New:         axes.Respect,
			Description: fmt.Sprintf("%s respect rises after %s", npc.Name, project.Title),
		})
	}
	if npc.Disposition != beforeDisposition {
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("npc.%s.disposition", npc.Name),
			Old:         beforeDisposition,
			New:         npc.Disposition,
			Description: fmt.Sprintf("%s warms to you after %s", npc.Name, project.Title),
		})
	}
	if world != nil {
		if change, ok := grantProjectReactionReward(world, project, ProjectReward{
			Kind:   "reaction",
			Label:  fmt.Sprintf("%s stands closer after %s", npc.Name, project.Title),
			Detail: note,
		}, currentTurn); ok {
			changes = append(changes, change)
		}
	}
	return changes
}

func grantProjectReactionReward(world *storage.WorldState, project *ProjectClock, reward ProjectReward, currentTurn int) (StateChange, bool) {
	if world == nil || project == nil {
		return StateChange{}, false
	}

	title := strings.TrimSpace(reward.Label)
	if title == "" {
		return StateChange{}, false
	}
	reaction := WorldReaction{
		ID:          fmt.Sprintf("project-reward:%s:%s", slugKey(project.ID), slugKey(title)),
		Kind:        "project",
		Title:       title,
		Detail:      firstNonEmpty(strings.TrimSpace(reward.Detail), strings.TrimSpace(project.Outcome), strings.TrimSpace(project.Summary)),
		Status:      "active",
		SourceTurn:  currentTurn,
		CreatedTurn: currentTurn,
	}
	reactions := loadWorldReactions(world)
	if idx := findWorldReactionIndex(reactions, reaction.ID, reaction.Title); idx >= 0 {
		reactions[idx] = reaction
	} else {
		reactions = append(reactions, reaction)
	}
	storeWorldReactions(world, reactions)

	return StateChange{
		Target:      "world",
		Field:       fmt.Sprintf("reaction.%s", reaction.Title),
		New:         reaction.Title,
		Description: fmt.Sprintf("Project fallout: %s", reaction.Title),
	}, true
}

func grantProjectHookReward(world *storage.WorldState, project *ProjectClock, reward ProjectReward, currentTurn int) (StateChange, bool) {
	if world == nil || project == nil {
		return StateChange{}, false
	}

	title := strings.TrimSpace(reward.Label)
	if title == "" {
		return StateChange{}, false
	}
	hook := StoryHook{
		ID:          fmt.Sprintf("project-hook:%s:%s", slugKey(project.ID), slugKey(title)),
		Kind:        "project",
		Title:       title,
		Detail:      firstNonEmpty(strings.TrimSpace(reward.Detail), strings.TrimSpace(project.Outcome), strings.TrimSpace(project.Summary)),
		Status:      "active",
		SourceTurn:  currentTurn,
		UpdatedTurn: currentTurn,
	}
	hooks := loadStoryHooks(world)
	if idx := findStoryHookIndex(hooks, hook.ID, hook.Title); idx >= 0 {
		hooks[idx] = hook
	} else {
		hooks = append(hooks, hook)
	}
	storeStoryHooks(world, hooks)

	return StateChange{
		Target:      "world",
		Field:       fmt.Sprintf("hook.%s", hook.Title),
		New:         hook.Title,
		Description: fmt.Sprintf("Project opens a new thread: %s", hook.Title),
	}, true
}

func grantProjectCompletionEcho(world *storage.WorldState, project *ProjectClock, currentTurn int) (StateChange, bool) {
	if world == nil || project == nil || strings.TrimSpace(project.Outcome) == "" {
		return StateChange{}, false
	}
	return grantProjectReactionReward(world, project, ProjectReward{
		Kind:   "reaction",
		Label:  fmt.Sprintf("Project completed: %s", project.Title),
		Detail: project.Outcome,
	}, currentTurn)
}

func findProjectRewardNPC(db *storage.DB, storyID string, project *ProjectClock, reward ProjectReward) *storage.NPC {
	if db == nil || project == nil {
		return nil
	}

	candidates := []string{
		strings.TrimSpace(reward.Label),
		strings.TrimSpace(project.Owner),
	}
	for _, link := range project.Links {
		if strings.EqualFold(strings.TrimSpace(link.Kind), "npc") {
			candidates = append(candidates, strings.TrimSpace(link.Label), strings.TrimSpace(link.RefID))
		}
	}

	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		key := strings.ToLower(candidate)
		if seen[key] {
			continue
		}
		seen[key] = true
		npc, err := db.GetNPCByName(storyID, candidate)
		if err == nil && npc != nil {
			return npc
		}
	}
	return nil
}

func cueProjectItemType(project *ProjectClock) string {
	if project == nil {
		return "quest"
	}
	switch strings.ToLower(strings.TrimSpace(project.Kind)) {
	case "crafting":
		return "tool"
	case "training":
		return "tool"
	default:
		return "quest"
	}
}
