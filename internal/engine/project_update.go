package engine

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

func ApplyProjectUpdate(char *storage.Character, stats map[string]interface{}, inventory *[]interface{}, world *storage.WorldState, db *storage.DB, storyID string, raw map[string]interface{}, currentTurn int) []StateChange {
	if world == nil || len(raw) == 0 {
		return nil
	}

	board := loadProjectBoard(world)
	projectIdx := ensureProjectClock(&board, raw, currentTurn)
	if projectIdx < 0 {
		return nil
	}
	project := &board.Projects[projectIdx]
	wasCompleted := strings.EqualFold(project.Status, "completed") || project.CompletedTurn > 0
	action := strings.ToLower(strings.TrimSpace(stringValue(raw["action"])))
	if action == "" {
		action = "advance"
	}

	changes := []StateChange{}
	amount := int(toFloat(raw["amount"]))
	if amount <= 0 {
		amount = 1
	}

	switch action {
	case "advance":
		project.Progress = clampRange(project.Progress+amount, 0, project.Segments)
		project.Status = "active"
		project.UpdatedTurn = currentTurn
	case "setback":
		project.Progress = clampRange(project.Progress-amount, 0, project.Segments)
		project.Status = firstNonEmpty(strings.TrimSpace(stringValue(raw["status"])), "active")
		project.UpdatedTurn = currentTurn
	case "pause":
		project.Status = "paused"
		project.UpdatedTurn = currentTurn
	case "resume":
		project.Status = "active"
		project.UpdatedTurn = currentTurn
	case "complete":
		project.Progress = project.Segments
		project.Status = "completed"
		project.CompletedTurn = currentTurn
		project.UpdatedTurn = currentTurn
	default:
		return nil
	}

	if summary := strings.TrimSpace(stringValue(raw["summary"])); summary != "" {
		project.Summary = summary
	}
	if stakes := strings.TrimSpace(stringValue(raw["stakes"])); stakes != "" {
		project.Stakes = stakes
	}
	if outcome := strings.TrimSpace(stringValue(raw["outcome"])); outcome != "" {
		project.Outcome = outcome
	}
	project.Links = normalizeProjectLinks(append(project.Links, toProjectLinks(raw["links"])...))
	project.Rewards = normalizeProjectRewards(append(project.Rewards, toProjectRewards(raw["rewards"])...))

	if cost := int(toFloat(raw["currency_cost"])); cost > 0 {
		currentCurrency := int(toFloat(stats["currency"]))
		stats["currency"] = maxInt(0, currentCurrency-cost)
		changes = append(changes, StateChange{
			Target:      "character",
			Field:       "currency",
			New:         stats["currency"],
			Description: fmt.Sprintf("Project cost paid: %s (-%d)", project.Title, cost),
		})
	}

	if frontChanges := applyProjectPressure(world, raw, project.Title, currentTurn); len(frontChanges) > 0 {
		changes = append(changes, frontChanges...)
	}
	if reaction := applyProjectFailForward(world, raw, project.Title, currentTurn); reaction != nil {
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("reaction.%s", reaction.Title),
			New:         reaction.Title,
			Description: fmt.Sprintf("World reacts: %s", reaction.Title),
		})
	}
	if strings.EqualFold(project.Status, "completed") && !wasCompleted {
		changes = append(changes, applyProjectCompletionRewards(char, stats, inventory, world, db, storyID, project, currentTurn)...)
	}

	storeProjectBoard(world, board)
	changes = append([]StateChange{{
		Target:      "world",
		Field:       fmt.Sprintf("project.%s", project.Title),
		New:         fmt.Sprintf("%d/%d", project.Progress, project.Segments),
		Description: fmt.Sprintf("Project %s: %s", action, project.Title),
	}}, changes...)
	return changes
}

func ensureProjectClock(board *ProjectBoard, raw map[string]interface{}, currentTurn int) int {
	if board == nil {
		return -1
	}
	projectID := strings.TrimSpace(stringValue(raw["id"]))
	title := strings.TrimSpace(stringValue(raw["title"]))
	if idx := findProjectClockIndex(board.Projects, projectID, title); idx >= 0 {
		return idx
	}
	if title == "" {
		return -1
	}
	project := ProjectClock{
		ID:          firstNonEmpty(projectID, "project:"+slugKey(title)),
		Title:       title,
		Kind:        strings.TrimSpace(strings.ToLower(stringValue(raw["kind"]))),
		Summary:     strings.TrimSpace(stringValue(raw["summary"])),
		Status:      "active",
		Segments:    maxInt(1, int(toFloat(raw["segments"]))),
		Progress:    clampRange(int(toFloat(raw["progress"])), 0, maxInt(1, int(toFloat(raw["segments"])))),
		StartedTurn: currentTurn,
		UpdatedTurn: currentTurn,
		Owner:       strings.TrimSpace(stringValue(raw["owner"])),
		Location:    strings.TrimSpace(stringValue(raw["location"])),
		Stakes:      strings.TrimSpace(stringValue(raw["stakes"])),
		Links:       normalizeProjectLinks(toProjectLinks(raw["links"])),
		Rewards:     normalizeProjectRewards(toProjectRewards(raw["rewards"])),
	}
	board.Projects = append(board.Projects, project)
	return len(board.Projects) - 1
}

func findProjectClockIndex(items []ProjectClock, id, title string) int {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	for i, item := range items {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if title != "" && strings.EqualFold(item.Title, title) {
			return i
		}
	}
	return -1
}

func toProjectLinks(raw interface{}) []ProjectLink {
	items := toObjectMaps(raw)
	if len(items) == 0 {
		return nil
	}
	links := make([]ProjectLink, 0, len(items))
	for _, item := range items {
		links = append(links, ProjectLink{
			Kind:  strings.TrimSpace(stringValue(item["kind"])),
			RefID: strings.TrimSpace(stringValue(item["ref_id"])),
			Label: strings.TrimSpace(stringValue(item["label"])),
		})
	}
	return links
}

func toProjectRewards(raw interface{}) []ProjectReward {
	items := toObjectMaps(raw)
	if len(items) == 0 {
		return nil
	}
	rewards := make([]ProjectReward, 0, len(items))
	for _, item := range items {
		rewards = append(rewards, ProjectReward{
			Kind:   strings.TrimSpace(stringValue(item["kind"])),
			Label:  strings.TrimSpace(stringValue(item["label"])),
			Detail: strings.TrimSpace(stringValue(item["detail"])),
		})
	}
	return rewards
}

func applyProjectPressure(world *storage.WorldState, raw map[string]interface{}, title string, currentTurn int) []StateChange {
	if world == nil {
		return nil
	}
	if strings.TrimSpace(stringValue(raw["front_id"])) == "" && strings.TrimSpace(stringValue(raw["front_title"])) == "" {
		return nil
	}
	payload := map[string]interface{}{
		"front_id":        raw["front_id"],
		"front_title":     raw["front_title"],
		"front_advance":   raw["front_advance"],
		"pressure_region": raw["pressure_region"],
		"pressure_kind":   raw["pressure_kind"],
		"pressure_change": raw["pressure_change"],
		"pressure_value":  raw["pressure_value"],
		"pressure_detail": firstNonEmpty(strings.TrimSpace(stringValue(raw["pressure_detail"])), fmt.Sprintf("Time spent on %s lets the wider world move.", title)),
	}
	return applyFailForwardToFronts(world, payload, currentTurn)
}

func applyProjectFailForward(world *storage.WorldState, raw map[string]interface{}, title string, currentTurn int) *WorldReaction {
	if world == nil {
		return nil
	}
	reactionTitle := strings.TrimSpace(stringValue(raw["fail_forward_title"]))
	if reactionTitle == "" {
		return nil
	}
	reaction := WorldReaction{
		ID:          "project-setback:" + slugKey(title) + ":" + slugKey(reactionTitle),
		Kind:        firstNonEmpty(strings.TrimSpace(stringValue(raw["fail_forward_kind"])), "setback"),
		Title:       reactionTitle,
		Detail:      firstNonEmpty(strings.TrimSpace(stringValue(raw["fail_forward_detail"])), fmt.Sprintf("%s runs into trouble.", title)),
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
	return &reaction
}
