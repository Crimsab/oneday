package storage

import (
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
