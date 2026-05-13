package engine

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/storage"
)

// ContextConfig controls how the context builder assembles prompts.
type ContextConfig struct {
	RecentMessageCount int      // how many recent messages to include (default 20)
	RAGChunks          []string // placeholder for Phase 5 RAG — empty for now
	NPCLookbackTurns   int      // how many turns back to load NPCs for context (default 20)
}

// DefaultContextConfig returns sensible defaults.
func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		RecentMessageCount: 20,
		RAGChunks:          nil,
		NPCLookbackTurns:   20,
	}
}

// BuildContext assembles the full message list for an AI call.
// Order: [system prompt, state summary, optional RAG, ...recent messages, current user input]
// npcs is the list of recently-seen NPCs; their data is injected into the system prompt and state summary.
// lastChapterSummary is the AI-generated summary of the previous chapter (empty if chapter 1).
// earnedAchievements is the list of already-earned achievements (name + category) to prevent duplicates.
func BuildContext(
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	npcs []storage.NPC,
	recentMessages []storage.ChatMessage,
	ragChunks []string,
	lastChapterSummary string,
	currentInput string,
	earnedAchievements []storage.Achievement,
	sceneProgression *SceneProgressionGuidance,
) []ai.Message {
	msgs := make([]ai.Message, 0, len(recentMessages)+5)

	// Build NPC context string from the provided NPC list.
	var npcsContext string
	if len(npcs) > 0 {
		var npcParts []string
		for i := range npcs {
			npcParts = append(npcParts, FormatNPCForContext(&npcs[i]))
		}
		npcsContext = strings.Join(npcParts, "\n---\n")
	}

	// 1. Build system prompt using the NarratorSystem prompt builder.
	// Include genre and tone from the story model so the narrator is properly
	// calibrated even when they are not embedded in the setting JSON.
	systemPrompt := prompts.NarratorSystem(
		story.Name,
		story.Genre,
		story.Tone,
		story.Language,
		story.WritingStyle,
		story.PromptDirectives,
		story.SettingJSON,
		story.StatsSchemaJSON,
		char.Name,
		char.Background,
		char.StatsJSON,
		npcsContext,
	)
	msgs = append(msgs, ai.Message{
		Role:    ai.RoleSystem,
		Content: systemPrompt,
	})

	// 2. Add a live state summary — reflects current state, not creation-time state.
	stateSummary := buildStateSummary(char, world, npcs, lastChapterSummary)
	msgs = append(msgs, ai.Message{
		Role:    ai.RoleSystem,
		Content: stateSummary,
	})

	if guidanceSummary := buildPlayerGuidanceSummary(world); guidanceSummary != "" {
		msgs = append(msgs, ai.Message{
			Role:    ai.RoleSystem,
			Content: guidanceSummary,
		})
	}

	if momentumSummary := buildNarrativeMomentumSummary(world, recentMessages, sceneProgression); momentumSummary != "" {
		msgs = append(msgs, ai.Message{
			Role:    ai.RoleSystem,
			Content: momentumSummary,
		})
	}

	if freeActionSummary := buildFreeActionInterpretationSummary(story, world, currentInput); freeActionSummary != "" {
		msgs = append(msgs, ai.Message{
			Role:    ai.RoleSystem,
			Content: freeActionSummary,
		})
	}

	// 3. Inject RAG chunks if provided (long-term memory from past turns).
	if len(ragChunks) > 0 {
		ragContent := "## Relevant Memory\n" + strings.Join(ragChunks, "\n---\n")
		msgs = append(msgs, ai.Message{
			Role:    ai.RoleSystem,
			Content: ragContent,
		})
	}

	// 4. Inject previously earned achievements so the AI avoids duplicates.
	if len(earnedAchievements) > 0 {
		achContent := "## Previously Earned Achievements\nDo NOT award any achievement with these names (already earned):\n"
		for _, a := range earnedAchievements {
			cat := strings.ToLower(a.Category)
			if cat == "" {
				cat = "story"
			}
			achContent += fmt.Sprintf("- \"%s\" (%s)\n", a.Name, cat)
		}
		msgs = append(msgs, ai.Message{
			Role:    ai.RoleSystem,
			Content: achContent,
		})
	}

	// 5. Convert recent DB messages to ai.Message.
	for _, m := range recentMessages {
		role := ai.RoleUser
		if m.Role == "assistant" {
			role = ai.RoleAssistant
		}
		msgs = append(msgs, ai.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	// 6. Append the current user input as the final message.
	msgs = append(msgs, ai.Message{
		Role:    ai.RoleUser,
		Content: currentInput,
	})

	return msgs
}

// buildStateSummary composes a concise state message with current character and world info.
// npcs is the list of NPCs included in the current context (used for the summary line).
// lastChapterSummary is the AI-generated summary of the previous chapter (empty if chapter 1).
func buildStateSummary(char *storage.Character, world *storage.WorldState, npcs []storage.NPC, lastChapterSummary string) string {
	var sb strings.Builder
	timeline := loadCharacterTimeline(world)
	sb.WriteString("## Current Game State\n")
	sb.WriteString(fmt.Sprintf("- Chapter: %d\n", world.CurrentChapter))
	sb.WriteString(fmt.Sprintf("- Turn: %d\n", world.CurrentTurn))
	sb.WriteString(fmt.Sprintf("- Location: %s\n", world.CurrentLocation))
	timelineSummary := formatCharacterTimelineSummary(timeline)
	recentTimelineMilestones := formatRecentTimelineMilestones(timeline, 2)
	if timelineSummary != "" {
		sb.WriteString(fmt.Sprintf("- Timeline: %s\n", timelineSummary))
	} else if recentTimelineMilestones != "" {
		sb.WriteString("- Timeline: unresolved\n")
	}
	if recentTimelineMilestones != "" {
		sb.WriteString(fmt.Sprintf("- Recent Milestones: %s\n", recentTimelineMilestones))
	}
	sb.WriteString(fmt.Sprintf("- Character: %s\n", char.Name))
	sb.WriteString(fmt.Sprintf("- Stats (live): %s\n", char.StatsJSON))
	if char.TraitsJSON != "" && char.TraitsJSON != "null" && char.TraitsJSON != "[]" {
		sb.WriteString(fmt.Sprintf("- Traits: %s\n", char.TraitsJSON))
	}
	if len(npcs) > 0 {
		names := make([]string, len(npcs))
		var nemeses []string
		for i, npc := range npcs {
			names[i] = npc.Name
			if profile := loadNemesisProfile(&npc); profile != nil && profile.Status == NemesisStatusActive {
				nemeses = append(nemeses, fmt.Sprintf("%s(tier %d)", npc.Name, profile.EscalationTier))
			}
		}
		sb.WriteString(fmt.Sprintf("- Known NPCs: %d (%s)\n", len(npcs), strings.Join(names, ", ")))
		if len(nemeses) > 0 {
			sb.WriteString(fmt.Sprintf("- Active Nemeses: %s\n", strings.Join(nemeses, ", ")))
		}
	}
	// World state: known locations.
	if world.KnownLocationsJSON != "" && world.KnownLocationsJSON != "null" && world.KnownLocationsJSON != "[]" {
		sb.WriteString(fmt.Sprintf("- Known Locations: %s\n", world.KnownLocationsJSON))
	}
	// World state: faction standings.
	if world.FactionStandingsJSON != "" && world.FactionStandingsJSON != "null" && world.FactionStandingsJSON != "{}" {
		sb.WriteString(fmt.Sprintf("- Faction Standings: %s\n", world.FactionStandingsJSON))
	}
	// World state: global events.
	if world.GlobalEventsJSON != "" && world.GlobalEventsJSON != "null" && world.GlobalEventsJSON != "[]" {
		sb.WriteString(fmt.Sprintf("- Global Events: %s\n", world.GlobalEventsJSON))
	}
	if hooks := activeStoryHooks(loadStoryHooks(world)); len(hooks) > 0 {
		parts := make([]string, 0, len(hooks))
		for _, hook := range hooks {
			line := hook.Title
			if hook.TimerTurns > 0 {
				line += fmt.Sprintf(" [timer:%d]", hook.TimerTurns)
			}
			if hook.NPCName != "" {
				line += " {" + hook.NPCName + "}"
			}
			parts = append(parts, line)
		}
		sb.WriteString(fmt.Sprintf("- Open Hooks: %s\n", strings.Join(parts, " | ")))
	}
	if reactions := visibleWorldReactions(loadWorldReactions(world)); len(reactions) > 0 {
		parts := make([]string, 0, len(reactions))
		for _, reaction := range reactions {
			parts = append(parts, reaction.Title)
		}
		sb.WriteString(fmt.Sprintf("- World Reactions: %s\n", strings.Join(parts, " | ")))
	}
	if board := loadInvestigationBoard(world); len(board.Cases) > 0 {
		parts := make([]string, 0, len(board.Cases))
		for _, invCase := range board.Cases {
			if strings.EqualFold(invCase.Status, "solved") {
				continue
			}
			line := invCase.Title
			if len(invCase.Clues) > 0 {
				line += fmt.Sprintf(" [clues:%d]", len(invCase.Clues))
			}
			if len(invCase.Contradictions) > 0 {
				line += fmt.Sprintf(" [contradictions:%d]", len(invCase.Contradictions))
			}
			if len(invCase.Theories) > 0 {
				line += fmt.Sprintf(" [theories:%d]", len(invCase.Theories))
			}
			parts = append(parts, line)
		}
		if len(parts) > 0 {
			sb.WriteString(fmt.Sprintf("- Investigations: %s\n", strings.Join(parts, " | ")))
		}
	}
	if projectBoard := loadProjectBoard(world); len(projectBoard.Projects) > 0 {
		if active, completed := buildProjectStateSummaryLines(projectBoard); len(active) > 0 || len(completed) > 0 {
			if len(active) > 0 {
				sb.WriteString(fmt.Sprintf("- Projects: %s\n", strings.Join(active, " | ")))
			}
			if len(completed) > 0 {
				sb.WriteString(fmt.Sprintf("- Completed Projects: %s\n", strings.Join(completed, " | ")))
			}
		}
	}
	if fronts := knownFronts(loadFronts(world)); len(fronts) > 0 {
		parts := make([]string, 0, len(fronts))
		for _, front := range fronts {
			parts = append(parts, formatKnownFrontSummary(front))
		}
		sb.WriteString(fmt.Sprintf("- Active Fronts: %s\n", strings.Join(parts, " | ")))
	}
	// Previous chapter summary for narrative continuity.
	if lastChapterSummary != "" {
		sb.WriteString(fmt.Sprintf("- Previous Chapter Summary: %s\n", lastChapterSummary))
	}
	return sb.String()
}

func buildPlayerGuidanceSummary(world *storage.WorldState) string {
	guidance := activePlayerGuidance(loadPlayerGuidance(world))
	if len(guidance) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Player Guidance\n")
	sb.WriteString("These are soft future-facing wishes from the player. Use them when they fit naturally. Do not force them immediately, and do not mention this meta guidance explicitly in the narration.\n")
	for _, item := range guidance {
		line := fmt.Sprintf("- [%s/%s] %s", item.Scope, item.Priority, item.Title)
		if item.Kind != "" {
			line += " {" + item.Kind + "}"
		}
		if item.Status != "" {
			line += " [" + item.Status + "]"
		}
		if item.Detail != "" {
			line += " — " + item.Detail
		}
		if item.Progress != "" {
			line += " | progress: " + item.Progress
		}
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

type recentAssistantBeat struct {
	location    string
	terms       []string
	choiceTerms map[string]struct{}
}

type narrativeMomentumSignal struct {
	recentTurns        int
	sameLocation       bool
	lowWorldPressure   bool
	similarChoicePairs int
	repeatedTerms      []string
}

func buildNarrativeMomentumSummary(world *storage.WorldState, recentMessages []storage.ChatMessage, sceneProgression *SceneProgressionGuidance) string {
	signal := detectNarrativeMomentumSignal(world, recentMessages)
	if signal == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Narrative Momentum\n")
	if sceneProgression != nil {
		sb.WriteString("Recent turns may be drifting. Use this scene progression directive for the NEXT response.\n")
	} else {
		sb.WriteString("Recent turns are circling the same micro-beat. Treat this as a high-priority correction for the NEXT response.\n")
	}
	if signal.sameLocation {
		sb.WriteString("- The scene has lingered in the same location across recent turns.\n")
	}
	if signal.lowWorldPressure {
		sb.WriteString("- The live world state currently lacks strong external pressure.\n")
	}
	if len(signal.repeatedTerms) > 0 {
		sb.WriteString(fmt.Sprintf("- Repeated motifs to stop recycling: %s\n", strings.Join(signal.repeatedTerms, ", ")))
	}
	if signal.similarChoicePairs > 0 {
		sb.WriteString(fmt.Sprintf("- Similar choice families repeated across %d recent turn pairs.\n", signal.similarChoicePairs))
	}
	if sceneProgression != nil {
		sb.WriteString(fmt.Sprintf("- Scene judge assessment: %s.\n", sceneProgression.Assessment))
		sb.WriteString(fmt.Sprintf("- Preferred strategy: %s.\n", sceneProgression.Strategy))
		sb.WriteString("- Reason: " + sceneProgression.Reason + "\n")
		sb.WriteString("- Apply now: " + sceneProgression.Instruction + "\n")
		if sceneProgression.Strategy == sceneProgressionStrategyTimeSkip {
			if sceneProgression.TimeSkipLabel != "" {
				sb.WriteString("- Time skip target: " + sceneProgression.TimeSkipLabel + "\n")
			}
			if sceneProgression.TimeSkipDetail != "" {
				sb.WriteString("- Time skip continuity: " + sceneProgression.TimeSkipDetail + "\n")
			}
			sb.WriteString("- If you time skip, jump directly to the next meaningful age, milestone, or changed situation. Do not play filler turns in between.\n")
		}
	} else {
		sb.WriteString("- Introduce one concrete change immediately: interruption, arrival, discovery, reveal, cost, countdown, hook, world reaction, project beat, or location shift.\n")
	}
	sb.WriteString("- Do NOT offer near-identical choices to the last turns.\n")
	sb.WriteString("- If the scene stays in the same place, materially change stakes, relationships, resources, or available information.\n")
	sb.WriteString("- Prefer 2-4 choices that open genuinely different directions instead of rephrasing the same action.\n")
	if signal.lowWorldPressure {
		sb.WriteString("- Seed at least one durable thread when it fits: open hook, visible world reaction, front clue, investigation lead, or project progress.\n")
	}
	return sb.String()
}

func detectNarrativeMomentumSignal(world *storage.WorldState, recentMessages []storage.ChatMessage) *narrativeMomentumSignal {
	beats := extractRecentAssistantBeats(recentMessages, 4)
	if len(beats) < 3 {
		return nil
	}

	repeatedTerms := repeatedBeatTerms(beats)
	similarChoicePairs := countSimilarChoicePairs(beats)
	if len(repeatedTerms) < 2 && similarChoicePairs < 2 {
		return nil
	}

	sameLocation := sameRecentLocation(beats)
	lowWorldPressure := worldHasLowNarrativePressure(world)
	if !sameLocation && !lowWorldPressure && len(repeatedTerms) < 3 {
		return nil
	}

	return &narrativeMomentumSignal{
		recentTurns:        len(beats),
		sameLocation:       sameLocation,
		lowWorldPressure:   lowWorldPressure,
		similarChoicePairs: similarChoicePairs,
		repeatedTerms:      repeatedTerms,
	}
}

func extractRecentAssistantBeats(recentMessages []storage.ChatMessage, limit int) []recentAssistantBeat {
	if limit <= 0 {
		limit = 4
	}

	beats := make([]recentAssistantBeat, 0, limit)
	for i := len(recentMessages) - 1; i >= 0 && len(beats) < limit; i-- {
		msg := recentMessages[i]
		if !strings.EqualFold(msg.Role, "assistant") {
			continue
		}

		beat, ok := buildRecentAssistantBeat(msg)
		if !ok {
			continue
		}
		beats = append(beats, beat)
	}

	for i, j := 0, len(beats)-1; i < j; i, j = i+1, j-1 {
		beats[i], beats[j] = beats[j], beats[i]
	}
	return beats
}

func buildRecentAssistantBeat(msg storage.ChatMessage) (recentAssistantBeat, bool) {
	var beat recentAssistantBeat

	narrative := normalizeNarrativeText(msg.Content)
	var choiceTexts []string
	if meta, ok := parsePersistedAssistantMeta(msg.MetadataJSON); ok {
		if meta.Output != nil {
			if text := normalizeNarrativeText(meta.Output.Narrative); text != "" {
				narrative = text
			}
			beat.location = firstNonEmpty(meta.Output.Location, meta.Location)
			if len(meta.Output.ChoicesData) > 0 {
				for _, choice := range meta.Output.ChoicesData {
					if text := strings.TrimSpace(choice.Text); text != "" {
						choiceTexts = append(choiceTexts, text)
					}
				}
			} else if len(meta.Output.Choices) > 0 {
				choiceTexts = append(choiceTexts, meta.Output.Choices...)
			}
		}
		if beat.location == "" {
			beat.location = strings.TrimSpace(meta.Location)
		}
		if len(choiceTexts) == 0 && len(meta.Choices) > 0 {
			choiceTexts = append(choiceTexts, meta.Choices...)
		}
	}

	if narrative == "" && len(choiceTexts) == 0 {
		return recentAssistantBeat{}, false
	}

	termSet := make(map[string]struct{})
	for _, token := range significantNarrativeTokens(narrative) {
		termSet[token] = struct{}{}
	}
	choiceTermSet := make(map[string]struct{})
	for _, choice := range choiceTexts {
		for _, token := range significantNarrativeTokens(choice) {
			termSet[token] = struct{}{}
			choiceTermSet[token] = struct{}{}
		}
	}

	beat.terms = sortedKeys(termSet)
	beat.choiceTerms = choiceTermSet
	return beat, len(beat.terms) > 0 || len(beat.choiceTerms) > 0
}

func repeatedBeatTerms(beats []recentAssistantBeat) []string {
	if len(beats) == 0 {
		return nil
	}

	threshold := len(beats)
	if threshold > 3 {
		threshold = 3
	}

	counts := make(map[string]int)
	for _, beat := range beats {
		seen := make(map[string]struct{}, len(beat.terms))
		for _, term := range beat.terms {
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			counts[term]++
		}
	}

	type repeatedTerm struct {
		term  string
		count int
	}
	var repeated []repeatedTerm
	for term, count := range counts {
		if count >= threshold {
			repeated = append(repeated, repeatedTerm{term: term, count: count})
		}
	}
	sort.Slice(repeated, func(i, j int) bool {
		if repeated[i].count != repeated[j].count {
			return repeated[i].count > repeated[j].count
		}
		return repeated[i].term < repeated[j].term
	})

	if len(repeated) == 0 {
		return nil
	}
	limit := len(repeated)
	if limit > 5 {
		limit = 5
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, repeated[i].term)
	}
	return out
}

func countSimilarChoicePairs(beats []recentAssistantBeat) int {
	if len(beats) < 2 {
		return 0
	}

	count := 0
	for i := 1; i < len(beats); i++ {
		if jaccardSimilarity(beats[i-1].choiceTerms, beats[i].choiceTerms) >= 0.4 {
			count++
		}
	}
	return count
}

func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	intersection := 0
	union := len(a)
	for term := range b {
		if _, ok := a[term]; ok {
			intersection++
			continue
		}
		union++
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func sameRecentLocation(beats []recentAssistantBeat) bool {
	if len(beats) < 3 {
		return false
	}

	location := strings.TrimSpace(beats[0].location)
	if location == "" {
		return false
	}
	for i := 1; i < len(beats); i++ {
		if !strings.EqualFold(location, strings.TrimSpace(beats[i].location)) {
			return false
		}
	}
	return true
}

func worldHasLowNarrativePressure(world *storage.WorldState) bool {
	if world == nil {
		return true
	}
	if world.GlobalEventsJSON != "" && world.GlobalEventsJSON != "null" && world.GlobalEventsJSON != "[]" {
		return false
	}
	if len(activeStoryHooks(loadStoryHooks(world))) > 0 {
		return false
	}
	if len(visibleWorldReactions(loadWorldReactions(world))) > 0 {
		return false
	}
	if len(knownFronts(loadFronts(world))) > 0 {
		return false
	}
	board := loadInvestigationBoard(world)
	for _, invCase := range board.Cases {
		if !strings.EqualFold(invCase.Status, "solved") {
			return false
		}
	}
	projectBoard := loadProjectBoard(world)
	for _, project := range projectBoard.Projects {
		if !strings.EqualFold(project.Status, "completed") {
			return false
		}
	}
	return true
}

func significantNarrativeTokens(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	if len(fields) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(fields))
	var out []string
	for _, field := range fields {
		if len(field) < 4 {
			continue
		}
		if narrativeMomentumStopwords[field] {
			continue
		}
		if isAllDigits(field) {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}

func sortedKeys(items map[string]struct{}) []string {
	if len(items) == 0 {
		return nil
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

var narrativeMomentumStopwords = map[string]bool{
	"alla": true, "alle": true, "anche": true, "ancora": true, "avere": true, "avrai": true, "bene": true,
	"come": true, "con": true, "cosa": true, "dalla": true, "dalle": true, "dello": true, "della": true,
	"delle": true, "dopo": true, "dove": true, "fare": true, "fino": true, "mentre": true, "nella": true,
	"nelle": true, "ogni": true, "perche": true, "quale": true, "quella": true, "quello": true, "questa": true,
	"queste": true, "questi": true, "questo": true, "resti": true, "resta": true, "sempre": true, "senza": true,
	"solo": true, "sono": true, "sotto": true, "sulla": true, "sulle": true,
	"about": true, "after": true, "again": true, "along": true, "always": true, "around": true, "before": true,
	"between": true, "choose": true, "choice": true, "continue": true, "could": true, "despite": true,
	"each": true, "from": true, "have": true, "into": true, "just": true, "like": true, "look": true,
	"more": true, "near": true, "onto": true, "over": true, "same": true, "scene": true, "since": true,
	"still": true, "take": true, "that": true, "their": true, "them": true, "then": true, "there": true,
	"these": true, "they": true, "this": true, "through": true, "toward": true, "under": true, "until": true,
	"very": true, "what": true, "when": true, "where": true, "while": true, "with": true, "would": true,
	"your": true,
}
