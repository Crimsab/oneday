#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <linux|windows> <version> <bundle-directory> <output-directory>" >&2
  exit 2
}

[[ $# -eq 4 ]] || usage
platform="$1"
version="$2"
bundle_dir="$3"
output_dir="$4"

if [[ ! "v${version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
  echo "invalid desktop version: ${version}" >&2
  exit 1
fi
mkdir -p "${output_dir}"

copy_unique() {
  local pattern="$1"
  local destination="$2"
  local -a matches=()
  mapfile -d '' matches < <(find "${bundle_dir}" -type f -path "${pattern}" -print0)
  if [[ "${#matches[@]}" -ne 1 ]]; then
    echo "expected one artifact matching ${pattern}, found ${#matches[@]}" >&2
    exit 1
  fi
  cp "${matches[0]}" "${output_dir}/${destination}"
}

case "${platform}" in
  linux)
    base="oneday-desktop-${version}-linux-x86_64.AppImage"
    copy_unique '*/appimage/*.AppImage' "${base}"
    copy_unique '*/appimage/*.AppImage.sig' "${base}.sig"
    copy_unique '*/deb/*.deb' "oneday-desktop-${version}-linux-amd64.deb"
    ;;
  windows)
    base="oneday-desktop-${version}-windows-x86_64-setup.exe"
    copy_unique '*/nsis/*-setup.exe' "${base}"
    copy_unique '*/nsis/*-setup.exe.sig' "${base}.sig"
    ;;
  *)
    usage
    ;;
esac
