# Phase 27: Product Hardening and Friend Packaging - Context

**Gathered:** 2026-05-13
**Status:** Ready for execution

<domain>
## Phase Boundary

Finish the approved follow-up feature set: CI command smoke, doctor JSON, friend export, config migration, RAG reindex, richer benchmark guidance, story-pack schema/selection, and non-live tests.
</domain>

<decisions>
## Implementation Decisions

### the agent's Discretion
- Keep changes incremental and commit each feature separately.
- Do not require live remote providers or Ollama in tests.
- Keep export friend-safe by default.
</decisions>
<code_context>
## Existing Code Insights

`cmd/oneday/main.go` contains simple command dispatch. `internal/config` owns config defaults/validation. `internal/rag/vectorstore.go` can prune stale chunks. Story pack discovery exists in `plugins/examples`.
</code_context>
