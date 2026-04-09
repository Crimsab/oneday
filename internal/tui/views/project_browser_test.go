package views

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/engine"
)

func TestProjectBrowserGroupsProjectsByStatus(t *testing.T) {
	t.Parallel()

	model := NewProjectBrowserModel("Projects", engine.ProjectBoard{
		Projects: []engine.ProjectClock{
			{ID: "project-complete", Title: "Restore the Lantern Loft", Status: "completed", Progress: 4, Segments: 4, Outcome: "You have a quiet place to disappear for a night."},
			{ID: "project-active", Title: "Train with Lyanna", Status: "active", Progress: 2, Segments: 4, Summary: "Your footwork is starting to look deliberate."},
			{ID: "project-paused", Title: "Decode the harbor ledger", Status: "paused", Progress: 1, Segments: 3, Stakes: "Learn who paid the courier."},
		},
	}, 120, 40)

	if len(model.rows) != 6 {
		t.Fatalf("len(rows) = %d, want 6", len(model.rows))
	}
	if model.rows[0].kind != projectBrowserRowSection || model.rows[0].label != "Active" {
		t.Fatalf("rows[0] = %+v, want Active section", model.rows[0])
	}
	if model.rows[1].kind != projectBrowserRowProject || model.board.Projects[model.rows[1].projectIndex].Title != "Train with Lyanna" {
		t.Fatalf("rows[1] = %+v, want active project row", model.rows[1])
	}
	if model.rows[2].kind != projectBrowserRowSection || model.rows[2].label != "Paused" {
		t.Fatalf("rows[2] = %+v, want Paused section", model.rows[2])
	}
	if model.rows[4].kind != projectBrowserRowSection || model.rows[4].label != "Completed" {
		t.Fatalf("rows[4] = %+v, want Completed section", model.rows[4])
	}
	if !strings.Contains(model.View(), "Train with Lyanna") {
		t.Fatalf("view missing active project:\n%s", model.View())
	}
	if !strings.Contains(model.View(), "Restore the Lantern Loft") {
		t.Fatalf("view missing completed project:\n%s", model.View())
	}
}

func TestProjectBrowserEnterShowsProjectDetail(t *testing.T) {
	t.Parallel()

	model := NewProjectBrowserModel("Projects", engine.ProjectBoard{
		Projects: []engine.ProjectClock{
			{
				ID:       "project-active",
				Title:    "Train with Lyanna",
				Kind:     "training",
				Status:   "active",
				Progress: 2,
				Segments: 4,
				Summary:  "Your footwork is finally looking deliberate.",
				Stakes:   "Be ready before the Bell Choir moves.",
				Rewards: []engine.ProjectReward{
					{Kind: "skill", Label: "Blade Forms"},
				},
				Links: []engine.ProjectLink{
					{Kind: "npc", Label: "Lyanna"},
					{Kind: "front", Label: "Whispers Around the Bell Tower"},
				},
			},
		},
	}, 120, 40)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !updated.detail.Visible() {
		t.Fatal("detail overlay not visible after opening selected project")
	}
	if !strings.Contains(updated.detail.Content, "Progress: 2/4") {
		t.Fatalf("detail missing progress:\n%s", updated.detail.Content)
	}
	if !strings.Contains(updated.detail.Content, "Skill: Blade Forms") {
		t.Fatalf("detail missing reward:\n%s", updated.detail.Content)
	}
	if !strings.Contains(updated.detail.Content, "Front: Whispers Around the Bell Tower") {
		t.Fatalf("detail missing linked front:\n%s", updated.detail.Content)
	}
}
