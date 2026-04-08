package engine

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

// ChallengeEngine resolves challenges mechanically.
type ChallengeEngine struct{}

// NewChallengeEngine creates a new ChallengeEngine.
func NewChallengeEngine() *ChallengeEngine {
	return &ChallengeEngine{}
}

// RollD100 rolls a 100-sided die. Returns a value in [1, 100].
func RollD100() int {
	return rand.Intn(100) + 1
}

// RollD20 rolls a 20-sided die. Returns a value in [1, 20].
func RollD20() int {
	return rand.Intn(20) + 1
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

	switch spec.Type {
	case ChallengeStatCheck:
		return ce.resolveStatCheck(spec, char)
	case ChallengeDiceRoll:
		return ce.resolveDiceRoll(spec)
	case ChallengeItemCheck:
		return ce.resolveItemCheck(spec, char)
	case ChallengeSkillCheck:
		return ce.resolveSkillCheck(spec, char)
	case ChallengeRelCheck:
		return ce.resolveRelationshipCheck(spec, char, db, storyID)
	case ChallengeMiniGame:
		return nil, fmt.Errorf("mini-game %q requires TUI interaction; use minigame resolvers directly", spec.MiniGame)
	default:
		return nil, fmt.Errorf("unknown challenge type: %q", spec.Type)
	}
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
	roll := RollD100()

	// Critical on raw roll before modifiers.
	isCriticalSuccess := roll >= 96
	isCriticalFailure := roll <= 5

	modSum := 0
	for _, m := range spec.Modifiers {
		modSum += m.Value
	}
	total := roll + modSum

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

	detail := fmt.Sprintf("rolled %d + modifiers %+d = %d vs difficulty %d → %s",
		roll, modSum, total, spec.Difficulty, outcome)

	return &ChallengeResult{
		Passed:     passed,
		Roll:       roll,
		Total:      total,
		Difficulty: spec.Difficulty,
		Modifiers:  spec.Modifiers,
		Detail:     detail,
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

// hasItem checks if an item name appears in inventory JSON (case-insensitive substring match).
func hasItem(jsonData string, itemName string) bool {
	if jsonData == "" || itemName == "" {
		return false
	}

	needle := strings.ToLower(itemName)

	// Try structured inventory: {"backpack": [{"name": "..."}]}
	var invMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &invMap); err == nil {
		// Check "backpack" array
		if backpack, ok := invMap["backpack"]; ok {
			if items, ok := backpack.([]interface{}); ok {
				for _, item := range items {
					switch i := item.(type) {
					case string:
						if strings.Contains(strings.ToLower(i), needle) {
							return true
						}
					case map[string]interface{}:
						if name, ok := i["name"].(string); ok {
							if strings.Contains(strings.ToLower(name), needle) {
								return true
							}
						}
					}
				}
			}
		}

		// Also check inside a nested "inventory" key (stats.json format)
		if inv, ok := invMap["inventory"]; ok {
			invJSON, _ := json.Marshal(inv)
			return hasItem(string(invJSON), itemName)
		}
	}

	// Fallback: raw string search (handles various formats)
	return strings.Contains(strings.ToLower(jsonData), needle)
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
