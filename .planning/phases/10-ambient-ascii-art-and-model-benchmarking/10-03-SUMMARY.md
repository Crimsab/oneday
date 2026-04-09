# Plan 10-03 Summary: ASCII Benchmarking + Provider Alignment

**Completed:** 2026-04-09
**Status:** Done

## Delivered

- Added a dedicated `oneday-ascii-benchmark` command for benchmarking ambient ASCII generation against OneDay-style prompts.
- Benchmark artifacts are saved under `docs/benchmarks/runs` as JSON and Markdown, with multiple rankings for runtime decision-making.
- Added a human-readable benchmark review summarizing winners, tradeoffs, and runtime recommendations.
- Local OneDay config was aligned to use a dedicated `ascii-ambient` model alias and the RAG embedding model.
- The LiteLLM virtual key used by OneDay was updated so embeddings work again.
- The AI proxy config and homelab docs were aligned with the new ASCII model aliases and benchmarked model set.

## Key Files

- `cmd/oneday-ascii-benchmark/main.go`
- `docs/benchmarks/oneday-ascii-benchmark.md`
- `docs/benchmarks/runs/2026-04-09-024455-oneday-ascii-benchmark-json_schema.md`
- `docs/benchmarks/2026-04-09-oneday-ascii-benchmark-review.md`
- `/opt/lab/docker/ai-proxy/config.yaml` (external ops config)
- `/workspace/homelab/docs/ai-proxy.md` (external lab docs)

## Notes

- Benchmark winner for ambient ASCII quality/runtime balance was `openai/gpt-oss-120b:free`, with `google/gemma-4-26b-a4b-it:free` as the best free fallback and `x-ai/grok-4.1-fast` as the best paid speed/quality option.
- In live proxy usage, `ascii-ambient` can still fall back to `main-fast` when free-model deployments are temporarily unhealthy; this is intentional for runtime resilience.
