# Verification 16

## Automated Checks

- `go test ./internal/ai/providers`
- `go test ./internal/rag`
- `go test ./internal/config ./internal/tui`
- `go test ./...`
- `go vet ./...`

## Verified Outcomes

- OpenAI-compatible sync and streaming requests now share the same structured-output guards and fallback behavior.
- Structured streaming retries without `response_format` when needed, and can degrade to complete-as-stream when upstream streaming is unsupported.
- RAG summary windows now operate on committed turn indices, including the first committed turn.
- Embedding selection now honors explicit config or deterministic `auto` mode instead of loosely following the first enabled chat provider.
- New-story start, normal resume, and load-save resume now share one narrative initialization path in the TUI layer.

## Residual Risks

- Embedding auto mode is capability-aware by declaration/config rather than live probing of `/embeddings`, so endpoint-specific incompatibilities may still need stronger runtime diagnostics in a future pass.
