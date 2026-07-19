# Security policy

## Supported versions

Security fixes target the latest released `1.x` version and the `main` branch.
Older releases may receive a fix only when the change can be backported safely.

## Report a vulnerability

Do not open a public issue for a vulnerability. Use a
[private GitHub security advisory](https://github.com/Crimsab/oneday/security/advisories/new)
and include:

- affected version or commit;
- impact and realistic attack path;
- the smallest safe reproduction;
- any suggested mitigation.

Remove API keys, authentication files, database content, private prompts, and
story text from reports. You should receive an initial response within seven
days. A fix and disclosure timeline will be coordinated according to severity.

## Scope

OneDay is designed for local and self-hosted use. Exposing the gateway directly
to an untrusted network requires deployment controls outside this repository,
including authentication, TLS, network policy, backups, and provider-key
protection.

## Release integrity

Official release assets are published only from a tagged `main` commit by
GitHub-hosted workflows. Each release includes SHA-256 checksums, coordinated
application/protocol/database/desktop metadata, and an SPDX JSON software bill
of materials. Public builds also receive GitHub artifact attestations. Verify a
download before running it:

```bash
sha256sum --check SHA256SUMS --ignore-missing
gh attestation verify oneday-X.Y.Z-linux-amd64.tar.gz \
  --repo Crimsab/oneday
```

Use the immutable container digest when verifying or deploying a container;
tags such as `latest` are convenience pointers. The release workflow scans
tracked content for secrets and scans locked dependencies and the published
container for known high and critical vulnerabilities. An automated scan is
evidence, not a guarantee that a release is vulnerability-free.

Desktop auto-updates require both HTTPS and a Tauri signature verified by the
public key embedded in the installed application. The update feed never contains
private signing material. A Tauri updater signature is separate from Windows
Authenticode: until an installer has a valid Authenticode signature and timestamp,
Windows may show an unidentified-publisher or SmartScreen warning.

Report checksum, SBOM, attestation, updater-signature, unexpected-publisher, or
release-lineage failures through a private security advisory. Do not post a
suspected signing-key compromise in a public issue.

## Maintainer secret handling

Release signing keys, certificate exports, passwords, OIDC credentials, and
recovery material must remain in an authorized external secret or managed signing
service. They must not be placed in repository variables, source files, workflow
artifacts, issue attachments, or diagnostic logs. Public updater keys and
certificate fingerprints are not secret, but changes to either are
release-sensitive and require the N-to-N+1 validation documented in
[Releases](docs/releases.md).
