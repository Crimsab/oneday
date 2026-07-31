#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "${fixture}"' EXIT

version="9.8.7"
tag="v${version}"
asset_dir="${fixture}/assets"
bundle_dir="${fixture}/bundle"
normalized_dir="${fixture}/normalized"
mkdir -p "${asset_dir}" "${bundle_dir}/dmg" "${bundle_dir}/macos"

mkdir -p "${fixture}/bin"
printf '#!/usr/bin/env bash\nexit 97\n' > "${fixture}/bin/date"
chmod +x "${fixture}/bin/date"
PATH="${fixture}/bin:${PATH}" \
  "${repo_root}/scripts/release-metadata.sh" \
  "${tag}" "${fixture}/release-metadata.json"
jq -e '
  .releaseTag == "v9.8.7" and
  (.sourceDate | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"))
' "${fixture}/release-metadata.json" >/dev/null

printf 'dmg fixture\n' > "${bundle_dir}/dmg/OneDay.dmg"
printf 'updater fixture\n' > "${bundle_dir}/macos/OneDay.app.tar.gz"
printf 'macOS artifact signature long enough for validation\n' > "${bundle_dir}/macos/OneDay.app.tar.gz.sig"
"${repo_root}/scripts/release-desktop-artifacts.sh" \
  macos-aarch64 "${version}" "${bundle_dir}" "${normalized_dir}"

test -s "${normalized_dir}/oneday-desktop-${version}-macos-aarch64.dmg"
test -s "${normalized_dir}/oneday-desktop-${version}-macos-aarch64.app.tar.gz"
test -s "${normalized_dir}/oneday-desktop-${version}-macos-aarch64.app.tar.gz.sig"

for asset in \
  "oneday-desktop-${version}-linux-x86_64.AppImage" \
  "oneday-desktop-${version}-windows-x86_64-setup.exe" \
  "oneday-desktop-${version}-macos-aarch64.app.tar.gz" \
  "oneday-desktop-${version}-macos-x86_64.app.tar.gz"
do
  printf 'release fixture for %s\n' "${asset}" > "${asset_dir}/${asset}"
  printf 'updater signature fixture for %s\n' "${asset}" > "${asset_dir}/${asset}.sig"
done

manifest="${fixture}/latest.json"
"${repo_root}/scripts/release-updater-manifest.sh" generate \
  "${tag}" Crimsab/oneday "${asset_dir}" "${manifest}" \
  "OneDay ${version}" "2026-07-30T12:00:00Z"
"${repo_root}/scripts/release-updater-manifest.sh" verify \
  "${tag}" Crimsab/oneday "${asset_dir}" "${manifest}"

jq -e '
  (.platforms | keys | sort) ==
    ["darwin-aarch64", "darwin-x86_64", "linux-x86_64", "windows-x86_64"]
' "${manifest}" >/dev/null

ONEDAY_UPDATER_ENDPOINT="https://github.com/Crimsab/oneday/releases/latest/download/latest.json" \
ONEDAY_UPDATER_PUBKEY="untrusted comment: test updater key
RWQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" \
  "${repo_root}/scripts/release-prepare-updater-config.sh" \
  "${repo_root}/desktop/src-tauri/tauri.release.conf.json" \
  "${fixture}/tauri.signed.conf.json"
jq -e '
  .bundle.createUpdaterArtifacts == true and
  .plugins.updater.endpoints == ["https://github.com/Crimsab/oneday/releases/latest/download/latest.json"] and
  (.plugins.updater.pubkey | startswith("untrusted comment:")) and
  .plugins.updater.windows.installMode == "passive"
' "${fixture}/tauri.signed.conf.json" >/dev/null

printf 'macOS release artifact tests passed\n'
