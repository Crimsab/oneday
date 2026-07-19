#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
timeout_seconds="${ONEDAY_MATRIX_TIMEOUT_SECONDS:-300}"
slice="${1:-all}"

usage() {
  cat <<'EOF'
Usage: scripts/first-run-matrix.sh [cli|all]

Runs first-run/portability proof slices against temporary state only. The CLI
slice uses the existing setup, doctor, fake narrator, and persistence contracts.
EOF
}

if [[ "$slice" == "--help" || "$slice" == "-h" ]]; then
  usage
  exit 0
fi
if [[ "$slice" != "cli" && "$slice" != "all" ]]; then
  usage >&2
  exit 2
fi
if [[ ! "$timeout_seconds" =~ ^[1-9][0-9]*$ ]]; then
  printf 'ONEDAY_MATRIX_TIMEOUT_SECONDS must be a positive integer.\n' >&2
  exit 2
fi
for command in go timeout; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'first-run matrix requires %s on PATH.\n' "$command" >&2
    exit 127
  fi
done

workspace="$(mktemp -d "${TMPDIR:-/tmp}/oneday-first-run-matrix.XXXXXX")"
cleanup() {
  if [[ -d "$workspace" && "$workspace" == "${TMPDIR:-/tmp}/oneday-first-run-matrix."* ]]; then
    rm -rf -- "$workspace"
  fi
}
trap cleanup EXIT

cd "$repo_root"

run() {
  printf '+ '
  printf '%q ' "$@"
  printf '\n'
  timeout --foreground "${timeout_seconds}s" "$@"
}

run_isolated() {
  run env -i \
    PATH="$PATH" \
    HOME="${HOME:-$workspace/home}" \
    TZ=UTC \
    GOPROXY=off \
    GOSUMDB=off \
    ONEDAY_CONFIG="$workspace/config.yaml" \
    ONEDAY_DB_PATH="$workspace/oneday.db" \
    "$@"
}

run_cli() {
  printf '== CLI: empty configuration, setup/doctor redaction, first canonical turn ==\n'
  run_isolated go test -count=1 ./cmd/oneday -run 'TestSetupConfigForChoice|TestSetupNoInputRequiresExistingNarrativeConfiguration|TestDoctorJSONAndTextShareReadinessProbesAndRequiredExit|TestDoctorUsesExplicitDatabasePathWithoutExposingIt'
  run_isolated go test -count=1 ./internal/game/service -run 'TestInProcessTurnServiceSubmitActionProducesOrderedEvents|TestInProcessTurnServiceCreateAndLoadSave'
}

run_cli
printf 'First-run matrix slice %s passed.\n' "$slice"
