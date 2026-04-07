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

func TestOpenAICompatComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth = %q, want Bearer test-key", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "test response"}},
			},
			"model": "test-model",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:         "test-provider",
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "test-model",
		Timeout:      5 * time.Second,
	})

	resp, err := provider.Complete(context.Background(), ai.Request{
		Messages:    []ai.Message{{Role: "user", Content: "hello"}},
		Temperature: 0.7,
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Content != "test response" {
		t.Errorf("Content = %q, want %q", resp.Content, "test response")
	}
	if resp.Provider != "test-provider" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "test-provider")
	}
}

func TestOpenAICompatServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:    "failing-provider",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := provider.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestOpenAICompatNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{},
			"model":   "test",
		})
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:    "empty-provider",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := provider.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Error("expected error for empty choices")
	}
}
