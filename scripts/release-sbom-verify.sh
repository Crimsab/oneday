#!/usr/bin/env bash
set -euo pipefail

[[ $# -eq 1 ]] || { echo "usage: $0 <sbom.spdx.json>" >&2; exit 2; }
sbom="$1"

jq -e '
  .spdxVersion == "SPDX-2.3" and
  .dataLicense == "CC0-1.0" and
  (.documentNamespace | startswith("https://")) and
  (.creationInfo.created | fromdateiso8601 | type == "number") and
  (.packages | type == "array" and length > 0)
' "${sbom}" >/dev/null
