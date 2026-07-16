package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockProvider is a test double for ai.Provider.
type mockProvider struct {
	name    string
	err     error
	content string
}

func TestObserveStreamClosesPromptlyWhenContextIsCancelled(t *testing.T) {
	r, err := NewRouter([]Provider{&mockProvider{name: "unused"}})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	source := make(chan StreamChunk, 64)
	for i := 0; i < cap(source); i++ {
		source <- StreamChunk{Content: "delta"}
	}
	out := r.observeStream(ctx, source, TelemetryRef{}, "", time.Now(), time.Now())
	time.Sleep(10 * time.Millisecond)
	cancel()

	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("observed stream did not close after cancellation")
		}
	}
}

type telemetryStreamProvider struct {
	mockProvider
	chunks    []StreamChunk
	streamErr error
}

func (m *telemetryStreamProvider) Stream(_ context.Context, _ Request) (<-chan StreamChunk, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	ch := make(chan StreamChunk, len(m.chunks))
	for _, chunk := range m.chunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

type recordedAttempt struct {
	start  TelemetryAttemptStart
	finish TelemetryCompletion
}

type recordingTelemetry struct {
	mu          sync.Mutex
	runs        []TelemetryRunStart
	runFinishes []TelemetryCompletion
	attempts    []recordedAttempt
	events      []string
}

func (r *recordingTelemetry) StartRun(_ context.Context, start TelemetryRunStart) (TelemetryRef, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs = append(r.runs, start)
	return TelemetryRef{RunID: fmt.Sprintf("run-%d", len(r.runs)), TraceID: "trace-test"}, nil
}

func (r *recordingTelemetry) StartAttempt(_ context.Context, start TelemetryAttemptStart) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts = append(r.attempts, recordedAttempt{start: start})
	return fmt.Sprintf("attempt-%d", len(r.attempts)), nil
}

func (r *recordingTelemetry) FinishAttempt(_ context.Context, id string, finish TelemetryCompletion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var index int
	if _, err := fmt.Sscanf(id, "attempt-%d", &index); err == nil && index > 0 && index <= len(r.attempts) {
		r.attempts[index-1].finish = finish
	}
	return nil
}

func (r *recordingTelemetry) FinishRun(_ context.Context, _ string, finish TelemetryCompletion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runFinishes = append(r.runFinishes, finish)
	return nil
}

func (r *recordingTelemetry) Event(_ context.Context, _, _, eventType string, _ map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, eventType)
	return nil
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

func TestRouterCompleteWithProviderDoesNotFallback(t *testing.T) {
	primary := &mockProvider{name: "primary", err: fmt.Errorf("unavailable")}
	fallback := &mockProvider{name: "fallback", content: "must not be used"}
	router, err := NewRouter([]Provider{primary, fallback})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := router.CompleteWithProvider(context.Background(), "primary", Request{}); err == nil {
		t.Fatal("CompleteWithProvider succeeded despite the selected provider failure")
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

func TestRouterTelemetryPersistsFallbackAttemptsAndSafePromptIdentity(t *testing.T) {
	r, err := NewRouter([]Provider{
		&mockProvider{name: "primary", err: fmt.Errorf("timeout api_key=super-secret")},
		&mockProvider{name: "fallback", content: "recovered"},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := &recordingTelemetry{}
	r.SetTelemetryRecorder(recorder)
	ctx := WithTelemetry(context.Background(), TelemetryMetadata{TraceID: "trace-parent", Stage: "narrator", PromptProfile: "narrator-v1", StoryID: "story-1"})
	resp, err := r.Complete(ctx, Request{Messages: []Message{{Role: RoleUser, Content: "private story prompt"}}, MaxTokens: 100})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Telemetry.RunID == "" || len(recorder.runs) != 1 || len(recorder.attempts) != 2 || len(recorder.runFinishes) != 1 {
		t.Fatalf("telemetry ref=%+v runs=%d attempts=%d finishes=%d", resp.Telemetry, len(recorder.runs), len(recorder.attempts), len(recorder.runFinishes))
	}
	if recorder.attempts[0].finish.ErrorClass != "timeout" || strings.Contains(recorder.attempts[0].finish.ErrorSummary, "super-secret") {
		t.Fatalf("unsafe or unclassified error: %+v", recorder.attempts[0].finish)
	}
	if recorder.attempts[1].start.RetryReason != "provider_fallback" || recorder.attempts[1].finish.Status != "succeeded" {
		t.Fatalf("fallback attempt=%+v", recorder.attempts[1])
	}
	encoded := recorder.runs[0].RequestConfigJSON
	if strings.Contains(encoded, "private story prompt") || recorder.runs[0].PromptHash == "" {
		t.Fatalf("request config leaked prompt or missed hash: %s / %s", encoded, recorder.runs[0].PromptHash)
	}
}

func TestRouterTelemetryObservesNativeStreamingAndFirstToken(t *testing.T) {
	provider := &telemetryStreamProvider{mockProvider: mockProvider{name: "stream"}, chunks: []StreamChunk{
		{Content: "hello", Model: "stream-model"},
		{Usage: Usage{PromptTokens: 2, CompletionTokens: 1, TotalTokens: 3, CostUSD: 0.001}, Model: "stream-model", Done: true},
	}}
	r, err := NewRouter([]Provider{provider})
	if err != nil {
		t.Fatal(err)
	}
	recorder := &recordingTelemetry{}
	r.SetTelemetryRecorder(recorder)
	stream, _, err := r.Stream(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "test"}}})
	if err != nil {
		t.Fatal(err)
	}
	var ref TelemetryRef
	for chunk := range stream {
		ref = chunk.Telemetry
	}
	if ref.RunID == "" || len(recorder.attempts) != 1 || len(recorder.runFinishes) != 1 {
		t.Fatalf("stream telemetry ref=%+v attempts=%d finishes=%d", ref, len(recorder.attempts), len(recorder.runFinishes))
	}
	finish := recorder.attempts[0].finish
	if !finish.ObservedStreaming || finish.Usage.TotalTokens != 3 || finish.Status != "succeeded" {
		t.Fatalf("stream finish=%+v", finish)
	}
	if len(recorder.events) != 1 || recorder.events[0] != "first_token" {
		t.Fatalf("events=%v", recorder.events)
	}
}

func TestRedactTelemetryPayloadRemovesSecretsAndReasoning(t *testing.T) {
	redacted := RedactTelemetryPayload(map[string]any{
		"api_key":   "secret",
		"reasoning": "private chain",
		"message":   "request failed bearer raw-token",
		"nested":    map[string]any{"password": "hidden", "safe": "visible"},
	})
	encoded, _ := json.Marshal(redacted)
	text := string(encoded)
	for _, forbidden := range []string{"secret", "private chain", "raw-token", "hidden"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("redaction leaked %q in %s", forbidden, text)
		}
	}
	if !strings.Contains(text, "visible") {
		t.Fatalf("redaction removed safe value: %s", text)
	}
}
