---
phase: 1
plan: 1.3
title: "AI provider router with fallback chain"
status: completed
commit: fe2681f
---

# Summary: Plan 1.3 — AI Provider Router with Fallback Chain

## What Was Built

Six files implementing the AI provider system:

| File | Purpose |
|------|---------|
| `internal/ai/provider.go` | `Provider` interface, `Message`, `Request`, `Response`, `ProviderError` types |
| `internal/ai/router.go` | `Router` with priority-ordered fallback chain |
| `internal/ai/router_test.go` | 5 unit tests covering success, fallback, all-fail, empty list, names |
| `internal/ai/providers/claudecode.go` | Claude Code CLI provider (shells out to `claude -p ... --output-format json`) |
| `internal/ai/providers/openai_compat.go` | OpenAI-compatible HTTP provider for LiteLLM + OpenRouter |
| `internal/ai/providers/openai_compat_test.go` | 3 tests with mock HTTP server (success, 500 error, empty choices) |
| `internal/aifactory/factory.go` | Factory that wires providers from `config.Config` into a `Router` |

## Design Decisions

### Import Cycle Fix
The plan placed the factory in `internal/ai/factory.go`, but this created an import cycle:
- `internal/ai` → `internal/ai/providers` → `internal/ai`

Resolution: moved factory to `internal/aifactory/factory.go` (separate package). The factory imports both `internal/ai` and `internal/ai/providers` without creating a cycle. Callers use `aifactory.NewRouterFromConfig(cfg)`.

### Provider Interface
```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req Request) (Response, error)
}
```
Non-pointer receiver, returns value types — simple and clean.

### Claude Code CLI
Messages are flattened into a single prompt with `[System]`, `[User]`, `[Assistant]` prefixes since the CLI doesn't support multi-turn natively.

### OpenAI-Compatible Provider
Single implementation handles both LiteLLM and OpenRouter. Constructed with `OpenAICompatConfig` which includes name, baseURL, apiKey, defaultModel, timeout.

## Test Results

```
go test ./internal/ai/ -v          → 5/5 PASS
go test ./internal/ai/providers/ -v → 3/3 PASS
go build ./...                     → SUCCESS
go vet ./...                       → no issues
```

## Requirements Addressed

- AI-01: Provider router with fallback chain ✓
