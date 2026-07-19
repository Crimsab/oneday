package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverModelsUsesConfiguredServerSideSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" { t.Fatalf("path = %s", r.URL.Path) }
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" { t.Fatalf("authorization = %q", got) }
		_, _ = w.Write([]byte(`{"data":[{"id":"story-model"},{"id":"image-model"}]}`))
	}))
	defer server.Close()
	cfg := Default()
	cfg.AI.LiteLLM.Enabled, cfg.AI.LiteLLM.BaseURL, cfg.AI.LiteLLM.APIKey = true, server.URL+"/v1", "test-token"
	discovery := DiscoverModels(context.Background(), cfg)
	if len(discovery.Sources) != 1 { t.Fatalf("sources = %#v", discovery.Sources) }
	if got := discovery.Sources[0]; got.ID != "litellm" || got.Status != "ready" || len(got.Models) != 2 { t.Fatalf("source = %#v", got) }
}

func TestDiscoverModelsFallsBackWhenBridgeDiscoveryIsMissing(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	cfg := Default(); cfg.AI.ImageGeneration.ImagegenBridgeURL = server.URL
	discovery := DiscoverModels(context.Background(), cfg)
	if len(discovery.Sources) != 1 || discovery.Sources[0].Status != "unavailable" { t.Fatalf("discovery = %#v", discovery) }
}
