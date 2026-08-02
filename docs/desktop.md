# OneDay Desktop

OneDay Desktop is a Tauri 2 client for Windows, Linux, and macOS. At first use,
choose one of two intentionally separate profiles:

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
desktop client when a bundled local gateway, tray behavior, native
notifications, autostart, or native file dialogs are useful. Public release
builds publish signed OneDay updates. Every desktop build can check that public
feed; it installs only a package that verifies with the embedded public key.

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
a working narrative provider before it can create narrative turns.

The desktop settings show all four narrative connection types. Codex uses this
subscription path:

1. OneDay checks the official `codex` CLI, the Codex/ChatGPT desktop app, and
   the obsolete `codex-cli` command separately. The desktop app can contain its
   own agent runtime without publishing a `codex` command to `PATH`; finding the
   app therefore does not falsely mark the CLI as ready.
2. If none is found, **Install Codex** downloads the pinned official OpenAI
   release for the current platform, verifies its size and SHA-256 digest, and
   extracts only the expected executable. On Windows it installs the CLI in
   `%LOCALAPPDATA%\Programs\OpenAI\Codex\bin`, adds that directory to the user
   `PATH`, and broadcasts the environment change. No download starts before
   that click. Linux and macOS retain the app-scoped component fallback.
3. **Sign in** starts `codex login`. A Windows global installation uses the
   normal `%USERPROFILE%\.codex` home shared by the Codex app and terminal.
   OneDay does not parse or copy the OAuth credential.
4. Open **Configure models** and select Codex and a model. The desktop restarts
   its gateway after sign-in so the executable is available.

Windows previews that already contain the older private OneDay component show
**Make global**. Migrating installs the verified executable in the normal
per-user CLI location; the private copy remains only as a fallback and no
credential is copied from its isolated home.

Codex is visibly marked **Recommended** because it is the most complete
subscription-backed OneDay path: the same sign-in can support narrative turns
and the local image bridge. The other providers remain first-class options and
are never installed or enabled implicitly.

When Codex is available, a standalone profile also starts its included
`imagegen-bridge` on a new loopback port with a fresh, in-memory bearer token.
The default Codex image model is `gpt-image-2`. It does not reuse a saved remote
bridge address. If the included component cannot start, OneDay disables that
image path for the local profile instead of reporting it as usable.

Claude Code has a parallel native path. OneDay detects `claude`, checks
`claude auth status`, and opens `claude auth login` when sign-in is needed. If
it is missing, Windows can install the official `Anthropic.ClaudeCode` WinGet
package and macOS can use the official Homebrew cask. On Linux, or when that
package manager is absent, OneDay opens Anthropic's official guide. OneDay does
not read or copy Claude credentials.

OpenRouter and LiteLLM-compatible endpoints appear beside those subscription
providers and open the same protected model configuration. Their API keys are
write-only after saving. Choosing Claude, OpenRouter, or LiteLLM never downloads
Codex. Images and speech can remain disabled: text-only media mode is supported
and media failures do not block canonical text turns.

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
| macOS | `~/Library/Application Support/dev.oneday.desktop/desktop.json` | `~/Library/Application Support/dev.oneday.desktop/profiles/<profile-id>/` |

`<profile-id>` is an opaque identifier, not a story name. A standalone profile
root contains its own `config.yaml`, `data/` directory, and bounded local
diagnostic log. Remote mode writes only `desktop.json`; its story data remains
on the remote server.

On Windows, a Codex CLI installed by OneDay lives under
`%LOCALAPPDATA%\Programs\OpenAI\Codex\bin` and uses the normal
`%USERPROFILE%\.codex` home. On Linux and macOS, a managed fallback lives under
the OneDay application data root in `components/codex/`; its OAuth home is
separate from a system Codex installation. Treat either Codex home as private
and include it in a device backup only if the backup storage is appropriate for
credentials.

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

- On Windows, release installers include the Microsoft Edge WebView2 Evergreen
  bootstrapper. Portable QA builds use the WebView2 Runtime already supplied
  by current Windows installations and intentionally contain only OneDay.
- A startup failure is never intentionally silent. Windows shows a native error
  dialog and writes a bounded diagnostic log to
  `%LOCALAPPDATA%\dev.oneday.desktop\logs\desktop-bootstrap.log`.
- An invalid `desktop.json` is renamed to a timestamped
  `desktop.invalid-*.json` and ignored. This resets only desktop connection
  settings; standalone story data and remote server data are not changed.
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

## Build locally

For a fast UI and setup test, use the development command first:

```bash
cd desktop
bun install --frozen-lockfile
bun run dev:desktop
```

Vite reloads TypeScript, HTML, and CSS changes in the running desktop window.
Rust changes rebuild the debug backend and restart the window. This development
mode intentionally omits the packaged engine, gateway, and image bridge, so it
is for the desktop UI, provider setup, and remote-server flow. It reports a
clear incomplete-package error if you try to start a standalone profile.

To create a quick native executable without an installer, run:

```bash
cd desktop
bun run build:debug-ui
```

On Windows this writes `src-tauri/target/debug/oneday-desktop.exe`. It has the
same UI/setup-only scope as development mode. Use the normal package build for
the complete standalone runtime.

Use a native release build only when you need to test the complete installer or
the bundled standalone runtime. That build compiles optimized Rust code and
packages the sidecars, so it is necessarily slower than development reloads.

Build the desktop executable through `tauri build`, including when a custom
Cargo runner is used for cross-compilation. Do not publish the result of a
direct `cargo build` or `cargo xwin build`: those commands do not select
Tauri's production launcher and can leave the executable pointed at the Vite
`devUrl`. Release workflows verify that the generated binary actually embeds
the hashed files from `desktop/dist` before uploading an installer.

Install WebKitGTK 4.1, AppIndicator, librsvg, and the other platform packages
listed by Tauri for the Linux distribution. Then run:

```bash
cd desktop
bun install --frozen-lockfile
bun run check
bun run tauri build --bundles appimage,deb
```

The bundles are written below `desktop/src-tauri/target/release/bundle/` and are
not tracked by Git. On a Mac, use the same setup and run:

```bash
cd desktop
bun install --frozen-lockfile
bun run check
bun run tauri build --bundles app,dmg
```

The dedicated desktop workflow builds Linux AppImage/deb, Windows NSIS, macOS
Apple Silicon app/DMG, and macOS Intel app/DMG packages on native hosted
runners. Ordinary workflow artifacts are deliberately updater-unsigned. Public
release jobs add Tauri updater signatures; Apple Developer signing and
notarization still require protected maintainer credentials.

## Signed updater

The settings window reports the installed version, checks the stable HTTPS
feed, shows the available version and notes, and requires an explicit
**Install and restart** action. The current build does not need to be code
signed to check the feed. Tauri verifies the downloaded updater artifact with
the embedded public key before OneDay stops the local gateway. A verification
or download failure therefore leaves both the running application and story
data unchanged.

The official endpoint and public updater key are embedded in the application.
A release operator can override them for a staging feed or planned key rotation:

```text
ONEDAY_UPDATER_ENDPOINT=https://github.com/Crimsab/oneday/releases/latest/download/latest.json
ONEDAY_UPDATER_PUBKEY=<base64-encoded public Minisign key>
```

The private signing key is supplied only to the release job through its secret
store and is never committed. The checked-in default keeps
`createUpdaterArtifacts` false: ordinary builds can check and install a
verified public update, but they never create or publish updater artifacts.

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
