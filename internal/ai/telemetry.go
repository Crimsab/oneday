package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

type telemetryContextKey struct{}

// TelemetryMetadata identifies the causal operation without carrying prompt
// bodies, credentials, or private reasoning into persisted diagnostics.
type TelemetryMetadata struct {
	TraceID        string            `json:"trace_id,omitempty"`
	ParentRunID    string            `json:"parent_run_id,omitempty"`
	StoryID        string            `json:"story_id,omitempty"`
	BranchID       string            `json:"branch_id,omitempty"`
	SourceCommitID string            `json:"source_commit_id,omitempty"`
	MessageID      *int64            `json:"message_id,omitempty"`
	Stage          string            `json:"stage,omitempty"`
	PromptProfile  string            `json:"prompt_profile,omitempty"`
	PromptTemplate string            `json:"prompt_template,omitempty"`
	RetryReason    string            `json:"retry_reason,omitempty"`
	SafeMetadata   map[string]string `json:"safe_metadata,omitempty"`
}

type TelemetryRef struct {
	RunID   string `json:"run_id,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

func (r TelemetryRef) Empty() bool { return r.RunID == "" }

func WithTelemetry(ctx context.Context, metadata TelemetryMetadata) context.Context {
	return context.WithValue(ctx, telemetryContextKey{}, metadata)
}

func TelemetryFromContext(ctx context.Context) TelemetryMetadata {
	if ctx == nil {
		return TelemetryMetadata{}
	}
	metadata, _ := ctx.Value(telemetryContextKey{}).(TelemetryMetadata)
	return metadata
}

type TelemetryRunStart struct {
	Metadata           TelemetryMetadata
	PromptHash         string
	ResponseSchemaHash string
	RequestConfigJSON  string
	RequestedStreaming bool
	StartedAt          time.Time
}

type TelemetryAttemptStart struct {
	RunID               string
	Sequence            int
	Provider            string
	RequestedModel      string
	ReasoningConfigJSON string
	RequestedStreaming  bool
	RetryReason         string
	StartedAt           time.Time
}

type TelemetryCompletion struct {
	Status            string
	ResolvedModel     string
	ObservedStreaming bool
	TTFTMs            int64
	DurationMs        int64
	Usage             Usage
	ErrorClass        string
	ErrorSummary      string
}

type TelemetryRecorder interface {
	StartRun(context.Context, TelemetryRunStart) (TelemetryRef, error)
	StartAttempt(context.Context, TelemetryAttemptStart) (string, error)
	FinishAttempt(context.Context, string, TelemetryCompletion) error
	FinishRun(context.Context, string, TelemetryCompletion) error
	Event(context.Context, string, string, string, map[string]any) error
}

func RequestPromptHash(req Request) string {
	payload, _ := json.Marshal(struct {
		Messages       []Message       `json:"messages"`
		Model          string          `json:"model,omitempty"`
		Temperature    float64         `json:"temperature,omitempty"`
		MaxTokens      int             `json:"max_tokens,omitempty"`
		ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	}{req.Messages, req.Model, req.Temperature, req.MaxTokens, req.ResponseFormat})
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func ResponseSchemaHash(format *ResponseFormat) string {
	if format == nil {
		return ""
	}
	payload, _ := json.Marshal(format)
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func SafeRequestConfigJSON(req Request) string {
	payload, _ := json.Marshal(map[string]any{
		"requested_model": req.Model,
		"temperature":     req.Temperature,
		"max_tokens":      req.MaxTokens,
		"response_format": responseFormatType(req.ResponseFormat),
		"plugin_ids":      pluginIDs(req.Plugins),
	})
	return string(payload)
}

func ReasoningConfigJSON() string {
	return `{"effort":"unspecified","summary":"not_persisted"}`
}

func ClassifyProviderError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "context canceled"), strings.Contains(text, "cancelled"):
		return "cancelled"
	case strings.Contains(text, "deadline"), strings.Contains(text, "timeout"):
		return "timeout"
	case strings.Contains(text, "401"), strings.Contains(text, "403"), strings.Contains(text, "unauthorized"):
		return "authentication"
	case strings.Contains(text, "429"), strings.Contains(text, "rate limit"):
		return "rate_limit"
	case strings.Contains(text, "invalid"), strings.Contains(text, "schema"):
		return "invalid_response"
	case strings.Contains(text, "connection"), strings.Contains(text, "network"), strings.Contains(text, "dns"):
		return "transport"
	default:
		return "provider_error"
	}
}

var secretLike = regexp.MustCompile(`(?i)(bearer\s+|api[_-]?key\s*[=:]\s*|token\s*[=:]\s*|password\s*[=:]\s*)[^\s,;]+`)

func SafeErrorSummary(err error) string {
	if err == nil {
		return ""
	}
	value := secretLike.ReplaceAllString(err.Error(), "$1[REDACTED]")
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 240 {
		value = value[:240]
	}
	return value
}

func RedactTelemetryPayload(payload map[string]any) map[string]any {
	redacted := make(map[string]any, len(payload))
	for key, value := range payload {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "reasoning") || strings.Contains(lower, "chain_of_thought") {
			redacted[key] = "[REDACTED]"
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			redacted[key] = RedactTelemetryPayload(typed)
		case string:
			redacted[key] = secretLike.ReplaceAllString(typed, "$1[REDACTED]")
		default:
			redacted[key] = value
		}
	}
	return redacted
}

func responseFormatType(format *ResponseFormat) string {
	if format == nil {
		return ""
	}
	return format.Type
}

func pluginIDs(plugins []Plugin) []string {
	ids := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		ids = append(ids, plugin.ID)
	}
	return ids
}
