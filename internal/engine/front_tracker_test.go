package engine

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestLoadFrontTrackerBoardSanitizesVisibleFrontsAndHotspots(t *testing.T) {
	t.Parallel()

	world := &storage.WorldState{
		StoryHooksJSON: `[
			{"id":"hook-cooling","kind":"warning","title":"Lay low until the harbor cools","detail":"The Old Guard is checking cargo manifests.","status":"cooling","updated_turn":8},
			{"id":"hook-active","kind":"promise","title":"Meet Lyanna before dawn","detail":"She asked for a private answer.","status":"active","timer_turns":2,"updated_turn":10}
		]`,
		WorldReactionsJSON: `[
			{"id":"reaction-low","kind":"rumor","title":"Harbor whispers spread","detail":"Dockhands keep repeating your name.","status":"active","created_turn":9,"source_turn":8},
			{"id":"reaction-high","kind":"front_pressure","title":"Bell Quarter grows watchful around Whispers Around the Bell Tower","detail":"Whispers Around the Bell Tower - Bell Quarter [suspicion 55 high]","status":"active","created_turn":10,"source_turn":10},
			{"id":"reaction-old","kind":"rumor","title":"Old rumor","detail":"Already spent.","status":"resolved","created_turn":2,"source_turn":1}
		]`,
		FrontsJSON: `[
			{
				"id":"front-rumor",
				"faction":"Ash Court",
				"title":"The Ash Court is buying judges in secret",
				"public_title":"Whispers in the Court Annex",
				"stakes":"They will own the city courts by moonrise.",
				"public_stakes":"Something is warping the court's decisions.",
				"visibility":"rumored",
				"status":"active",
				"segments":4,
				"progress":1,
				"pressures":[{"region":"Court Annex","kind":"influence","level":30,"detail":"Petitions keep disappearing.","updated_turn":9}]
			},
			{
				"id":"front-known",
				"faction":"Bell Choir",
				"title":"The Silent Bell Choir is seeding sleepers across the district",
				"public_title":"Whispers Around the Bell Tower",
				"stakes":"Sleeper-priests will take the guard towers.",
				"public_stakes":"Something ugly is taking hold around the tower.",
				"visibility":"known",
				"status":"active",
				"segments":6,
				"progress":3,
				"last_advanced_turn":10,
				"next_escalation_turn":12,
				"pressures":[{"region":"Bell Quarter","kind":"suspicion","level":55,"detail":"Street sermons are drawing nervous crowds.","updated_turn":10}]
			},
			{
				"id":"front-resolved",
				"faction":"Harbor Syndicate",
				"title":"The harbor crackdown is broken",
				"public_title":"Harbor Crackdown Broken",
				"stakes":"The syndicate loses control of the harbor gates.",
				"visibility":"known",
				"status":"resolved",
				"segments":4,
				"progress":4,
				"resolution":"The harbor masters turn on the syndicate and reopen the gates."
			},
			{
				"id":"front-hidden",
				"faction":"Ash Court",
				"title":"The Ash Court owns the judges already",
				"stakes":"They will close every case against them.",
				"visibility":"hidden",
				"status":"active",
				"segments":4,
				"progress":2
			}
		]`,
	}

	board := LoadFrontTrackerBoard(world)
	if len(board.Hooks) != 2 {
		t.Fatalf("len(board.Hooks) = %d, want 2", len(board.Hooks))
	}
	if board.Hooks[0].Title != "Meet Lyanna before dawn" {
		t.Fatalf("hooks[0].Title = %q, want active timed hook first", board.Hooks[0].Title)
	}

	if len(board.Fronts) != 3 {
		t.Fatalf("len(board.Fronts) = %d, want 3 visible fronts", len(board.Fronts))
	}
	if board.Fronts[0].Title != "Whispers Around the Bell Tower" {
		t.Fatalf("fronts[0].Title = %q, want known active front first", board.Fronts[0].Title)
	}
	if board.Fronts[0].Faction != "Bell Choir" {
		t.Fatalf("known front faction = %q, want Bell Choir", board.Fronts[0].Faction)
	}
	if board.Fronts[1].Title != "Whispers in the Court Annex" {
		t.Fatalf("fronts[1].Title = %q, want rumored front second", board.Fronts[1].Title)
	}
	if board.Fronts[1].Faction != "" {
		t.Fatalf("rumored front leaked faction: %+v", board.Fronts[1])
	}
	if board.Fronts[1].Stakes != "Something is warping the court's decisions." {
		t.Fatalf("rumored front stakes = %q, want public stakes", board.Fronts[1].Stakes)
	}
	if board.Fronts[2].Resolution != "The harbor masters turn on the syndicate and reopen the gates." {
		t.Fatalf("resolved known front resolution = %q, want preserved resolution", board.Fronts[2].Resolution)
	}

	if len(board.Hotspots) != 2 {
		t.Fatalf("len(board.Hotspots) = %d, want 2 visible hotspots", len(board.Hotspots))
	}
	if board.Hotspots[0].FrontTitle != "Whispers Around the Bell Tower" || board.Hotspots[0].Level != 55 {
		t.Fatalf("hotspots[0] = %+v, want Bell Choir pressure first", board.Hotspots[0])
	}

	if len(board.Reactions) != 2 {
		t.Fatalf("len(board.Reactions) = %d, want only active visible reactions", len(board.Reactions))
	}
	if board.Reactions[0].Title != "Bell Quarter grows watchful around Whispers Around the Bell Tower" {
		t.Fatalf("reactions[0].Title = %q, want newest fallout first", board.Reactions[0].Title)
	}

	serialized := strings.Join([]string{
		board.Fronts[0].Title,
		board.Fronts[1].Title,
		board.Fronts[1].Stakes,
		board.Reactions[0].Detail,
	}, "\n")
	if strings.Contains(serialized, "owns the judges already") || strings.Contains(serialized, "They will close every case against them.") {
		t.Fatalf("tracker board leaked hidden front content:\n%s", serialized)
	}
}

func TestLoadFrontTrackerBoardHandlesNilWorld(t *testing.T) {
	t.Parallel()

	board := LoadFrontTrackerBoard(nil)
	if len(board.Hooks) != 0 || len(board.Fronts) != 0 || len(board.Hotspots) != 0 || len(board.Reactions) != 0 {
		t.Fatalf("board = %+v, want empty board", board)
	}
}
