package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/engine"
)

func TestFrontTrackerModelShowsRumoredKnownResolvedAndFallout(t *testing.T) {
	t.Parallel()

	model := NewFrontTrackerModel("Fronts & Fallout", engine.FrontTrackerBoard{
		Hooks: []engine.StoryHook{
			{ID: "hook-1", Title: "Meet Lyanna before dawn", Detail: "She is waiting at the old lighthouse.", Status: "active", TimerTurns: 2, UpdatedTurn: 10},
		},
		Fronts: []engine.FrontTrackerFront{
			{
				ID:                 "front-known",
				Title:              "Whispers Around the Bell Tower",
				Faction:            "Bell Choir",
				Stakes:             "Sleeper-priests will take the guard towers.",
				Status:             "active",
				Visibility:         "known",
				Progress:           3,
				Segments:           6,
				NextEscalationTurn: 12,
				Pressures: []engine.FrontTrackerPressure{
					{Region: "Bell Quarter", Kind: "suspicion", Level: 55, Severity: "high", Summary: "Bell Quarter [suspicion 55 high] - guards question strangers"},
				},
			},
			{
				ID:         "front-rumor",
				Title:      "Whispers in the Court Annex",
				Stakes:     "Something is warping the court's decisions.",
				Status:     "active",
				Visibility: "rumored",
			},
			{
				ID:         "front-resolved",
				Title:      "Harbor Crackdown Broken",
				Faction:    "Harbor Syndicate",
				Status:     "resolved",
				Visibility: "known",
				Progress:   4,
				Segments:   4,
				Resolution: "The harbor masters reopen the gates.",
			},
		},
		Hotspots: []engine.FrontTrackerPressure{
			{FrontTitle: "Whispers Around the Bell Tower", Region: "Bell Quarter", Kind: "suspicion", Level: 55, Severity: "high", Summary: "Bell Quarter [suspicion 55 high] - guards question strangers", UpdatedTurn: 10},
		},
		Reactions: []engine.WorldReaction{
			{ID: "reaction-1", Kind: "front_pressure", Title: "Bell Quarter grows watchful around Whispers Around the Bell Tower", Detail: "Whispers Around the Bell Tower - Bell Quarter [suspicion 55 high]", Status: "active", CreatedTurn: 10},
		},
	}, 120, 40)

	view := model.View()
	for _, want := range []string{
		"Open Hooks",
		"Active Fronts",
		"Resolved Fronts",
		"Pressure Hotspots",
		"Recent Fallout",
		"Whispers Around the Bell Tower",
		"Whispers in the Court Annex",
		"Harbor Crackdown Broken",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("tracker view missing %q:\n%s", want, view)
		}
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.detail.Visible() {
		t.Fatal("detail overlay not visible for selected front")
	}
	if strings.Contains(updated.detail.Content, "Ash Court") {
		t.Fatalf("rumored front detail leaked hidden faction:\n%s", updated.detail.Content)
	}
	if !strings.Contains(updated.detail.Content, "Something is warping the court's decisions.") {
		t.Fatalf("rumored front detail missing public stakes:\n%s", updated.detail.Content)
	}

	model.selected = 6
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(updated.detail.Content, "The harbor masters reopen the gates.") {
		t.Fatalf("resolved front detail missing outcome:\n%s", updated.detail.Content)
	}
}

func TestFrontTrackerModelEmptyState(t *testing.T) {
	t.Parallel()

	model := NewFrontTrackerModel("Fronts & Fallout", engine.FrontTrackerBoard{}, 100, 30)
	view := model.View()
	if !strings.Contains(view, "No open hooks, visible fronts, or active fallout yet.") {
		t.Fatalf("empty tracker view missing empty state:\n%s", view)
	}
}
