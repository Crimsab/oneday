package engine

import (
	"encoding/json"
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
		ID:          "test-char-id",
		StoryID:     "test-story-id",
		Name:        "Test Hero",
		Background:  "A test background",
		StatsJSON:   baseStatsJSON(),
		TraitsJSON:  "[]",
		SkillsJSON:  "{}",
		InventoryJSON: "[]",
		CreatedAt:   now,
		UpdatedAt:   now,
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

// TestApplyStateChanges_SkillXP_AutoLearn verifies XP grant on unknown skill creates it at level 1.
func TestApplyStateChanges_SkillXP_AutoLearn(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	// Grant XP to a skill that was never explicitly learned.
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
	skill, ok := skills["Stealth"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Stealth to be auto-created, got nil")
	}
	if testToInt(skill["level"]) != 1 {
		t.Errorf("expected level=1 for auto-created skill, got %v", skill["level"])
	}
	if testToInt(skill["xp"]) != 25 {
		t.Errorf("expected xp=25, got %v", skill["xp"])
	}
}

// TODO: NPC-related tests (new_npc, disposition) require a live DB.
// Integration tests to be added in a future phase once a test DB helper is available.
