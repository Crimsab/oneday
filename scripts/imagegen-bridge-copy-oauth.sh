#!/usr/bin/env bash
# Copy exactly one locally authenticated Codex OAuth file into the dedicated
# Compose volume. This script never prints or stores credential contents.
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_env_file="${1:-$root_dir/.env.imagegen-bridge}"
oauth_source="${CODEX_AUTH_FILE:-${CODEX_HOME:-$HOME/.codex}/auth.json}"

if [[ ! -f "$compose_env_file" ]]; then
  printf 'Missing environment file: %s\nCopy compose.imagegen-bridge.env.example first.\n' "$compose_env_file" >&2
  exit 1
fi

if [[ ! -s "$oauth_source" ]]; then
  printf 'Missing Codex OAuth file: %s\nRun codex login on this host, or set CODEX_AUTH_FILE.\n' "$oauth_source" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  printf 'Docker Compose v2 is required.\n' >&2
  exit 1
fi

project_name="$(
  cd "$root_dir"
  docker compose --env-file "$compose_env_file" \
    -f compose.yaml -f compose.imagegen-bridge.yaml \
    --profile imagegen-bridge config --format json \
    | sed -n 's/.*"name":"\([^"]*\)".*/\1/p' \
    | head -n 1
)"

if [[ -z "$project_name" ]]; then
  project_name="oneday"
fi

volume_name="${project_name}_imagegen_bridge_codex_oauth"
docker volume create "$volume_name" >/dev/null
docker run --rm \
  -v "$volume_name:/codex-home" \
  -v "$oauth_source:/source/auth.json:ro" \
  alpine:3.23 \
  sh -ceu 'cp /source/auth.json /codex-home/auth.json && chmod 0600 /codex-home/auth.json && chown 10001:10001 /codex-home/auth.json'

printf 'Copied Codex OAuth state into Docker volume %s. Start the opt-in profile next.\n' "$volume_name"
