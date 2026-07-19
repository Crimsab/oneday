# OneDay Desktop

OneDay Desktop is a Tauri 2 client for Windows and Linux. At first use, choose
one of two intentionally separate profiles:

| Profile | What runs | Where canonical stories live |
| --- | --- | --- |
| Remote | A native window pointed at one existing OneDay gateway | The configured server's SQLite data directory |
| Standalone | Bundled OneDay engine, gateway, and web UI on a local loopback port | A new, isolated desktop profile on this device |

Remote mode does not create a story database, queue offline turns, or retain a
copy of remote stories. Standalone mode does create a local database, but it
does not read, merge with, or synchronize the remote server or terminal
configuration. A profile switch is only a connection switch. Use explicit
server import/export and a backup when moving data between stores.

Use the browser PWA when an installable browser experience is enough. Use this
desktop client when tray behavior, native notifications, autostart, or native
file dialogs are useful. Package update support is not assumed: it is disabled
in ordinary builds and requires a separately operated signed update feed.

## First-run readiness

For a remote profile, enter a complete server origin such as
`https://oneday.example.com`. The app accepts plain HTTP only for `localhost`,
`127.0.0.1`, or `::1` development. It rejects a URL containing credentials, a
query, fragment, or extra path. The remote server must already be healthy,
configured with a narrative provider, and deployed with TLS/authentication as
appropriate for its network.

For standalone, choose **Run on this device**. Startup verifies that the desktop
bundle contains matching engine and gateway sidecars and a bundled web build,
then starts the gateway on a fresh loopback port. The local profile still needs
a working narrative provider before it can create narrative turns. Run
`oneday doctor` against the profile configuration when provider or storage
readiness is unclear. Images and speech can remain disabled: text-only media
mode is supported and media failures do not block canonical text turns.

Standalone is not a promise of fully offline AI. It keeps data and the gateway
local, but a configured remote narrative provider still needs network access.
Use a local provider only if you require local model execution, and verify it
with the normal readiness checks.

## Local locations and backups

The desktop uses the operating system's per-user application directories with
the public bundle identifier `dev.oneday.desktop`. Environment overrides such
as `XDG_CONFIG_HOME` and `XDG_DATA_HOME` take precedence on Linux.

| Platform | Desktop setting | Standalone profile root |
| --- | --- | --- |
| Linux | `$XDG_CONFIG_HOME/dev.oneday.desktop/desktop.json` (default `~/.config/...`) | `$XDG_DATA_HOME/dev.oneday.desktop/profiles/<profile-id>/` (default `~/.local/share/...`) |
| Windows | `%APPDATA%\dev.oneday.desktop\desktop.json` | `%APPDATA%\dev.oneday.desktop\profiles\<profile-id>\` |

`<profile-id>` is an opaque identifier, not a story name. A standalone profile
root contains its own `config.yaml`, `data/` directory, and bounded local
diagnostic log. Remote mode writes only `desktop.json`; its story data remains
on the remote server.

To back up standalone data, stop its local gateway from the settings window or
quit the desktop app, then copy the whole `data/` directory (or the entire
profile root if you also need its configuration). Restore only into a stopped
standalone profile and keep the directory contents together. For remote mode,
back up on the server according to [Configuration](configuration.md#game-and-storage)
or [Docker](docker.md#backup-and-restore). Restoring is replacement of one
target store, not synchronization or merging.

## Security model

The app has two deliberately separate webviews:

- `settings` loads only the bundled local UI and receives the small capability
  set declared in `desktop/src-tauri/capabilities/settings.json`;
- `main` loads the configured OneDay server and receives no Tauri capability or
  remote IPC access.

Navigation in `main` is restricted to the configured origin. The first-run URL
validator requires HTTPS, except for `localhost`, `127.0.0.1`, and `::1` during
development. User names, passwords, query strings, fragments, and additional
paths are rejected. No private hostname is compiled into the application.

The native import/export commands open their own operating-system dialog. They
do not accept arbitrary paths from web content. Imports are limited to OneDay
ZIP archives and JSON world templates and use the same size limits as the
gateway. Complete archives are streamed to the server rather than copied into a
desktop database in remote mode; standalone imports go to that standalone
profile's local gateway.

The standalone launch credential is generated per process launch, passed only
through the child process environment, and not stored in desktop settings or
command-line arguments. The desktop contains the local gateway/engine process
tree (a process group on Unix or job object on Windows) and terminates it on
stop/quit. Bounded local diagnostics redact that launch credential, but logs
can still contain operational context: treat them as private and sanitize them
before sharing.

## User behavior

- Closing either window hides it; **Quit** in the tray menu exits the process.
- Autostart is opt-in and starts OneDay minimized in the tray.
- Notification permission is requested only after the user enables it.
- Complete ZIP and world-template import/export use the configured server.
- A reverse proxy that adds interactive authentication must also provide an
  authentication design for native import/export requests. The remote story
  webview continues to use its normal browser session.
- The desktop validates its configured server origin and blocks navigation to a
  different scheme, host, or port. This prevents ordinary web navigation from
  acquiring native desktop permissions; it does not replace server-side access
  controls.

## Build locally on Linux

Install WebKitGTK 4.1, AppIndicator, librsvg, and the other platform packages
listed by Tauri for the Linux distribution. Then run:

```bash
cd desktop
bun install --frozen-lockfile
bun run check
bun run tauri build --bundles appimage,deb
```

The bundles are written below `desktop/src-tauri/target/release/bundle/` and are
not tracked by Git. Windows NSIS installers are built by the dedicated desktop
workflow on a Windows runner.

## Signed updater

Updater artifacts and the updater runtime are disabled in ordinary builds. A
release operator must first create a Tauri signing key outside the repository,
publish an HTTPS update feed, and build with both values present:

```text
ONEDAY_UPDATER_ENDPOINT=https://releases.example.com/oneday/{{target}}/{{arch}}/{{current_version}}
ONEDAY_UPDATER_PUBKEY=<public minisign key>
```

The private signing key must only be supplied to the release job through its
secret store. It must never be committed. A signed release configuration must
also enable updater artifacts in a private Tauri configuration overlay; the
checked-in default keeps `createUpdaterArtifacts` false so a normal build cannot
pretend to publish usable updates.

## Scope and limitations

The desktop client requires a reachable server for story reads and mutations.
Offline editing and local-to-server database synchronization are intentionally
out of scope: conflict-free story synchronization would require a separate
product protocol, authentication, ownership rules, asset reconciliation, and
user-visible conflict resolution. Standalone provides local persistence only
for its own profile; it is not a replicated remote client.

The desktop package is only as complete as the bundle supplied to it. It fails
instead of attempting a mismatched local start when matching sidecars or web
assets are absent. External narrative, image, speech, embedding, and update
services remain separately configured dependencies. See the [security threat
model](security-threat-model.md) for the trust and supply-chain boundaries.
