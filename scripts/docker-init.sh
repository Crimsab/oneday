#!/bin/sh
set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
env_file="$root_dir/.env"
config_file="$root_dir/config.yaml"
prepare_only=false

usage() {
  printf '%s\n' \
    "Usage: ./scripts/docker-init.sh [--prepare-only]" \
    "" \
    "Creates private local configuration, generates the browser bootstrap token," \
    "and starts OneDay with Docker Compose. Existing settings are preserved."
}

case "${1:-}" in
  "")
    ;;
  --prepare-only)
    prepare_only=true
    ;;
  -h|--help)
    usage
    exit 0
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

umask 077

if [ ! -f "$config_file" ]; then
  cp "$root_dir/config.example.yaml" "$config_file"
  chmod 600 "$config_file"
  printf 'Created private config.yaml from the public template.\n'
fi

if [ ! -f "$env_file" ]; then
  cp "$root_dir/.env.example" "$env_file"
  chmod 600 "$env_file"
  printf 'Created private .env from the public template.\n'
fi

read_env_value() {
  key=$1
  sed -n "s/^${key}=//p" "$env_file" | tail -n 1
}

replace_env_value() {
  key=$1
  value=$2
  temp_file=$(mktemp "$root_dir/.env.oneday.XXXXXX")
  awk -v key="$key" -v value="$value" '
    BEGIN { found = 0 }
    index($0, key "=") == 1 {
      if (!found) {
        print key "=" value
        found = 1
      }
      next
    }
    { print }
    END {
      if (!found) print key "=" value
    }
  ' "$env_file" >"$temp_file"
  chmod 600 "$temp_file"
  mv "$temp_file" "$env_file"
}

generate_token() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
    return
  fi
  if command -v od >/dev/null 2>&1; then
    od -An -N32 -tx1 /dev/urandom | tr -d ' \n'
    return
  fi
  printf 'OneDay needs openssl or od to generate a secure bootstrap token.\n' >&2
  exit 1
}

bootstrap_token=$(read_env_value ONEDAY_GATEWAY_BOOTSTRAP_TOKEN)
if [ -z "$bootstrap_token" ]; then
  replace_env_value ONEDAY_GATEWAY_BOOTSTRAP_TOKEN "$(generate_token)"
  printf 'Generated a private reusable browser bootstrap token.\n'
fi

port=$(read_env_value ONEDAY_PORT)
port=${port:-8788}
allowed_hosts=$(read_env_value ONEDAY_GATEWAY_ALLOWED_HOSTS)
default_hosts="localhost:${port},127.0.0.1:${port},oneday-gateway:8788"
if [ -z "$allowed_hosts" ] || [ "$allowed_hosts" = "localhost:8788,127.0.0.1:8788,oneday-gateway:8788" ]; then
  replace_env_value ONEDAY_GATEWAY_ALLOWED_HOSTS "$default_hosts"
fi

if [ "$prepare_only" = true ]; then
  printf '%s\n' \
    "OneDay Docker configuration is ready." \
    "Retrieve the browser credential only when needed with:" \
    "  ./scripts/docker-bootstrap-token.sh"
  exit 0
fi

if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  printf 'Docker Engine with Compose v2 is required.\n' >&2
  exit 1
fi

cd "$root_dir"
docker compose pull
docker compose up -d

printf '%s\n' \
  "" \
  "OneDay is starting at http://localhost:${port}" \
  "Retrieve the browser credential only when needed with:" \
  "  ./scripts/docker-bootstrap-token.sh" \
  "" \
  "After signing in, open Setup to configure a narrative provider."
