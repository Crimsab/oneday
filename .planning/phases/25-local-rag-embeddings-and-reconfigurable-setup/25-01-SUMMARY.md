---
phase: 25-local-rag-embeddings-and-reconfigurable-setup
plan: 25-01
subsystem: rag
tags: [local-rag, embeddings, ollama, custom-http]
requirements-completed: [LOCAL-RAG-01, LOCAL-RAG-02]
key-files:
  created:
    - internal/ai/providers/ollama.go
    - internal/ai/providers/local_http.go
  modified:
    - internal/config/config.go
    - internal/aifactory/embedding.go
    - internal/tui/app.go
completed: 2026-05-13
---

# Plan 25.1 Summary

**Local RAG embeddings now support Ollama and custom local HTTP backends without API keys**

Implemented `ai.embedding.provider: local` with backend type, base URL, model, and dimensions. Runtime RAG now builds Ollama/custom/OpenAI-compatible providers from the shared embedding selector.

Verification: `go test ./cmd/oneday ./internal/config ./internal/aifactory ./internal/ai/providers ./internal/tui ./internal/rag`.
