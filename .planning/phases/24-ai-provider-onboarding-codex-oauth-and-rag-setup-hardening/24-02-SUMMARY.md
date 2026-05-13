---
phase: 24-ai-provider-onboarding-codex-oauth-and-rag-setup-hardening
plan: 02
subsystem: cli
tags: [setup, doctor, codex, oauth, env]
requires:
  - phase: 24-01
    provides: explicit model roles and embedding defaults
provides:
  - first-time setup for Codex OAuth, LiteLLM, and OpenRouter
  - read-only doctor command for local tools, auth, provider smoke, embeddings, and RAG readiness
  - non-destructive .env creation that respects already-exported variables
affects: [onboarding, support, release-readiness]
tech-stack:
  added: []
  patterns: [read-only diagnostics, non-destructive setup]
key-files:
  created: []
  modified:
    - cmd/oneday/main.go
    - README.md
    - .env.example
key-decisions:
  - "Codex-only setup disables RAG explicitly because Codex does not provide embeddings."
  - "doctor reports failures as actionable status lines instead of exiting on provider smoke failures."
  - ".env placeholders are written only when .env is absent and no shell value already exists."
patterns-established:
  - "Setup writes local config; doctor observes and reports without writing."
requirements-completed: [AI-SETUP-02, AI-SETUP-03]
duration: 35min
completed: 2026-05-13
---

# Phase 24-02 Summary

**First-time setup and doctor diagnostics for Codex OAuth, provider auth, model smoke, embeddings, and RAG readiness**

## Accomplishments

- Added `oneday doctor` / `go run ./cmd/oneday doctor` for OS/tool checks, Codex login status, provider smoke, embedding smoke, and RAG status.
- Kept setup non-destructive: existing `config.yaml` is left untouched with the existing message.
- Made Codex-first setup use local `codex login`, default `gpt-5.5`, reasoning `off`, and explicit RAG disabled status.
- Adjusted `.env` generation so exported env vars are not shadowed.

## Files Created/Modified

- `cmd/oneday/main.go` - Added doctor command, provider smoke, embedding smoke, and setup refinements.
- `README.md` - Added setup/doctor/Codex OAuth guidance.
- `.env.example` - Added comments with empty placeholders only.

## Verification

- `go run ./cmd/oneday doctor`
- `go test ./...`
- `make verify`

## Deviations from Plan

No functional deviations. Commits were intentionally skipped because the worktree is shared/dirty.

## User Setup Required

Run `codex login` for Codex OAuth if not already logged in, then run `oneday doctor`.
