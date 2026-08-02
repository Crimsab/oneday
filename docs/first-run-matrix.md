# First-run and portability matrix

`make first-run-matrix` is the repeatable, credential-free proof for the
first-run and portability contracts. It creates a new temporary workspace for
each invocation, runs with an empty OneDay configuration/data location, and
makes Go and Cargo dependency resolution offline. Every provider-facing test
uses a fake in-process server, fixture, or mocked browser route; it never calls
a configured provider or requires a real credential.

The runner always creates a fresh `HOME` below its temporary workspace; it
never inherits the caller's OAuth, provider, or application configuration. It
automatically reuses the host's non-sensitive Go module cache, installed
Playwright browsers, Cargo registry, Cargo Git checkouts, and Rust toolchain.
An offline parent runner may override those cache/toolchain directories through
`ONEDAY_MATRIX_GOMODCACHE`,
`ONEDAY_MATRIX_CARGO_REGISTRY_DIR`, `ONEDAY_MATRIX_CARGO_GIT_DIR`, and
`ONEDAY_MATRIX_RUSTUP_HOME`, plus an existing
`ONEDAY_MATRIX_CARGO_TARGET_DIR` when compiled artifacts should be reused,
`ONEDAY_MATRIX_PLAYWRIGHT_BROWSERS_PATH`
for already-installed Playwright browsers, and
`ONEDAY_MATRIX_BUN_INSTALL_CACHE_DIR` for Bun's package cache. The runner
links or references those directories from an empty temporary tool home; it
does not assume or impose read-only mounts, and it does not load the caller's
application configuration or credentials. It copies web and desktop
sources from Git-tracked files only before the offline, lockfile-pinned Bun
install. Untracked `.env` files, `node_modules`, build output and test
artifacts therefore cannot enter the test workspace; the working tree never
receives `node_modules`, Vite outputs, or desktop test outputs. Playwright
runs use polling so the proof does not depend on a host file-watcher quota.

The desktop copy receives inert, version-matched sidecar and web-resource
fixtures before Cargo evaluates the Tauri bundle contract. They are created
only inside the temporary tracked-source copy and are never executable release
artifacts.

The matrix is intentionally bounded. Each command has a five-minute default
limit (`ONEDAY_MATRIX_TIMEOUT_SECONDS` may lower or raise it), and all temporary
state, default build caches, default Cargo targets, and Playwright output are
removed when the runner exits. Host dependency and browser caches are only
read/reused; an explicitly supplied external Cargo target directory is reused
and retained. Required toolchains and already-cached
dependencies must be installed before running it; the matrix does not download
them as a side effect.

## Evidence boundary

The gateway/web slice contains the mocked browser flow from an empty
installation through onboarding to a submitted action. The CLI slice joins
empty-profile setup, redacted doctor readiness, a deterministic fixture story,
and a fake-provider first action in one test before reopening the database to
verify persistence. The desktop slice remains a Tauri/UI contract suite and
is not a first-playable packaged-desktop proof; that criterion remains pending
until its end-to-end fixture exists and passes.

Run one proof slice with `./scripts/first-run-matrix.sh <slice>` or run the
complete matrix with `make first-run-matrix`.

| Contract | Repeatable proof |
| --- | --- |
| Empty CLI first run | One test joins empty-profile setup, doctor redaction/readiness, a deterministic fixture story, a fake-provider first action through the gateway handler, and database reopen/persistence. |
| Empty gateway/web installation | Rust authentication tests prove protected data, one-shot bootstrap, and direct bearer separation. Vitest and Playwright use empty-install and gateway route mocks to exercise installation readiness, distinct story onboarding, and one submitted playable action. |
| Desktop standalone | Existing Tauri config/standalone/lifecycle tests prove a fresh standalone profile has private, isolated config/data paths, version-matched sidecar planning, loopback readiness inputs, bounded diagnostics, and launch-secret redaction. |
| Desktop remote and transfer | Tauri tests prove remote profiles have no data path, validate the isolated server origin, reject path/type/oversize imports before dispatch, sanitize export names, and retain the configured origin. Gateway archive and world-template tests exercise the real archive/template contracts. |
| Provider matrix | Text-only rejection, compatible-local capability probes, local bridge transport, remote HTTPS bearer transport, direct adapters, authentication, capability failures, and retry-safe failure states run against local Axum/HTTP fakes. |
| Profile and recovery isolation | Profile tests keep standalone/remote state separate. The previous-release SQL fixture is copied before upgrade and the test proves the source DB bytes stay unchanged; the backup/restore fixture verifies source immutability through a failed recovery migration and refuses a non-empty target. |

The desktop slice now includes a real Chromium pass over the launcher with its
Tauri bridge mocked at the command boundary. It proves provider parity, explicit
update consent, responsive reflow, keyboard focus, hover states, target sizing,
and Axe accessibility without using credentials. It still does not build or
launch a signed AppImage, deb, Windows installer, macOS app, or DMG. Native
package behavior remains covered by the separate platform packaging workflow;
this matrix keeps that boundary explicit.

After matrix changes, also run the proportional repository gates:

```bash
shellcheck scripts/first-run-matrix.sh
bun scripts/check-docs.ts
make friend-safe-check
git diff --check
```

Run the relevant Go, gateway Rust, web, and desktop checks from
[Testing](testing.md) when the toolchains are available. The matrix does not
replace full release packaging, live-provider smoke tests, or a manual signed
desktop package check.
