# Releases and changelog

OneDay uses Release Please in manifest mode. The source-controlled files are:

- `release-please-config.json` — release type and user-facing changelog sections.
- `.release-please-manifest.json` — the last released version.
- `CHANGELOG.md` — generated release history.

## Flow

1. Conventional Commits land on `main`.
2. `.github/workflows/release-please.yml` opens or updates one release PR.
3. The PR contains the next version and generated changelog entry.
4. Merging it creates the `vX.Y.Z` tag and GitHub Release.
5. The workflow runs the release gates, builds Linux/Windows archives, and uploads them to the release.

The manifest currently tracks the actual latest tag, preventing Release Please
from restarting at `1.0.0` when configuration changes.

`last-release-sha` temporarily anchors the `1.8.0` changelog to the `1.7.0`
release merge on the current `main` lineage. Remove that override after `v1.8.0`
is published; the new tag will then be the normal release marker.

## Version rules

- `feat:` → minor version
- `fix:` or `perf:` → patch version
- `BREAKING CHANGE:` footer or `!` marker → major version
- documentation, tests, CI, build, and chore commits remain hidden from the user-facing changelog

Use `Release-As: X.Y.Z` in a commit footer only when intentionally overriding the
calculated version.

## Before merging a release PR

```bash
make release-check
git status --short
```

Confirm the changelog is accurate, CI is green, and the working tree is clean.
Do not hand-edit a generated release PR unless the correction also belongs in
source or in `release-please-config.json`.
