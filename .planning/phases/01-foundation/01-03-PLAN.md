---
phase: 1
plan: 1.3
title: "AI provider router with fallback chain"
wave: 2
depends_on: [1.1]
files_modified:
  - internal/ai/provider.go
  - internal/ai/router.go
  - internal/ai/router_test.go
  - internal/ai/providers/claudecode.go
  - internal/ai/providers/openai_compat.go
  - internal/ai/providers/openai_compat_test.go
requirements_addressed: [AI-01]
autonomous: true
---

# Plan 1.3: AI Provider Router with Fallback Chain

## Objective

Implement the AI provider router that supports Claude Code CLI, LiteLLM, and OpenRouter with a configurable fallback chain. The router tries each provider in priority order until one succeeds. Claude Code uses CLI shell-out; LiteLLM and OpenRouter use the same OpenAI-compatible HTTP client. The router depends on config (Plan 1.1) for provider settings and priority.

## must_haves

- Provider interface with `Complete(ctx, messages) (response, error)` contract
- Claude Code provider shells out to `claude -p "prompt" --output-format json`
- LiteLLM and OpenRouter share one OpenAI-compatible HTTP provider
- Router tries providers in config priority order, skipping disabled ones
- Router falls back to next provider on error
- Router returns error only when ALL providers fail

## Tasks

### Task 1: Define Provider interface and types

<read_first>
- internal/config/config.go (AIConfig, provider structs)
- docs/design.md (AI section, provider chain description)
</read_first>

<action>
Create `internal/ai/provider.go`:

```go
package ai

import (
	"context"
	"fmt"
)

// Role constants for chat messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message represents a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request holds parameters for an AI completion request.
type Request struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Response holds the result of an AI completion.
type Response struct {
	Content   string `json:"content"`
	Model     string `json:"model"`
	LatencyMs int64  `json:"latency_ms"`
	Provider  string `json:"provider"`
}

// Provider is the interface that all AI providers must implement.
type Provider interface {
	// Name returns the provider identifier (e.g., "claude-code", "litellm", "openrouter").
	Name() string

	// Complete sends a chat completion request and returns the response.
	Complete(ctx context.Context, req Request) (Response, error)
}

// ProviderError wraps an error with the provider name that caused it.
type ProviderError struct {
	ProviderName string
	Err          error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s: %s", e.ProviderName, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
```
</action>

<acceptance_criteria>
- `grep "type Provider interface" internal/ai/provider.go` matches
- `grep "type Message struct" internal/ai/provider.go` matches
- `grep "type Request struct" internal/ai/provider.go` matches
- `grep "type Response struct" internal/ai/provider.go` matches
- `grep "func.*Complete" internal/ai/provider.go` matches
</acceptance_criteria>

### Task 2: Implement Claude Code CLI provider

<read_first>
- internal/ai/provider.go (just created)
- config.example.yaml (claude_code section)
</read_first>

<action>
Create `internal/ai/providers/claudecode.go`:

```go
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
)

// ClaudeCode implements ai.Provider by shelling out to the Claude Code CLI.
type ClaudeCode struct {
	binary string
}

// claudeCodeResponse is the JSON output from `claude --output-format json`.
type claudeCodeResponse struct {
	Result string `json:"result"`
	Model  string `json:"model"`
}

// NewClaudeCode creates a Claude Code CLI provider.
func NewClaudeCode(cfg config.ClaudeCodeConfig) *ClaudeCode {
	binary := cfg.Binary
	if binary == "" {
		binary = "claude"
	}
	return &ClaudeCode{binary: binary}
}

func (c *ClaudeCode) Name() string {
	return "claude-code"
}

// Complete sends the last user message as a prompt to claude CLI.
// System messages and history are concatenated into the prompt since the CLI
// does not support multi-turn conversation directly.
func (c *ClaudeCode) Complete(ctx context.Context, req ai.Request) (ai.Response, error) {
	start := time.Now()

	// Build prompt from messages
	prompt := buildPrompt(req.Messages)
	if prompt == "" {
		return ai.Response{}, fmt.Errorf("no messages to send")
	}

	cmd := exec.CommandContext(ctx, c.binary, "-p", prompt, "--output-format", "json")
	output, err := cmd.Output()
	if err != nil {
		return ai.Response{}, fmt.Errorf("claude CLI exec: %w", err)
	}

	var resp claudeCodeResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return ai.Response{}, fmt.Errorf("parsing claude CLI output: %w", err)
	}

	return ai.Response{
		Content:   resp.Result,
		Model:     resp.Model,
		LatencyMs: time.Since(start).Milliseconds(),
		Provider:  c.Name(),
	}, nil
}

// buildPrompt concatenates system and user messages into a single prompt string.
func buildPrompt(messages []ai.Message) string {
	var parts []string
	for _, m := range messages {
		switch m.Role {
		case ai.RoleSystem:
			parts = append(parts, "[System]\n"+m.Content)
		case ai.RoleUser:
			parts = append(parts, "[User]\n"+m.Content)
		case ai.RoleAssistant:
			parts = append(parts, "[Assistant]\n"+m.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}
```
</action>

<acceptance_criteria>
- `grep "type ClaudeCode struct" internal/ai/providers/claudecode.go` matches
- `grep "func NewClaudeCode" internal/ai/providers/claudecode.go` matches
- `grep 'output-format.*json' internal/ai/providers/claudecode.go` matches
- `grep "func.*ClaudeCode.*Complete" internal/ai/providers/claudecode.go` matches
- `grep "func buildPrompt" internal/ai/providers/claudecode.go` matches
</acceptance_criteria>

### Task 3: Implement OpenAI-compatible provider (LiteLLM + OpenRouter)

<read_first>
- internal/ai/provider.go
- config.example.yaml (litellm and openrouter sections)
</read_first>

<action>
Create `internal/ai/providers/openai_compat.go`:

```go
package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/crimsab/oneday/internal/ai"
)

// OpenAICompat implements ai.Provider for any OpenAI-compatible API
// (LiteLLM, OpenRouter, etc.).
type OpenAICompat struct {
	name         string
	baseURL      string
	apiKey       string
	defaultModel string
	client       *http.Client
}

// OpenAICompatConfig holds settings for creating an OpenAI-compatible provider.
type OpenAICompatConfig struct {
	Name         string
	BaseURL      string
	APIKey       string
	DefaultModel string
	Timeout      time.Duration
}

// openAIChatRequest is the request body for /v1/chat/completions.
type openAIChatRequest struct {
	Model       string       `json:"model"`
	Messages    []ai.Message `json:"messages"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
}

// openAIChatResponse is the response from /v1/chat/completions.
type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
}

// NewOpenAICompat creates a new OpenAI-compatible provider.
func NewOpenAICompat(cfg OpenAICompatConfig) *OpenAICompat {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &OpenAICompat{
		name:         cfg.Name,
		baseURL:      cfg.BaseURL,
		apiKey:       cfg.APIKey,
		defaultModel: cfg.DefaultModel,
		client:       &http.Client{Timeout: timeout},
	}
}

func (o *OpenAICompat) Name() string {
	return o.name
}

// Complete sends a chat completion request to the OpenAI-compatible endpoint.
func (o *OpenAICompat) Complete(ctx context.Context, req ai.Request) (ai.Response, error) {
	start := time.Now()

	model := req.Model
	if model == "" {
		model = o.defaultModel
	}

	body := openAIChatRequest{
		Model:       model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ai.Response{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := o.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return ai.Response{}, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return ai.Response{}, fmt.Errorf("HTTP request to %s: %w", o.name, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ai.Response{}, fmt.Errorf("reading response from %s: %w", o.name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return ai.Response{}, fmt.Errorf("%s returned status %d: %s", o.name, resp.StatusCode, string(respBody))
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return ai.Response{}, fmt.Errorf("parsing response from %s: %w", o.name, err)
	}

	if len(chatResp.Choices) == 0 {
		return ai.Response{}, fmt.Errorf("%s returned no choices", o.name)
	}

	return ai.Response{
		Content:   chatResp.Choices[0].Message.Content,
		Model:     chatResp.Model,
		LatencyMs: time.Since(start).Milliseconds(),
		Provider:  o.name,
	}, nil
}
```
</action>

<acceptance_criteria>
- `grep "type OpenAICompat struct" internal/ai/providers/openai_compat.go` matches
- `grep "func NewOpenAICompat" internal/ai/providers/openai_compat.go` matches
- `grep "func.*OpenAICompat.*Complete" internal/ai/providers/openai_compat.go` matches
- `grep "/chat/completions" internal/ai/providers/openai_compat.go` matches
- `grep "Authorization" internal/ai/providers/openai_compat.go` matches
</acceptance_criteria>

### Task 4: Implement the Router with fallback chain

<read_first>
- internal/ai/provider.go
- internal/config/config.go (EnabledProviders method)
</read_first>

<action>
Create `internal/ai/router.go`:

```go
package ai

import (
	"context"
	"fmt"
	"strings"
)

// Router routes AI requests through a priority chain of providers.
// It tries each provider in order and falls back to the next on failure.
type Router struct {
	providers []Provider
}

// NewRouter creates a router with the given providers in priority order.
// Providers should already be filtered to only enabled ones.
func NewRouter(providers []Provider) (*Router, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one AI provider is required")
	}
	return &Router{providers: providers}, nil
}

// Complete tries each provider in order until one succeeds.
// Returns the first successful response or an error if all providers fail.
func (r *Router) Complete(ctx context.Context, req Request) (Response, error) {
	var errors []string

	for _, p := range r.providers {
		resp, err := p.Complete(ctx, req)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", p.Name(), err))
			continue
		}
		return resp, nil
	}

	return Response{}, fmt.Errorf("all AI providers failed:\n  %s", strings.Join(errors, "\n  "))
}

// Providers returns the list of providers in priority order.
func (r *Router) Providers() []Provider {
	return r.providers
}

// ProviderNames returns the names of all providers in order.
func (r *Router) ProviderNames() []string {
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}
```
</action>

<acceptance_criteria>
- `grep "type Router struct" internal/ai/router.go` matches
- `grep "func NewRouter" internal/ai/router.go` matches
- `grep "func.*Router.*Complete" internal/ai/router.go` matches
- `grep "all AI providers failed" internal/ai/router.go` matches
- `grep "func.*ProviderNames" internal/ai/router.go` matches
</acceptance_criteria>

### Task 5: Create router and provider tests

<read_first>
- internal/ai/router.go
- internal/ai/provider.go
</read_first>

<action>
Create `internal/ai/router_test.go`:

```go
package ai

import (
	"context"
	"fmt"
	"testing"
)

// mockProvider is a test double for ai.Provider.
type mockProvider struct {
	name    string
	err     error
	content string
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Complete(_ context.Context, _ Request) (Response, error) {
	if m.err != nil {
		return Response{}, m.err
	}
	return Response{
		Content:   m.content,
		Model:     "test-model",
		Provider:  m.name,
		LatencyMs: 100,
	}, nil
}

func TestRouterFirstProviderSucceeds(t *testing.T) {
	r, err := NewRouter([]Provider{
		&mockProvider{name: "primary", content: "hello"},
		&mockProvider{name: "fallback", content: "world"},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := r.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Provider != "primary" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "primary")
	}
	if resp.Content != "hello" {
		t.Errorf("Content = %q, want %q", resp.Content, "hello")
	}
}

func TestRouterFallsBack(t *testing.T) {
	r, err := NewRouter([]Provider{
		&mockProvider{name: "primary", err: fmt.Errorf("connection refused")},
		&mockProvider{name: "fallback", content: "recovered"},
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := r.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Provider != "fallback" {
		t.Errorf("Provider = %q, want %q", resp.Provider, "fallback")
	}
	if resp.Content != "recovered" {
		t.Errorf("Content = %q, want %q", resp.Content, "recovered")
	}
}

func TestRouterAllFail(t *testing.T) {
	r, err := NewRouter([]Provider{
		&mockProvider{name: "p1", err: fmt.Errorf("fail 1")},
		&mockProvider{name: "p2", err: fmt.Errorf("fail 2")},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Error("expected error when all providers fail")
	}
}

func TestRouterNoProviders(t *testing.T) {
	_, err := NewRouter([]Provider{})
	if err == nil {
		t.Error("expected error for empty provider list")
	}
}

func TestRouterProviderNames(t *testing.T) {
	r, _ := NewRouter([]Provider{
		&mockProvider{name: "a"},
		&mockProvider{name: "b"},
		&mockProvider{name: "c"},
	})
	names := r.ProviderNames()
	if len(names) != 3 {
		t.Fatalf("ProviderNames length = %d, want 3", len(names))
	}
	if names[0] != "a" || names[1] != "b" || names[2] != "c" {
		t.Errorf("ProviderNames = %v, want [a b c]", names)
	}
}
```

Create `internal/ai/providers/openai_compat_test.go` for the OpenAI-compatible provider with a mock HTTP server:

```go
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
```
</action>

<acceptance_criteria>
- `grep "func TestRouterFirstProviderSucceeds" internal/ai/router_test.go` matches
- `grep "func TestRouterFallsBack" internal/ai/router_test.go` matches
- `grep "func TestRouterAllFail" internal/ai/router_test.go` matches
- `grep "func TestOpenAICompatComplete" internal/ai/providers/openai_compat_test.go` matches
- `grep "func TestOpenAICompatServerError" internal/ai/providers/openai_compat_test.go` matches
- `go test ./internal/ai/ -v` passes all tests
- `go test ./internal/ai/providers/ -v` passes all tests
</acceptance_criteria>

### Task 6: Create provider factory wiring

<read_first>
- internal/config/config.go (full config struct)
- internal/ai/router.go
- internal/ai/providers/claudecode.go
- internal/ai/providers/openai_compat.go
</read_first>

<action>
Add a factory function to `internal/ai/router.go` (append to the file) OR create a new file `internal/ai/factory.go`:

Create `internal/ai/factory.go`:

```go
package ai

import (
	"fmt"
	"time"

	"github.com/crimsab/oneday/internal/ai/providers"
	"github.com/crimsab/oneday/internal/config"
)

// NewRouterFromConfig builds a Router from the application config.
// It creates provider instances for each enabled provider in the priority chain.
func NewRouterFromConfig(cfg config.Config) (*Router, error) {
	enabledNames := cfg.EnabledProviders()
	if len(enabledNames) == 0 {
		return nil, fmt.Errorf("no AI providers are enabled in config")
	}

	timeout := time.Duration(cfg.AI.Generation.TimeoutSeconds) * time.Second

	providerList := make([]Provider, 0, len(enabledNames))
	for _, name := range enabledNames {
		p, err := buildProvider(name, cfg, timeout)
		if err != nil {
			return nil, fmt.Errorf("building provider %s: %w", name, err)
		}
		providerList = append(providerList, p)
	}

	return NewRouter(providerList)
}

func buildProvider(name string, cfg config.Config, timeout time.Duration) (Provider, error) {
	switch name {
	case "claude-code":
		return providers.NewClaudeCode(cfg.AI.ClaudeCode), nil
	case "litellm":
		return providers.NewOpenAICompat(providers.OpenAICompatConfig{
			Name:         "litellm",
			BaseURL:      cfg.AI.LiteLLM.BaseURL,
			APIKey:       cfg.AI.LiteLLM.APIKey,
			DefaultModel: cfg.AI.LiteLLM.DefaultModel,
			Timeout:      timeout,
		}), nil
	case "openrouter":
		return providers.NewOpenAICompat(providers.OpenAICompatConfig{
			Name:         "openrouter",
			BaseURL:      cfg.AI.OpenRouter.BaseURL,
			APIKey:       cfg.AI.OpenRouter.APIKey,
			DefaultModel: cfg.AI.OpenRouter.DefaultModel,
			Timeout:      timeout,
		}), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
```
</action>

<acceptance_criteria>
- `grep "func NewRouterFromConfig" internal/ai/factory.go` matches
- `grep "func buildProvider" internal/ai/factory.go` matches
- `grep "providers.NewClaudeCode" internal/ai/factory.go` matches
- `grep "providers.NewOpenAICompat" internal/ai/factory.go` matches
- `go build ./internal/ai/` succeeds (no compilation errors)
</acceptance_criteria>
