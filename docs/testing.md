# Testing

OneDay spans a Go engine and terminal client, a Rust gateway, a React frontend,
SQLite migrations, generated contracts, and a Docker image. A passing unit test
in one layer is not sufficient release evidence.

## Fast local verification

```bash
make verify
cargo test --manifest-path gateway/Cargo.toml
cd gateway/web
bun install --frozen-lockfile
bun run test
bun run build
```

Run the focused cross-system regression sweep with:

```bash
./scripts/qa-matrix.sh --automated-only
```

It rebuilds the CLI and exercises save/resume, rewind provenance, canonical
codex visibility, investigations, projects, social-duel aftermath, and TUI
handoffs.

## Browser checks

Install Chromium once, then run the desktop and mobile Playwright gates:

```bash
cd gateway/web
bunx playwright install --with-deps chromium
bun run test:e2e
```

Browser tests verify the production-facing interaction contract. They should
cover keyboard access, responsive layouts, canonical state rendering, and the
action composer without depending on a live model provider.

## Full CI coverage

The `CI` workflow runs:

- workflow syntax validation with Actionlint;
- a Gitleaks scan of tracked content;
- Go verification, migration compatibility, command smoke checks, and Linux and
  Windows cross-compilation;
- Rust formatting, tests, Clippy, and a debug build;
- frontend unit tests, production build, and Playwright desktop/mobile gates;
- a complete gateway Docker image build.

Private repository runs use the dedicated OneDay runner. Public pull requests
use GitHub-hosted runners and never execute contributor code on private lab
infrastructure.

## Release verification

`make release-check` is the local release gate. The Release Please workflow
repeats the relevant Go, Rust, web, browser, and packaging checks before it
uploads Linux and Windows archives to a GitHub Release.

Before a user-facing release, also verify manually:

- first boot against an empty database;
- create, play, save, restore, rewind, and branch a story;
- browser and terminal visibility of the same canonical turn;
- provider failure and recovery without partial state commits;
- backup and restore of the SQLite data directory;
- upgrade from the previous supported release.

Live provider smoke tests are intentionally separate because they require
credentials, can cost money, and may fail for reasons outside the repository.
