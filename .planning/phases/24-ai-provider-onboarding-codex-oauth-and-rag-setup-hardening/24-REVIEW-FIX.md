---
phase: 24-ai-provider-onboarding-codex-oauth-and-rag-setup-hardening
review: 24-REVIEW.md
status: fixed
fixed: 2026-05-13
---

# Phase 24 Review Fixes

## WR-01: RAG enabled with missing embedding API key

Fixed in `internal/aifactory/embedding.go`.

- Explicit provider selection now rejects missing `api_key`.
- Auto provider selection skips embedding-capable providers without an API key and continues to the next viable provider.
- Added tests in `internal/aifactory/embedding_test.go`.
- Updated TUI selector tests to configure API keys when expecting RAG-enabled providers.

## IN-01: Setup-generated config branches missing tests

Fixed in `cmd/oneday/main.go` and `cmd/oneday/main_test.go`.

- Extracted setup choice mapping into `setupConfigForChoice`.
- Added table tests for Codex, LiteLLM, and OpenRouter setup branches.
- Covered provider order, provider enablement, RAG enablement, placeholder keys, and config validation.

## Verification

- `go test ./cmd/oneday ./internal/aifactory ./internal/tui ./internal/ai/providers ./internal/rag ./internal/config`
- `go test ./...`
- `go run ./cmd/oneday doctor`
- `make friend-safe-check`
- `make verify`
