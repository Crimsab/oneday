package engine

import (
	"fmt"
	"strings"

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
func BuildContext(
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	npcs []storage.NPC,
	recentMessages []storage.ChatMessage,
	ragChunks []string,
	lastChapterSummary string,
	currentInput string,
) []ai.Message {
	msgs := make([]ai.Message, 0, len(recentMessages)+4)

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

	// 3. Inject RAG chunks if provided (long-term memory from past turns).
	if len(ragChunks) > 0 {
		ragContent := "## Relevant Memory\n" + strings.Join(ragChunks, "\n---\n")
		msgs = append(msgs, ai.Message{
			Role:    ai.RoleSystem,
			Content: ragContent,
		})
	}

	// 4. Convert recent DB messages to ai.Message.
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

	// 5. Append the current user input as the final message.
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
	sb.WriteString("## Current Game State\n")
	sb.WriteString(fmt.Sprintf("- Chapter: %d\n", world.CurrentChapter))
	sb.WriteString(fmt.Sprintf("- Turn: %d\n", world.CurrentTurn))
	sb.WriteString(fmt.Sprintf("- Location: %s\n", world.CurrentLocation))
	sb.WriteString(fmt.Sprintf("- Character: %s\n", char.Name))
	sb.WriteString(fmt.Sprintf("- Stats (live): %s\n", char.StatsJSON))
	if char.TraitsJSON != "" && char.TraitsJSON != "null" && char.TraitsJSON != "[]" {
		sb.WriteString(fmt.Sprintf("- Traits: %s\n", char.TraitsJSON))
	}
	if len(npcs) > 0 {
		names := make([]string, len(npcs))
		for i, npc := range npcs {
			names[i] = npc.Name
		}
		sb.WriteString(fmt.Sprintf("- Known NPCs: %d (%s)\n", len(npcs), strings.Join(names, ", ")))
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
	// Previous chapter summary for narrative continuity.
	if lastChapterSummary != "" {
		sb.WriteString(fmt.Sprintf("- Previous Chapter Summary: %s\n", lastChapterSummary))
	}
	return sb.String()
}
