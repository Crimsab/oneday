# First-run and portability matrix

`make first-run-matrix` is the repeatable, credential-free proof for the
first-run and portability contracts. It creates a new temporary workspace for
each invocation, runs with an empty OneDay configuration/data location, and
makes Go and Cargo dependency resolution offline. Every provider-facing test
uses a fake in-process server, fixture, or mocked browser route; it never calls
a configured provider or requires a real credential.

The matrix is intentionally bounded. Each command has a five-minute default
limit (`ONEDAY_MATRIX_TIMEOUT_SECONDS` may lower or raise it), and all temporary
state, build caches, Cargo targets, and Playwright output are removed when the
runner exits. Required toolchains and already-cached
dependencies must be installed before running it; the matrix does not download
them as a side effect.

Run one proof slice with `./scripts/first-run-matrix.sh <slice>` or run the
complete matrix with `make first-run-matrix`.

| Contract | Repeatable proof |
| --- | --- |
| Empty CLI first run | Setup and doctor handlers use temporary configuration/data paths; readiness reports redact private paths and fake-provider failures. The in-process turn service uses its existing fake narrator to commit a first action, then creates and restores a save. |
| Empty gateway/web installation | Rust authentication tests prove protected data, one-shot bootstrap, and direct bearer separation. Vitest and Playwright use empty-install and gateway route mocks to exercise installation readiness, distinct story onboarding, and one submitted playable action. |
| Desktop standalone | Existing Tauri config/standalone/lifecycle tests prove a fresh standalone profile has private, isolated config/data paths, version-matched sidecar planning, loopback readiness inputs, bounded diagnostics, and launch-secret redaction. |
| Desktop remote and transfer | Tauri tests prove remote profiles have no data path, validate the isolated server origin, reject path/type/oversize imports before dispatch, sanitize export names, and retain the configured origin. Gateway archive and world-template tests exercise the real archive/template contracts. |
| Provider matrix | Text-only rejection, compatible-local capability probes, local bridge transport, remote HTTPS bearer transport, direct adapters, authentication, capability failures, and retry-safe failure states run against local Axum/HTTP fakes. |
| Profile and recovery isolation | Profile tests keep standalone/remote state separate. The release SQL fixture upgrades from the previous supported schema; the backup/restore fixture verifies the source checksum survives a failed recovery migration and refuses a non-empty target. |

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
