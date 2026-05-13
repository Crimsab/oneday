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

func TestOpenAICompatMissingLiteLLMKeyIsActionable(t *testing.T) {
	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:         "litellm",
		BaseURL:      "http://example.invalid",
		DefaultModel: "test-model",
		Timeout:      time.Second,
	})

	_, err := provider.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected missing key error")
	}
	msg := err.Error()
	for _, want := range []string{"ONEDAY_LITELLM_API_KEY", "oneday doctor"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestOpenAICompatOpenRouter401IsActionable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:         "openrouter",
		BaseURL:      server.URL,
		APIKey:       "bad-key",
		DefaultModel: "test-model",
		Timeout:      time.Second,
	})

	_, err := provider.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	msg := err.Error()
	for _, want := range []string{"OpenRouter authentication failed", "ONEDAY_OPENROUTER_API_KEY", "oneday doctor"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestOpenAICompatEmbedding401IsActionable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:         "litellm-embed",
		BaseURL:      server.URL,
		APIKey:       "bad-key",
		DefaultModel: "text-embedding-3-small",
		Timeout:      time.Second,
	})

	_, err := provider.Embed(context.Background(), ai.EmbeddingRequest{
		Input: "hello",
		Model: "text-embedding-3-small",
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	msg := err.Error()
	for _, want := range []string{"LiteLLM authentication failed", "ONEDAY_LITELLM_API_KEY"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
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
		plugins, ok := body["plugins"].([]any)
		if !ok || len(plugins) == 0 {
			t.Fatalf("plugins missing from request body")
		}
		plugin, ok := plugins[0].(map[string]any)
		if !ok || plugin["id"] != "response-healing" {
			t.Fatalf("expected response-healing plugin, got %#v", plugins)
		}
		provider, ok := body["provider"].(map[string]any)
		if !ok || provider["require_parameters"] != true {
			t.Fatalf("provider.require_parameters missing or false: %#v", body["provider"])
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
		Name:         "litellm",
		BaseURL:      server.URL,
		APIKey:       "test-key",
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

func TestOpenAICompatCompletePreservesExplicitPluginConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		plugins, ok := body["plugins"].([]any)
		if !ok || len(plugins) != 1 {
			t.Fatalf("plugins = %#v, want one explicit plugin preserved", body["plugins"])
		}
		plugin := plugins[0].(map[string]any)
		if plugin["id"] != "response-healing" {
			t.Fatalf("plugin id = %v, want response-healing", plugin["id"])
		}
		provider := body["provider"].(map[string]any)
		if provider["require_parameters"] != true {
			t.Fatalf("provider.require_parameters = %v, want true", provider["require_parameters"])
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
		Name:         "litellm",
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "test-model",
		Timeout:      5 * time.Second,
	})

	_, err := provider.Complete(context.Background(), ai.Request{
		Messages:       []ai.Message{{Role: "user", Content: "hello"}},
		ResponseFormat: ai.NarrativeResponseFormat(),
		Plugins:        []ai.Plugin{{ID: "response-healing"}},
		Provider:       &ai.ProviderConfig{RequireParameters: false},
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
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		// Verify stream:true was sent.
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["stream"] != true {
			t.Errorf("stream = %v, want true", body["stream"])
		}
		plugins, ok := body["plugins"].([]any)
		if !ok || len(plugins) == 0 {
			t.Errorf("plugins missing from streaming request body")
		}
		plugin, ok := plugins[0].(map[string]any)
		if !ok || plugin["id"] != "response-healing" {
			t.Errorf("expected response-healing plugin in stream request, got %#v", body["plugins"])
		}
		provider, ok := body["provider"].(map[string]any)
		if !ok || provider["require_parameters"] != true {
			t.Errorf("provider.require_parameters missing or false in stream request: %#v", body["provider"])
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
		Name:    "litellm",
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 5 * time.Second,
	})

	ch, err := provider.Stream(context.Background(), ai.Request{
		Messages:       []ai.Message{{Role: "user", Content: "hi"}},
		ResponseFormat: ai.NarrativeResponseFormat(),
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

func TestOpenAICompatStreamRetriesWithoutUnsupportedResponseFormat(t *testing.T) {
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
				t.Fatalf("expected first stream request to carry response_format")
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"response_format json_schema not supported"}}`))
			return
		}

		if hasResponseFormat {
			t.Fatalf("expected retry stream request without response_format")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		fmt.Fprint(w, sseEvent(`{"narrative":"fallback ok","choices":[{"id":1,"text":"Continue"}]}`))
		flusher.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:         "litellm",
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "test-model",
		Timeout:      5 * time.Second,
	})

	ch, err := provider.Stream(context.Background(), ai.Request{
		Messages:       []ai.Message{{Role: "user", Content: "hello"}},
		ResponseFormat: ai.NarrativeResponseFormat(),
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var got strings.Builder
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("chunk error: %v", chunk.Error)
		}
		got.WriteString(chunk.Content)
	}
	if !strings.Contains(got.String(), "fallback ok") {
		t.Fatalf("unexpected fallback stream content: %q", got.String())
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestOpenAICompatStreamFallsBackToCompleteWhenStreamingUnsupported(t *testing.T) {
	var streamCalls int
	var completeCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["stream"] == true {
			streamCalls++
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = w.Write([]byte(`{"error":{"message":"streaming not supported for this route"}}`))
			return
		}

		completeCalls++
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"narrative":"complete fallback","choices":[{"id":1,"text":"Continue"}]}`}},
			},
			"model": "test-model",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAICompat(OpenAICompatConfig{
		Name:         "litellm",
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "test-model",
		Timeout:      5 * time.Second,
	})

	ch, err := provider.Stream(context.Background(), ai.Request{
		Messages:       []ai.Message{{Role: "user", Content: "hello"}},
		ResponseFormat: ai.NarrativeResponseFormat(),
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var chunks []ai.StreamChunk
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("chunk error: %v", chunk.Error)
		}
		chunks = append(chunks, chunk)
	}
	if streamCalls != 1 {
		t.Fatalf("streamCalls = %d, want 1", streamCalls)
	}
	if completeCalls != 1 {
		t.Fatalf("completeCalls = %d, want 1", completeCalls)
	}
	if len(chunks) < 2 || !strings.Contains(chunks[0].Content, "complete fallback") || !chunks[len(chunks)-1].Done {
		t.Fatalf("unexpected fallback chunks: %#v", chunks)
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
