package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/storage"
)

// CombatEngine manages a single combat encounter.
type CombatEngine struct {
	state     *CombatState
	narrator  *Narrator
	session   *GameSession
	challenge *ChallengeEngine
}

// NewCombatEngine starts a combat encounter.
// enemy is the AI-generated enemy spec. The engine validates and clamps values.
func NewCombatEngine(narrator *Narrator, enemy *EnemyStats) (*CombatEngine, error) {
	// Validate and clamp enemy stats.
	ValidateEnemy(enemy)

	// Open a sub-session for this combat.
	subSessionID, err := narrator.session.OpenSubSession("combat")
	if err != nil {
		return nil, fmt.Errorf("opening combat sub-session: %w", err)
	}

	// Read player's current HP from character stats.
	playerHP, playerMaxHP := getPlayerVitals(narrator.character)

	state := &CombatState{
		Enemy:        *enemy,
		PlayerHP:     playerHP,
		PlayerMaxHP:  playerMaxHP,
		Turn:         1,
		SubSessionID: subSessionID,
		Phase:        "player_turn",
		Resolved:     false,
	}

	return &CombatEngine{
		state:     state,
		narrator:  narrator,
		session:   narrator.session,
		challenge: NewChallengeEngine(),
	}, nil
}

// State returns a copy of the current combat state (read-only for TUI).
func (ce *CombatEngine) State() *CombatState {
	s := *ce.state
	return &s
}

// ValidateEnemy clamps enemy stats to reasonable ranges and sets defaults.
func ValidateEnemy(enemy *EnemyStats) {
	if enemy.HP < 1 {
		enemy.HP = 1
	} else if enemy.HP > 999 {
		enemy.HP = 999
	}
	if enemy.MaxHP < enemy.HP {
		enemy.MaxHP = enemy.HP
	}
	if enemy.Attack < 0 {
		enemy.Attack = 0
	} else if enemy.Attack > 50 {
		enemy.Attack = 50
	}
	if enemy.Defense < 0 {
		enemy.Defense = 0
	} else if enemy.Defense > 30 {
		enemy.Defense = 30
	}
	if enemy.Behavior == "" {
		enemy.Behavior = BehaviorAggressive
	}
}

// PlayerAction sends the player's combat action to AI and processes the full turn
// (player action + enemy counter-attack). Returns CombatTurnResult for the TUI.
func (ce *CombatEngine) PlayerAction(ctx context.Context, action string) (*CombatTurnResult, error) {
	if ce.state.Resolved {
		return nil, fmt.Errorf("combat is already resolved")
	}

	// Build combat context for AI.
	enemyJSON, _ := json.Marshal(ce.state.Enemy)
	settingJSON := ce.narrator.story.SettingJSON

	systemPrompt := prompts.CombatSystem(
		ce.narrator.story.Name,
		ce.narrator.story.Language,
		ce.narrator.story.WritingStyle,
		ce.narrator.story.PromptDirectives,
		settingJSON,
		ce.narrator.character.Name,
		ce.narrator.character.StatsJSON,
		string(enemyJSON),
		ce.state.Turn,
	)

	// Build messages: system + recent combat turns + player action.
	messages := []ai.Message{
		{Role: ai.RoleSystem, Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("[Combat Turn %d] Player action: %s", ce.state.Turn, action)},
	}

	start := time.Now()
	req := ai.Request{
		Messages:       messages,
		Temperature:    0.85,
		MaxTokens:      1024,
		ResponseFormat: ai.NarrativeResponseFormat(),
	}

	resp, err := ce.narrator.router.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI combat response: %w", err)
	}
	latency := time.Since(start).Milliseconds()

	// Parse AI narrative response.
	narrative, parseErr := parseNarrativeFromAI(resp.Content)
	if parseErr != nil {
		// Fall back to minimal narrative.
		narrative = &NarrativeResponse{
			Narrative: resp.Content,
			Choices: []Choice{
				{ID: 1, Text: "Attack"},
				{ID: 2, Text: "Defend"},
				{ID: 3, Text: "Try to flee"},
				{ID: 4, Text: "Try talking"},
			},
			Mood: "tense",
		}
	}

	result := &CombatTurnResult{
		Narrative: narrative.Narrative,
		Choices:   narrative.Choices,
		Mood:      narrative.Mood,
	}

	// --- Engine calculates player damage (if action is attack-like) ---
	isAttack := isAttackAction(action)
	if isAttack {
		weaponBase := getWeaponBase(ce.narrator.character)
		attrBonus := getAttributeBonus(ce.narrator.character, "str")
		d20 := RollD20()
		rawDamage := weaponBase + attrBonus + d20
		actualDamage := rawDamage - ce.state.Enemy.Defense
		if actualDamage < 0 {
			actualDamage = 0
		}
		ce.state.Enemy.HP -= actualDamage
		if ce.state.Enemy.HP < 0 {
			ce.state.Enemy.HP = 0
		}
		result.EnemyDamage = actualDamage
	}

	// --- Enemy counter-attack (unless combat is already over) ---
	playerDamage := 0
	if ce.state.Enemy.HP > 0 && !isFleeing(action) {
		enemyAttack := ce.state.Enemy.Attack + behaviorModifier(ce.state.Enemy.Behavior, ce.state.Turn, ce.state.PlayerHP, ce.state.PlayerMaxHP, ce.state.Enemy.HP, ce.state.Enemy.MaxHP)
		d20 := RollD20()
		playerDefense := getPlayerDefense(ce.narrator.character)
		rawPlayerDamage := enemyAttack + d20 - playerDefense
		if rawPlayerDamage < 0 {
			rawPlayerDamage = 0
		}
		ce.state.PlayerHP -= rawPlayerDamage
		if ce.state.PlayerHP < 0 {
			ce.state.PlayerHP = 0
		}
		playerDamage = rawPlayerDamage
	}

	result.PlayerDamage = playerDamage
	result.PlayerHP = ce.state.PlayerHP
	result.EnemyHP = ce.state.Enemy.HP

	// --- Check win/lose conditions ---
	if ce.state.Enemy.HP <= 0 {
		ce.state.Resolved = true
		ce.state.Victory = true
		ce.state.Phase = "resolved"

		// Ask AI to narrate victory.
		victoryNarrative, victoryErr := ce.narrateVictory(ctx)
		if victoryErr == nil {
			result.Narrative = victoryNarrative.Narrative
			result.Choices = victoryNarrative.Choices
			result.Mood = victoryNarrative.Mood
		}

		summary := fmt.Sprintf("Combat victory against %s after %d turns. Player HP: %d/%d.",
			ce.state.Enemy.Name, ce.state.Turn, ce.state.PlayerHP, ce.state.PlayerMaxHP)
		ce.state.Summary = summary
		result.Summary = summary
		result.CombatOver = true
		result.Victory = true

		// Sync player HP to character stats.
		ce.syncPlayerHP()

		// Close sub-session.
		_ = ce.session.CloseSubSession(ce.state.SubSessionID)
	} else if ce.state.PlayerHP <= 0 {
		ce.state.Resolved = true
		ce.state.Victory = false
		ce.state.Phase = "resolved"

		// Ask AI to decide defeat outcome.
		defeatNarrative, defeatOutcome, defeatErr := ce.narrateDefeat(ctx)
		if defeatErr == nil {
			result.Narrative = defeatNarrative
			ce.state.DefeatOutcome = defeatOutcome
		} else {
			ce.state.DefeatOutcome = "unconscious"
		}

		summary := fmt.Sprintf("Combat defeat against %s after %d turns. Outcome: %s.",
			ce.state.Enemy.Name, ce.state.Turn, ce.state.DefeatOutcome)
		ce.state.Summary = summary
		result.Summary = summary
		result.CombatOver = true
		result.Victory = false
		result.DefeatOutcome = ce.state.DefeatOutcome

		// Sync player HP to character stats.
		ce.syncPlayerHP()

		// Close sub-session.
		_ = ce.session.CloseSubSession(ce.state.SubSessionID)
	} else {
		// Combat continues: check flee action.
		if isFleeing(action) {
			// Dice roll to determine if flee succeeds.
			fleeRoll := RollD100()
			if fleeRoll >= 50 {
				// Flee succeeds.
				ce.state.Resolved = true
				ce.state.Victory = false
				ce.state.Phase = "resolved"
				ce.state.DefeatOutcome = "retreat"
				summary := fmt.Sprintf("Player fled from %s after %d turns.", ce.state.Enemy.Name, ce.state.Turn)
				ce.state.Summary = summary
				result.Summary = summary
				result.CombatOver = true
				result.Victory = false
				result.DefeatOutcome = "retreat"
				result.Narrative += fmt.Sprintf("\n\n[You managed to escape! (fled on roll %d/100)]", fleeRoll)
				ce.syncPlayerHP()
				_ = ce.session.CloseSubSession(ce.state.SubSessionID)
			} else {
				result.Narrative += fmt.Sprintf("\n\n[Escape failed! (rolled %d/100, needed 50+)]", fleeRoll)
			}
		}
	}

	// Log this turn to sub-session JSONL (only if not resolved, or if just resolved).
	if !result.CombatOver || ce.state.Resolved {
		entry := ChatEntry{
			Timestamp:   time.Now(),
			MessageType: "combat",
			Input: &ChatInput{
				Type: "combat_action",
				Text: action,
			},
			Output: &ChatOutput{
				Narrative: result.Narrative,
				Mood:      result.Mood,
			},
			AIModel:   resp.Model,
			AILatency: latency,
		}
		if !result.CombatOver {
			// Only append if sub-session is still open.
			_ = ce.session.AppendSubTurn(ce.state.SubSessionID, entry)
		}
	}

	// Advance turn counter.
	if !result.CombatOver {
		ce.state.Turn++
	}

	return result, nil
}

// narrateVictory asks AI to narrate the victory moment.
func (ce *CombatEngine) narrateVictory(ctx context.Context) (*NarrativeResponse, error) {
	prompt := prompts.CombatVictoryPrompt(
		ce.narrator.story.Language,
		ce.narrator.story.WritingStyle,
		ce.narrator.story.PromptDirectives,
		ce.narrator.character.Name,
		ce.state.Enemy.Name,
		ce.state.Turn,
	)
	messages := []ai.Message{
		{Role: "user", Content: prompt},
	}
	req := ai.Request{
		Messages:       messages,
		Temperature:    0.85,
		MaxTokens:      512,
		ResponseFormat: ai.NarrativeResponseFormat(),
	}
	resp, err := ce.narrator.router.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	return parseNarrativeFromAI(resp.Content)
}

// narrateDefeat asks AI to decide and narrate the defeat outcome.
func (ce *CombatEngine) narrateDefeat(ctx context.Context) (string, string, error) {
	summary := fmt.Sprintf("Player reached 0 HP on turn %d. Enemy had %d/%d HP remaining.",
		ce.state.Turn, ce.state.Enemy.HP, ce.state.Enemy.MaxHP)

	prompt := prompts.CombatDefeatPrompt(
		ce.narrator.story.Name,
		ce.narrator.story.Language,
		ce.narrator.story.WritingStyle,
		ce.narrator.story.PromptDirectives,
		ce.narrator.character.Name,
		ce.state.Enemy.Name,
		summary,
	)
	messages := []ai.Message{
		{Role: "user", Content: prompt},
	}
	req := ai.Request{
		Messages:       messages,
		Temperature:    0.85,
		MaxTokens:      512,
		ResponseFormat: ai.CombatDefeatResponseFormat(),
	}
	resp, err := ce.narrator.router.Complete(ctx, req)
	if err != nil {
		return "", "unconscious", err
	}

	// Parse defeat response.
	type defeatResponse struct {
		Outcome   string `json:"outcome"`
		Narrative string `json:"narrative"`
	}

	raw := resp.Content
	// Try to extract JSON block.
	var defeatResp defeatResponse
	if payload, err := ai.ExtractJSONPayload(raw); err == nil && payload != "" {
		_ = json.Unmarshal([]byte(payload), &defeatResp)
	}
	if defeatResp.Outcome == "" {
		defeatResp.Outcome = "unconscious"
	}
	if defeatResp.Narrative == "" {
		defeatResp.Narrative = raw
	}

	return defeatResp.Narrative, defeatResp.Outcome, nil
}

// syncPlayerHP writes the current player HP back to character.StatsJSON.
func (ce *CombatEngine) syncPlayerHP() {
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(ce.narrator.character.StatsJSON), &stats); err != nil {
		return
	}
	vitals, ok := stats["vitals"].(map[string]interface{})
	if !ok {
		return
	}
	hp, ok := vitals["hp"].(map[string]interface{})
	if !ok {
		return
	}
	hp["current"] = ce.state.PlayerHP
	vitals["hp"] = hp
	stats["vitals"] = vitals

	b, err := json.Marshal(stats)
	if err != nil {
		return
	}
	ce.narrator.character.StatsJSON = string(b)
	_ = ce.narrator.db.UpdateCharacterFull(ce.narrator.character)
}

// WriteSummaryToMain writes the combat outcome summary to the main narrative
// history without consuming a new story turn.
func (ce *CombatEngine) WriteSummaryToMain() error {
	if ce.state.Summary == "" {
		return nil
	}
	entry := ChatEntry{
		Turn:        ce.narrator.World().CurrentTurn,
		Timestamp:   time.Now(),
		MessageType: "combat_summary",
		Chapter:     ce.narrator.World().CurrentChapter,
		Location:    ce.narrator.World().CurrentLocation,
		Output: &ChatOutput{
			Narrative: ce.state.Summary,
			Mood:      "neutral",
			Location:  ce.narrator.World().CurrentLocation,
		},
	}
	return ce.session.AppendHistoryEntry(ce.narrator.db, entry)
}

// --- Helper functions ---

// getPlayerVitals reads current/max HP from character stats.
// Returns sensible defaults if stats are not parseable.
func getPlayerVitals(char *storage.Character) (currentHP, maxHP int) {
	currentHP = 20
	maxHP = 20

	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(char.StatsJSON), &stats); err != nil {
		return
	}
	vitals, ok := stats["vitals"].(map[string]interface{})
	if !ok {
		return
	}
	hp, ok := vitals["hp"].(map[string]interface{})
	if !ok {
		return
	}
	if cur, ok := hp["current"].(float64); ok {
		currentHP = int(cur)
	}
	if max, ok := hp["max"].(float64); ok {
		maxHP = int(max)
	}
	return
}

// getWeaponBase checks the equipped weapon for a base damage value.
// Returns 1 if no weapon is equipped (fist fighting).
func getWeaponBase(char *storage.Character) int {
	// Parse inventory to find equipped weapon.
	var inventory map[string]interface{}
	if err := json.Unmarshal([]byte(char.InventoryJSON), &inventory); err != nil {
		return 1
	}
	equipped, ok := inventory["equipped"].(map[string]interface{})
	if !ok {
		return 1
	}
	weapon, ok := equipped["weapon"].(map[string]interface{})
	if !ok {
		return 1
	}
	if damage, ok := weapon["damage"].(float64); ok {
		return int(damage)
	}
	// Check stats_json for equipped weapon if not in inventory.
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(char.StatsJSON), &stats); err != nil {
		return 1
	}
	if inv, ok := stats["inventory"].(map[string]interface{}); ok {
		if eq, ok := inv["equipped"].(map[string]interface{}); ok {
			if w, ok := eq["weapon"].(map[string]interface{}); ok {
				if d, ok := w["damage"].(float64); ok {
					return int(d)
				}
			}
		}
	}
	return 1
}

// getAttributeBonus returns the bonus from an attribute (value / 3, rounded down).
func getAttributeBonus(char *storage.Character, attr string) int {
	val, err := getStatValue(char.StatsJSON, "attributes."+attr)
	if err != nil {
		return 0
	}
	return int(val) / 3
}

// getPlayerDefense returns the player's defense value from armor + endurance bonus.
func getPlayerDefense(char *storage.Character) int {
	defense := 0

	// Check equipped armor.
	var inventory map[string]interface{}
	if err := json.Unmarshal([]byte(char.InventoryJSON), &inventory); err == nil {
		if equipped, ok := inventory["equipped"].(map[string]interface{}); ok {
			if armor, ok := equipped["armor"].(map[string]interface{}); ok {
				if def, ok := armor["defense"].(float64); ok {
					defense += int(def)
				}
			}
		}
	}

	// Add endurance bonus.
	endBonus := getAttributeBonus(char, "end")
	defense += endBonus

	return defense
}

// behaviorModifier adjusts enemy attack power based on behavior pattern, turn, and HP.
func behaviorModifier(behavior EnemyBehavior, turn, playerHP, playerMaxHP, enemyHP, enemyMaxHP int) int {
	switch behavior {
	case BehaviorAggressive:
		return 2

	case BehaviorDefensive:
		// -2 normally, but on even turns the enemy counter-attacks hard.
		if turn%2 == 0 {
			return 4
		}
		return -2

	case BehaviorTactical:
		// Alternates between +1 and +3.
		if turn%2 == 0 {
			return 3
		}
		return 1

	case BehaviorBeast:
		// +3 on turns divisible by 3 (telegraphed big attack), 0 otherwise.
		if turn%3 == 0 {
			return 3
		}
		return 0

	case BehaviorIntelligent:
		// Adapts based on HP percentages.
		playerHPPct := 0
		if playerMaxHP > 0 {
			playerHPPct = playerHP * 100 / playerMaxHP
		}
		enemyHPPct := 100
		if enemyMaxHP > 0 {
			enemyHPPct = enemyHP * 100 / enemyMaxHP
		}
		if playerHPPct < 30 {
			// Player is low — press the advantage.
			return 3
		}
		if enemyHPPct < 30 {
			// Enemy is low — cautious/retreating.
			return -2
		}
		return 1

	default:
		return 0
	}
}

// isAttackAction returns true if the action text represents an attack.
// Actions mentioning attack, hit, strike, slash, cast, shoot, etc. are attacks.
// Flee, talk, hide, wait actions are not attacks.
func isAttackAction(action string) bool {
	lower := strings.ToLower(action)
	fleeWords := []string{"flee", "run", "escape", "retreat", "hide", "talk", "negotiate", "surrender", "wait", "defend"}
	for _, w := range fleeWords {
		if strings.Contains(lower, w) {
			return false
		}
	}
	// Default: treat unknown actions as attacks.
	return true
}

// isFleeing returns true if the player is trying to flee combat.
func isFleeing(action string) bool {
	lower := strings.ToLower(action)
	fleeWords := []string{"flee", "run away", "escape", "retreat", "get out"}
	for _, w := range fleeWords {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}

// LogCombatResult saves combat outcome to the combat_log table.
func (ce *CombatEngine) LogCombatResult(db *storage.DB, storyID string) error {
	log := &storage.CombatLog{
		StoryID:       storyID,
		SessionID:     ce.session.SessionID(),
		EnemyName:     ce.state.Enemy.Name,
		EnemyHP:       ce.state.Enemy.MaxHP,
		Turns:         ce.state.Turn,
		Victory:       ce.state.Victory,
		DefeatOutcome: ce.state.DefeatOutcome,
		PlayerHPStart: ce.state.PlayerMaxHP,
		PlayerHPEnd:   ce.state.PlayerHP,
		CreatedAt:     time.Now(),
	}
	return db.InsertCombatLog(log)
}

// RandomEnemyStats creates a placeholder enemy for testing.
// In production, enemies always come from AI.
func RandomEnemyStats(name string) *EnemyStats {
	return &EnemyStats{
		Name:     name,
		HP:       rand.Intn(40) + 10,
		MaxHP:    0, // ValidateEnemy will set this
		Attack:   rand.Intn(10) + 3,
		Defense:  rand.Intn(5),
		Behavior: BehaviorAggressive,
	}
}
