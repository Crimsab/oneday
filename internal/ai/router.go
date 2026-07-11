package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Router routes AI requests through a priority chain of providers.
// It tries each provider in order and falls back to the next on failure.
type Router struct {
	providers []Provider
	recorder  TelemetryRecorder
}

func (r *Router) SetTelemetryRecorder(recorder TelemetryRecorder) {
	if r != nil {
		r.recorder = recorder
	}
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
	ref, runStarted := r.startTelemetryRun(ctx, req, false)

	for index, p := range r.providers {
		attemptStarted := time.Now()
		attemptID := r.startTelemetryAttempt(ctx, ref, index+1, p.Name(), req.Model, false, retryReason(index), attemptStarted)
		resp, err := p.Complete(ctx, req)
		if err != nil {
			r.finishTelemetryAttempt(ctx, attemptID, failedTelemetry(err, time.Since(attemptStarted).Milliseconds()))
			errors = append(errors, fmt.Sprintf("%s: %s", p.Name(), err))
			continue
		}
		if resp.Provider == "" {
			resp.Provider = p.Name()
		}
		completion := successfulTelemetry(resp.Model, false, 0, time.Since(attemptStarted).Milliseconds(), resp.Usage)
		r.finishTelemetryAttempt(ctx, attemptID, completion)
		r.finishTelemetryRun(ctx, ref, completion, runStarted)
		resp.Telemetry = ref
		return resp, nil
	}

	err := fmt.Errorf("all AI providers failed:\n  %s", strings.Join(errors, "\n  "))
	r.finishTelemetryRun(ctx, ref, failedTelemetry(err, time.Since(runStarted).Milliseconds()), runStarted)
	return Response{}, err
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

// Stream tries each provider in order, preferring StreamProvider implementations.
// If no provider supports streaming it falls back to Complete and wraps the result
// in a single-chunk channel to simulate streaming.
// Returns the channel, the name of the provider used, and any error.
func (r *Router) Stream(ctx context.Context, req Request) (<-chan StreamChunk, string, error) {
	var errors []string
	ref, runStarted := r.startTelemetryRun(ctx, req, true)
	sequence := 0

	// First pass: prefer providers that natively support streaming.
	for _, p := range r.providers {
		sp, ok := p.(StreamProvider)
		if !ok {
			continue
		}
		sequence++
		attemptStarted := time.Now()
		attemptID := r.startTelemetryAttempt(ctx, ref, sequence, p.Name(), req.Model, true, retryReason(sequence-1), attemptStarted)
		ch, err := sp.Stream(ctx, req)
		if err != nil {
			r.finishTelemetryAttempt(ctx, attemptID, failedTelemetry(err, time.Since(attemptStarted).Milliseconds()))
			errors = append(errors, fmt.Sprintf("%s(stream): %s", p.Name(), err))
			continue
		}
		return r.observeStream(ctx, ch, ref, attemptID, attemptStarted, runStarted), p.Name(), nil
	}

	// Second pass: fall back to Complete and simulate a one-chunk stream.
	for _, p := range r.providers {
		sequence++
		attemptStarted := time.Now()
		attemptID := r.startTelemetryAttempt(ctx, ref, sequence, p.Name(), req.Model, true, retryReason(sequence-1), attemptStarted)
		resp, err := p.Complete(ctx, req)
		if err != nil {
			r.finishTelemetryAttempt(ctx, attemptID, failedTelemetry(err, time.Since(attemptStarted).Milliseconds()))
			errors = append(errors, fmt.Sprintf("%s: %s", p.Name(), err))
			continue
		}
		if resp.Provider == "" {
			resp.Provider = p.Name()
		}
		completion := successfulTelemetry(resp.Model, false, 0, time.Since(attemptStarted).Milliseconds(), resp.Usage)
		r.finishTelemetryAttempt(ctx, attemptID, completion)
		r.finishTelemetryRun(ctx, ref, completion, runStarted)
		ch := make(chan StreamChunk, 2)
		go func() {
			ch <- StreamChunk{Content: resp.Content, Model: resp.Model, Usage: resp.Usage, Telemetry: ref}
			ch <- StreamChunk{Model: resp.Model, Usage: resp.Usage, Done: true, Telemetry: ref}
			close(ch)
		}()
		return ch, p.Name(), nil
	}

	err := fmt.Errorf("all AI providers failed:\n  %s", strings.Join(errors, "\n  "))
	r.finishTelemetryRun(ctx, ref, failedTelemetry(err, time.Since(runStarted).Milliseconds()), runStarted)
	return nil, "", err
}

func (r *Router) startTelemetryRun(ctx context.Context, req Request, streaming bool) (TelemetryRef, time.Time) {
	started := time.Now()
	if r == nil || r.recorder == nil {
		return TelemetryRef{}, started
	}
	ref, err := r.recorder.StartRun(ctx, TelemetryRunStart{
		Metadata:           TelemetryFromContext(ctx),
		PromptHash:         RequestPromptHash(req),
		ResponseSchemaHash: ResponseSchemaHash(req.ResponseFormat),
		RequestConfigJSON:  SafeRequestConfigJSON(req),
		RequestedStreaming: streaming,
		StartedAt:          started,
	})
	if err != nil {
		return TelemetryRef{}, started
	}
	return ref, started
}

func (r *Router) startTelemetryAttempt(ctx context.Context, ref TelemetryRef, sequence int, provider, model string, streaming bool, retryReason string, started time.Time) string {
	if r == nil || r.recorder == nil || ref.Empty() {
		return ""
	}
	id, err := r.recorder.StartAttempt(ctx, TelemetryAttemptStart{RunID: ref.RunID, Sequence: sequence, Provider: provider, RequestedModel: model, ReasoningConfigJSON: ReasoningConfigJSON(), RequestedStreaming: streaming, RetryReason: retryReason, StartedAt: started})
	if err != nil {
		return ""
	}
	return id
}

func (r *Router) finishTelemetryAttempt(ctx context.Context, id string, completion TelemetryCompletion) {
	if r != nil && r.recorder != nil && id != "" {
		_ = r.recorder.FinishAttempt(ctx, id, completion)
	}
}

func (r *Router) finishTelemetryRun(ctx context.Context, ref TelemetryRef, completion TelemetryCompletion, started time.Time) {
	if r == nil || r.recorder == nil || ref.Empty() {
		return
	}
	if completion.DurationMs <= 0 {
		completion.DurationMs = time.Since(started).Milliseconds()
	}
	_ = r.recorder.FinishRun(ctx, ref.RunID, completion)
}

func (r *Router) observeStream(ctx context.Context, source <-chan StreamChunk, ref TelemetryRef, attemptID string, attemptStarted, runStarted time.Time) <-chan StreamChunk {
	out := make(chan StreamChunk, 16)
	go func() {
		defer close(out)
		var usage Usage
		var model string
		var ttft int64
		observed := false
		finished := false
		for chunk := range source {
			chunk.Telemetry = ref
			if chunk.Model != "" {
				model = chunk.Model
			}
			if usageNonZero(chunk.Usage) {
				usage = chunk.Usage
			}
			if chunk.Content != "" && !observed {
				observed = true
				ttft = time.Since(attemptStarted).Milliseconds()
				if r.recorder != nil && !ref.Empty() {
					_ = r.recorder.Event(ctx, ref.RunID, attemptID, "first_token", map[string]any{"offset_ms": ttft})
				}
			}
			if chunk.Error != nil {
				completion := failedTelemetry(chunk.Error, time.Since(attemptStarted).Milliseconds())
				completion.ObservedStreaming = observed
				completion.TTFTMs = ttft
				r.finishTelemetryAttempt(ctx, attemptID, completion)
				r.finishTelemetryRun(ctx, ref, completion, runStarted)
				out <- chunk
				return
			}
			out <- chunk
			if chunk.Done {
				completion := successfulTelemetry(model, observed, ttft, time.Since(attemptStarted).Milliseconds(), usage)
				r.finishTelemetryAttempt(ctx, attemptID, completion)
				r.finishTelemetryRun(ctx, ref, completion, runStarted)
				finished = true
				return
			}
		}
		if !finished {
			err := fmt.Errorf("provider stream closed before completion")
			completion := failedTelemetry(err, time.Since(attemptStarted).Milliseconds())
			completion.ObservedStreaming = observed
			completion.TTFTMs = ttft
			r.finishTelemetryAttempt(ctx, attemptID, completion)
			r.finishTelemetryRun(ctx, ref, completion, runStarted)
		}
	}()
	return out
}

func successfulTelemetry(model string, streamed bool, ttft, duration int64, usage Usage) TelemetryCompletion {
	return TelemetryCompletion{Status: "succeeded", ResolvedModel: model, ObservedStreaming: streamed, TTFTMs: ttft, DurationMs: duration, Usage: usage}
}

func failedTelemetry(err error, duration int64) TelemetryCompletion {
	status := "failed"
	if ClassifyProviderError(err) == "cancelled" {
		status = "cancelled"
	}
	return TelemetryCompletion{Status: status, DurationMs: duration, ErrorClass: ClassifyProviderError(err), ErrorSummary: SafeErrorSummary(err)}
}

func retryReason(index int) string {
	if index > 0 {
		return "provider_fallback"
	}
	return ""
}

func usageNonZero(usage Usage) bool {
	return usage.TotalTokens != 0 || usage.PromptTokens != 0 || usage.CompletionTokens != 0 || usage.ReasoningTokens != 0 || usage.CachedPromptTokens != 0 || usage.CostUSD != 0
}
