package engine

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

type NemesisResolutionOutcome string

const (
	NemesisResolutionCapture     NemesisResolutionOutcome = "capture"
	NemesisResolutionTruce       NemesisResolutionOutcome = "truce"
	NemesisResolutionAlliance    NemesisResolutionOutcome = "alliance"
	NemesisResolutionExile       NemesisResolutionOutcome = "exile"
	NemesisResolutionSuccession  NemesisResolutionOutcome = "succession"
	NemesisResolutionDeath       NemesisResolutionOutcome = "death"
	NemesisResolutionHumiliation NemesisResolutionOutcome = "humiliation"
)

type NemesisResolutionSpec struct {
	Name           string
	Outcome        NemesisResolutionOutcome
	Detail         string
	FrontID        string
	FrontTitle     string
	Successor      string
	HookTitle      string
	ReactionTitle  string
	ReactionDetail string
}

func parseNemesisResolutionSpec(raw map[string]interface{}) NemesisResolutionSpec {
	return NemesisResolutionSpec{
		Name:           strings.TrimSpace(stringValue(raw["name"])),
		Outcome:        NemesisResolutionOutcome(strings.ToLower(strings.TrimSpace(stringValue(raw["outcome"])))),
		Detail:         strings.TrimSpace(stringValue(raw["detail"])),
		FrontID:        strings.TrimSpace(stringValue(raw["front_id"])),
		FrontTitle:     strings.TrimSpace(stringValue(raw["front_title"])),
		Successor:      strings.TrimSpace(stringValue(raw["successor"])),
		HookTitle:      strings.TrimSpace(stringValue(raw["hook_title"])),
		ReactionTitle:  strings.TrimSpace(stringValue(raw["reaction_title"])),
		ReactionDetail: strings.TrimSpace(stringValue(raw["reaction_detail"])),
	}
}

func (s NemesisResolutionSpec) valid() bool {
	if s.Name == "" {
		return false
	}
	switch s.Outcome {
	case NemesisResolutionCapture, NemesisResolutionTruce, NemesisResolutionAlliance,
		NemesisResolutionExile, NemesisResolutionSuccession, NemesisResolutionDeath,
		NemesisResolutionHumiliation:
		return true
	default:
		return false
	}
}

func ApplyNemesisResolution(db *storage.DB, storyID string, world *storage.WorldState, currentTurn int, spec NemesisResolutionSpec) ([]StateChange, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if !spec.valid() {
		return nil, fmt.Errorf("invalid nemesis resolution")
	}

	npc, err := db.GetNPCByName(storyID, spec.Name)
	if err != nil {
		return nil, err
	}
	if npc == nil {
		return nil, fmt.Errorf("npc %q not found", spec.Name)
	}

	profile := loadNemesisProfile(npc)
	if profile == nil || (profile.Status != NemesisStatusActive && profile.Status != NemesisStatusRival) {
		return nil, fmt.Errorf("npc %q is not an active rival or nemesis", spec.Name)
	}

	changes := []StateChange{}
	applyNemesisResolutionToNPC(npc, profile, spec, currentTurn)
	if err := db.UpdateNPC(npc); err != nil {
		return nil, err
	}
	changes = append(changes, StateChange{
		Target:      "npc",
		Field:       fmt.Sprintf("nemesis.%s", npc.Name),
		New:         string(spec.Outcome),
		Description: fmt.Sprintf("Nemesis arc changes: %s -> %s", npc.Name, spec.Outcome),
	})

	if world != nil {
		reaction := nemesisResolutionReaction(spec, currentTurn)
		reactions := loadWorldReactions(world)
		if idx := findWorldReactionIndex(reactions, reaction.ID, reaction.Title); idx >= 0 {
			reactions[idx] = reaction
		} else {
			reactions = append(reactions, reaction)
		}
		storeWorldReactions(world, reactions)
		changes = append(changes, StateChange{
			Target:      "world",
			Field:       fmt.Sprintf("reaction.%s", reaction.Title),
			New:         reaction.Title,
			Description: fmt.Sprintf("World reacts: %s", reaction.Title),
		})

		if hook := nemesisResolutionHook(spec, currentTurn); hook != nil {
			hooks := loadStoryHooks(world)
			if idx := findStoryHookIndex(hooks, hook.ID, hook.Title); idx >= 0 {
				hooks[idx] = *hook
			} else {
				hooks = append(hooks, *hook)
			}
			storeStoryHooks(world, hooks)
			changes = append(changes, StateChange{
				Target:      "world",
				Field:       fmt.Sprintf("hook.%s", hook.Title),
				New:         hook.Title,
				Description: fmt.Sprintf("New hook: %s", hook.Title),
			})
		}

		if frontChange := applyNemesisResolutionToFronts(world, npc, profile, spec, currentTurn); frontChange != nil {
			changes = append(changes, *frontChange)
		}
	}

	return changes, nil
}

func applyNemesisResolutionToNPC(npc *storage.NPC, profile *NemesisProfile, spec NemesisResolutionSpec, currentTurn int) {
	if npc == nil || profile == nil {
		return
	}

	profile.Status = NemesisStatusResolved
	profile.LastSeenTurn = maxInt(profile.LastSeenTurn, currentTurn)
	profile.Vow = ""
	profile.LastOutcome = firstNonEmpty(spec.Detail, nemesisResolutionDefaultDetail(spec))
	profile.ThreatPosture = nemesisResolutionPosture(spec.Outcome)
	profile.EventHistory = append(profile.EventHistory, NemesisEvent{
		Kind:    "resolution",
		Outcome: string(spec.Outcome),
		Detail:  profile.LastOutcome,
		Turn:    currentTurn,
		Impact:  3,
	})
	if len(profile.EventHistory) > 8 {
		profile.EventHistory = profile.EventHistory[len(profile.EventHistory)-8:]
	}

	axes := loadRelationshipAxes(npc)
	switch spec.Outcome {
	case NemesisResolutionCapture:
		axes.Fear = clampRelationshipValue(axes.Fear + 8)
		axes.Respect = clampRelationshipValue(axes.Respect + 3)
		npc.Disposition = clampRange(npc.Disposition-10, -100, 100)
	case NemesisResolutionTruce:
		axes.Trust = clampRelationshipValue(axes.Trust + 4)
		axes.Respect = clampRelationshipValue(axes.Respect + 2)
		npc.Disposition = clampRange(npc.Disposition+6, -100, 100)
	case NemesisResolutionAlliance:
		axes.Trust = clampRelationshipValue(axes.Trust + 10)
		axes.Debt = clampRelationshipValue(axes.Debt + 4)
		axes.Respect = clampRelationshipValue(axes.Respect + 5)
		npc.Disposition = clampRange(npc.Disposition+18, -100, 100)
	case NemesisResolutionExile:
		axes.Fear = clampRelationshipValue(axes.Fear + 4)
		npc.Disposition = clampRange(npc.Disposition-6, -100, 100)
	case NemesisResolutionSuccession:
		axes.Respect = clampRelationshipValue(axes.Respect + 1)
	case NemesisResolutionDeath:
		npc.IsAlive = false
	case NemesisResolutionHumiliation:
		axes.Fear = clampRelationshipValue(axes.Fear + 8)
		axes.Respect = clampRelationshipValue(axes.Respect - 4)
		npc.Disposition = clampRange(npc.Disposition-18, -100, 100)
	}
	storeRelationshipAxes(npc, axes)
	storeNemesisProfile(npc, profile)
}

func nemesisResolutionReaction(spec NemesisResolutionSpec, currentTurn int) WorldReaction {
	title := firstNonEmpty(spec.ReactionTitle, nemesisResolutionDefaultReactionTitle(spec))
	detail := firstNonEmpty(spec.ReactionDetail, spec.Detail, nemesisResolutionDefaultDetail(spec))
	return WorldReaction{
		ID:          "nemesis-resolution:" + slugKey(spec.Name) + ":" + string(spec.Outcome),
		Kind:        "fallout",
		Title:       title,
		Detail:      detail,
		Status:      "active",
		SourceTurn:  currentTurn,
		CreatedTurn: currentTurn,
	}
}

func nemesisResolutionHook(spec NemesisResolutionSpec, currentTurn int) *StoryHook {
	title := strings.TrimSpace(spec.HookTitle)
	detail := firstNonEmpty(spec.Detail, nemesisResolutionDefaultDetail(spec))

	switch spec.Outcome {
	case NemesisResolutionCapture:
		if title == "" {
			title = fmt.Sprintf("Holding %s", spec.Name)
		}
		return &StoryHook{
			ID:          "nemesis-resolution:hook:" + slugKey(spec.Name) + ":capture",
			Kind:        "debt",
			Title:       title,
			Detail:      firstNonEmpty(detail, fmt.Sprintf("%s is contained for now, but that will not stay simple.", spec.Name)),
			Status:      "active",
			NPCName:     spec.Name,
			SourceTurn:  currentTurn,
			UpdatedTurn: currentTurn,
		}
	case NemesisResolutionTruce:
		if title == "" {
			title = fmt.Sprintf("Terms of the truce with %s", spec.Name)
		}
		return &StoryHook{
			ID:          "nemesis-resolution:hook:" + slugKey(spec.Name) + ":truce",
			Kind:        "promise",
			Title:       title,
			Detail:      firstNonEmpty(detail, fmt.Sprintf("The peace with %s will hold only if both sides honor it.", spec.Name)),
			Status:      "cooling",
			NPCName:     spec.Name,
			SourceTurn:  currentTurn,
			UpdatedTurn: currentTurn,
		}
	case NemesisResolutionAlliance:
		if title == "" {
			title = fmt.Sprintf("What %s wants from the alliance", spec.Name)
		}
		return &StoryHook{
			ID:          "nemesis-resolution:hook:" + slugKey(spec.Name) + ":alliance",
			Kind:        "goal",
			Title:       title,
			Detail:      firstNonEmpty(detail, fmt.Sprintf("%s is now aligned with you, but alliances always come with terms.", spec.Name)),
			Status:      "active",
			NPCName:     spec.Name,
			SourceTurn:  currentTurn,
			UpdatedTurn: currentTurn,
		}
	case NemesisResolutionSuccession:
		if title == "" {
			title = firstNonEmpty(spec.Successor, fmt.Sprintf("A successor rises after %s", spec.Name))
		}
		return &StoryHook{
			ID:          "nemesis-resolution:hook:" + slugKey(spec.Name) + ":succession",
			Kind:        "mystery",
			Title:       title,
			Detail:      firstNonEmpty(detail, fmt.Sprintf("Someone is picking up %s's unfinished war.", spec.Name)),
			Status:      "active",
			NPCName:     firstNonEmpty(spec.Successor, spec.Name),
			SourceTurn:  currentTurn,
			UpdatedTurn: currentTurn,
		}
	default:
		return nil
	}
}

func applyNemesisResolutionToFronts(world *storage.WorldState, npc *storage.NPC, profile *NemesisProfile, spec NemesisResolutionSpec, currentTurn int) *StateChange {
	if world == nil {
		return nil
	}

	fronts := loadFronts(world)
	if len(fronts) == 0 {
		return nil
	}
	idx := findFrontIndex(fronts, spec.FrontID, spec.FrontTitle)
	if idx < 0 {
		for i := range fronts {
			if nemesisProfileTouchesFront(profile, fronts[i]) {
				idx = i
				break
			}
			if npc != nil && (strings.Contains(strings.ToLower(fronts[i].Title), strings.ToLower(npc.Name)) ||
				strings.Contains(strings.ToLower(fronts[i].PublicTitle), strings.ToLower(npc.Name))) {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return nil
	}

	front := &fronts[idx]
	switch spec.Outcome {
	case NemesisResolutionAlliance, NemesisResolutionDeath:
		front.Status = "resolved"
		front.Progress = maxInt(front.Progress, front.Segments)
		front.Resolution = firstNonEmpty(spec.Detail, fmt.Sprintf("%s's arc closes this front for now.", spec.Name))
	case NemesisResolutionSuccession:
		front.Status = "active"
		front.Progress = minInt(front.Segments, front.Progress+1)
		front.NextEscalationTurn = maxInt(front.NextEscalationTurn, currentTurn+2)
		front.Resolution = ""
	default:
		front.Status = "stalled"
		front.NextEscalationTurn = currentTurn + 3
	}
	front.LastAdvancedTurn = currentTurn
	storeFronts(world, fronts)

	desc := fmt.Sprintf("Front shifts after %s's %s", spec.Name, spec.Outcome)
	if front.Status == "resolved" {
		desc = fmt.Sprintf("Front resolved after %s's %s", spec.Name, spec.Outcome)
	}
	return &StateChange{
		Target:      "world",
		Field:       fmt.Sprintf("front.%s", frontDisplayTitle(*front)),
		New:         front.Status,
		Description: desc,
	}
}

func nemesisResolutionPosture(outcome NemesisResolutionOutcome) string {
	switch outcome {
	case NemesisResolutionCapture:
		return "contained"
	case NemesisResolutionTruce:
		return "cold_truce"
	case NemesisResolutionAlliance:
		return "allied"
	case NemesisResolutionExile:
		return "exiled"
	case NemesisResolutionSuccession:
		return "succession"
	case NemesisResolutionDeath:
		return "dead"
	case NemesisResolutionHumiliation:
		return "broken"
	default:
		return "resolved"
	}
}

func nemesisResolutionDefaultReactionTitle(spec NemesisResolutionSpec) string {
	switch spec.Outcome {
	case NemesisResolutionCapture:
		return fmt.Sprintf("%s is in chains", spec.Name)
	case NemesisResolutionTruce:
		return fmt.Sprintf("A hard truce settles around %s", spec.Name)
	case NemesisResolutionAlliance:
		return fmt.Sprintf("%s turns from rival to ally", spec.Name)
	case NemesisResolutionExile:
		return fmt.Sprintf("%s is driven into exile", spec.Name)
	case NemesisResolutionSuccession:
		return firstNonEmpty(spec.Successor, fmt.Sprintf("A successor moves after %s", spec.Name))
	case NemesisResolutionDeath:
		return fmt.Sprintf("%s's war ends in blood", spec.Name)
	case NemesisResolutionHumiliation:
		return fmt.Sprintf("%s cannot hide the humiliation", spec.Name)
	default:
		return fmt.Sprintf("%s's arc changes", spec.Name)
	}
}

func nemesisResolutionDefaultDetail(spec NemesisResolutionSpec) string {
	switch spec.Outcome {
	case NemesisResolutionCapture:
		return fmt.Sprintf("%s is contained, but holding them will create new pressure.", spec.Name)
	case NemesisResolutionTruce:
		return fmt.Sprintf("The feud with %s cools into a tense truce.", spec.Name)
	case NemesisResolutionAlliance:
		return fmt.Sprintf("%s now stands on your side, at least for this chapter.", spec.Name)
	case NemesisResolutionExile:
		return fmt.Sprintf("%s is forced out of the region, leaving a dangerous vacuum behind.", spec.Name)
	case NemesisResolutionSuccession:
		if spec.Successor != "" {
			return fmt.Sprintf("%s's grudge passes to %s.", spec.Name, spec.Successor)
		}
		return fmt.Sprintf("Someone inherits %s's unfinished war.", spec.Name)
	case NemesisResolutionDeath:
		return fmt.Sprintf("%s dies and their campaign collapses into fallout.", spec.Name)
	case NemesisResolutionHumiliation:
		return fmt.Sprintf("%s loses standing in full view of the people who mattered.", spec.Name)
	default:
		return ""
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
