# OneDay Desktop

OneDay Desktop is a Tauri 2 client for Windows and Linux. It connects to an
existing OneDay gateway; the Go engine and the gateway SQLite database remain
the only canonical story state. The desktop application does not create a
second database, queue offline turns, or attempt bidirectional synchronization.

Use the browser PWA when an installable browser experience is enough. Use this
desktop client when tray behavior, native notifications, autostart, native file
dialogs, or signed application updates are useful.

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
desktop database.

## User behavior

- Closing either window hides it; **Quit** in the tray menu exits the process.
- Autostart is opt-in and starts OneDay minimized in the tray.
- Notification permission is requested only after the user enables it.
- Complete ZIP and world-template import/export use the configured server.
- A reverse proxy that adds interactive authentication must also provide an
  authentication design for native import/export requests. The remote story
  webview continues to use its normal browser session.

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

## Scope of version 1

The desktop client requires a reachable server for story reads and mutations.
Offline editing and local-to-server database synchronization are intentionally
out of scope: conflict-free story synchronization would require a separate
product protocol, authentication, ownership rules, asset reconciliation, and
user-visible conflict resolution.
