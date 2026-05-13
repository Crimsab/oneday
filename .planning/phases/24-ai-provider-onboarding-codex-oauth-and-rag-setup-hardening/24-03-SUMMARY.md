---
phase: 24-ai-provider-onboarding-codex-oauth-and-rag-setup-hardening
plan: 03
subsystem: rag
tags: [rag, embeddings, openrouter, litellm, sqlite]
requires:
  - phase: 24-01
    provides: embedding model and provider config roles
provides:
  - shared embedding provider selection helper
  - explicit RAG enabled/disabled status strings
  - embedding dimension mismatch validation
  - reserved local embedding config stub
affects: [rag, doctor, tui]
tech-stack:
  added: []
  patterns: [shared provider selection, explicit degraded-mode logging]
key-files:
  created:
    - internal/aifactory/embedding.go
  modified:
    - internal/tui/app.go
    - internal/tui/app_test.go
    - internal/rag/embeddings.go
    - internal/config/config.go
    - config.example.yaml
key-decisions:
  - "RAG provider selection lives in aifactory so doctor and TUI use the same rules."
  - "Codex, Claude Code, and local config stubs are not treated as embedding-capable."
  - "Local RAG embeddings are represented as disabled config only until a real offline backend exists."
patterns-established:
  - "RAG startup uses exact status strings: RAG: enabled and RAG: disabled, reason: ..."
requirements-completed: [RAG-SETUP-01]
duration: 30min
completed: 2026-05-13
---

# Phase 24-03 Summary

**RAG embedding selection hardened with shared provider logic, clear disabled states, and vector dimension validation**

## Accomplishments

- Moved embedding-capable provider selection into `internal/aifactory`.
- Reused the same selection logic from TUI and doctor.
- Added exact RAG status logging for enabled and disabled cases.
- Added dimension validation so incompatible embedding responses fail clearly.
- Added disabled local embedding config space without pretending offline embeddings are implemented.

## Files Created/Modified

- `internal/aifactory/embedding.go` - Shared embedding provider selector.
- `internal/tui/app.go` - Uses shared selector and explicit RAG logs.
- `internal/tui/app_test.go` - Updated selector tests.
- `internal/rag/embeddings.go` - Validates embedding dimensions.
- `internal/config/config.go` and `config.example.yaml` - Added disabled local embedding config stub.

## Verification

- `go test ./internal/config ./internal/aifactory ./internal/rag ./internal/ai/providers ./internal/tui`
- `go test ./...`
- `go run ./cmd/oneday doctor`

## Deviations from Plan

Local embedding fallback was implemented as a config/documentation stub only. That is intentional: adding a real local embedding model would require model packaging/downloads and runtime dependencies outside this phase.

## User Setup Required

For RAG, configure LiteLLM or OpenRouter with an embedding-capable route for `text-embedding-3-small`.
