# Troubleshooting

Start with safe diagnostics:

```bash
oneday doctor
oneday doctor --json
oneday config show --safe
```

`oneday doctor --json` is the canonical readiness and recovery contract used by
the terminal and browser. Each probe has a stable `code`, `status`, `required`,
safe `summary`, and redacted `action`. Actions are one of `configure`,
`check_credentials`, `check_connection`, `retry_later`, `check_capability`,
`review_billing`, `create_backup`, `restore_empty_target`, or
`preserve_original`; clients can localize them without parsing provider errors.
The report intentionally omits credentials, provider responses, and local paths.

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

## Browser login, Host, Origin, or reverse-proxy requests are rejected

- Use the exact public origin configured for the proxy. The desktop client also
  requires an origin root, not a URL with a path, query, fragment, or embedded
  credentials.
- Confirm the proxy preserves the public `Host` header and that the host (with
  a non-default port if applicable) appears in `ONEDAY_GATEWAY_ALLOWED_HOSTS`.
- A one-shot bootstrap URL is consumed after use. Start a new local interactive
  gateway or provide a fresh configured bootstrap token; do not reuse a URL
  from logs or browser history.
- Do not send the direct bearer credential as a bootstrap token. It intentionally
  disables browser bootstrap when it is the only configured credential.
- Remote browser access requires HTTPS. Plain HTTP is only accepted for
  loopback development.

## Desktop cannot connect or standalone will not start

- In remote mode, use `https://` and the bare server origin. Verify the server
  itself is healthy and that its reverse proxy supports the normal web session.
- Remote mode stores no local stories. If a story seems absent, connect to the
  server/profile where it was created; desktop does not synchronize stores.
- In standalone mode, the desktop build needs matching bundled gateway and
  engine sidecars plus the bundled web UI. A missing component is a packaging
  failure, not a provider setting you can repair in the remote profile.
- A local provider may still fail even when standalone starts. Run the profile's
  normal setup/doctor flow, configure a narrative provider, and keep media
  disabled for text-only play until its provider is ready.
- Stop/restart the standalone local gateway from the Desktop settings window;
  if it repeatedly fails, retain only sanitized bounded diagnostics and avoid
  sharing any token-bearing startup URL.

## Backup or restore is incomplete

- Back up the complete configured data directory, including generated assets,
  not only `oneday.db`.
- Stop a standalone profile before copying it. For an active server, take a
  SQLite-safe backup rather than a raw copy during writes.
- The source checkout includes a checksummed SQLite-safe verification workflow:

  ```bash
  ./scripts/verify-sqlite-backup-restore.sh --db source.sqlite --backup oneday-backup.sqlite --restore-dir empty-recovery-target
  ```

  The restore directory must already exist and be empty. The workflow verifies
  the checksum, SQLite integrity, and foreign keys before it makes the restored
  database visible. It refuses a non-empty target and never writes to the source.
- If another process creates the checksum name during publication, OneDay never
  overwrites or deletes that path. The valid database backup may remain without
  its trusted checksum; treat it as incomplete, do not restore from it, and
  remove it manually only after confirming that it is your artifact.
- For an upgrade or migration failure, keep the original stopped profile or
  server data untouched. Restore the backup into a distinct empty recovery
  target, start the new version against that target, and only promote it after
  its normal migration and `oneday doctor` checks pass. If migration fails,
  discard the recovery target and investigate; do not retry against or overwrite
  the original.
- Restoring replaces only the chosen target profile. It does not merge remote
  and standalone stories. A browser connected to a server reports that server’s
  backup state; standalone backups are created from the local standalone profile.

## Images stay pending or fail

- Check `ai.image_generation.provider`, its URL, and provider credentials.
- For `imagegen-bridge`, check `/health/ready`, the bearer token, selected
  upstream provider capabilities, and any configured fallback routes.
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
