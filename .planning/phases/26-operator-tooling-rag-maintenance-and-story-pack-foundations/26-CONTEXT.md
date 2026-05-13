# Phase 26: Operator Tooling, RAG Maintenance, and Story Pack Foundations - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning/execution

<domain>
## Phase Boundary

Add operational commands and maintenance paths that make the new AI/RAG setup easier to inspect, benchmark, repair, and share.
</domain>

<decisions>
## Implementation Decisions

### the agent's Discretion
- Keep commands simple and terminal-native.
- Avoid real provider calls in tests; use deterministic local/fake endpoints.
- Do not commit unrelated dirty narrative/engine work.
</decisions>

<code_context>
## Existing Code Insights

Setup and doctor live in `cmd/oneday/main.go`. RAG vector storage lives in `internal/rag/vectorstore.go`. Config models live in `internal/config/config.go`. Local embedding providers live in `internal/ai/providers`.
</code_context>

<specifics>
## Specific Ideas

Implement user-requested features: safe config show, local RAG benchmark, automatic dimension migration, deeper doctor, better setup output, story pack discovery, and non-live E2E tests.
</specifics>

<deferred>
## Deferred Ideas

Full story pack runtime injection into story generation can follow after discovery/listing stabilizes.
</deferred>
