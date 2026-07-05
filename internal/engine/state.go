package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
)

// StateChange records a single change that was applied to character or world state.
type StateChange struct {
	Target string // "character" or "world"
	Field  string // e.g. "vitals.hp.current", "location", "attributes.str"
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
	var npcs npcStateStore
	if db != nil {
		npcs = directNPCStore{db: db}
	}
	return applyStateChangesWithNPCStore(changes, char, world, db, npcs, storyID, currentTurn)
}

func applyStateChangesWithNPCStore(
	changes map[string]interface{},
	char *storage.Character,
	world *storage.WorldState,
	db *storage.DB,
	npcs npcStateStore,
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

	// Parse character inventory JSON into a separate mutable slice.
	// We keep inventory in char.InventoryJSON (not inside stats_json) so that
	// the full item objects (name, type, rarity, effects, description) survive.
	var invItems []interface{}
	if char.InventoryJSON != "" && char.InventoryJSON != "null" {
		parsedItems, err := parseInventoryItems(char.InventoryJSON)
		if err != nil {
			return nil, fmt.Errorf("parsing character inventory: %w", err)
		}
		invItems = parsedItems
	}
	if invItems == nil {
		invItems = []interface{}{}
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
				maxVal := toFloat(existing["max"])
				if maxVal < 0 {
					maxVal = 0
				}
				if _, ok := existing["max"]; ok {
					existing["max"] = maxVal
				}
				currentVal := toFloat(existing["current"])
				if currentVal < 0 {
					currentVal = 0
				}
				if maxVal > 0 && currentVal > maxVal {
					currentVal = maxVal
				}
				if _, ok := existing["current"]; ok {
					existing["current"] = currentVal
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
			// Accept both string items and full object items from AI.
			// String  → normalize to {"name": s, "type": "misc", "effects": []}
			// Object  → use as-is (name, type, rarity, effects, description)
			rawSlice, ok := val.([]interface{})
			if !ok {
				// Single item passed as non-slice — wrap it.
				rawSlice = []interface{}{val}
			}
			for _, rawItem := range rawSlice {
				var itemObj map[string]interface{}
				switch v := rawItem.(type) {
				case string:
					if v == "" {
						continue
					}
					itemObj = map[string]interface{}{
						"name":    v,
						"type":    "misc",
						"effects": []interface{}{},
					}
				case map[string]interface{}:
					if v["name"] == "" || v["name"] == nil {
						continue
					}
					itemObj = v
				default:
					continue
				}
				invItems = append(invItems, itemObj)
				applied = append(applied, StateChange{
					Target: "character",
					Field:  "inventory",
					Old:    nil,
					New:    itemObj,
				})
			}

		case "inventory_remove":
			// Remove by name — case-insensitive match against item.name or string item.
			removeNames := toStringSlice(val)
			if len(removeNames) == 0 {
				continue
			}
			removeSet := make(map[string]bool, len(removeNames))
			for _, n := range removeNames {
				removeSet[strings.ToLower(n)] = true
			}
			newItems := make([]interface{}, 0, len(invItems))
			for _, rawItem := range invItems {
				var itemName string
				switch v := rawItem.(type) {
				case string:
					itemName = v
				case map[string]interface{}:
					itemName, _ = v["name"].(string)
				}
				if removeSet[strings.ToLower(itemName)] {
					applied = append(applied, StateChange{
						Target: "character",
						Field:  "inventory",
						Old:    rawItem,
						New:    nil,
					})
				} else {
					newItems = append(newItems, rawItem)
				}
			}
			invItems = newItems

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
				pending := toSkillsMap(stats["pending_skill_discovery"])
				pendingSkill, _ := pending[skillName].(map[string]interface{})
				if pendingSkill == nil {
					pendingSkill = map[string]interface{}{"xp": 0, "evidence": 0}
				}
				pendingSkill["xp"] = int(toFloat(pendingSkill["xp"])) + xpGain
				pendingSkill["evidence"] = int(toFloat(pendingSkill["evidence"])) + 1
				pending[skillName] = pendingSkill
				stats["pending_skill_discovery"] = pending
				applied = append(applied, StateChange{
					Target:      "character",
					Field:       fmt.Sprintf("pending_skill_discovery.%s", skillName),
					Old:         nil,
					New:         pendingSkill,
					Description: fmt.Sprintf("Skill discovery progressed: %s", skillName),
				})
				continue
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
			if !ok || npcs == nil {
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
			existing, err := npcs.GetNPCByName(storyID, npcData.Name)
			if err != nil {
				continue
			}
			if existing != nil {
				// Update last_seen_turn and persist
				existing.LastSeenTurn = currentTurn
				_ = npcs.UpdateNPC(existing)
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
				if err := npcs.CreateNPC(npc); err != nil {
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
			if !ok || npcs == nil {
				continue
			}
			npcName, _ := dispMap["name"].(string)
			if npcName == "" {
				continue
			}
			npc, err := ensureNPCForStateChange(npcs, storyID, npcName, currentTurn)
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
			npc.Disposition = newDisp
			_ = npcs.UpdateNPC(npc)
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
			if !ok || npcs == nil {
				continue
			}
			npcName, _ := thoughtMap["name"].(string)
			thought, _ := thoughtMap["thought"].(string)
			if npcName == "" || thought == "" {
				continue
			}
			npc, err := ensureNPCForStateChange(npcs, storyID, npcName, currentTurn)
			if err != nil || npc == nil {
				continue
			}
			var thoughts []string
			_ = json.Unmarshal([]byte(npc.PrivateThoughts), &thoughts)
			thoughts = append(thoughts, thought)
			if tb, err := json.Marshal(thoughts); err == nil {
				npc.PrivateThoughts = string(tb)
			}
			_ = npcs.UpdateNPC(npc)
			applied = append(applied, StateChange{
				Target:      "world",
				Field:       fmt.Sprintf("npc.%s.private_thoughts", npcName),
				New:         thought,
				Description: fmt.Sprintf("%s had a new thought (private)", npcName),
			})

		case "npc_notes":
			noteMap, ok := toStringMap(val)
			if !ok || npcs == nil {
				continue
			}
			npcName, _ := noteMap["name"].(string)
			note, _ := noteMap["note"].(string)
			if npcName == "" || note == "" {
				continue
			}
			npc, err := ensureNPCForStateChange(npcs, storyID, npcName, currentTurn)
			if err != nil || npc == nil {
				continue
			}
			var notes []string
			_ = json.Unmarshal([]byte(npc.NotesOnProtagonist), &notes)
			notes = append(notes, note)
			if nb, err := json.Marshal(notes); err == nil {
				npc.NotesOnProtagonist = string(nb)
			}
			_ = npcs.UpdateNPC(npc)
			applied = append(applied, StateChange{
				Target:      "world",
				Field:       fmt.Sprintf("npc.%s.notes_on_protagonist", npcName),
				New:         note,
				Description: fmt.Sprintf("%s noted something about protagonist (private)", npcName),
			})

		// npc_desire_update and npc_desires both update an NPC's desires text.
		case "npc_desire_update", "npc_desires":
			desireMap, ok := toStringMap(val)
			if !ok || npcs == nil {
				continue
			}
			npcName, _ := desireMap["name"].(string)
			desire, _ := desireMap["desire"].(string)
			if npcName == "" || desire == "" {
				continue
			}
			npc, err := ensureNPCForStateChange(npcs, storyID, npcName, currentTurn)
			if err != nil || npc == nil {
				continue
			}
			if npc.Desires != "" {
				npc.Desires = npc.Desires + "; " + desire
			} else {
				npc.Desires = desire
			}
			_ = npcs.UpdateNPC(npc)
			applied = append(applied, StateChange{
				Target:      "world",
				Field:       fmt.Sprintf("npc.%s.desires", npcName),
				New:         desire,
				Description: fmt.Sprintf("%s's desires updated", npcName),
			})

		case "npc_relationship":
			if npcs == nil {
				continue
			}
			for _, relMap := range toObjectMaps(val) {
				npcName, _ := relMap["name"].(string)
				if npcName == "" {
					continue
				}
				npc, err := ensureNPCForStateChange(npcs, storyID, npcName, currentTurn)
				if err != nil || npc == nil {
					continue
				}

				axes := loadRelationshipAxes(npc)
				changed := false
				for _, axis := range []struct {
					name string
					ptr  *int
				}{
					{name: "trust", ptr: &axes.Trust},
					{name: "fear", ptr: &axes.Fear},
					{name: "debt", ptr: &axes.Debt},
					{name: "respect", ptr: &axes.Respect},
					{name: "intimacy", ptr: &axes.Intimacy},
				} {
					next, ok := applyRelationshipAxisUpdate(*axis.ptr, relMap[axis.name])
					if !ok || next == *axis.ptr {
						continue
					}
					oldVal := *axis.ptr
					*axis.ptr = next
					diff := next - oldVal
					sign := "+"
					if diff < 0 {
						sign = ""
					}
					applied = append(applied, StateChange{
						Target:      "world",
						Field:       fmt.Sprintf("relationship.%s.%s", npcName, axis.name),
						Old:         oldVal,
						New:         next,
						Description: fmt.Sprintf("%s %s: %s%d (now %d)", npcName, axis.name, sign, diff, next),
					})
					changed = true
				}
				if changed {
					storeRelationshipAxes(npc, axes)
					_ = npcs.UpdateNPC(npc)
				}
			}

		case "nemesis_resolution":
			if db == nil {
				continue
			}
			for _, resolutionMap := range toObjectMaps(val) {
				spec := parseNemesisResolutionSpec(resolutionMap)
				resolved, err := ApplyNemesisResolution(db, storyID, world, currentTurn, spec)
				if err != nil {
					continue
				}
				applied = append(applied, resolved...)
			}

		case "investigation_update":
			for _, updateMap := range toObjectMaps(val) {
				applied = append(applied, ensureNPCsFromInvestigationSuspects(npcs, storyID, updateMap, currentTurn)...)
				applied = append(applied, ApplyInvestigationUpdate(world, updateMap, currentTurn)...)
			}

		case "project_update":
			for _, updateMap := range toObjectMaps(val) {
				applied = append(applied, ApplyProjectUpdate(char, stats, &invItems, world, db, storyID, updateMap, currentTurn)...)
			}

		case "hook_add":
			hooks := loadStoryHooks(world)
			updated := false
			for _, hookMap := range toObjectMapsOrStrings(val, "title") {
				title, _ := hookMap["title"].(string)
				title = strings.TrimSpace(title)
				if title == "" {
					continue
				}
				hook := StoryHook{
					ID:          strings.TrimSpace(stringValue(hookMap["id"])),
					Kind:        strings.TrimSpace(stringValue(hookMap["kind"])),
					Title:       title,
					Detail:      strings.TrimSpace(stringValue(hookMap["detail"])),
					Status:      firstNonEmpty(strings.TrimSpace(stringValue(hookMap["status"])), "active"),
					NPCName:     strings.TrimSpace(stringValue(hookMap["npc"])),
					TimerTurns:  int(toFloat(hookMap["timer_turns"])),
					SourceTurn:  currentTurn,
					UpdatedTurn: currentTurn,
				}
				idx := findStoryHookIndex(hooks, hook.ID, hook.Title)
				if idx >= 0 {
					if hook.Kind != "" {
						hooks[idx].Kind = hook.Kind
					}
					if hook.Detail != "" {
						hooks[idx].Detail = hook.Detail
					}
					if hook.NPCName != "" {
						hooks[idx].NPCName = hook.NPCName
					}
					if hook.TimerTurns > 0 {
						hooks[idx].TimerTurns = hook.TimerTurns
					}
					hooks[idx].Status = hook.Status
					hooks[idx].UpdatedTurn = currentTurn
					applied = append(applied, StateChange{
						Target:      "world",
						Field:       fmt.Sprintf("hook.%s", hook.Title),
						New:         hook.Title,
						Description: fmt.Sprintf("Hook updated: %s", hook.Title),
					})
				} else {
					if hook.ID == "" {
						hook.ID = uuid.NewString()
					}
					hooks = append(hooks, hook)
					applied = append(applied, StateChange{
						Target:      "world",
						Field:       fmt.Sprintf("hook.%s", hook.Title),
						New:         hook.Title,
						Description: fmt.Sprintf("New hook: %s", hook.Title),
					})
				}
				updated = true
			}
			if updated {
				storeStoryHooks(world, hooks)
			}

		case "hook_update":
			hooks := loadStoryHooks(world)
			updated := false
			for _, hookMap := range toObjectMaps(val) {
				title := strings.TrimSpace(stringValue(hookMap["title"]))
				idx := findStoryHookIndex(hooks, strings.TrimSpace(stringValue(hookMap["id"])), title)
				if idx < 0 {
					continue
				}
				if detail := strings.TrimSpace(stringValue(hookMap["detail"])); detail != "" {
					hooks[idx].Detail = detail
				}
				if status := strings.TrimSpace(stringValue(hookMap["status"])); status != "" {
					hooks[idx].Status = status
				}
				if npcName := strings.TrimSpace(stringValue(hookMap["npc"])); npcName != "" {
					hooks[idx].NPCName = npcName
				}
				if timerTurns := int(toFloat(hookMap["timer_turns"])); timerTurns > 0 {
					hooks[idx].TimerTurns = timerTurns
				}
				hooks[idx].UpdatedTurn = currentTurn
				applied = append(applied, StateChange{
					Target:      "world",
					Field:       fmt.Sprintf("hook.%s", hooks[idx].Title),
					New:         hooks[idx].Title,
					Description: fmt.Sprintf("Hook progressed: %s", hooks[idx].Title),
				})
				updated = true
			}
			if updated {
				storeStoryHooks(world, hooks)
			}

		case "hook_resolve":
			hooks := loadStoryHooks(world)
			updated := false
			for _, hookMap := range toObjectMapsOrStrings(val, "title") {
				title := strings.TrimSpace(stringValue(hookMap["title"]))
				idx := findStoryHookIndex(hooks, strings.TrimSpace(stringValue(hookMap["id"])), title)
				if idx < 0 {
					continue
				}
				hooks[idx].Status = "resolved"
				if resolution := strings.TrimSpace(stringValue(hookMap["detail"])); resolution != "" {
					hooks[idx].Detail = resolution
				}
				hooks[idx].UpdatedTurn = currentTurn
				applied = append(applied, StateChange{
					Target:      "world",
					Field:       fmt.Sprintf("hook.%s", hooks[idx].Title),
					New:         hooks[idx].Title,
					Description: fmt.Sprintf("Hook resolved: %s", hooks[idx].Title),
				})
				updated = true
			}
			if updated {
				storeStoryHooks(world, hooks)
			}

		case "guide_update":
			guidance := loadPlayerGuidance(world)
			updated := false
			updates := make([]PlayerGuidance, 0, 1)
			for _, guideMap := range toObjectMaps(val) {
				title := strings.TrimSpace(stringValue(guideMap["title"]))
				id := strings.TrimSpace(stringValue(guideMap["id"]))
				kind := strings.TrimSpace(stringValue(guideMap["kind"]))
				idx := findPlayerGuidanceIndex(guidance, id, kind, title)
				if idx < 0 {
					continue
				}
				update := PlayerGuidance{
					ID:       id,
					Kind:     kind,
					Title:    guidance[idx].Title,
					Status:   strings.TrimSpace(stringValue(guideMap["status"])),
					Progress: strings.TrimSpace(stringValue(guideMap["progress"])),
					Detail:   strings.TrimSpace(stringValue(guideMap["detail"])),
				}
				updates = append(updates, update)

				desc := fmt.Sprintf("Guidance progressed: %s", guidance[idx].Title)
				if strings.EqualFold(update.Status, "fulfilled") {
					desc = fmt.Sprintf("Guidance fulfilled: %s", guidance[idx].Title)
				} else if strings.EqualFold(update.Status, "seeded") {
					desc = fmt.Sprintf("Guidance seeded: %s", guidance[idx].Title)
				}
				applied = append(applied, StateChange{
					Target:      "world",
					Field:       fmt.Sprintf("guidance.%s", guidance[idx].Title),
					New:         update.Status,
					Description: desc,
				})
				updated = true
			}
			if updated {
				storePlayerGuidance(world, updatePlayerGuidance(guidance, updates, currentTurn))
			}

		case "timeline_update":
			for _, updateMap := range toObjectMaps(val) {
				applied = append(applied, ApplyTimelineUpdate(world, updateMap, currentTurn)...)
			}

		case "world_reaction_add", "fail_forward":
			reactions := loadWorldReactions(world)
			updated := false
			defaultKind := "reaction"
			if key == "fail_forward" {
				defaultKind = "setback"
			}
			for _, reactionMap := range toObjectMapsOrStrings(val, "title") {
				title := strings.TrimSpace(stringValue(reactionMap["title"]))
				if title == "" {
					continue
				}
				reaction := WorldReaction{
					ID:          strings.TrimSpace(stringValue(reactionMap["id"])),
					Kind:        firstNonEmpty(strings.TrimSpace(stringValue(reactionMap["kind"])), defaultKind),
					Title:       title,
					Detail:      strings.TrimSpace(stringValue(reactionMap["detail"])),
					Status:      firstNonEmpty(strings.TrimSpace(stringValue(reactionMap["status"])), "active"),
					SourceTurn:  currentTurn,
					CreatedTurn: currentTurn,
				}
				idx := findWorldReactionIndex(reactions, reaction.ID, reaction.Title)
				if idx >= 0 {
					if reaction.Kind != "" {
						reactions[idx].Kind = reaction.Kind
					}
					if reaction.Detail != "" {
						reactions[idx].Detail = reaction.Detail
					}
					if reaction.Status != "" {
						reactions[idx].Status = reaction.Status
					}
					applied = append(applied, StateChange{
						Target:      "world",
						Field:       fmt.Sprintf("reaction.%s", reaction.Title),
						New:         reaction.Title,
						Description: fmt.Sprintf("World reacts: %s", reaction.Title),
					})
				} else {
					if reaction.ID == "" {
						reaction.ID = uuid.NewString()
					}
					reactions = append(reactions, reaction)
					applied = append(applied, StateChange{
						Target:      "world",
						Field:       fmt.Sprintf("reaction.%s", reaction.Title),
						New:         reaction.Title,
						Description: fmt.Sprintf("World reacts: %s", reaction.Title),
					})
				}
				updated = true
			}
			if updated {
				storeWorldReactions(world, reactions)
			}
			if key == "fail_forward" {
				applied = append(applied, applyFailForwardToFronts(world, val, currentTurn)...)
			}

		case "combat_start":
			// AI signals combat should begin. Records the event; the TUI/narrator
			// layer starts the actual CombatEngine when it sees NarrativeResponse.CombatStart.
			enemyMap, ok := toStringMap(val)
			if !ok {
				continue
			}
			applied = append(applied, StateChange{
				Target:      "world",
				Field:       "combat_start",
				New:         enemyMap,
				Description: "Combat initiated!",
			})

		case "crafting_start":
			// AI signals a crafting session should begin.
			applied = append(applied, StateChange{
				Target:      "world",
				Field:       "crafting_start",
				New:         val,
				Description: "Crafting session initiated",
			})

		default:
			if isNarratorManagedStateChangeKey(key) {
				continue
			}
			applied = append(applied, StateChange{
				Target:      "system",
				Field:       "unknown_state_change",
				New:         key,
				Description: fmt.Sprintf("Unknown state change ignored: %s", key),
			})
		}
	}

	// Marshal the modified stats back to character.
	statsBytes, err := json.Marshal(stats)
	if err != nil {
		return applied, fmt.Errorf("marshaling updated character stats: %w", err)
	}
	char.StatsJSON = string(statsBytes)
	if traits, err := json.Marshal(toStringSlice(stats["traits"])); err == nil {
		char.TraitsJSON = string(traits)
	}
	if skills, err := json.Marshal(toSkillsMap(stats["skills"])); err == nil {
		char.SkillsJSON = string(skills)
	}

	// Marshal the inventory items back to char.InventoryJSON (separate column).
	invBytes, err := json.Marshal(invItems)
	if err != nil {
		return applied, fmt.Errorf("marshaling updated inventory: %w", err)
	}
	char.InventoryJSON = string(invBytes)

	char.UpdatedAt = time.Now()
	world.UpdatedAt = time.Now()

	return applied, nil
}

func isNarratorManagedStateChangeKey(key string) bool {
	switch key {
	case "setting_factions_add",
		"setting_cultures_add",
		"setting_dangers_add",
		"setting_rules_add",
		"setting_tone_add",
		"world_location_add",
		"world_event_add",
		"world_faction_standing",
		"front_add",
		"front_advance",
		"front_reveal",
		"front_stall",
		"front_resolve",
		"front_pressure":
		return true
	default:
		return false
	}
}

func ensureNPCForStateChange(npcs npcStateStore, storyID, npcName string, currentTurn int) (*storage.NPC, error) {
	npcName = strings.TrimSpace(npcName)
	if npcs == nil || npcName == "" {
		return nil, nil
	}
	npc, err := npcs.GetNPCByName(storyID, npcName)
	if err != nil || npc != nil {
		return npc, err
	}
	npc, err = NPCToStorage(&NPCData{
		Name:       npcName,
		Role:       "person of interest",
		Appearance: "Unidentified figure first noticed in the current scene; preserve any concrete details implied by the name and story context.",
		Personality: NPCPersonality{
			Traits:      []string{"unknown"},
			SpeechStyle: "not established",
		},
		Desires:     []NPCDesire{},
		Disposition: 0,
		CanHelp:     true,
	}, storyID, currentTurn)
	if err != nil {
		return nil, err
	}
	if err := npcs.CreateNPC(npc); err != nil {
		return nil, err
	}
	return npc, nil
}

func ensureNPCsFromInvestigationSuspects(npcs npcStateStore, storyID string, updateMap map[string]interface{}, currentTurn int) []StateChange {
	if npcs == nil || len(updateMap) == 0 {
		return nil
	}
	var applied []StateChange
	for _, suspect := range toObjectMaps(updateMap["suspects"]) {
		action := normalizedEvidenceAction(suspect["action"], "add")
		if action == "discredit" || action == "collapse" {
			continue
		}
		name := strings.TrimSpace(stringValue(suspect["name"]))
		if !looksLikeTrackableNPCName(name) {
			continue
		}
		existing, err := npcs.GetNPCByName(storyID, name)
		if err != nil || existing != nil {
			continue
		}
		npc, err := ensureNPCForStateChange(npcs, storyID, name, currentTurn)
		if err != nil || npc == nil {
			continue
		}
		detail := strings.TrimSpace(stringValue(suspect["detail"]))
		if detail != "" {
			npc.Desires = "[]"
			npc.NotesOnProtagonist = marshalStringsOrDefault([]string{detail}, "[]")
			_ = npcs.UpdateNPC(npc)
		}
		applied = append(applied, StateChange{
			Target:      "world",
			Field:       "npc",
			New:         name,
			Description: fmt.Sprintf("New NPC encountered: %s (%s)", name, npc.Role),
		})
	}
	return applied
}

func looksLikeTrackableNPCName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 80 {
		return false
	}
	lower := strings.ToLower(name)
	blockedFragments := []string{
		"qualcuno", "sconosciuto", "unknown", "unidentified", "contatto sconosciuto",
		"rete", "fazione", "confraternita", "gilda", "culto", "guardie", "polizia",
		"famiglia", "gruppo", "gang", "societa", "società",
	}
	for _, fragment := range blockedFragments {
		if strings.Contains(lower, fragment) {
			return false
		}
	}
	return true
}

func marshalStringsOrDefault(items []string, fallback string) string {
	b, err := json.Marshal(items)
	if err != nil {
		return fallback
	}
	return string(b)
}

// toStringMap attempts to cast val to map[string]interface{}.
func toStringMap(val interface{}) (map[string]interface{}, bool) {
	m, ok := val.(map[string]interface{})
	return m, ok
}

func toObjectMaps(val interface{}) []map[string]interface{} {
	if val == nil {
		return nil
	}
	if item, ok := val.(map[string]interface{}); ok {
		return []map[string]interface{}{item}
	}
	raw, ok := val.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		if decoded, ok := item.(map[string]interface{}); ok {
			out = append(out, decoded)
		}
	}
	return out
}

func parseInventoryItems(raw string) ([]interface{}, error) {
	var decoded interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, err
	}
	return inventoryItemsFromValue(decoded), nil
}

func inventoryItemsFromValue(value interface{}) []interface{} {
	switch typed := value.(type) {
	case nil:
		return nil
	case []interface{}:
		return append([]interface{}{}, typed...)
	case map[string]interface{}:
		if _, hasName := typed["name"]; hasName {
			return []interface{}{typed}
		}
		if _, hasID := typed["id"]; hasID {
			return []interface{}{typed}
		}
		var out []interface{}
		for _, key := range []string{"backpack", "items", "equipped", "quest", "inventory"} {
			if nested, ok := typed[key]; ok {
				out = append(out, inventoryItemsFromValue(nested)...)
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []interface{}{typed}
	default:
		return nil
	}
}

func toObjectMapsOrStrings(val interface{}, stringKey string) []map[string]interface{} {
	switch v := val.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []map[string]interface{}{{stringKey: v}}
	case map[string]interface{}:
		return []map[string]interface{}{v}
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			switch typed := item.(type) {
			case string:
				if strings.TrimSpace(typed) == "" {
					continue
				}
				out = append(out, map[string]interface{}{stringKey: typed})
			case map[string]interface{}:
				out = append(out, typed)
			}
		}
		return out
	default:
		return nil
	}
}

func stringValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func applyRelationshipAxisUpdate(current int, raw interface{}) (int, bool) {
	switch v := raw.(type) {
	case nil:
		return current, false
	case float64, int, int64, json.Number:
		return clampRelationshipValue(int(toFloat(v))), true
	case map[string]interface{}:
		if changeVal, ok := v["change"]; ok {
			return clampRelationshipValue(current + int(toFloat(changeVal))), true
		}
		if valueVal, ok := v["value"]; ok {
			return clampRelationshipValue(int(toFloat(valueVal))), true
		}
	}
	return current, false
}

func findStoryHookIndex(hooks []StoryHook, id, title string) int {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	for i, hook := range hooks {
		if id != "" && strings.EqualFold(hook.ID, id) {
			return i
		}
		if title != "" && strings.EqualFold(hook.Title, title) {
			return i
		}
	}
	return -1
}

func findWorldReactionIndex(reactions []WorldReaction, id, title string) int {
	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	for i, reaction := range reactions {
		if id != "" && strings.EqualFold(reaction.ID, id) {
			return i
		}
		if title != "" && strings.EqualFold(reaction.Title, title) {
			return i
		}
	}
	return -1
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
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return f
		}
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
	frontsModified := false
	currentTurn := 0
	if world != nil {
		currentTurn = world.CurrentTurn
	}

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

		case "front_add":
			fronts := loadFronts(world)
			updated := false
			for _, frontMap := range toObjectMapsOrStrings(val, "title") {
				title := strings.TrimSpace(stringValue(frontMap["title"]))
				if title == "" {
					continue
				}
				front := Front{
					ID:                 strings.TrimSpace(stringValue(frontMap["id"])),
					Faction:            strings.TrimSpace(stringValue(frontMap["faction"])),
					Title:              title,
					PublicTitle:        strings.TrimSpace(stringValue(frontMap["public_title"])),
					Stakes:             strings.TrimSpace(stringValue(frontMap["stakes"])),
					PublicStakes:       strings.TrimSpace(stringValue(frontMap["public_stakes"])),
					Status:             firstNonEmpty(strings.TrimSpace(stringValue(frontMap["status"])), "active"),
					Visibility:         firstNonEmpty(strings.TrimSpace(stringValue(frontMap["visibility"])), "hidden"),
					Segments:           int(toFloat(frontMap["segments"])),
					Progress:           int(toFloat(frontMap["progress"])),
					LastAdvancedTurn:   currentTurn,
					NextEscalationTurn: int(toFloat(frontMap["next_escalation_turn"])),
				}
				if idx := findFrontIndex(fronts, front.ID, front.Title); idx >= 0 {
					if front.Faction != "" {
						fronts[idx].Faction = front.Faction
					}
					if front.PublicTitle != "" {
						fronts[idx].PublicTitle = front.PublicTitle
					}
					if front.Stakes != "" {
						fronts[idx].Stakes = front.Stakes
					}
					if front.PublicStakes != "" {
						fronts[idx].PublicStakes = front.PublicStakes
					}
					if front.Status != "" {
						fronts[idx].Status = front.Status
					}
					if front.Visibility != "" {
						fronts[idx].Visibility = front.Visibility
					}
					if front.Segments > 0 {
						fronts[idx].Segments = front.Segments
					}
					if front.Progress >= 0 {
						fronts[idx].Progress = front.Progress
					}
					if front.NextEscalationTurn > 0 {
						fronts[idx].NextEscalationTurn = front.NextEscalationTurn
					}
					fronts[idx].LastAdvancedTurn = currentTurn
				} else {
					fronts = append(fronts, front)
				}
				updated = true
			}
			if updated {
				storeFronts(world, fronts)
				worldModified = true
				frontsModified = true
			}

		case "front_advance":
			fronts := loadFronts(world)
			updated := false
			for _, frontMap := range toObjectMaps(val) {
				idx := findFrontIndex(fronts, strings.TrimSpace(stringValue(frontMap["id"])), strings.TrimSpace(stringValue(frontMap["title"])))
				if idx < 0 {
					continue
				}
				delta := int(toFloat(frontMap["amount"]))
				if delta == 0 {
					delta = 1
				}
				fronts[idx].Progress += delta
				fronts[idx].LastAdvancedTurn = currentTurn
				if fronts[idx].Progress > fronts[idx].Segments {
					fronts[idx].Progress = fronts[idx].Segments
				}
				if status := strings.TrimSpace(stringValue(frontMap["status"])); status != "" {
					fronts[idx].Status = status
				}
				updated = true
			}
			if updated {
				storeFronts(world, fronts)
				worldModified = true
				frontsModified = true
			}

		case "front_reveal":
			fronts := loadFronts(world)
			updated := false
			for _, frontMap := range toObjectMaps(val) {
				idx := findFrontIndex(fronts, strings.TrimSpace(stringValue(frontMap["id"])), strings.TrimSpace(stringValue(frontMap["title"])))
				if idx < 0 {
					continue
				}
				fronts[idx].Visibility = firstNonEmpty(strings.TrimSpace(stringValue(frontMap["visibility"])), "known")
				if publicTitle := strings.TrimSpace(stringValue(frontMap["public_title"])); publicTitle != "" {
					fronts[idx].PublicTitle = publicTitle
				}
				if publicStakes := strings.TrimSpace(stringValue(frontMap["public_stakes"])); publicStakes != "" {
					fronts[idx].PublicStakes = publicStakes
				}
				updated = true
			}
			if updated {
				storeFronts(world, fronts)
				worldModified = true
				frontsModified = true
			}

		case "front_stall":
			fronts := loadFronts(world)
			updated := false
			for _, frontMap := range toObjectMaps(val) {
				idx := findFrontIndex(fronts, strings.TrimSpace(stringValue(frontMap["id"])), strings.TrimSpace(stringValue(frontMap["title"])))
				if idx < 0 {
					continue
				}
				fronts[idx].Status = "stalled"
				if nextTurn := int(toFloat(frontMap["next_escalation_turn"])); nextTurn > 0 {
					fronts[idx].NextEscalationTurn = nextTurn
				}
				updated = true
			}
			if updated {
				storeFronts(world, fronts)
				worldModified = true
				frontsModified = true
			}

		case "front_resolve":
			fronts := loadFronts(world)
			updated := false
			for _, frontMap := range toObjectMaps(val) {
				idx := findFrontIndex(fronts, strings.TrimSpace(stringValue(frontMap["id"])), strings.TrimSpace(stringValue(frontMap["title"])))
				if idx < 0 {
					continue
				}
				fronts[idx].Status = firstNonEmpty(strings.TrimSpace(stringValue(frontMap["status"])), "resolved")
				fronts[idx].Resolution = firstNonEmpty(strings.TrimSpace(stringValue(frontMap["resolution"])), strings.TrimSpace(stringValue(frontMap["detail"])))
				fronts[idx].Progress = fronts[idx].Segments
				updated = true
			}
			if updated {
				storeFronts(world, fronts)
				worldModified = true
				frontsModified = true
			}

		case "front_pressure":
			fronts := loadFronts(world)
			updated := false
			for _, frontMap := range toObjectMaps(val) {
				idx := findFrontIndex(fronts, strings.TrimSpace(stringValue(frontMap["id"])), strings.TrimSpace(stringValue(frontMap["title"])))
				if idx < 0 {
					continue
				}
				region := strings.TrimSpace(stringValue(frontMap["region"]))
				kind := strings.TrimSpace(stringValue(frontMap["kind"]))
				if region == "" || kind == "" {
					continue
				}
				level := int(toFloat(frontMap["value"]))
				if changeVal, ok := frontMap["change"]; ok {
					level = -1
					for i := range fronts[idx].Pressures {
						if strings.EqualFold(fronts[idx].Pressures[i].Region, region) && strings.EqualFold(fronts[idx].Pressures[i].Kind, kind) {
							level = fronts[idx].Pressures[i].Level + int(toFloat(changeVal))
							break
						}
					}
					if level < 0 {
						level = int(toFloat(changeVal))
					}
				}
				pressure := FrontPressure{
					Region:      region,
					Kind:        kind,
					Level:       level,
					Detail:      strings.TrimSpace(stringValue(frontMap["detail"])),
					UpdatedTurn: currentTurn,
				}
				replaced := false
				for i := range fronts[idx].Pressures {
					if strings.EqualFold(fronts[idx].Pressures[i].Region, pressure.Region) && strings.EqualFold(fronts[idx].Pressures[i].Kind, pressure.Kind) {
						fronts[idx].Pressures[i] = pressure
						replaced = true
						break
					}
				}
				if !replaced {
					fronts[idx].Pressures = append(fronts[idx].Pressures, pressure)
				}
				updated = true
			}
			if updated {
				storeFronts(world, fronts)
				worldModified = true
				frontsModified = true
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

	if frontsModified {
		syncKnownFrontContinuity(world, currentTurn)
		worldModified = true
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
