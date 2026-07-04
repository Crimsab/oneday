package aifactory

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/config"
)

func TestSelectEmbeddingProviderSkipsAutoProviderWithoutAPIKey(t *testing.T) {
	cfg := config.Default()
	cfg.AI.LiteLLM.Enabled = true
	cfg.AI.LiteLLM.APIKey = ""
	cfg.AI.OpenRouter.Enabled = true
	cfg.AI.OpenRouter.APIKey = "openrouter-key"
	cfg.AI.ProviderPriority = []string{"litellm", "openrouter"}

	spec, reason := SelectEmbeddingProvider(cfg)
	if reason != "" {
		t.Fatalf("SelectEmbeddingProvider reason = %q, want none", reason)
	}
	if spec.Name != "openrouter" {
		t.Fatalf("SelectEmbeddingProvider picked %q, want openrouter", spec.Name)
	}
}

func TestSelectEmbeddingProviderRejectsExplicitProviderWithoutAPIKey(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Embedding.Provider = "litellm"
	cfg.AI.LiteLLM.Enabled = true
	cfg.AI.LiteLLM.APIKey = ""

	_, reason := SelectEmbeddingProvider(cfg)
	if !strings.Contains(reason, "api_key") {
		t.Fatalf("reason = %q, want missing api_key", reason)
	}
}

func TestSelectEmbeddingProviderAllowsLocalWithoutAPIKey(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Embedding.Provider = "local"
	cfg.AI.Embedding.Local.Enabled = true
	cfg.AI.Embedding.Local.Type = "ollama"
	cfg.AI.Embedding.Local.BaseURL = "http://127.0.0.1:11434"
	cfg.AI.Embedding.Local.Model = "test-local-embedding-model"
	cfg.AI.Embedding.Local.Dimensions = 1024

	spec, reason := SelectEmbeddingProvider(cfg)
	if reason != "" {
		t.Fatalf("SelectEmbeddingProvider reason = %q, want none", reason)
	}
	if spec.Kind != "ollama" || spec.Model != "test-local-embedding-model" || spec.Dimensions != 1024 {
		t.Fatalf("unexpected local spec: %#v", spec)
	}
}
