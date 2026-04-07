// Package rag implements the Retrieval-Augmented Generation pipeline for OneDay.
// It embeds periodic story summaries into a vector store and retrieves relevant
// context chunks to inject into AI prompts for long-term narrative memory.
package rag

import (
	"context"
	"fmt"
	"log"

	"github.com/crimsab/oneday/internal/storage"
)

// RAG orchestrates the full retrieval-augmented generation pipeline.
type RAG struct {
	embedder   *Embedder
	store      *VectorStore
	summarizer *Summarizer
	storyID    string
	topK       int
}

// NewRAG creates a RAG orchestrator.
func NewRAG(embedder *Embedder, store *VectorStore, summarizer *Summarizer, storyID string, topK int) *RAG {
	return &RAG{
		embedder:   embedder,
		store:      store,
		summarizer: summarizer,
		storyID:    storyID,
		topK:       topK,
	}
}

// Retrieve returns the top-K most relevant text chunks for a query.
// Used by the context builder before each AI call.
// Returns empty slice on error — RAG is an optional enhancement, not a hard dependency.
func (r *RAG) Retrieve(ctx context.Context, query string) ([]string, error) {
	if query == "" {
		return nil, nil
	}

	embedding, err := r.embedder.Generate(ctx, query)
	if err != nil {
		// Non-fatal: log and return empty — gameplay continues without RAG context
		log.Printf("[rag] embedding generation failed for query: %v", err)
		return nil, nil
	}

	results, err := r.store.Search(ctx, r.storyID, embedding, r.topK)
	if err != nil {
		log.Printf("[rag] vector search failed: %v", err)
		return nil, nil
	}

	texts := make([]string, 0, len(results))
	for _, res := range results {
		texts = append(texts, res.Chunk.Text)
	}
	return texts, nil
}

// MaybeSummarize checks if summarization is due and runs it if needed.
// Called after each turn (typically in a goroutine — fire-and-forget).
// Returns true if summarization was performed.
func (r *RAG) MaybeSummarize(ctx context.Context, messages []storage.ChatMessage, currentTurn int) (bool, error) {
	should, err := r.summarizer.ShouldSummarize(ctx, currentTurn)
	if err != nil {
		return false, fmt.Errorf("rag: checking summarize condition: %w", err)
	}
	if !should {
		return false, nil
	}

	if err := r.summarizer.Summarize(ctx, messages, currentTurn); err != nil {
		return false, fmt.Errorf("rag: summarization failed: %w", err)
	}
	return true, nil
}

// SummarizerInterval returns the configured summarization interval.
func (r *RAG) SummarizerInterval() int {
	return r.summarizer.Interval()
}
