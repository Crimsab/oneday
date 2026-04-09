package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/storage"
)

// PlayerGuidance is a soft, future-facing authorial directive requested by the
// player. It should influence future turns without forcing an immediate scene.
type PlayerGuidance struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Title         string `json:"title"`
	Detail        string `json:"detail,omitempty"`
	Scope         string `json:"scope,omitempty"`
	Priority      string `json:"priority,omitempty"`
	Status        string `json:"status,omitempty"` // active, seeded, fulfilled
	Progress      string `json:"progress,omitempty"`
	RequestedTurn int    `json:"requested_turn,omitempty"`
	UpdatedTurn   int    `json:"updated_turn,omitempty"`
}

func loadPlayerGuidance(world *storage.WorldState) []PlayerGuidance {
	if world == nil || strings.TrimSpace(world.PlayerGuidanceJSON) == "" || strings.TrimSpace(world.PlayerGuidanceJSON) == "[]" {
		return nil
	}
	var guidance []PlayerGuidance
	if err := json.Unmarshal([]byte(world.PlayerGuidanceJSON), &guidance); err != nil {
		return nil
	}
	return normalizePlayerGuidance(guidance)
}

func storePlayerGuidance(world *storage.WorldState, guidance []PlayerGuidance) {
	if world == nil {
		return
	}
	if payload, err := json.Marshal(normalizePlayerGuidance(guidance)); err == nil {
		world.PlayerGuidanceJSON = string(payload)
	}
}

func normalizePlayerGuidance(guidance []PlayerGuidance) []PlayerGuidance {
	if len(guidance) == 0 {
		return nil
	}
	out := make([]PlayerGuidance, 0, len(guidance))
	seen := map[string]bool{}
	for _, item := range guidance {
		item.ID = strings.TrimSpace(item.ID)
		item.Kind = strings.TrimSpace(item.Kind)
		item.Title = strings.TrimSpace(item.Title)
		item.Detail = strings.TrimSpace(item.Detail)
		item.Scope = strings.TrimSpace(item.Scope)
		item.Priority = strings.TrimSpace(item.Priority)
		item.Status = strings.TrimSpace(item.Status)
		item.Progress = strings.TrimSpace(item.Progress)
		if item.ID == "" {
			item.ID = uuid.NewString()
		}
		if item.Status == "" {
			item.Status = "active"
		}
		if item.Priority == "" {
			item.Priority = "medium"
		}
		if item.Scope == "" {
			item.Scope = "chapter"
		}
		if item.Title == "" {
			continue
		}
		key := strings.ToLower(item.Kind + "|" + item.Title + "|" + item.Status)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func activePlayerGuidance(guidance []PlayerGuidance) []PlayerGuidance {
	if len(guidance) == 0 {
		return nil
	}
	active := make([]PlayerGuidance, 0, len(guidance))
	for _, item := range normalizePlayerGuidance(guidance) {
		if strings.EqualFold(item.Status, "fulfilled") {
			continue
		}
		active = append(active, item)
	}
	if len(active) == 0 {
		return nil
	}
	return active
}

func upsertPlayerGuidance(existing, incoming []PlayerGuidance, currentTurn int) []PlayerGuidance {
	if len(incoming) == 0 {
		return normalizePlayerGuidance(existing)
	}

	guidance := normalizePlayerGuidance(existing)
	for _, item := range normalizePlayerGuidance(incoming) {
		idx := findPlayerGuidanceIndex(guidance, item.ID, item.Kind, item.Title)
		if idx >= 0 && strings.EqualFold(guidance[idx].Status, "fulfilled") {
			idx = -1
		}
		if idx >= 0 {
			if item.Kind != "" {
				guidance[idx].Kind = item.Kind
			}
			if item.Detail != "" {
				guidance[idx].Detail = item.Detail
			}
			if item.Scope != "" {
				guidance[idx].Scope = item.Scope
			}
			if item.Priority != "" {
				guidance[idx].Priority = item.Priority
			}
			if item.Status != "" {
				guidance[idx].Status = item.Status
			}
			if item.Progress != "" {
				guidance[idx].Progress = item.Progress
			}
			if guidance[idx].RequestedTurn == 0 {
				guidance[idx].RequestedTurn = currentTurn
			}
			guidance[idx].UpdatedTurn = currentTurn
			continue
		}

		if item.ID == "" {
			item.ID = uuid.NewString()
		}
		if item.RequestedTurn == 0 {
			item.RequestedTurn = currentTurn
		}
		item.UpdatedTurn = currentTurn
		guidance = append(guidance, item)
	}

	return normalizePlayerGuidance(guidance)
}

func updatePlayerGuidance(existing []PlayerGuidance, updates []PlayerGuidance, currentTurn int) []PlayerGuidance {
	if len(existing) == 0 || len(updates) == 0 {
		return normalizePlayerGuidance(existing)
	}

	guidance := normalizePlayerGuidance(existing)
	for _, update := range normalizePlayerGuidance(updates) {
		idx := findPlayerGuidanceIndex(guidance, update.ID, update.Kind, update.Title)
		if idx < 0 {
			continue
		}
		if update.Status != "" {
			guidance[idx].Status = update.Status
		}
		if update.Progress != "" {
			guidance[idx].Progress = update.Progress
		}
		if update.Detail != "" {
			guidance[idx].Detail = update.Detail
		}
		guidance[idx].UpdatedTurn = currentTurn
	}
	return normalizePlayerGuidance(guidance)
}

func findPlayerGuidanceIndex(guidance []PlayerGuidance, id, kind, title string) int {
	id = strings.TrimSpace(id)
	kind = strings.TrimSpace(kind)
	title = strings.TrimSpace(title)
	for i, item := range guidance {
		if id != "" && strings.EqualFold(item.ID, id) {
			return i
		}
		if title != "" && strings.EqualFold(item.Title, title) {
			if kind == "" || item.Kind == "" || strings.EqualFold(item.Kind, kind) {
				return i
			}
		}
	}
	return -1
}

func formatPlayerGuidanceList(guidance []PlayerGuidance) string {
	guidance = normalizePlayerGuidance(guidance)
	if len(guidance) == 0 {
		return ""
	}

	var lines []string
	for _, item := range guidance {
		line := fmt.Sprintf("- %s", item.Title)
		if item.Kind != "" {
			line += " {" + item.Kind + "}"
		}
		if item.Scope != "" || item.Priority != "" || item.Status != "" {
			line += " ["
			parts := make([]string, 0, 3)
			if item.Scope != "" {
				parts = append(parts, item.Scope)
			}
			if item.Priority != "" {
				parts = append(parts, item.Priority)
			}
			if item.Status != "" {
				parts = append(parts, item.Status)
			}
			line += strings.Join(parts, "/") + "]"
		}
		if item.Detail != "" {
			line += " — " + item.Detail
		}
		if item.Progress != "" {
			line += " | progress: " + item.Progress
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
