package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
)

// OllamaEmbedding implements local embeddings via Ollama's /api/embed endpoint.
type OllamaEmbedding struct {
	name    string
	baseURL string
	model   string
	client  *http.Client
}

type OllamaEmbeddingConfig struct {
	BaseURL string
	Model   string
	Timeout time.Duration
}

func NewOllamaEmbedding(cfg OllamaEmbeddingConfig) *OllamaEmbedding {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &OllamaEmbedding{
		name:    "local-ollama",
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: timeout},
	}
}

func (o *OllamaEmbedding) Embed(ctx context.Context, req ai.EmbeddingRequest) (ai.EmbeddingResponse, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = o.model
	}
	if model == "" {
		return ai.EmbeddingResponse{}, fmt.Errorf("Ollama embedding model is empty")
	}

	body := map[string]any{
		"model": model,
		"input": req.Input,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("marshaling Ollama embedding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/embed", bytes.NewReader(jsonBody))
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("creating Ollama embedding request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("Ollama embedding request failed: %w; is Ollama running at %s?", err, o.baseURL)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("reading Ollama embedding response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return ai.EmbeddingResponse{}, fmt.Errorf("Ollama embedding returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	embedding, resolvedModel, err := parseEmbeddingResponse(respBody, model)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("parsing Ollama embedding response: %w", err)
	}
	return ai.EmbeddingResponse{Embedding: embedding, Model: resolvedModel}, nil
}
