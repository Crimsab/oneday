package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
)

// LocalHTTPEmbedding calls a custom local embedding endpoint. It accepts
// OpenAI-style, Ollama-style, or simple {embedding:[...]} responses.
type LocalHTTPEmbedding struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewLocalHTTPEmbedding(baseURL, model string, timeout time.Duration) *LocalHTTPEmbedding {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &LocalHTTPEmbedding{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}
}

func (l *LocalHTTPEmbedding) Embed(ctx context.Context, req ai.EmbeddingRequest) (ai.EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = l.model
	}
	jsonBody, err := json.Marshal(map[string]any{"model": model, "input": req.Input})
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("marshaling local embedding request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, l.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("creating local embedding request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := l.client.Do(httpReq)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("local embedding request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := readResponseBody(resp.Body)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("reading local embedding response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return ai.EmbeddingResponse{}, fmt.Errorf("local embedding returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	embedding, resolvedModel, err := parseEmbeddingResponse(respBody, model)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("parsing local embedding response: %w", err)
	}
	return ai.EmbeddingResponse{Embedding: embedding, Model: resolvedModel}, nil
}

func parseEmbeddingResponse(respBody []byte, fallbackModel string) ([]float32, string, error) {
	var payload struct {
		Model      string      `json:"model"`
		Embedding  []float32   `json:"embedding"`
		Embeddings [][]float32 `json:"embeddings"`
		Data       []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, "", err
	}
	model := payload.Model
	if model == "" {
		model = fallbackModel
	}
	switch {
	case len(payload.Embedding) > 0:
		return payload.Embedding, model, nil
	case len(payload.Embeddings) > 0 && len(payload.Embeddings[0]) > 0:
		return payload.Embeddings[0], model, nil
	case len(payload.Data) > 0 && len(payload.Data[0].Embedding) > 0:
		return payload.Data[0].Embedding, model, nil
	default:
		return nil, model, fmt.Errorf("response contained no embedding vector")
	}
}
