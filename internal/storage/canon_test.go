package storage

import (
	"errors"
	"testing"
	"time"
)

func TestNPCCompatibilityCreatesCanonicalEntityIdentityAndBaseForm(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	now := time.Now()
	npc := &NPC{ID: "npc-mask", StoryID: story.ID, Name: "The Mask", Appearance: "silver mask", PersonalityJSON: "{}", RelationshipJSON: "{}", NemesisJSON: "{}", DiscoveryJSON: "{}", PrivateThoughts: "[]", NotesOnProtagonist: "[]", Desires: "[]", IsAlive: true, CreatedAt: now, UpdatedAt: now}
	if err := db.CreateNPC(npc); err != nil {
		t.Fatal(err)
	}
	loaded, err := db.GetNPC(npc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CanonicalEntityID != npc.ID {
		t.Fatalf("canonical id=%q", loaded.CanonicalEntityID)
	}
	for table, want := range map[string]int{"canonical_entities": 1, "identity_claims": 1, "entity_aliases": 1, "entity_forms": 1} {
		var count int
		if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE story_id=?`, story.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s=%d want %d", table, count, want)
		}
	}
}

func TestFormsControllersAndObserverClaimsPreserveIdentityHistory(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	for _, e := range []*CanonicalEntity{{ID: "true-self", StoryID: story.ID, Kind: "character", CanonicalName: "Hidden Queen", ProfileJSON: "{}"}, {ID: "possessor", StoryID: story.ID, Kind: "character", CanonicalName: "Wraith", ProfileJSON: "{}"}, {ID: "player", StoryID: story.ID, Kind: "protagonist", CanonicalName: "Player", ProfileJSON: "{}"}} {
		if err := db.CreateCanonicalEntity(e); err != nil {
			t.Fatal(err)
		}
	}
	form := &EntityForm{ID: "borrowed-body", StoryID: story.ID, EntityID: "true-self", Name: "Borrowed body", Kind: "transformation", BodyEntityID: "possessor", AppearanceJSON: `{"face":"unknown"}`, ValidFromTurn: 3}
	if err := db.AddEntityForm(form); err != nil {
		t.Fatal(err)
	}
	if err := db.AddEntityControllerEvent(&EntityControllerEvent{StoryID: story.ID, FormID: form.ID, ControllerEntityID: "possessor", ControlKind: "possession", Status: "started", Turn: 4}); err != nil {
		t.Fatal(err)
	}
	public := &IdentityClaim{StoryID: story.ID, SubjectEntityID: "true-self", Label: "The Courier", Kind: "alias", Status: "observed", Confidence: .8, Visibility: "player", LearnedTurn: 2}
	if err := db.AddIdentityClaim(public); err != nil {
		t.Fatal(err)
	}
	hidden := &IdentityClaim{StoryID: story.ID, SubjectEntityID: "true-self", ClaimedEntityID: "true-self", ObserverEntityID: "possessor", Label: "Hidden Queen", Kind: "true_identity", Status: "confirmed", Confidence: 1, Visibility: "private", LearnedTurn: 1}
	if err := db.AddIdentityClaim(hidden); err != nil {
		t.Fatal(err)
	}
	view, err := db.GetPlayerSafeEntity(story.ID, "true-self", "player", 10)
	if err != nil {
		t.Fatal(err)
	}
	if view.DisplayName != "The Courier" || len(view.IdentityClaims) != 1 {
		t.Fatalf("unsafe projection: %#v", view)
	}
	if _, err := db.Conn().Exec(`UPDATE identity_claims SET label='leaked' WHERE id=?`, hidden.ID); err == nil {
		t.Fatal("append-only identity accepted update")
	}
}

func TestFactsRetractionsLocksAndPlayerSafeProjection(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	entity := &CanonicalEntity{ID: "subject", StoryID: story.ID, Kind: "character", CanonicalName: "Secret Name", ProfileJSON: `{"eye_color":"amber","scar":"left"}`}
	if err := db.CreateCanonicalEntity(entity); err != nil {
		t.Fatal(err)
	}
	if err := db.AddIdentityClaim(&IdentityClaim{StoryID: story.ID, SubjectEntityID: entity.ID, Label: "Known Stranger", Kind: "display", Status: "confirmed", Confidence: 1, Visibility: "player"}); err != nil {
		t.Fatal(err)
	}
	known := &CharacterFact{StoryID: story.ID, SubjectEntityID: entity.ID, Predicate: "occupation", ObjectJSON: `"courier"`, LearnedTurn: 2, Confidence: .9, Visibility: "player"}
	if err := db.AddCharacterFact(known); err != nil {
		t.Fatal(err)
	}
	hidden := &CharacterFact{StoryID: story.ID, SubjectEntityID: entity.ID, Predicate: "allegiance", ObjectJSON: `"hidden cult"`, LearnedTurn: 1, Confidence: 1, Visibility: "private"}
	if err := db.AddCharacterFact(hidden); err != nil {
		t.Fatal(err)
	}
	head, err := db.GetActiveTimeline(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := db.GetStoryRevision(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	alternate, err := db.ForkStoryBranch(story.ID, head.Commit.ID, "alternate-facts", revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO character_facts
		(id,story_id,subject_entity_id,predicate,object_json,source_event_id,learned_turn,confidence,visibility,retracts_fact_id,evidence_json,branch_id,source_commit_id)
		VALUES ('alternate-retraction',?,?, 'retraction','true','',3,1,'player',?,'[]',?,?)`,
		story.ID, entity.ID, known.ID, alternate.ID, head.Commit.ID); err != nil {
		t.Fatal(err)
	}
	view, err := db.GetPlayerSafeEntity(story.ID, entity.ID, "player", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Facts) != 1 || view.Facts[0].Predicate != "occupation" {
		t.Fatalf("hidden fact leaked: %#v", view.Facts)
	}
	if err := db.AddCharacterFact(&CharacterFact{StoryID: story.ID, SubjectEntityID: entity.ID, Predicate: "retraction", ObjectJSON: `true`, RetractsFactID: known.ID, LearnedTurn: 4, Confidence: 1, Visibility: "player"}); err != nil {
		t.Fatal(err)
	}
	view, err = db.GetPlayerSafeEntity(story.ID, entity.ID, "player", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Facts) != 0 {
		t.Fatalf("retracted fact still visible: %#v", view.Facts)
	}
	if _, err := db.Conn().Exec(`UPDATE character_facts SET predicate='tampered' WHERE id=?`, known.ID); err == nil {
		t.Fatal("append-only fact accepted update")
	}
	if err := db.LockCanonicalField(entity.ID, "eye_color", "profile", "amber"); err != nil {
		t.Fatal(err)
	}
	if err := db.MergeCanonicalProfile(entity.ID, map[string]any{"scar": "right"}); err != nil {
		t.Fatal(err)
	}
	if err := db.MergeCanonicalProfile(entity.ID, map[string]any{"eye_color": "blue"}); !errors.Is(err, ErrCanonicalFieldLocked) {
		t.Fatalf("locked merge err=%v", err)
	}
	if err := db.LockCanonicalField(entity.ID, "portrait_prompt", "visual", "ink portrait"); err != nil {
		t.Fatal(err)
	}
	if err := db.MergeCanonicalVisual(entity.ID, map[string]any{"portrait_prompt": "photo"}); !errors.Is(err, ErrCanonicalFieldLocked) {
		t.Fatalf("visual lock err=%v", err)
	}
}

func TestReputationLedgerIsProvenancedAndClamped(t *testing.T) {
	db, story := newTimelineStory(t)
	defer db.Close()
	entity := &CanonicalEntity{ID: "hero", StoryID: story.ID, Kind: "protagonist", CanonicalName: "Hero", ProfileJSON: "{}"}
	if err := db.CreateCanonicalEntity(entity); err != nil {
		t.Fatal(err)
	}
	faction := &Faction{StoryID: story.ID, Name: "Glass Guild", Visibility: "player"}
	if err := db.CreateFaction(faction); err != nil {
		t.Fatal(err)
	}
	if err := db.AddFactionMembershipEvent(&FactionMembershipEvent{StoryID: story.ID, FactionID: faction.ID, EntityID: entity.ID, Status: "joined", Role: "ally", Visibility: "player", Turn: 1}); err != nil {
		t.Fatal(err)
	}
	for _, e := range []*ReputationEvent{{StoryID: story.ID, FactionID: faction.ID, EntityID: entity.ID, Delta: 80, Reason: "saved caravan", SourceEventID: "event-1", Turn: 2}, {StoryID: story.ID, FactionID: faction.ID, EntityID: entity.ID, Delta: 70, Reason: "returned relic", SourceEventID: "event-2", Turn: 3}} {
		if err := db.AddReputationEvent(e); err != nil {
			t.Fatal(err)
		}
		if e.BranchID == "" || e.SourceCommitID == "" {
			t.Fatalf("missing lineage: %#v", e)
		}
	}
	score, err := db.ReputationScore(story.ID, faction.ID, entity.ID)
	if err != nil {
		t.Fatal(err)
	}
	if score != 100 {
		t.Fatalf("score=%d", score)
	}
	projections, err := db.ListPlayerSafeFactions(story.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(projections) != 1 || len(projections[0].KnownMembers) != 1 || projections[0].Reputation[entity.ID] != 100 {
		t.Fatalf("faction projection=%#v", projections)
	}
	if _, err := db.Conn().Exec(`UPDATE reputation_events SET delta=0 WHERE faction_id=?`, faction.ID); err == nil {
		t.Fatal("append-only reputation accepted update")
	}
}
