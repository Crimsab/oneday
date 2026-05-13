---
phase: 25-local-rag-embeddings-and-reconfigurable-setup
plan: 25-03
subsystem: diagnostics
tags: [doctor, local-rag, embeddings]
requirements-completed: [LOCAL-RAG-02, LOCAL-RAG-06]
key-files:
  modified:
    - cmd/oneday/main.go
    - internal/tui/app.go
completed: 2026-05-13
---

# Plan 25.3 Summary

**Doctor and runtime RAG report local embedding backend, URL, model, dimensions, and smoke-test failures**

`oneday doctor` now instantiates the selected local backend and smoke-tests embeddings through the same provider path used by runtime RAG. Runtime RAG logs the selected local/remote provider and uses configured local dimensions.

Verification: `go run ./cmd/oneday doctor`, `go test ./...`.
