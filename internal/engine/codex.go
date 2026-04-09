package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

// CodexCategory groups player-facing codex entries.
type CodexCategory struct {
	Key   string
	Label string
}

// CodexLink is a navigable relation from one codex entry to another.
type CodexLink struct {
	EntryID string
	Label   string
}

// CodexSection is a titled block of descriptive lines within an entry.
type CodexSection struct {
	Title string
	Lines []string
}

// CodexEntry is a single descriptive codex or dossier entry.
type CodexEntry struct {
	ID       string
	Category string
	Title    string
	Subtitle string
	Summary  string
	Sections []CodexSection
	Related  []CodexLink
}

// CodexIndex is the full browseable codex for a story.
type CodexIndex struct {
	StoryID         string
	StoryName       string
	Categories      []CodexCategory
	CategoryEntries map[string][]string
	Entries         map[string]CodexEntry
}

// Entry returns a codex entry by id.
func (c *CodexIndex) Entry(id string) (CodexEntry, bool) {
	if c == nil {
		return CodexEntry{}, false
	}
	entry, ok := c.Entries[id]
	return entry, ok
}

// ProtagonistCodexEntryID returns the protagonist dossier entry id.
func ProtagonistCodexEntryID() string {
	return "people:protagonist"
}

// BuildStoryCodexByID loads and materializes the player-facing codex for a story.
func BuildStoryCodexByID(db *storage.DB, storyID string) (*CodexIndex, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	story, err := db.GetStory(storyID)
	if err != nil {
		return nil, err
	}
	char, _ := db.GetCharacterByStory(storyID)
	world, _ := db.GetWorldState(storyID)
	return BuildStoryCodex(db, story, char, world)
}

// BuildStoryCodex materializes the player-facing codex for a story.
func BuildStoryCodex(db *storage.DB, story *storage.Story, char *storage.Character, world *storage.WorldState) (*CodexIndex, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if story == nil {
		return nil, fmt.Errorf("story is nil")
	}

	index := &CodexIndex{
		StoryID:         story.ID,
		StoryName:       story.Name,
		Categories:      []CodexCategory{},
		CategoryEntries: map[string][]string{},
		Entries:         map[string]CodexEntry{},
	}

	addEntry := func(entry CodexEntry) {
		if strings.TrimSpace(entry.ID) == "" || strings.TrimSpace(entry.Title) == "" {
			return
		}
		entry.Category = strings.TrimSpace(strings.ToLower(entry.Category))
		if entry.Category == "" {
			return
		}
		index.Entries[entry.ID] = entry
		index.CategoryEntries[entry.Category] = append(index.CategoryEntries[entry.Category], entry.ID)
	}

	addCategory := func(key, label string) {
		if len(index.CategoryEntries[key]) == 0 {
			return
		}
		index.Categories = append(index.Categories, CodexCategory{Key: key, Label: label})
	}

	npcs, _ := db.ListNPCs(story.ID)
	chapters, _ := db.ListChapters(story.ID)
	achievements, _ := db.ListAchievements(story.ID)
	messages, _ := db.GetRecentMessagesByStory(story.ID, 8)
	fronts := knownFronts(loadFronts(world))
	investigations := loadInvestigationBoard(world)
	projects := loadProjectBoard(world)

	if entry := buildProtagonistCodexEntry(story, char, world, achievements, npcs, fronts, investigations); entry.Title != "" {
		addEntry(entry)
	}

	for _, npc := range npcs {
		addEntry(buildNPCCodexEntry(&npc))
	}

	for _, loc := range parseKnownLocations(valueOrEmpty(world, func(w *storage.WorldState) string { return w.KnownLocationsJSON })) {
		addEntry(buildLocationCodexEntry(loc, world, fronts))
	}

	for _, faction := range parseFactionEntries(story.SettingJSON) {
		addEntry(faction)
	}

	for _, front := range fronts {
		addEntry(buildFrontCodexEntry(front))
	}
	for _, invCase := range investigations.Cases {
		addEntry(buildInvestigationCodexEntry(invCase))
	}
	for _, project := range projects.Projects {
		addEntry(buildProjectCodexEntry(project))
	}

	hooks := activeStoryHooks(loadStoryHooks(world))
	for _, hook := range hooks {
		if strings.EqualFold(hook.Kind, "mystery") || strings.EqualFold(hook.Kind, "rumor") {
			addEntry(buildMysteryEntry(hook))
		}
		addEntry(buildThreadEntryFromHook(hook))
	}

	reactions := visibleWorldReactions(loadWorldReactions(world))
	for _, reaction := range reactions {
		addEntry(buildThreadEntryFromReaction(reaction))
	}

	for id, entry := range index.Entries {
		entry.Related = appendUniqueLinks(entry.Related)
		entry.Sections = compactCodexSections(entry.Sections)
		index.Entries[id] = entry
	}

	for _, ids := range index.CategoryEntries {
		sort.SliceStable(ids, func(i, j int) bool {
			left := index.Entries[ids[i]].Title
			right := index.Entries[ids[j]].Title
			return strings.ToLower(left) < strings.ToLower(right)
		})
	}

	addCategory("people", "People")
	addCategory("places", "Places")
	addCategory("factions", "Factions")
	addCategory("fronts", "Fronts")
	addCategory("mysteries", "Mysteries")
	addCategory("threads", "Threads")
	addCategory("investigations", "Investigations")
	addCategory("projects", "Projects")

	enrichCodexLinks(index, npcs, hooks, reactions, fronts, chapters, messages, investigations, projects)
	return index, nil
}

func buildProtagonistCodexEntry(story *storage.Story, char *storage.Character, world *storage.WorldState, achievements []storage.Achievement, npcs []storage.NPC, fronts []Front, board InvestigationBoard) CodexEntry {
	if story == nil || char == nil {
		return CodexEntry{}
	}

	stats := parseLooseJSONMap(char.StatsJSON)
	schema := parseLooseJSONMap(story.StatsSchemaJSON)

	entry := CodexEntry{
		ID:       ProtagonistCodexEntryID(),
		Category: "people",
		Title:    char.Name,
		Subtitle: "Protagonist",
		Summary:  strings.TrimSpace(char.Background),
	}
	projectBoard := loadProjectBoard(world)

	overview := []string{}
	if char.Background != "" {
		overview = append(overview, char.Background)
	}
	if story.Genre != "" {
		overview = append(overview, fmt.Sprintf("Genre: %s", story.Genre))
	}
	if story.Tone != "" {
		overview = append(overview, fmt.Sprintf("Tone: %s", story.Tone))
	}
	if world != nil {
		if world.CurrentLocation != "" {
			overview = append(overview, fmt.Sprintf("Current location: %s", world.CurrentLocation))
		}
		if world.CurrentChapter > 0 || world.CurrentTurn > 0 {
			overview = append(overview, fmt.Sprintf("Chapter %d · Turn %d", world.CurrentChapter, world.CurrentTurn))
		}
	}
	entry.Sections = append(entry.Sections, CodexSection{Title: "Overview", Lines: overview})

	if vitals := formatStatGroupLines(stats["vitals"], schema["vitals"]); len(vitals) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Vitals", Lines: vitals})
	}
	if attrs := formatStatGroupLines(stats["attributes"], schema["attributes"]); len(attrs) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Attributes", Lines: attrs})
	}
	if secondary := formatStatGroupLines(stats["secondary"], schema["secondary"]); len(secondary) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Secondary", Lines: secondary})
	}

	if skills := formatSkillsLines(char, stats); len(skills) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Skills", Lines: skills})
	}
	if traits := formatStringSliceLines(mergeStringLists(parseJSONStringSlice(char.TraitsJSON), parseLooseJSONStringSlice(stats["traits"]))); len(traits) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Traits", Lines: traits})
	}
	if titles := formatStringSliceLines(parseLooseJSONStringSlice(stats["titles"])); len(titles) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Titles", Lines: titles})
	}
	if deaths := codexToInt(stats["deaths"]); deaths > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Hard Lessons", Lines: []string{fmt.Sprintf("Deaths: %d", deaths)}})
	}

	if len(achievements) > 0 {
		lines := make([]string, 0, len(achievements))
		for _, achievement := range achievements {
			lines = append(lines, fmt.Sprintf("%s [%s]", achievement.Name, strings.ToLower(achievement.Rarity)))
		}
		entry.Sections = append(entry.Sections, CodexSection{Title: "Achievements", Lines: lines})
	}

	if len(npcs) > 0 {
		lines := make([]string, 0, len(npcs))
		for _, npc := range npcs {
			label := npc.Name
			if npc.Role != "" {
				label += " (" + npc.Role + ")"
			}
			lines = append(lines, fmt.Sprintf("%s — %s", label, codexRelationshipAxesSummary(npc.RelationshipJSON)))
			entry.Related = append(entry.Related, CodexLink{
				EntryID: codexNPCEntryID(npc.Name),
				Label:   npc.Name,
			})
		}
		entry.Sections = append(entry.Sections, CodexSection{Title: "Known People", Lines: lines})
	}
	if nemesisLines := buildProtagonistNemesisLines(npcs); len(nemesisLines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Nemeses", Lines: nemesisLines})
	}

	if len(fronts) > 0 {
		lines := make([]string, 0, len(fronts))
		for _, front := range fronts {
			lines = append(lines, formatKnownFrontSummary(front))
			entry.Related = append(entry.Related, CodexLink{
				EntryID: codexFrontEntryID(front.ID),
				Label:   frontDisplayTitle(front),
			})
		}
		entry.Sections = append(entry.Sections, CodexSection{Title: "Known Fronts", Lines: lines})
	}
	if lines := buildProtagonistInvestigationLines(board); len(lines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Open Investigations", Lines: lines})
		for _, invCase := range board.Cases {
			if strings.EqualFold(invCase.Status, "solved") {
				continue
			}
			entry.Related = append(entry.Related, CodexLink{
				EntryID: codexInvestigationEntryID(invCase.ID),
				Label:   invCase.Title,
			})
		}
	}
	if activeProjects, completedProjects := buildProtagonistProjectLines(projectBoard); len(activeProjects) > 0 || len(completedProjects) > 0 {
		if len(activeProjects) > 0 {
			entry.Sections = append(entry.Sections, CodexSection{Title: "Active Projects", Lines: activeProjects})
		}
		if len(completedProjects) > 0 {
			entry.Sections = append(entry.Sections, CodexSection{Title: "Completed Projects", Lines: completedProjects})
		}
		for _, project := range projectBoard.Projects {
			entry.Related = append(entry.Related, CodexLink{
				EntryID: codexProjectEntryID(project.ID),
				Label:   project.Title,
			})
		}
	}

	return entry
}

func buildNPCCodexEntry(npc *storage.NPC) CodexEntry {
	if npc == nil || strings.TrimSpace(npc.Name) == "" {
		return CodexEntry{}
	}

	entry := CodexEntry{
		ID:       codexNPCEntryID(npc.Name),
		Category: "people",
		Title:    npc.Name,
		Subtitle: strings.TrimSpace(npc.Role),
		Summary:  strings.TrimSpace(npc.Appearance),
	}

	if summary := strings.TrimSpace(FormatNPCForPlayer(npc)); summary != "" {
		entry.Sections = append(entry.Sections, CodexSection{
			Title: "Profile",
			Lines: splitBulletLines(summary),
		})
	}
	if section := buildNPCNemesisCodexSection(npc); len(section.Lines) > 0 {
		entry.Sections = append(entry.Sections, section)
	}

	if npc.LastSeenTurn > 0 {
		entry.Sections = append(entry.Sections, CodexSection{
			Title: "Story Relevance",
			Lines: []string{
				fmt.Sprintf("First appeared: turn %d", npc.FirstAppearedTurn),
				fmt.Sprintf("Last seen: turn %d", npc.LastSeenTurn),
			},
		})
	}
	return entry
}

func buildLocationCodexEntry(loc KnownLocation, world *storage.WorldState, fronts []Front) CodexEntry {
	if strings.TrimSpace(loc.Name) == "" {
		return CodexEntry{}
	}

	lines := []string{}
	if loc.Region != "" {
		lines = append(lines, "Region: "+loc.Region)
	}
	if loc.DiscoveredTurn > 0 {
		lines = append(lines, fmt.Sprintf("Discovered: turn %d", loc.DiscoveredTurn))
	}
	if loc.Description != "" {
		lines = append(lines, loc.Description)
	}
	if world != nil && strings.EqualFold(loc.Name, world.CurrentLocation) {
		lines = append(lines, "Current location.")
	}

	entry := CodexEntry{
		ID:       codexLocationEntryID(loc.Name),
		Category: "places",
		Title:    loc.Name,
		Subtitle: strings.TrimSpace(loc.Region),
		Summary:  strings.TrimSpace(loc.Description),
		Sections: []CodexSection{{Title: "Overview", Lines: lines}},
	}
	pressureLines := []string{}
	for _, front := range fronts {
		matched := false
		for _, pressure := range normalizeFrontPressures(front.Pressures) {
			if !strings.EqualFold(pressure.Region, loc.Name) && (loc.Region == "" || !strings.EqualFold(pressure.Region, loc.Region)) {
				continue
			}
			pressureLines = append(pressureLines, frontDisplayTitle(front)+" - "+formatFrontPressureDisplay(pressure))
			matched = true
		}
		if matched {
			entry.Related = append(entry.Related, CodexLink{
				EntryID: codexFrontEntryID(front.ID),
				Label:   frontDisplayTitle(front),
			})
		}
	}
	if len(pressureLines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Pressure", Lines: pressureLines})
	}
	return entry
}

func buildFrontCodexEntry(front Front) CodexEntry {
	title := frontDisplayTitle(front)
	if title == "" {
		return CodexEntry{}
	}

	faction := ""
	if strings.EqualFold(front.Visibility, "known") {
		faction = strings.TrimSpace(front.Faction)
	}

	subtitleParts := []string{}
	if faction != "" {
		subtitleParts = append(subtitleParts, faction)
	}
	if status := strings.TrimSpace(front.Status); status != "" {
		subtitleParts = append(subtitleParts, strings.Title(strings.ToLower(status)))
	}

	summary := frontDisplayStakes(front)
	if summary == "" && len(front.Pressures) > 0 {
		summary = formatFrontPressureDisplay(normalizeFrontPressures(front.Pressures)[0])
	}

	entry := CodexEntry{
		ID:       codexFrontEntryID(front.ID),
		Category: "fronts",
		Title:    title,
		Subtitle: strings.Join(subtitleParts, " · "),
		Summary:  summary,
	}

	overview := []string{}
	if summary != "" {
		overview = append(overview, summary)
	}
	if front.Segments > 0 {
		overview = append(overview, fmt.Sprintf("Progress: %d/%d", front.Progress, front.Segments))
	}
	if faction != "" {
		overview = append(overview, "Faction: "+faction)
		entry.Related = append(entry.Related, CodexLink{
			EntryID: codexFactionEntryID(faction),
			Label:   faction,
		})
	}
	if front.LastAdvancedTurn > 0 {
		overview = append(overview, fmt.Sprintf("Last advanced on turn %d", front.LastAdvancedTurn))
	}
	if front.NextEscalationTurn > 0 && !strings.EqualFold(front.Status, "resolved") {
		overview = append(overview, fmt.Sprintf("Next escalation on turn %d", front.NextEscalationTurn))
	}
	entry.Sections = append(entry.Sections, CodexSection{Title: "Overview", Lines: overview})

	if pressures := normalizeFrontPressures(front.Pressures); len(pressures) > 0 {
		lines := make([]string, 0, len(pressures))
		for _, pressure := range pressures {
			lines = append(lines, formatFrontPressureDisplay(pressure))
			entry.Related = append(entry.Related, CodexLink{
				EntryID: codexLocationEntryID(pressure.Region),
				Label:   pressure.Region,
			})
		}
		entry.Sections = append(entry.Sections, CodexSection{Title: "Pressure", Lines: lines})
	}
	if resolution := strings.TrimSpace(front.Resolution); resolution != "" {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Outcome", Lines: []string{resolution}})
	}
	return entry
}

func buildInvestigationCodexEntry(invCase InvestigationCase) CodexEntry {
	if strings.TrimSpace(invCase.Title) == "" {
		return CodexEntry{}
	}

	entry := CodexEntry{
		ID:       codexInvestigationEntryID(invCase.ID),
		Category: "investigations",
		Title:    invCase.Title,
		Subtitle: strings.Title(strings.ToLower(firstNonEmpty(invCase.Status, "open"))),
		Summary:  strings.TrimSpace(invCase.Summary),
	}

	overview := []string{}
	if invCase.Summary != "" {
		overview = append(overview, invCase.Summary)
	}
	overview = append(overview, "Status: "+firstNonEmpty(invCase.Status, "open"))
	if invCase.UpdatedTurn > 0 {
		overview = append(overview, fmt.Sprintf("Updated on turn %d", invCase.UpdatedTurn))
	}
	entry.Sections = append(entry.Sections, CodexSection{Title: "Overview", Lines: compactLines(overview)})

	if lines := formatInvestigationClueLines(invCase.Clues); len(lines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Clues", Lines: lines})
	}
	if lines := formatInvestigationSuspectLines(invCase.Suspects); len(lines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Suspects", Lines: lines})
	}
	if lines := formatInvestigationClaimLines(invCase.Claims); len(lines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Claims", Lines: lines})
	}
	if lines := formatInvestigationContradictionLines(invCase.Contradictions); len(lines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Contradictions", Lines: lines})
	}
	if lines := formatInvestigationLeadLines(invCase.Leads); len(lines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Leads", Lines: lines})
	}
	if lines := formatInvestigationTheoryLines(invCase.Theories); len(lines) > 0 {
		entry.Sections = append(entry.Sections, CodexSection{Title: "Theories", Lines: lines})
	}

	entry.Related = append(entry.Related, investigationCaseCodexLinks(invCase)...)
	return entry
}

func parseFactionEntries(raw string) []CodexEntry {
	setting := parseLooseJSONMap(raw)
	rawFactions, ok := setting["factions"].([]interface{})
	if !ok || len(rawFactions) == 0 {
		return nil
	}

	entries := make([]CodexEntry, 0, len(rawFactions))
	for _, rawFaction := range rawFactions {
		switch faction := rawFaction.(type) {
		case string:
			name := strings.TrimSpace(faction)
			if name == "" {
				continue
			}
			entries = append(entries, CodexEntry{
				ID:       codexFactionEntryID(name),
				Category: "factions",
				Title:    name,
				Sections: []CodexSection{{Title: "Overview", Lines: []string{"Known faction in this story setting."}}},
			})
		case map[string]interface{}:
			name := firstNonEmptyString(faction["name"], faction["title"], faction["label"])
			if name == "" {
				continue
			}
			lines := []string{}
			for _, key := range []string{"description", "goal", "motive", "reputation"} {
				if value := firstNonEmptyString(faction[key]); value != "" {
					lines = append(lines, value)
				}
			}
			entries = append(entries, CodexEntry{
				ID:       codexFactionEntryID(name),
				Category: "factions",
				Title:    name,
				Summary:  firstNonEmptyString(faction["description"]),
				Sections: []CodexSection{{Title: "Overview", Lines: lines}},
			})
		}
	}
	return entries
}

func buildMysteryEntry(hook StoryHook) CodexEntry {
	lines := []string{hook.Detail}
	if hook.SourceTurn > 0 {
		lines = append(lines, fmt.Sprintf("Opened on turn %d", hook.SourceTurn))
	}
	if hook.NPCName != "" {
		lines = append(lines, fmt.Sprintf("Involves: %s", hook.NPCName))
	}

	entry := CodexEntry{
		ID:       codexMysteryEntryID(hook.ID),
		Category: "mysteries",
		Title:    hook.Title,
		Subtitle: strings.Title(strings.ToLower(hook.Kind)),
		Summary:  hook.Detail,
		Sections: []CodexSection{{Title: "Open Question", Lines: compactLines(lines)}},
	}
	if hook.NPCName != "" {
		entry.Related = append(entry.Related, CodexLink{
			EntryID: codexNPCEntryID(hook.NPCName),
			Label:   hook.NPCName,
		})
	}
	return entry
}

func buildThreadEntryFromHook(hook StoryHook) CodexEntry {
	lines := []string{}
	if hook.Detail != "" {
		lines = append(lines, hook.Detail)
	}
	if hook.SourceTurn > 0 {
		lines = append(lines, fmt.Sprintf("Opened on turn %d", hook.SourceTurn))
	}
	if hook.UpdatedTurn > 0 && hook.UpdatedTurn != hook.SourceTurn {
		lines = append(lines, fmt.Sprintf("Last updated on turn %d", hook.UpdatedTurn))
	}
	if hook.Status != "" {
		lines = append(lines, "Status: "+hook.Status)
	}
	if hook.TimerTurns > 0 {
		lines = append(lines, fmt.Sprintf("Timer: %d turns", hook.TimerTurns))
	}

	entry := CodexEntry{
		ID:       codexThreadEntryID("hook", hook.ID),
		Category: "threads",
		Title:    hook.Title,
		Subtitle: strings.Title(strings.ToLower(hook.Kind)),
		Summary:  hook.Detail,
		Sections: []CodexSection{{Title: "Thread", Lines: compactLines(lines)}},
	}
	if hook.NPCName != "" {
		entry.Related = append(entry.Related, CodexLink{
			EntryID: codexNPCEntryID(hook.NPCName),
			Label:   hook.NPCName,
		})
	}
	return entry
}

func buildThreadEntryFromReaction(reaction WorldReaction) CodexEntry {
	lines := []string{}
	if reaction.Detail != "" {
		lines = append(lines, reaction.Detail)
	}
	if reaction.CreatedTurn > 0 {
		lines = append(lines, fmt.Sprintf("First surfaced on turn %d", reaction.CreatedTurn))
	}
	if reaction.SourceTurn > 0 {
		lines = append(lines, fmt.Sprintf("Triggered by turn %d", reaction.SourceTurn))
	}
	if reaction.Status != "" {
		lines = append(lines, "Status: "+reaction.Status)
	}
	return CodexEntry{
		ID:       codexThreadEntryID("reaction", reaction.ID),
		Category: "threads",
		Title:    reaction.Title,
		Subtitle: strings.Title(strings.ToLower(reaction.Kind)),
		Summary:  reaction.Detail,
		Sections: []CodexSection{{Title: "World Reaction", Lines: compactLines(lines)}},
	}
}

func enrichCodexLinks(index *CodexIndex, npcs []storage.NPC, hooks []StoryHook, reactions []WorldReaction, fronts []Front, chapters []storage.Chapter, messages []storage.ChatMessage, board InvestigationBoard, projects ProjectBoard) {
	if index == nil {
		return
	}

	for id, entry := range index.Entries {
		switch entry.Category {
		case "people":
			profile := findNPCNemesisProfileByName(npcs, entry.Title)
			for _, hook := range hooks {
				if entry.Title != "" && strings.EqualFold(hook.NPCName, entry.Title) {
					entry.Related = append(entry.Related, CodexLink{
						EntryID: codexThreadEntryID("hook", hook.ID),
						Label:   hook.Title,
					})
					if strings.EqualFold(hook.Kind, "mystery") || strings.EqualFold(hook.Kind, "rumor") {
						entry.Related = append(entry.Related, CodexLink{
							EntryID: codexMysteryEntryID(hook.ID),
							Label:   hook.Title,
						})
					}
				}
			}
			for _, reaction := range reactions {
				if codexReactionMentionsNPC(reaction, entry.Title) {
					entry.Related = append(entry.Related, CodexLink{
						EntryID: codexThreadEntryID("reaction", reaction.ID),
						Label:   reaction.Title,
					})
				}
			}
			if profile != nil {
				for _, front := range fronts {
					if nemesisProfileTouchesFront(profile, front) {
						entry.Related = append(entry.Related, CodexLink{
							EntryID: codexFrontEntryID(front.ID),
							Label:   frontDisplayTitle(front),
						})
					}
				}
				for _, linked := range index.Entries {
					if linked.Category != "places" {
						continue
					}
					if nemesisProfileTouchesLocation(profile, linked.Title) {
						entry.Related = append(entry.Related, CodexLink{
							EntryID: linked.ID,
							Label:   linked.Title,
						})
					}
				}
			}
		case "places":
			for _, message := range messages {
				if strings.Contains(strings.ToLower(message.Content), strings.ToLower(entry.Title)) {
					entry.Sections = append(entry.Sections, CodexSection{
						Title: "Recent Mention",
						Lines: []string{strings.TrimSpace(message.Content)},
					})
					break
				}
			}
			for _, front := range fronts {
				for _, pressure := range normalizeFrontPressures(front.Pressures) {
					if !strings.EqualFold(pressure.Region, entry.Title) {
						continue
					}
					entry.Related = append(entry.Related, CodexLink{
						EntryID: codexFrontEntryID(front.ID),
						Label:   frontDisplayTitle(front),
					})
				}
			}
			for _, npc := range npcs {
				profile := loadNemesisProfile(&npc)
				if profile == nil || !nemesisProfileTouchesLocation(profile, entry.Title) {
					continue
				}
				entry.Related = append(entry.Related, CodexLink{
					EntryID: codexNPCEntryID(npc.Name),
					Label:   npc.Name,
				})
			}
		case "factions":
			for _, front := range fronts {
				if !strings.EqualFold(front.Visibility, "known") || !strings.EqualFold(front.Faction, entry.Title) {
					continue
				}
				entry.Related = append(entry.Related, CodexLink{
					EntryID: codexFrontEntryID(front.ID),
					Label:   frontDisplayTitle(front),
				})
			}
		case "fronts":
			for _, front := range fronts {
				if !strings.EqualFold(codexFrontEntryID(front.ID), id) {
					continue
				}
				if _, ok := index.Entries[codexThreadEntryID("hook", "front-hook:"+front.ID)]; ok {
					entry.Related = append(entry.Related, CodexLink{
						EntryID: codexThreadEntryID("hook", "front-hook:"+front.ID),
						Label:   "Front thread",
					})
				}
				for _, pressure := range normalizeFrontPressures(front.Pressures) {
					reactionEntryID := codexThreadEntryID("reaction", "front-pressure:"+front.ID+":"+slugKey(pressure.Region)+":"+slugKey(pressure.Kind))
					if _, ok := index.Entries[reactionEntryID]; ok {
						entry.Related = append(entry.Related, CodexLink{
							EntryID: reactionEntryID,
							Label:   pressure.Region + " pressure",
						})
					}
				}
				for _, npc := range npcs {
					profile := loadNemesisProfile(&npc)
					if profile == nil || !nemesisProfileTouchesFront(profile, front) {
						continue
					}
					entry.Related = append(entry.Related, CodexLink{
						EntryID: codexNPCEntryID(npc.Name),
						Label:   npc.Name,
					})
				}
			}
		}
		if entry.Category != "investigations" {
			for _, invCase := range board.Cases {
				if !investigationCaseTouchesCodexEntry(invCase, id, entry) {
					continue
				}
				entry.Related = append(entry.Related, CodexLink{
					EntryID: codexInvestigationEntryID(invCase.ID),
					Label:   invCase.Title,
				})
			}
		}
		if entry.Category != "projects" {
			for _, project := range projects.Projects {
				if !projectTouchesCodexEntry(project, id, entry) {
					continue
				}
				entry.Related = append(entry.Related, CodexLink{
					EntryID: codexProjectEntryID(project.ID),
					Label:   project.Title,
				})
			}
		}
		index.Entries[id] = entry
	}

	if protagonist, ok := index.Entries[ProtagonistCodexEntryID()]; ok {
		for _, chapter := range chapters {
			if strings.TrimSpace(chapter.Title) == "" && strings.TrimSpace(chapter.Summary) == "" {
				continue
			}
			line := chapter.Title
			if strings.TrimSpace(chapter.Summary) != "" {
				line += " — " + strings.TrimSpace(chapter.Summary)
			}
			protagonist.Sections = append(protagonist.Sections, CodexSection{
				Title: "Recent Chapters",
				Lines: []string{line},
			})
		}
		index.Entries[ProtagonistCodexEntryID()] = protagonist
	}

	_ = npcs
	_ = reactions
}

func codexNPCEntryID(name string) string {
	return "people:npc:" + strings.ToLower(strings.TrimSpace(name))
}

func codexLocationEntryID(name string) string {
	return "places:" + strings.ToLower(strings.TrimSpace(name))
}

func codexFactionEntryID(name string) string {
	return "factions:" + strings.ToLower(strings.TrimSpace(name))
}

func codexFrontEntryID(id string) string {
	return "fronts:" + strings.TrimSpace(id)
}

func codexMysteryEntryID(id string) string {
	return "mysteries:" + strings.TrimSpace(id)
}

func codexThreadEntryID(kind, id string) string {
	return "threads:" + strings.TrimSpace(kind) + ":" + strings.TrimSpace(id)
}

func codexInvestigationEntryID(id string) string {
	return "investigations:" + strings.TrimSpace(id)
}

func parseLooseJSONMap(raw string) map[string]interface{} {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "null" {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

func parseJSONStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "null" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func parseLooseJSONStringSlice(raw interface{}) []string {
	values, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func mergeStringLists(groups ...[]string) []string {
	var out []string
	seen := map[string]bool{}
	for _, group := range groups {
		for _, item := range group {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			key := strings.ToLower(item)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, item)
		}
	}
	return out
}

func formatStringSliceLines(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			lines = append(lines, item)
		}
	}
	return lines
}

func buildProtagonistInvestigationLines(board InvestigationBoard) []string {
	lines := []string{}
	for _, invCase := range board.Cases {
		if strings.EqualFold(invCase.Status, "solved") {
			continue
		}
		line := invCase.Title
		if len(invCase.Clues) > 0 {
			line += fmt.Sprintf(" · clues %d", len(invCase.Clues))
		}
		if len(invCase.Contradictions) > 0 {
			line += fmt.Sprintf(" · contradictions %d", len(invCase.Contradictions))
		}
		if len(invCase.Theories) > 0 {
			line += fmt.Sprintf(" · theories %d", len(invCase.Theories))
		}
		lines = append(lines, line)
	}
	return lines
}

func formatInvestigationClueLines(items []InvestigationClue) []string {
	lines := []string{}
	for _, item := range items {
		line := item.Label
		if item.Detail != "" {
			line += " — " + item.Detail
		}
		if item.Status != "" && !strings.EqualFold(item.Status, "known") {
			line += fmt.Sprintf(" [%s]", item.Status)
		}
		lines = append(lines, line)
	}
	return lines
}

func formatInvestigationSuspectLines(items []InvestigationSuspect) []string {
	lines := []string{}
	for _, item := range items {
		line := item.Name
		if item.Detail != "" {
			line += " — " + item.Detail
		}
		if item.Status != "" {
			line += fmt.Sprintf(" [%s]", item.Status)
		}
		lines = append(lines, line)
	}
	return lines
}

func formatInvestigationClaimLines(items []InvestigationClaim) []string {
	lines := []string{}
	for _, item := range items {
		line := item.Statement
		meta := []string{}
		if item.Confidence != "" {
			meta = append(meta, item.Confidence)
		}
		if item.Status != "" {
			meta = append(meta, item.Status)
		}
		if len(meta) > 0 {
			line += " [" + strings.Join(meta, ", ") + "]"
		}
		lines = append(lines, line)
	}
	return lines
}

func formatInvestigationContradictionLines(items []InvestigationContradiction) []string {
	lines := []string{}
	for _, item := range items {
		line := item.Label
		if item.Detail != "" {
			line += " — " + item.Detail
		}
		if item.Status != "" {
			line += fmt.Sprintf(" [%s]", item.Status)
		}
		lines = append(lines, line)
	}
	return lines
}

func formatInvestigationLeadLines(items []InvestigationLead) []string {
	lines := []string{}
	for _, item := range items {
		line := item.Title
		if item.Detail != "" {
			line += " — " + item.Detail
		}
		if item.Status != "" {
			line += fmt.Sprintf(" [%s]", item.Status)
		}
		lines = append(lines, line)
	}
	return lines
}

func formatInvestigationTheoryLines(items []InvestigationTheory) []string {
	lines := []string{}
	for _, item := range items {
		line := item.Statement
		meta := []string{}
		if item.Confidence != "" {
			meta = append(meta, item.Confidence)
		}
		if item.Status != "" {
			meta = append(meta, item.Status)
		}
		if len(meta) > 0 {
			line += " [" + strings.Join(meta, ", ") + "]"
		}
		lines = append(lines, line)
	}
	return lines
}

func investigationCaseCodexLinks(invCase InvestigationCase) []CodexLink {
	links := []CodexLink{}
	appendLinks := func(items []InvestigationLink) {
		for _, link := range items {
			if codexLink, ok := codexLinkFromInvestigationLink(link); ok {
				links = append(links, codexLink)
			}
		}
	}

	appendLinks(invCase.Links)
	for _, item := range invCase.Clues {
		appendLinks(item.Links)
	}
	for _, item := range invCase.Suspects {
		appendLinks(item.Links)
	}
	for _, item := range invCase.Claims {
		appendLinks(item.Links)
	}
	for _, item := range invCase.Contradictions {
		appendLinks(item.Links)
	}
	for _, item := range invCase.Leads {
		appendLinks(item.Links)
	}
	for _, item := range invCase.Theories {
		appendLinks(item.Links)
	}
	return appendUniqueLinks(links)
}

func codexLinkFromInvestigationLink(link InvestigationLink) (CodexLink, bool) {
	kind := strings.ToLower(strings.TrimSpace(link.Kind))
	refID := strings.TrimSpace(link.RefID)
	label := strings.TrimSpace(link.Label)
	if refID == "" && label == "" {
		return CodexLink{}, false
	}
	switch {
	case strings.Contains(refID, ":"):
		return CodexLink{EntryID: refID, Label: firstNonEmpty(label, refID)}, true
	case kind == "npc":
		return CodexLink{EntryID: codexNPCEntryID(firstNonEmpty(label, refID)), Label: firstNonEmpty(label, refID)}, true
	case kind == "place" || kind == "location":
		return CodexLink{EntryID: codexLocationEntryID(firstNonEmpty(label, refID)), Label: firstNonEmpty(label, refID)}, true
	case kind == "front":
		return CodexLink{EntryID: codexFrontEntryID(firstNonEmpty(refID, label)), Label: firstNonEmpty(label, refID)}, true
	case kind == "hook":
		return CodexLink{EntryID: codexThreadEntryID("hook", refID), Label: firstNonEmpty(label, refID)}, true
	case kind == "reaction":
		return CodexLink{EntryID: codexThreadEntryID("reaction", refID), Label: firstNonEmpty(label, refID)}, true
	case kind == "faction":
		return CodexLink{EntryID: codexFactionEntryID(firstNonEmpty(label, refID)), Label: firstNonEmpty(label, refID)}, true
	default:
		return CodexLink{}, false
	}
}

func investigationCaseTouchesCodexEntry(invCase InvestigationCase, entryID string, entry CodexEntry) bool {
	for _, link := range investigationCaseCodexLinks(invCase) {
		if strings.EqualFold(link.EntryID, entryID) {
			return true
		}
	}
	if entry.Category == "people" {
		for _, suspect := range invCase.Suspects {
			if strings.EqualFold(strings.TrimSpace(suspect.Name), strings.TrimSpace(entry.Title)) {
				return true
			}
		}
	}
	return false
}

func buildProtagonistNemesisLines(npcs []storage.NPC) []string {
	lines := []string{}
	for _, npc := range npcs {
		profile := loadNemesisProfile(&npc)
		if profile == nil || profile.Status == NemesisStatusResolved {
			continue
		}
		line := fmt.Sprintf("%s — %s", npc.Name, nemesisStatusLabel(profile.Status))
		if profile.EscalationTier > 0 {
			line += fmt.Sprintf(" · Tier %d", profile.EscalationTier)
		}
		if agenda := strings.TrimSpace(nemesisAgendaHint(profile.ThreatPosture)); agenda != "" {
			line += " · " + agenda
		}
		if npc.LastSeenTurn > 0 {
			line += fmt.Sprintf(" · Last seen turn %d", npc.LastSeenTurn)
		}
		lines = append(lines, line)
	}
	return lines
}

func buildNPCNemesisCodexSection(npc *storage.NPC) CodexSection {
	profile := loadNemesisProfile(npc)
	if profile == nil {
		return CodexSection{}
	}

	lines := []string{
		"Status: " + nemesisStatusLabel(profile.Status),
		fmt.Sprintf("Escalation: Tier %d", maxInt(1, profile.EscalationTier)),
	}
	if agenda := strings.TrimSpace(nemesisAgendaHint(profile.ThreatPosture)); agenda != "" {
		lines = append(lines, "Suspected agenda: "+agenda)
	}
	if outcome := strings.TrimSpace(profile.LastOutcome); outcome != "" {
		lines = append(lines, "Last outcome: "+outcome)
	}
	if len(profile.VisibleScars) > 0 {
		lines = append(lines, "Visible scars: "+strings.Join(profile.VisibleScars, "; "))
	}

	for _, trace := range nemesisEscalationTrace(profile.EventHistory) {
		lines = append(lines, trace)
	}

	title := "Rivalry Trace"
	if profile.Status == NemesisStatusActive {
		title = "Nemesis Trail"
	}
	return CodexSection{Title: title, Lines: compactLines(lines)}
}

func findNPCNemesisProfileByName(npcs []storage.NPC, name string) *NemesisProfile {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	for i := range npcs {
		if strings.EqualFold(strings.TrimSpace(npcs[i].Name), name) {
			return loadNemesisProfile(&npcs[i])
		}
	}
	return nil
}

func codexReactionMentionsNPC(reaction WorldReaction, npcName string) bool {
	npcName = strings.ToLower(strings.TrimSpace(npcName))
	if npcName == "" {
		return false
	}
	return strings.Contains(strings.ToLower(reaction.Title), npcName) ||
		strings.Contains(strings.ToLower(reaction.Detail), npcName)
}

func nemesisStatusLabel(status NemesisStatus) string {
	switch status {
	case NemesisStatusActive:
		return "Active nemesis"
	case NemesisStatusResolved:
		return "Resolved rival"
	default:
		return "Recurring rival"
	}
}

func nemesisAgendaHint(posture string) string {
	switch strings.ToLower(strings.TrimSpace(posture)) {
	case "hunting", "vengeful":
		return "Direct revenge looks likely."
	case "political":
		return "Pressure through allies, rumor, or institutions seems likely."
	case "obsessive":
		return "They keep circling the same grievance."
	case "watching":
		return "They have not dropped the grudge."
	default:
		return ""
	}
}

func nemesisEscalationTrace(events []NemesisEvent) []string {
	if len(events) == 0 {
		return nil
	}
	start := maxInt(0, len(events)-3)
	lines := make([]string, 0, len(events)-start)
	for _, event := range events[start:] {
		label := strings.Title(strings.ReplaceAll(strings.TrimSpace(event.Kind), "_", " "))
		line := label
		if event.Turn > 0 {
			line = fmt.Sprintf("Turn %d: %s", event.Turn, label)
		}
		if detail := strings.TrimSpace(event.Detail); detail != "" {
			line += " — " + detail
		}
		lines = append(lines, line)
	}
	return lines
}

func nemesisProfileTouchesLocation(profile *NemesisProfile, location string) bool {
	location = strings.ToLower(strings.TrimSpace(location))
	if profile == nil || location == "" {
		return false
	}
	return strings.Contains(nemesisProfileFootprint(profile), location)
}

func nemesisProfileTouchesFront(profile *NemesisProfile, front Front) bool {
	if profile == nil {
		return false
	}
	footprint := nemesisProfileFootprint(profile)
	if footprint == "" {
		return false
	}
	terms := []string{
		frontDisplayTitle(front),
		strings.TrimSpace(front.Faction),
	}
	for _, pressure := range normalizeFrontPressures(front.Pressures) {
		terms = append(terms, pressure.Region)
	}
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term != "" && strings.Contains(footprint, term) {
			return true
		}
	}
	return false
}

func nemesisProfileFootprint(profile *NemesisProfile) string {
	if profile == nil {
		return ""
	}
	parts := []string{profile.LastOutcome}
	parts = append(parts, profile.VisibleScars...)
	for _, event := range profile.EventHistory {
		parts = append(parts, event.Detail, event.Outcome)
	}
	return strings.ToLower(strings.Join(parts, " | "))
}

func formatSkillsLines(char *storage.Character, stats map[string]interface{}) []string {
	skills := map[string]interface{}{}
	if char != nil && strings.TrimSpace(char.SkillsJSON) != "" && strings.TrimSpace(char.SkillsJSON) != "{}" {
		_ = json.Unmarshal([]byte(char.SkillsJSON), &skills)
	}
	if fromStats, ok := stats["skills"].(map[string]interface{}); ok {
		for key, value := range fromStats {
			skills[key] = value
		}
	}
	if len(skills) == 0 {
		return nil
	}

	keys := make([]string, 0, len(skills))
	for key := range skills {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		level := 1
		xp := 0
		if payload, ok := skills[key].(map[string]interface{}); ok {
			level = codexToInt(payload["level"])
			if level < 1 {
				level = 1
			}
			xp = codexToInt(payload["xp"])
		}
		lines = append(lines, fmt.Sprintf("%s — level %d, %d XP", key, level, xp))
	}
	return lines
}

func formatStatGroupLines(rawStats interface{}, rawSchema interface{}) []string {
	statsMap, ok := rawStats.(map[string]interface{})
	if !ok || len(statsMap) == 0 {
		return nil
	}

	labelMap := map[string]string{}
	order := []string{}
	if defs, ok := rawSchema.([]interface{}); ok {
		for _, rawDef := range defs {
			def, ok := rawDef.(map[string]interface{})
			if !ok {
				continue
			}
			key := firstNonEmptyString(def["key"])
			if key == "" {
				continue
			}
			order = append(order, key)
			labelMap[key] = firstNonEmptyString(def["label"])
		}
	}
	if len(order) == 0 {
		for key := range statsMap {
			order = append(order, key)
		}
		sort.Strings(order)
	}

	lines := make([]string, 0, len(order))
	for _, key := range order {
		label := labelMap[key]
		if label == "" {
			label = strings.Title(strings.ReplaceAll(key, "_", " "))
		}
		switch payload := statsMap[key].(type) {
		case map[string]interface{}:
			current := codexToInt(payload["current"])
			max := codexToInt(payload["max"])
			if max > 0 {
				lines = append(lines, fmt.Sprintf("%s: %d/%d", label, current, max))
			} else {
				lines = append(lines, fmt.Sprintf("%s: %d", label, current))
			}
		default:
			lines = append(lines, fmt.Sprintf("%s: %d", label, codexToInt(payload)))
		}
	}
	return lines
}

func compactCodexSections(sections []CodexSection) []CodexSection {
	if len(sections) == 0 {
		return nil
	}
	out := make([]CodexSection, 0, len(sections))
	for _, section := range sections {
		section.Title = strings.TrimSpace(section.Title)
		section.Lines = compactLines(section.Lines)
		if section.Title == "" || len(section.Lines) == 0 {
			continue
		}
		out = append(out, section)
	}
	return out
}

func compactLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, 0, len(lines))
	seen := map[string]bool{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key := strings.ToLower(line)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, line)
	}
	return out
}

func appendUniqueLinks(links []CodexLink) []CodexLink {
	if len(links) == 0 {
		return nil
	}
	out := make([]CodexLink, 0, len(links))
	seen := map[string]bool{}
	for _, link := range links {
		link.EntryID = strings.TrimSpace(link.EntryID)
		link.Label = strings.TrimSpace(link.Label)
		if link.EntryID == "" || link.Label == "" {
			continue
		}
		key := link.EntryID + "|" + strings.ToLower(link.Label)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, link)
	}
	return out
}

func firstNonEmptyString(values ...interface{}) string {
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func codexToInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case int32:
		return int(n)
	}
	return 0
}

func codexRelationshipAxesSummary(raw string) string {
	axes := loadRelationshipAxes(&storage.NPC{RelationshipJSON: raw})
	return fmt.Sprintf("trust %d · fear %d · debt %d · respect %d · intimacy %d",
		axes.Trust, axes.Fear, axes.Debt, axes.Respect, axes.Intimacy)
}

func valueOrEmpty[T any](value *T, pick func(*T) string) string {
	if value == nil {
		return ""
	}
	return pick(value)
}

func splitBulletLines(block string) []string {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil
	}
	rawLines := strings.Split(block, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(strings.TrimLeft(line, "-*•"))
		if line != "" && !strings.HasPrefix(strings.ToLower(line), "private thoughts") {
			lines = append(lines, line)
		}
	}
	return lines
}
