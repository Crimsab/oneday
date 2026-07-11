package aifactory

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/config"
)

// EmbeddingProviderSpec describes a provider that can be used for RAG embeddings.
type EmbeddingProviderSpec struct {
	Name               string
	Kind               string
	BaseURL            string
	APIKey             string
	Model              string
	Dimensions         int
	SupportsEmbeddings bool
}

// SelectEmbeddingProvider picks the configured embedding provider or the first
// enabled generation provider that also supports OpenAI-compatible embeddings.
func SelectEmbeddingProvider(cfg config.Config) (EmbeddingProviderSpec, string) {
	requested := strings.TrimSpace(cfg.AI.Embedding.Provider)
	if requested == "" {
		requested = "auto"
	}

	if requested != "auto" {
		spec, reason := embeddingProviderSpecForName(cfg, requested)
		if reason != "" {
			return EmbeddingProviderSpec{}, reason
		}
		return validateEmbeddingProviderSpec(spec, requested)
	}

	for _, name := range cfg.EnabledProviders() {
		spec, reason := embeddingProviderSpecForName(cfg, name)
		if reason != "" {
			continue
		}
		spec, reason = validateEmbeddingProviderSpec(spec, name)
		if reason != "" {
			continue
		}
		return spec, ""
	}
	return EmbeddingProviderSpec{}, "no embedding-capable provider configured"
}

func validateEmbeddingProviderSpec(spec EmbeddingProviderSpec, requested string) (EmbeddingProviderSpec, string) {
	if !spec.SupportsEmbeddings {
		return EmbeddingProviderSpec{}, fmt.Sprintf("embedding provider %q does not support embeddings", requested)
	}
	if strings.TrimSpace(spec.BaseURL) == "" {
		return EmbeddingProviderSpec{}, fmt.Sprintf("embedding provider %q has no base_url configured", requested)
	}
	if strings.TrimSpace(spec.APIKey) == "" {
		if spec.Kind == "ollama" || spec.Kind == "custom" {
			return spec, ""
		}
		return EmbeddingProviderSpec{}, fmt.Sprintf("embedding provider %q has no api_key configured", requested)
	}
	return spec, ""
}

func embeddingProviderSpecForName(cfg config.Config, name string) (EmbeddingProviderSpec, string) {
	switch name {
	case "codex":
		if !cfg.AI.Codex.Enabled {
			return EmbeddingProviderSpec{}, `embedding provider "codex" is disabled`
		}
		return EmbeddingProviderSpec{
			Name:               "codex",
			SupportsEmbeddings: false,
		}, ""
	case "litellm":
		if !cfg.AI.LiteLLM.Enabled {
			return EmbeddingProviderSpec{}, `embedding provider "litellm" is disabled`
		}
		return EmbeddingProviderSpec{
			Name:               "litellm-embed",
			Kind:               "openai-compatible",
			BaseURL:            cfg.AI.LiteLLM.BaseURL,
			APIKey:             cfg.AI.LiteLLM.APIKey,
			Model:              cfg.AI.Embedding.Model,
			Dimensions:         cfg.RAG.Dimensions,
			SupportsEmbeddings: true,
		}, ""
	case "openrouter":
		if !cfg.AI.OpenRouter.Enabled {
			return EmbeddingProviderSpec{}, `embedding provider "openrouter" is disabled`
		}
		return EmbeddingProviderSpec{
			Name:               "openrouter",
			Kind:               "openai-compatible",
			BaseURL:            cfg.AI.OpenRouter.BaseURL,
			APIKey:             cfg.AI.OpenRouter.APIKey,
			Model:              cfg.AI.Embedding.Model,
			Dimensions:         cfg.RAG.Dimensions,
			SupportsEmbeddings: true,
		}, ""
	case "local":
		if !cfg.AI.Embedding.Local.Enabled {
			return EmbeddingProviderSpec{}, `embedding provider "local" is disabled`
		}
		return EmbeddingProviderSpec{
			Name:               "local-" + cfg.AI.Embedding.Local.Type,
			Kind:               cfg.AI.Embedding.Local.Type,
			BaseURL:            cfg.AI.Embedding.Local.BaseURL,
			Model:              cfg.AI.Embedding.Local.Model,
			Dimensions:         cfg.AI.Embedding.Local.Dimensions,
			SupportsEmbeddings: true,
		}, ""
	case "claude-code":
		if !cfg.AI.ClaudeCode.Enabled {
			return EmbeddingProviderSpec{}, `embedding provider "claude-code" is disabled`
		}
		return EmbeddingProviderSpec{
			Name:               "claude-code",
			SupportsEmbeddings: false,
		}, ""
	default:
		return EmbeddingProviderSpec{}, fmt.Sprintf("unknown embedding provider %q", name)
	}
}
