# Docker deployment

The Docker image contains the Go engine, Rust gateway, and compiled React UI.
The public `compose.yaml` is portable and has no host-specific paths or networks.

## Start

```bash
cp config.example.yaml config.yaml
cp .env.example .env
$EDITOR config.yaml
$EDITOR .env
docker compose up -d --build
```

The service listens on `${ONEDAY_PORT:-8788}` and exposes `/api/health`.
Compose persists application data in the `oneday_data` named volume and mounts
`config.yaml` read/write so model settings saved in the browser reach the Go engine.

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
    openclaw_bridge_url: "http://host.docker.internal:8099/generate"
```

The Compose file supplies the Linux `host-gateway` mapping automatically.

## Update

```bash
git pull --ff-only
docker compose build --pull oneday-gateway
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
