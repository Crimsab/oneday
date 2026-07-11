package engine

import (
	"fmt"
	"sort"
	"strings"
)

func buildProjectCodexEntry(project ProjectClock) CodexEntry {
	if strings.TrimSpace(project.Title) == "" {
		return CodexEntry{}
	}

	entry := CodexEntry{
		ID:       codexProjectEntryID(project.ID),
		Category: "projects",
		Title:    project.Title,
		Subtitle: strings.Title(strings.ToLower(firstNonEmpty(project.Kind, "project"))),
		Summary:  strings.TrimSpace(firstNonEmpty(project.Summary, project.Outcome, project.Stakes)),
	}

	overview := []string{
		"Status: " + strings.ToLower(firstNonEmpty(project.Status, "active")),
		fmt.Sprintf("Progress: %d/%d", project.Progress, maxInt(1, project.Segments)),
	}
	if summary := strings.TrimSpace(project.Summary); summary != "" {
		overview = append([]string{summary}, overview...)
	}
	if project.Owner != "" {
		overview = append(overview, "Owner: "+project.Owner)
	}
	if project.Location != "" {
		overview = append(overview, "Location: "+project.Location)
	}
	if project.Stakes != "" {
		overview = append(overview, "Stakes: "+project.Stakes)
	}
	if project.StartedTurn > 0 {
		overview = append(overview, fmt.Sprintf("Started on turn %d", project.StartedTurn))
	}
	if project.UpdatedTurn > 0 && project.UpdatedTurn != project.StartedTurn {
		overview = append(overview, fmt.Sprintf("Updated on turn %d", project.UpdatedTurn))
	}
	if project.CompletedTurn > 0 {
		overview = append(overview, fmt.Sprintf("Completed on turn %d", project.CompletedTurn))
	}
	entry.Sections = append(entry.Sections, CodexSection{Title: "Overview", Lines: compactLines(overview)})

	if outcome := strings.TrimSpace(project.Outcome); outcome != "" {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Outcome", Lines: []string{outcome}})
	}
	if rewards := formatProjectRewardLines(project.Rewards); len(rewards) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Rewards", Lines: rewards})
	}

	entry.Related = append(entry.Related, projectCodexLinks(project)...)
	return entry
}

func buildProtagonistProjectLines(board ProjectBoard) ([]string, []string) {
	active := []string{}
	completed := []string{}
	for _, project := range sortedProjectClocks(board) {
		line := project.Title
		if project.Segments > 0 {
			line += fmt.Sprintf(" %d/%d", project.Progress, project.Segments)
		}
		if project.Kind != "" {
			line += " · " + project.Kind
		}
		if strings.EqualFold(project.Status, "completed") {
			if outcome := strings.TrimSpace(project.Outcome); outcome != "" {
				line += " · " + outcome
			} else if reward := firstProjectRewardLabel(project.Rewards); reward != "" {
				line += " · reward: " + reward
			}
			completed = append(completed, line)
			continue
		}
		if !strings.EqualFold(project.Status, "active") {
			line += " · " + project.Status
		}
		if stakes := strings.TrimSpace(project.Stakes); stakes != "" {
			line += " · stakes: " + stakes
		}
		active = append(active, line)
	}
	return active, completed
}

func buildProjectStateSummaryLines(board ProjectBoard) ([]string, []string) {
	active := []string{}
	completed := []string{}
	for _, project := range sortedProjectClocks(board) {
		line := project.Title
		if project.Segments > 0 {
			line += fmt.Sprintf(" %d/%d", project.Progress, project.Segments)
		}
		if project.Kind != "" {
			line += fmt.Sprintf(" [%s]", project.Kind)
		}
		if strings.EqualFold(project.Status, "completed") {
			if outcome := strings.TrimSpace(project.Outcome); outcome != "" {
				line += " — " + outcome
			}
			completed = append(completed, line)
			if len(completed) >= 3 {
				continue
			}
			continue
		}
		if !strings.EqualFold(project.Status, "active") {
			line += fmt.Sprintf(" {%s}", project.Status)
		}
		active = append(active, line)
		if len(active) >= 4 {
			continue
		}
	}
	if len(active) > 4 {
		active = active[:4]
	}
	if len(completed) > 3 {
		completed = completed[:3]
	}
	return active, completed
}

func formatProjectRewardLines(rewards []ProjectReward) []string {
	lines := []string{}
	for _, reward := range normalizeProjectRewards(rewards) {
		line := strings.TrimSpace(reward.Label)
		if line == "" {
			continue
		}
		if reward.Kind != "" {
			line = strings.Title(strings.ReplaceAll(reward.Kind, "_", " ")) + ": " + line
		}
		if reward.Detail != "" {
			line += " — " + reward.Detail
		}
		lines = append(lines, line)
	}
	return compactLines(lines)
}

func projectCodexLinks(project ProjectClock) []CodexLink {
	links := []CodexLink{}
	for _, link := range normalizeProjectLinks(project.Links) {
		if codexLink, ok := codexLinkFromProjectLink(link); ok {
			links = append(links, codexLink)
		}
	}
	if owner := strings.TrimSpace(project.Owner); owner != "" {
		links = append(links, CodexLink{EntryID: codexNPCEntryID(owner), Label: owner})
	}
	if location := strings.TrimSpace(project.Location); location != "" {
		links = append(links, CodexLink{EntryID: codexLocationEntryID(location), Label: location})
	}
	for _, reward := range normalizeProjectRewards(project.Rewards) {
		if !strings.EqualFold(reward.Kind, "relationship") && !strings.EqualFold(reward.Kind, "bond") {
			continue
		}
		name := strings.TrimSpace(reward.Label)
		if name == "" {
			continue
		}
		links = append(links, CodexLink{EntryID: codexNPCEntryID(name), Label: name})
	}
	return appendUniqueLinks(links)
}

func codexLinkFromProjectLink(link ProjectLink) (CodexLink, bool) {
	kind := strings.ToLower(strings.TrimSpace(link.Kind))
	refID := strings.TrimSpace(link.RefID)
	label := strings.TrimSpace(link.Label)
	if refID == "" && label == "" {
		return CodexLink{}, false
	}
	switch {
	case strings.Contains(refID, ":"):
		return CodexLink{EntryID: refID, Label: firstNonEmpty(label, refID)}, true
	case kind == "npc":
		return CodexLink{EntryID: codexNPCEntryID(firstNonEmpty(label, refID)), Label: firstNonEmpty(label, refID)}, true
	case kind == "place" || kind == "location":
		return CodexLink{EntryID: codexLocationEntryID(firstNonEmpty(label, refID)), Label: firstNonEmpty(label, refID)}, true
	case kind == "front":
		return CodexLink{EntryID: codexFrontEntryID(firstNonEmpty(refID, label)), Label: firstNonEmpty(label, refID)}, true
	case kind == "hook":
		return CodexLink{EntryID: codexThreadEntryID("hook", refID), Label: firstNonEmpty(label, refID)}, true
	case kind == "reaction":
		return CodexLink{EntryID: codexThreadEntryID("reaction", refID), Label: firstNonEmpty(label, refID)}, true
	case kind == "faction":
		return CodexLink{EntryID: codexFactionEntryID(firstNonEmpty(label, refID)), Label: firstNonEmpty(label, refID)}, true
	case kind == "investigation":
		return CodexLink{EntryID: codexInvestigationEntryID(firstNonEmpty(refID, label)), Label: firstNonEmpty(label, refID)}, true
	case kind == "project":
		return CodexLink{EntryID: codexProjectEntryID(firstNonEmpty(refID, label)), Label: firstNonEmpty(label, refID)}, true
	default:
		return CodexLink{}, false
	}
}

func projectTouchesCodexEntry(project ProjectClock, entryID string, entry CodexEntry) bool {
	for _, link := range projectCodexLinks(project) {
		if strings.EqualFold(link.EntryID, entryID) {
			return true
		}
	}
	if entry.Category == "people" && strings.EqualFold(strings.TrimSpace(project.Owner), strings.TrimSpace(entry.Title)) {
		return true
	}
	if entry.Category == "places" && strings.EqualFold(strings.TrimSpace(project.Location), strings.TrimSpace(entry.Title)) {
		return true
	}
	for _, reward := range normalizeProjectRewards(project.Rewards) {
		if (strings.EqualFold(reward.Kind, "relationship") || strings.EqualFold(reward.Kind, "bond")) &&
			entry.Category == "people" &&
			strings.EqualFold(strings.TrimSpace(reward.Label), strings.TrimSpace(entry.Title)) {
			return true
		}
	}
	return false
}

func codexProjectEntryID(id string) string {
	return "projects:" + strings.TrimSpace(id)
}

func sortedProjectClocks(board ProjectBoard) []ProjectClock {
	if len(board.Projects) == 0 {
		return nil
	}
	items := append([]ProjectClock{}, board.Projects...)
	sort.SliceStable(items, func(i, j int) bool {
		leftCompleted := strings.EqualFold(items[i].Status, "completed")
		rightCompleted := strings.EqualFold(items[j].Status, "completed")
		if leftCompleted != rightCompleted {
			return !leftCompleted
		}
		if items[i].UpdatedTurn != items[j].UpdatedTurn {
			return items[i].UpdatedTurn > items[j].UpdatedTurn
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
	return items
}

func firstProjectRewardLabel(rewards []ProjectReward) string {
	for _, reward := range normalizeProjectRewards(rewards) {
		if label := strings.TrimSpace(reward.Label); label != "" {
			return label
		}
	}
	return ""
}
