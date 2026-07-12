package storage

import (
	"database/sql"
	"testing"
	"time"
)

func TestCanonicalWorldProjectionTimeWeatherAndTravel(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	now := time.Now()
	world := &WorldState{ID: "world-canon", StoryID: story.ID, CurrentLocation: "Harbor", KnownLocationsJSON: `["Harbor","Old Road"]`, GlobalEventsJSON: "[]", FactionStandingsJSON: "{}", StoryHooksJSON: "[]", WorldReactionsJSON: "[]", InvestigationBoardJSON: "{}", ProjectClocksJSON: "{}", PlayerGuidanceJSON: "[]", FrontsJSON: "[]", CharacterTimelineJSON: "{}", SceneContractJSON: "{}", CurrentChapter: 1, CurrentTurn: 7, UpdatedAt: now}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatal(err)
	}
	if world.CurrentLocationID == "" {
		t.Fatal("compatibility world did not receive location id")
	}
	p, err := db.PlayerWorld(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Locations) != 2 || p.Clock.Day != 0 || p.Weather.Tracked || p.Weather.Label != "Not tracked" {
		t.Fatalf("initial projection=%#v", p)
	}
	var roadID string
	for _, l := range p.Locations {
		if l.Name == "Old Road" {
			roadID = l.ID
		}
	}
	if err := db.ConnectLocations(&LocationEdge{StoryID: story.ID, FromLocationID: world.CurrentLocationID, ToLocationID: roadID, Direction: "north", TravelMinutes: 90, ConditionsJSON: `{"requires":"road open"}`, Visibility: "player"}); err != nil {
		t.Fatal(err)
	}
	clock, err := db.AdvanceWorldTime(story.ID, "downtime", 1500, 7)
	if err != nil {
		t.Fatal(err)
	}
	if clock.Day != 1 || clock.MinuteOfDay != 60 {
		t.Fatalf("clock=%#v", clock)
	}
	if err := db.RecordWeather(story.ID, world.CurrentLocationID, "rain", "heavy", "Cold rain", 1, 0); err != nil {
		t.Fatal(err)
	}
	p, err = db.PlayerWorld(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Weather.Tracked || p.Weather.Label != "rain" {
		t.Fatalf("weather projection=%#v", p.Weather)
	}
	if err := db.SetCurrentLocation(story.ID, "hero", "Old Road", 8); err != nil {
		t.Fatal(err)
	}
	p, err = db.PlayerWorld(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.CurrentLocationID != roadID || len(p.Edges) != 1 {
		t.Fatalf("travel projection=%#v", p)
	}
	if _, err := db.Conn().Exec(`UPDATE world_time_events SET reason='tampered' WHERE story_id=?`, story.ID); err == nil {
		t.Fatal("append-only world time event accepted update")
	}
}

func TestLocationTransitionBuildsHierarchyRoutesAndClock(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	world := &WorldState{ID: "world-spatial", StoryID: story.ID, CurrentLocation: "Dock 7", KnownLocationsJSON: `["Dock 7"]`, CurrentChapter: 1, CurrentTurn: 3, UpdatedAt: time.Now()}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatal(err)
	}
	transition := LocationTransition{
		From:       &SpatialLocationRef{Name: "Dock 7", Kind: "site", RegionPath: []string{"Vharrow", "Port District"}},
		To:         SpatialLocationRef{Name: "Access Lane", Kind: "subzone", RegionPath: []string{"Vharrow", "Port District"}, ParentLocation: "Dock 7", Description: "A narrow service lane."},
		Discovered: []SpatialLocationRef{{Name: "Old Pump House", Kind: "landmark", RegionPath: []string{"Vharrow", "Port District"}}},
		Routes:     []SpatialRouteRef{{From: "Dock 7", To: "Access Lane", Direction: "inside", TravelMinutes: 7, TravelMode: "walk", Bidirectional: true, Conditions: map[string]any{"requires": "gate open"}}},
	}
	var destination *CanonicalLocation
	if err := db.WithTx(func(tx *sql.Tx) error {
		var err error
		destination, err = db.ApplyLocationTransitionTx(tx, story.ID, "hero", transition, 3)
		if err != nil {
			return err
		}
		world.CurrentLocation = destination.Name
		world.CurrentLocationID = destination.ID
		return db.UpdateWorldStateTx(tx, world)
	}); err != nil {
		t.Fatal(err)
	}
	projection, err := db.PlayerWorld(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	if projection.CurrentLocation != "Access Lane" || projection.CurrentLocationID != destination.ID {
		t.Fatalf("current location = %q/%q", projection.CurrentLocation, projection.CurrentLocationID)
	}
	hasNestedRegion := false
	for _, region := range projection.Regions {
		hasNestedRegion = hasNestedRegion || region.ParentRegionID != ""
	}
	if len(projection.Regions) != 2 || !hasNestedRegion {
		t.Fatalf("regions = %#v", projection.Regions)
	}
	if len(projection.Locations) != 3 {
		t.Fatalf("locations = %#v", projection.Locations)
	}
	var dock, lane *CanonicalLocation
	for i := range projection.Locations {
		switch projection.Locations[i].Name {
		case "Dock 7":
			dock = &projection.Locations[i]
		case "Access Lane":
			lane = &projection.Locations[i]
		}
	}
	if dock == nil || lane == nil || lane.ParentLocationID != dock.ID || lane.RegionID == "" || lane.Kind != "subzone" {
		t.Fatalf("hierarchy dock=%#v lane=%#v", dock, lane)
	}
	if len(projection.Edges) != 1 || !projection.Edges[0].Bidirectional || projection.Edges[0].TravelMode != "walk" {
		t.Fatalf("edges = %#v", projection.Edges)
	}
	if projection.Clock.MinuteOfDay != 7 {
		t.Fatalf("clock = %#v", projection.Clock)
	}
}

func TestWeatherRemainsExplicitlyUntrackedWhenAbsent(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	if err := db.CreateWorldState(&WorldState{ID: "world-none", StoryID: story.ID, KnownLocationsJSON: "[]", GlobalEventsJSON: "[]", FactionStandingsJSON: "{}", CurrentChapter: 1, UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	p, err := db.PlayerWorld(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Weather.Label != "Not tracked" || p.Weather.Tracked {
		t.Fatalf("weather=%#v", p.Weather)
	}
}

func TestLegacyKnownLocationObjectsSeedCanonicalRegionAndDescription(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	world := &WorldState{
		ID: "world-legacy-spatial", StoryID: story.ID, CurrentLocation: "Old Gate",
		KnownLocationsJSON: `[{"name":"Old Gate","description":"A gate in the rain.","region":"North Ward","discovered_turn":2}]`,
		CurrentTurn:        7, UpdatedAt: time.Now(),
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatal(err)
	}
	projection, err := db.PlayerWorld(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Regions) != 1 || projection.Regions[0].Name != "North Ward" {
		t.Fatalf("regions = %#v", projection.Regions)
	}
	if len(projection.Locations) != 1 || projection.Locations[0].Description != "A gate in the rain." || projection.Locations[0].RegionID != projection.Regions[0].ID {
		t.Fatalf("locations = %#v", projection.Locations)
	}
	if projection.Locations[0].DiscoveredTurn != 2 {
		t.Fatalf("discovered turn = %d, want legacy turn 2", projection.Locations[0].DiscoveredTurn)
	}
}
