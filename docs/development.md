# Development

## Toolchains

- Go 1.25.12+ (the `toolchain` directive pins the minimum secure patch)
- Rust 1.97 with `rustfmt` and `clippy`
- Bun 1.3.14+
- Docker with Compose v2 for the production image and browser E2E environment

Install frontend dependencies with Bun:

```bash
cd gateway/web
bun install --frozen-lockfile
```

## Repository layout

| Path | Responsibility |
| --- | --- |
| `cmd/oneday/` | Main CLI/TUI and browser bridge commands |
| `internal/` | Domain engine, AI providers, storage, RAG, contracts, and terminal UI |
| `gateway/` | Rust/Axum HTTP gateway and generated bridge types |
| `gateway/web/` | React/TypeScript browser client |
| `contracts/` | Cross-language JSON schemas |
| `plugins/` | Story pack and minigame extension examples |
| `scripts/` | QA, release, and hygiene gates |
| `docs/` | User, operator, architecture, and benchmark documentation |

## Fast checks

```bash
go test ./...
go vet ./...
cargo fmt --manifest-path gateway/Cargo.toml -- --check
cargo test --manifest-path gateway/Cargo.toml
cargo clippy --manifest-path gateway/Cargo.toml --all-targets -- -A clippy::too_many_arguments -D warnings
cd gateway/web && bun run test && bun run build
```

The complete local gates are:

```bash
make verify
make release-check
```

Browser E2E requires Chromium:

```bash
cd gateway/web
bunx playwright install --with-deps chromium
bun run test:e2e
```

See [Testing](testing.md) for the complete CI matrix, focused regression sweep,
and manual release checks.

## Generated contracts

If Go gateway protocol types change, regenerate and verify the checked-in schema
before committing. The contract test reports the exact generator command when
the file is stale.

## Commit and release style

Use Conventional Commits. `feat:` creates a minor release; `fix:` and `perf:`
create a patch release; a breaking-change footer creates a major release.
Scopes such as `web`, `gateway`, `engine`, `imagegen`, or `ci` keep the automated
changelog readable.

Do not commit local configuration, `.env`, story databases, generated benchmark
runs, build output, or private planning and deployment metadata. Repository-wide
agent guidance belongs in the tracked `AGENTS.md`; `CLAUDE.md` remains its
symbolic link. Run `make friend-safe-check` before handing source to someone
else.

See [CONTRIBUTING.md](../CONTRIBUTING.md) for the pull request checklist and
[Releases](releases.md) for automation details.
