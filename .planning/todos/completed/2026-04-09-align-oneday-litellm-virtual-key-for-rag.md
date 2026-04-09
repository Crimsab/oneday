# Align OneDay LiteLLM Virtual Key For RAG

**Captured:** 2026-04-09
**Status:** Completed
**Completed:** 2026-04-09
**Type:** Ops / proxy configuration

## Resolution

The active OneDay LiteLLM virtual key was aligned with the models the app actually uses.

- embeddings: `text-embedding-3-small`
- narrator / fallback path: `grok-4.1-fast`, `main-fast`, `gemini-2.5-flash-lite`
- ambient ASCII path: `ascii-ambient`

## Verification

- embedding smoke test returned `200 OK`
- chat completion smoke test for `ascii-ambient` returned `200 OK`

## Delivered In

- `/opt/lab/docker/ai-proxy/config.yaml`
- `/opt/lab/docker/oneday/config.yaml`
- `/workspace/homelab/docs/ai-proxy.md`
- `.planning/phases/10-ambient-ascii-art-and-model-benchmarking/10-03-SUMMARY.md`
