---
phase: 24-ai-provider-onboarding-codex-oauth-and-rag-setup-hardening
plan: 04
subsystem: release
tags: [errors, release, hygiene, friend-safe]
requires:
  - phase: 24-02
    provides: setup and doctor flows
provides:
  - actionable LiteLLM/OpenRouter auth errors
  - actionable Codex CLI/auth errors
  - friend-safe release hygiene check
  - setup and provider docs
affects: [support, release, onboarding]
tech-stack:
  added: []
  patterns: [actionable auth errors, release hygiene script]
key-files:
  created:
    - scripts/friend-safe-check.sh
  modified:
    - internal/ai/providers/openai_compat.go
    - internal/ai/providers/openai_compat_test.go
    - internal/ai/providers/codex.go
    - internal/ai/providers/codex_test.go
    - README.md
    - .env.example
    - Makefile
    - scripts/release-gate.sh
key-decisions:
  - "401/403 errors name the exact env var and suggest oneday doctor."
  - "Missing Codex binary and auth failures mention codex login and oneday doctor."
  - "Release hygiene blocks tracked config/env/data/db/binary artifacts."
patterns-established:
  - "Provider errors are translated at provider boundaries before router aggregation."
requirements-completed: [AI-SETUP-04, DIST-01]
duration: 35min
completed: 2026-05-13
---

# Phase 24-04 Summary

**Actionable provider auth failures plus friend-safe release hygiene for sharing OneDay without local secrets or artifacts**

## Accomplishments

- Added actionable missing-key and 401/403 errors for LiteLLM/OpenRouter generation, streaming, and embeddings.
- Added actionable Codex CLI missing/auth failure errors.
- Added `make friend-safe-check` and wired it into release gate after verification.
- Documented friend-safe sharing, Codex OAuth, doctor, model defaults, and embedding limits.

## Files Created/Modified

- `internal/ai/providers/openai_compat.go` - Provider auth error translation.
- `internal/ai/providers/openai_compat_test.go` - Missing-key and auth failure coverage.
- `internal/ai/providers/codex.go` - Codex CLI/auth error translation.
- `internal/ai/providers/codex_test.go` - Missing binary and auth failure coverage.
- `scripts/friend-safe-check.sh` - Release/share hygiene guard.
- `Makefile` and `scripts/release-gate.sh` - Friend-safe target and gate integration.
- `README.md` and `.env.example` - Setup and sharing documentation.

## Verification

- `go test ./internal/config ./internal/aifactory ./internal/rag ./internal/ai/providers ./internal/tui`
- `go test ./...`
- `make verify`
- `make friend-safe-check`

## Deviations from Plan

`make release-check` was not run because the current worktree is intentionally dirty and the release gate requires a clean git worktree before building clean provenance artifacts. The new friend-safe check itself passed.

## User Setup Required

None for code. Users need real virtual keys in `.env` for LiteLLM/OpenRouter and `codex login` for Codex OAuth.
