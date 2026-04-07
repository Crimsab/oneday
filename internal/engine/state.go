package engine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

// StateChange records a single change that was applied to character or world state.
type StateChange struct {
	Target string      // "character" or "world"
	Field  string      // e.g. "vitals.hp.current", "location", "attributes.str"
	Old    interface{}
	New    interface{}
}

// ApplyStateChanges takes the state_changes map from an AI response
// and applies validated changes to the character and world state.
// Returns a list of changes that were applied (for logging/display).
func ApplyStateChanges(
	changes map[string]interface{},
	char *storage.Character,
	world *storage.WorldState,
) ([]StateChange, error) {
	if len(changes) == 0 {
		return nil, nil
	}

	// Parse character stats JSON into a mutable map.
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(char.StatsJSON), &stats); err != nil {
		return nil, fmt.Errorf("parsing character stats: %w", err)
	}

	var applied []StateChange

	for key, val := range changes {
		switch key {
		case "vitals":
			vitalsMap, ok := toStringMap(val)
			if !ok {
				continue
			}
			statsVitals, _ := stats["vitals"].(map[string]interface{})
			if statsVitals == nil {
				statsVitals = map[string]interface{}{}
				stats["vitals"] = statsVitals
			}
			for vitalKey, vitalVal := range vitalsMap {
				subMap, ok := toStringMap(vitalVal)
				if !ok {
					continue
				}
				existing, _ := statsVitals[vitalKey].(map[string]interface{})
				if existing == nil {
					existing = map[string]interface{}{}
				}
				for subKey, subVal := range subMap {
					oldVal := existing[subKey]
					newVal := toFloat(subVal)
					// Clamp HP-like values to >= 0
					if newVal < 0 {
						newVal = 0
					}
					existing[subKey] = newVal
					applied = append(applied, StateChange{
						Target: "character",
						Field:  fmt.Sprintf("vitals.%s.%s", vitalKey, subKey),
						Old:    oldVal,
						New:    newVal,
					})
				}
				statsVitals[vitalKey] = existing
			}

		case "attributes":
			attrMap, ok := toStringMap(val)
			if !ok {
				continue
			}
			statsAttrs, _ := stats["attributes"].(map[string]interface{})
			if statsAttrs == nil {
				statsAttrs = map[string]interface{}{}
				stats["attributes"] = statsAttrs
			}
			for attrKey, attrVal := range attrMap {
				oldVal := statsAttrs[attrKey]
				newVal := toFloat(attrVal)
				if newVal < 0 {
					newVal = 0
				}
				statsAttrs[attrKey] = newVal
				applied = append(applied, StateChange{
					Target: "character",
					Field:  fmt.Sprintf("attributes.%s", attrKey),
					Old:    oldVal,
					New:    newVal,
				})
			}

		case "secondary":
			secMap, ok := toStringMap(val)
			if !ok {
				continue
			}
			statsSec, _ := stats["secondary"].(map[string]interface{})
			if statsSec == nil {
				statsSec = map[string]interface{}{}
				stats["secondary"] = statsSec
			}
			for secKey, secVal := range secMap {
				oldVal := statsSec[secKey]
				newVal := toFloat(secVal)
				statsSec[secKey] = newVal
				applied = append(applied, StateChange{
					Target: "character",
					Field:  fmt.Sprintf("secondary.%s", secKey),
					Old:    oldVal,
					New:    newVal,
				})
			}

		case "location":
			locStr, ok := val.(string)
			if !ok {
				continue
			}
			if locStr != "" && locStr != world.CurrentLocation {
				old := world.CurrentLocation
				world.CurrentLocation = locStr
				applied = append(applied, StateChange{
					Target: "world",
					Field:  "location",
					Old:    old,
					New:    locStr,
				})
			}

		case "currency":
			oldVal := stats["currency"]
			newVal := toFloat(val)
			if newVal < 0 {
				newVal = 0
			}
			stats["currency"] = newVal
			applied = append(applied, StateChange{
				Target: "character",
				Field:  "currency",
				Old:    oldVal,
				New:    newVal,
			})

		case "inventory_add":
			items := toStringSlice(val)
			if len(items) == 0 {
				continue
			}
			inv, _ := stats["inventory"].(map[string]interface{})
			if inv == nil {
				inv = map[string]interface{}{"backpack": []interface{}{}}
				stats["inventory"] = inv
			}
			backpack, _ := inv["backpack"].([]interface{})
			for _, item := range items {
				backpack = append(backpack, item)
				applied = append(applied, StateChange{
					Target: "character",
					Field:  "inventory.backpack",
					Old:    nil,
					New:    item,
				})
			}
			inv["backpack"] = backpack

		case "inventory_remove":
			items := toStringSlice(val)
			if len(items) == 0 {
				continue
			}
			inv, _ := stats["inventory"].(map[string]interface{})
			if inv == nil {
				continue
			}
			backpack, _ := inv["backpack"].([]interface{})
			removeSet := make(map[string]bool)
			for _, item := range items {
				removeSet[item] = true
			}
			newBackpack := make([]interface{}, 0, len(backpack))
			for _, item := range backpack {
				itemStr, _ := item.(string)
				if !removeSet[itemStr] {
					newBackpack = append(newBackpack, item)
				} else {
					applied = append(applied, StateChange{
						Target: "character",
						Field:  "inventory.backpack",
						Old:    item,
						New:    nil,
					})
				}
			}
			inv["backpack"] = newBackpack
		}
	}

	// Marshal the modified stats back to character.
	statsBytes, err := json.Marshal(stats)
	if err != nil {
		return applied, fmt.Errorf("marshaling updated character stats: %w", err)
	}
	char.StatsJSON = string(statsBytes)
	char.UpdatedAt = time.Now()
	world.UpdatedAt = time.Now()

	return applied, nil
}

// toStringMap attempts to cast val to map[string]interface{}.
func toStringMap(val interface{}) (map[string]interface{}, bool) {
	m, ok := val.(map[string]interface{})
	return m, ok
}

// toFloat converts a JSON numeric value (float64, int, etc.) to float64.
func toFloat(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	}
	return 0
}

// toStringSlice converts an interface{} to []string.
func toStringSlice(val interface{}) []string {
	slice, ok := val.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
