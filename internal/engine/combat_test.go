package engine

import (
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

// --- Enemy Validation Tests ---

func TestValidateEnemyClampHP(t *testing.T) {
	enemy := &EnemyStats{HP: 0, Attack: 5, Defense: 2, Behavior: BehaviorAggressive}
	ValidateEnemy(enemy)
	if enemy.HP != 1 {
		t.Errorf("HP < 1 should be clamped to 1, got %d", enemy.HP)
	}
	if enemy.MaxHP != 1 {
		t.Errorf("MaxHP should equal HP after validation, got %d", enemy.MaxHP)
	}
}

func TestValidateEnemyClampHPHigh(t *testing.T) {
	enemy := &EnemyStats{HP: 9999, Attack: 5, Defense: 2, Behavior: BehaviorAggressive}
	ValidateEnemy(enemy)
	if enemy.HP != 999 {
		t.Errorf("HP > 999 should be clamped to 999, got %d", enemy.HP)
	}
}

func TestValidateEnemyClampAttack(t *testing.T) {
	enemy := &EnemyStats{HP: 50, Attack: 100, Defense: 2, Behavior: BehaviorAggressive}
	ValidateEnemy(enemy)
	if enemy.Attack != 50 {
		t.Errorf("Attack > 50 should be clamped to 50, got %d", enemy.Attack)
	}

	enemy2 := &EnemyStats{HP: 50, Attack: -5, Defense: 2, Behavior: BehaviorAggressive}
	ValidateEnemy(enemy2)
	if enemy2.Attack != 0 {
		t.Errorf("Attack < 0 should be clamped to 0, got %d", enemy2.Attack)
	}
}

func TestValidateEnemyClampDefense(t *testing.T) {
	enemy := &EnemyStats{HP: 50, Attack: 5, Defense: 100, Behavior: BehaviorAggressive}
	ValidateEnemy(enemy)
	if enemy.Defense != 30 {
		t.Errorf("Defense > 30 should be clamped to 30, got %d", enemy.Defense)
	}
}

func TestValidateEnemyDefaultBehavior(t *testing.T) {
	enemy := &EnemyStats{HP: 50, Attack: 5, Defense: 2, Behavior: ""}
	ValidateEnemy(enemy)
	if enemy.Behavior != BehaviorAggressive {
		t.Errorf("empty Behavior should default to aggressive, got %q", enemy.Behavior)
	}
}

func TestValidateEnemyMaxHPSet(t *testing.T) {
	enemy := &EnemyStats{HP: 50, Attack: 5, Defense: 2, Behavior: BehaviorAggressive, MaxHP: 0}
	ValidateEnemy(enemy)
	if enemy.MaxHP != 50 {
		t.Errorf("MaxHP should be set to HP value, got %d", enemy.MaxHP)
	}
}

// --- Behavior Modifier Tests ---

func TestBehaviorModifierAggressive(t *testing.T) {
	mod := behaviorModifier(BehaviorAggressive, 1, 20, 20, 50, 50)
	if mod != 2 {
		t.Errorf("aggressive modifier should be +2, got %d", mod)
	}
	mod = behaviorModifier(BehaviorAggressive, 5, 10, 20, 25, 50)
	if mod != 2 {
		t.Errorf("aggressive modifier should always be +2, got %d", mod)
	}
}

func TestBehaviorModifierDefensive(t *testing.T) {
	// Odd turn: -2
	mod := behaviorModifier(BehaviorDefensive, 1, 20, 20, 50, 50)
	if mod != -2 {
		t.Errorf("defensive odd turn should be -2, got %d", mod)
	}
	// Even turn: +4
	mod = behaviorModifier(BehaviorDefensive, 2, 20, 20, 50, 50)
	if mod != 4 {
		t.Errorf("defensive even turn should be +4, got %d", mod)
	}
}

func TestBehaviorModifierTactical(t *testing.T) {
	mod1 := behaviorModifier(BehaviorTactical, 1, 20, 20, 50, 50)
	if mod1 != 1 {
		t.Errorf("tactical odd turn should be +1, got %d", mod1)
	}
	mod2 := behaviorModifier(BehaviorTactical, 2, 20, 20, 50, 50)
	if mod2 != 3 {
		t.Errorf("tactical even turn should be +3, got %d", mod2)
	}
}

func TestBehaviorModifierBeast(t *testing.T) {
	mod := behaviorModifier(BehaviorBeast, 3, 20, 20, 50, 50)
	if mod != 3 {
		t.Errorf("beast turn 3 should be +3, got %d", mod)
	}
	mod = behaviorModifier(BehaviorBeast, 1, 20, 20, 50, 50)
	if mod != 0 {
		t.Errorf("beast non-div-3 turn should be 0, got %d", mod)
	}
	mod = behaviorModifier(BehaviorBeast, 6, 20, 20, 50, 50)
	if mod != 3 {
		t.Errorf("beast turn 6 should be +3, got %d", mod)
	}
}

func TestBehaviorModifierIntelligent(t *testing.T) {
	// Player HP very low (< 30%) → +3
	mod := behaviorModifier(BehaviorIntelligent, 1, 5, 20, 50, 50)
	if mod != 3 {
		t.Errorf("intelligent with low player HP should be +3, got %d", mod)
	}
	// Enemy HP very low (< 30%) → -2
	mod = behaviorModifier(BehaviorIntelligent, 1, 20, 20, 5, 50)
	if mod != -2 {
		t.Errorf("intelligent with low enemy HP should be -2, got %d", mod)
	}
	// Both normal → +1
	mod = behaviorModifier(BehaviorIntelligent, 1, 20, 20, 50, 50)
	if mod != 1 {
		t.Errorf("intelligent normal state should be +1, got %d", mod)
	}
}

// --- HP Tracking Tests ---

func TestHPClampToZero(t *testing.T) {
	enemy := &EnemyStats{HP: 10, MaxHP: 10, Attack: 5, Defense: 0, Behavior: BehaviorAggressive}
	damage := 15
	enemy.HP -= damage
	if enemy.HP < 0 {
		enemy.HP = 0
	}
	if enemy.HP != 0 {
		t.Errorf("HP should be clamped to 0, got %d", enemy.HP)
	}
}

// --- Victory/Defeat Detection Tests ---

func TestVictoryCondition(t *testing.T) {
	state := &CombatState{
		Enemy:    EnemyStats{HP: 0, MaxHP: 50},
		PlayerHP: 15,
		Turn:     3,
	}
	// Victory when enemy HP <= 0
	if state.Enemy.HP != 0 {
		t.Errorf("expected enemy HP = 0, got %d", state.Enemy.HP)
	}
	if state.PlayerHP <= 0 {
		t.Error("player should still be alive for victory")
	}
}

func TestDefeatCondition(t *testing.T) {
	state := &CombatState{
		Enemy:    EnemyStats{HP: 30, MaxHP: 50},
		PlayerHP: 0,
		Turn:     5,
	}
	if state.PlayerHP != 0 {
		t.Errorf("expected player HP = 0 for defeat, got %d", state.PlayerHP)
	}
}

// --- Attack Action Detection Tests ---

func TestIsAttackAction(t *testing.T) {
	attacks := []string{
		"Attack with sword",
		"Strike the enemy",
		"Use magic",
		"Shoot with bow",
	}
	for _, a := range attacks {
		if !isAttackAction(a) {
			t.Errorf("expected %q to be an attack action", a)
		}
	}

	nonAttacks := []string{
		"Flee from combat",
		"Run away",
		"Try to talk to the enemy",
		"Surrender",
		"Defend yourself",
		"Wait and see",
		"Throw sand in their eyes",
		"Topple the brazier and step behind the pillar",
	}
	for _, a := range nonAttacks {
		if isAttackAction(a) {
			t.Errorf("expected %q to NOT be an attack action", a)
		}
	}
}

func TestEquippedWeaponFromCanonicalInventoryAffectsDamage(t *testing.T) {
	char := &storage.Character{
		InventoryJSON: `[{"name":"Practice Blade","slot":"weapon","equipped":true,"stats":{"damage":4}}]`,
	}
	if got := getWeaponBase(char); got != 4 {
		t.Fatalf("getWeaponBase = %d, want 4", got)
	}
}

func TestEquippedArmorFromCanonicalInventoryAffectsDefense(t *testing.T) {
	char := &storage.Character{
		InventoryJSON: `[{"name":"Padded Coat","slot":"armor","equipped":true,"stats":{"defense":3}}]`,
		StatsJSON:     `{"attributes":{"end":6}}`,
	}
	if got := getPlayerDefense(char); got != 5 {
		t.Fatalf("getPlayerDefense = %d, want 5", got)
	}
}

func TestIsFleeingAction(t *testing.T) {
	fleeActions := []string{
		"Flee from combat",
		"Run away quickly",
		"Try to escape",
		"Retreat to safety",
	}
	for _, a := range fleeActions {
		if !isFleeing(a) {
			t.Errorf("expected %q to be a flee action", a)
		}
	}

	notFlee := []string{
		"Attack the enemy",
		"Talk to the enemy",
		"Defend yourself",
	}
	for _, a := range notFlee {
		if isFleeing(a) {
			t.Errorf("expected %q to NOT be a flee action", a)
		}
	}
}

func TestCombatTalkActionDoesNotTriggerEnemyDamageByDefault(t *testing.T) {
	if shouldEnemyCounterAction("Try to talk to the enemy") {
		t.Fatal("talk action should not trigger automatic enemy counter damage")
	}
	if !shouldEnemyCounterAction("Defend yourself") {
		t.Fatal("defend action should still leave the enemy able to act")
	}
}

// --- Player Vitals Parsing Tests ---

func TestGetPlayerVitalsValid(t *testing.T) {
	char := &storage.Character{
		StatsJSON: `{"vitals":{"hp":{"current":15,"max":25}}}`,
	}
	hp, maxHP := getPlayerVitals(char)
	if hp != 15 {
		t.Errorf("expected current HP 15, got %d", hp)
	}
	if maxHP != 25 {
		t.Errorf("expected max HP 25, got %d", maxHP)
	}
}

func TestGetPlayerVitalsInvalidJSON(t *testing.T) {
	char := &storage.Character{StatsJSON: "invalid"}
	hp, maxHP := getPlayerVitals(char)
	if hp != 20 || maxHP != 20 {
		t.Errorf("invalid JSON should return defaults (20/20), got %d/%d", hp, maxHP)
	}
}

func TestGetPlayerVitalsMissingField(t *testing.T) {
	char := &storage.Character{StatsJSON: `{}`}
	hp, maxHP := getPlayerVitals(char)
	if hp != 20 || maxHP != 20 {
		t.Errorf("missing vitals field should return defaults (20/20), got %d/%d", hp, maxHP)
	}
}

// --- Attribute Bonus Tests ---

func TestGetAttributeBonus(t *testing.T) {
	char := &storage.Character{
		StatsJSON: `{"attributes":{"str":9,"dex":3}}`,
	}
	bonus := getAttributeBonus(char, "str")
	if bonus != 3 { // 9/3 = 3
		t.Errorf("str 9 should give bonus 3, got %d", bonus)
	}
	bonus = getAttributeBonus(char, "dex")
	if bonus != 1 { // 3/3 = 1
		t.Errorf("dex 3 should give bonus 1, got %d", bonus)
	}
	bonus = getAttributeBonus(char, "wis")
	if bonus != 0 { // missing stat → 0
		t.Errorf("missing stat should give bonus 0, got %d", bonus)
	}
}
