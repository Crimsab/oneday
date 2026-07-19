#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <vX.Y.Z[-prerelease]> <output-directory>" >&2
  exit 2
}

[[ $# -eq 2 ]] || usage

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
tag="$1"
version="${tag#v}"
mkdir -p "$2"
output_dir="$(cd "$2" && pwd)"
cd "${repo_root}"

metadata="$(mktemp)"
work_dir="$(mktemp -d)"
cleanup() {
  rm -f "${metadata}"
  rm -rf "${work_dir}"
}
trap cleanup EXIT

"${script_dir}/release-metadata.sh" "${tag}" "${metadata}"
source_date_epoch="$(jq -er '.sourceDateEpoch' "${metadata}")"
build_date="$(jq -er '.sourceDate' "${metadata}")"
source_commit="$(jq -er '.sourceCommit' "${metadata}")"

export VERSION="${tag}"
export COMMIT="${source_commit}"
export BUILD_DATE="${build_date}"
export DIRTY=false
export SOURCE_DATE_EPOCH="${source_date_epoch}"

ldflags="$(bash "${script_dir}/build-ldflags.sh")"
go_build=(go build -trimpath -buildvcs=false -ldflags "${ldflags}")

mkdir -p "${work_dir}/bin" "${work_dir}/package"

build_target() {
  local goos="$1"
  local extension="$2"
  local target_dir="${work_dir}/bin/${goos}"
  mkdir -p "${target_dir}"
  GOOS="${goos}" GOARCH=amd64 "${go_build[@]}" -o "${target_dir}/oneday${extension}" "${repo_root}/cmd/oneday"
  GOOS="${goos}" GOARCH=amd64 "${go_build[@]}" -o "${target_dir}/oneday-benchmark${extension}" "${repo_root}/cmd/oneday-benchmark"
  GOOS="${goos}" GOARCH=amd64 "${go_build[@]}" -o "${target_dir}/oneday-ascii-benchmark${extension}" "${repo_root}/cmd/oneday-ascii-benchmark"
}

build_target linux ""
build_target windows ".exe"

version_output="$("${work_dir}/bin/linux/oneday" --version)"
grep -Fqx "oneday ${tag}" <<<"${version_output}"
grep -Fqx "commit: ${source_commit:0:12}" <<<"${version_output}"
grep -Fqx "dirty: false" <<<"${version_output}"

linux_name="oneday-${version}-linux-amd64"
windows_name="oneday-${version}-windows-amd64"
mkdir -p "${work_dir}/package/${linux_name}" "${work_dir}/package/${windows_name}"
cp "${work_dir}/bin/linux/oneday" "${work_dir}/package/${linux_name}/oneday"
cp "${work_dir}/bin/linux/oneday-benchmark" "${work_dir}/package/${linux_name}/oneday-benchmark"
cp "${work_dir}/bin/linux/oneday-ascii-benchmark" "${work_dir}/package/${linux_name}/oneday-ascii-benchmark"
cp "${work_dir}/bin/windows/oneday.exe" "${work_dir}/package/${windows_name}/oneday.exe"
cp "${work_dir}/bin/windows/oneday-benchmark.exe" "${work_dir}/package/${windows_name}/oneday-benchmark.exe"
cp "${work_dir}/bin/windows/oneday-ascii-benchmark.exe" "${work_dir}/package/${windows_name}/oneday-ascii-benchmark.exe"
cp "${repo_root}/config.example.yaml" "${work_dir}/package/${linux_name}/config.example.yaml"
cp "${repo_root}/config.example.yaml" "${work_dir}/package/${windows_name}/config.example.yaml"
cp "${metadata}" "${work_dir}/package/${linux_name}/release-metadata.json"
cp "${metadata}" "${work_dir}/package/${windows_name}/release-metadata.json"

find "${work_dir}/package" -exec touch -h -d "@${source_date_epoch}" {} +
tar --sort=name \
  --mtime="@${source_date_epoch}" \
  --owner=0 --group=0 --numeric-owner \
  -C "${work_dir}/package" \
  -czf "${output_dir}/${linux_name}.tar.gz" \
  "${linux_name}"

(
  cd "${work_dir}/package"
  find "${windows_name}" -type f -print0 \
    | LC_ALL=C sort -z \
    | xargs -0 zip -X -q "${output_dir}/${windows_name}.zip"
)
cp "${metadata}" "${output_dir}/release-metadata.json"
