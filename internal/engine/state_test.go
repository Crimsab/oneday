package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

// baseStatsJSON returns a minimal valid stats JSON for a fresh character.
func baseStatsJSON() string {
	stats := map[string]interface{}{
		"vitals": map[string]interface{}{
			"hp": map[string]interface{}{"current": 20, "max": 20},
		},
		"attributes": map[string]interface{}{
			"str": 3, "dex": 3,
		},
		"secondary": map[string]interface{}{
			"reputation": 0,
		},
		"currency": 0,
		"traits":   []interface{}{},
		"titles":   []interface{}{},
		"skills":   map[string]interface{}{},
		"deaths":   0,
	}
	b, _ := json.Marshal(stats)
	return string(b)
}

// newTestChar returns a fresh character for testing.
func newTestChar() *storage.Character {
	now := time.Now()
	return &storage.Character{
		ID:            "test-char-id",
		StoryID:       "test-story-id",
		Name:          "Test Hero",
		Background:    "A test background",
		StatsJSON:     baseStatsJSON(),
		TraitsJSON:    "[]",
		SkillsJSON:    "{}",
		InventoryJSON: "[]",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// newTestWorld returns a minimal world state.
func newTestWorld() *storage.WorldState {
	now := time.Now()
	return &storage.WorldState{
		ID:              "test-world-id",
		CurrentTurn:     1,
		CurrentChapter:  1,
		CurrentLocation: "Starting Village",
		UpdatedAt:       now,
	}
}

// testToInt converts interface{} to int using the same logic as toFloat from state.go.
func testToInt(v interface{}) int {
	return int(toFloat(v))
}

// parseStats unmarshals StatsJSON into a map for assertions.
func parseStats(t *testing.T, char *storage.Character) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(char.StatsJSON), &m); err != nil {
		t.Fatalf("failed to parse StatsJSON: %v", err)
	}
	return m
}

// TestApplyStateChanges_TraitAdd verifies that a new trait is added.
func TestApplyStateChanges_TraitAdd(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	changes := map[string]interface{}{
		"trait_add": "Brave",
	}
	applied, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 1)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied change, got %d", len(applied))
	}

	stats := parseStats(t, char)
	traits := toStringSlice(stats["traits"])
	if len(traits) != 1 || traits[0] != "Brave" {
		t.Errorf("expected traits=[Brave], got %v", traits)
	}
	// TraitsJSON should also be updated.
	var charTraits []string
	if err := json.Unmarshal([]byte(char.TraitsJSON), &charTraits); err != nil {
		t.Fatalf("failed to parse TraitsJSON: %v", err)
	}
	if len(charTraits) != 1 || charTraits[0] != "Brave" {
		t.Errorf("TraitsJSON expected [Brave], got %v", charTraits)
	}
}

// TestApplyStateChanges_DuplicateTrait verifies no duplicate trait is added.
func TestApplyStateChanges_DuplicateTrait(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	// Add the same trait twice.
	changes := map[string]interface{}{"trait_add": "Clever"}
	_, _ = ApplyStateChanges(changes, char, world, nil, "test-story-id", 1)
	applied, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 2)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}
	// Second application should produce 0 changes (duplicate skipped).
	if len(applied) != 0 {
		t.Errorf("expected 0 applied changes for duplicate trait, got %d", len(applied))
	}

	stats := parseStats(t, char)
	traits := toStringSlice(stats["traits"])
	if len(traits) != 1 {
		t.Errorf("expected exactly 1 trait after duplicate add, got %v", traits)
	}
}

// TestApplyStateChanges_DuplicateTrait_CaseInsensitive verifies case-insensitive dedup.
func TestApplyStateChanges_DuplicateTrait_CaseInsensitive(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	_, _ = ApplyStateChanges(map[string]interface{}{"trait_add": "Bold"}, char, world, nil, "test-story-id", 1)
	applied, _ := ApplyStateChanges(map[string]interface{}{"trait_add": "bold"}, char, world, nil, "test-story-id", 2)

	if len(applied) != 0 {
		t.Errorf("expected 0 applied changes for case-insensitive duplicate, got %d", len(applied))
	}
	stats := parseStats(t, char)
	if len(toStringSlice(stats["traits"])) != 1 {
		t.Errorf("expected exactly 1 trait, got %v", toStringSlice(stats["traits"]))
	}
}

func TestNPCMutationRecorderStagesUntilCommit(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:              "story-npc-stage",
		Name:            "NPC Stage",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-stage-lyanna",
		StoryID:           story.ID,
		Name:              "Lyanna",
		Role:              "scout",
		PersonalityJSON:   `{}`,
		RelationshipJSON:  `{}`,
		Disposition:       0,
		IsAlive:           true,
		FirstAppearedTurn: 1,
		LastSeenTurn:      1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID
	recorder := newNPCMutationRecorder(directNPCStore{db: db})

	applied, err := applyStateChangesWithNPCStore(map[string]interface{}{
		"npc_relationship": map[string]interface{}{
			"name":  "Lyanna",
			"trust": map[string]interface{}{"change": 8},
		},
	}, char, world, db, recorder, story.ID, 2)
	if err != nil {
		t.Fatalf("applyStateChangesWithNPCStore: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected staged relationship change")
	}

	before, err := db.GetNPCByName(story.ID, "Lyanna")
	if err != nil || before == nil {
		t.Fatalf("GetNPCByName before: %v, npc=%+v", err, before)
	}
	if axes := loadRelationshipAxes(before); axes.Trust != 0 {
		t.Fatalf("trust before commit = %d, want 0", axes.Trust)
	}

	if err := db.WithTx(func(tx *sql.Tx) error {
		return recorder.Commit(txNPCStore{db: db, tx: tx})
	}); err != nil {
		t.Fatalf("Commit recorder: %v", err)
	}
	after, err := db.GetNPCByName(story.ID, "Lyanna")
	if err != nil || after == nil {
		t.Fatalf("GetNPCByName after: %v, npc=%+v", err, after)
	}
	if axes := loadRelationshipAxes(after); axes.Trust != 8 {
		t.Fatalf("trust after commit = %d, want 8", axes.Trust)
	}
}

func TestNewNPCPromotesPlaceholderDiscovery(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:              "story-npc-promote",
		Name:            "NPC Promote",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID

	_, err := ApplyStateChanges(map[string]interface{}{
		"npc_disposition": map[string]interface{}{
			"name":   "Marek",
			"change": 5,
		},
	}, char, world, db, story.ID, 2)
	if err != nil {
		t.Fatalf("ApplyStateChanges placeholder: %v", err)
	}
	placeholder, err := db.GetNPCByName(story.ID, "Marek")
	if err != nil || placeholder == nil {
		t.Fatalf("placeholder lookup: %v npc=%+v", err, placeholder)
	}
	if discovery := npcDiscoveryFromStorage(placeholder); discovery.VisualReadiness != NPCVisualNone {
		t.Fatalf("placeholder visual readiness = %q, want none", discovery.VisualReadiness)
	}

	_, err = ApplyStateChanges(map[string]interface{}{
		"new_npc": map[string]interface{}{
			"name":       "Marek",
			"role":       "dock intermediary",
			"appearance": "Gaunt man with ink-stained gloves and a rain-dark coat.",
			"personality": map[string]interface{}{
				"traits":       []interface{}{"nervous", "calculating"},
				"speech_style": "low, clipped, evasive",
			},
			"desires": []interface{}{
				map[string]interface{}{"desire": "Leave Dock 7 before the purge.", "priority": "high", "known_to_player": false},
			},
			"can_help": true,
		},
	}, char, world, db, story.ID, 3)
	if err != nil {
		t.Fatalf("ApplyStateChanges new_npc: %v", err)
	}
	npc, err := db.GetNPCByName(story.ID, "Marek")
	if err != nil || npc == nil {
		t.Fatalf("promoted lookup: %v npc=%+v", err, npc)
	}
	if npc.Role != "dock intermediary" {
		t.Fatalf("role = %q, want dock intermediary", npc.Role)
	}
	if !strings.Contains(npc.Appearance, "ink-stained gloves") {
		t.Fatalf("appearance not promoted: %q", npc.Appearance)
	}
	discovery := npcDiscoveryFromStorage(npc)
	if discovery.Stage != NPCStageEstablished {
		t.Fatalf("stage = %q, want established", discovery.Stage)
	}
	if discovery.VisualReadiness != NPCVisualCanonical {
		t.Fatalf("visual readiness = %q, want canonical", discovery.VisualReadiness)
	}
}

func TestNPCReferenceCreatesRumorWithoutVisualReadiness(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:              "story-npc-rumor",
		Name:            "NPC Rumor",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID

	applied, err := ApplyStateChanges(map[string]interface{}{
		"npc_reference": map[string]interface{}{
			"name":         "Serra Vale",
			"stage":        "rumor",
			"source":       "ledger",
			"confidence":   "rumored",
			"detail":       "Named in a wet ledger but not yet seen.",
			"public_label": "Serra Vale",
		},
	}, char, world, db, story.ID, 4)
	if err != nil {
		t.Fatalf("ApplyStateChanges npc_reference: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected npc_reference applied change")
	}
	npc, err := db.GetNPCByName(story.ID, "Serra Vale")
	if err != nil || npc == nil {
		t.Fatalf("rumor lookup: %v npc=%+v", err, npc)
	}
	discovery := npcDiscoveryFromStorage(npc)
	if discovery.Stage != NPCStageRumor {
		t.Fatalf("stage = %q, want rumor", discovery.Stage)
	}
	if discovery.VisualReadiness != NPCVisualNone {
		t.Fatalf("visual readiness = %q, want none", discovery.VisualReadiness)
	}
	if !strings.Contains(npc.NotesOnProtagonist, "wet ledger") {
		t.Fatalf("notes = %q, want detail recorded", npc.NotesOnProtagonist)
	}
}

func TestApplyStateChangesCreatesPlaceholderNPCForUnknownUpdates(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:              "story-unknown-npc",
		Name:            "Unknown NPC Story",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID

	applied, err := ApplyStateChanges(map[string]interface{}{
		"npc_disposition": map[string]interface{}{
			"name":   "Uomo col giornale",
			"change": -3,
		},
		"npc_notes": map[string]interface{}{
			"name": "Uomo col giornale",
			"note": "Ha notato domande brevi e mirate.",
		},
	}, char, world, db, story.ID, 5)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied NPC updates")
	}

	npc, err := db.GetNPCByName(story.ID, "Uomo col giornale")
	if err != nil || npc == nil {
		t.Fatalf("GetNPCByName: %v, npc=%+v", err, npc)
	}
	if npc.Role != "person of interest" {
		t.Fatalf("Role = %q, want placeholder role", npc.Role)
	}
	if npc.Disposition != -3 {
		t.Fatalf("Disposition = %d, want -3", npc.Disposition)
	}
	if npc.FirstAppearedTurn != 5 || npc.LastSeenTurn != 5 {
		t.Fatalf("turn tracking = %d/%d, want 5/5", npc.FirstAppearedTurn, npc.LastSeenTurn)
	}
	if !strings.Contains(npc.NotesOnProtagonist, "domande brevi") {
		t.Fatalf("NotesOnProtagonist = %q, want note preserved", npc.NotesOnProtagonist)
	}
}

func TestApplyStateChangesCreatesNPCFromInvestigationSuspect(t *testing.T) {
	db, _ := newSaveTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:              "story-investigation-npc",
		Name:            "Investigation NPC Story",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := newTestChar()
	char.StoryID = story.ID
	world := newTestWorld()
	world.StoryID = story.ID

	applied, err := ApplyStateChanges(map[string]interface{}{
		"investigation_update": map[string]interface{}{
			"case_title": "Dock 7",
			"summary":    "Marek is now a concrete person of interest.",
			"suspects": []interface{}{
				map[string]interface{}{
					"action": "add",
					"name":   "Marek",
					"detail": "Intermediario spaventato; sa più di quanto dica.",
				},
			},
		},
	}, char, world, db, story.ID, 8)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected applied changes")
	}

	npc, err := db.GetNPCByName(story.ID, "Marek")
	if err != nil || npc == nil {
		t.Fatalf("GetNPCByName: %v, npc=%+v", err, npc)
	}
	if npc.Role != "person of interest" {
		t.Fatalf("Role = %q, want person of interest", npc.Role)
	}
	if !strings.Contains(npc.NotesOnProtagonist, "Intermediario spaventato") {
		t.Fatalf("NotesOnProtagonist = %q, want suspect detail", npc.NotesOnProtagonist)
	}
	if !strings.Contains(world.InvestigationBoardJSON, "Marek") {
		t.Fatalf("InvestigationBoardJSON = %q, want Marek suspect", world.InvestigationBoardJSON)
	}
}

// TestApplyStateChanges_TitleAdd verifies that a new title is added.
func TestApplyStateChanges_TitleAdd(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	changes := map[string]interface{}{
		"title_add": "Dragon Slayer",
	}
	applied, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 1)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied change, got %d", len(applied))
	}

	stats := parseStats(t, char)
	titles := toStringSlice(stats["titles"])
	if len(titles) != 1 || titles[0] != "Dragon Slayer" {
		t.Errorf("expected titles=[Dragon Slayer], got %v", titles)
	}
}

// TestApplyStateChanges_TitleAdd_NoDuplicate verifies no duplicate title is added.
func TestApplyStateChanges_TitleAdd_NoDuplicate(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	changes := map[string]interface{}{"title_add": "The Betrayed"}
	_, _ = ApplyStateChanges(changes, char, world, nil, "test-story-id", 1)
	applied, _ := ApplyStateChanges(changes, char, world, nil, "test-story-id", 2)

	if len(applied) != 0 {
		t.Errorf("expected 0 changes for duplicate title, got %d", len(applied))
	}
	stats := parseStats(t, char)
	if len(toStringSlice(stats["titles"])) != 1 {
		t.Errorf("expected 1 title after duplicate add")
	}
}

// TestApplyStateChanges_SkillLearn verifies a new skill is learned.
func TestApplyStateChanges_SkillLearn(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	changes := map[string]interface{}{
		"skill_learn": "Lockpicking",
	}
	applied, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 1)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied change, got %d", len(applied))
	}

	stats := parseStats(t, char)
	skills := toSkillsMap(stats["skills"])
	skill, ok := skills["Lockpicking"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Lockpicking skill map, got %T", skills["Lockpicking"])
	}
	if testToInt(skill["level"]) != 1 {
		t.Errorf("expected level=1, got %v", skill["level"])
	}
	if testToInt(skill["xp"]) != 0 {
		t.Errorf("expected xp=0, got %v", skill["xp"])
	}

	// SkillsJSON should be updated.
	var skillsMap map[string]interface{}
	if err := json.Unmarshal([]byte(char.SkillsJSON), &skillsMap); err != nil {
		t.Fatalf("failed to parse SkillsJSON: %v", err)
	}
	if _, exists := skillsMap["Lockpicking"]; !exists {
		t.Errorf("expected Lockpicking in SkillsJSON")
	}
}

// TestApplyStateChanges_SkillXP verifies XP is added to an existing skill.
func TestApplyStateChanges_SkillXP(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	// First learn the skill.
	_, _ = ApplyStateChanges(map[string]interface{}{"skill_learn": "Alchemy"}, char, world, nil, "test-story-id", 1)

	// Then gain XP.
	changes := map[string]interface{}{
		"skill_xp": map[string]interface{}{
			"skill": "Alchemy",
			"xp":    30,
		},
	}
	applied, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 2)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 applied change, got %d", len(applied))
	}

	stats := parseStats(t, char)
	skills := toSkillsMap(stats["skills"])
	skill := skills["Alchemy"].(map[string]interface{})
	if testToInt(skill["xp"]) != 30 {
		t.Errorf("expected xp=30, got %v", skill["xp"])
	}
	if testToInt(skill["level"]) != 1 {
		t.Errorf("expected level still 1, got %v", skill["level"])
	}
}

// TestApplyStateChanges_SkillLevelUp verifies leveling up at the XP threshold (level*100).
func TestApplyStateChanges_SkillLevelUp(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	// Learn skill at level 1.
	_, _ = ApplyStateChanges(map[string]interface{}{"skill_learn": "Swordplay"}, char, world, nil, "test-story-id", 1)

	// Gain exactly 100 XP — should trigger level up (threshold = 1*100 = 100).
	changes := map[string]interface{}{
		"skill_xp": map[string]interface{}{
			"skill": "Swordplay",
			"xp":    100,
		},
	}
	_, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 2)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}

	stats := parseStats(t, char)
	skills := toSkillsMap(stats["skills"])
	skill := skills["Swordplay"].(map[string]interface{})

	if testToInt(skill["level"]) != 2 {
		t.Errorf("expected level=2 after leveling up, got %v", skill["level"])
	}
	// XP after level-up: 100 - 100 = 0 remaining.
	if testToInt(skill["xp"]) != 0 {
		t.Errorf("expected xp=0 after exact level-up, got %v", skill["xp"])
	}
}

// TestApplyStateChanges_SkillLevelUp_MultiLevel verifies multiple level-ups in one XP gain.
func TestApplyStateChanges_SkillLevelUp_MultiLevel(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	_, _ = ApplyStateChanges(map[string]interface{}{"skill_learn": "Magic"}, char, world, nil, "test-story-id", 1)

	// Gain 350 XP: level 1→2 at 100, level 2→3 at 200 (100+200=300 consumed), 50 XP remains.
	changes := map[string]interface{}{
		"skill_xp": map[string]interface{}{
			"skill": "Magic",
			"xp":    350,
		},
	}
	_, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 2)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}

	stats := parseStats(t, char)
	skills := toSkillsMap(stats["skills"])
	skill := skills["Magic"].(map[string]interface{})

	if testToInt(skill["level"]) != 3 {
		t.Errorf("expected level=3 after multi-level-up, got %v", skill["level"])
	}
	if testToInt(skill["xp"]) != 50 {
		t.Errorf("expected xp=50 remaining, got %v", skill["xp"])
	}
}

func TestApplyStateChanges_SkillXP_UnknownSkillGoesToPendingDiscovery(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	changes := map[string]interface{}{
		"skill_xp": map[string]interface{}{
			"skill": "Stealth",
			"xp":    25,
		},
	}
	_, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 1)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}

	stats := parseStats(t, char)
	skills := toSkillsMap(stats["skills"])
	if _, ok := skills["Stealth"]; ok {
		t.Fatalf("Stealth should not be auto-created from first XP grant: %+v", skills)
	}
	pending := toSkillsMap(stats["pending_skill_discovery"])
	discovery, ok := pending["Stealth"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Stealth pending discovery, got %+v", pending)
	}
	if testToInt(discovery["xp"]) != 25 {
		t.Errorf("pending xp = %v, want 25", discovery["xp"])
	}
	if testToInt(discovery["evidence"]) != 1 {
		t.Errorf("pending evidence = %v, want 1", discovery["evidence"])
	}
}

func TestApplyStateChangesStringNumberDoesNotBecomeZero(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	_, err := ApplyStateChanges(map[string]interface{}{"currency": "10"}, char, world, nil, "test-story-id", 1)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}

	stats := parseStats(t, char)
	if got := testToInt(stats["currency"]); got != 10 {
		t.Fatalf("currency = %d, want 10", got)
	}
}

func TestApplyStateChangesInvalidInventoryDoesNotWipeInventory(t *testing.T) {
	char := newTestChar()
	char.InventoryJSON = `{"items": [`
	world := newTestWorld()

	_, err := ApplyStateChanges(map[string]interface{}{"inventory_add": "Rope"}, char, world, nil, "test-story-id", 1)
	if err == nil {
		t.Fatal("ApplyStateChanges error = nil, want invalid inventory error")
	}
	if char.InventoryJSON != `{"items": [` {
		t.Fatalf("InventoryJSON = %q, want original corrupt inventory preserved", char.InventoryJSON)
	}
}

func TestApplyStateChangesReportsUnknownKeysInDiagnostics(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	applied, err := ApplyStateChanges(map[string]interface{}{"skil_xp": map[string]interface{}{"skill": "Stealth", "xp": 25}}, char, world, nil, "test-story-id", 1)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("applied len = %d, want 1 diagnostic", len(applied))
	}
	if applied[0].Field != "unknown_state_change" || applied[0].New != "skil_xp" {
		t.Fatalf("diagnostic = %+v, want unknown_state_change for skil_xp", applied[0])
	}
}

func TestApplyStateChangesDefersNarratorManagedKeysWithoutDiagnostics(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	story := &storage.Story{ID: "test-story-id", SettingJSON: `{}`}
	changes := map[string]interface{}{
		"world_location_add": "Dock 7",
		"world_event_add":    "The fugitive bell rang under the wharf.",
	}

	applied, err := ApplyStateChanges(changes, char, world, nil, story.ID, 4)
	if err != nil {
		t.Fatalf("ApplyStateChanges error: %v", err)
	}
	for _, change := range applied {
		if change.Field == "unknown_state_change" {
			t.Fatalf("deferred narrator key produced unknown diagnostic: %+v", change)
		}
	}

	if err := ApplyNarratorStateChanges(context.Background(), changes, nil, story, world, nil); err != nil {
		t.Fatalf("ApplyNarratorStateChanges error: %v", err)
	}
	if !strings.Contains(world.KnownLocationsJSON, "Dock 7") {
		t.Fatalf("KnownLocationsJSON = %q, want Dock 7", world.KnownLocationsJSON)
	}
	if !strings.Contains(world.GlobalEventsJSON, "fugitive bell") {
		t.Fatalf("GlobalEventsJSON = %q, want fugitive bell event", world.GlobalEventsJSON)
	}
}

func TestApplyStateChangesUsesDeterministicDependencyOrder(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		char := newTestChar()
		world := newTestWorld()
		changes := map[string]interface{}{
			"skill_xp":    map[string]interface{}{"skill": "Alchemy", "xp": 120},
			"skill_learn": "Alchemy",
			"trait_add":   "Methodical",
			"z_unknown":   true,
			"a_unknown":   true,
		}

		applied, err := ApplyStateChanges(changes, char, world, nil, "test-story-id", 1)
		if err != nil {
			t.Fatalf("iteration %d: ApplyStateChanges: %v", iteration, err)
		}
		fields := make([]string, 0, len(applied))
		for _, change := range applied {
			fields = append(fields, change.Field)
		}
		want := []string{"traits", "skills.Alchemy", "skills.Alchemy", "unknown_state_change", "unknown_state_change"}
		if strings.Join(fields, "|") != strings.Join(want, "|") {
			t.Fatalf("iteration %d: fields = %v, want %v", iteration, fields, want)
		}
		if applied[len(applied)-2].New != "a_unknown" || applied[len(applied)-1].New != "z_unknown" {
			t.Fatalf("iteration %d: unknown keys are not sorted: %+v", iteration, applied[len(applied)-2:])
		}
		skill := toSkillsMap(parseStats(t, char)["skills"])["Alchemy"].(map[string]interface{})
		if level, xp := int(toFloat(skill["level"])), int(toFloat(skill["xp"])); level != 2 || xp != 20 {
			t.Fatalf("iteration %d: Alchemy = %+v, want level 2 with 20 XP", iteration, skill)
		}
	}
}

// TODO: NPC-related tests (new_npc, disposition) require a live DB.
// Integration tests to be added in a future phase once a test DB helper is available.
