package views

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
)

type slashCommandSpec struct {
	Name          string
	Hint          string
	Aliases       []string
	TrailingSpace bool
}

type talkIntentSpec struct {
	Name    string
	HintKey string
}

var slashCommandSpecs = contractSlashCommandSpecs()

var talkIntentSpecs = []talkIntentSpec{
	{Name: "ask", HintKey: "talk.intent.ask"},
	{Name: "probe", HintKey: "talk.intent.probe"},
	{Name: "bond", HintKey: "talk.intent.bond"},
	{Name: "bargain", HintKey: "talk.intent.bargain"},
	{Name: "threaten", HintKey: "talk.intent.threaten"},
	{Name: "promise", HintKey: "talk.intent.promise"},
	{Name: "lie", HintKey: "talk.intent.lie"},
	{Name: "confess", HintKey: "talk.intent.confess"},
}

func contractSlashCommandSpecs(localizers ...appi18n.Localizer) []slashCommandSpec {
	descriptors := contracts.CommandDescriptors(localizers...)
	specs := make([]slashCommandSpec, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Parity == contracts.CommandParityBrowserOnly {
			continue
		}
		name := descriptor.ID
		if descriptor.Canonical != "hooks" {
			name = descriptor.Canonical
		}
		aliases := append([]string{}, descriptor.Aliases...)
		if descriptor.Canonical != "" && descriptor.Canonical != name {
			aliases = append([]string{descriptor.Canonical}, aliases...)
		}
		specs = append(specs, slashCommandSpec{
			Name:          name,
			Hint:          descriptor.Description,
			Aliases:       aliases,
			TrailingSpace: descriptor.TrailingSpace,
		})
	}
	return specs
}

func (m *NarrativeModel) refreshSlashSuggestions() {
	if m == nil {
		return
	}
	if !m.inputFocus {
		m.slashSuggestions.SetItems(nil)
		return
	}
	m.slashSuggestions.SetItems(m.buildSlashSuggestions(m.input.Value()))
}

func (m *NarrativeModel) buildSlashSuggestions(rawValue string) []components.SuggestionItem {
	trimmed := strings.TrimSpace(rawValue)
	if !strings.HasPrefix(trimmed, "/") {
		return nil
	}

	commandBody := strings.TrimPrefix(trimmed, "/")
	parts := strings.Fields(commandBody)
	if len(parts) == 0 {
		return buildCommandSuggestionsForSpecs("", m.availableSlashCommandSpecs())
	}

	query := strings.ToLower(strings.TrimSpace(parts[0]))
	canonical := engine.CommandRegistry[query]
	trailingSpace := strings.HasSuffix(rawValue, " ")

	if canonical == "talk" {
		if len(parts) == 1 && !trailingSpace {
			return buildCommandSuggestionsForSpecs(query, m.availableSlashCommandSpecs())
		}
		return m.buildTalkSuggestions(rawValue)
	}

	if len(parts) == 1 && !trailingSpace {
		return buildCommandSuggestionsForSpecs(query, m.availableSlashCommandSpecs())
	}

	return nil
}

func buildCommandSuggestions(query string) []components.SuggestionItem {
	return buildCommandSuggestionsForSpecs(query, slashCommandSpecs)
}

func buildCommandSuggestionsForSpecs(query string, specs []slashCommandSpec) []components.SuggestionItem {
	query = strings.ToLower(strings.TrimSpace(query))

	items := make([]components.SuggestionItem, 0, len(specs))
	for _, spec := range specs {
		if query != "" && !slashCommandMatches(spec, query) {
			continue
		}

		label := "/" + spec.Name
		hint := spec.Hint
		if len(spec.Aliases) > 0 {
			aliases := make([]string, 0, len(spec.Aliases))
			for _, alias := range spec.Aliases {
				aliases = append(aliases, "/"+alias)
			}
			hint = fmt.Sprintf("%s · %s", hint, strings.Join(aliases, " "))
		}

		value := "/" + spec.Name
		if spec.TrailingSpace {
			value += " "
		}
		items = append(items, components.SuggestionItem{
			Value: value,
			Label: label,
			Hint:  hint,
		})
	}
	return items
}

func (m NarrativeModel) availableSlashCommandSpecs() []slashCommandSpec {
	source := m.commandSpecs
	if len(source) == 0 {
		source = slashCommandSpecs
	}
	specs := make([]slashCommandSpec, 0, len(source))
	for _, spec := range source {
		if spec.Name == "thoughts" && !m.visiblePrivateThoughts {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func slashCommandMatches(spec slashCommandSpec, query string) bool {
	if strings.HasPrefix(spec.Name, query) {
		return true
	}
	for _, alias := range spec.Aliases {
		if strings.HasPrefix(alias, query) {
			return true
		}
	}
	return false
}

func (m *NarrativeModel) buildTalkSuggestions(rawValue string) []components.SuggestionItem {
	if m == nil || m.narrator == nil || m.narrator.DB() == nil || m.narrator.Story() == nil {
		return nil
	}

	npcs, err := engine.NearbyNPCs(m.narrator.DB(), m.narrator.Story().ID, m.narrator.Turn(), 6)
	trimmed := strings.TrimSpace(rawValue)
	commandBody := strings.TrimPrefix(trimmed, "/")
	if !strings.HasPrefix(strings.ToLower(commandBody), "talk") {
		return nil
	}

	argsText := strings.TrimSpace(strings.TrimPrefix(commandBody, strings.Fields(commandBody)[0]))
	if argsText == "" {
		return nearbyTalkNPCSuggestionItems(m.narrator.DB(), m.narrator.Story().ID, m.narrator.Turn(), 6, "", m.loc)
	}

	if err != nil {
		return nil
	}
	trailingSpace := strings.HasSuffix(rawValue, " ")
	if npcName, rest, matched := matchTalkNPCInput(argsText, npcs); matched && (rest != "" || trailingSpace) {
		return buildTalkIntentSuggestionItems(npcName, rest, m.loc)
	}

	return buildTalkNPCSuggestionItems(npcs, argsText, m.loc)
}

func buildTalkNPCSuggestionItems(npcs []storage.NPC, query string, localizers ...appi18n.Localizer) []components.SuggestionItem {
	loc := viewLocalizer(localizers)
	query = strings.ToLower(strings.TrimSpace(query))
	items := make([]components.SuggestionItem, 0, len(npcs))
	for _, npc := range npcs {
		name := strings.TrimSpace(npc.Name)
		if name == "" {
			continue
		}
		nameLower := strings.ToLower(name)
		if query != "" && !strings.Contains(nameLower, query) {
			continue
		}

		hintParts := []string{}
		if role := strings.TrimSpace(npc.Role); role != "" {
			hintParts = append(hintParts, role)
		}
		if npc.LastSeenTurn > 0 {
			hintParts = append(hintParts, loc.T("talk.seen_turn", npc.LastSeenTurn))
		}
		items = append(items, components.SuggestionItem{
			Value: "/talk " + name + " ",
			Label: name,
			Hint:  strings.Join(hintParts, " · "),
		})
	}
	return items
}

func nearbyTalkNPCSuggestionItems(db *storage.DB, storyID string, currentTurn, limit int, query string, localizers ...appi18n.Localizer) []components.SuggestionItem {
	npcs, err := engine.NearbyNPCs(db, storyID, currentTurn, limit)
	if err != nil {
		return nil
	}
	return buildTalkNPCSuggestionItems(npcs, query, localizers...)
}

func buildTalkIntentSuggestionItems(npcName, query string, localizers ...appi18n.Localizer) []components.SuggestionItem {
	loc := viewLocalizer(localizers)
	query = strings.ToLower(strings.TrimSpace(query))
	items := make([]components.SuggestionItem, 0, len(talkIntentSpecs))
	for _, spec := range talkIntentSpecs {
		if query != "" && !strings.HasPrefix(spec.Name, query) {
			continue
		}
		items = append(items, components.SuggestionItem{
			Value: "/talk " + npcName + " " + spec.Name,
			Label: spec.Name,
			Hint:  loc.T(spec.HintKey),
		})
	}
	return items
}

func matchTalkNPCInput(argsText string, npcs []storage.NPC) (string, string, bool) {
	argsText = strings.TrimSpace(argsText)
	if argsText == "" {
		return "", "", false
	}

	bestName := ""
	bestRest := ""
	bestLen := 0
	lowerArgs := strings.ToLower(argsText)
	for _, npc := range npcs {
		name := strings.TrimSpace(npc.Name)
		if name == "" {
			continue
		}
		lowerName := strings.ToLower(name)
		switch {
		case lowerArgs == lowerName:
			if len(lowerName) > bestLen {
				bestName = name
				bestRest = ""
				bestLen = len(lowerName)
			}
		case strings.HasPrefix(lowerArgs, lowerName+" "):
			if len(lowerName) > bestLen {
				bestName = name
				bestRest = strings.TrimSpace(argsText[len(name):])
				bestLen = len(lowerName)
			}
		}
	}

	if bestName == "" {
		return "", "", false
	}
	return bestName, bestRest, true
}
