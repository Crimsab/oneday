# Contributing to OneDay

Thanks for helping improve OneDay. Small, focused pull requests are the easiest
to review and the safest for persistent story data.

## Before you start

- Search existing issues and pull requests.
- Open an issue first for large behavior, schema, protocol, or UI changes.
- Never include API keys, local configuration, databases, or private story content.
- Keep the Go engine and SQLite as the canonical game state; browser code must
  not create a second source of truth.

## Local setup

Follow [docs/getting-started.md](docs/getting-started.md) and
[docs/development.md](docs/development.md). The main verification commands are:

```bash
make verify
cargo fmt --manifest-path gateway/Cargo.toml -- --check
cargo test --manifest-path gateway/Cargo.toml
cd gateway/web && bun run test && bun run build
```

Run the relevant subset while developing and `make release-check` before a
release-sensitive pull request.

## Pull requests

- Use Conventional Commits (`feat:`, `fix:`, `perf:`, `docs:`, `test:`, `ci:`).
- Add or update tests for behavior changes.
- Update public docs when configuration, setup, contracts, or operator behavior changes.
- Explain data migration and backward-compatibility impact when storage changes.
- Keep generated files in sync when a contract generator reports drift.
- Include screenshots for visible browser changes, at desktop and mobile widths.

By contributing, you agree that your contribution is provided under the
project's repository license once that license is present.
