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
	Messages       []Message       `json:"messages"`
	Model          string          `json:"model,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// Response holds the result of an AI completion.
type Response struct {
	Content   string `json:"content"`
	Model     string `json:"model"`
	LatencyMs int64  `json:"latency_ms"`
	Provider  string `json:"provider"`
	Usage     Usage  `json:"usage"`
}

// ResponseFormat requests provider-side response shaping for compatible APIs.
// OpenAI-compatible providers may honor json_object/json_schema, while others
// simply ignore this field.
type ResponseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *JSONSchemaConfig `json:"json_schema,omitempty"`
}

// JSONSchemaConfig is the OpenAI-compatible nested schema envelope used when
// response_format.type is "json_schema".
type JSONSchemaConfig struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict,omitempty"`
	Schema map[string]any `json:"schema"`
}

// Provider is the interface that all AI providers must implement.
type Provider interface {
	// Name returns the provider identifier (e.g., "claude-code", "litellm", "openrouter").
	Name() string

	// Complete sends a chat completion request and returns the response.
	Complete(ctx context.Context, req Request) (Response, error)
}

// StreamChunk is a piece of a streamed AI response.
type StreamChunk struct {
	Content string // incremental text delta
	Model   string // resolved model for the stream
	Usage   Usage  // final usage snapshot when provided by the upstream
	Done    bool   // true on the final chunk (no more data)
	Error   error  // non-nil if the stream encountered an error
}

// Usage captures token and cost metadata from a provider response when
// available. Zero values mean the provider did not return the field.
type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	ReasoningTokens  int     `json:"reasoning_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CostUSD          float64 `json:"cost_usd"`
}

// StreamProvider extends Provider with streaming capability.
// Providers that support streaming implement this interface in addition to Provider.
type StreamProvider interface {
	Provider
	// Stream sends a request and returns a channel of incremental chunks.
	// The channel is closed when streaming completes (after a Done chunk).
	// The caller must consume all chunks or cancel the context to avoid goroutine leaks.
	Stream(ctx context.Context, req Request) (<-chan StreamChunk, error)
}

// EmbeddingRequest holds parameters for a text embedding request.
type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model,omitempty"`
}

// EmbeddingResponse holds the result of an embedding generation.
type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
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
