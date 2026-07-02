package engine

type SceneDeltaKind string

const (
	SceneDeltaInformation  SceneDeltaKind = "information"
	SceneDeltaCost         SceneDeltaKind = "cost"
	SceneDeltaRelationship SceneDeltaKind = "relationship"
	SceneDeltaPosition     SceneDeltaKind = "position"
	SceneDeltaResource     SceneDeltaKind = "resource"
	SceneDeltaClock        SceneDeltaKind = "clock"
	SceneDeltaThreat       SceneDeltaKind = "threat"
	SceneDeltaExit         SceneDeltaKind = "exit"
)

// SceneContract is the durable shape the engine can use to prevent a scene
// from circling the same beat without a validated delta.
type SceneContract struct {
	ID             string           `json:"id"`
	Goal           string           `json:"goal"`
	StartedTurn    int              `json:"started_turn"`
	MaxTurns       int              `json:"max_turns"`
	RequiredDeltas []SceneDeltaKind `json:"required_deltas,omitempty"`
	ExitConditions []string         `json:"exit_conditions,omitempty"`
	PressureClock  int              `json:"pressure_clock,omitempty"`
}

func narrativeHasValidatedSceneDelta(narrative *NarrativeResponse) bool {
	if narrative == nil {
		return false
	}
	if len(narrative.EventCallouts) > 0 ||
		len(narrative.OpenHooks) > 0 ||
		len(narrative.WorldReactions) > 0 ||
		len(narrative.Challenges) > 0 ||
		narrative.SocialDuel != nil ||
		narrative.CombatStart != nil ||
		narrative.ChapterEnd {
		return true
	}
	for key := range narrative.StateChanges {
		if stateChangeKeyCanMoveScene(key) {
			return true
		}
	}
	return false
}

func stateChangeKeyCanMoveScene(key string) bool {
	switch key {
	case "vitals", "attributes", "secondary", "location", "currency",
		"inventory_add", "inventory_remove", "trait_add", "title_add",
		"skill_learn", "skill_xp", "new_npc", "npc_disposition",
		"npc_relationship", "nemesis_resolution", "investigation_update",
		"project_update", "hook_add", "hook_update", "hook_resolve",
		"guide_update", "timeline_update", "world_reaction_add",
		"fail_forward", "combat_start", "crafting_start",
		"setting_factions_add", "setting_cultures_add", "setting_dangers_add",
		"setting_rules_add", "setting_tone_add", "world_location_add",
		"world_event_add", "world_faction_standing", "front_add",
		"front_advance", "front_reveal", "front_stall", "front_resolve",
		"front_pressure", "npc_desires":
		return true
	default:
		return false
	}
}
