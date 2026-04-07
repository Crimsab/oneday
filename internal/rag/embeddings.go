package rag

import (
	"context"
	"fmt"

	"github.com/crimsab/oneday/internal/ai"
)

// EmbeddingProvider is the interface for embedding generation (allows mocking in tests).
type EmbeddingProvider interface {
	Embed(ctx context.Context, req ai.EmbeddingRequest) (ai.EmbeddingResponse, error)
}

// Embedder generates text embeddings via an AI provider.
type Embedder struct {
	provider   EmbeddingProvider
	model      string
	dimensions int
}

// NewEmbedder creates an Embedder using the given provider and model.
func NewEmbedder(provider EmbeddingProvider, model string, dimensions int) *Embedder {
	return &Embedder{
		provider:   provider,
		model:      model,
		dimensions: dimensions,
	}
}

// Generate produces an embedding vector for the given text.
func (e *Embedder) Generate(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("embedding: input text is empty")
	}

	resp, err := e.provider.Embed(ctx, ai.EmbeddingRequest{
		Input: text,
		Model: e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("embedding generation: %w", err)
	}

	if len(resp.Embedding) == 0 {
		return nil, fmt.Errorf("embedding: provider returned empty vector")
	}

	return resp.Embedding, nil
}

// Dimensions returns the configured embedding dimensions.
func (e *Embedder) Dimensions() int {
	return e.dimensions
}
