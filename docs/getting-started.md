# Getting started

OneDay can run as a native terminal application, a browser application, or a
desktop application. Terminal and browser clients can use the same configured
engine/database. Desktop makes you choose between a remote server and a new,
isolated standalone local profile—there is no automatic sync between them.

## Choose a runtime

| Runtime | Best for | Requirements |
| --- | --- | --- |
| Terminal client | Local play, Codex OAuth, Claude Code, development | Go 1.25.12+ or a release binary |
| Browser with Docker | Self-hosting and the complete React interface | Docker Engine with Compose v2 |
| Desktop, remote profile | A native window for an existing server | Reachable HTTPS gateway (HTTP loopback only for development) |
| Desktop, standalone profile | A local packaged experience | A desktop build that includes version-matched engine/gateway/web sidecars |

At least one narrative provider must be configured. OneDay supports Codex CLI,
Claude Code, LiteLLM-compatible endpoints, and OpenRouter. RAG embeddings are
optional and can be remote or local through Ollama/custom HTTP.

For the shortest walkthrough, including creation of the first world, see
[Your first story](first-story.md).

### Decide where a story lives first

Pick one canonical store before creating a story. A terminal configuration or
Docker volume owns its own SQLite data. Desktop **remote** mode opens that
server and stores no local story database. Desktop **standalone** mode creates
a different local profile and data directory. Neither profile copies, merges,
or synchronizes stories with the other. Use a supported export/import transfer
and a backup when you mean to move data.

## Terminal client

Use Go 1.25.12 or newer. The repository's `toolchain` directive records the
minimum security-patched toolchain used by CI and release builds.

Clone the repository and enter it:

```bash
git clone https://github.com/Crimsab/oneday.git
cd oneday
```

Run the interactive setup, verify it, and start the game:

```bash
go run ./cmd/oneday setup
go run ./cmd/oneday doctor
go run ./cmd/oneday
```

The setup wizard creates `config.yaml` with mode `0600`. If it already exists,
rerun the wizard with `setup --reconfigure`. Provider keys belong in `.env` or
the shell, not in tracked YAML.

### Provider examples

Codex uses the local CLI login:

```bash
codex login
go run ./cmd/oneday setup --reconfigure
```

OpenRouter uses an environment key:

```bash
cp .env.example .env
# Set ONEDAY_OPENROUTER_API_KEY in .env, then select OpenRouter in setup.
go run ./cmd/oneday setup --reconfigure
```

For a local or hosted LiteLLM-compatible endpoint, set its URL/model in
`config.yaml` and provide `ONEDAY_LITELLM_API_KEY`.

## Browser with Docker

Prepare local configuration before starting Compose:

```bash
cp config.example.yaml config.yaml
cp .env.example .env
```

Edit `config.yaml` so at least one provider is enabled and set its key in `.env`.
Then pull and start the complete Go + Rust + React image:

```bash
docker compose pull
docker compose up -d
curl -fsS http://localhost:8788/api/health
```

Open `http://localhost:8788`. The first start creates and migrates the SQLite
database automatically in the `oneday_data` named volume.

### Review setup from the browser

Open **Setup** in the browser chrome, or navigate directly to `/setup` on the
same OneDay origin (for example, `https://oneday.example.com/setup`). This
screen shows the gateway's canonical readiness checks and links to the shared
configuration surface; it does not store credentials in the browser. Native
terminal setup remains `oneday setup`; rerun it with `oneday setup
--reconfigure` when the local CLI configuration needs to change.

Generated images and speech remain disabled until their providers are
configured. They are optional and never block canonical text turns.

Docker does not bundle your host Codex or Claude CLI credentials. Use
LiteLLM/OpenRouter in the standard container setup, or create a private Compose
override that mounts the relevant CLI binary and its authentication files.

If a provider or image bridge runs on the Docker host, use
`host.docker.internal` rather than `127.0.0.1` from `config.yaml`; the supplied
Compose file maps that hostname on Linux as well as Docker Desktop.

### Browser access and first login

The gateway is loopback-first. An interactive local gateway can emit a one-shot
bootstrap URL that establishes a browser session; it is secret material, not a
shareable URL. When the gateway runs non-interactively or beyond loopback,
configure the bootstrap credential and a trusted reverse-proxy origin before
opening it remotely. That configured credential can be entered again when the
web interface asks to reconnect; the browser does not store it. A direct bearer
token is a separate API/desktop-launch credential, not a value to put in a
browser URL. See [Configuration](configuration.md#gateway-authentication-and-reverse-proxies).

## Desktop profiles

On first launch, choose **Connect to a server** for remote mode or **Run on this
device** for standalone mode. Remote mode requires a root server origin such as
`https://oneday.example.com`; it rejects embedded credentials, query strings,
path prefixes, and ordinary HTTP. Standalone mode runs a fresh loopback gateway
for that desktop profile and opens its bundled web UI.

Before relying on standalone mode, confirm the desktop package actually
contains its matching gateway, engine, and web UI. Optional narrative/media
providers are not bundled merely because the sidecars are present. For profile
locations, backups, and shutdown details, read [Desktop](desktop.md).

To build the current checkout instead of pulling a release image, use:

```bash
docker compose -f compose.yaml -f compose.build.yaml up -d --build
```

## Verify the installation

Useful non-destructive checks:

```bash
go run ./cmd/oneday doctor --json
go run ./cmd/oneday config show --safe
go run ./cmd/oneday rag benchmark
```

For Docker, also check:

```bash
docker compose ps
docker compose logs --tail 100 oneday-gateway
```

Continue with [Configuration](configuration.md) for all provider settings,
[Generated media](media.md) for image and speech setup, or
[Troubleshooting](troubleshooting.md) if a check fails.
