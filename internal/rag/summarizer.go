package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/storage"
)

// AICompleter is the interface for AI completion (summarization). Satisfied by *ai.Router.
type AICompleter interface {
	Complete(ctx context.Context, req ai.Request) (ai.Response, error)
}

// Summarizer periodically condenses narrative turns into embedded chunks for long-term memory.
type Summarizer struct {
	embedder *Embedder
	store    *VectorStore
	ai       AICompleter
	storyID  string
	interval int // summarize every N turns
}

// NewSummarizer creates a Summarizer.
func NewSummarizer(embedder *Embedder, store *VectorStore, aiCompleter AICompleter, storyID string, interval int) *Summarizer {
	return &Summarizer{
		embedder: embedder,
		store:    store,
		ai:       aiCompleter,
		storyID:  storyID,
		interval: interval,
	}
}

// ShouldSummarize returns true if there are enough unsummarized turns to warrant a new summary.
func (s *Summarizer) ShouldSummarize(ctx context.Context, currentTurn int) (bool, error) {
	lastTurn, err := s.store.LastSummarizedTurn(ctx, s.storyID)
	if err != nil {
		return false, fmt.Errorf("summarizer: checking last summarized turn: %w", err)
	}
	gap := currentTurn - lastTurn
	return gap >= s.interval, nil
}

// Summarize generates a summary for unsummarized turns [lastSummarized+1 .. currentTurn],
// embeds it, and stores it as a chunk with type "summary".
func (s *Summarizer) Summarize(ctx context.Context, messages []storage.ChatMessage, currentTurn int) error {
	if len(messages) == 0 {
		return nil
	}

	lastTurn, err := s.store.LastSummarizedTurn(ctx, s.storyID)
	if err != nil {
		return fmt.Errorf("summarizer: getting last summarized turn: %w", err)
	}

	// Build the conversation text from unsummarized messages.
	var sb strings.Builder
	turnStart := currentTurn
	turnEnd := lastTurn

	for _, m := range messages {
		if m.Turn <= lastTurn {
			continue // already summarized
		}
		if m.Turn < turnStart {
			turnStart = m.Turn
		}
		if m.Turn > turnEnd {
			turnEnd = m.Turn
		}

		role := "Player"
		if m.Role == "assistant" {
			role = "Narrator"
		}
		sb.WriteString(fmt.Sprintf("[Turn %d - %s]: %s\n\n", m.Turn, role, m.Content))
	}

	text := sb.String()
	if strings.TrimSpace(text) == "" {
		return nil // nothing new to summarize
	}

	// Call AI to generate a concise summary.
	summaryResp, err := s.ai.Complete(ctx, ai.Request{
		Messages: []ai.Message{
			{Role: ai.RoleSystem, Content: prompts.SummarizerSystem},
			{Role: ai.RoleUser, Content: text},
		},
		Temperature: 0.3,
		MaxTokens:   600,
	})
	if err != nil {
		return fmt.Errorf("summarizer: AI completion failed: %w", err)
	}

	summaryText := strings.TrimSpace(summaryResp.Content)
	if summaryText == "" {
		return fmt.Errorf("summarizer: AI returned empty summary")
	}

	// Embed the summary.
	embedding, err := s.embedder.Generate(ctx, summaryText)
	if err != nil {
		return fmt.Errorf("summarizer: embedding generation failed: %w", err)
	}

	// Store as a RAG chunk.
	chunk := &Chunk{
		StoryID:   s.storyID,
		Text:      summaryText,
		ChunkType: "summary",
		TurnStart: turnStart,
		TurnEnd:   turnEnd,
		Embedding: embedding,
	}
	if err := s.store.Insert(ctx, chunk); err != nil {
		return fmt.Errorf("summarizer: storing chunk: %w", err)
	}

	return nil
}

// Interval returns the summarization interval (every N turns).
func (s *Summarizer) Interval() int {
	return s.interval
}
