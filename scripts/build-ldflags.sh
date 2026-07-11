#!/usr/bin/env bash
set -euo pipefail

buildinfo_pkg="github.com/crimsab/oneday/internal/buildinfo"

version="${VERSION:-}"
if [[ -z "$version" ]]; then
  version="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
fi

commit="${COMMIT:-}"
if [[ -z "$commit" ]]; then
  commit="$(git rev-parse HEAD 2>/dev/null || echo unknown)"
fi

build_date="${BUILD_DATE:-}"
if [[ -z "$build_date" ]]; then
  build_date="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
fi

dirty="${DIRTY:-}"
if [[ -z "$dirty" ]]; then
  if [[ -n "$(git status --porcelain 2>/dev/null)" ]]; then
    dirty="true"
  else
    dirty="false"
  fi
fi

printf "%s" "-X ${buildinfo_pkg}.Version=${version} -X ${buildinfo_pkg}.Commit=${commit} -X ${buildinfo_pkg}.BuildDate=${build_date} -X ${buildinfo_pkg}.Dirty=${dirty}"
