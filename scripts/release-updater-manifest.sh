#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage:
  release-updater-manifest.sh generate <tag> <owner/repo> <asset-dir> <output.json> [notes] [pub-date]
  release-updater-manifest.sh verify <tag> <owner/repo> <asset-dir> <latest.json> [public-key]
EOF
  exit 2
}

minisign_public_key() {
  local supplied="$1"
  local decoded
  local line
  local -a candidates=()

  while IFS= read -r line; do
    line="${line%$'\r'}"
    [[ -z "${line//[[:space:]]/}" ]] && continue
    [[ "${line}" =~ ^[[:space:]]*(untrusted|trusted)[[:space:]]comment: ]] && continue
    line="${line//[[:space:]]/}"
    candidates+=("${line}")
  done <<< "${supplied}"

  if [[ ${#candidates[@]} -eq 1 && "${candidates[0]}" =~ ^[A-Za-z0-9+/]{56}$ ]]; then
    printf '%s\n' "${candidates[0]}"
    return
  fi

  # Tauri stores the complete minisign public-key file as one additional
  # base64 layer. Accept that canonical form as well as a raw minisign file.
  if [[ ${#candidates[@]} -eq 1 ]] && \
    decoded="$(printf '%s' "${candidates[0]}" | base64 --decode 2>/dev/null)"
  then
    candidates=()
    while IFS= read -r line; do
      line="${line%$'\r'}"
      [[ -z "${line//[[:space:]]/}" ]] && continue
      [[ "${line}" =~ ^[[:space:]]*(untrusted|trusted)[[:space:]]comment: ]] && continue
      line="${line//[[:space:]]/}"
      candidates+=("${line}")
    done <<< "${decoded}"
  fi

  if [[ ${#candidates[@]} -ne 1 || ! "${candidates[0]}" =~ ^[A-Za-z0-9+/]{56}$ ]]; then
    echo "invalid minisign public key: expected a raw, file-form, or Tauri-encoded public key" >&2
    return 1
  fi

  printf '%s\n' "${candidates[0]}"
}

[[ $# -ge 5 ]] || usage
mode="$1"
tag="$2"
repository="$3"
asset_dir="$4"
manifest="$5"
version="${tag#v}"

if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
  echo "invalid updater tag: ${tag}" >&2
  exit 1
fi
if [[ ! "${repository}" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
  echo "invalid GitHub repository: ${repository}" >&2
  exit 1
fi

linux_asset="oneday-desktop-${version}-linux-x86_64.AppImage"
windows_asset="oneday-desktop-${version}-windows-x86_64-setup.exe"
macos_aarch64_asset="oneday-desktop-${version}-macos-aarch64.app.tar.gz"
macos_x86_64_asset="oneday-desktop-${version}-macos-x86_64.app.tar.gz"
for asset in \
  "${linux_asset}" \
  "${windows_asset}" \
  "${macos_aarch64_asset}" \
  "${macos_x86_64_asset}"
do
  [[ -s "${asset_dir}/${asset}" ]] || { echo "missing updater asset: ${asset}" >&2; exit 1; }
  [[ -s "${asset_dir}/${asset}.sig" ]] || { echo "missing updater signature: ${asset}.sig" >&2; exit 1; }
done

case "${mode}" in
  generate)
    notes="${6:-OneDay ${version}}"
    linux_signature="$(<"${asset_dir}/${linux_asset}.sig")"
    windows_signature="$(<"${asset_dir}/${windows_asset}.sig")"
    macos_aarch64_signature="$(<"${asset_dir}/${macos_aarch64_asset}.sig")"
    macos_x86_64_signature="$(<"${asset_dir}/${macos_x86_64_asset}.sig")"
    pub_date="${7:-$(date -u +'%Y-%m-%dT%H:%M:%SZ')}"
    jq -n \
      --arg version "${version}" \
      --arg notes "${notes}" \
      --arg pub_date "${pub_date}" \
      --arg linux_signature "${linux_signature}" \
      --arg linux_url "https://github.com/${repository}/releases/download/${tag}/${linux_asset}" \
      --arg windows_signature "${windows_signature}" \
      --arg windows_url "https://github.com/${repository}/releases/download/${tag}/${windows_asset}" \
      --arg macos_aarch64_signature "${macos_aarch64_signature}" \
      --arg macos_aarch64_url "https://github.com/${repository}/releases/download/${tag}/${macos_aarch64_asset}" \
      --arg macos_x86_64_signature "${macos_x86_64_signature}" \
      --arg macos_x86_64_url "https://github.com/${repository}/releases/download/${tag}/${macos_x86_64_asset}" \
      '{
        version: $version,
        notes: $notes,
        pub_date: $pub_date,
        platforms: {
          "linux-x86_64": { signature: $linux_signature, url: $linux_url },
          "windows-x86_64": { signature: $windows_signature, url: $windows_url },
          "darwin-aarch64": { signature: $macos_aarch64_signature, url: $macos_aarch64_url },
          "darwin-x86_64": { signature: $macos_x86_64_signature, url: $macos_x86_64_url }
        }
      }' > "${manifest}"
    "$0" verify "${tag}" "${repository}" "${asset_dir}" "${manifest}"
    ;;
  verify)
    jq -e \
      --arg version "${version}" \
      --arg linux_url "https://github.com/${repository}/releases/download/${tag}/${linux_asset}" \
      --arg windows_url "https://github.com/${repository}/releases/download/${tag}/${windows_asset}" \
      --arg macos_aarch64_url "https://github.com/${repository}/releases/download/${tag}/${macos_aarch64_asset}" \
      --arg macos_x86_64_url "https://github.com/${repository}/releases/download/${tag}/${macos_x86_64_asset}" '
        .version == $version and
        (.pub_date | fromdateiso8601 | type == "number") and
        .platforms["linux-x86_64"].url == $linux_url and
        .platforms["windows-x86_64"].url == $windows_url and
        .platforms["darwin-aarch64"].url == $macos_aarch64_url and
        .platforms["darwin-x86_64"].url == $macos_x86_64_url and
        (.platforms["linux-x86_64"].signature | length > 20) and
        (.platforms["windows-x86_64"].signature | length > 20) and
        (.platforms["darwin-aarch64"].signature | length > 20) and
        (.platforms["darwin-x86_64"].signature | length > 20)
      ' "${manifest}" >/dev/null
    [[ "$(jq -r '.platforms["linux-x86_64"].signature' "${manifest}")" == "$(<"${asset_dir}/${linux_asset}.sig")" ]]
    [[ "$(jq -r '.platforms["windows-x86_64"].signature' "${manifest}")" == "$(<"${asset_dir}/${windows_asset}.sig")" ]]
    [[ "$(jq -r '.platforms["darwin-aarch64"].signature' "${manifest}")" == "$(<"${asset_dir}/${macos_aarch64_asset}.sig")" ]]
    [[ "$(jq -r '.platforms["darwin-x86_64"].signature' "${manifest}")" == "$(<"${asset_dir}/${macos_x86_64_asset}.sig")" ]]
    if [[ -n "${6:-}" ]]; then
      command -v minisign >/dev/null 2>&1 || { echo "minisign is required for signature verification" >&2; exit 1; }
      public_key="$(minisign_public_key "$6")"
      for asset in \
        "${linux_asset}" \
        "${windows_asset}" \
        "${macos_aarch64_asset}" \
        "${macos_x86_64_asset}"
      do
        signature_path="${asset_dir}/${asset}.sig"
        if grep -q '^[[:space:]]*untrusted comment:' "${signature_path}"; then
          minisign -Vm "${asset_dir}/${asset}" -x "${signature_path}" -P "${public_key}"
        else
          # Tauri base64-encodes the complete minisign signature file before
          # writing the .sig artifact and embedding it in latest.json.
          minisign -Vm "${asset_dir}/${asset}" \
            -x <(base64 --decode < "${signature_path}") -P "${public_key}"
        fi
      done
    fi
    ;;
  *)
    usage
    ;;
esac
