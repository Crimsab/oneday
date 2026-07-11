---
phase: 27-product-hardening-and-friend-packaging
verified: 2026-05-13T16:35:00Z
status: passed
score: 8/8 must-haves verified
---

# Phase 27 Verification

All requested hardening items are implemented.

- HARDEN-01: CI runs non-live command smoke checks.
- HARDEN-02: `doctor --json` emits safe diagnostics.
- HARDEN-03: `export` creates a clean safe-by-default handoff directory.
- HARDEN-04: config version and migration helper added.
- HARDEN-05: `rag reindex` clears stale embeddings.
- HARDEN-06: `rag benchmark` reports next steps.
- HARDEN-07: story packs validate and can be selected during setup.
- HARDEN-08: non-live tests cover new helpers.

Checks:

- `go test ./...`
- `go run ./cmd/oneday doctor --json`
- `go run ./cmd/oneday config show --safe`
- `go run ./cmd/oneday rag benchmark`
- `go run ./cmd/oneday rag reindex`
- `go run ./cmd/oneday story-packs list`
- temporary `export`
