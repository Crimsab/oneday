package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildStoryCodexShowsVisibleFrontsWithoutLeakingHiddenState(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()

	story := &storage.Story{
		ID:          "story-codex-fronts",
		Name:        "Codex Front Story",
		SettingJSON: `{"factions":["Bell Choir","Ash Court"]}`,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	world := &storage.WorldState{
		ID:              "world-codex-fronts",
		StoryID:         story.ID,
		CurrentLocation: "Bell Quarter",
		KnownLocationsJSON: `[
			{"name":"Bell Quarter","description":"A district of bells and narrow alleys.","discovered_turn":1},
			{"name":"Harbor Ward","region":"Bell Quarter","description":"Warehouses and watchfires.","discovered_turn":2}
		]`,
		CurrentChapter: 1,
		CurrentTurn:    5,
		UpdatedAt:      now,
	}
	storeFronts(world, []Front{
		{
			ID:           "front-rumor",
			Faction:      "Bell Choir",
			Title:        "The Silent Bell Choir is seeding sleepers across the district",
			PublicTitle:  "Whispers Around the Bell Tower",
			Stakes:       "Sleeper-priests will take the guard towers.",
			PublicStakes: "Something ugly is taking hold around the tower.",
			Visibility:   "rumored",
			Segments:     6,
			Progress:     3,
			Pressures: []FrontPressure{
				{Region: "Bell Quarter", Kind: "suspicion", Level: 35, UpdatedTurn: 5},
			},
		},
		{
			ID:         "front-hidden",
			Faction:    "Ash Court",
			Title:      "The Ash Court is buying judges in secret",
			Stakes:     "They will own the courts by moonrise.",
			Visibility: "hidden",
			Segments:   4,
			Progress:   1,
		},
	})
	syncKnownFrontContinuity(world, world.CurrentTurn)
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	index, err := BuildStoryCodex(db, story, char, world)
	if err != nil {
		t.Fatalf("BuildStoryCodex: %v", err)
	}

	frontIDs := index.CategoryEntries["fronts"]
	if len(frontIDs) != 1 {
		t.Fatalf("front category entries = %v, want only the visible front", frontIDs)
	}

	entry, ok := index.Entry(codexFrontEntryID("front-rumor"))
	if !ok {
		t.Fatalf("missing visible front entry in codex")
	}
	if entry.Title != "Whispers Around the Bell Tower" {
		t.Fatalf("front entry title = %q, want public title", entry.Title)
	}
	if strings.Contains(entry.Subtitle, "Bell Choir") {
		t.Fatalf("rumored front leaked faction in subtitle: %+v", entry)
	}

	joinedSections := strings.Join(flattenCodexSections(entry.Sections), "\n")
	if strings.Contains(joinedSections, "Bell Choir") {
		t.Fatalf("rumored front leaked faction in sections:\n%s", joinedSections)
	}
	if strings.Contains(joinedSections, "Sleeper-priests will take the guard towers.") {
		t.Fatalf("rumored front leaked hidden stakes:\n%s", joinedSections)
	}
	if !strings.Contains(joinedSections, "Something ugly is taking hold around the tower.") {
		t.Fatalf("rumored front should retain public stakes:\n%s", joinedSections)
	}

	locationEntry, ok := index.Entry(codexLocationEntryID("Bell Quarter"))
	if !ok {
		t.Fatalf("missing location entry for Bell Quarter")
	}
	locationSections := strings.Join(flattenCodexSections(locationEntry.Sections), "\n")
	if !strings.Contains(locationSections, "Whispers Around the Bell Tower - Bell Quarter [suspicion 35 rising]") {
		t.Fatalf("location entry missing front pressure trace:\n%s", locationSections)
	}

	protagonist, ok := index.Entry(ProtagonistCodexEntryID())
	if !ok {
		t.Fatalf("missing protagonist codex entry")
	}
	if !strings.Contains(strings.Join(flattenCodexSections(protagonist.Sections), "\n"), "Whispers Around the Bell Tower 3/6") {
		t.Fatalf("protagonist entry missing known front summary")
	}
}

func flattenCodexSections(sections []CodexSection) []string {
	lines := make([]string, 0, len(sections)*2)
	for _, section := range sections {
		lines = append(lines, section.Title)
		lines = append(lines, section.Lines...)
	}
	return lines
}
