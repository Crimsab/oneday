package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
)

// StateChange records a single change that was applied to character or world state.
type StateChange struct {
	Target string      // "character" or "world"
	Field  string      // e.g. "vitals.hp.current", "location", "attributes.str"
	Old    interface{}
	New    interface{}
	// Description is a human-readable summary (used for character growth events).
	Description string
}

// ApplyStateChanges takes the state_changes map from an AI response
// and applies validated changes to the character and world state.
// Returns a list of changes that were applied (for logging/display).
// db and storyID are required for NPC operations; currentTurn tracks the current turn number.
func ApplyStateChanges(
	changes map[string]interface{},
	char *storage.Character,
	world *storage.WorldState,
	db *storage.DB,
	storyID string,
	currentTurn int,
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

		// --- Character growth cases ---

		case "trait_add":
			traitName, ok := val.(string)
			if !ok || traitName == "" {
				continue
			}
			traits := toStringSlice(stats["traits"])
			// Check for duplicate (case-insensitive)
			dup := false
			for _, t := range traits {
				if strings.EqualFold(t, traitName) {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			traits = append(traits, traitName)
			stats["traits"] = toInterfaceSlice(traits)
			// Sync dedicated traits_json column
			if tb, err := json.Marshal(traits); err == nil {
				char.TraitsJSON = string(tb)
			}
			applied = append(applied, StateChange{
				Target:      "character",
				Field:       "traits",
				Old:         nil,
				New:         traitName,
				Description: fmt.Sprintf("Gained trait: %s", traitName),
			})

		case "title_add":
			titleName, ok := val.(string)
			if !ok || titleName == "" {
				continue
			}
			titles := toStringSlice(stats["titles"])
			// Check for duplicate (case-insensitive)
			dup := false
			for _, t := range titles {
				if strings.EqualFold(t, titleName) {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
			titles = append(titles, titleName)
			stats["titles"] = toInterfaceSlice(titles)
			applied = append(applied, StateChange{
				Target:      "character",
				Field:       "titles",
				Old:         nil,
				New:         titleName,
				Description: fmt.Sprintf("Earned title: %s", titleName),
			})

		case "skill_learn":
			skillName, ok := val.(string)
			if !ok || skillName == "" {
				continue
			}
			skills := toSkillsMap(stats["skills"])
			if _, exists := skills[skillName]; exists {
				continue // already known
			}
			skills[skillName] = map[string]interface{}{"level": 1, "xp": 0}
			stats["skills"] = skills
			// Sync dedicated skills_json column
			if sb, err := json.Marshal(skills); err == nil {
				char.SkillsJSON = string(sb)
			}
			applied = append(applied, StateChange{
				Target:      "character",
				Field:       fmt.Sprintf("skills.%s", skillName),
				Old:         nil,
				New:         skills[skillName],
				Description: fmt.Sprintf("Learned new skill: %s", skillName),
			})

		case "skill_xp":
			xpMap, ok := toStringMap(val)
			if !ok {
				continue
			}
			skillName, _ := xpMap["skill"].(string)
			if skillName == "" {
				continue
			}
			xpGain := int(toFloat(xpMap["xp"]))
			if xpGain <= 0 {
				continue
			}
			skills := toSkillsMap(stats["skills"])
			skill, exists := skills[skillName]
			if !exists {
				skill = map[string]interface{}{"level": 1, "xp": 0}
			}
			skillMap, _ := skill.(map[string]interface{})
			if skillMap == nil {
				skillMap = map[string]interface{}{"level": 1, "xp": 0}
			}
			level := int(toFloat(skillMap["level"]))
			if level < 1 {
				level = 1
			}
			xp := int(toFloat(skillMap["xp"])) + xpGain
			desc := fmt.Sprintf("Gained %d XP in %s", xpGain, skillName)
			// Level up: threshold = level * 100
			for xp >= level*100 {
				xp -= level * 100
				level++
				desc += fmt.Sprintf(" | %s leveled up to %d!", skillName, level)
			}
			skillMap["level"] = level
			skillMap["xp"] = xp
			skills[skillName] = skillMap
			stats["skills"] = skills
			// Sync dedicated skills_json column
			if sb, err := json.Marshal(skills); err == nil {
				char.SkillsJSON = string(sb)
			}
			applied = append(applied, StateChange{
				Target:      "character",
				Field:       fmt.Sprintf("skills.%s", skillName),
				Old:         nil,
				New:         skillMap,
				Description: desc,
			})

		// --- NPC operation cases ---

		case "new_npc":
			npcRaw, ok := toStringMap(val)
			if !ok || db == nil {
				continue
			}
			npcData, err := ParseNPCData(npcRaw)
			if err != nil {
				// Non-fatal: log but continue
				applied = append(applied, StateChange{
					Target:      "world",
					Field:       "npc",
					Description: fmt.Sprintf("NPC parse error: %v", err),
				})
				continue
			}
			// Check if NPC already exists
			existing, err := db.GetNPCByName(storyID, npcData.Name)
			if err != nil {
				continue
			}
			if existing != nil {
				// Update last_seen_turn and persist
				existing.LastSeenTurn = currentTurn
				_ = db.UpdateNPC(existing)
				applied = append(applied, StateChange{
					Target:      "world",
					Field:       "npc",
					New:         npcData.Name,
					Description: fmt.Sprintf("NPC seen again: %s", npcData.Name),
				})
			} else {
				// Create new NPC
				npc, err := NPCToStorage(npcData, storyID, currentTurn)
				if err != nil {
					continue
				}
				if err := db.CreateNPC(npc); err != nil {
					continue
				}
				applied = append(applied, StateChange{
					Target:      "world",
					Field:       "npc",
					New:         npcData.Name,
					Description: fmt.Sprintf("New NPC encountered: %s (%s)", npcData.Name, npcData.Role),
				})
			}

		case "npc_disposition":
			dispMap, ok := toStringMap(val)
			if !ok || db == nil {
				continue
			}
			npcName, _ := dispMap["name"].(string)
			if npcName == "" {
				continue
			}
			npc, err := db.GetNPCByName(storyID, npcName)
			if err != nil || npc == nil {
				continue
			}
			oldDisp := npc.Disposition
			var newDisp int
			if changeVal, hasChange := dispMap["change"]; hasChange {
				newDisp = oldDisp + int(toFloat(changeVal))
			} else if setValue, hasValue := dispMap["value"]; hasValue {
				newDisp = int(toFloat(setValue))
			} else {
				continue
			}
			// Clamp to [-100, 100]
			if newDisp > 100 {
				newDisp = 100
			} else if newDisp < -100 {
				newDisp = -100
			}
			_ = db.UpdateNPCDisposition(npc.ID, newDisp)
			diff := newDisp - oldDisp
			sign := "+"
			if diff < 0 {
				sign = ""
			}
			applied = append(applied, StateChange{
				Target:      "world",
				Field:       fmt.Sprintf("npc.%s.disposition", npcName),
				Old:         oldDisp,
				New:         newDisp,
				Description: fmt.Sprintf("%s's disposition: %s%d (now %d)", npcName, sign, diff, newDisp),
			})

		case "npc_thoughts":
			thoughtMap, ok := toStringMap(val)
			if !ok || db == nil {
				continue
			}
			npcName, _ := thoughtMap["name"].(string)
			thought, _ := thoughtMap["thought"].(string)
			if npcName == "" || thought == "" {
				continue
			}
			npc, err := db.GetNPCByName(storyID, npcName)
			if err != nil || npc == nil {
				continue
			}
			var thoughts []string
			_ = json.Unmarshal([]byte(npc.PrivateThoughts), &thoughts)
			thoughts = append(thoughts, thought)
			if tb, err := json.Marshal(thoughts); err == nil {
				npc.PrivateThoughts = string(tb)
			}
			_ = db.UpdateNPC(npc)
			applied = append(applied, StateChange{
				Target:      "world",
				Field:       fmt.Sprintf("npc.%s.private_thoughts", npcName),
				New:         thought,
				Description: fmt.Sprintf("%s had a new thought (private)", npcName),
			})

		case "npc_notes":
			noteMap, ok := toStringMap(val)
			if !ok || db == nil {
				continue
			}
			npcName, _ := noteMap["name"].(string)
			note, _ := noteMap["note"].(string)
			if npcName == "" || note == "" {
				continue
			}
			npc, err := db.GetNPCByName(storyID, npcName)
			if err != nil || npc == nil {
				continue
			}
			var notes []string
			_ = json.Unmarshal([]byte(npc.NotesOnProtagonist), &notes)
			notes = append(notes, note)
			if nb, err := json.Marshal(notes); err == nil {
				npc.NotesOnProtagonist = string(nb)
			}
			_ = db.UpdateNPC(npc)
			applied = append(applied, StateChange{
				Target:      "world",
				Field:       fmt.Sprintf("npc.%s.notes_on_protagonist", npcName),
				New:         note,
				Description: fmt.Sprintf("%s noted something about protagonist (private)", npcName),
			})
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

// toInterfaceSlice converts []string to []interface{} for JSON marshal into stats.
func toInterfaceSlice(ss []string) []interface{} {
	result := make([]interface{}, len(ss))
	for i, s := range ss {
		result[i] = s
	}
	return result
}

// toSkillsMap extracts the skills map from stats (key "skills").
// Returns an empty map if not present or wrong type.
func toSkillsMap(val interface{}) map[string]interface{} {
	if m, ok := val.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// ApplyNarratorStateChanges handles narrator-specific (meta/world-building) state changes.
// These are changes that come from /narrator commands and affect the story setting or world state.
// db, story, and world are required. ragPipeline is optional (for embedding lore).
func ApplyNarratorStateChanges(
	ctx context.Context,
	changes map[string]interface{},
	db *storage.DB,
	story *storage.Story,
	world *storage.WorldState,
	ragPipeline *rag.RAG,
) error {
	if len(changes) == 0 {
		return nil
	}

	// Parse the story's setting JSON.
	var setting map[string]interface{}
	if err := json.Unmarshal([]byte(story.SettingJSON), &setting); err != nil {
		setting = map[string]interface{}{}
	}

	settingModified := false
	worldModified := false

	for key, val := range changes {
		switch key {

		// --- Story setting mutations ---

		case "setting_factions_add":
			items := toStringOrSlice(val)
			if len(items) == 0 {
				continue
			}
			factions := toStringSlice(setting["factions"])
			factions = appendUnique(factions, items...)
			setting["factions"] = toInterfaceSlice(factions)
			settingModified = true

		case "setting_cultures_add":
			items := toStringOrSlice(val)
			if len(items) == 0 {
				continue
			}
			cultures := toStringSlice(setting["cultures"])
			cultures = appendUnique(cultures, items...)
			setting["cultures"] = toInterfaceSlice(cultures)
			settingModified = true

		case "setting_dangers_add":
			items := toStringOrSlice(val)
			if len(items) == 0 {
				continue
			}
			dangers := toStringSlice(setting["dangers"])
			dangers = appendUnique(dangers, items...)
			setting["dangers"] = toInterfaceSlice(dangers)
			settingModified = true

		case "setting_rules_add":
			items := toStringOrSlice(val)
			if len(items) == 0 {
				continue
			}
			rules := toStringSlice(setting["rules"])
			rules = appendUnique(rules, items...)
			setting["rules"] = toInterfaceSlice(rules)
			settingModified = true

		case "setting_tone_add":
			toneStr, ok := val.(string)
			if !ok || toneStr == "" {
				continue
			}
			existing, _ := setting["tone_guidelines"].(string)
			if existing != "" {
				setting["tone_guidelines"] = existing + "; " + toneStr
			} else {
				setting["tone_guidelines"] = toneStr
			}
			settingModified = true

		// --- World state mutations ---

		case "world_location_add":
			locStr, ok := val.(string)
			if !ok || locStr == "" {
				continue
			}
			var locs []interface{}
			_ = json.Unmarshal([]byte(world.KnownLocationsJSON), &locs)
			// Avoid duplicates.
			for _, l := range locs {
				if ls, ok := l.(string); ok && strings.EqualFold(ls, locStr) {
					locStr = "" // already exists
					break
				}
			}
			if locStr != "" {
				locs = append(locs, locStr)
				if lb, err := json.Marshal(locs); err == nil {
					world.KnownLocationsJSON = string(lb)
					worldModified = true
				}
			}

		case "world_event_add":
			evStr, ok := val.(string)
			if !ok || evStr == "" {
				continue
			}
			var events []interface{}
			_ = json.Unmarshal([]byte(world.GlobalEventsJSON), &events)
			events = append(events, evStr)
			if eb, err := json.Marshal(events); err == nil {
				world.GlobalEventsJSON = string(eb)
				worldModified = true
			}

		case "world_faction_standing":
			fsMap, ok := toStringMap(val)
			if !ok {
				continue
			}
			factionName, _ := fsMap["faction"].(string)
			if factionName == "" {
				continue
			}
			standing := int(toFloat(fsMap["standing"]))
			var standings map[string]interface{}
			if err := json.Unmarshal([]byte(world.FactionStandingsJSON), &standings); err != nil {
				standings = map[string]interface{}{}
			}
			standings[factionName] = standing
			if fsb, err := json.Marshal(standings); err == nil {
				world.FactionStandingsJSON = string(fsb)
				worldModified = true
			}

		// --- NPC desire updates ---

		case "npc_desires":
			if db == nil {
				continue
			}
			desireMap, ok := toStringMap(val)
			if !ok {
				continue
			}
			npcName, _ := desireMap["name"].(string)
			desire, _ := desireMap["desire"].(string)
			if npcName == "" || desire == "" {
				continue
			}
			npc, err := db.GetNPCByName(story.ID, npcName)
			if err != nil || npc == nil {
				continue
			}
			if npc.Desires != "" {
				npc.Desires = npc.Desires + "; " + desire
			} else {
				npc.Desires = desire
			}
			_ = db.UpdateNPC(npc)
		}
	}

	// Persist story setting changes.
	if settingModified {
		newSettingJSON, err := json.Marshal(setting)
		if err == nil {
			story.SettingJSON = string(newSettingJSON)
			if db != nil {
				_ = db.UpdateStorySetting(story.ID, story.SettingJSON)
			}
		}
	}

	// Persist world state changes.
	if worldModified {
		world.UpdatedAt = time.Now()
		if db != nil {
			_ = db.UpdateWorldState(world)
		}
	}

	// Embed lore additions into RAG for long-term memory.
	if ragPipeline != nil && (settingModified || worldModified) {
		loreText := buildLoreChunk(changes)
		if loreText != "" {
			go func() {
				bgCtx := context.Background()
				_ = ragPipeline.StoreChunk(bgCtx, story.ID, loreText, "narrator", world.CurrentTurn, world.CurrentTurn)
			}()
		}
	}

	return nil
}

// toStringOrSlice converts a value to a []string.
// Handles both a single string and a []interface{} of strings.
func toStringOrSlice(val interface{}) []string {
	if s, ok := val.(string); ok && s != "" {
		return []string{s}
	}
	return toStringSlice(val)
}

// appendUnique appends items to a slice, skipping case-insensitive duplicates.
func appendUnique(existing []string, items ...string) []string {
	set := make(map[string]bool, len(existing))
	for _, e := range existing {
		set[strings.ToLower(e)] = true
	}
	for _, item := range items {
		if item != "" && !set[strings.ToLower(item)] {
			existing = append(existing, item)
			set[strings.ToLower(item)] = true
		}
	}
	return existing
}

// buildLoreChunk creates a text representation of narrator changes for RAG embedding.
func buildLoreChunk(changes map[string]interface{}) string {
	var parts []string
	for k, v := range changes {
		switch k {
		case "setting_factions_add", "setting_cultures_add", "setting_dangers_add", "setting_rules_add":
			items := toStringOrSlice(v)
			label := strings.TrimSuffix(strings.TrimPrefix(k, "setting_"), "_add")
			for _, item := range items {
				parts = append(parts, fmt.Sprintf("[%s] %s", label, item))
			}
		case "setting_tone_add":
			if s, ok := v.(string); ok && s != "" {
				parts = append(parts, fmt.Sprintf("[tone] %s", s))
			}
		case "world_location_add":
			if s, ok := v.(string); ok && s != "" {
				parts = append(parts, fmt.Sprintf("[location] %s", s))
			}
		case "world_event_add":
			if s, ok := v.(string); ok && s != "" {
				parts = append(parts, fmt.Sprintf("[event] %s", s))
			}
		case "npc_desires":
			if m, ok := toStringMap(v); ok {
				name, _ := m["name"].(string)
				desire, _ := m["desire"].(string)
				if name != "" && desire != "" {
					parts = append(parts, fmt.Sprintf("[npc desire] %s: %s", name, desire))
				}
			}
		}
	}
	return strings.Join(parts, "\n")
}
