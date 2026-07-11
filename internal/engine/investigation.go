package engine

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/storage"
)

type InvestigationLink struct {
	Kind  string `json:"kind,omitempty"`
	RefID string `json:"ref_id,omitempty"`
	Label string `json:"label,omitempty"`
}

type InvestigationClue struct {
	ID             string              `json:"id"`
	Label          string              `json:"label"`
	Detail         string              `json:"detail,omitempty"`
	Source         string              `json:"source,omitempty"`
	DiscoveredTurn int                 `json:"discovered_turn,omitempty"`
	Status         string              `json:"status,omitempty"`
	Links          []InvestigationLink `json:"links,omitempty"`
}

type InvestigationSuspect struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Detail string              `json:"detail,omitempty"`
	Status string              `json:"status,omitempty"`
	Links  []InvestigationLink `json:"links,omitempty"`
}

type InvestigationClaim struct {
	ID         string              `json:"id"`
	Statement  string              `json:"statement"`
	Confidence string              `json:"confidence,omitempty"`
	Status     string              `json:"status,omitempty"`
	Links      []InvestigationLink `json:"links,omitempty"`
}

type InvestigationContradiction struct {
	ID     string              `json:"id"`
	Label  string              `json:"label"`
	Detail string              `json:"detail,omitempty"`
	Status string              `json:"status,omitempty"`
	Links  []InvestigationLink `json:"links,omitempty"`
}

type InvestigationLead struct {
	ID     string              `json:"id"`
	Title  string              `json:"title"`
	Detail string              `json:"detail,omitempty"`
	Status string              `json:"status,omitempty"`
	Links  []InvestigationLink `json:"links,omitempty"`
}

type InvestigationTheory struct {
	ID         string              `json:"id"`
	Statement  string              `json:"statement"`
	Confidence string              `json:"confidence,omitempty"`
	Status     string              `json:"status,omitempty"`
	Links      []InvestigationLink `json:"links,omitempty"`
}

type InvestigationHiddenTruth struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
	Status string `json:"status,omitempty"`
}

type InvestigationCase struct {
	ID             string                       `json:"id"`
	Title          string                       `json:"title"`
	Summary        string                       `json:"summary,omitempty"`
	Status         string                       `json:"status,omitempty"`
	UpdatedTurn    int                          `json:"updated_turn,omitempty"`
	Links          []InvestigationLink          `json:"links,omitempty"`
	Clues          []InvestigationClue          `json:"clues,omitempty"`
	Suspects       []InvestigationSuspect       `json:"suspects,omitempty"`
	Claims         []InvestigationClaim         `json:"claims,omitempty"`
	Contradictions []InvestigationContradiction `json:"contradictions,omitempty"`
	Leads          []InvestigationLead          `json:"leads,omitempty"`
	Theories       []InvestigationTheory        `json:"theories,omitempty"`
	HiddenTruths   []InvestigationHiddenTruth   `json:"hidden_truths,omitempty"`
}

type InvestigationBoard struct {
	Cases []InvestigationCase `json:"cases,omitempty"`
}

// LoadInvestigationBoard returns the normalized canonical investigation board for a world state.
func LoadInvestigationBoard(world *storage.WorldState) InvestigationBoard {
	return loadInvestigationBoard(world)
}

func loadInvestigationBoard(world *storage.WorldState) InvestigationBoard {
	if world == nil || strings.TrimSpace(world.InvestigationBoardJSON) == "" || strings.TrimSpace(world.InvestigationBoardJSON) == "{}" {
		return InvestigationBoard{}
	}
	var board InvestigationBoard
	if err := json.Unmarshal([]byte(world.InvestigationBoardJSON), &board); err != nil {
		return InvestigationBoard{}
	}
	return normalizeInvestigationBoard(board)
}

func storeInvestigationBoard(world *storage.WorldState, board InvestigationBoard) {
	if world == nil {
		return
	}
	normalized := normalizeInvestigationBoard(board)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return
	}
	world.InvestigationBoardJSON = string(payload)
}

func normalizeInvestigationBoard(board InvestigationBoard) InvestigationBoard {
	if len(board.Cases) == 0 {
		return InvestigationBoard{}
	}

	out := InvestigationBoard{Cases: make([]InvestigationCase, 0, len(board.Cases))}
	seen := map[string]bool{}
	for _, c := range board.Cases {
		c.Title = strings.TrimSpace(c.Title)
		if c.Title == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(c.ID), c.Title))
		if c.ID == "" {
			c.ID = "case:" + uuid.NewString()
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		c.Summary = strings.TrimSpace(c.Summary)
		if c.Status == "" {
			c.Status = "open"
		}
		c.Links = normalizeInvestigationLinks(c.Links)
		c.Clues = normalizeInvestigationClues(c.Clues)
		c.Suspects = normalizeInvestigationSuspects(c.Suspects)
		c.Claims = normalizeInvestigationClaims(c.Claims)
		c.Contradictions = normalizeInvestigationContradictions(c.Contradictions)
		c.Leads = normalizeInvestigationLeads(c.Leads)
		c.Theories = normalizeInvestigationTheories(c.Theories)
		c.HiddenTruths = normalizeInvestigationHiddenTruths(c.HiddenTruths)
		out.Cases = append(out.Cases, c)
	}
	return out
}

func normalizeInvestigationLinks(links []InvestigationLink) []InvestigationLink {
	if len(links) == 0 {
		return nil
	}
	out := make([]InvestigationLink, 0, len(links))
	seen := map[string]bool{}
	for _, link := range links {
		link.Kind = strings.TrimSpace(strings.ToLower(link.Kind))
		link.RefID = strings.TrimSpace(link.RefID)
		link.Label = strings.TrimSpace(link.Label)
		if link.RefID == "" && link.Label == "" {
			continue
		}
		key := strings.ToLower(link.Kind + "|" + firstNonEmpty(link.RefID, link.Label))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, link)
	}
	return out
}

func normalizeInvestigationClues(items []InvestigationClue) []InvestigationClue {
	if len(items) == 0 {
		return nil
	}
	out := make([]InvestigationClue, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Label = strings.TrimSpace(item.Label)
		if item.Label == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(item.ID), item.Label))
		if item.ID == "" {
			item.ID = "clue:" + uuid.NewString()
		}
		if item.Status == "" {
			item.Status = "known"
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		item.Detail = strings.TrimSpace(item.Detail)
		item.Source = strings.TrimSpace(item.Source)
		item.Links = normalizeInvestigationLinks(item.Links)
		out = append(out, item)
	}
	return out
}

func normalizeInvestigationSuspects(items []InvestigationSuspect) []InvestigationSuspect {
	if len(items) == 0 {
		return nil
	}
	out := make([]InvestigationSuspect, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(item.ID), item.Name))
		if item.ID == "" {
			item.ID = "suspect:" + uuid.NewString()
		}
		if item.Status == "" {
			item.Status = "person_of_interest"
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		item.Detail = strings.TrimSpace(item.Detail)
		item.Links = normalizeInvestigationLinks(item.Links)
		out = append(out, item)
	}
	return out
}

func normalizeInvestigationClaims(items []InvestigationClaim) []InvestigationClaim {
	if len(items) == 0 {
		return nil
	}
	out := make([]InvestigationClaim, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Statement = strings.TrimSpace(item.Statement)
		if item.Statement == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(item.ID), item.Statement))
		if item.ID == "" {
			item.ID = "claim:" + uuid.NewString()
		}
		if item.Status == "" {
			item.Status = "open"
		}
		item.Confidence = firstNonEmpty(strings.TrimSpace(item.Confidence), "uncertain")
		if seen[key] {
			continue
		}
		seen[key] = true
		item.Links = normalizeInvestigationLinks(item.Links)
		out = append(out, item)
	}
	return out
}

func normalizeInvestigationContradictions(items []InvestigationContradiction) []InvestigationContradiction {
	if len(items) == 0 {
		return nil
	}
	out := make([]InvestigationContradiction, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Label = strings.TrimSpace(item.Label)
		if item.Label == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(item.ID), item.Label))
		if item.ID == "" {
			item.ID = "contradiction:" + uuid.NewString()
		}
		if item.Status == "" {
			item.Status = "open"
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		item.Detail = strings.TrimSpace(item.Detail)
		item.Links = normalizeInvestigationLinks(item.Links)
		out = append(out, item)
	}
	return out
}

func normalizeInvestigationLeads(items []InvestigationLead) []InvestigationLead {
	if len(items) == 0 {
		return nil
	}
	out := make([]InvestigationLead, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Title = strings.TrimSpace(item.Title)
		if item.Title == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(item.ID), item.Title))
		if item.ID == "" {
			item.ID = "lead:" + uuid.NewString()
		}
		if item.Status == "" {
			item.Status = "open"
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		item.Detail = strings.TrimSpace(item.Detail)
		item.Links = normalizeInvestigationLinks(item.Links)
		out = append(out, item)
	}
	return out
}

func normalizeInvestigationTheories(items []InvestigationTheory) []InvestigationTheory {
	if len(items) == 0 {
		return nil
	}
	out := make([]InvestigationTheory, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Statement = strings.TrimSpace(item.Statement)
		if item.Statement == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(item.ID), item.Statement))
		if item.ID == "" {
			item.ID = "theory:" + uuid.NewString()
		}
		if item.Status == "" {
			item.Status = "forming"
		}
		item.Confidence = firstNonEmpty(strings.TrimSpace(item.Confidence), "fragile")
		if seen[key] {
			continue
		}
		seen[key] = true
		item.Links = normalizeInvestigationLinks(item.Links)
		out = append(out, item)
	}
	return out
}

func normalizeInvestigationHiddenTruths(items []InvestigationHiddenTruth) []InvestigationHiddenTruth {
	if len(items) == 0 {
		return nil
	}
	out := make([]InvestigationHiddenTruth, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Label = strings.TrimSpace(item.Label)
		if item.Label == "" {
			continue
		}
		key := strings.ToLower(firstNonEmpty(strings.TrimSpace(item.ID), item.Label))
		if item.ID == "" {
			item.ID = "truth:" + uuid.NewString()
		}
		if item.Status == "" {
			item.Status = "hidden"
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		item.Detail = strings.TrimSpace(item.Detail)
		out = append(out, item)
	}
	return out
}
