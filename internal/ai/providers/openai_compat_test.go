package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestOpenAICompatCompletePassesResponseFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		rf, ok := body["response_format"].(map[string]any)
		if !ok {
			t.Fatalf("response_format missing from request body")
		}
		if rf["type"] != "json_schema" {
			t.Fatalf("response_format.type = %v, want json_schema", rf["type"])
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"narrative":"ok","choices":[{"id":1,"text":"Continue"}]}`}},
			},
			"model": "test-model",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:         "test-provider",
		BaseURL:      server.URL,
		DefaultModel: "test-model",
		Timeout:      5 * time.Second,
	})

	_, err := provider.Complete(context.Background(), ai.Request{
		Messages:       []ai.Message{{Role: "user", Content: "hello"}},
		ResponseFormat: ai.NarrativeResponseFormat(),
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
}

func TestOpenAICompatCompleteRetriesWithoutUnsupportedResponseFormat(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		_, hasResponseFormat := body["response_format"]
		if calls == 1 {
			if !hasResponseFormat {
				t.Fatalf("expected first request to carry response_format")
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"response_format json_schema not supported"}}`))
			return
		}

		if hasResponseFormat {
			t.Fatalf("expected retry request without response_format")
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"narrative":"fallback ok","choices":[{"id":1,"text":"Continue"}]}`}},
			},
			"model": "test-model",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:         "test-provider",
		BaseURL:      server.URL,
		DefaultModel: "test-model",
		Timeout:      5 * time.Second,
	})

	resp, err := provider.Complete(context.Background(), ai.Request{
		Messages:       []ai.Message{{Role: "user", Content: "hello"}},
		ResponseFormat: ai.NarrativeResponseFormat(),
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !strings.Contains(resp.Content, "fallback ok") {
		t.Fatalf("unexpected fallback content: %q", resp.Content)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// sseEvent formats a single SSE data line for testing.
func sseEvent(content string) string {
	payload := fmt.Sprintf(`{"choices":[{"delta":{"content":%q}}]}`, content)
	return "data: " + payload + "\n\n"
}

func TestOpenAICompatStream(t *testing.T) {
	chunks := []string{"Hello", ", ", "world", "!"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		// Verify stream:true was sent.
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["stream"] != true {
			t.Errorf("stream = %v, want true", body["stream"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for _, c := range chunks {
			fmt.Fprint(w, sseEvent(c))
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:    "sse-provider",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	ch, err := provider.Stream(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var got string
	var gotDone bool
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("chunk error: %v", chunk.Error)
		}
		if chunk.Done {
			gotDone = true
			break
		}
		got += chunk.Content
	}

	if got != "Hello, world!" {
		t.Errorf("content = %q, want %q", got, "Hello, world!")
	}
	if !gotDone {
		t.Error("expected Done chunk")
	}
}

func TestOpenAICompatStreamServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:    "error-stream",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	})

	_, err := provider.Stream(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Error("expected error for non-200 stream response")
	}
}

func TestOpenAICompatStreamImplementsStreamProvider(t *testing.T) {
	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:    "type-check",
		BaseURL: "http://localhost",
		Timeout: time.Second,
	})
	// Compile-time check: OpenAICompat must satisfy ai.StreamProvider.
	var _ ai.StreamProvider = provider
}
