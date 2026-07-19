# Security threat model

This document describes the public OneDay trust boundaries, what the software
does to reduce common risks, and what a deployer must still provide. It is not a
security guarantee, a substitute for the [security policy](../SECURITY.md), or
an authorization to expose a development listener to the internet.

## Assets and trust boundaries

Important assets include story databases and media, provider credentials and
paid-provider authority, desktop profile data, browser sessions, import/export
archives, and the integrity of desktop sidecars and updater metadata.

OneDay separates these boundaries:

- **Go engine and SQLite** own canonical state and transactional turn commits.
- **Gateway** is the HTTP/SSE/media/process adapter. It treats browser content,
  request metadata, imported files, and provider responses as untrusted input.
- **Browser** renders server state and submits intents; it is not an authority
  for canonical state or secrets.
- **Desktop remote profile** is a constrained webview for one server origin. It
  does not create a local copy of the remote story database.
- **Desktop standalone profile** owns a separate local configuration and data
  directory. Its loopback gateway and bundled sidecars are local trusted code,
  but their provider endpoints and the operating-system user account remain
  outside that boundary.
- **External providers and reverse proxies** are separate systems with their
  own identity, retention, billing, and availability policies.

Choose a data owner before play. Remote and standalone stores do not synchronize
or merge; a backup/restore or import/export operation deliberately crosses that
boundary and must be treated as data movement.

## Gateway web boundary

### Loopback, Host, Origin, and CSRF

The gateway listens on loopback by default. Its browser bootstrap credential is
one-shot: exchanging it consumes the credential, invalidates it in memory, and
issues a browser session that expires after 12 hours. The session is set in an
HttpOnly, `SameSite=Strict` cookie; a secure public origin receives a secure
host-only cookie. Browser session, bootstrap, and direct bearer credentials are
distinct values.

For every request, the gateway validates a single `Host` header against the
listener and configured allowed hosts. Authenticated mutations validate a
same-origin `Origin` when one is supplied, reject cross-site fetch metadata,
and reject a cookie-authenticated mutation that omits `Origin`. These controls
reduce DNS-rebinding, hostile-page, and CSRF risks; they do not authenticate an
untrusted public network by themselves.

Deployment requirements:

- keep the gateway loopback/private when a reverse proxy is present;
- terminate TLS at the public origin, preserve the public `Host`, and set
  `ONEDAY_GATEWAY_ALLOWED_HOSTS` to that exact authority;
- use a proxy that supplies the intended remote authentication, authorization,
  rate limits, request-size limits, and logging policy;
- never put a bootstrap token or direct bearer credential in a public URL,
  bookmark, support ticket, proxy log, or browser history;
- use a new bootstrap credential when the one-shot value has been consumed.

An interactive loopback terminal launch can print a bootstrap URL to that
terminal. Non-interactive or non-loopback starts require explicit bootstrap
configuration unless they are deliberately direct-bearer-only. Direct bearer
access (`ONEDAY_GATEWAY_AUTH_TOKEN`) is for trusted non-browser callers and is
not a browser-login substitute. It must be distinct from
`ONEDAY_GATEWAY_BOOTSTRAP_TOKEN`, at least 32 bytes, and held in a secret store
or process environment.

## Desktop and sidecar containment

The desktop settings webview is bundled local content with a deliberately small
native capability set. The remote story webview receives no Tauri native IPC
capabilities and is constrained to the configured scheme/host/port. This limits
the impact of a malicious page navigating away from the selected OneDay origin;
it does not make a compromised trusted server harmless.

Standalone starts version-matched gateway and engine sidecars only after
checking that the required sidecars and bundled web UI are present. It assigns a
fresh loopback endpoint and generates a per-launch direct bearer secret. That
secret is passed through the child environment, never persisted in settings or
placed on the command line. The desktop uses a Unix process group or Windows job
object so stopping/quitting can terminate the child process tree. Diagnostic
logs are bounded and redact the launch secret, but users should still sanitize
diagnostic material before sharing it.

The remaining risk is local code execution in the user's account: another
process running as that user can potentially read user-accessible data or attack
local provider services. Keep the operating system patched, limit account
access, and do not treat loopback as a cross-user security boundary.

## Imports, exports, and paths

Import archives are attacker-controlled. The gateway constrains supported
archive formats, archive size, entry count, and archive paths; it rejects unsafe
path components so an archive cannot write outside its intended import staging
area. Image uploads are size- and decode-limited and normalized before use.
Desktop file operations use native dialogs rather than arbitrary file paths
provided by web content.

These checks reduce zip-slip, path traversal, decompression, and oversized-image
risks, but they cannot make an imported story trustworthy. Treat imported text,
metadata, and media as untrusted content. Scan archives where policy requires,
review the source, and keep backups before replacing an existing store.

## Secrets and provider dispatch

Provider keys, bridge bearer tokens, and gateway credentials are write-only
configuration from the user interface: use environment variables or a secret
manager, not tracked YAML, browser storage, URLs, screenshots, or logs. Safe
configuration/health views report redacted state rather than secret values.

Narrative prompts, story text, and media requests can be transmitted to the
enabled provider. Enabling a provider therefore authorizes that provider to
receive the request content needed for the feature, subject to its own service
terms and retention rules. Configure only providers you trust, use least-privilege
credentials and spending limits, and disable optional image, speech, embedding,
or observability integrations when they are not required.

Provider routing is configuration, not an implicit promise of free usage. Check
the enabled provider order, selected models, fallback configuration, and image
bridge route before a paid run. A fallback can make another configured provider
eligible after an error or unavailability condition. Do not enable a paid
provider as a fallback unless that dispatch is intentional and budgeted. Text
turns can continue without image or speech generation; set image generation to
`text-only`/disabled when you do not want media requests.

## Updates, packages, and provenance

The checked-in desktop configuration disables updater artifacts. A usable
desktop updater requires an independently operated HTTPS feed, a configured
public verification key, and releases signed by the corresponding private key.
The private signing key belongs only in the release secret store. Do not trust
an update merely because it uses the OneDay name; verify the publisher, version,
target platform, signature metadata where provided, and the feed origin.

Standalone sidecars are expected to match the desktop package version. A missing
or mismatched component must fail startup rather than be substituted from an
unknown path. External provider CLIs, local model servers, containers, and
reverse proxies are also supply-chain dependencies; obtain them from their own
trusted channels and patch them independently.

## Public repository sanitation

Public documentation, examples, fixtures, issues, and commits must remain safe
to publish. Never add real credentials, `.env` files, private provider URLs,
private addresses, hostnames, absolute operator paths, story databases, saves,
generated user media, raw diagnostics, or production archives. Use placeholders
such as `https://oneday.example.com` and sanitized `oneday doctor --json`
output instead.

Before opening a public issue or pull request, inspect the diff and attachments
for secrets and environment details. Report suspected vulnerabilities through
the [security policy](../SECURITY.md), not by publishing a working exploit or
credential-bearing reproduction.

## Residual risk and response

OneDay cannot protect data once a trusted provider, reverse proxy, operating
system account, or administrator is compromised. It also cannot guarantee the
availability, price, privacy, or model behavior of third-party services. Keep
tested backups, define who may access a remotely exposed deployment, monitor
provider billing, rotate a leaked credential immediately, invalidate/restart a
gateway after bootstrap-token exposure, and follow the security policy for a
suspected product vulnerability.
