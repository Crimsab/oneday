# Phase 24: AI Provider Onboarding, Codex OAuth, and RAG Setup Hardening - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning
**Source:** Conversation + current implementation state

<domain>
## Phase Boundary

This phase turns the current experimental Codex/LiteLLM/OpenRouter wiring into a complete handoff-ready setup experience. It covers provider configuration, first-time setup, diagnostics, RAG embedding viability, friend-safe sharing, and clear error messages. It does not redesign the narrative engine or gameplay systems.

</domain>

<decisions>
## Implementation Decisions

### Codex OAuth Defaults
- Keep Codex OAuth as an experimental but first-class setup choice.
- Use Codex `gpt-5.5` as the default primary narrator model.
- Use Codex `gpt-5.4-mini` as the default ancillary/utility model for repair, validation, and small side calls when those calls can use Codex.
- Preserve a configurable reasoning setting with accepted values `off`, `none`, `minimal`, `low`, `medium`, `high`, and `xhigh`; map `off` to Codex CLI `none`.

### RAG And Embeddings
- Codex CLI must not be treated as embedding-capable.
- RAG must require an explicit embedding-capable provider.
- Default remote embedding path should continue using the existing `text-embedding-3-small` model unless research proves the current proxy/OpenRouter stack needs a different working alias.
- Setup must either configure a working remote embedding provider or disable RAG with a clear warning.
- A local embedding fallback is desirable but should be planned as optional, with explicit tradeoffs for binary size, dependency complexity, and cross-platform setup.

### Setup And Diagnostics
- `oneday setup` should create `config.yaml` only when missing, write secrets to `.env` rather than YAML, and not overwrite existing exported env vars.
- `oneday doctor` should be safe to run repeatedly and should never mutate config.
- Diagnostics should check OS/arch, Go availability, Codex CLI presence, Codex login status, provider env vars, provider smoke response, embedding smoke response, RAG enabled/disabled consistency, and local ignored-file hygiene.
- LiteLLM/OpenRouter 401 errors should be translated into actionable messages naming `ONEDAY_LITELLM_API_KEY` or `ONEDAY_OPENROUTER_API_KEY` when relevant.

### Friend-Safe Sharing
- `config.yaml`, `.env`, `oneday_data/`, binaries, release archives, and local databases must remain ignored.
- `config.example.yaml`, `.env.example`, and README setup steps are the supported handoff path.
- The current local `config.yaml` can remain Codex-first for the developer machine because it is ignored.

### the agent's Discretion
- Exact internal type names and file layout for ancillary model settings.
- Whether to implement local embeddings in this phase or leave the local fallback as documented future work after remote embedding diagnostics are robust.
- The exact text layout of setup/doctor output, provided it is grep/test-verifiable.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Rules
- `AGENTS.md` — HomeLab conventions, bun/npm rules, no git in operational dirs unless requested, and Codex/frontend/lab constraints.

### Current AI Provider Implementation
- `internal/config/config.go` — Config schema, defaults, validation, provider priority, env expansion.
- `internal/config/dotenv.go` — `.env` loader semantics.
- `internal/aifactory/factory.go` — Provider construction from config.
- `internal/ai/provider.go` — Provider, stream provider, embedding interfaces.
- `internal/ai/providers/openai_compat.go` — LiteLLM/OpenRouter chat, stream, model capability, and embedding behavior.
- `internal/ai/providers/codex.go` — Experimental Codex CLI provider.
- `internal/ai/providers/claudecode.go` — Existing CLI-provider pattern.
- `internal/tui/app.go` — RAG embedding-provider selection and narrative boot wiring.

### Setup And Distribution
- `cmd/oneday/main.go` — CLI entrypoint, version/setup flow, config and dotenv path resolution.
- `config.example.yaml` — Shared safe template.
- `.env.example` — Shared safe environment template.
- `.gitignore` — Friend-safe exclusion policy.
- `README.md` — User-facing setup instructions.
- `Makefile` — Build, verify, and release helper surface.

### Prior Related Phases
- `.planning/phases/10-ambient-ascii-art-and-model-benchmarking/10-03-SUMMARY.md` — Prior provider/proxy alignment work.
- `.planning/phases/12-state-rollback-integrity-and-narrative-persistence/12-03-SUMMARY.md` — Prior provider-decoupled embedding selection work.
- `.planning/phases/16-streaming-memory-reliability-and-provider-capability-hardening/16-03-SUMMARY.md` — Prior deterministic embedding fallback and diagnostics work.

</canonical_refs>

<specifics>
## Specific Ideas

- Add an `ai.utility` or provider-specific utility block so repair/ASCII/model-smoke calls can default to `gpt-5.4-mini` under Codex without overloading the narrator model.
- Keep `ai.generation.repair_model` backward compatible while introducing a clearer provider-aware model reference format if needed.
- Doctor should include a remote embedding probe equivalent to `/v1/embeddings` with the configured embedding model and a tiny input.
- Doctor should report: `RAG: enabled, embedding provider: openrouter/litellm, model: text-embedding-3-small` or `RAG: disabled, reason: no embedding-capable provider configured`.
- If local embeddings are included, prefer an isolated adapter that does not affect the remote provider path and can be disabled on unsupported platforms.

</specifics>

<deferred>
## Deferred Ideas

- Full local model download/management UI.
- Replacing SQLite BLOB cosine retrieval with a separate vector database.
- Multi-account Codex auth switching.
- Changing gameplay prompt architecture beyond provider/model selection.

</deferred>

---

*Phase: 24-ai-provider-onboarding-codex-oauth-and-rag-setup-hardening*
*Context gathered: 2026-05-13 via conversation*
