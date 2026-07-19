#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

step() {
  printf "\n== %s ==\n" "$1"
}

run() {
  printf "+ %s\n" "$*"
  "$@"
}

if [[ -n "$(git status --porcelain)" ]]; then
  echo "release gate requires a clean git worktree" >&2
  git status --short >&2
  exit 1
fi

step "Verification"
run make universal-release-check
run make friend-safe-check

step "Release Supply Chain"
run bash -n \
  scripts/release-metadata.sh \
  scripts/release-package.sh \
  scripts/release-prepare-desktop.sh \
  scripts/release-desktop-artifacts.sh \
  scripts/release-updater-manifest.sh \
  scripts/release-sbom-verify.sh
release_metadata="$(mktemp)"
trap 'rm -f "${release_metadata}"' EXIT
run ./scripts/release-metadata.sh v0.0.0-release-check "${release_metadata}"
run jq -e '
  .applicationVersion == .desktopVersion and
  (.gatewayProtocolVersion | type == "number") and
  (.databaseSchemaVersion | type == "number") and
  (.sourceCommit | length == 40)
' "${release_metadata}"

step "Release Artifact Builds"
run make build
run make build-bench
run make build-ascii-bench
run make build-cross

step "Build Provenance"
version_output="$(./oneday --version)"
printf "%s\n" "${version_output}"

expected_commit="$(git rev-parse --short=12 HEAD)"
actual_commit="$(printf "%s\n" "${version_output}" | awk '/^commit:/ {print $2}')"
actual_dirty="$(printf "%s\n" "${version_output}" | awk '/^dirty:/ {print $2}')"

if [[ "${actual_commit}" != "${expected_commit}" ]]; then
  echo "binary commit mismatch: expected ${expected_commit}, got ${actual_commit}" >&2
  exit 1
fi

if [[ "${actual_dirty}" != "false" ]]; then
  echo "binary is dirty; rebuild from a clean tree before shipping" >&2
  exit 1
fi

for artifact in \
  ./oneday \
  ./oneday-benchmark \
  ./oneday-ascii-benchmark \
  ./build/oneday-linux-amd64 \
  ./build/oneday-windows-amd64.exe
do
  if [[ ! -f "${artifact}" ]]; then
    echo "missing release artifact: ${artifact}" >&2
    exit 1
  fi
done

step "Release Gate Passed"
printf "All verification checks and expected artifacts are present.\n"
