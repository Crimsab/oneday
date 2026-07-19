#!/usr/bin/env bash
# Validate the optional imagegen-bridge Compose profile without starting it.
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_base="$root_dir/compose.yaml"
compose_bridge="$root_dir/compose.imagegen-bridge.yaml"
env_example="$root_dir/compose.imagegen-bridge.env.example"

for required_file in "$compose_base" "$compose_bridge" "$env_example"; do
  if [[ ! -f "$required_file" ]]; then
    printf 'Missing required Compose integration file: %s\n' "$required_file" >&2
    exit 1
  fi
done

if ! rg --quiet '^    profiles: \[imagegen-bridge\]$' "$compose_bridge" \
  || ! rg --quiet 'ghcr\.io/crimsab/imagegen-bridge:0\.3\.0@sha256:8ca87e645c03415bd2dd6c0bdcf2f43db361198e676179fe0e92c8de9ee7b267' "$compose_bridge" \
  || ! rg --quiet '127\.0\.0\.1:\$\{IMAGEGEN_BRIDGE_PORT:-8787\}:8787' "$compose_bridge" \
  || ! rg --quiet 'imagegen_bridge_codex_oauth:/codex-home' "$compose_bridge"; then
  printf 'The imagegen-bridge profile must remain opt-in, release-pinned, loopback-bound, and OAuth-isolated.\n' >&2
  exit 1
fi

if [[ "${1:-}" == "--static" ]]; then
  printf 'Static imagegen-bridge Compose checks passed.\n'
  exit 0
fi

if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  printf 'Docker Compose v2 is required for full validation; static checks passed.\n' >&2
  exit 1
fi

cd "$root_dir"
IMAGEGEN_BRIDGE_BEARER_TOKEN=compose-validation-only \
IMAGEGEN_BRIDGE_IMAGE=ghcr.io/crimsab/imagegen-bridge:0.3.0@sha256:8ca87e645c03415bd2dd6c0bdcf2f43db361198e676179fe0e92c8de9ee7b267 \
docker compose \
  -f compose.yaml -f compose.imagegen-bridge.yaml \
  --profile imagegen-bridge config --quiet

printf 'Imagegen-bridge Compose profile validation passed.\n'
