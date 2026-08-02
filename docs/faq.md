# Frequently asked questions

## Can I use my Codex subscription?

Yes. Codex is the most complete subscription-backed path in OneDay because it
can cover narrative generation and generated images without an OpenAI Platform
API key. The exact setup depends on the runtime:

New native and desktop-local configurations select `gpt-5.6-luna` with `low`
reasoning automatically. Both values stay visible and editable in the guided
operator setup; **Final check** performs a real provider readiness request.

The Codex/ChatGPT desktop app and the `codex` command are related but are not
the same installation surface. The app may keep its native agent private and
therefore work even when PowerShell cannot find `codex`. OneDay reports this as
**App found**, then offers **Add Codex CLI**. A legacy `codex-cli` executable is
reported separately and is never mistaken for the current official CLI.

- **Terminal:** install the Codex CLI, run `codex login`, and choose
  **Codex OAuth** in `oneday setup`. OneDay calls `codex exec`; the Codex CLI
  reads and refreshes its own login.
- **Desktop, Run on this device:** choose Codex in the desktop settings. OneDay
  first detects the app and CLI independently. If the official CLI is missing,
  **Install Codex** downloads
  a pinned official OpenAI release only after you click it, verifies its
  SHA-256 digest, and installs it in the normal per-user CLI location. On
  Windows it also updates the user `PATH`; the terminal, Codex app, and OneDay
  share `%USERPROFILE%\.codex`. Sign in, then choose Codex and a model in
  OneDay Setup.
- **Desktop, Connect to a server:** the desktop does not run a local provider.
  The remote gateway must have its own provider configuration.
- **Browser with a native gateway:** yes, when `codex` and its authenticated
  state are available to the gateway process. The browser itself never runs or
  reads Codex credentials; the server calls `codex exec`.
- **Docker image generation:** authenticate Codex on the host and enable the optional
  `imagegen-bridge` profile. Its helper copies only the host `auth.json` into a
  dedicated Docker volume. This is explicit rather than automatic, so OneDay
  never mounts your full home directory or silently imports a credential.
- **Standard Docker narrative generation:** not currently supported through the
  host Codex login. The standard gateway image does not contain the Codex CLI.
  Use LiteLLM or OpenRouter for narrative generation, or run the terminal client
  on the authenticated host.

For the Docker image profile, follow
[Optional Codex OAuth imagegen-bridge profile](docker.md#optional-codex-oauth-imagegen-bridge-profile).
The bridge bearer and Codex OAuth file are different secrets.

After creating `.env.imagegen-bridge` as described in that guide, the credential
step is:

```bash
codex login
./scripts/imagegen-bridge-copy-oauth.sh
```

## Can I use my Claude subscription?

Yes, for narrative generation through an installed and authenticated
**Claude Code** CLI. OneDay calls `claude -p`, so Claude Code remains responsible
for login and subscription access.

Desktop standalone detects an existing Claude Code installation. If it is
missing, **Install Claude** uses the official WinGet package on Windows or the
official Homebrew cask on macOS when that package manager is available. Other
systems open Anthropic's installation guide. **Sign in** runs
`claude auth login`; OneDay checks readiness with `claude auth status` and never
parses the resulting credential.

Enable **Claude Code** in the operator model connections and move it to the
front of the provider priority, or configure `ai.claude_code` in
`config.yaml`. Run `oneday doctor` before creating a story.

Claude Code does not provide OneDay's image-generation route and does not
provide embeddings in the current integration. Pair it with a separate image
provider and, when RAG is enabled, a separate embedding provider. The standard
Docker image also does not contain the Claude CLI, so copying the host Claude
login into that image alone is not enough.

## Does OneDay import Codex or Claude credentials automatically?

Not from an arbitrary host installation. In a terminal or native-gateway
installation, OneDay launches the selected local CLI and the CLI reads its own
authenticated state. OneDay does not parse or duplicate that state.

Desktop standalone has explicit provider actions. It detects system Codex and
Claude Code installations first. **Install Codex** downloads a pinned official
binary; on Windows it installs globally for the current user and uses the
normal Codex home, while Linux and macOS retain an app-scoped fallback.
**Install Claude** delegates to the official system package when supported.
Each CLI owns its login flow and credential storage. Nothing is copied from
another home directory, and remote desktop mode never receives either local
credential.

## Why does the Windows desktop executable appear to do nothing?

OneDay Desktop uses Microsoft Edge WebView2 for its interface. Current release
installers install that prerequisite when necessary. Portable QA packages use
the Runtime already supplied by current Windows installations; extract the
whole OneDay folder, not only the main executable.

Other startup failures produce a native dialog and a diagnostic log at
`%LOCALAPPDATA%\dev.oneday.desktop\logs\desktop-bootstrap.log`. Invalid desktop
connection settings are quarantined automatically without deleting stories.

The Docker Codex image profile is the one deliberate exception: its helper
copies only Codex `auth.json` into an isolated volume used by
`imagegen-bridge`. You must run that helper explicitly. OneDay does not support
an equivalent Claude credential import because the standard image has no Claude
runtime.

## What context does the model receive?

Every turn receives structured story state without requiring embeddings:
setting, character and world state, relevant NPCs, recent messages, branch
state, chapter information, spatial context, and the current action. This is
the normal context window and works with Codex, Claude Code, LiteLLM, and
OpenRouter.

Optional RAG adds long-term recall. OneDay periodically summarizes older story
windows, embeds those summaries and durable lore in SQLite, embeds the current
action, retrieves the closest chunks, and adds them as **Relevant Memory** to
the same request sent to the active narrative provider. Retrieval failure is
non-blocking: the turn continues with normal story context.

## Do Codex and Claude Code create embeddings?

No. OneDay's Codex and Claude Code adapters are generation adapters. They do
not expose an embeddings endpoint. RAG still works with either as the narrator
when you configure one of these embedding paths:

- local Ollama;
- a custom local HTTP embedding endpoint;
- a LiteLLM-compatible embeddings endpoint;
- OpenRouter when the selected route supports embeddings.

The narrator and embedder are independent. For example, Codex can write the
turn while local Ollama creates and searches the vectors.

## Does setup let me choose the narrator and embeddings?

Yes, with one current difference between the interfaces:

- Terminal setup offers Codex OAuth, LiteLLM, OpenRouter, and
  **Codex OAuth + local RAG embeddings**. LiteLLM and OpenRouter setup also ask
  whether to configure RAG.
- The protected browser configuration lists Codex, Claude Code, LiteLLM, and
  OpenRouter in the provider connections and priority controls. A provider only
  works when its required CLI or endpoint is reachable from the runtime.
- Claude Code is integrated in the provider router and browser configuration,
  but it does not yet have a dedicated one-click choice in the terminal setup
  wizard. It can be enabled in `config.yaml`.

Use `oneday doctor` for narrative and embedding readiness. Use
`oneday rag benchmark` for a real embedding request and dimension check.

## Can I change providers later?

Yes. Use the protected operator configuration or run
`oneday setup --reconfigure`. Provider priority is an ordered fallback chain:
disabled providers are skipped and enabled providers are tried in order.

If you change the embedding model or vector dimensions, run
`oneday rag reindex --all` so existing long-term memory uses the new vector
shape. Changing the narrator alone does not require a RAG reindex.
