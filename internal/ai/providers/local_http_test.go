package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/ai"
)

func TestLocalHTTPEmbeddingParsesOpenAIStyleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embed" {
			t.Fatalf("path = %q, want /embed", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "local-test",
			"data":  []map[string]any{{"embedding": []float32{1, 2, 3}}},
		})
	}))
	defer server.Close()

	provider := NewLocalHTTPEmbedding(server.URL+"/embed", "local-test", time.Second)
	resp, err := provider.Embed(context.Background(), ai.EmbeddingRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if resp.Model != "local-test" || len(resp.Embedding) != 3 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestOllamaEmbeddingParsesEmbedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Fatalf("path = %q, want /api/embed", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":      "test-local-embedding-model",
			"embeddings": [][]float32{{1, 2, 3, 4}},
		})
	}))
	defer server.Close()

	provider := NewOllamaEmbedding(OllamaEmbeddingConfig{BaseURL: server.URL, Model: "test-local-embedding-model", Timeout: time.Second})
	resp, err := provider.Embed(context.Background(), ai.EmbeddingRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if resp.Model != "test-local-embedding-model" || len(resp.Embedding) != 4 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
