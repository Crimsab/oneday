# Releases and changelog

OneDay uses Release Please in manifest mode. The source-controlled files are:

- `release-please-config.json` — release type and user-facing changelog sections.
- `.release-please-manifest.json` — the last released version.
- `CHANGELOG.md` — generated release history.

## Flow

1. Conventional Commits land on `main`.
2. `.github/workflows/release-please.yml` opens or updates one release PR.
3. The workflow explicitly dispatches CI on the release branch. This avoids the
   approval-only run created by GitHub when `GITHUB_TOKEN` opens a pull request.
4. The PR contains the next version and generated changelog entry.
5. Merging it creates the `vX.Y.Z` tag and GitHub Release.
6. The workflow runs the release gates, builds Linux/Windows archives, uploads
   them to the release, and publishes the versioned container image to GHCR.

Container releases use the tags `X.Y.Z`, `X.Y`, `X`, and `latest`. The image
includes OCI source metadata, a software bill of materials, and BuildKit
provenance. Public-repository releases also receive a GitHub artifact
attestation.

The manifest tracks the latest released version, and every release tag points to
the matching commit on the `main` lineage. Together these prevent Release Please
from restarting at `1.0.0` or including changes from an already published version.

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
