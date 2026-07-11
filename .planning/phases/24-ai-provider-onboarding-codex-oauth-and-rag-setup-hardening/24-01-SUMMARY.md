---
phase: 24-ai-provider-onboarding-codex-oauth-and-rag-setup-hardening
plan: 01
subsystem: config
tags: [ai, models, codex, embeddings]
requires: []
provides:
  - explicit generation, utility, repair, and embedding model roles
  - config validation for utility model presence
  - config template documenting Codex and embedding boundaries
affects: [ai-provider-setup, rag, onboarding]
tech-stack:
  added: []
  patterns: [typed config defaults, env-expanded runtime config]
key-files:
  created: []
  modified:
    - internal/config/config.go
    - internal/config/config_test.go
    - config.example.yaml
key-decisions:
  - "Generation stays on gpt-5.5 for Codex by default."
  - "Utility/ancillary model defaults to gpt-5.4-mini but does not override an explicit repair model."
  - "Embedding defaults remain text-embedding-3-small and are separate from Codex generation."
patterns-established:
  - "Model roles are explicit fields instead of implicit reuse of the narrator model."
requirements-completed: [AI-SETUP-01, RAG-SETUP-01]
duration: 20min
completed: 2026-05-13
---

# Phase 24-01 Summary

**Explicit AI model roles with Codex generation, gpt-5.4-mini utility fallback, and text-embedding-3-small embeddings preserved**

## Accomplishments

- Added `ai.generation.utility_model` with validation and default `gpt-5.4-mini`.
- Kept repair model precedence intact: utility model is used only when no explicit repair model is configured.
- Documented Codex as generation-only and preserved `text-embedding-3-small` as the embedding default.

## Files Created/Modified

- `internal/config/config.go` - Added utility model defaults, validation, and repair fallback behavior.
- `internal/config/config_test.go` - Covered default, validation, and repair candidate ordering.
- `config.example.yaml` - Documented Codex, utility model, and embedding roles.

## Verification

- `go test ./internal/config ./internal/aifactory ./internal/rag ./internal/ai/providers ./internal/tui`
- `go test ./...`
- `make verify`

## Deviations from Plan

No functional deviations. Commits were intentionally skipped because this operational repo already had unrelated dirty changes and the user asked to avoid interlacing local config/work.

## User Setup Required

None.
