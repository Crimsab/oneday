package providers

import (
	"bufio"
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

// openAIEmbeddingRequest is the request body for /v1/embeddings.
type openAIEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// openAIEmbeddingResponse is the response from /v1/embeddings.
type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
}

// Embed generates an embedding vector for the given text using the /v1/embeddings endpoint.
func (o *OpenAICompat) Embed(ctx context.Context, req ai.EmbeddingRequest) (ai.EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = o.defaultModel
	}

	body := openAIEmbeddingRequest{
		Model: model,
		Input: req.Input,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("marshaling embedding request: %w", err)
	}

	url := o.baseURL + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("creating embedding request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("HTTP embedding request to %s: %w", o.name, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("reading embedding response from %s: %w", o.name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return ai.EmbeddingResponse{}, fmt.Errorf("%s embedding returned status %d: %s", o.name, resp.StatusCode, string(respBody))
	}

	var embResp openAIEmbeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return ai.EmbeddingResponse{}, fmt.Errorf("parsing embedding response from %s: %w", o.name, err)
	}

	if len(embResp.Data) == 0 {
		return ai.EmbeddingResponse{}, fmt.Errorf("%s returned no embedding data", o.name)
	}

	return ai.EmbeddingResponse{
		Embedding: embResp.Data[0].Embedding,
		Model:     embResp.Model,
	}, nil
}

// Stream implements ai.StreamProvider using Server-Sent Events (SSE).
// It calls /v1/chat/completions with stream:true and parses the event stream.
func (o *OpenAICompat) Stream(ctx context.Context, req ai.Request) (<-chan ai.StreamChunk, error) {
	model := req.Model
	if model == "" {
		model = o.defaultModel
	}

	body := map[string]interface{}{
		"model":       model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling stream request: %w", err)
	}

	url := o.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP stream request to %s: %w", o.name, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s stream returned status %d", o.name, resp.StatusCode)
	}

	ch := make(chan ai.StreamChunk, 32)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// sseChunk mirrors the SSE delta payload from OpenAI-compatible endpoints.
		type sseChunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- ai.StreamChunk{Done: true}
				return
			}
			var chunk sseChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue // skip malformed chunks
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- ai.StreamChunk{Content: chunk.Choices[0].Delta.Content}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- ai.StreamChunk{Error: fmt.Errorf("%s stream scanner: %w", o.name, err)}
		}
		ch <- ai.StreamChunk{Done: true}
	}()

	return ch, nil
}
