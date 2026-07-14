# Troubleshooting

Start with safe diagnostics:

```bash
oneday doctor
oneday doctor --json
oneday config show --safe
```

When running from source, replace `oneday` with `go run ./cmd/oneday`.

## No narrative provider works

- Confirm at least one provider is enabled and appears in `ai.provider_priority`.
- For Codex, run `codex login status`; for Claude Code, verify the CLI is on `PATH`.
- For LiteLLM/OpenRouter, verify the matching environment key is set and the model exists.
- A `401`/`403` normally means the key or provider account is invalid; a timeout normally means the base URL is unreachable.
- In Docker, use `host.docker.internal` for services running on the host.

## RAG is unavailable or dimensions mismatch

Run `oneday rag benchmark`. Confirm the embedding model returns exactly
`rag.dimensions` values. After changing the model or dimensions:

```bash
oneday rag reindex --all
```

For Ollama, confirm the service is running and the configured model is pulled.

## Browser container does not become healthy

```bash
docker compose ps
docker compose logs --tail 100 oneday-gateway
docker compose config --quiet
```

Ensure `config.yaml` is a file (not a directory accidentally created by Docker),
port 8788 is free, and the data volume is writable. Fresh databases are created
and migrated automatically.

## Images stay pending or fail

- Check `ai.image_generation.provider`, its URL, and provider credentials.
- `openclaw-bridge` does not require an API key but its bridge must be reachable.
- General art and transparent map icons may intentionally use different models.
- Image generation is asynchronous; narrative turns remain usable when it fails.
- Inspect bounded gateway logs for the request ID and provider error.

## Frontend contract test says `go` is missing

`gateway/web/src/commands.contract.test.ts` executes the Go command descriptor
generator. Install Go and keep it on `PATH`; running only Bun is insufficient for
the full frontend test suite.

## Release or CI checks fail

Run `make release-check` from a clean worktree. GitHub workflow syntax can be
checked with actionlint, and the repository CI also scans the checked-out source
with Gitleaks. See [Development](development.md) for individual gates.

If the problem is reproducible and not configuration-specific, open a GitHub
issue with sanitized `doctor --json` output, the OneDay version, OS/architecture,
and the smallest reproduction. Never attach `.env`, `config.yaml`, databases, or
provider responses containing private story content.
