#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <release-config.json> <output.json>" >&2
  exit 2
fi

input="$1"
output="$2"
endpoint="${ONEDAY_UPDATER_ENDPOINT:-}"
pubkey="${ONEDAY_UPDATER_PUBKEY:-}"

[[ -f "${input}" ]] || { echo "missing Tauri release config: ${input}" >&2; exit 1; }
[[ "${endpoint}" == https://* ]] || { echo "the updater endpoint must use HTTPS" >&2; exit 1; }
[[ "${pubkey}" =~ ^[A-Za-z0-9+/=]+$ && ${#pubkey} -ge 40 ]] || {
  echo "the updater public key must be the canonical base64-encoded Minisign public key" >&2
  exit 1
}

output_dir="$(dirname "${output}")"
mkdir -p "${output_dir}"
temporary="$(mktemp "${output_dir}/tauri-updater.XXXXXX.json")"
trap 'rm -f "${temporary}"' EXIT

jq \
  --arg endpoint "${endpoint}" \
  --arg pubkey "${pubkey}" '
    .plugins.updater = ((.plugins.updater // {}) + {
      endpoints: [$endpoint],
      pubkey: $pubkey
    })
  ' "${input}" > "${temporary}"

jq -e \
  --arg endpoint "${endpoint}" \
  --arg pubkey "${pubkey}" '
    .bundle.createUpdaterArtifacts == true and
    .plugins.updater.endpoints == [$endpoint] and
    .plugins.updater.pubkey == $pubkey
  ' "${temporary}" >/dev/null

mv "${temporary}" "${output}"
