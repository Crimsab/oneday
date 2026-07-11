package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/storage"
)

// SocialDuelAftermath captures the canonical traces left by a resolved or significant duel exchange.
type SocialDuelAftermath struct {
	Summary            []string         `json:"summary,omitempty"`
	DispositionDelta   int              `json:"disposition_delta,omitempty"`
	RelationshipDelta  RelationshipAxes `json:"relationship_delta,omitempty"`
	ReactionTitle      string           `json:"reaction_title,omitempty"`
	FrontPressureTitle string           `json:"front_pressure_title,omitempty"`
	NPCNote            string           `json:"npc_note,omitempty"`
}

func ApplySocialDuelAftermath(db *storage.DB, world *storage.WorldState, npc *storage.NPC, state *SocialDuelState, result *SocialRoundResult, cue *SocialDuelCue, currentTurn int) *SocialDuelAftermath {
	if state == nil || result == nil {
		return nil
	}

	aftermath := &SocialDuelAftermath{}
	changedNPC := false
	changedWorld := false

	if npc != nil {
		beforeStatus := NemesisStatus("")
		if existing := loadNemesisProfile(npc); existing != nil {
			beforeStatus = existing.Status
		}

		dispDelta, relDelta := socialDuelRelationshipDelta(state, result)
		if dispDelta != 0 || !relationshipAxesZero(relDelta) {
			axes := loadRelationshipAxes(npc)
			axes.Trust = clampRelationshipValue(axes.Trust + relDelta.Trust)
			axes.Fear = clampRelationshipValue(axes.Fear + relDelta.Fear)
			axes.Debt = clampRelationshipValue(axes.Debt + relDelta.Debt)
			axes.Respect = clampRelationshipValue(axes.Respect + relDelta.Respect)
			axes.Intimacy = clampRelationshipValue(axes.Intimacy + relDelta.Intimacy)
			storeRelationshipAxes(npc, axes)
			npc.Disposition = clampRange(npc.Disposition+dispDelta, -100, 100)
			aftermath.DispositionDelta = dispDelta
			aftermath.RelationshipDelta = relDelta
			aftermath.Summary = append(aftermath.Summary, socialDuelRelationshipSummary(npc.Name, dispDelta, relDelta))
			changedNPC = true
		}

		npc.LastSeenTurn = maxInt(npc.LastSeenTurn, currentTurn)
		note := socialDuelNPCNote(npc, state, result, cue, currentTurn)
		if note != "" {
			npc.NotesOnProtagonist = appendJSONStringUnique(npc.NotesOnProtagonist, note)
			aftermath.NPCNote = note
			aftermath.Summary = append(aftermath.Summary, fmt.Sprintf("%s updates their dossier on you.", npc.Name))
			changedNPC = true
		}

		thought := socialDuelPrivateThought(npc, state, result)
		if thought != "" {
			npc.PrivateThoughts = appendJSONStringUnique(npc.PrivateThoughts, thought)
			changedNPC = true
		}

		if nemesis := RecordNemesisEvent(npc, socialDuelNemesisEvent(state, result, cue, currentTurn)); nemesis != nil {
			changedNPC = true
			switch {
			case beforeStatus != NemesisStatusActive && nemesis.Status == NemesisStatusActive:
				aftermath.Summary = append(aftermath.Summary, fmt.Sprintf("%s crosses the line from rival to nemesis.", npc.Name))
			case nemesis.Status == NemesisStatusActive:
				aftermath.Summary = append(aftermath.Summary, fmt.Sprintf("%s's rivalry escalates to tier %d.", npc.Name, nemesis.EscalationTier))
			}
		}
	}

	if reaction := socialDuelWorldReaction(state, result, cue, currentTurn); reaction != nil && world != nil {
		reactions := loadWorldReactions(world)
		idx := findWorldReactionIndex(reactions, reaction.ID, reaction.Title)
		if idx >= 0 {
			reactions[idx] = *reaction
		} else {
			reactions = append(reactions, *reaction)
		}
		storeWorldReactions(world, reactions)
		aftermath.ReactionTitle = reaction.Title
		aftermath.Summary = append(aftermath.Summary, fmt.Sprintf("World reaction: %s", reaction.Title))
		changedWorld = true
	}

	if world != nil {
		if frontTitle := applySocialDuelFrontPressure(world, npc, state, result, currentTurn); frontTitle != "" {
			aftermath.FrontPressureTitle = frontTitle
			aftermath.Summary = append(aftermath.Summary, frontTitle)
			changedWorld = true
		}
	}

	if db != nil && (changedNPC || changedWorld) {
		storyID := ""
		if world != nil {
			storyID = world.StoryID
		} else if npc != nil {
			storyID = npc.StoryID
		}
		_ = db.WithTx(func(tx *sql.Tx) error {
			if changedNPC && npc != nil {
				if err := db.UpdateNPCTx(tx, npc); err != nil {
					return err
				}
			}
			if changedWorld && world != nil {
				if err := db.UpdateWorldStateTx(tx, world); err != nil {
					return err
				}
			}
			if storyID == "" {
				return nil
			}
			_, err := db.BumpStoryRevisionTx(tx, storyID)
			return err
		})
	}

	if len(aftermath.Summary) == 0 && aftermath.NPCNote == "" {
		return nil
	}
	return aftermath
}

func socialDuelRelationshipDelta(state *SocialDuelState, result *SocialRoundResult) (int, RelationshipAxes) {
	if state == nil || result == nil {
		return 0, RelationshipAxes{}
	}

	scale := 1
	if result.Resolved {
		scale = 2
	}

	playerAhead := result.NPCDamage > result.PlayerDamage || (result.Resolved && state.Winner != "" && state.Winner != state.NPCName)
	playerBehind := result.PlayerDamage > result.NPCDamage || (result.Resolved && state.Winner == state.NPCName)

	switch {
	case playerAhead:
		switch result.PlayerAction {
		case SocialActionAppeal:
			return 2 * scale, RelationshipAxes{Trust: 3 * scale, Respect: 2 * scale}
		case SocialActionPressure:
			return -1 * scale, RelationshipAxes{Fear: 3 * scale, Respect: 2 * scale}
		case SocialActionDeceive:
			return -1 * scale, RelationshipAxes{Respect: 2 * scale, Trust: -1 * scale}
		case SocialActionConcede:
			return 1 * scale, RelationshipAxes{Trust: 2 * scale, Respect: 1 * scale}
		case SocialActionExpose:
			return -1 * scale, RelationshipAxes{Fear: 3 * scale, Respect: 3 * scale}
		case SocialActionEscalate:
			return -2 * scale, RelationshipAxes{Fear: 4 * scale, Respect: 1 * scale}
		default:
			return 0, RelationshipAxes{}
		}
	case playerBehind:
		switch result.PlayerAction {
		case SocialActionAppeal:
			return -2 * scale, RelationshipAxes{Trust: -3 * scale, Respect: -1 * scale}
		case SocialActionPressure:
			return -3 * scale, RelationshipAxes{Fear: -2 * scale, Respect: -2 * scale}
		case SocialActionDeceive, SocialActionExpose:
			return -3 * scale, RelationshipAxes{Trust: -4 * scale, Respect: -2 * scale}
		case SocialActionConcede:
			return -1 * scale, RelationshipAxes{Debt: 1 * scale, Respect: -1 * scale}
		case SocialActionWithdraw:
			return -1 * scale, RelationshipAxes{Debt: 1 * scale}
		case SocialActionEscalate:
			return -3 * scale, RelationshipAxes{Fear: -2 * scale, Respect: -3 * scale}
		default:
			return 0, RelationshipAxes{}
		}
	default:
		return 0, RelationshipAxes{}
	}
}

func socialDuelRelationshipSummary(npcName string, dispositionDelta int, delta RelationshipAxes) string {
	var parts []string
	if dispositionDelta != 0 {
		parts = append(parts, fmt.Sprintf("disposition %s%d", signedInt(dispositionDelta), dispositionDelta))
	}
	for _, axis := range []struct {
		label string
		value int
	}{
		{"trust", delta.Trust},
		{"fear", delta.Fear},
		{"debt", delta.Debt},
		{"respect", delta.Respect},
		{"intimacy", delta.Intimacy},
	} {
		if axis.value == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s%d", axis.label, signedInt(axis.value), axis.value))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s's view of you holds steady.", npcName)
	}
	return fmt.Sprintf("%s's view of you shifts: %s.", npcName, strings.Join(parts, ", "))
}

func socialDuelNPCNote(npc *storage.NPC, state *SocialDuelState, result *SocialRoundResult, cue *SocialDuelCue, currentTurn int) string {
	if npc == nil || state == nil || result == nil {
		return ""
	}
	objective := firstNonEmpty(strings.TrimSpace(state.Objective), cueFieldValue(cue, func(c *SocialDuelCue) string { return c.Objective }))
	if objective == "" {
		objective = "an unresolved argument"
	}
	outcome := firstNonEmpty(strings.TrimSpace(result.Outcome), "exchange")
	return fmt.Sprintf("Turn %d: %s pressed %q against %s (%s via %s).",
		currentTurn,
		firstNonEmpty(strings.TrimSpace(state.Winner), "Someone"),
		objective,
		npc.Name,
		outcome,
		string(result.PlayerAction),
	)
}

func socialDuelPrivateThought(npc *storage.NPC, state *SocialDuelState, result *SocialRoundResult) string {
	if npc == nil || state == nil || result == nil || !result.Resolved {
		return ""
	}

	switch {
	case state.Winner == npc.Name:
		return fmt.Sprintf("I can still move %s off balance when the stakes rise.", firstNonEmpty(strings.TrimSpace(state.Objective), "them"))
	case result.PlayerAction == SocialActionExpose || result.PlayerAction == SocialActionPressure || result.PlayerAction == SocialActionEscalate:
		return "They know how to corner me when the room is listening."
	default:
		return "They can make me yield without ever drawing steel."
	}
}

func socialDuelNemesisEvent(state *SocialDuelState, result *SocialRoundResult, cue *SocialDuelCue, currentTurn int) NemesisEvent {
	event := NemesisEvent{
		Kind:    "social_duel",
		Outcome: result.Outcome,
		Detail:  firstNonEmpty(strings.TrimSpace(state.Objective), cueFieldValue(cue, func(c *SocialDuelCue) string { return c.Objective })),
		Turn:    currentTurn,
		Impact:  1,
		Pattern: string(result.PlayerAction),
	}

	if result.FailForward != nil {
		event.Kind = "political_fallout"
		event.Impact = 3
		event.Detail = firstNonEmpty(strings.TrimSpace(result.FailForward.Title), event.Detail)
		return event
	}

	if result.Resolved && state.Winner == state.NPCName {
		event.Impact = 2
		return event
	}

	if result.Resolved && (result.PlayerAction == SocialActionPressure || result.PlayerAction == SocialActionExpose || result.PlayerAction == SocialActionEscalate) {
		event.Kind = "humiliation"
		event.Impact = 3
		event.Scar = fmt.Sprintf("Publicly bent over %s", firstNonEmpty(strings.TrimSpace(state.Objective), "a high-stakes exchange"))
		return event
	}

	if result.Resolved {
		event.Impact = 2
	}
	return event
}

func socialDuelWorldReaction(state *SocialDuelState, result *SocialRoundResult, cue *SocialDuelCue, currentTurn int) *WorldReaction {
	if state == nil || result == nil || (!result.Resolved && result.FailForward == nil) {
		return nil
	}

	reaction := &WorldReaction{
		ID:          uuid.NewString(),
		Status:      "active",
		SourceTurn:  currentTurn,
		CreatedTurn: currentTurn,
	}

	switch {
	case result.FailForward != nil:
		reaction.Kind = firstNonEmpty(strings.TrimSpace(result.FailForward.Kind), "setback")
		reaction.Title = firstNonEmpty(strings.TrimSpace(result.FailForward.Title), fmt.Sprintf("Fallout from %s", state.NPCName))
		reaction.Detail = firstNonEmpty(strings.TrimSpace(result.FailForward.Detail), cueFieldValue(cue, func(c *SocialDuelCue) string { return c.Stakes }))
	case state.Winner != "" && state.Winner != state.NPCName:
		if result.PlayerAction == SocialActionPressure || result.PlayerAction == SocialActionExpose || result.PlayerAction == SocialActionEscalate {
			reaction.Kind = "heat"
			reaction.Title = fmt.Sprintf("Word spreads after your showdown with %s", state.NPCName)
		} else {
			reaction.Kind = "rumor"
			reaction.Title = fmt.Sprintf("%s bends under pressure", state.NPCName)
		}
		reaction.Detail = firstNonEmpty(strings.TrimSpace(state.Objective), cueFieldValue(cue, func(c *SocialDuelCue) string { return c.Objective }))
	case state.Winner == state.NPCName:
		reaction.Kind = "setback"
		reaction.Title = fmt.Sprintf("%s controls the next move", state.NPCName)
		reaction.Detail = firstNonEmpty(strings.TrimSpace(state.Stakes), cueFieldValue(cue, func(c *SocialDuelCue) string { return c.Stakes }))
	default:
		return nil
	}

	return reaction
}

func applySocialDuelFrontPressure(world *storage.WorldState, npc *storage.NPC, state *SocialDuelState, result *SocialRoundResult, currentTurn int) string {
	if world == nil || state == nil || result == nil {
		return ""
	}
	fronts := loadFronts(world)
	if len(fronts) == 0 {
		return ""
	}
	idx := findRelevantSocialFront(fronts, npc, state)
	if idx < 0 {
		return ""
	}

	kind := "setback"
	level := 1
	detail := fmt.Sprintf("Your exchange with %s unsettles the front.", state.NPCName)
	if result.FailForward != nil || state.Winner == state.NPCName {
		kind = "heat"
		level = 2
		detail = fmt.Sprintf("The duel with %s gives this front more leverage in %s.", state.NPCName, world.CurrentLocation)
	}

	pressure := FrontPressure{
		Region:      firstNonEmpty(strings.TrimSpace(world.CurrentLocation), "Unknown"),
		Kind:        kind,
		Level:       level,
		Detail:      detail,
		UpdatedTurn: currentTurn,
	}
	upsertFrontPressure(&fronts[idx], pressure)
	storeFronts(world, fronts)
	return fmt.Sprintf("Front pressure shifts: %s (%s in %s).", fronts[idx].Title, pressure.Kind, pressure.Region)
}

func findRelevantSocialFront(fronts []Front, npc *storage.NPC, state *SocialDuelState) int {
	candidates := []string{state.NPCName, state.Objective, state.Stakes}
	if npc != nil {
		candidates = append(candidates, npc.Name, npc.Role)
	}

	for i, front := range fronts {
		haystack := strings.ToLower(strings.Join([]string{
			front.Faction,
			front.Title,
			front.PublicTitle,
			front.Stakes,
			front.PublicStakes,
		}, " "))
		for _, candidate := range candidates {
			for _, token := range strings.Fields(strings.ToLower(candidate)) {
				if len(token) < 4 {
					continue
				}
				if strings.Contains(haystack, token) {
					return i
				}
			}
		}
	}
	return -1
}

func appendJSONStringUnique(raw, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return raw
	}

	var items []string
	_ = json.Unmarshal([]byte(raw), &items)
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			payload, _ := json.Marshal(items)
			return string(payload)
		}
	}
	items = append(items, value)
	payload, err := json.Marshal(items)
	if err != nil {
		return raw
	}
	return string(payload)
}

func relationshipAxesZero(delta RelationshipAxes) bool {
	return delta.Trust == 0 && delta.Fear == 0 && delta.Debt == 0 && delta.Respect == 0 && delta.Intimacy == 0
}

func signedInt(value int) string {
	if value < 0 {
		return ""
	}
	return "+"
}

func cueFieldValue(cue *SocialDuelCue, fn func(*SocialDuelCue) string) string {
	if cue == nil {
		return ""
	}
	return strings.TrimSpace(fn(cue))
}
