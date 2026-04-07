package ai

import (
	"context"
	"fmt"
	"strings"
)

// Router routes AI requests through a priority chain of providers.
// It tries each provider in order and falls back to the next on failure.
type Router struct {
	providers []Provider
}

// NewRouter creates a router with the given providers in priority order.
// Providers should already be filtered to only enabled ones.
func NewRouter(providers []Provider) (*Router, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one AI provider is required")
	}
	return &Router{providers: providers}, nil
}

// Complete tries each provider in order until one succeeds.
// Returns the first successful response or an error if all providers fail.
func (r *Router) Complete(ctx context.Context, req Request) (Response, error) {
	var errors []string

	for _, p := range r.providers {
		resp, err := p.Complete(ctx, req)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", p.Name(), err))
			continue
		}
		return resp, nil
	}

	return Response{}, fmt.Errorf("all AI providers failed:\n  %s", strings.Join(errors, "\n  "))
}

// Providers returns the list of providers in priority order.
func (r *Router) Providers() []Provider {
	return r.providers
}

// ProviderNames returns the names of all providers in order.
func (r *Router) ProviderNames() []string {
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return names
}
