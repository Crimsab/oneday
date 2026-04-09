# Phase 10 Verification: Ambient ASCII Art and Model Benchmarking

**Date:** 2026-04-09
**Status:** Passed

## Automated Verification

- `go test ./...`
- `go build -o ./oneday ./cmd/oneday`
- `go build -o ./oneday-benchmark ./cmd/oneday-benchmark`
- `go build -o ./oneday-ascii-benchmark ./cmd/oneday-ascii-benchmark`
- `GOOS=linux GOARCH=amd64 go build -o build/oneday-linux-amd64 ./cmd/oneday`
- `GOOS=windows GOARCH=amd64 go build -o build/oneday-windows-amd64.exe ./cmd/oneday`
- `make all`

## Provider Smoke Tests

- `POST http://lite.homelab.local/v1/chat/completions` with model `ascii-ambient` returned `200 OK`
- `POST http://lite.homelab.local/v1/embeddings` with model `text-embedding-3-small` returned `200 OK`

## Verified Outcomes

- Narrative turns can request ambient ASCII via structured `ascii_cue` without bloating the main narrator output.
- Ambient ASCII generation is same-turn, isolated, and failure-tolerant.
- Choice stat badges are inspectable from the keyboard inside the narrative view.
- Repo-root `./oneday` is refreshed by the local build flow, so manual testing from the project root runs the current code.
- Dedicated ASCII benchmark artifacts and review docs exist in `docs/benchmarks`.
- The OneDay LiteLLM configuration once again supports both ASCII generation and embeddings.

## Operational Notes

- `ascii-ambient` is configured as a deterministic primary alias with explicit fallbacks.
- When free-model deployments are temporarily unhealthy, LiteLLM can still fall back to `main-fast`; this keeps the feature usable even if the preferred ASCII model pool is unavailable.
