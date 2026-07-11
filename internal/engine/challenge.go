package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

// ChallengeEngine resolves challenges mechanically.
type ChallengeEngine struct {
	rng *RNGService
}

// NewChallengeEngine creates a new ChallengeEngine.
func NewChallengeEngine() *ChallengeEngine {
	return NewChallengeEngineWithRNG(defaultRNGService())
}

func NewChallengeEngineWithRNG(rng *RNGService) *ChallengeEngine {
	if rng == nil {
		rng = defaultRNGService()
	}
	return &ChallengeEngine{rng: rng}
}

// Resolve executes a challenge spec against the current game state.
// db and storyID are needed only for relationship checks.
func (ce *ChallengeEngine) Resolve(
	spec *ChallengeSpec,
	char *storage.Character,
	db *storage.DB,
	storyID string,
) (*ChallengeResult, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil challenge spec")
	}

	var result *ChallengeResult
	var err error
	switch spec.Type {
	case ChallengeStatCheck:
		result, err = ce.resolveStatCheck(spec, char)
	case ChallengeDiceRoll:
		result, err = ce.resolveDiceRoll(spec)
	case ChallengeItemCheck:
		result, err = ce.resolveItemCheck(spec, char)
	case ChallengeSkillCheck:
		result, err = ce.resolveSkillCheck(spec, char)
	case ChallengeRelCheck:
		result, err = ce.resolveRelationshipCheck(spec, char, db, storyID)
	case ChallengeMiniGame:
		return nil, fmt.Errorf("mini-game %q requires TUI interaction; use minigame resolvers directly", spec.MiniGame)
	default:
		return nil, fmt.Errorf("unknown challenge type: %q", spec.Type)
	}
	if err != nil || result == nil {
		return result, err
	}
	if result.Outcome == nil {
		outcome := OutcomeFromLegacy(result.Passed, result.Difficulty)
		if len(result.RollLog) > 0 || result.Roll != 0 {
			outcome.Roll, outcome.Total = result.Roll, result.Total
			outcome.Margin = result.Total - result.Difficulty
		}
		if len(result.RollLog) > 0 {
			outcome.Seed = result.RollLog[0].Seed & contracts.MaxPortableChallengeSeed
		}
		result.Outcome = &outcome
	}
	return result, nil
}

// resolveStatCheck compares a character stat value against a difficulty threshold.
func (ce *ChallengeEngine) resolveStatCheck(spec *ChallengeSpec, char *storage.Character) (*ChallengeResult, error) {
	statVal, err := getStatValue(char.StatsJSON, "attributes."+spec.Stat)
	if err != nil {
		// Try vitals path (e.g., "hp" maps to vitals.hp.current)
		statVal, err = getStatValue(char.StatsJSON, "vitals."+spec.Stat+".current")
		if err != nil {
			return nil, fmt.Errorf("stat %q not found: %w", spec.Stat, err)
		}
	}

	val := int(statVal)
	passed := val >= spec.Difficulty
	outcome := "PASS"
	if !passed {
		outcome = "FAIL"
	}

	return &ChallengeResult{
		Passed:     passed,
		Difficulty: spec.Difficulty,
		Detail:     fmt.Sprintf("%s %d vs difficulty %d → %s", strings.ToUpper(spec.Stat), val, spec.Difficulty, outcome),
	}, nil
}

// resolveDiceRoll performs a d100 roll with modifiers against a difficulty.
func (ce *ChallengeEngine) resolveDiceRoll(spec *ChallengeSpec) (*ChallengeResult, error) {
	rollRecord := ce.rng.Roll("challenge.dice_roll", 100)
	roll := rollRecord.Raw

	// Critical on raw roll before modifiers.
	isCriticalSuccess := roll >= 96
	isCriticalFailure := roll <= 5

	modSum := 0
	for _, m := range spec.Modifiers {
		modSum += m.Value
	}
	total := roll + modSum
	rollRecord.Modifiers = spec.Modifiers
	rollRecord.Total = total
	rollRecord.Target = spec.Difficulty

	passed := total >= spec.Difficulty
	// Critical overrides everything.
	if isCriticalSuccess {
		passed = true
	}
	if isCriticalFailure {
		passed = false
	}

	outcome := "PASS"
	if !passed {
		outcome = "FAIL"
	}
	if isCriticalSuccess {
		outcome = "CRITICAL SUCCESS"
	}
	if isCriticalFailure {
		outcome = "CRITICAL FAILURE"
	}
	rollRecord.Outcome = outcome

	detail := fmt.Sprintf("rolled %d + modifiers %+d = %d vs difficulty %d → %s",
		roll, modSum, total, spec.Difficulty, outcome)

	return &ChallengeResult{
		Passed:     passed,
		Roll:       roll,
		Total:      total,
		Difficulty: spec.Difficulty,
		Modifiers:  spec.Modifiers,
		Detail:     detail,
		RollLog:    []RollRecord{rollRecord},
	}, nil
}

// resolveItemCheck verifies that the character has a required item in inventory.
func (ce *ChallengeEngine) resolveItemCheck(spec *ChallengeSpec, char *storage.Character) (*ChallengeResult, error) {
	found := hasItem(char.InventoryJSON, spec.Item)
	if !found {
		// Fallback: check StatsJSON inventory (older format)
		found = hasItem(char.StatsJSON, spec.Item)
	}

	outcome := "PASS"
	if !found {
		outcome = "FAIL"
	}

	return &ChallengeResult{
		Passed: found,
		Detail: fmt.Sprintf("item %q in inventory → %s", spec.Item, outcome),
	}, nil
}

// resolveSkillCheck verifies that the character has a skill at the required level.
func (ce *ChallengeEngine) resolveSkillCheck(spec *ChallengeSpec, char *storage.Character) (*ChallengeResult, error) {
	level, found := getSkillLevel(char.SkillsJSON, spec.Skill)
	if !found {
		// Fallback: check StatsJSON skills
		level, found = getSkillLevel(char.StatsJSON, spec.Skill)
	}

	if !found {
		return &ChallengeResult{
			Passed: false,
			Detail: fmt.Sprintf("skill %q not learned → FAIL", spec.Skill),
		}, nil
	}

	passed := true
	if spec.SkillLevel > 0 {
		passed = level >= spec.SkillLevel
	}

	outcome := "PASS"
	if !passed {
		outcome = "FAIL"
	}

	detail := fmt.Sprintf("skill %q level %d", spec.Skill, level)
	if spec.SkillLevel > 0 {
		detail += fmt.Sprintf(" vs required %d → %s", spec.SkillLevel, outcome)
	} else {
		detail += fmt.Sprintf(" (present) → %s", outcome)
	}

	return &ChallengeResult{
		Passed: passed,
		Detail: detail,
	}, nil
}

// resolveRelationshipCheck verifies an NPC's disposition meets a threshold.
func (ce *ChallengeEngine) resolveRelationshipCheck(
	spec *ChallengeSpec,
	char *storage.Character,
	db *storage.DB,
	storyID string,
) (*ChallengeResult, error) {
	if db == nil {
		return nil, fmt.Errorf("db required for relationship check")
	}

	npc, err := db.GetNPCByName(storyID, spec.NPCName)
	if err != nil || npc == nil {
		return &ChallengeResult{
			Passed: false,
			Detail: fmt.Sprintf("NPC %q not found → FAIL", spec.NPCName),
		}, nil
	}

	passed := npc.Disposition >= spec.Disposition
	outcome := "PASS"
	if !passed {
		outcome = "FAIL"
	}

	return &ChallengeResult{
		Passed: passed,
		Detail: fmt.Sprintf("%s disposition %d vs required %d → %s",
			spec.NPCName, npc.Disposition, spec.Disposition, outcome),
	}, nil
}

// --- Helpers ---

// getStatValue navigates a nested JSON object using a dot-separated path.
// e.g., "attributes.str" retrieves stats["attributes"]["str"].
func getStatValue(statsJSON string, path string) (float64, error) {
	if statsJSON == "" {
		return 0, fmt.Errorf("empty stats JSON")
	}

	var root map[string]interface{}
	if err := json.Unmarshal([]byte(statsJSON), &root); err != nil {
		return 0, fmt.Errorf("parsing stats JSON: %w", err)
	}

	parts := strings.SplitN(path, ".", 2)
	key := parts[0]
	val, ok := root[key]
	if !ok {
		return 0, fmt.Errorf("key %q not found", key)
	}

	if len(parts) == 1 {
		// Leaf: convert to float64
		switch v := val.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case json.Number:
			f, err := v.Float64()
			return f, err
		default:
			return 0, fmt.Errorf("key %q is not a number", key)
		}
	}

	// Recurse into nested map.
	nested, ok := val.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("key %q is not an object", key)
	}

	nestedJSON, err := json.Marshal(nested)
	if err != nil {
		return 0, err
	}

	return getStatValue(string(nestedJSON), parts[1])
}

// hasItem checks canonical item identifiers in inventory JSON.
// It matches id, slug, or name after normalization; descriptions and raw JSON
// text are intentionally ignored so challenge requirements cannot pass through
// accidental substring matches.
func hasItem(jsonData string, itemName string) bool {
	if jsonData == "" || itemName == "" {
		return false
	}

	needle := normalizeItemLookupKey(itemName)
	if needle == "" {
		return false
	}

	var decoded interface{}
	if err := json.Unmarshal([]byte(jsonData), &decoded); err != nil {
		return false
	}
	return inventoryValueHasItem(decoded, needle)
}

func inventoryValueHasItem(value interface{}, needle string) bool {
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			if inventoryValueHasItem(item, needle) {
				return true
			}
		}
	case map[string]interface{}:
		if inventoryMapItemMatches(typed, needle) {
			return true
		}
		for _, key := range []string{"backpack", "inventory", "items", "equipped", "quest"} {
			if nested, ok := typed[key]; ok && inventoryValueHasItem(nested, needle) {
				return true
			}
		}
	case string:
		return normalizeItemLookupKey(typed) == needle
	}
	return false
}

func inventoryMapItemMatches(item map[string]interface{}, needle string) bool {
	for _, key := range []string{"id", "slug", "name"} {
		if value, ok := item[key].(string); ok && normalizeItemLookupKey(value) == needle {
			return true
		}
	}
	return false
}

func normalizeItemLookupKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("_", " ", "-", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

// getSkillLevel looks up a skill by name and returns its level.
// Returns (level, found).
func getSkillLevel(jsonData string, skillName string) (int, bool) {
	if jsonData == "" || skillName == "" {
		return 0, false
	}

	needle := strings.ToLower(skillName)

	var root map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &root); err != nil {
		return 0, false
	}

	// Look for skills in root or inside a "skills" key.
	skillsMap, ok := root["skills"].(map[string]interface{})
	if !ok {
		// Try root itself as skills map
		skillsMap = root
	}

	for k, v := range skillsMap {
		if strings.ToLower(k) == needle {
			switch sv := v.(type) {
			case float64:
				return int(sv), true
			case map[string]interface{}:
				if level, ok := sv["level"].(float64); ok {
					return int(level), true
				}
				return 1, true // skill exists but no level field
			default:
				return 1, true // skill exists
			}
		}
	}

	return 0, false
}
