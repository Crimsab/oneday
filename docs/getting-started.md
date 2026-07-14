# Getting started

OneDay can run as a native terminal application or as a browser application. Both
surfaces use the same Go story engine and SQLite data.

## Choose a runtime

| Runtime | Best for | Requirements |
| --- | --- | --- |
| Terminal client | Local play, Codex OAuth, Claude Code, development | Go 1.25+ or a release binary |
| Browser with Docker | Self-hosting and the complete React interface | Docker Engine with Compose v2 |

At least one narrative provider must be configured. OneDay supports Codex CLI,
Claude Code, LiteLLM-compatible endpoints, and OpenRouter. RAG embeddings are
optional and can be remote or local through Ollama/custom HTTP.

## Terminal client

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

Docker does not bundle your host Codex or Claude CLI credentials. Use
LiteLLM/OpenRouter in the standard container setup, or create a private Compose
override that mounts the relevant CLI binary and its authentication files.

If a provider or image bridge runs on the Docker host, use
`host.docker.internal` rather than `127.0.0.1` from `config.yaml`; the supplied
Compose file maps that hostname on Linux as well as Docker Desktop.

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

Continue with [Configuration](configuration.md) for all provider and media
settings, or [Troubleshooting](troubleshooting.md) if a check fails.
