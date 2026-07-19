package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoverModelsUsesConfiguredServerSideSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"story-model"},{"id":"image-model"}]}`))
	}))
	defer server.Close()
	cfg := Default()
	cfg.AI.ImageGeneration.ImagegenBridgeURL = ""
	cfg.AI.LiteLLM.Enabled, cfg.AI.LiteLLM.BaseURL, cfg.AI.LiteLLM.APIKey = true, server.URL+"/v1", "test-token"
	discovery := DiscoverModels(context.Background(), cfg)
	if len(discovery.Sources) != 1 {
		t.Fatalf("sources = %#v", discovery.Sources)
	}
	if got := discovery.Sources[0]; got.ID != "litellm" || got.Status != "ready" || len(got.Models) != 2 {
		t.Fatalf("source = %#v", got)
	}
	serialized, err := json.Marshal(discovery)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "test-token") {
		t.Fatal("discovery leaked the configured credential")
	}
}

func TestDiscoverModelsFallsBackWhenBridgeDiscoveryIsMissing(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	cfg := Default()
	cfg.AI.ImageGeneration.ImagegenBridgeURL = server.URL
	discovery := DiscoverModels(context.Background(), cfg)
	if len(discovery.Sources) != 1 || discovery.Sources[0].Status != "unavailable" {
		t.Fatalf("discovery = %#v", discovery)
	}
}

func TestDiscoverModelsReadsBridgeProviderMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"items":[{"name":"codex-responses","display_name":"Codex OAuth Responses","version":"0.3.0","experimental":false,"models":[{"id":"gpt-image-2"},"gpt-image-1"]}]}`))
	}))
	defer server.Close()
	cfg := Default()
	cfg.AI.ImageGeneration.ImagegenBridgeURL = server.URL
	discovery := DiscoverModels(context.Background(), cfg)
	if got := discovery.Sources[0]; got.Status != "ready" || len(got.Models) != 2 || got.Models[0] != "gpt-image-1" {
		t.Fatalf("source = %#v", got)
	}
}
