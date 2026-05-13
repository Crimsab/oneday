---
phase: 25-local-rag-embeddings-and-reconfigurable-setup
plan: 25-02
subsystem: cli
tags: [setup, reconfigure, local-rag, ollama]
requirements-completed: [LOCAL-RAG-03, LOCAL-RAG-04, LOCAL-RAG-05, SETUP-RECONF-01, SETUP-RECONF-02, SETUP-RECONF-03]
key-files:
  modified:
    - cmd/oneday/main.go
    - cmd/oneday/main_test.go
    - config.example.yaml
completed: 2026-05-13
---

# Plan 25.2 Summary

**Setup can now be re-run intentionally and configure no-RAG, remote RAG, Ollama local RAG, or custom local RAG**

Added `setup --force` / `setup --reconfigure`, model trade-off prompts, Ollama model selection/pull/smoke, and custom endpoint URL/model/dimension setup. Existing configs are preserved by default with a clear reconfigure hint.

Verification: `go run ./cmd/oneday setup`, temporary-directory `setup --reconfigure` smoke, and targeted tests.
