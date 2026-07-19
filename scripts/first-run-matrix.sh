#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
timeout_seconds="${ONEDAY_MATRIX_TIMEOUT_SECONDS:-300}"
slice="${1:-all}"

usage() {
  cat <<'EOF'
Usage: scripts/first-run-matrix.sh [cli|web|desktop|portability|providers|all]

Runs first-run/portability proof slices against temporary state only. Every
provider-facing proof uses a fixture, in-process fake, or mocked browser route.
EOF
}

if [[ "$slice" == "--help" || "$slice" == "-h" ]]; then
  usage
  exit 0
fi
if [[ ! "$slice" =~ ^(cli|web|desktop|portability|providers|all)$ ]]; then
  usage >&2
  exit 2
fi
if [[ ! "$timeout_seconds" =~ ^[1-9][0-9]*$ ]]; then
  printf 'ONEDAY_MATRIX_TIMEOUT_SECONDS must be a positive integer.\n' >&2
  exit 2
fi
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

require_commands() {
  local command
  for command in "$@"; do
    if ! command -v "$command" >/dev/null 2>&1; then
      printf 'first-run matrix requires %s on PATH.\n' "$command" >&2
      exit 127
    fi
  done
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
  require_commands go timeout
  printf '== CLI: empty configuration, setup/doctor redaction, first canonical turn ==\n'
  run_isolated go test -count=1 ./cmd/oneday -run 'TestSetupConfigForChoice|TestSetupNoInputRequiresExistingNarrativeConfiguration|TestDoctorJSONAndTextShareReadinessProbesAndRequiredExit|TestDoctorUsesExplicitDatabasePathWithoutExposingIt'
  run_isolated go test -count=1 ./internal/game/service -run 'TestInProcessTurnServiceSubmitActionProducesOrderedEvents|TestInProcessTurnServiceCreateAndLoadSave'
}

run_web() {
  require_commands cargo bun timeout
  printf '== Gateway/web: empty installation, auth bootstrap, onboarding, first action ==\n'
  run_isolated cargo test --manifest-path gateway/Cargo.toml auth::tests
  run_isolated bun --cwd gateway/web x --no-install vitest run src/features/installation-onboarding/InstallationOnboarding.test.tsx src/features/portability/templateCode.test.ts
  run_isolated bun --cwd gateway/web x --no-install playwright test e2e/oneday.e2e.ts --grep 'submits once|keeps installation readiness|reviews a story preset'
}

run_desktop() {
  require_commands cargo bun timeout
  printf '== Desktop: isolated standalone and remote profile contracts ==\n'
  run_isolated cargo test --manifest-path desktop/src-tauri/Cargo.toml
  run_isolated bun --cwd desktop run test
}

run_portability() {
  require_commands cargo bash timeout
  printf '== Portability: archive/template round-trip and recovery boundaries ==\n'
  run_isolated cargo test --manifest-path gateway/Cargo.toml portability::tests
  run_isolated bash scripts/verify-sqlite-backup-restore-test.sh
  require_commands go
  run_isolated go test -count=1 ./internal/storage -run TestReleaseUpgradeFromV114
}

run_providers() {
  require_commands cargo go timeout
  printf '== Providers: text-only, local/remote bridge and direct adapter contracts ==\n'
  run_isolated cargo test --manifest-path gateway/Cargo.toml imagegen::adapter_tests
  run_isolated go test -count=1 ./internal/ai/providers ./internal/aifactory
}

case "$slice" in
  cli) run_cli ;;
  web) run_web ;;
  desktop) run_desktop ;;
  portability) run_portability ;;
  providers) run_providers ;;
  all)
    run_cli
    run_web
    run_desktop
    run_portability
    run_providers
    ;;
esac
printf 'First-run matrix slice %s passed.\n' "$slice"
