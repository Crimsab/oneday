---
phase: 24-ai-provider-onboarding-codex-oauth-and-rag-setup-hardening
verified: 2026-05-13T13:49:49Z
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
---

# Phase 24: AI Provider Onboarding, Codex OAuth, and RAG Setup Hardening Verification Report

**Phase Goal:** Make OneDay easy and safe to hand to another player by providing a complete first-time AI setup flow, Codex OAuth defaults, explicit ancillary model routing, deterministic embedding/RAG configuration, and diagnostics that explain provider failures before gameplay starts.
**Verified:** 2026-05-13T13:49:49Z
**Status:** passed
**Re-verification:** No - initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A first-time player can run `oneday setup` and end with a working `config.yaml` plus optional `.env` without manually editing secrets into tracked files. | VERIFIED | `cmd/oneday/main.go` implements `wantsSetup`/`runSetup`; isolated `/tmp` setup run created `config.yaml`, preserved env-based key placeholders, and printed the no-RAG warning for Codex-only setup. |
| 2 | Codex OAuth can be selected as the primary generation provider, with `gpt-5.5` as narrator default and `gpt-5.4-mini` as utility/repair/ancillary default. | VERIFIED | `internal/config/config.go` defaults `AI.Codex.Model` to `gpt-5.5` and `Generation.UtilityModel` to `gpt-5.4-mini`; setup option `1) Codex OAuth` enables Codex first. |
| 3 | RAG embeddings are configured explicitly: setup either provisions a compatible remote embedding provider using `text-embedding-3-small`, or disables RAG with a clear warning when no embedding provider is available. | VERIFIED | `aifactory.SelectEmbeddingProvider` only returns LiteLLM/OpenRouter as embedding-capable; TUI and doctor print `RAG: enabled...` or `RAG: disabled, reason...`; Codex-only setup disables RAG. |
| 4 | `oneday doctor` checks OS, Go, Codex login, provider reachability, model smoke responses, env/config consistency, and embedding/RAG viability. | VERIFIED | `runDoctor` prints tool/env/config/model sections, runs provider smoke, selects embedding provider, and runs embedding smoke when RAG is viable. Local spot-check printed Codex login OK, Provider smoke OK, and RAG disabled reason. |
| 5 | LiteLLM/OpenRouter 401 and missing-key failures produce actionable messages naming the missing env var or disabled provider instead of raw upstream errors. | VERIFIED | `openai_compat.go` translates missing keys and 401/403 for LiteLLM/OpenRouter with `ONEDAY_LITELLM_API_KEY`, `ONEDAY_OPENROUTER_API_KEY`, and `oneday doctor`; provider tests cover these paths. |
| 6 | Release/shared artifacts remain friend-safe: no local `config.yaml`, `.env`, `oneday_data`, binaries, databases, archives, or secrets are included by default. | VERIFIED | `.gitignore` covers local config/env/data/binaries/build/dist/release/archive/database patterns; `scripts/friend-safe-check.sh` passed and is wired into `release-gate.sh`. |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/config/config.go` | AI schema/defaults/validation/model roles | VERIFIED | Explicit Codex, utility, ASCII, and embedding model defaults; embedding provider allowlist excludes Codex/Claude Code. |
| `internal/aifactory/factory.go` | Provider construction from config | VERIFIED | `NewRouterFromConfig` is used by app startup and doctor. |
| `internal/aifactory/embedding.go` | Shared embedding provider selection | VERIFIED | Selects only LiteLLM/OpenRouter for embeddings; returns clear reasons for unsupported/disabled/missing providers. |
| `cmd/oneday/main.go` | Setup and doctor CLI commands | VERIFIED | Command dispatch, non-destructive setup, doctor smoke checks, env loading, RAG status, and embedding smoke are implemented. |
| `internal/config/dotenv.go` | `.env` loading preserving exported env vars | VERIFIED | Used before config load in normal startup and doctor. |
| `internal/tui/app.go` | Runtime RAG provider selection and no-RAG degradation | VERIFIED | `buildRAG` uses shared selector, logs enabled/disabled state, and returns nil instead of crashing when unavailable. |
| `internal/rag/embeddings.go` | Embedding call and dimension checks | VERIFIED | Empty embedding and dimension mismatch return contextual errors. |
| `internal/ai/providers/openai_compat.go` | OpenAI-compatible chat/stream/embedding auth errors | VERIFIED | Missing-key and HTTP auth translation covers generation, stream setup, and embeddings. |
| `internal/ai/providers/codex.go` | Codex CLI auth/setup errors | VERIFIED | Missing CLI and auth/login failures mention `codex login` and `oneday doctor`. |
| `README.md`, `.env.example`, `config.example.yaml` | Friend-safe setup and config docs | VERIFIED | Document setup/doctor/Codex OAuth/model defaults/RAG behavior; examples keep secret values empty or env-based. |
| `.gitignore`, `Makefile`, `scripts/friend-safe-check.sh`, `scripts/release-gate.sh` | Friend-safe release hygiene | VERIFIED | Hygiene check passed; release gate invokes it after verification. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/config/config.go` | `config.example.yaml` | Matching YAML keys/defaults | VERIFIED | Defaults and example both include `gpt-5.5`, `gpt-5.4-mini`, `text-embedding-3-small`, and embedding provider guidance. |
| `cmd/oneday/main.go` | `internal/aifactory/factory.go` | Doctor builds providers from loaded config | VERIFIED | `runDoctor` calls `aifactory.NewRouterFromConfig(cfg)` for provider smoke. |
| `internal/tui/app.go` | `internal/ai/providers/openai_compat.go` | Runtime RAG constructs OpenAI-compatible embedding provider | VERIFIED | Planned grep expected old selector location, but actual wiring is `app.go -> aifactory.SelectEmbeddingProvider -> providers.NewOpenAICompat`; this is substantive and shared with doctor. |
| `README.md` | `cmd/oneday/main.go` | Documents implemented setup/doctor commands | VERIFIED | README documents `go run ./cmd/oneday setup`, `go run ./cmd/oneday doctor`, and friend-safe handoff. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `runDoctor` | `cfg`, `router`, embedding spec | `.env` + `config.Load` + `aifactory.NewRouterFromConfig` + `SelectEmbeddingProvider` | Yes | VERIFIED |
| `buildRAG` | embedding provider/model | app config passed to `aifactory.SelectEmbeddingProvider` and `providers.NewOpenAICompat` | Yes | VERIFIED |
| `OpenAICompat.Embed` | embedding vector | `/v1/embeddings` HTTP response parsed into `ai.EmbeddingResponse` | Yes when configured; actionable error otherwise | VERIFIED |
| `RepairModelCandidates` | repair model list | explicit repair model, fallback list, utility model only when repair missing | Yes | VERIFIED |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Targeted phase packages pass | `go test ./internal/config ./internal/aifactory ./internal/rag ./internal/ai/providers ./internal/tui ./cmd/oneday` | All packages passed | PASS |
| Full test suite passes | `go test ./...` | All packages passed | PASS |
| Doctor runs read-only diagnostics | `go run ./cmd/oneday doctor` | Printed `OneDay doctor`, Codex login OK, provider smoke OK, RAG disabled reason, and embedding smoke skip | PASS |
| Isolated setup creates safe config | Built `/tmp/oneday-phase24-verify`, ran setup in `mktemp` with Codex choice | Wrote config with `gpt-5.5`, `gpt-5.4-mini`, `text-embedding-3-small`; no repo files touched | PASS |
| Friend-safe hygiene passes | `make friend-safe-check` | Printed `friend-safe release hygiene` and found no tracked forbidden artifacts | PASS |
| Version command works | `go run ./cmd/oneday --version` | Printed dev version/build metadata | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| AI-SETUP-01 | 24-01 | Explicit model roles without secrets in tracked YAML | SATISFIED | Config/defaults expose Codex primary, utility/repair, ASCII, and embedding fields; examples use env placeholders. |
| AI-SETUP-02 | 24-02 | Setup creates local provider config and stores API keys in env storage | SATISFIED | `runSetup` handles Codex/LiteLLM/OpenRouter and `.env` placeholders; isolated setup passed. |
| AI-SETUP-03 | 24-01, 24-02, 24-03 | Doctor reports provider/Codex/model/embedding/RAG readiness | SATISFIED | `runDoctor` performs tool, env, config, provider, and embedding checks; local doctor passed. |
| AI-SETUP-04 | 24-04 | Provider auth/capability failures are actionable | SATISFIED | Provider error translation names env vars, `codex login`, and `oneday doctor`; tests cover OpenAI-compatible and Codex paths. |
| RAG-SETUP-01 | 24-02, 24-03, 24-04 | Explicit embedding-capable provider, `text-embedding-3-small`, clear no-RAG warning | SATISFIED | Shared selector excludes Codex/Claude Code; TUI/setup/doctor produce clear enabled/disabled status. |
| DIST-01 | 24-04 | Shared/release artifacts exclude local config/env/data/binaries/secrets and document setup handoff | SATISFIED | `.gitignore`, README, `.env.example`, and `friend-safe-check` implement the handoff hygiene. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| README.md | 93 | Mentions `${ENV_VAR}` placeholders | Info | Documentation only; not a placeholder implementation or stub. |

### Human Verification Required

None. Provider smoke was exercised locally with the configured Codex login; remote LiteLLM/OpenRouter success still depends on real keys, but missing/auth failure behavior is covered by tests and code-level diagnostics.

### Gaps Summary

No blocking gaps found. Commits were intentionally skipped because this is a dirty operational repo; verification judged code, files, and tests rather than commit presence. `make release-check` was not run because its own gate requires a clean worktree, but the phase-specific `friend-safe-check` and full Go test suite passed.

---

_Verified: 2026-05-13T13:49:49Z_
_Verifier: Claude (gsd-verifier)_
