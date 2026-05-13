#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

printf "friend-safe release hygiene\n"

forbidden_regex='(^|/)(config\.yaml|\.env|oneday_data/|oneday\.db|.*\.(db|sqlite|sqlite3)|oneday|oneday-benchmark|oneday-ascii-benchmark)$'

if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  tracked="$(git ls-files)"
else
  tracked="$(find . -type f -not -path './.git/*' | sed 's#^\./##')"
fi

bad="$(printf "%s\n" "${tracked}" | grep -E "${forbidden_regex}" || true)"
if [[ -n "${bad}" ]]; then
  echo "Forbidden local/share artifacts are tracked or package-visible:" >&2
  printf "%s\n" "${bad}" >&2
  exit 1
fi

printf "OK: no tracked config/env/data/db/binary artifacts found.\n"
