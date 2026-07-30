#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture_root=$(mktemp -d)
trap 'rm -rf "$fixture_root"' EXIT INT TERM

mkdir -p "$fixture_root/scripts"
cp "$repo_root/config.example.yaml" "$fixture_root/config.example.yaml"
cp "$repo_root/.env.example" "$fixture_root/.env.example"
cp "$repo_root/scripts/docker-init.sh" "$fixture_root/scripts/docker-init.sh"
cp "$repo_root/scripts/docker-bootstrap-token.sh" "$fixture_root/scripts/docker-bootstrap-token.sh"
chmod +x "$fixture_root/scripts/docker-init.sh" "$fixture_root/scripts/docker-bootstrap-token.sh"

"$fixture_root/scripts/docker-init.sh" --prepare-only >/dev/null

test -f "$fixture_root/config.yaml"
test -f "$fixture_root/.env"
test "$(stat -c '%a' "$fixture_root/config.yaml")" = "600"
test "$(stat -c '%a' "$fixture_root/.env")" = "600"

first_token=$("$fixture_root/scripts/docker-bootstrap-token.sh")
test "${#first_token}" -ge 64

"$fixture_root/scripts/docker-init.sh" --prepare-only >/dev/null
second_token=$("$fixture_root/scripts/docker-bootstrap-token.sh")
test "$first_token" = "$second_token"

sed -i 's/^ONEDAY_PORT=.*/ONEDAY_PORT=9988/' "$fixture_root/.env"
sed -i 's/^ONEDAY_GATEWAY_ALLOWED_HOSTS=.*/ONEDAY_GATEWAY_ALLOWED_HOSTS=/' "$fixture_root/.env"
"$fixture_root/scripts/docker-init.sh" --prepare-only >/dev/null
grep -q '^ONEDAY_GATEWAY_ALLOWED_HOSTS=localhost:9988,127.0.0.1:9988,oneday-gateway:8788$' "$fixture_root/.env"

printf 'Docker initialization tests passed\n'
