package engine

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	choiceNPCPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\b(?:Chiedere a|Chiedi a|Parlare con|Parla con|Premi|Interroga(?:re)?|Osservare ancora|Segui(?:re)?|Aiuta(?:re)?|Convincere)\s+([A-ZÀ-Ý][\p{L}'-]{2,}(?:\s+[A-ZÀ-Ý][\p{L}'-]{2,}){0,2})`),
	}
	narrativeNPCSpeechPattern = regexp.MustCompile(`(?:^|[.!?]\s+)([A-ZÀ-Ý][\p{L}'-]{2,}(?:\s+[A-ZÀ-Ý][\p{L}'-]{2,}){0,2})\s+(?:deglutisce|stringe|dice|mormora|risponde|sussurra|abbassa|guarda|annuisce|sorride|esita|indietreggia|avanza)\b`)
)

func inferNPCsFromNarrativeResponse(
	narrative *NarrativeResponse,
	storyID string,
	currentTurn int,
	npcs npcStateStore,
) []StateChange {
	if narrative == nil || npcs == nil {
		return nil
	}

	candidates := map[string]string{}
	addCandidate := func(name, detail string) {
		name = cleanInferredNPCName(name)
		if !looksLikeTrackableNPCName(name) {
			return
		}
		if _, ok := candidates[strings.ToLower(name)]; ok {
			return
		}
		candidates[name] = strings.TrimSpace(detail)
	}

	for _, entity := range narrative.EntitiesMentioned {
		if strings.EqualFold(entity.Type, "npc") || strings.EqualFold(entity.Type, "person") || strings.EqualFold(entity.Type, "character") {
			addCandidate(entity.Name, "Mentioned as an important character in this turn.")
		}
	}
	for _, block := range narrative.DialogueBlocks {
		if isPlayerOrNarratorSpeaker(block.Speaker) {
			continue
		}
		addCandidate(block.Speaker, compactSpeakerDetail(block.Text))
	}
	for _, match := range narrativeNPCSpeechPattern.FindAllStringSubmatch(narrative.Narrative, -1) {
		if len(match) > 1 {
			addCandidate(match[1], "Named speaker or actor surfaced in narrative prose.")
		}
	}
	for _, choice := range narrative.Choices {
		for _, pattern := range choiceNPCPatterns {
			for _, match := range pattern.FindAllStringSubmatch(choice.Text, -1) {
				if len(match) > 1 {
					addCandidate(match[1], "Referenced by a current player choice.")
				}
			}
		}
	}

	applied := make([]StateChange, 0, len(candidates))
	for name, detail := range candidates {
		existing, err := npcs.GetNPCByName(storyID, name)
		if err != nil || existing != nil {
			continue
		}
		npc, err := ensureNPCForStateChange(npcs, storyID, name, currentTurn)
		if err != nil || npc == nil {
			continue
		}
		if detail != "" {
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

func cleanInferredNPCName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, " \t\n\r.,;:!?\"'`()[]{}")
	for _, prefix := range []string{"il ", "lo ", "la ", "l'", "un ", "una "} {
		if len(name) > len(prefix) && strings.HasPrefix(strings.ToLower(name), prefix) {
			return strings.TrimSpace(name[len(prefix):])
		}
	}
	return name
}

func isPlayerOrNarratorSpeaker(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "player", "protagonist", "narrator", "system", "you", "tu":
		return true
	default:
		return false
	}
}

func compactSpeakerDetail(text string) string {
	text = strings.TrimSpace(text)
	if len([]rune(text)) <= 140 {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:140])) + "..."
}
