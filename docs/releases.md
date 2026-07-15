# Releases and changelog

OneDay uses Release Please in manifest mode. The source-controlled files are:

- `release-please-config.json` — release type and user-facing changelog sections.
- `.release-please-manifest.json` — the last released version.
- `CHANGELOG.md` — generated release history.

## Flow

1. Conventional Commits land on `main`.
2. The consolidated `CI` workflow verifies that exact `main` commit once.
3. A successful main CI run triggers `.github/workflows/release-please.yml`,
   which opens or updates one release PR.
4. `.github/workflows/release-pr.yml` confirms that the PR base has a successful
   `Full verification` check, rejects files other than the generated manifest
   and changelog, and atomically merges the exact release SHA while `main` still
   points to the verified base.
5. The merge dispatches the publication phase, which creates the `vX.Y.Z` tag
   and GitHub Release.
6. Publication builds and smoke-checks Linux/Windows archives, uploads them to
   the release, and publishes the versioned container image to GHCR. It does not
   repeat the full application test suite already passed by the tagged lineage.

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

## Local release-sensitive verification

```bash
make release-check
git status --short
```

The automated release PR is metadata-only and merges without a manual click
only when its exact base and head satisfy the checks above. Do not hand-edit a
generated release PR unless the correction also belongs in source or in
`release-please-config.json`.
