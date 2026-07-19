#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <vX.Y.Z[-prerelease]> <target-triple> <oneday-bin> <gateway-bin> <web-dist>" >&2
  exit 2
}

[[ $# -eq 5 ]] || usage

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
tag="$1"
target="$2"
engine_bin="$3"
gateway_bin="$4"
web_dist="$5"
version="${tag#v}"

"${script_dir}/release-metadata.sh" "${tag}" >/dev/null
[[ -f "${engine_bin}" ]] || { echo "missing OneDay sidecar: ${engine_bin}" >&2; exit 1; }
[[ -f "${gateway_bin}" ]] || { echo "missing gateway sidecar: ${gateway_bin}" >&2; exit 1; }
[[ -f "${web_dist}/index.html" ]] || { echo "missing built web client: ${web_dist}" >&2; exit 1; }
if [[ ! "${target}" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]*$ ]]; then
  echo "invalid Rust target triple: ${target}" >&2
  exit 1
fi

current_package_version="$(jq -er '.version' "${repo_root}/desktop/package.json")"
current_cargo_version="$(
  awk '
    /^\[package\]$/ { package = 1; next }
    /^\[/ { package = 0 }
    package && /^version = "/ { gsub(/^version = "|"$/, ""); print; exit }
  ' "${repo_root}/desktop/src-tauri/Cargo.toml"
)"
if [[ "${current_package_version}" != "${current_cargo_version}" ]]; then
  echo "desktop package versions are not coordinated" >&2
  exit 1
fi

package_tmp="$(mktemp)"
jq --arg version "${version}" '.version = $version' \
  "${repo_root}/desktop/package.json" > "${package_tmp}"
mv "${package_tmp}" "${repo_root}/desktop/package.json"

rewrite_toml_version() {
  local input="$1"
  local package_name="$2"
  local output
  output="$(mktemp)"
  awk -v wanted_name="${package_name}" -v wanted_version="${version}" '
    /^\[\[package\]\]$/ { in_lock_package = 1; name_matches = 0; print; next }
    /^\[package\]$/ { in_root_package = 1; print; next }
    /^\[/ && $0 != "[package]" { in_root_package = 0 }
    in_lock_package && /^name = / {
      name_matches = ($0 == "name = \"" wanted_name "\"")
    }
    in_root_package && /^version = / {
      print "version = \"" wanted_version "\""
      in_root_package = 0
      next
    }
    in_lock_package && name_matches && /^version = / {
      print "version = \"" wanted_version "\""
      name_matches = 0
      next
    }
    { print }
  ' "${input}" > "${output}"
  mv "${output}" "${input}"
}

rewrite_toml_version "${repo_root}/desktop/src-tauri/Cargo.toml" "oneday-desktop"
rewrite_toml_version "${repo_root}/desktop/src-tauri/Cargo.lock" "oneday-desktop"

config_tmp="$(mktemp)"
sidecar_extension=""
if [[ "${target}" == *-windows-* ]]; then
  sidecar_extension=".exe"
fi
jq --arg version "${version}" --arg target "${target}" --arg extension "${sidecar_extension}" '
  .bundle.externalBin = [
  ] |
  .bundle.resources += [
    "binaries/oneday-v\($version)-\($target)\($extension)",
    "binaries/oneday-gateway-v\($version)-\($target)\($extension)"
  ]
' "${repo_root}/desktop/src-tauri/tauri.conf.json" > "${config_tmp}"
mv "${config_tmp}" "${repo_root}/desktop/src-tauri/tauri.conf.json"

sidecar_dir="${repo_root}/desktop/src-tauri/binaries"
resource_dir="${repo_root}/desktop/gateway/web/dist"
mkdir -p "${sidecar_dir}" "${resource_dir}"
cp "${engine_bin}" "${sidecar_dir}/oneday-v${version}-${target}${sidecar_extension}"
cp "${gateway_bin}" "${sidecar_dir}/oneday-gateway-v${version}-${target}${sidecar_extension}"
cp -R "${web_dist}/." "${resource_dir}/"

cargo metadata --locked --no-deps \
  --manifest-path "${repo_root}/desktop/src-tauri/Cargo.toml" \
  --format-version 1 >/dev/null

prepared_package_version="$(jq -er '.version' "${repo_root}/desktop/package.json")"
prepared_cargo_version="$(
  cargo metadata --locked --no-deps \
    --manifest-path "${repo_root}/desktop/src-tauri/Cargo.toml" \
    --format-version 1 \
    | jq -er '.packages[] | select(.name == "oneday-desktop") | .version'
)"
if [[ "${prepared_package_version}" != "${version}" || "${prepared_cargo_version}" != "${version}" ]]; then
  echo "desktop release version synchronization failed" >&2
  exit 1
fi
