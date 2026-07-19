# First-run and portability matrix

`make first-run-matrix` is the repeatable, credential-free proof for the
first-run and portability contracts. It creates a new temporary workspace for
each invocation, runs with an empty OneDay configuration/data location, and
makes Go and Cargo dependency resolution offline. Every provider-facing test
uses a fake in-process server, fixture, or mocked browser route; it never calls
a configured provider or requires a real credential.

The runner always creates a fresh `HOME` below its temporary workspace; it
never inherits the caller's OAuth, provider, or application configuration. An
offline parent runner may supply only non-sensitive pre-populated cache/toolchain
directories through `ONEDAY_MATRIX_GOMODCACHE`,
`ONEDAY_MATRIX_CARGO_REGISTRY_DIR`, `ONEDAY_MATRIX_CARGO_GIT_DIR`, and
`ONEDAY_MATRIX_RUSTUP_HOME`, plus `ONEDAY_MATRIX_PLAYWRIGHT_BROWSERS_PATH`
for already-installed Playwright browsers, and
`ONEDAY_MATRIX_BUN_INSTALL_CACHE_DIR` for Bun's package cache. The runner
links or references those directories from an empty temporary tool home; it
does not assume or impose read-only mounts, and it does not read the caller's
`HOME`, `CARGO_HOME`, or Rustup configuration. It copies web and desktop
sources from Git-tracked files only before the offline, lockfile-pinned Bun
install. Untracked `.env` files, `node_modules`, build output and test
artifacts therefore cannot enter the test workspace; the working tree never
receives `node_modules`, Vite outputs, or desktop test outputs. Playwright
runs use polling so the proof does not depend on a host file-watcher quota.

The matrix is intentionally bounded. Each command has a five-minute default
limit (`ONEDAY_MATRIX_TIMEOUT_SECONDS` may lower or raise it), and all temporary
state, build caches, Cargo targets, and Playwright output are removed when the
runner exits. Required toolchains and already-cached
dependencies must be installed before running it; the matrix does not download
them as a side effect.

## Evidence boundary

The gateway/web slice contains the mocked browser flow from an empty
installation through onboarding to a submitted action. The CLI slice currently
executes setup/doctor and the fake-provider turn service as separate focused
tests; it is not yet a single CLI-process proof from setup to first playable
story. Likewise, the desktop slice is a Tauri/UI contract suite and is not a
first-playable packaged-desktop proof. These two criteria must remain pending
until their end-to-end fixtures exist and pass.

Run one proof slice with `./scripts/first-run-matrix.sh <slice>` or run the
complete matrix with `make first-run-matrix`.

| Contract | Repeatable proof |
| --- | --- |
| Empty CLI first run | Setup and doctor handlers use temporary configuration/data paths; readiness reports redact private paths and fake-provider failures. The in-process turn service uses its existing fake narrator to commit a first action, then creates and restores a save. |
| Empty gateway/web installation | Rust authentication tests prove protected data, one-shot bootstrap, and direct bearer separation. Vitest and Playwright use empty-install and gateway route mocks to exercise installation readiness, distinct story onboarding, and one submitted playable action. |
| Desktop standalone | Existing Tauri config/standalone/lifecycle tests prove a fresh standalone profile has private, isolated config/data paths, version-matched sidecar planning, loopback readiness inputs, bounded diagnostics, and launch-secret redaction. |
| Desktop remote and transfer | Tauri tests prove remote profiles have no data path, validate the isolated server origin, reject path/type/oversize imports before dispatch, sanitize export names, and retain the configured origin. Gateway archive and world-template tests exercise the real archive/template contracts. |
| Provider matrix | Text-only rejection, compatible-local capability probes, local bridge transport, remote HTTPS bearer transport, direct adapters, authentication, capability failures, and retry-safe failure states run against local Axum/HTTP fakes. |
| Profile and recovery isolation | Profile tests keep standalone/remote state separate. The previous-release SQL fixture is copied before upgrade and the test proves the source DB bytes stay unchanged; the backup/restore fixture verifies source immutability through a failed recovery migration and refuses a non-empty target. |

The desktop package itself is deliberately not claimed as an end-to-end proof by
this matrix. It does not build or launch a signed AppImage, deb, or Windows
installer. Package behavior remains covered by the separate platform packaging
workflow; this matrix runs the existing Tauri and desktop UI tests and makes
that boundary explicit.

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
