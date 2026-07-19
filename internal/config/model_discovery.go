package config

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const modelDiscoveryResponseLimit = 1024 * 1024

// ModelDiscovery is a redacted, server-side model catalog. Endpoint and
// credential details deliberately stay in Config and never cross the gateway.
type ModelDiscovery struct {
	Sources []ModelDiscoverySource `json:"sources"`
}

type ModelDiscoverySource struct {
	ID        string   `json:"id"`
	Status    string   `json:"status"`
	Models    []string `json:"models"`
	CheckedAt string   `json:"checked_at"`
}

func DiscoverModels(ctx context.Context, cfg Config) ModelDiscovery {
	client := &http.Client{Timeout: 5 * time.Second}
	sources := []modelDiscoveryCandidate{}
	if cfg.AI.LiteLLM.Enabled {
		sources = append(sources, modelDiscoveryCandidate{"litellm", cfg.AI.LiteLLM.BaseURL, cfg.AI.LiteLLM.APIKey, "/models", false})
	}
	if cfg.AI.OpenRouter.Enabled {
		sources = append(sources, modelDiscoveryCandidate{"openrouter", cfg.AI.OpenRouter.BaseURL, cfg.AI.OpenRouter.APIKey, "/models", false})
	}
	if compatible, ok := cfg.AI.ImageGeneration.Providers[ImageProviderOpenAICompatible]; ok && strings.TrimSpace(compatible.BaseURL) != "" {
		sources = append(sources, modelDiscoveryCandidate{"openai-compatible", compatible.BaseURL, compatible.APIKey, "/models", false})
	}
	if strings.TrimSpace(cfg.AI.ImageGeneration.ImagegenBridgeURL) != "" {
		sources = append(sources, modelDiscoveryCandidate{"imagegen-bridge", cfg.AI.ImageGeneration.ImagegenBridgeURL, cfg.AI.ImageGeneration.ImagegenBridgeToken, "/v1/providers", true})
	}
	result := ModelDiscovery{Sources: make([]ModelDiscoverySource, 0, len(sources))}
	for _, source := range sources {
		result.Sources = append(result.Sources, discoverModelSource(ctx, client, source))
	}
	return result
}

type modelDiscoveryCandidate struct {
	id      string
	baseURL string
	token   string
	path    string
	bridge  bool
}

func discoverModelSource(ctx context.Context, client *http.Client, source modelDiscoveryCandidate) ModelDiscoverySource {
	result := ModelDiscoverySource{
		ID:        source.id,
		Status:    "unavailable",
		Models:    []string{},
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
	endpoint, err := joinDiscoveryEndpoint(source.baseURL, source.path)
	if err != nil {
		return result
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return result
	}
	if token := strings.TrimSpace(source.token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return result
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, modelDiscoveryResponseLimit+1))
	if err != nil || len(body) > modelDiscoveryResponseLimit {
		return result
	}
	models := parseDiscoveredModels(body, source.bridge)
	if len(models) == 0 {
		result.Status = "empty"
		return result
	}
	result.Status, result.Models = "ready", models
	return result
}

func joinDiscoveryEndpoint(base, path string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawQuery = ""
	return u.String(), nil
}

func parseDiscoveredModels(body []byte, bridge bool) []string {
	var payload map[string]json.RawMessage
	if json.Unmarshal(body, &payload) != nil {
		return []string{}
	}
	models := append(modelIDs(payload["data"]), modelIDs(payload["models"])...)
	if bridge {
		models = append(models, providerModels(payload["items"])...)
		// Early 0.3 development builds used "providers"; retaining it is
		// harmless and keeps discovery additive across those payloads.
		models = append(models, providerModels(payload["providers"])...)
	}
	return uniqueSortedModels(models)
}

func providerModels(raw json.RawMessage) []string {
	var providers []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &providers) != nil {
		return nil
	}
	models := []string{}
	for _, provider := range providers {
		models = append(models, modelIDs(provider["models"])...)
	}
	return models
}

// modelIDs accepts OpenAI /models entries as well as bridge 0.3 provider
// metadata, whose model list may contain either strings or {id: ...} objects.
func modelIDs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	models := make([]string, 0, len(items))
	for _, item := range items {
		var name string
		if json.Unmarshal(item, &name) == nil {
			models = append(models, name)
			continue
		}
		var entry struct {
			ID    string `json:"id"`
			Model string `json:"model"`
			Name  string `json:"name"`
		}
		if json.Unmarshal(item, &entry) == nil {
			models = append(models, entry.ID, entry.Model, entry.Name)
		}
	}
	return models
}

func uniqueSortedModels(models []string) []string {
	seen := map[string]struct{}{}
	clean := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; !ok {
			seen[model] = struct{}{}
			clean = append(clean, model)
		}
	}
	sort.Strings(clean)
	return clean
}
