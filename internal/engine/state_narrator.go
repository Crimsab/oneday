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

	for _, operation := range orderedStateChangeOperations(changes, narratorStateChangeOrder) {
		key, val := operation.Key, operation.Value
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
			storyID := story.ID
			turn := world.CurrentTurn
			submitRAGTask(storyID, ragTaskKey("state-lore", loreText), func(taskCtx context.Context) error {
				return ragPipeline.StoreChunk(taskCtx, storyID, loreText, "narrator", turn, turn)
			})
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
