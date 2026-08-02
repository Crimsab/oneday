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
bridge_manifest="${repo_root}/desktop/src-tauri/imagegen-bridge-components.json"

"${script_dir}/release-metadata.sh" "${tag}" >/dev/null
[[ -f "${engine_bin}" ]] || { echo "missing OneDay sidecar: ${engine_bin}" >&2; exit 1; }
[[ -f "${gateway_bin}" ]] || { echo "missing gateway sidecar: ${gateway_bin}" >&2; exit 1; }
[[ -f "${web_dist}/index.html" ]] || { echo "missing built web client: ${web_dist}" >&2; exit 1; }
[[ -f "${bridge_manifest}" ]] || { echo "missing imagegen bridge component manifest" >&2; exit 1; }
if [[ ! "${target}" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]*$ ]]; then
  echo "invalid Rust target triple: ${target}" >&2
  exit 1
fi

bridge_release_tag="$(jq -er '.releaseTag' "${bridge_manifest}")"
bridge_url="$(jq -er --arg target "${target}" '.targets[$target].url' "${bridge_manifest}")"
bridge_sha256="$(jq -er --arg target "${target}" '.targets[$target].sha256' "${bridge_manifest}")"
bridge_size="$(jq -er --arg target "${target}" '.targets[$target].size' "${bridge_manifest}")"
bridge_archive="$(jq -er --arg target "${target}" '.targets[$target].archive' "${bridge_manifest}")"
bridge_entry="$(jq -er --arg target "${target}" '.targets[$target].entry' "${bridge_manifest}")"
expected_bridge_prefix="https://github.com/Crimsab/imagegen-bridge/releases/download/${bridge_release_tag}/"
if [[ "${bridge_url}" != "${expected_bridge_prefix}"* ]] ||
  [[ ! "${bridge_sha256}" =~ ^[[:xdigit:]]{64}$ ]] ||
  [[ ! "${bridge_size}" =~ ^[1-9][0-9]*$ ]] ||
  [[ ! "${bridge_archive}" =~ ^(zip|tar.gz)$ ]] ||
  [[ "${bridge_entry}" == */* ]] || [[ "${bridge_entry}" == *\\* ]] || [[ -z "${bridge_entry}" ]]
then
  echo "invalid imagegen bridge component manifest entry for ${target}" >&2
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
  .bundle.resources = (((.bundle.resources // {}) | with_entries(select(
    ((.key | startswith("binaries/oneday-v")) or
     (.key | startswith("binaries/oneday-gateway-v")) or
     (.key | startswith("binaries/imagegen-bridge-"))) | not
  ))) + {
    ("binaries/oneday-v\($version)-\($target)\($extension)"): "binaries/oneday-v\($version)-\($target)\($extension)",
    ("binaries/oneday-gateway-v\($version)-\($target)\($extension)"): "binaries/oneday-gateway-v\($version)-\($target)\($extension)",
    ("binaries/imagegen-bridge-\($target)\($extension)"): "binaries/imagegen-bridge-\($target)\($extension)"
  })
' "${repo_root}/desktop/src-tauri/tauri.conf.json" > "${config_tmp}"
mv "${config_tmp}" "${repo_root}/desktop/src-tauri/tauri.conf.json"

sidecar_dir="${repo_root}/desktop/src-tauri/binaries"
resource_dir="${repo_root}/desktop/gateway/web/dist"
mkdir -p "${sidecar_dir}"
find "${sidecar_dir}" -maxdepth 1 -type f \
  \( -name "oneday-v*-${target}${sidecar_extension}" \
  -o -name "oneday-gateway-v*-${target}${sidecar_extension}" \
  -o -name "imagegen-bridge-${target}${sidecar_extension}" \) \
  -delete
if [[ -e "${resource_dir}" ]]; then
  [[ "${resource_dir}" == "${repo_root}/desktop/gateway/web/dist" ]] || {
    echo "refusing to replace an unexpected desktop web resource directory" >&2
    exit 1
  }
  rm -rf -- "${resource_dir}"
fi
mkdir -p "${resource_dir}"
cp "${engine_bin}" "${sidecar_dir}/oneday-v${version}-${target}${sidecar_extension}"
cp "${gateway_bin}" "${sidecar_dir}/oneday-gateway-v${version}-${target}${sidecar_extension}"
cp -R "${web_dist}/." "${resource_dir}/"

bridge_download="$(mktemp)"
bridge_binary="${sidecar_dir}/imagegen-bridge-${target}${sidecar_extension}"
trap 'rm -f "${bridge_download}"' EXIT
curl --fail --location --retry 3 --retry-all-errors --output "${bridge_download}" "${bridge_url}"
if [[ "$(wc -c < "${bridge_download}" | tr -d '[:space:]')" != "${bridge_size}" ]]; then
  echo "imagegen bridge download size does not match the pinned manifest" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  actual_bridge_sha256="$(sha256sum "${bridge_download}" | awk '{print tolower($1)}')"
elif command -v shasum >/dev/null 2>&1; then
  actual_bridge_sha256="$(shasum -a 256 "${bridge_download}" | awk '{print tolower($1)}')"
else
  echo "a SHA-256 utility is required to verify the imagegen bridge" >&2
  exit 1
fi
expected_bridge_sha256="$(printf '%s' "${bridge_sha256}" | tr '[:upper:]' '[:lower:]')"
if [[ "${actual_bridge_sha256}" != "${expected_bridge_sha256}" ]]; then
  echo "imagegen bridge download SHA-256 does not match the pinned manifest" >&2
  exit 1
fi
case "${bridge_archive}" in
  zip)
    if command -v unzip >/dev/null 2>&1; then
      unzip -p "${bridge_download}" "${bridge_entry}" > "${bridge_binary}"
    elif command -v powershell.exe >/dev/null 2>&1; then
      BRIDGE_ARCHIVE="${bridge_download}" BRIDGE_ENTRY="${bridge_entry}" BRIDGE_OUTPUT="${bridge_binary}" \
        powershell.exe -NoProfile -NonInteractive -Command '
          Add-Type -AssemblyName System.IO.Compression.FileSystem
          $archive = [System.IO.Compression.ZipFile]::OpenRead($env:BRIDGE_ARCHIVE)
          try {
            $entry = $archive.GetEntry($env:BRIDGE_ENTRY)
            if ($null -eq $entry) { throw "expected bridge entry is missing" }
            $input = $entry.Open()
            try {
              $output = [System.IO.File]::Open($env:BRIDGE_OUTPUT, [System.IO.FileMode]::CreateNew, [System.IO.FileAccess]::Write)
              try { $input.CopyTo($output) } finally { $output.Dispose() }
            } finally { $input.Dispose() }
          } finally { $archive.Dispose() }
        '
    else
      echo "unzip or PowerShell is required to extract the imagegen bridge ZIP" >&2
      exit 1
    fi
    ;;
  tar.gz)
    tar -xOzf "${bridge_download}" "${bridge_entry}" > "${bridge_binary}"
    ;;
esac
[[ -s "${bridge_binary}" ]] || { echo "imagegen bridge archive did not contain an executable" >&2; exit 1; }
if [[ "${sidecar_extension}" != ".exe" ]]; then
  chmod 0755 "${bridge_binary}"
fi

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
