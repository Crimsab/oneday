---
phase: 24-ai-provider-onboarding-codex-oauth-and-rag-setup-hardening
reviewed: 2026-05-13T13:48:52Z
depth: standard
files_reviewed: 19
files_reviewed_list:
  - cmd/oneday/main.go
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/config/dotenv.go
  - internal/aifactory/embedding.go
  - internal/tui/app.go
  - internal/tui/app_test.go
  - internal/tui/app_smoke_test.go
  - internal/rag/embeddings.go
  - internal/ai/providers/codex.go
  - internal/ai/providers/codex_test.go
  - internal/ai/providers/openai_compat.go
  - internal/ai/providers/openai_compat_test.go
  - README.md
  - config.example.yaml
  - .env.example
  - Makefile
  - scripts/friend-safe-check.sh
  - scripts/release-gate.sh
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: fixed
---

# Phase 24: Code Review Report

**Reviewed:** 2026-05-13T13:48:52Z
**Depth:** standard
**Files Reviewed:** 19
**Status:** fixed

## Fix Status

WR-01 was fixed after review. See `24-REVIEW-FIX.md`.

## Summary

Reviewed the AI setup, Codex OAuth provider, OpenAI-compatible provider, RAG embedding selection, config/env loading, docs, and release hygiene scripts. The scoped Go package tests pass:

```bash
go test ./internal/config ./internal/ai/providers ./internal/rag ./internal/tui
```

No critical security issues were found. The main correctness concern is that RAG can still be mounted as "enabled" with an embedding provider that cannot authenticate because its API key is empty.

## Warnings

### WR-01: RAG is enabled even when the selected embedding provider has no API key

**File:** `internal/aifactory/embedding.go:64`

**Issue:** `SelectEmbeddingProvider` accepts `litellm` and `openrouter` as embedding-capable as long as they are enabled and have a `base_url`, but it does not reject an empty `APIKey`. With the tracked `config.example.yaml`, missing environment variables are expanded to empty strings by `config.Load`, so `App.buildRAG` logs RAG as enabled and mounts a pipeline. Retrieval and summarization then fail later inside `OpenAICompat.requireAPIKey`, producing silent/no-context RAG behavior instead of the intended clear "disabled, reason" setup hardening.

**Fix:** Treat missing auth as a selection failure for providers that require keys, and add a regression test for auto-selection with an enabled provider whose API key is empty.

```go
func validateEmbeddingProviderSpec(spec EmbeddingProviderSpec, requested string) (EmbeddingProviderSpec, string) {
	if !spec.SupportsEmbeddings {
		return EmbeddingProviderSpec{}, fmt.Sprintf("embedding provider %q does not support embeddings", requested)
	}
	if strings.TrimSpace(spec.BaseURL) == "" {
		return EmbeddingProviderSpec{}, fmt.Sprintf("embedding provider %q has no base_url configured", requested)
	}
	if strings.TrimSpace(spec.APIKey) == "" {
		return EmbeddingProviderSpec{}, fmt.Sprintf("embedding provider %q has no api_key configured", requested)
	}
	return spec, ""
}
```

## Info

### IN-01: Setup-generated config paths are not covered by tests

**File:** `cmd/oneday/main.go:117`

**Issue:** The new `runSetup` path mutates provider priority, provider enablement, RAG enablement, env placeholders, and writes `config.yaml`, but there are no tests that assert the generated Codex-only, LiteLLM, and OpenRouter configs remain valid and preserve the intended RAG behavior. This is a test coverage gap around the main onboarding flow.

**Fix:** Extract the provider-choice-to-config logic into a small pure helper, then table-test choices `1`, `2`, and `3` for `Validate()`, enabled providers, placeholder keys, and Codex-only `RAG.Enabled == false`.

---

_Reviewed: 2026-05-13T13:48:52Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
