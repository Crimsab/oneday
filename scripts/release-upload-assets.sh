#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <vX.Y.Z[-prerelease]> <owner/repo> <asset-directory>" >&2
  exit 2
}

[[ $# -eq 3 ]] || usage
tag="$1"
repository="$2"
asset_dir="$3"

if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
  echo "invalid release tag: ${tag}" >&2
  exit 1
fi
if [[ ! "${repository}" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
  echo "invalid GitHub repository: ${repository}" >&2
  exit 1
fi
[[ -d "${asset_dir}" ]] || { echo "missing release asset directory: ${asset_dir}" >&2; exit 1; }
command -v gh >/dev/null 2>&1 || { echo "gh is required to publish release assets" >&2; exit 1; }

mapfile -d '' local_assets < <(
  find "${asset_dir}" -maxdepth 1 -type f -printf '%p\0' | LC_ALL=C sort -z
)
if [[ "${#local_assets[@]}" -eq 0 ]]; then
  echo "release asset directory is empty: ${asset_dir}" >&2
  exit 1
fi
download_dir="$(mktemp -d)"
cleanup() {
  rm -r -- "${download_dir}"
}
trap cleanup EXIT
remote_asset_list="${download_dir}/remote-assets.txt"
gh release view "${tag}" \
  --repo "${repository}" \
  --json assets \
  --jq '.assets[].name' > "${remote_asset_list}"
mapfile -t remote_assets < "${remote_asset_list}"

declare -a missing_assets=()
for local_asset in "${local_assets[@]}"; do
  name="$(basename "${local_asset}")"
  exists=false
  for remote_asset in "${remote_assets[@]}"; do
    if [[ "${remote_asset}" == "${name}" ]]; then
      exists=true
      break
    fi
  done

  if [[ "${exists}" == "false" ]]; then
    missing_assets+=("${local_asset}")
    continue
  fi

  asset_download_dir="${download_dir}/${name}"
  mkdir -p "${asset_download_dir}"
  gh release download "${tag}" \
    --repo "${repository}" \
    --pattern "${name}" \
    --dir "${asset_download_dir}"
  downloaded_asset="${asset_download_dir}/${name}"
  if [[ ! -f "${downloaded_asset}" ]] || ! cmp -s "${local_asset}" "${downloaded_asset}"; then
    echo "release asset conflict: ${name} already exists with different bytes" >&2
    exit 1
  fi
  echo "release asset already exists with identical bytes; skipping: ${name}"
done

if [[ "${#missing_assets[@]}" -eq 0 ]]; then
  echo "all release assets already exist with identical bytes"
  exit 0
fi

# This upload is intentionally additive-only. A concurrent conflicting upload
# fails at GitHub because no replacement flag is supplied.
gh release upload "${tag}" "${missing_assets[@]}" --repo "${repository}"
