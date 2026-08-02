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
	if len(discovery.Sources) != 2 {
		t.Fatalf("sources = %#v", discovery.Sources)
	}
	if got := discoverySource(t, discovery, "codex"); got.Status != "catalog" || !sameStrings(got.Models, curatedCodexNarrativeModels) {
		t.Fatalf("codex source = %#v", got)
	}
	if got := discoverySource(t, discovery, "litellm"); got.Status != "ready" || len(got.Models) != 2 {
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

func TestDiscoverModelsKeepsProviderCatalogsSeparate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		switch r.URL.Path {
		case "/litellm/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"litellm/story"}]}`))
		case "/openrouter/models":
			// OpenRouter entries carry both an ID and a display name. Only the ID
			// may be reused as a provider request model.
			_, _ = w.Write([]byte(`{"data":[{"id":"openrouter/story","name":"Readable display name"}]}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer server.Close()

	cfg := Default()
	cfg.AI.ImageGeneration.ImagegenBridgeURL = ""
	cfg.AI.LiteLLM.Enabled, cfg.AI.LiteLLM.BaseURL, cfg.AI.LiteLLM.APIKey = true, server.URL+"/litellm", "test-token"
	cfg.AI.OpenRouter.Enabled, cfg.AI.OpenRouter.BaseURL, cfg.AI.OpenRouter.APIKey = true, server.URL+"/openrouter", "test-token"

	discovery := DiscoverModels(context.Background(), cfg)
	litellm := discoverySource(t, discovery, "litellm")
	openrouter := discoverySource(t, discovery, "openrouter")
	if !sameStrings(litellm.Models, []string{"litellm/story"}) {
		t.Fatalf("LiteLLM models = %#v", litellm.Models)
	}
	if !sameStrings(openrouter.Models, []string{"openrouter/story"}) {
		t.Fatalf("OpenRouter models = %#v", openrouter.Models)
	}
	if strings.Contains(strings.Join(litellm.Models, ","), "openrouter") || strings.Contains(strings.Join(openrouter.Models, ","), "litellm") {
		t.Fatalf("provider models were mixed: %#v", discovery.Sources)
	}
}

func TestDiscoverModelsFallsBackWhenBridgeDiscoveryIsMissing(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	cfg := Default()
	cfg.AI.ImageGeneration.ImagegenBridgeURL = server.URL
	discovery := DiscoverModels(context.Background(), cfg)
	if len(discovery.Sources) != 2 || discoverySource(t, discovery, "imagegen-bridge").Status != "unavailable" {
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
	if got := discoverySource(t, discovery, "imagegen-bridge"); got.Status != "ready" || len(got.Models) != 2 || got.Models[0] != "gpt-image-1" {
		t.Fatalf("source = %#v", got)
	}
}

func TestDiscoverModelsPrefersPerLaunchBridgeEnvironment(t *testing.T) {
	configServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("configured bridge must not receive discovery: %s", r.URL.String())
	}))
	defer configServer.Close()
	runtimeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/providers" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer runtime-bridge-token" {
			t.Fatalf("unexpected bridge authorization")
		}
		_, _ = w.Write([]byte(`{"items":[{"models":["gpt-image-2"]}]}`))
	}))
	defer runtimeServer.Close()

	t.Setenv("ONEDAY_IMAGEGEN_BRIDGE_URL", runtimeServer.URL)
	t.Setenv("ONEDAY_IMAGEGEN_BRIDGE_TOKEN", "runtime-bridge-token")
	cfg := Default()
	cfg.AI.ImageGeneration.ImagegenBridgeURL = configServer.URL
	cfg.AI.ImageGeneration.ImagegenBridgeToken = "config-bridge-token"

	discovery := DiscoverModels(context.Background(), cfg)
	bridge := discoverySource(t, discovery, "imagegen-bridge")
	if bridge.Status != "ready" || !sameStrings(bridge.Models, []string{"gpt-image-2"}) {
		t.Fatalf("bridge discovery = %#v", bridge)
	}
	serialized, err := json.Marshal(discovery)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "runtime-bridge-token") || strings.Contains(string(serialized), "config-bridge-token") {
		t.Fatal("bridge credential leaked from discovery")
	}
}

func discoverySource(t *testing.T, discovery ModelDiscovery, id string) ModelDiscoverySource {
	t.Helper()
	for _, source := range discovery.Sources {
		if source.ID == id {
			return source
		}
	}
	t.Fatalf("missing discovery source %q in %#v", id, discovery.Sources)
	return ModelDiscoverySource{}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
