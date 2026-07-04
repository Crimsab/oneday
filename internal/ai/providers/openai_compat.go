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
	"sync"
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
	capsMu       sync.RWMutex
	capsLoaded   bool
	modelCaps    map[string]modelCapabilities
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
	Model          string             `json:"model"`
	Messages       []ai.Message       `json:"messages"`
	Temperature    float64            `json:"temperature,omitempty"`
	MaxTokens      int                `json:"max_tokens,omitempty"`
	ResponseFormat *ai.ResponseFormat `json:"response_format,omitempty"`
	Plugins        []ai.Plugin        `json:"plugins,omitempty"`
	Provider       *ai.ProviderConfig `json:"provider,omitempty"`
	Stream         bool               `json:"stream,omitempty"`
	StreamOptions  *streamOptions     `json:"stream_options,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// openAIChatResponse is the response from /v1/chat/completions.
type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string         `json:"model"`
	Usage openAIUsageDTO `json:"usage"`
}

type openAICompatHTTPError struct {
	StatusCode int
	Body       string
}

func (e *openAICompatHTTPError) Error() string {
	return fmt.Sprintf("status %d: %s", e.StatusCode, e.Body)
}

type modelCapabilities struct {
	Known                  bool
	SupportsResponseFormat bool
	SupportsStructured     bool
}

type openAIModelCatalogResponse struct {
	Data []openAIModelCatalogEntry `json:"data"`
}

type openAIModelCatalogEntry struct {
	ID                  string   `json:"id"`
	SupportedParameters []string `json:"supported_parameters"`
}

type openAIUsageDTO struct {
	PromptTokens        int     `json:"prompt_tokens"`
	CompletionTokens    int     `json:"completion_tokens"`
	TotalTokens         int     `json:"total_tokens"`
	Cost                float64 `json:"cost"`
	PromptTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
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
		modelCaps:    map[string]modelCapabilities{},
	}
}

func (o *OpenAICompat) Name() string {
	return o.name
}

// Complete sends a chat completion request to the OpenAI-compatible endpoint.
func (o *OpenAICompat) Complete(ctx context.Context, req ai.Request) (ai.Response, error) {
	start := time.Now()
	body, err := o.buildChatRequest(ctx, req, false)
	if err != nil {
		return ai.Response{}, err
	}
	content, resolvedModel, usage, err := o.completeOnce(ctx, body)
	if err != nil {
		httpErr, ok := err.(*openAICompatHTTPError)
		if body.ResponseFormat != nil && ok && shouldRetryWithoutResponseFormat(httpErr) {
			content, resolvedModel, usage, err = o.completeOnce(ctx, withoutResponseFormat(body))
		}
		if err != nil {
			return ai.Response{}, fmt.Errorf("HTTP request to %s: %w", o.name, err)
		}
	}

	return ai.Response{
		Content:   content,
		Model:     resolvedModel,
		LatencyMs: time.Since(start).Milliseconds(),
		Provider:  o.name,
		Usage:     usage,
	}, nil
}

func (o *OpenAICompat) completeOnce(ctx context.Context, body openAIChatRequest) (string, string, ai.Usage, error) {
	if err := o.requireAPIKey(); err != nil {
		return "", "", ai.Usage{}, err
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", "", ai.Usage{}, fmt.Errorf("marshaling request: %w", err)
	}

	url := o.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", ai.Usage{}, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return "", "", ai.Usage{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", ai.Usage{}, fmt.Errorf("reading response from %s: %w", o.name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", ai.Usage{}, &openAICompatHTTPError{
			StatusCode: resp.StatusCode,
			Body:       o.actionableHTTPError(resp.StatusCode, string(respBody)),
		}
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", "", ai.Usage{}, fmt.Errorf("parsing response from %s: %w", o.name, err)
	}

	if len(chatResp.Choices) == 0 {
		return "", "", ai.Usage{}, fmt.Errorf("%s returned no choices", o.name)
	}

	return chatResp.Choices[0].Message.Content, chatResp.Model, usageFromDTO(chatResp.Usage), nil
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
	if err := o.requireAPIKey(); err != nil {
		return ai.EmbeddingResponse{}, err
	}
	model, err := o.resolveModel(req.Model)
	if err != nil {
		return ai.EmbeddingResponse{}, err
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
		return ai.EmbeddingResponse{}, fmt.Errorf("%s embedding returned status %d: %s", o.name, resp.StatusCode, o.actionableHTTPError(resp.StatusCode, string(respBody)))
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
	body, err := o.buildChatRequest(ctx, req, true)
	if err != nil {
		return nil, err
	}
	resp, err := o.openStream(ctx, body)
	if err != nil {
		httpErr, ok := err.(*openAICompatHTTPError)
		if body.ResponseFormat != nil && ok && shouldRetryWithoutResponseFormat(httpErr) {
			resp, err = o.openStream(ctx, withoutResponseFormat(body))
		}
		if err != nil {
			httpErr, ok := err.(*openAICompatHTTPError)
			if ok && shouldFallbackStreamToComplete(httpErr) {
				return o.completeAsStream(ctx, req)
			}
			return nil, fmt.Errorf("HTTP stream request to %s: %w", o.name, err)
		}
	}

	ch := make(chan ai.StreamChunk, 32)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// sseChunk mirrors the SSE delta payload from OpenAI-compatible endpoints.
		type sseChunk struct {
			Model   string         `json:"model"`
			Usage   openAIUsageDTO `json:"usage"`
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
			if chunk.Usage.TotalTokens > 0 || chunk.Usage.Cost > 0 {
				ch <- ai.StreamChunk{
					Model: chunk.Model,
					Usage: usageFromDTO(chunk.Usage),
				}
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- ai.StreamChunk{
					Content: chunk.Choices[0].Delta.Content,
					Model:   chunk.Model,
				}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- ai.StreamChunk{Error: fmt.Errorf("%s stream scanner: %w", o.name, err)}
		}
		ch <- ai.StreamChunk{Done: true}
	}()

	return ch, nil
}

func (o *OpenAICompat) buildChatRequest(ctx context.Context, req ai.Request, stream bool) (openAIChatRequest, error) {
	model, err := o.resolveModel(req.Model)
	if err != nil {
		return openAIChatRequest{}, err
	}

	body := openAIChatRequest{
		Model:          model,
		Messages:       req.Messages,
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
		ResponseFormat: o.selectResponseFormat(ctx, model, req.ResponseFormat),
		Plugins:        req.Plugins,
		Provider:       req.Provider,
	}
	if stream {
		body.Stream = true
		body.StreamOptions = &streamOptions{IncludeUsage: true}
	}
	o.applyStructuredJSONGuards(&body)
	return body, nil
}

func withoutResponseFormat(body openAIChatRequest) openAIChatRequest {
	body.ResponseFormat = nil
	body.Plugins = nil
	body.Provider = nil
	return body
}

func (o *OpenAICompat) openStream(ctx context.Context, body openAIChatRequest) (*http.Response, error) {
	if err := o.requireAPIKey(); err != nil {
		return nil, err
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
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("reading error response from %s: %w", o.name, readErr)
		}
		return nil, &openAICompatHTTPError{
			StatusCode: resp.StatusCode,
			Body:       o.actionableHTTPError(resp.StatusCode, string(respBody)),
		}
	}

	return resp, nil
}

func (o *OpenAICompat) requireAPIKey() error {
	if strings.TrimSpace(o.apiKey) != "" {
		return nil
	}
	switch o.name {
	case "litellm", "litellm-embed":
		return fmt.Errorf("LiteLLM authentication missing: set ONEDAY_LITELLM_API_KEY in .env or your shell, then run `oneday doctor`")
	case "openrouter":
		return fmt.Errorf("OpenRouter authentication missing: set ONEDAY_OPENROUTER_API_KEY in .env or your shell, then run `oneday doctor`")
	default:
		return nil
	}
}

func (o *OpenAICompat) resolveModel(requested string) (string, error) {
	model := strings.TrimSpace(requested)
	if model == "" {
		model = strings.TrimSpace(o.defaultModel)
	}
	if model != "" {
		return model, nil
	}
	switch o.name {
	case "litellm", "litellm-embed":
		return "", fmt.Errorf("LiteLLM model missing: set ai.litellm.default_model in config.yaml or choose a LiteLLM model in the browser Options panel")
	case "openrouter":
		return "", fmt.Errorf("OpenRouter model missing: set ai.openrouter.default_model in config.yaml or choose an OpenRouter model in the browser Options panel")
	default:
		return "", fmt.Errorf("%s model missing: set a request model or provider default model in config.yaml", o.name)
	}
}

func (o *OpenAICompat) actionableHTTPError(statusCode int, body string) string {
	body = sanitizeHTTPErrorBody(body)
	if statusCode != http.StatusUnauthorized && statusCode != http.StatusForbidden {
		return body
	}
	switch o.name {
	case "litellm", "litellm-embed":
		return fmt.Sprintf("LiteLLM authentication failed: check ONEDAY_LITELLM_API_KEY and the LiteLLM virtual key, then run `oneday doctor` (%s)", body)
	case "openrouter":
		return fmt.Sprintf("OpenRouter authentication failed: check ONEDAY_OPENROUTER_API_KEY and OpenRouter account credits, then run `oneday doctor` (%s)", body)
	default:
		return body
	}
}

func sanitizeHTTPErrorBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "empty response body"
	}
	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.ReplaceAll(body, "\r", " ")
	if len(body) > 500 {
		return body[:500] + "...(truncated)"
	}
	return body
}

func (o *OpenAICompat) completeAsStream(ctx context.Context, req ai.Request) (<-chan ai.StreamChunk, error) {
	resp, err := o.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan ai.StreamChunk, 3)
	go func() {
		defer close(ch)
		if resp.Content != "" {
			ch <- ai.StreamChunk{
				Content: resp.Content,
				Model:   resp.Model,
				Usage:   resp.Usage,
			}
		}
		ch <- ai.StreamChunk{
			Model: resp.Model,
			Usage: resp.Usage,
			Done:  true,
		}
	}()
	return ch, nil
}

func (o *OpenAICompat) applyStructuredJSONGuards(body *openAIChatRequest) {
	if body == nil || body.ResponseFormat == nil {
		return
	}
	if !supportsOpenRouterExtensions(o.name) {
		return
	}
	if !hasPlugin(body.Plugins, "response-healing") {
		body.Plugins = append(body.Plugins, ai.Plugin{ID: "response-healing"})
	}
	if body.Provider == nil {
		body.Provider = &ai.ProviderConfig{RequireParameters: true}
		return
	}
	body.Provider.RequireParameters = true
}

func supportsOpenRouterExtensions(providerName string) bool {
	switch providerName {
	case "openrouter", "litellm":
		return true
	default:
		return false
	}
}

func hasPlugin(plugins []ai.Plugin, id string) bool {
	for _, plugin := range plugins {
		if plugin.ID == id {
			return true
		}
	}
	return false
}

func usageFromDTO(dto openAIUsageDTO) ai.Usage {
	return ai.Usage{
		PromptTokens:       dto.PromptTokens,
		CompletionTokens:   dto.CompletionTokens,
		ReasoningTokens:    dto.CompletionTokensDetails.ReasoningTokens,
		TotalTokens:        dto.TotalTokens,
		CachedPromptTokens: dto.PromptTokensDetails.CachedTokens,
		CostUSD:            dto.Cost,
	}
}

func (o *OpenAICompat) selectResponseFormat(ctx context.Context, model string, requested *ai.ResponseFormat) *ai.ResponseFormat {
	if requested == nil {
		return nil
	}

	caps, ok := o.lookupModelCapabilities(ctx, model)
	if !ok {
		return requested
	}

	switch requested.Type {
	case "json_object":
		if caps.SupportsResponseFormat {
			return requested
		}
	case "json_schema":
		if caps.SupportsStructured {
			return requested
		}
	}

	return nil
}

func (o *OpenAICompat) lookupModelCapabilities(ctx context.Context, model string) (modelCapabilities, bool) {
	o.capsMu.RLock()
	if o.capsLoaded {
		caps, ok := o.modelCaps[model]
		o.capsMu.RUnlock()
		if !ok {
			return modelCapabilities{}, false
		}
		return caps, caps.Known
	}
	caps, ok := o.modelCaps[model]
	o.capsMu.RUnlock()
	if ok {
		return caps, caps.Known
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/models", nil)
	if err != nil {
		o.markCapabilitiesLoaded()
		return modelCapabilities{}, false
	}
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		o.markCapabilitiesLoaded()
		return modelCapabilities{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		o.markCapabilitiesLoaded()
		return modelCapabilities{}, false
	}

	var payload openAIModelCatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		o.markCapabilitiesLoaded()
		return modelCapabilities{}, false
	}

	index := map[string]modelCapabilities{}
	for _, entry := range payload.Data {
		index[entry.ID] = modelCapabilities{
			Known:                  true,
			SupportsResponseFormat: contains(entry.SupportedParameters, "response_format"),
			SupportsStructured:     contains(entry.SupportedParameters, "structured_outputs"),
		}
	}

	o.capsMu.Lock()
	for id, entry := range index {
		o.modelCaps[id] = entry
	}
	o.capsLoaded = true
	caps, ok = o.modelCaps[model]
	o.capsMu.Unlock()
	if !ok {
		return modelCapabilities{}, false
	}
	return caps, caps.Known
}

func (o *OpenAICompat) markCapabilitiesLoaded() {
	o.capsMu.Lock()
	o.capsLoaded = true
	o.capsMu.Unlock()
}

func shouldRetryWithoutResponseFormat(err *openAICompatHTTPError) bool {
	if err == nil {
		return false
	}
	if err.StatusCode != http.StatusBadRequest && err.StatusCode != http.StatusUnprocessableEntity {
		return false
	}

	body := strings.ToLower(err.Body)
	keywords := []string{
		"response_format",
		"json_schema",
		"structured",
		"schema",
		"unsupported",
		"not supported",
		"invalid schema",
	}
	for _, keyword := range keywords {
		if strings.Contains(body, keyword) {
			return true
		}
	}
	return false
}

func shouldFallbackStreamToComplete(err *openAICompatHTTPError) bool {
	if err == nil {
		return false
	}
	if err.StatusCode != http.StatusBadRequest &&
		err.StatusCode != http.StatusNotImplemented &&
		err.StatusCode != http.StatusMethodNotAllowed &&
		err.StatusCode != http.StatusServiceUnavailable {
		return false
	}

	body := strings.ToLower(err.Body)
	keywords := []string{
		"stream",
		"streaming",
		"event stream",
		"not implemented",
		"not supported",
		"unsupported",
	}
	for _, keyword := range keywords {
		if strings.Contains(body, keyword) {
			return true
		}
	}
	return false
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
