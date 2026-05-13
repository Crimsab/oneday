# Phase 25: Local RAG Embeddings and Reconfigurable Setup - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase turns local RAG embeddings from a disabled config stub into a working setup/runtime path. It must support both a simple Ollama path and a generic custom local endpoint path, while keeping remote LiteLLM/OpenRouter and no-RAG paths intact.

</domain>

<decisions>
## Implementation Decisions

### Local Backend Strategy
- Do not force Ollama. Treat Ollama as the recommended easy backend, and add a generic custom local endpoint path for users who already run Python, llama.cpp, ONNX, or other embedding services.
- The config should represent local embeddings explicitly with backend type, base URL, model, and dimensions.
- Local embedding paths must not require API keys.
- Runtime RAG and doctor must share the same local provider selection logic.

### Setup UX
- Existing `config.yaml` is preserved by default.
- `oneday setup --force` and `oneday setup --reconfigure` reopen the setup wizard and can rewrite local config.
- The setup must explain model trade-offs before asking the user to choose.
- Setup should ask before running external install/download commands; pull/smoke the selected model when the user agrees.

### Model Defaults
- Recommended default: `bge-m3` for multilingual/Italian-friendly local RAG, dimensions 1024.
- Fast/lightweight: `nomic-embed-text`, dimensions 768.
- English quality: `mxbai-embed-large`, dimensions 1024.
- Quality/heavier: `qwen3-embedding`, dimensions should be configurable/defaulted conservatively and smoke-tested.

### the agent's Discretion
Keep implementation scoped and testable. Prefer small helper functions for setup choice mapping and local smoke tests.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/oneday/main.go` already contains setup, doctor, env loading, provider smoke, and embedding smoke flows.
- `internal/aifactory/embedding.go` centralizes embedding provider selection.
- `internal/rag/embeddings.go` validates configured embedding dimensions.
- `internal/ai/providers/openai_compat.go` covers OpenAI-compatible remote embeddings.

### Established Patterns
- Configuration lives in `internal/config/config.go` and `config.example.yaml`.
- Setup writes `config.yaml` locally and never tracks secrets.
- Tests already cover config validation, setup choice mapping, provider errors, and embedding provider selection.

### Integration Points
- Runtime RAG is constructed in `internal/tui/app.go`.
- Doctor and TUI should both use `aifactory.SelectEmbeddingProvider`.
- Local embedding clients should implement the same `Embed(ctx, ai.EmbeddingRequest)` contract used by RAG.

</code_context>

<specifics>
## Specific Ideas

User wants the full flow to work end-to-end: setup, local embedding model selection, download/setup where applicable, config write, doctor diagnostics, runtime RAG, and tests.

</specifics>

<deferred>
## Deferred Ideas

Bundling an embedded ONNX model directly inside OneDay is deferred. This phase supports local services cleanly; a future phase can add fully bundled offline inference if desired.

</deferred>
