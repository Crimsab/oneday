package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
)

type recordingTranslationProvider struct {
	name    string
	request ai.Request
	calls   int
}

func (provider *recordingTranslationProvider) Name() string { return provider.name }

func (provider *recordingTranslationProvider) Complete(_ context.Context, request ai.Request) (ai.Response, error) {
	provider.calls++
	provider.request = request
	return ai.Response{
		Content:  "Porta [[ODP_0001]]",
		Model:    request.Model,
		Provider: provider.name,
	}, nil
}

func TestRunGatewayTranslateUsesOnlySelectedProvider(t *testing.T) {
	unused := &recordingTranslationProvider{name: "unused"}
	selected := &recordingTranslationProvider{name: "selected"}
	router, err := ai.NewRouter([]ai.Provider{unused, selected})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	input := `{
		"text":"Door [[ODP_0001]]",
		"source_language":"en",
		"target_language":"it",
		"provider":"selected",
		"model":"translator-model",
		"style":"literary",
		"glossary":{"Door":"Porta"}
	}`
	var output bytes.Buffer
	if err := runGatewayTranslate(context.Background(), config.Config{}, router, strings.NewReader(input), &output); err != nil {
		t.Fatalf("runGatewayTranslate: %v", err)
	}
	var response gatewayTranslateResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != "" || response.Text != "Porta [[ODP_0001]]" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if unused.calls != 0 || selected.calls != 1 {
		t.Fatalf("provider calls: unused=%d selected=%d", unused.calls, selected.calls)
	}
	if selected.request.Model != "translator-model" || selected.request.Temperature != 0.45 {
		t.Fatalf("unexpected model controls: %+v", selected.request)
	}
	prompt := selected.request.Messages[1].Content
	for _, required := range []string{"Target language: it", "Style: literary", "Door => Porta", "Door [[ODP_0001]]"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt does not contain %q: %s", required, prompt)
		}
	}
}

func TestRunGatewayTranslateDoesNotFallbackFromUnknownProvider(t *testing.T) {
	configured := &recordingTranslationProvider{name: "configured"}
	router, err := ai.NewRouter([]ai.Provider{configured})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	input := `{"text":"Hello","target_language":"it","provider":"missing"}`
	var output bytes.Buffer
	if err := runGatewayTranslate(context.Background(), config.Config{}, router, strings.NewReader(input), &output); err != nil {
		t.Fatalf("runGatewayTranslate: %v", err)
	}
	var response gatewayTranslateResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == "" || configured.calls != 0 {
		t.Fatalf("expected safe provider error without fallback: response=%+v calls=%d", response, configured.calls)
	}
}
