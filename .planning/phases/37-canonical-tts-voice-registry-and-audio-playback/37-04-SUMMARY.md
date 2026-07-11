# Summary 37.4 - Pronunciation, retry, cleanup, and export

Multilingual pronunciation entries are revisioned per story/language and enter
cache identity. The browser can add and delete provider-guidance, IPA, and
X-SAMPA entries. Manual retry resets exhausted attempts explicitly; cancel and
retry update jobs/assets atomically and reject inactive-branch mutations.
Completion and failure paths cannot overwrite a concurrent cancellation.

Cache maintenance is audit-first. It invalidates unsafe/missing ready rows and
removes only unreferenced regular audio files inside the canonical root; dry
runs do not mutate, external paths and referenced files are retained, and
errors are bounded. The branch-specific audio manifest includes policy,
provider/voice registry, assignments, lexicon, immutable asset/job lineage, and
public media URLs while redacting internal file paths.

Local gates passed: Go 536 across 25 packages, `go vet`, Rust 47, Clippy clean
with only the documented pre-existing image-signature lint allowance, browser
100, production build, and Playwright 10/10 desktop/mobile. A copied-live-DB
preflight preserved schema V35, integrity, two stories, 15 visual assets, and
zero audio jobs/assets. Live `homelab-main` passed health, pronunciation,
cleanup dry-run, redacted export, desktop/mobile rendering, zero overflow/log
errors, and restart count zero. Backup:
`/opt/lab/backups/oneday/phase37-4-20260711`.
