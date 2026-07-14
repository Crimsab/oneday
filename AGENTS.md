# OneDay Agent Guide

This file defines repository-wide instructions for coding agents and automated
contributors. A more specific `AGENTS.md` in a subdirectory may add or narrow
rules for that subtree.

`CLAUDE.md` must remain a symbolic link to this file so every supported agent
uses the same guidance.

## Read Before Editing

Start with the documents that match the change:

- [README.md](README.md) for the product and supported workflows.
- [Architecture](docs/architecture.md) for system boundaries and invariants.
- [Contributing](CONTRIBUTING.md) for the contributor workflow.
- [Development](docs/development.md) for toolchains and commands.
- [Testing](docs/testing.md) for the validation matrix.
- The relevant document under `docs/` for user-visible or operational changes.

Prefer the repository's existing patterns over introducing a new framework,
dependency, abstraction, or configuration mechanism.

## Architecture Invariants

- The Go engine and SQLite database own canonical game state.
- The Rust/Axum gateway is an HTTP, SSE, media, and process adapter. Do not copy
  domain rules or persistence into it.
- The React client renders canonical server state and submits intents. Do not
  create an independent game state or persistence layer in the browser.
- Turns must remain atomic and idempotent. Validation, provider, or storage
  failures must never leave a partially committed turn.
- Generated media is asynchronous and non-blocking. Text and canonical state
  remain usable when image generation is unavailable or fails.
- Public defaults and examples must be portable. Do not encode assumptions
  about a contributor's LAN, hostnames, filesystem, proxy, or deployment.

If a requested change conflicts with one of these invariants, stop and explain
the conflict before implementing it.

## Repository Map

| Path | Responsibility |
| --- | --- |
| `cmd/oneday/` | Main CLI/TUI and browser bridge commands |
| `internal/` | Domain engine, providers, storage, RAG, contracts, and terminal UI |
| `gateway/src/` | Rust gateway, bridge integration, HTTP, SSE, and media adapters |
| `gateway/web/src/` | React/TypeScript browser client |
| `contracts/` | Checked-in cross-language JSON schemas |
| `plugins/` | Story pack and minigame extension examples |
| `scripts/` | QA, release, and repository hygiene gates |
| `docs/` | User, operator, architecture, and benchmark documentation |

Keep changes in the layer that owns the behavior. Cross-layer changes should
update their contracts and tests in the same commit.

## Toolchain and Commands

Use the tool versions documented in [Development](docs/development.md):

- Go 1.25 or newer.
- Rust 1.97 with `rustfmt` and `clippy`.
- Bun 1.3.14 or newer for frontend dependencies and scripts.
- Docker Compose v2 where container validation is required.

Use Bun, not npm, pnpm, or Yarn:

```bash
cd gateway/web
bun install --frozen-lockfile
```

Do not replace or regenerate lockfiles unless the dependency graph intentionally
changes.

## Implementation Rules

- Make the smallest coherent change that solves the problem.
- Keep modules focused and side effects behind existing boundaries.
- Reuse established types and helpers when that improves consistency; avoid
  speculative abstractions.
- Preserve backward compatibility for saved games, configuration, APIs, plugin
  formats, and checked-in contracts unless a breaking change is explicit.
- Keep errors actionable and preserve cancellation, timeouts, and error causes.
- Use deterministic fixtures, fake providers, and temporary databases in tests.
  Do not make paid or live provider calls unless the task explicitly requires
  an authorized integration test.
- Add or update documentation when behavior, configuration, setup, or a public
  contract changes.

### Storage and migrations

- SQLite migrations in `internal/storage/migrations.go` are append-only.
- Never reorder, renumber, or edit a migration that may already have shipped.
- Add upgrade coverage for schema changes and preserve existing saved games.
- Keep a turn's state changes in one transaction.

### Gateway contracts

- Change the Go protocol source before generated or downstream representations.
- Regenerate the checked-in schema after protocol changes:

  ```bash
  go run ./cmd/oneday-gateway-schema > contracts/gateway-v1.schema.json
  ```

- Run both Go and Rust contract tests when a bridge type or schema changes.
- Do not hand-edit generated contract output.

### Frontend

- Keep TypeScript types aligned with the gateway contract.
- Preserve keyboard access, visible focus, semantic controls, reduced-motion
  behavior, and responsive layouts.
- Treat model and story content as untrusted. Do not inject it as raw HTML.
- Test behavior rather than implementation details.

## Security and Repository Hygiene

Never commit:

- `.env` files, local configuration, credentials, tokens, or private keys.
- Story databases, save data, logs, or user-generated media.
- Raw benchmark runs, coverage reports, build output, or dependency directories.
- Private planning files, machine-specific deployment overrides, internal host
  names, private addresses, or absolute operator paths.

Examples must use placeholders or documentation-safe values. Preserve existing
path validation, idempotency, input validation, and secret-handling safeguards;
do not weaken them merely because a deployment is local.

Run `make friend-safe-check` when a change touches configuration, packaging,
documentation, fixtures, or repository hygiene.

## Validation

Run focused tests while iterating, then the smallest complete set that covers
every changed layer.

### Go

```bash
go test ./...
go vet ./...
```

### Rust gateway

```bash
cargo fmt --manifest-path gateway/Cargo.toml -- --check
cargo test --manifest-path gateway/Cargo.toml
cargo clippy --manifest-path gateway/Cargo.toml --all-targets -- \
  -A clippy::too_many_arguments -D warnings
```

### Web client

```bash
cd gateway/web
bun run test
bun run build
```

For user-interface behavior, also run:

```bash
cd gateway/web
bunx playwright install --with-deps chromium
bun run test:e2e
```

### Cross-layer and release-sensitive changes

```bash
make verify
make release-check
```

For Compose changes, also run `docker compose config --quiet`. Do not claim a
check passed if it was skipped or its required service was unavailable; report
that limitation explicitly.

## Git and Releases

- Preserve unrelated user changes and keep commits focused.
- Use Conventional Commits with an appropriate scope when useful.
- Include tests, migrations, generated contracts, and documentation required by
  the behavior change in the same logical commit.
- Do not hand-edit automated release pull requests or generated changelog
  entries. Release Please owns release versioning and changelog updates.
- Do not commit generated build artifacts.

## Definition of Done

A change is complete only when:

- The implementation respects the architecture invariants above.
- Relevant tests and static checks pass.
- Migrations and generated contracts are synchronized where applicable.
- User-visible behavior and configuration are documented.
- No secrets, private infrastructure details, or generated local data are added.
- The final diff contains only intentional files and the validation performed is
  reported accurately.
