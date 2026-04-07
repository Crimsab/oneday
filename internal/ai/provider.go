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

// StreamChunk is a piece of a streamed AI response.
type StreamChunk struct {
	Content string // incremental text delta
	Done    bool   // true on the final chunk (no more data)
	Error   error  // non-nil if the stream encountered an error
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
