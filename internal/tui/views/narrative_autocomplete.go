package views

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/engine"
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
	Name string
	Hint string
}

var slashCommandSpecs = []slashCommandSpec{
	{Name: "inventory", Hint: "Show your inventory", Aliases: []string{"i"}},
	{Name: "stats", Hint: "Show character sheet", Aliases: []string{"s"}},
	{Name: "map", Hint: "Show discovered world map", Aliases: []string{"m"}},
	{Name: "journal", Hint: "Show chapter journal", Aliases: []string{"j"}},
	{Name: "thoughts", Hint: "Inspect saved NPC private thoughts"},
	{Name: "codex", Hint: "Open the story codex"},
	{Name: "fronts", Hint: "Open the fronts and fallout tracker", Aliases: []string{"hooks"}},
	{Name: "investigations", Hint: "Open the investigation workspace"},
	{Name: "projects", Hint: "Open the project workspace"},
	{Name: "btw", Hint: "Ask a contextual side question", TrailingSpace: true},
	{Name: "guide", Hint: "Store soft future story guidance", TrailingSpace: true},
	{Name: "advance", Hint: "Push to the next meaningful beat; free text accepted", TrailingSpace: true},
	{Name: "timeskip", Hint: "Jump ahead to a later meaningful moment; free text accepted", TrailingSpace: true},
	{Name: "achievements", Hint: "Show earned achievements", Aliases: []string{"a"}},
	{Name: "narrator", Hint: "Direct narrator canon", Aliases: []string{"n"}, TrailingSpace: true},
	{Name: "craft", Hint: "Open the crafting station"},
	{Name: "talk", Hint: "Talk to a nearby NPC", TrailingSpace: true},
	{Name: "downtime", Hint: "Request a quieter scene", TrailingSpace: true},
	{Name: "save", Hint: "Save your game", TrailingSpace: true},
	{Name: "load", Hint: "Open the save picker"},
	{Name: "help", Hint: "Show available commands"},
	{Name: "quit", Hint: "Save and leave the session", Aliases: []string{"q"}},
}

var talkIntentSpecs = []talkIntentSpec{
	{Name: "ask", Hint: "Ask directly for facts or help"},
	{Name: "probe", Hint: "Press for subtext or hidden motives"},
	{Name: "bond", Hint: "Open up and build trust"},
	{Name: "bargain", Hint: "Negotiate terms or leverage"},
	{Name: "threaten", Hint: "Push with fear or pressure"},
	{Name: "promise", Hint: "Offer a commitment"},
	{Name: "lie", Hint: "Mislead or hide the truth"},
	{Name: "confess", Hint: "Reveal something vulnerable"},
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
	specs := make([]slashCommandSpec, 0, len(slashCommandSpecs))
	for _, spec := range slashCommandSpecs {
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
		return nearbyTalkNPCSuggestionItems(m.narrator.DB(), m.narrator.Story().ID, m.narrator.Turn(), 6, "")
	}

	if err != nil {
		return nil
	}
	trailingSpace := strings.HasSuffix(rawValue, " ")
	if npcName, rest, matched := matchTalkNPCInput(argsText, npcs); matched && (rest != "" || trailingSpace) {
		return buildTalkIntentSuggestionItems(npcName, rest)
	}

	return buildTalkNPCSuggestionItems(npcs, argsText)
}

func buildTalkNPCSuggestionItems(npcs []storage.NPC, query string) []components.SuggestionItem {
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
			hintParts = append(hintParts, fmt.Sprintf("seen turn %d", npc.LastSeenTurn))
		}
		items = append(items, components.SuggestionItem{
			Value: "/talk " + name + " ",
			Label: name,
			Hint:  strings.Join(hintParts, " · "),
		})
	}
	return items
}

func nearbyTalkNPCSuggestionItems(db *storage.DB, storyID string, currentTurn, limit int, query string) []components.SuggestionItem {
	npcs, err := engine.NearbyNPCs(db, storyID, currentTurn, limit)
	if err != nil {
		return nil
	}
	return buildTalkNPCSuggestionItems(npcs, query)
}

func buildTalkIntentSuggestionItems(npcName, query string) []components.SuggestionItem {
	query = strings.ToLower(strings.TrimSpace(query))
	items := make([]components.SuggestionItem, 0, len(talkIntentSpecs))
	for _, spec := range talkIntentSpecs {
		if query != "" && !strings.HasPrefix(spec.Name, query) {
			continue
		}
		items = append(items, components.SuggestionItem{
			Value: "/talk " + npcName + " " + spec.Name,
			Label: spec.Name,
			Hint:  spec.Hint,
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
