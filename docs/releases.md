# Releases and changelog

OneDay uses Release Please in manifest mode. The release tag is the authority for
the application and desktop version; generated metadata records the protocol and
database compatibility levels shipped with that tag.

The source-controlled release metadata is:

- `release-please-config.json` — release type and changelog sections;
- `.release-please-manifest.json` — the latest released application version;
- `CHANGELOG.md` — generated release history.

## Public release flow

1. Conventional Commits land on `main`.
2. `CI` verifies the exact commit on a GitHub-hosted runner. It includes Actionlint,
   Gitleaks, Trivy, `govulncheck`, tests, cross-compilation, browser tests, and a
   complete container build.
3. A successful main run triggers Release Please, which opens or updates one
   metadata-only release pull request.
4. `release-pr.yml` verifies the successful `Full verification` check, immutable
   base and head SHAs, changelog version, and metadata-only diff before merging.
5. Release Please creates `vX.Y.Z` and its GitHub Release, then dispatches
   `publish-release.yml` for that exact tag.
6. The publisher independently rebuilds the CLI archives twice, compares their
   digests, builds the container, scans release inputs, creates an SPDX SBOM,
   creates GitHub artifact attestations, and uploads one final checksum index.
7. When signed desktop publication is enabled, native Linux, Windows, macOS
   Apple Silicon, and macOS Intel jobs build version-matched engine and gateway
   sidecars, Tauri installers, updater signatures, and an HTTPS `latest.json`
   feed.

All jobs in this path use GitHub-hosted runners. The release path has no private self-hosted
runner, hostname, filesystem, cache, or credential dependency. Intermediate
workflow artifacts expire after one day; unsigned pull-request desktop packages
expire after two days. GitHub Release assets are the durable deliverables.

The first live end-to-end run of the desktop jobs and updater feed must occur after
the repository is public. Static validation can prove workflow syntax, scripts,
locked inputs, and unsigned native packaging, but it cannot prove public Sigstore
attestation, GitHub Release URL behavior, or the repository's external signing
configuration. That public-run smoke is explicitly **pending**.

## Coordinated versions

`scripts/release-metadata.sh vX.Y.Z` writes the compatibility record included in
the CLI archives and uploaded beside them:

| Field | Source |
| --- | --- |
| `applicationVersion` | Release tag without `v` |
| `desktopVersion` | Same release tag, injected into the ephemeral desktop manifests |
| `gatewayProtocolVersion` | Canonical Go gateway protocol constant |
| `databaseSchemaVersion` | Highest append-only SQLite migration |
| `sourceCommit` and `sourceDate` | Tagged Git commit |

The desktop source manifests may keep a development version between releases.
The release job updates its clean checkout only, updates the local lock entry,
builds sidecars named for the release version and native target, and validates
the resolved Cargo and JavaScript package versions before invoking Tauri. These
ephemeral edits are never committed by the workflow.

Container releases use `X.Y.Z`, `X.Y`, `X`, and `latest` and publish one
multi-platform index for `linux/amd64` and `linux/arm64`. The image carries OCI
source metadata, a BuildKit SBOM, and maximum-mode provenance. Public releases
also receive a GitHub artifact attestation bound to the image digest.

## Reproducible package inputs

Linux and Windows CLI archives use:

- the tagged source commit and the `go.mod` toolchain;
- `-trimpath`, disabled VCS auto-stamping, explicit build metadata, and
  `SOURCE_DATE_EPOCH` from the commit;
- stable file ordering, ownership, timestamps, and ZIP metadata;
- two independent builds whose SHA-256 lists must match before publication.

Desktop jobs use native GitHub-hosted Linux, Windows, macOS Apple Silicon, and
macOS Intel runners, locked Cargo and Bun dependency graphs, the same tag-derived
build metadata, and version-matched sidecars. Native Tauri, AppImage, deb, NSIS,
DMG, and cryptographic-signature tooling can add platform metadata or timestamps,
so the workflow claims reproducible **inputs**, not bit-for-bit identical signed
installers.

Every release publishes `SHA256SUMS`, `release-metadata.json`, and an SPDX JSON
SBOM. For public repositories, GitHub's Sigstore-backed attestations bind build
provenance to CLI archives, desktop artifacts, the checksum index, updater feed,
and container digest.

Release asset names are additive-only. Before uploading, the publisher downloads
every existing same-name asset and compares its bytes. Identical assets are
skipped, any mismatch fails the job before new files are uploaded, and only
missing names are sent to GitHub without replacement semantics. If a job stops
after a partial upload, retry the publish job while its one-day workflow inputs
remain available; the preflight safely skips the identical subset. A rebuilt
workflow may continue only when its bytes match. A mismatch requires
investigation and a new release tag, never deletion or silent replacement of the
published binary.

## Signed desktop updater

Normal and pull-request builds keep `createUpdaterArtifacts` disabled. Release
builds overlay `desktop/src-tauri/tauri.release.conf.json`, which enables Tauri v2
updater artifacts. The application embeds only:

```text
https://github.com/Crimsab/oneday/releases/latest/download/latest.json
```

and the configured public updater key. The private updater key is read only by
the native release jobs. It is never stored in the repository, artifacts, logs,
or `latest.json`.

Signed desktop publication is gated by the repository variable
`ONEDAY_DESKTOP_RELEASES_ENABLED=true`. The release environment must already
provide:

- repository variable `ONEDAY_UPDATER_PUBKEY`;
- secret `TAURI_SIGNING_PRIVATE_KEY`;
- optional secret `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`.

The workflow does not create, print, or transport those values. Keep signed
desktop publication disabled until an authorized release operator has provisioned
and backed up the key outside the repository. Losing the private key prevents
updates to existing installations; replacing the public key requires a planned
transition release signed by the old trust root.

`latest.json` contains only SemVer, release notes, the tagged commit date, HTTPS
URLs, and the literal Tauri signatures for `linux-x86_64`,
`windows-x86_64`, `darwin-aarch64`, and `darwin-x86_64`. Using the commit date
keeps an identical partial-run retry from changing the feed bytes. Publication
verifies:

- all four platform entries and assets exist;
- every URL is the exact HTTPS URL for the tagged GitHub Release;
- manifest signatures equal the generated `.sig` files;
- Minisign verifies each updater asset with the configured public key.

Tauri updater signing authenticates application updates. It is not Windows
Authenticode signing or Apple code signing/notarization.

## Authenticode and SmartScreen

The current public workflow prepares updater-signed NSIS installers but does not
provision an Authenticode identity. Until an authorized maintainer adds a
hardware-backed, managed signing service or certificate workflow, Windows users
should expect an unidentified-publisher or SmartScreen warning. Do not describe
an installer as Windows-signed merely because its Tauri `.sig` exists.

Before advertising a warning-free Windows installation:

1. sign the executable and final NSIS installer with a trusted Authenticode
   certificate and RFC 3161 timestamp;
2. verify the signature with `Get-AuthenticodeSignature` and `signtool verify /pa`;
3. confirm the displayed publisher matches the project's legal publisher;
4. test a fresh browser download on supported Windows versions;
5. document that OV certificates can require reputation-building and may still
   trigger SmartScreen, while reputation policy remains controlled by Microsoft.

Authenticode material and credentials must use an external managed signing
service or GitHub environment secrets. They must never enter source control or a
workflow artifact.

## Apple code signing and notarization

The current workflows prepare updater-signed `.app.tar.gz` packages and DMGs for
both Apple Silicon and Intel, but do not provision an Apple Developer identity or
notarization credentials. Until an authorized maintainer adds those credentials,
downloaded macOS builds may require an explicit user approval and must not be
described as notarized.

Before advertising a warning-free macOS installation:

1. sign the application, bundled sidecars, and DMG with the intended Developer ID;
2. submit the package to Apple's notary service and wait for success;
3. staple the notarization ticket where supported;
4. verify with `codesign --verify --deep --strict` and `spctl --assess`;
5. smoke-test fresh downloads on Apple Silicon and Intel macOS.

Apple signing identities, App Store Connect credentials, and notarization
secrets belong in a protected GitHub environment. They must never enter source
control or workflow artifacts.

## N-to-N+1 updater test

Run this procedure before enabling desktop updates and after changing updater,
installer, signing, or feed configuration. Use a staging HTTPS origin and the
same secured updater trust root intended for release; do not test by weakening
TLS or signature checks.

1. Build version N from its tag with the staging HTTPS endpoint embedded. Record
   `release-metadata.json`, `SHA256SUMS`, the public key fingerprint, and the two
   installer hashes for every tested platform.
2. Publish N's updater assets and valid feed to the staging origin. Install N on
   clean supported Ubuntu, Windows, and macOS machines.
3. Start standalone mode, create a story database, complete at least one turn,
   export a backup, and record the database schema reported by
   `oneday-db-check`.
4. Build N+1 from the next tag with the same endpoint and public key. Confirm its
   compatibility record has the intended application, desktop, protocol, and
   database versions.
5. Generate the N+1 feed with `scripts/release-updater-manifest.sh generate`, then
   run `verify` with the public key before publishing it over HTTPS.
6. From the running N installation, check for updates, install N+1, restart, and
   confirm the application reports N+1. Start standalone mode and verify the
   bundled version-matched sidecars launch.
7. Open the existing story, check SQLite integrity and foreign keys, play another
   turn, and confirm the backup from step 3 remains restorable.
8. Repeat on Linux AppImage, Windows NSIS, macOS Apple Silicon, and macOS Intel.
   On Windows, confirm the updater can close the running app and that the
   current-user install mode does not request elevation. On macOS, confirm the
   signed application passes Gatekeeper and relaunches after updating.
9. Negative-test a modified updater asset, modified signature, wrong public key,
   malformed platform entry, and HTTP feed. Every case must fail before install
   without altering the existing application or database.
10. Restore the staging feed to N+1, repeat one successful update, and retain the
    test record with hashes and observed versions outside the repository.

Do not exercise N-to-N+1 against the production `latest.json` until both versions
are intended public releases; the GitHub `releases/latest` route is shared by all
installed production clients.

## Version rules

- `feat:` → minor version
- `fix:` or `perf:` → patch version
- `BREAKING CHANGE:` footer or `!` marker → major version
- documentation, tests, CI, build, and chore commits remain hidden from the
  user-facing changelog

Use `Release-As: X.Y.Z` in a commit footer only when intentionally overriding the
calculated version.

## Verification

After downloading release assets:

```bash
sha256sum --check SHA256SUMS --ignore-missing
gh attestation verify oneday-X.Y.Z-linux-amd64.tar.gz \
  --repo Crimsab/oneday
```

The SBOM is a release asset and can be inspected or scanned independently. The
container attestation is bound to its registry digest rather than a mutable tag.

For local release-sensitive changes, run:

```bash
make release-check
git status --short
```

Do not hand-edit a generated release pull request. Correct source configuration
or Release Please configuration and let the automation regenerate it.
