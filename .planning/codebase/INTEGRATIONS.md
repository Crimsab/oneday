# External Integrations

**Analysis Date:** 2026-04-09

## APIs & External Services

**AI Completion / Embeddings:**
- LiteLLM or another OpenAI-compatible proxy - primary gameplay provider and embedding endpoint
  - SDK/Client: `internal/ai/providers/openai_compat.go`
  - Auth: `ai.litellm.api_key` in `config.yaml`
  - Config files: `config.example.yaml`, `internal/config/config.go`
- OpenRouter - optional OpenAI-compatible fallback provider
  - SDK/Client: `internal/ai/providers/openai_compat.go`
  - Auth: `ai.openrouter.api_key` in `config.yaml`
- Claude Code CLI - optional local fallback provider invoked as a subprocess
  - SDK/Client: `internal/ai/providers/claudecode.go`
  - Auth: external Claude CLI session; no repo-managed secret field beyond `ai.claude_code.binary`

**Provider-side structured output helpers:**
- OpenRouter-compatible `response-healing` plugin is auto-added for structured responses in `internal/ai/providers/openai_compat.go`

## Data Storage

**Databases:**
- SQLite file database at `<data_dir>/oneday.db`
  - Connection: `Config.DataDir` from `internal/config/config.go`
  - Client: `internal/storage/db.go` via `modernc.org/sqlite`

**File Storage:**
- Local filesystem only
- Session logs: `oneday_data/stories/<story-id>/sessions/*.jsonl` via `internal/engine/session.go`
- Save snapshots: `oneday_data/stories/<story-id>/saves/*.json` via `internal/engine/save.go`
- Benchmark reports: `docs/benchmarks/runs/` via `cmd/oneday-benchmark/main.go`, `cmd/oneday-ascii-benchmark/main.go`, and `cmd/oneday-schema-benchmark/main.go`

**Caching:**
- In-memory model capability cache inside `internal/ai/providers/openai_compat.go`
- No external cache service detected

## Authentication & Identity

**Auth Provider:**
- Not applicable for end users
  - Implementation: single-user local CLI; provider auth is outbound API auth only

## Monitoring & Observability

**Error Tracking:**
- None detected

**Logs:**
- Startup failures print to stderr in `cmd/oneday/main.go`
- RAG errors log through the standard library logger in `internal/rag/rag.go`

## CI/CD & Deployment

**Hosting:**
- GitHub Releases for packaged binaries (`.github/workflows/build-release.yml`, `.github/workflows/release-please.yml`)

**CI Pipeline:**
- GitHub Actions runs `go test ./...`, `go vet ./...`, and cross-builds binaries
- Release automation uses `googleapis/release-please-action`

## Environment Configuration

**Required env vars:**
- Main app: none required by code path in `cmd/oneday/main.go`; runtime secrets come from `config.yaml`
- Benchmark tools:
  - `ONEDAY_BENCH_BASE_URL`
  - `ONEDAY_BENCH_API_KEY`
  - `ONEDAY_BENCH_MODELS`
  - `ONEDAY_BENCH_OUTPUT_DIR`
  - `ONEDAY_ASCII_BENCH_BASE_URL`
  - `ONEDAY_ASCII_BENCH_API_KEY`
  - `ONEDAY_ASCII_BENCH_MODELS`
  - `ONEDAY_ASCII_BENCH_OUTPUT_DIR`
  - `OPENAI_BASE_URL`
  - `OPENAI_API_KEY`
  - `OPENROUTER_API_KEY`

**Secrets location:**
- `config.yaml` for gameplay provider endpoints and API keys (`.gitignore` marks it as secret-bearing)
- CLI flags or env vars for benchmark tools in `cmd/oneday-benchmark/main.go` and `cmd/oneday-ascii-benchmark/main.go`

## Webhooks & Callbacks

**Incoming:**
- None detected

**Outgoing:**
- POST `.../chat/completions` and `.../embeddings` in `internal/ai/providers/openai_compat.go`
- `claude -p ... --output-format json` subprocess execution in `internal/ai/providers/claudecode.go`

---

*Integration audit: 2026-04-09*
