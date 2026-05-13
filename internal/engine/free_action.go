package engine

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

type freeActionProfile struct {
	Macro          bool
	HasTimeSkip    bool
	HasDream       bool
	HasRelocation  bool
	HasSceneBundle bool
}

func buildFreeActionInterpretationSummary(story *storage.Story, world *storage.WorldState, currentInput string) string {
	input := strings.TrimSpace(currentInput)
	if input == "" || strings.HasPrefix(input, "[") {
		return ""
	}

	profile := analyzeFreeActionInput(input)
	if !profile.Macro && !profile.HasTimeSkip && !profile.HasDream && !profile.HasRelocation {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Free Action Handling\n")
	sb.WriteString("The current player input is a free-form action or desired direction, not already-resolved canon.\n")
	sb.WriteString("- Preserve the canonical story continuity: same world, genre, tone, era, and story language.\n")
	if story != nil && strings.TrimSpace(story.Name) != "" {
		sb.WriteString(fmt.Sprintf("- This is still the story \"%s\". Do NOT drift into an unrelated setting or genre.\n", strings.TrimSpace(story.Name)))
	}
	if world != nil && strings.TrimSpace(world.CurrentLocation) != "" {
		sb.WriteString(fmt.Sprintf("- Anchor the next response to the current reality around %s unless the fiction clearly earns a change.\n", strings.TrimSpace(world.CurrentLocation)))
	}
	sb.WriteString("- Translate the player's intent into the next coherent scene. Do NOT literalize every clause if the input bundles multiple beats.\n")
	sb.WriteString("- Durable changes must still be expressed through valid state_changes and grounded narration.\n")
	if profile.HasDream {
		sb.WriteString("- If you use a dream, omen, or vision, keep it tied to established lore and return clearly to the same story reality.\n")
	}
	if profile.HasRelocation {
		sb.WriteString("- Any relocation, portal, or teleport beat must still belong to this story's established world unless the fiction has explicitly opened a new realm.\n")
	}
	if profile.HasTimeSkip {
		sb.WriteString("- If you honor a time jump, make continuity explicit: what changed, what stayed true, and why this new moment matters. Emit timeline_update when appropriate.\n")
	}
	if profile.HasSceneBundle {
		sb.WriteString("- Favor one strong consequence or transition over a rushed montage of many unrelated events in a single beat.\n")
	}
	return sb.String()
}

func analyzeFreeActionInput(input string) freeActionProfile {
	lower := strings.ToLower(strings.TrimSpace(input))
	if lower == "" {
		return freeActionProfile{}
	}

	wordCount := len(strings.Fields(lower))
	clauseCount := strings.Count(lower, ".") + strings.Count(lower, ";") + strings.Count(lower, ":")

	timeSkipTerms := []string{
		"anni", "anno", "mesi", "mese", "settimane", "settimana", "giorni", "giorno",
		"piu tardi", "più tardi", "dopo", "ho dormito", "mi sveglio", "mi risveglio",
		"sono cresciuto", "sono cresciuta", "cresciuto", "cresciuta", "passano",
	}
	dreamTerms := []string{"sogno", "sogna", "sognai", "visione", "visione", "premonizione"}
	relocationTerms := []string{
		"teletrasport", "portale", "frattura nello spazio", "altro posto", "altrove",
		"mi ritrovo", "vengo trasportato", "vengo teletrasportato",
	}
	sceneBundleTerms := []string{
		"e poi", "mentre", "allora", "dopo di che", "dopodiche", "succede", "arriva", "improvvisamente",
	}

	profile := freeActionProfile{
		HasTimeSkip:    containsAny(lower, timeSkipTerms),
		HasDream:       containsAny(lower, dreamTerms),
		HasRelocation:  containsAny(lower, relocationTerms),
		HasSceneBundle: containsAny(lower, sceneBundleTerms),
	}
	if wordCount >= 30 || clauseCount >= 3 {
		profile.Macro = true
	}
	if profile.HasTimeSkip || profile.HasDream || profile.HasRelocation {
		profile.Macro = true
	}
	if profile.HasSceneBundle && wordCount >= 18 {
		profile.Macro = true
	}
	return profile
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
