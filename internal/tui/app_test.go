package tui

import (
	"testing"

	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/config"
)

func TestSelectEmbeddingProviderUsesFirstEmbeddingCapableProvider(t *testing.T) {
	cfg := config.Default()
	cfg.AI.ClaudeCode.Enabled = true
	cfg.AI.OpenRouter.Enabled = true
	cfg.AI.OpenRouter.APIKey = "openrouter-key"
	cfg.AI.LiteLLM.Enabled = false
	cfg.AI.ProviderPriority = []string{"claude-code", "openrouter", "litellm"}

	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		t.Fatalf("selectEmbeddingProvider returned unexpected reason: %s", reason)
	}
	if spec.Name != "openrouter" {
		t.Fatalf("selectEmbeddingProvider picked %q, want openrouter", spec.Name)
	}
	if spec.BaseURL != cfg.AI.OpenRouter.BaseURL {
		t.Fatalf("selectEmbeddingProvider base URL = %q, want %q", spec.BaseURL, cfg.AI.OpenRouter.BaseURL)
	}
}

func TestSelectEmbeddingProviderHonorsExplicitProvider(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Embedding.Provider = "openrouter"
	cfg.AI.LiteLLM.Enabled = true
	cfg.AI.OpenRouter.Enabled = true
	cfg.AI.OpenRouter.APIKey = "openrouter-key"

	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		t.Fatalf("selectEmbeddingProvider returned unexpected reason: %s", reason)
	}
	if spec.Name != "openrouter" {
		t.Fatalf("selectEmbeddingProvider picked %q, want openrouter", spec.Name)
	}
}

func TestSelectEmbeddingProviderRejectsExplicitUnsupportedProvider(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Embedding.Provider = "claude-code"
	cfg.AI.ClaudeCode.Enabled = true

	_, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason == "" {
		t.Fatal("selectEmbeddingProvider reason = empty, want unsupported-provider explanation")
	}
}

func TestSelectEmbeddingProviderReportsMissingSupport(t *testing.T) {
	cfg := config.Default()
	cfg.AI.ClaudeCode.Enabled = true
	cfg.AI.LiteLLM.Enabled = false
	cfg.AI.OpenRouter.Enabled = false
	cfg.AI.ProviderPriority = []string{"claude-code"}

	_, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason == "" {
		t.Fatal("selectEmbeddingProvider reason = empty, want a missing-provider explanation")
	}
}
