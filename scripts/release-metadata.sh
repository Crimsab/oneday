#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <vX.Y.Z[-prerelease]> [output.json]" >&2
  exit 2
}

[[ $# -ge 1 && $# -le 2 ]] || usage

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
tag="$1"
output="${2:-}"

if [[ ! "${tag}" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
  echo "invalid release tag: ${tag}" >&2
  exit 1
fi
version="${tag#v}"

protocol_version="$(
  awk '/^const Version = [0-9]+$/ { print $4; exit }' \
    "${repo_root}/internal/game/gatewayprotocol/types.go"
)"
database_version="$(
  sed -nE 's/^[[:space:]]*\{([0-9]+), migrationV([0-9]+)\},$/\1 \2/p' \
    "${repo_root}/internal/storage/migrations.go" | tail -n 1
)"

if [[ ! "${protocol_version}" =~ ^[0-9]+$ ]]; then
  echo "could not resolve the gateway protocol version" >&2
  exit 1
fi
read -r database_schema_version database_symbol_version <<<"${database_version}"
if [[ ! "${database_schema_version:-}" =~ ^[0-9]+$ ]] || \
   [[ "${database_schema_version}" != "${database_symbol_version:-}" ]]; then
  echo "could not resolve a consistent database schema version" >&2
  exit 1
fi

source_commit="$(git -C "${repo_root}" rev-parse HEAD)"
source_date_epoch="$(git -C "${repo_root}" show -s --format=%ct HEAD)"
source_date="$(
  TZ=UTC git -C "${repo_root}" show -s \
    --format=%cd \
    --date=format-local:'%Y-%m-%dT%H:%M:%SZ' \
    HEAD
)"

payload="$(jq -n \
  --arg tag "${tag}" \
  --arg version "${version}" \
  --arg commit "${source_commit}" \
  --arg source_date "${source_date}" \
  --argjson source_date_epoch "${source_date_epoch}" \
  --argjson protocol_version "${protocol_version}" \
  --argjson database_schema_version "${database_schema_version}" \
  '{
    metadataVersion: 1,
    releaseTag: $tag,
    applicationVersion: $version,
    desktopVersion: $version,
    gatewayProtocolVersion: $protocol_version,
    databaseSchemaVersion: $database_schema_version,
    sourceCommit: $commit,
    sourceDate: $source_date,
    sourceDateEpoch: $source_date_epoch
  }')"

if [[ -n "${output}" ]]; then
  mkdir -p "$(dirname "${output}")"
  printf '%s\n' "${payload}" > "${output}"
else
  printf '%s\n' "${payload}"
fi
