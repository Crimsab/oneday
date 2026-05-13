// Package aifactory wires AI providers from config into a Router.
// It lives outside internal/ai to avoid an import cycle
// (internal/ai/providers imports internal/ai for shared types).
package aifactory

import (
	"fmt"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/providers"
	"github.com/crimsab/oneday/internal/config"
)

// NewRouterFromConfig builds a Router from the application config.
// It creates provider instances for each enabled provider in the priority chain.
func NewRouterFromConfig(cfg config.Config) (*ai.Router, error) {
	enabledNames := cfg.EnabledProviders()
	if len(enabledNames) == 0 {
		return nil, fmt.Errorf("no AI providers are enabled in config")
	}

	timeout := time.Duration(cfg.AI.Generation.TimeoutSeconds) * time.Second

	providerList := make([]ai.Provider, 0, len(enabledNames))
	for _, name := range enabledNames {
		p, err := buildProvider(name, cfg, timeout)
		if err != nil {
			return nil, fmt.Errorf("building provider %s: %w", name, err)
		}
		providerList = append(providerList, p)
	}

	return ai.NewRouter(providerList)
}

func buildProvider(name string, cfg config.Config, timeout time.Duration) (ai.Provider, error) {
	switch name {
	case "codex":
		return providers.NewCodex(cfg.AI.Codex), nil
	case "claude-code":
		return providers.NewClaudeCode(cfg.AI.ClaudeCode), nil
	case "litellm":
		return providers.NewOpenAICompat(providers.OpenAICompatConfig{
			Name:         "litellm",
			BaseURL:      cfg.AI.LiteLLM.BaseURL,
			APIKey:       cfg.AI.LiteLLM.APIKey,
			DefaultModel: cfg.AI.LiteLLM.DefaultModel,
			Timeout:      timeout,
		}), nil
	case "openrouter":
		return providers.NewOpenAICompat(providers.OpenAICompatConfig{
			Name:         "openrouter",
			BaseURL:      cfg.AI.OpenRouter.BaseURL,
			APIKey:       cfg.AI.OpenRouter.APIKey,
			DefaultModel: cfg.AI.OpenRouter.DefaultModel,
			Timeout:      timeout,
		}), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
