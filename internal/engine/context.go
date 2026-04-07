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
}

// DefaultContextConfig returns sensible defaults.
func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		RecentMessageCount: 20,
		RAGChunks:          nil,
	}
}

// BuildContext assembles the full message list for an AI call.
// Order: [system prompt, state summary, optional RAG, ...recent messages, current user input]
func BuildContext(
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	recentMessages []storage.ChatMessage,
	ragChunks []string,
	currentInput string,
) []ai.Message {
	msgs := make([]ai.Message, 0, len(recentMessages)+4)

	// 1. Build system prompt using the NarratorSystem prompt builder.
	systemPrompt := prompts.NarratorSystem(
		story.Name,
		story.SettingJSON,
		story.StatsSchemaJSON,
		char.Name,
		char.Background,
		char.StatsJSON,
	)
	msgs = append(msgs, ai.Message{
		Role:    ai.RoleSystem,
		Content: systemPrompt,
	})

	// 2. Add a live state summary — reflects current state, not creation-time state.
	stateSummary := buildStateSummary(char, world)
	msgs = append(msgs, ai.Message{
		Role:    ai.RoleSystem,
		Content: stateSummary,
	})

	// 3. Inject RAG chunks if provided (placeholder for Phase 5).
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
func buildStateSummary(char *storage.Character, world *storage.WorldState) string {
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
	return sb.String()
}
