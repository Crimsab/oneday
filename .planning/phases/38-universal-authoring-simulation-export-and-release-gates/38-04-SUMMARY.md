# Summary 38.4 - Universal compatibility and release gates

Release is now gated by one reproducible matrix covering all Go packages,
targeted migration and legacy compatibility suites, strict story-pack loading,
the versioned minigame evaluation corpus, Rust formatting/tests/clippy/release,
React/Vitest, the production frontend build, desktop/mobile Playwright, and the
gateway Docker image. CI and release-please run the same relevant checks.

The existing release gate now invokes that universal matrix, verifies
friend-safe repository hygiene, builds the main, benchmark, ASCII benchmark,
Linux, and Windows artifacts, and rejects mismatched or dirty provenance.
Playwright outputs are explicitly ignored so a successful E2E run cannot make
an otherwise clean release appear dirty.

The final clean-checkout gate passed on commit `431c5a3`: Go 541 tests, Rust 47
tests, browser 100 tests, Playwright 12/12, minigame corpus 12/12 with every
required concern represented, Docker build, five release artifacts, and
`dirty: false` provenance. Live `homelab-main` verification retained schema
V35, integrity `ok`, zero foreign-key violations, two stories, 15 visual
assets, valid EPUB/replay exports, safe audio cleanup preview, and restart count
zero.
