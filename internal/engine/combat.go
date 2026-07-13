package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

// CombatEngine manages a single combat encounter.
type CombatEngine struct {
	state            *CombatState
	narrator         *Narrator
	session          *GameSession
	rng              *RNGService
	expectedBranchID string
	expectedHeadID   string
	expectedRevision int64
}

// NewCombatEngine starts a combat encounter.
// enemy is the AI-generated enemy spec. The engine validates and clamps values.
func NewCombatEngine(narrator *Narrator, enemy *EnemyStats) (*CombatEngine, error) {
	// Validate and clamp enemy stats.
	ValidateEnemy(enemy)

	timeline, err := narrator.db.GetActiveTimeline(narrator.story.ID)
	if err != nil {
		return nil, fmt.Errorf("loading combat timeline: %w", err)
	}
	if timeline.Branch.ID != narrator.story.ActiveBranchID {
		return nil, fmt.Errorf("combat story branch is stale")
	}

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
		state:            state,
		narrator:         narrator,
		session:          narrator.session,
		rng:              NewDefaultRNGService(),
		expectedBranchID: timeline.Branch.ID,
		expectedHeadID:   timeline.Commit.ID,
		expectedRevision: narrator.story.Revision,
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
	originalState := *ce.state
	originalCharacter := *ce.narrator.character
	originalRNG := ce.rng.snapshot()
	completed := false
	defer func() {
		if !completed {
			*ce.state = originalState
			*ce.narrator.character = originalCharacter
			ce.rng.restore(originalRNG)
		}
	}()

	result := &CombatTurnResult{
		Choices: defaultCombatChoices(),
		Mood:    "tense",
	}
	var resp ai.Response
	var latency int64
	var mechanicalNote string

	// --- Engine calculates player damage (if action is attack-like) ---
	isAttack := isAttackAction(action)
	if isAttack {
		weaponBase := getWeaponBase(ce.narrator.character)
		attrBonus := getAttributeBonus(ce.narrator.character, "str")
		playerRoll := ce.rng.Roll("combat.player_attack", 20)
		d20 := playerRoll.Raw
		rawDamage := weaponBase + attrBonus + d20
		playerRoll.Modifiers = []Modifier{{Source: "Weapon", Value: weaponBase}, {Source: "STR", Value: attrBonus}}
		playerRoll.Total = rawDamage
		playerRoll.Target = ce.state.Enemy.Defense
		actualDamage := rawDamage - ce.state.Enemy.Defense
		if actualDamage < 0 {
			actualDamage = 0
		}
		playerRoll.Outcome = fmt.Sprintf("damage:%d", actualDamage)
		ce.state.Enemy.HP -= actualDamage
		if ce.state.Enemy.HP < 0 {
			ce.state.Enemy.HP = 0
		}
		result.EnemyDamage = actualDamage
		result.RollLog = append(result.RollLog, playerRoll)
	}

	// --- Enemy counter-attack (unless combat is already over) ---
	playerDamage := 0
	if ce.state.Enemy.HP > 0 && shouldEnemyCounterAction(action) {
		enemyAttack := ce.state.Enemy.Attack + behaviorModifier(ce.state.Enemy.Behavior, ce.state.Turn, ce.state.PlayerHP, ce.state.PlayerMaxHP, ce.state.Enemy.HP, ce.state.Enemy.MaxHP)
		enemyRoll := ce.rng.Roll("combat.enemy_attack", 20)
		d20 := enemyRoll.Raw
		playerDefense := getPlayerDefense(ce.narrator.character)
		rawPlayerDamage := enemyAttack + d20 - playerDefense
		enemyRoll.Modifiers = []Modifier{{Source: "Enemy attack", Value: enemyAttack}, {Source: "Player defense", Value: -playerDefense}}
		enemyRoll.Total = rawPlayerDamage
		enemyRoll.Target = playerDefense
		if rawPlayerDamage < 0 {
			rawPlayerDamage = 0
		}
		enemyRoll.Outcome = fmt.Sprintf("damage:%d", rawPlayerDamage)
		ce.state.PlayerHP -= rawPlayerDamage
		if ce.state.PlayerHP < 0 {
			ce.state.PlayerHP = 0
		}
		playerDamage = rawPlayerDamage
		result.RollLog = append(result.RollLog, enemyRoll)
	}

	result.PlayerDamage = playerDamage
	result.PlayerHP = ce.state.PlayerHP
	result.EnemyHP = ce.state.Enemy.HP
	combatOutcome := OutcomeFromLegacy(result.EnemyDamage > 0, ce.state.Enemy.Defense)
	if len(result.RollLog) > 0 {
		combatOutcome.Seed = result.RollLog[0].Seed & contracts.MaxPortableChallengeSeed
		combatOutcome.Roll = result.RollLog[0].Raw
		combatOutcome.Total = result.RollLog[0].Total
		combatOutcome.Margin = result.RollLog[0].Total - result.RollLog[0].Target
	}
	if result.EnemyDamage > 0 && result.PlayerDamage > 0 {
		combatOutcome.Degree = contracts.OutcomeSuccessWithCost
		applyDefaultOutcomeBudget(&combatOutcome, DefaultOutcomePolicy("combat", "balanced"))
	} else if result.EnemyDamage == 0 && result.PlayerDamage > 0 {
		combatOutcome.Degree = contracts.OutcomeHardFailure
		applyDefaultOutcomeBudget(&combatOutcome, DefaultOutcomePolicy("combat", "balanced"))
	}
	result.Outcome = &combatOutcome
	if ce.state.Enemy.HP <= 0 {
		result.Outcome.Degree = contracts.OutcomeCriticalSuccess
	}
	if ce.state.PlayerHP <= 0 {
		result.Outcome.Degree = contracts.OutcomeCatastrophe
		applyDefaultOutcomeBudget(result.Outcome, DefaultOutcomePolicy("combat", "balanced"))
	}

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
		if err := ce.syncPlayerHP(); err != nil {
			return nil, err
		}
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
		if err := ce.syncPlayerHP(); err != nil {
			return nil, err
		}
	} else {
		// Combat continues: check flee action.
		if isFleeing(action) {
			// Dice roll to determine if flee succeeds.
			fleeRecord := ce.rng.Roll("combat.flee", 100)
			fleeRoll := fleeRecord.Raw
			fleeRecord.Target = 50
			if fleeRoll >= 50 {
				fleeRecord.Outcome = "success"
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
				mechanicalNote = fmt.Sprintf("[You managed to escape! (fled on roll %d/100)]", fleeRoll)
				if err := ce.syncPlayerHP(); err != nil {
					return nil, err
				}
			} else {
				fleeRecord.Outcome = "failure"
				mechanicalNote = fmt.Sprintf("[Escape failed! (rolled %d/100, needed 50+)]", fleeRoll)
			}
			result.RollLog = append(result.RollLog, fleeRecord)
		}
	}

	if result.Narrative == "" {
		instance := NewOrdinaryActionChallenge(ce.narrator.story.ID, ce.narrator.story.ActiveBranchID, ce.narrator.session.Turn(), fmt.Sprintf("combat:%d:%s", ce.state.Turn, action), DefaultOutcomePolicy("combat", "balanced"))
		instance.Definition.ID, instance.Definition.Kind, instance.Definition.Difficulty = "combat-turn", "combat", maxInt(1, ce.state.Enemy.Defense)
		if result.Outcome.Seed == 0 {
			result.Outcome.Seed = instance.Seed
		}
		resolution := contracts.ChallengeResolution{ProtocolVersion: contracts.ChallengeProtocolVersion, InstanceID: instance.ID, Input: contracts.ChallengeInput{ActorID: ce.narrator.character.ID, Intent: action}, Outcome: *result.Outcome}
		result.ChallengeInstance, result.ChallengeResolution = &instance, &resolution
		narrative, aiResp, aiLatency, err := ce.narrateResolvedCombatTurn(ctx, action, result)
		if err != nil {
			return nil, err
		}
		resp = aiResp
		latency = aiLatency
		result.Narrative = narrative.Narrative
		result.Choices = narrative.Choices
		result.Mood = narrative.Mood
	}
	if result.ChallengeResolution == nil {
		instance := NewOrdinaryActionChallenge(ce.narrator.story.ID, ce.narrator.story.ActiveBranchID, ce.narrator.session.Turn(), fmt.Sprintf("combat:%d:%s", ce.state.Turn, action), DefaultOutcomePolicy("combat", "balanced"))
		instance.Definition.ID, instance.Definition.Kind, instance.Definition.Difficulty = "combat-turn", "combat", maxInt(1, ce.state.Enemy.Defense)
		if result.Outcome.Seed == 0 {
			result.Outcome.Seed = instance.Seed
		}
		resolution := contracts.ChallengeResolution{ProtocolVersion: contracts.ChallengeProtocolVersion, InstanceID: instance.ID, Input: contracts.ChallengeInput{ActorID: ce.narrator.character.ID, Intent: action}, Outcome: *result.Outcome}
		result.ChallengeInstance, result.ChallengeResolution = &instance, &resolution
	}
	committedRevision, summaryEntry, err := ce.commitCombatOutcome(result)
	if err != nil {
		return nil, err
	}
	ce.narrator.story.Revision = committedRevision
	ce.expectedRevision = committedRevision
	if mechanicalNote != "" {
		result.Narrative += "\n\n" + mechanicalNote
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
				Narrative:           result.Narrative,
				Mood:                result.Mood,
				RollLog:             result.RollLog,
				ResolvedOutcome:     result.Outcome,
				ChallengeInstance:   result.ChallengeInstance,
				ChallengeResolution: result.ChallengeResolution,
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
	} else if err := ce.session.CloseSubSession(ce.state.SubSessionID); err != nil {
		log.Printf("oneday: combat outcome persisted canonically but closing sub-session failed: %v", err)
	}
	if summaryEntry != nil {
		if err := ce.session.writeJSONLEntry(*summaryEntry); err != nil {
			log.Printf("oneday: combat summary persisted canonically but jsonl mirror failed: %v", err)
		}
	}

	completed = true
	return result, nil
}

func (ce *CombatEngine) commitCombatOutcome(result *CombatTurnResult) (int64, *ChatEntry, error) {
	if !result.CombatOver {
		var revision int64
		err := ce.narrator.db.WithTx(func(tx *sql.Tx) error {
			if err := ce.requireExpectedTimelineTx(tx); err != nil {
				return err
			}
			if err := ce.narrator.db.RecordChallengeResolutionAtHeadTx(tx, ce.narrator.story.ID, ce.narrator.session.SessionID(), ce.narrator.session.Turn(), *result.ChallengeInstance, *result.ChallengeResolution); err != nil {
				return err
			}
			if err := ce.narrator.db.UpdateCharacterFullTx(tx, ce.narrator.character); err != nil {
				return err
			}
			var err error
			revision, err = ce.narrator.db.BumpStoryRevisionTx(tx, ce.narrator.story.ID)
			return err
		})
		if err != nil {
			return 0, nil, fmt.Errorf("persisting combat outcome and character: %w", err)
		}
		return revision, nil, nil
	}

	entry := ce.combatSummaryEntry()
	combatLog := ce.combatLog()
	var revision int64
	err := ce.narrator.db.WithTx(func(tx *sql.Tx) error {
		if err := ce.requireExpectedTimelineTx(tx); err != nil {
			return err
		}
		if err := ce.narrator.db.RecordChallengeResolutionAtHeadTx(tx, ce.narrator.story.ID, ce.narrator.session.SessionID(), ce.narrator.session.Turn(), *result.ChallengeInstance, *result.ChallengeResolution); err != nil {
			return err
		}
		if err := ce.narrator.db.UpdateCharacterFullTx(tx, ce.narrator.character); err != nil {
			return err
		}
		if err := ce.narrator.db.InsertCombatLogTx(tx, combatLog); err != nil {
			return err
		}
		if err := ce.session.appendEntryToDBAtLineage(tx, ce.narrator.db, entry, ce.expectedBranchID, ce.expectedHeadID); err != nil {
			return err
		}
		var err error
		revision, err = ce.narrator.db.BumpStoryRevisionTx(tx, ce.narrator.story.ID)
		return err
	})
	if err != nil {
		return 0, nil, fmt.Errorf("finalizing combat outcome: %w", err)
	}
	return revision, &entry, nil
}

func (ce *CombatEngine) requireExpectedTimelineTx(tx *sql.Tx) error {
	if err := ce.narrator.db.RequireStoryRevisionTx(tx, ce.narrator.story.ID, ce.expectedRevision); err != nil {
		return err
	}
	timeline, err := ce.narrator.db.EnsureStoryTimelineTx(tx, ce.narrator.story.ID)
	if err != nil {
		return err
	}
	if timeline.Branch.ID != ce.expectedBranchID || timeline.Commit.ID != ce.expectedHeadID {
		return fmt.Errorf("combat timeline changed while encounter was active")
	}
	return nil
}

func defaultCombatChoices() []Choice {
	return []Choice{
		{ID: 1, Text: "Attack"},
		{ID: 2, Text: "Defend"},
		{ID: 3, Text: "Try to flee"},
		{ID: 4, Text: "Try talking"},
	}
}

func (ce *CombatEngine) narrateResolvedCombatTurn(ctx context.Context, action string, result *CombatTurnResult) (*NarrativeResponse, ai.Response, int64, error) {
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

	messages := []ai.Message{
		{Role: ai.RoleSystem, Content: systemPrompt},
		{Role: ai.RoleUser, Content: fmt.Sprintf(
			"[Combat Turn %d] Player action: %s\n\nEngine resolution already happened. Narrate these facts without contradicting them:\n%s",
			ce.state.Turn,
			action,
			combatResolutionForPrompt(result),
		)},
	}
	if result.Outcome != nil {
		messages = appendOutcomeGuidance(messages, "Authoritative combat outcome: "+mustOutcomeJSON(result.Outcome))
	}

	start := time.Now()
	req := ai.Request{
		Messages:       messages,
		Temperature:    0.85,
		MaxTokens:      1024,
		ResponseFormat: ai.NarrativeResponseFormat(),
	}

	resp, err := ce.narrator.router.Complete(ce.narrator.telemetryContext(ctx, "combat_narration", ""), req)
	if err != nil {
		return nil, resp, 0, fmt.Errorf("AI combat response: %w", err)
	}
	latency := time.Since(start).Milliseconds()

	narrative, parseErr := parseNarrativeFromAI(resp.Content)
	if parseErr != nil {
		narrative = &NarrativeResponse{
			Narrative: resp.Content,
			Choices:   defaultCombatChoices(),
			Mood:      "tense",
		}
	}
	EnforceOutcomeNarrative(narrative, result.Outcome)
	return narrative, resp, latency, nil
}

func mustOutcomeJSON(outcome *contracts.OutcomeEnvelope) string {
	raw, _ := json.Marshal(outcome)
	return string(raw)
}

func combatResolutionForPrompt(result *CombatTurnResult) string {
	var lines []string
	if result.EnemyDamage > 0 {
		lines = append(lines, fmt.Sprintf("- Player dealt %d damage to the enemy.", result.EnemyDamage))
	} else {
		lines = append(lines, "- Player dealt no direct damage to the enemy.")
	}
	if result.PlayerDamage > 0 {
		lines = append(lines, fmt.Sprintf("- Enemy dealt %d damage to the player.", result.PlayerDamage))
	} else {
		lines = append(lines, "- Enemy dealt no direct damage to the player.")
	}
	lines = append(lines, fmt.Sprintf("- Player HP is now %d.", result.PlayerHP))
	lines = append(lines, fmt.Sprintf("- Enemy HP is now %d.", result.EnemyHP))
	for _, roll := range result.RollLog {
		lines = append(lines, fmt.Sprintf("- %s: %s raw %d total %d outcome %s.", roll.Source, roll.Die, roll.Raw, roll.Total, roll.Outcome))
	}
	if result.CombatOver {
		if result.Victory {
			lines = append(lines, "- Combat is over: player victory.")
		} else {
			lines = append(lines, fmt.Sprintf("- Combat is over: %s.", result.DefeatOutcome))
		}
	}
	return strings.Join(lines, "\n")
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
	resp, err := ce.narrator.router.Complete(ce.narrator.telemetryContext(ctx, "combat_victory", ""), req)
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
	resp, err := ce.narrator.router.Complete(ce.narrator.telemetryContext(ctx, "combat_defeat", ""), req)
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
func (ce *CombatEngine) syncPlayerHP() error {
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(ce.narrator.character.StatsJSON), &stats); err != nil {
		return fmt.Errorf("parsing character stats while syncing combat HP: %w", err)
	}
	vitals, ok := stats["vitals"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("character stats are missing vitals while syncing combat HP")
	}
	hp, ok := vitals["hp"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("character vitals are missing hp while syncing combat HP")
	}
	hp["current"] = ce.state.PlayerHP
	vitals["hp"] = hp
	stats["vitals"] = vitals

	b, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshaling character stats while syncing combat HP: %w", err)
	}
	ce.narrator.character.StatsJSON = string(b)
	return nil
}

func (ce *CombatEngine) combatSummaryEntry() ChatEntry {
	return ChatEntry{
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
	if damage := maxEquippedItemStat(char.InventoryJSON, "weapon", "damage"); damage > 0 {
		return damage
	}
	if damage := maxEquippedItemStat(char.StatsJSON, "weapon", "damage"); damage > 0 {
		return damage
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
	defense := sumEquippedItemStat(char.InventoryJSON, "armor", "defense")
	defense += sumEquippedItemStat(char.StatsJSON, "armor", "defense")

	// Add endurance bonus.
	endBonus := getAttributeBonus(char, "end")
	defense += endBonus

	return defense
}

func maxEquippedItemStat(jsonData, slot, stat string) int {
	values := collectEquippedItemStats(jsonData, slot, stat)
	maxValue := 0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func sumEquippedItemStat(jsonData, slot, stat string) int {
	total := 0
	for _, value := range collectEquippedItemStats(jsonData, slot, stat) {
		total += value
	}
	return total
}

func collectEquippedItemStats(jsonData, slot, stat string) []int {
	if strings.TrimSpace(jsonData) == "" || strings.TrimSpace(jsonData) == "null" {
		return nil
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(jsonData), &decoded); err != nil {
		return nil
	}
	return collectEquippedItemStatsFromValue(decoded, normalizeItemLookupKey(slot), stat)
}

func collectEquippedItemStatsFromValue(value interface{}, slot, stat string) []int {
	var values []int
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			values = append(values, collectEquippedItemStatsFromValue(item, slot, stat)...)
		}
	case map[string]interface{}:
		if statValue, ok := equippedItemStat(typed, slot, stat); ok {
			values = append(values, statValue)
		}
		if equipped, ok := typed["equipped"]; ok {
			switch eq := equipped.(type) {
			case []interface{}:
				for _, item := range eq {
					values = append(values, collectEquippedItemStatsFromValue(item, slot, stat)...)
				}
			case map[string]interface{}:
				if item, ok := eq[slot].(map[string]interface{}); ok {
					if statValue, ok := itemNumericStat(item, stat); ok {
						values = append(values, statValue)
					}
				}
				values = append(values, collectEquippedItemStatsFromValue(eq, slot, stat)...)
			}
		}
		for _, key := range []string{"inventory", "items", "backpack"} {
			if nested, ok := typed[key]; ok {
				values = append(values, collectEquippedItemStatsFromValue(nested, slot, stat)...)
			}
		}
	}
	return values
}

func equippedItemStat(item map[string]interface{}, slot, stat string) (int, bool) {
	equipped, _ := item["equipped"].(bool)
	if !equipped {
		return 0, false
	}
	itemSlot := normalizeItemLookupKey(stringValue(item["slot"]))
	itemType := normalizeItemLookupKey(stringValue(item["type"]))
	if itemSlot != slot && itemType != slot {
		return 0, false
	}
	return itemNumericStat(item, stat)
}

func itemNumericStat(item map[string]interface{}, stat string) (int, bool) {
	if value, ok := item[stat]; ok {
		statValue := int(toFloat(value))
		if statValue > 0 {
			return statValue, true
		}
	}
	if statsMap, ok := item["stats"].(map[string]interface{}); ok {
		statValue := int(toFloat(statsMap[stat]))
		if statValue > 0 {
			return statValue, true
		}
	}
	return 0, false
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
	attackWords := []string{"attack", "hit", "strike", "slash", "stab", "punch", "kick", "shoot", "fire", "cast", "spell", "magic", "blast", "smash", "cut", "bite"}
	for _, w := range attackWords {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}

func shouldEnemyCounterAction(action string) bool {
	lower := strings.ToLower(action)
	if isFleeing(action) {
		return false
	}
	nonCounterWords := []string{"talk", "negotiate", "surrender", "wait"}
	for _, w := range nonCounterWords {
		if strings.Contains(lower, w) {
			return false
		}
	}
	return isAttackAction(action) || strings.Contains(lower, "defend") || strings.Contains(lower, "guard") || strings.Contains(lower, "block")
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

func (ce *CombatEngine) combatLog() *storage.CombatLog {
	return &storage.CombatLog{
		StoryID:        ce.narrator.story.ID,
		SessionID:      ce.session.SessionID(),
		EnemyName:      ce.state.Enemy.Name,
		EnemyHP:        ce.state.Enemy.MaxHP,
		Turns:          ce.state.Turn,
		Victory:        ce.state.Victory,
		DefeatOutcome:  ce.state.DefeatOutcome,
		PlayerHPStart:  ce.state.PlayerMaxHP,
		PlayerHPEnd:    ce.state.PlayerHP,
		CreatedAt:      time.Now(),
		BranchID:       ce.expectedBranchID,
		SourceCommitID: ce.expectedHeadID,
	}
}

// RandomEnemyStats creates a placeholder enemy for testing.
// In production, enemies always come from AI.
func RandomEnemyStats(name string) *EnemyStats {
	rng := defaultRNGService()
	return &EnemyStats{
		Name:     name,
		HP:       rng.Roll("combat.random_enemy.hp", 40).Raw + 9,
		MaxHP:    0, // ValidateEnemy will set this
		Attack:   rng.Roll("combat.random_enemy.attack", 10).Raw + 2,
		Defense:  rng.Roll("combat.random_enemy.defense", 5).Raw - 1,
		Behavior: BehaviorAggressive,
	}
}
