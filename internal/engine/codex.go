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

	if entry := buildProtagonistCodexEntry(story, char, world, achievements, npcs); entry.Title != "" {
		addEntry(entry)
	}

	for _, npc := range npcs {
		addEntry(buildNPCCodexEntry(&npc))
	}

	for _, loc := range parseKnownLocations(valueOrEmpty(world, func(w *storage.WorldState) string { return w.KnownLocationsJSON })) {
		addEntry(buildLocationCodexEntry(loc, world))
	}

	for _, faction := range parseFactionEntries(story.SettingJSON) {
		addEntry(faction)
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
	addCategory("mysteries", "Mysteries")
	addCategory("threads", "Threads")

	enrichCodexLinks(index, npcs, hooks, reactions, chapters, messages)
	return index, nil
}

func buildProtagonistCodexEntry(story *storage.Story, char *storage.Character, world *storage.WorldState, achievements []storage.Achievement, npcs []storage.NPC) CodexEntry {
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

func buildLocationCodexEntry(loc KnownLocation, world *storage.WorldState) CodexEntry {
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

	return CodexEntry{
		ID:       codexLocationEntryID(loc.Name),
		Category: "places",
		Title:    loc.Name,
		Subtitle: strings.TrimSpace(loc.Region),
		Summary:  strings.TrimSpace(loc.Description),
		Sections: []CodexSection{{Title: "Overview", Lines: lines}},
	}
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

func enrichCodexLinks(index *CodexIndex, npcs []storage.NPC, hooks []StoryHook, reactions []WorldReaction, chapters []storage.Chapter, messages []storage.ChatMessage) {
	if index == nil {
		return
	}

	for id, entry := range index.Entries {
		switch entry.Category {
		case "people":
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

func codexMysteryEntryID(id string) string {
	return "mysteries:" + strings.TrimSpace(id)
}

func codexThreadEntryID(kind, id string) string {
	return "threads:" + strings.TrimSpace(kind) + ":" + strings.TrimSpace(id)
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
