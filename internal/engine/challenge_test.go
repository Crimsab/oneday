package engine

import (
	"math/rand"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestStatCheckPass(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		StatsJSON: `{"attributes":{"str":10,"dex":5}}`,
	}
	spec := &ChallengeSpec{
		Type:       ChallengeStatCheck,
		Stat:       "str",
		Difficulty: 8,
	}
	result, err := ce.Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("STR 10 vs difficulty 8 should pass, got: %s", result.Detail)
	}
}

func TestStatCheckFail(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		StatsJSON: `{"attributes":{"str":5}}`,
	}
	spec := &ChallengeSpec{
		Type:       ChallengeStatCheck,
		Stat:       "str",
		Difficulty: 8,
	}
	result, err := ce.Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Errorf("STR 5 vs difficulty 8 should fail, got: %s", result.Detail)
	}
}

func TestStatCheckBoundary(t *testing.T) {
	ce := NewChallengeEngine()
	// Exactly at boundary: value == difficulty → pass
	char := &storage.Character{
		StatsJSON: `{"attributes":{"per":8}}`,
	}
	spec := &ChallengeSpec{
		Type:       ChallengeStatCheck,
		Stat:       "per",
		Difficulty: 8,
	}
	result, err := ce.Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("PER 8 vs difficulty 8 should pass at boundary, got: %s", result.Detail)
	}
}

func TestDiceRollDeterministic(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{StatsJSON: "{}"}

	// Seed rand to get deterministic results.
	rand.Seed(42)
	firstRoll := RollD100()

	rand.Seed(42)
	spec := &ChallengeSpec{
		Type:       ChallengeDiceRoll,
		Difficulty: 50,
		Modifiers:  []Modifier{{Source: "Lucky Charm", Value: 10}},
	}
	result, err := ce.Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With seed 42, we know the first roll value; just verify Total = Roll + 10.
	_ = firstRoll
	if result.Total != result.Roll+10 {
		t.Errorf("Total should be Roll+10 modifier, got Roll=%d Total=%d", result.Roll, result.Total)
	}
}

func TestSeededRNGReplayProducesSameRollLog(t *testing.T) {
	char := &storage.Character{StatsJSON: "{}"}
	spec := &ChallengeSpec{
		Type:       ChallengeDiceRoll,
		Difficulty: 50,
		Modifiers:  []Modifier{{Source: "Lucky Charm", Value: 10}},
	}

	first, err := NewChallengeEngineWithRNG(NewRNGService(1234)).Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	second, err := NewChallengeEngineWithRNG(NewRNGService(1234)).Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}

	if first.Roll != second.Roll || first.Total != second.Total {
		t.Fatalf("seeded rolls differ: first=%+v second=%+v", first, second)
	}
	if len(first.RollLog) != 1 || len(second.RollLog) != 1 {
		t.Fatalf("roll logs len = %d/%d, want 1/1", len(first.RollLog), len(second.RollLog))
	}
	if first.RollLog[0].Raw != second.RollLog[0].Raw || first.RollLog[0].Seed != 1234 {
		t.Fatalf("roll log not reproducible: first=%+v second=%+v", first.RollLog[0], second.RollLog[0])
	}
}

func TestChallengeRollUsesInjectedRNGNotGlobalRand(t *testing.T) {
	char := &storage.Character{StatsJSON: "{}"}
	spec := &ChallengeSpec{Type: ChallengeDiceRoll, Difficulty: 50}

	rand.Seed(1)
	first, err := NewChallengeEngineWithRNG(NewRNGService(99)).Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}

	rand.Seed(999)
	second, err := NewChallengeEngineWithRNG(NewRNGService(99)).Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}

	if first.Roll != second.Roll {
		t.Fatalf("global rand affected injected RNG: first=%d second=%d", first.Roll, second.Roll)
	}
}

func TestDiceRollCriticalSuccess(t *testing.T) {
	// A roll of 96-100 should always be critical success (pass regardless of difficulty).
	ce := NewChallengeEngine()
	char := &storage.Character{StatsJSON: "{}"}

	// Try many times to hit a critical success range.
	gotCritical := false
	for i := 0; i < 200; i++ {
		result, err := ce.Resolve(&ChallengeSpec{
			Type:       ChallengeDiceRoll,
			Difficulty: 200, // impossibly high
		}, char, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Roll >= 96 && !result.Passed {
			t.Errorf("roll %d should be critical success and pass", result.Roll)
		}
		if result.Roll >= 96 {
			gotCritical = true
			break
		}
	}
	if !gotCritical {
		t.Skip("did not hit critical success range in 200 tries (unlikely but possible)")
	}
}

func TestDiceRollCriticalFailure(t *testing.T) {
	// A roll of 1-5 should always be critical failure (fail regardless of modifiers).
	ce := NewChallengeEngine()
	char := &storage.Character{StatsJSON: "{}"}

	// Try many times to hit critical failure range.
	gotCritical := false
	for i := 0; i < 200; i++ {
		result, err := ce.Resolve(&ChallengeSpec{
			Type:       ChallengeDiceRoll,
			Difficulty: 1, // trivially easy
			Modifiers:  []Modifier{{Source: "Bonus", Value: 100}},
		}, char, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Roll <= 5 && result.Passed {
			t.Errorf("roll %d should be critical failure and not pass", result.Roll)
		}
		if result.Roll <= 5 {
			gotCritical = true
			break
		}
	}
	if !gotCritical {
		t.Skip("did not hit critical failure range in 200 tries")
	}
}

func TestItemCheckPresent(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		InventoryJSON: `{"backpack":[{"name":"Iron Key","type":"tool"}]}`,
	}
	spec := &ChallengeSpec{
		Type: ChallengeItemCheck,
		Item: "iron key",
	}
	result, err := ce.Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("iron key present should pass, got: %s", result.Detail)
	}
}

func TestItemCheckAbsent(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		InventoryJSON: `{"backpack":[{"name":"Torch","type":"tool"}]}`,
	}
	spec := &ChallengeSpec{
		Type: ChallengeItemCheck,
		Item: "magic sword",
	}
	result, err := ce.Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Errorf("magic sword absent should fail, got: %s", result.Detail)
	}
}

func TestHasItemDoesNotMatchDescriptionSubstring(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		InventoryJSON: `{"backpack":[{"name":"Monkey Wrench","description":"The handle has a small key charm."}]}`,
	}
	result, err := ce.Resolve(&ChallengeSpec{Type: ChallengeItemCheck, Item: "key"}, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatalf("description substring should not satisfy item check: %s", result.Detail)
	}
}

func TestHasItemMatchesCanonicalIDSlugOrName(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		InventoryJSON: `[
			{"id":"iron-key-01","slug":"iron_key","name":"Iron Key"},
			{"id":"torch-01","slug":"old_torch","name":"Old Torch"}
		]`,
	}
	for _, item := range []string{"iron-key-01", "iron key", "Iron Key"} {
		result, err := ce.Resolve(&ChallengeSpec{Type: ChallengeItemCheck, Item: item}, char, nil, "")
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", item, err)
		}
		if !result.Passed {
			t.Fatalf("%q should match canonical inventory item, got: %s", item, result.Detail)
		}
	}
}

func TestSkillCheckPresent(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		StatsJSON: `{"skills":{"Lockpicking":{"level":3,"xp":150}}}`,
	}
	spec := &ChallengeSpec{
		Type:  ChallengeSkillCheck,
		Skill: "lockpicking",
	}
	result, err := ce.Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("lockpicking skill present should pass, got: %s", result.Detail)
	}
}

func TestSkillCheckWithLevel(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		StatsJSON: `{"skills":{"Lockpicking":{"level":2,"xp":150}}}`,
	}

	// Level 2 vs required level 3 → fail
	specFail := &ChallengeSpec{
		Type:       ChallengeSkillCheck,
		Skill:      "lockpicking",
		SkillLevel: 3,
	}
	result, err := ce.Resolve(specFail, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Errorf("lockpicking level 2 vs required 3 should fail, got: %s", result.Detail)
	}

	// Level 2 vs required level 2 → pass
	specPass := &ChallengeSpec{
		Type:       ChallengeSkillCheck,
		Skill:      "lockpicking",
		SkillLevel: 2,
	}
	result, err = ce.Resolve(specPass, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("lockpicking level 2 vs required 2 should pass, got: %s", result.Detail)
	}
}

func TestSkillCheckAbsent(t *testing.T) {
	ce := NewChallengeEngine()
	char := &storage.Character{
		StatsJSON: `{"skills":{}}`,
	}
	spec := &ChallengeSpec{
		Type:  ChallengeSkillCheck,
		Skill: "alchemy",
	}
	result, err := ce.Resolve(spec, char, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Errorf("alchemy absent should fail, got: %s", result.Detail)
	}
}

func TestMiniGameReturnsError(t *testing.T) {
	// mini_game challenges require TUI interaction — Resolve should return an error.
	ce := NewChallengeEngine()
	char := &storage.Character{StatsJSON: "{}"}
	spec := &ChallengeSpec{
		Type:     ChallengeMiniGame,
		MiniGame: "rps",
	}
	_, err := ce.Resolve(spec, char, nil, "")
	if err == nil {
		t.Error("mini_game type should return error (requires TUI interaction)")
	}
}
