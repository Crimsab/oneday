package engine

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/storage"
)

type ProjectLink struct {
	Kind  string `json:"kind,omitempty"`
	RefID string `json:"ref_id,omitempty"`
	Label string `json:"label,omitempty"`
}

type ProjectReward struct {
	Kind   string `json:"kind,omitempty"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

type ProjectClock struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Kind          string          `json:"kind,omitempty"`
	Summary       string          `json:"summary,omitempty"`
	Status        string          `json:"status,omitempty"`
	Progress      int             `json:"progress,omitempty"`
	Segments      int             `json:"segments,omitempty"`
	StartedTurn   int             `json:"started_turn,omitempty"`
	UpdatedTurn   int             `json:"updated_turn,omitempty"`
	CompletedTurn int             `json:"completed_turn,omitempty"`
	Owner         string          `json:"owner,omitempty"`
	Location      string          `json:"location,omitempty"`
	Stakes        string          `json:"stakes,omitempty"`
	Outcome       string          `json:"outcome,omitempty"`
	Rewards       []ProjectReward `json:"rewards,omitempty"`
	Links         []ProjectLink   `json:"links,omitempty"`
}

type ProjectBoard struct {
	Projects []ProjectClock `json:"projects,omitempty"`
}

func loadProjectBoard(world *storage.WorldState) ProjectBoard {
	if world == nil || strings.TrimSpace(world.ProjectClocksJSON) == "" || strings.TrimSpace(world.ProjectClocksJSON) == "{}" {
		return ProjectBoard{}
	}
	var board ProjectBoard
	if err := json.Unmarshal([]byte(world.ProjectClocksJSON), &board); err != nil {
		return ProjectBoard{}
	}
	return normalizeProjectBoard(board)
}

func storeProjectBoard(world *storage.WorldState, board ProjectBoard) {
	if world == nil {
		return
	}
	normalized := normalizeProjectBoard(board)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return
	}
	world.ProjectClocksJSON = string(payload)
}

func normalizeProjectBoard(board ProjectBoard) ProjectBoard {
	if len(board.Projects) == 0 {
		return ProjectBoard{}
	}
	out := ProjectBoard{Projects: make([]ProjectClock, 0, len(board.Projects))}
	seen := map[string]bool{}
	for _, project := range board.Projects {
		project.Title = strings.TrimSpace(project.Title)
		if project.Title == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(project.ID), project.Title))
		if seen[key] {
			continue
		}
		seen[key] = true
		if project.ID == "" {
			project.ID = "project:" + uuid.NewString()
		}
		project.Kind = strings.TrimSpace(strings.ToLower(project.Kind))
		project.Summary = strings.TrimSpace(project.Summary)
		project.Status = firstNonEmpty(strings.TrimSpace(project.Status), "active")
		project.Segments = maxInt(1, project.Segments)
		project.Progress = clampRange(project.Progress, 0, project.Segments)
		project.Owner = strings.TrimSpace(project.Owner)
		project.Location = strings.TrimSpace(project.Location)
		project.Stakes = strings.TrimSpace(project.Stakes)
		project.Outcome = strings.TrimSpace(project.Outcome)
		if strings.EqualFold(project.Status, "completed") && project.CompletedTurn == 0 {
			project.CompletedTurn = project.UpdatedTurn
		}
		project.Rewards = normalizeProjectRewards(project.Rewards)
		project.Links = normalizeProjectLinks(project.Links)
		out.Projects = append(out.Projects, project)
	}
	return out
}

func normalizeProjectRewards(items []ProjectReward) []ProjectReward {
	if len(items) == 0 {
		return nil
	}
	out := make([]ProjectReward, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Kind = strings.TrimSpace(strings.ToLower(item.Kind))
		item.Label = strings.TrimSpace(item.Label)
		item.Detail = strings.TrimSpace(item.Detail)
		if item.Label == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(item.Kind, "reward") + "|" + item.Label)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}

func normalizeProjectLinks(items []ProjectLink) []ProjectLink {
	if len(items) == 0 {
		return nil
	}
	out := make([]ProjectLink, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Kind = strings.TrimSpace(strings.ToLower(item.Kind))
		item.RefID = strings.TrimSpace(item.RefID)
		item.Label = strings.TrimSpace(item.Label)
		if item.RefID == "" && item.Label == "" {
			continue
		}
		key := strings.ToLower(item.Kind + "|" + firstNonEmpty(item.RefID, item.Label))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}
