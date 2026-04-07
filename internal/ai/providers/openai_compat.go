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
