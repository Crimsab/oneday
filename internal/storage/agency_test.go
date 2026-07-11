package storage

import (
	"database/sql"
	"testing"
	"time"
)

func TestOffscreenAgencyIsBoundedCanonicalAndCooldownAware(t *testing.T) {
	db, story := newAudioStory(t)
	defer db.Close()
	for index, item := range []struct {
		name        string
		disposition int
	}{{"Ally", 60}, {"Rival", -60}, {"Wanderer", 0}} {
		npc := &NPC{ID: "agency-" + item.name, StoryID: story.ID, Name: item.name, Role: "agent", Disposition: item.disposition, IsAlive: true, FirstAppearedTurn: 0, LastSeenTurn: 0, PersonalityJSON: `{}`, CreatedAt: time.Now().UTC().Add(time.Duration(index) * time.Second)}
		if err := db.CreateNPC(npc); err != nil {
			t.Fatal(err)
		}
	}
	head, _ := db.GetActiveTimeline(story.ID)
	var planned []CanonicalEventInput
	if err := db.WithTx(func(tx *sql.Tx) error {
		var err error
		planned, err = db.PlanOffscreenAgencyEventsTx(tx, story.ID, head.Branch.ID, 5, 2)
		if err != nil {
			return err
		}
		_, err = db.AppendTurnCommitTx(tx, AppendTurnCommitParams{StoryID: story.ID, BranchID: head.Branch.ID, ExpectedHeadID: head.Commit.ID, CanonicalTurn: 5, PayloadJSON: `{"turn":5}`, Events: planned})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(planned) != 2 {
		t.Fatalf("bounded events=%d", len(planned))
	}
	events, err := db.ListRecentAgencyEvents(story.ID, 10)
	if err != nil || len(events) != 2 || events[0].CommitID == "" || events[0].BranchID != head.Branch.ID {
		t.Fatalf("events=%+v err=%v", events, err)
	}
	if events[0].Action != "prepares_help" && events[0].Action != "advances_pressure" && events[0].Action != "pursues_goal" {
		t.Fatalf("action=%q", events[0].Action)
	}
	newHead, _ := db.GetActiveTimeline(story.ID)
	if err := db.WithTx(func(tx *sql.Tx) error {
		again, err := db.PlanOffscreenAgencyEventsTx(tx, story.ID, newHead.Branch.ID, 6, 2)
		if err == nil && len(again) != 1 {
			t.Fatalf("cooldown should leave only third NPC, got %d", len(again))
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
