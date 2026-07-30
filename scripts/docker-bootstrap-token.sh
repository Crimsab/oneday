#!/bin/sh
set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
env_file="$root_dir/.env"

if [ ! -f "$env_file" ]; then
  printf 'No private .env exists. Run ./scripts/docker-init.sh first.\n' >&2
  exit 1
fi

token=$(sed -n 's/^ONEDAY_GATEWAY_BOOTSTRAP_TOKEN=//p' "$env_file" | tail -n 1)
if [ -z "$token" ]; then
  printf 'No bootstrap token is configured. Run ./scripts/docker-init.sh --prepare-only.\n' >&2
  exit 1
fi

printf '%s\n' "$token"
