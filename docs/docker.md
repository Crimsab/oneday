# Docker deployment

The Docker image contains the Go engine, Rust gateway, and compiled React UI.
The public `compose.yaml` is portable and has no host-specific paths or networks.

## Start from the current source

This path works before a container release exists. Docker builds the Go engine,
Rust gateway, and React client. The host does not need these toolchains.

Build the image:

```bash
docker compose -f compose.yaml -f compose.build.yaml build oneday-gateway
```

Initialize and start OneDay:

```bash
docker compose -f compose.yaml -f compose.build.yaml run --rm oneday-tools docker init
docker compose -f compose.yaml -f compose.build.yaml up -d
```

The initializer preserves existing files. It creates private `config.yaml` and
`.env` files when they are missing. It also creates a high-entropy browser
credential. It does not print the credential. Show it only when the browser
asks you to reconnect:

```bash
docker compose -f compose.yaml -f compose.build.yaml run --rm oneday-tools docker token
```

These commands work in PowerShell, macOS, and Linux. Windows does not need WSL
or Git Bash.

## Start from a published image

Use this shorter path only when the
[container page](https://github.com/Crimsab/oneday/pkgs/container/oneday)
contains the selected tag. `latest` means the newest stable container release.
For a reproducible installation, set `ONEDAY_IMAGE` to a versioned tag.

```bash
docker compose pull
docker compose run --rm oneday-tools docker init
docker compose up -d
docker compose run --rm oneday-tools docker token
```

The service listens on `${ONEDAY_PORT:-8788}` and exposes `/api/health`.
Compose persists application data in the `oneday_data` named volume and mounts
`config.yaml` read/write so model settings saved in the browser reach the Go engine.
Automatic images and TTS are off in the public template until their providers
are configured; neither is required for story generation. After the first login,
open **Setup** and configure at least one narrative provider in the protected
operator workspace.

For automation that must only prepare files, run the applicable `docker init`
command without the subsequent `up` command.

## Provider networking

`127.0.0.1` inside the container is the container itself. For a LiteLLM, Ollama,
or image bridge running on the host, use URLs such as:

```yaml
ai:
  litellm:
    base_url: "http://host.docker.internal:4000/v1"
  embedding:
    local:
      base_url: "http://host.docker.internal:11434"
  image_generation:
    imagegen_bridge_url: "http://host.docker.internal:8787"
    openclaw_bridge_url: "http://host.docker.internal:8099/generate"
```

The Compose file supplies the Linux `host-gateway` mapping automatically.

## Optional Codex OAuth imagegen-bridge profile

The default Docker stack does **not** start imagegen-bridge, copy a Codex
login, or enable automatic images. If you deliberately want Codex OAuth image
generation inside the same private Compose network, enable the separately
versioned profile. It uses the released
[`ghcr.io/crimsab/imagegen-bridge:0.3.0`](https://github.com/Crimsab/imagegen-bridge/pkgs/container/imagegen-bridge)
image pinned to its multi-architecture OCI index digest, and keeps the bridge's
Codex OAuth state, session/job state, and generated artifacts in three separate
named volumes.

First create an untracked environment file with a distinct random bridge
bearer. The bearer protects the bridge HTTP API; it is not a Codex OAuth token.

```bash
cp compose.imagegen-bridge.env.example .env.imagegen-bridge
chmod 600 .env.imagegen-bridge
# Set IMAGEGEN_BRIDGE_BEARER_TOKEN to the output of: openssl rand -hex 32
$EDITOR .env.imagegen-bridge
```

Authenticate Codex on the host, then copy only its `auth.json` into the
dedicated named volume. The helper does not print the credential and never
mounts an entire home directory into a container. Set `CODEX_AUTH_FILE` when
the source file is not under `${CODEX_HOME:-$HOME/.codex}`.

```bash
codex login
./scripts/imagegen-bridge-copy-oauth.sh
```

Start the profile. Its bridge endpoint is available to OneDay at
`http://imagegen-bridge:8787` only on the private Compose network. The optional
host dashboard/API mapping is fixed to `127.0.0.1`, so it cannot accept LAN
connections.

```bash
docker compose --env-file .env.imagegen-bridge \
  -f compose.yaml -f compose.imagegen-bridge.yaml \
  --profile imagegen-bridge up -d
docker compose --env-file .env.imagegen-bridge \
  -f compose.yaml -f compose.imagegen-bridge.yaml \
  --profile imagegen-bridge ps
curl --fail http://127.0.0.1:8787/health/live
```

OneDay receives the bearer only as `ONEDAY_IMAGEGEN_BRIDGE_TOKEN`; it does not
receive the Codex OAuth file. The profile selects the `codex-responses` route
and leaves automatic visual generation controlled by the existing OneDay
configuration. OAuth use remains an explicit operator opt-in.

Do not change the published port to `0.0.0.0` or expose it through an untrusted
network. A remote bridge must stay on a trusted private network (or behind a
trusted TLS reverse proxy) and must retain a unique high-entropy bearer. Do not
send a bearer or OAuth state over public cleartext HTTP.

### Profile lifecycle

The profile's image tag is intentionally fixed rather than `latest`. Before
upgrading, read the upstream release notes, back up the bridge state if needed,
then update `IMAGEGEN_BRIDGE_IMAGE` in `.env.imagegen-bridge` to an explicitly
chosen released tag and recreate only the bridge:

```bash
docker compose --env-file .env.imagegen-bridge \
  -f compose.yaml -f compose.imagegen-bridge.yaml \
  --profile imagegen-bridge pull imagegen-bridge
docker compose --env-file .env.imagegen-bridge \
  -f compose.yaml -f compose.imagegen-bridge.yaml \
  --profile imagegen-bridge up -d imagegen-bridge
```

If you renew the host login with `codex login`, run
`./scripts/imagegen-bridge-copy-oauth.sh` again before restarting the bridge.
The dedicated OAuth volume is secret material and may rotate while the bridge
runs; treat encrypted backups accordingly. `docker compose down` preserves all
three bridge volumes. Adding `--volumes` permanently deletes the dedicated
OAuth, bridge-state, and artifact volumes as well as OneDay data.

Validate this optional deployment without starting it:

```bash
./scripts/check-imagegen-bridge-compose.sh
```

### Image alternatives

Image generation is optional; story text remains fully usable when no image
provider is configured. Choose one of these paths instead of the bundled
profile when it better fits the deployment:

- Use a hosted provider (OpenAI Platform, Gemini, fal.ai, Replicate, Stability,
  or Azure OpenAI) through OneDay's direct provider configuration. Their API
  keys do not pass through imagegen-bridge.
- Point `imagegen_bridge_url` and `imagegen_bridge_token` at a separately
  operated bridge on a trusted private network. Keep its bearer distinct from
  Codex OAuth and use TLS when traffic leaves the local host.
- Use a local OpenAI-compatible image endpoint, including a Z-Image-class
  deployment, only when it implements the documented image-generation request
  and response contract. Configure it as `openai-compatible` with an explicit
  endpoint and capability probe; chat-completions compatibility alone is not
  sufficient.
- Use text-only mode: do not enable this profile, leave image auto-generation
  disabled, and omit image provider credentials. Failed or absent media never
  blocks canonical story text or state.

## Rebuild the current source

After a source update, rebuild and recreate the gateway:

```bash
docker compose -f compose.yaml -f compose.build.yaml up -d --build oneday-gateway
```

This produces `oneday-gateway:local`. It does not change the release defaults
in `.env` or `compose.yaml`.

## Update

```bash
git pull --ff-only
docker compose pull oneday-gateway
docker compose up -d oneday-gateway
curl -fsS http://localhost:${ONEDAY_PORT:-8788}/api/health
```

Review the [changelog](../CHANGELOG.md) before updating persistent installations.

## Backup and restore

Stop writes before copying the named volume:

```bash
docker compose stop oneday-gateway
docker run --rm \
  -v oneday_oneday_data:/data:ro \
  -v "$PWD:/backup" \
  alpine:3.23 tar -C /data -czf /backup/oneday-data.tar.gz .
docker compose start oneday-gateway
```

The exact volume prefix is the Compose project name. Confirm it with
`docker volume ls --filter name=oneday_data` before backing up or restoring.

## Operations

```bash
docker compose ps
docker compose logs --tail 100 oneday-gateway
docker compose restart oneday-gateway
docker compose down             # keeps the data volume
docker compose down --volumes   # deletes persistent story data
```

Do not use `down --volumes` unless the data is disposable or backed up.
