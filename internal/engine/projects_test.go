package engine

import (
	"strings"
	"testing"
)

func TestStoreAndLoadProjectBoardPreservesProgressAndRewards(t *testing.T) {
	world := newTestWorld()

	board := ProjectBoard{
		Projects: []ProjectClock{
			{
				Title:       "Train with Lyanna",
				Kind:        "training",
				Summary:     "Work your footwork against a sharper partner.",
				Progress:    2,
				Segments:    4,
				StartedTurn: 5,
				UpdatedTurn: 7,
				Owner:       "Mara",
				Location:    "Bell Quarter",
				Stakes:      "If you stop now, the lesson goes cold.",
				Rewards: []ProjectReward{
					{Kind: "skill", Label: "Footwork +1"},
					{Kind: "skill", Label: "Footwork +1"},
				},
				Links: []ProjectLink{
					{Kind: "npc", Label: "Lyanna"},
				},
			},
		},
	}

	storeProjectBoard(world, board)
	if !strings.Contains(world.ProjectClocksJSON, "Train with Lyanna") {
		t.Fatalf("stored project board = %s, want serialized project payload", world.ProjectClocksJSON)
	}

	loaded := loadProjectBoard(world)
	if len(loaded.Projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(loaded.Projects))
	}
	project := loaded.Projects[0]
	if project.Status != "active" {
		t.Fatalf("status = %q, want active", project.Status)
	}
	if project.Progress != 2 || project.Segments != 4 {
		t.Fatalf("progress = %d/%d, want 2/4", project.Progress, project.Segments)
	}
	if len(project.Rewards) != 1 {
		t.Fatalf("rewards = %+v, want duplicate reward collapsed", project.Rewards)
	}
	if len(project.Links) != 1 || project.Links[0].Label != "Lyanna" {
		t.Fatalf("links = %+v, want preserved npc link", project.Links)
	}
}

func TestLoadProjectBoardHandlesInvalidPayloadGracefully(t *testing.T) {
	world := newTestWorld()
	world.ProjectClocksJSON = `{"projects":[{"title":""}],"broken":`

	loaded := loadProjectBoard(world)
	if len(loaded.Projects) != 0 {
		t.Fatalf("loaded projects = %+v, want empty board on invalid payload", loaded.Projects)
	}
}
