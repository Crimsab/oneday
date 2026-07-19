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
mkdir -p "$workspace"/{bun-cache,cache,cargo-home,cargo-target,desktop,gateway-web,go-cache,go-mod,go-tmp,home,playwright-browsers,playwright-results,rustup,tmp}
go_mod_cache="${ONEDAY_MATRIX_GOMODCACHE:-$workspace/go-mod}"
cargo_registry_dir="${ONEDAY_MATRIX_CARGO_REGISTRY_DIR:-}"
cargo_git_dir="${ONEDAY_MATRIX_CARGO_GIT_DIR:-}"
rustup_home="${ONEDAY_MATRIX_RUSTUP_HOME:-$workspace/rustup}"
playwright_browsers_path="${ONEDAY_MATRIX_PLAYWRIGHT_BROWSERS_PATH:-$workspace/playwright-browsers}"
bun_install_cache_dir="${ONEDAY_MATRIX_BUN_INSTALL_CACHE_DIR:-$workspace/bun-cache}"

require_cache_directory() {
  local name="$1"
  local path="$2"
  if [[ ! -d "$path" ]]; then
    printf '%s must name an existing directory: %s\n' "$name" "$path" >&2
    exit 2
  fi
}

if [[ -n "$cargo_registry_dir" ]]; then
  require_cache_directory ONEDAY_MATRIX_CARGO_REGISTRY_DIR "$cargo_registry_dir"
  ln -s "$cargo_registry_dir" "$workspace/cargo-home/registry"
fi
if [[ -n "$cargo_git_dir" ]]; then
  require_cache_directory ONEDAY_MATRIX_CARGO_GIT_DIR "$cargo_git_dir"
  ln -s "$cargo_git_dir" "$workspace/cargo-home/git"
fi
require_cache_directory ONEDAY_MATRIX_GOMODCACHE "$go_mod_cache"
require_cache_directory ONEDAY_MATRIX_RUSTUP_HOME "$rustup_home"
require_cache_directory ONEDAY_MATRIX_PLAYWRIGHT_BROWSERS_PATH "$playwright_browsers_path"
require_cache_directory ONEDAY_MATRIX_BUN_INSTALL_CACHE_DIR "$bun_install_cache_dir"
matrix_environment=(
  env -i
  "PATH=$PATH"
  "HOME=$workspace/home"
  TZ=UTC
  "TMPDIR=$workspace/tmp"
  "XDG_CACHE_HOME=$workspace/cache"
  "CARGO_HOME=$workspace/cargo-home"
  CARGO_NET_OFFLINE=true
  "CARGO_TARGET_DIR=$workspace/cargo-target"
  "GOCACHE=$workspace/go-cache"
  "GOMODCACHE=$go_mod_cache"
  "GOTMPDIR=$workspace/go-tmp"
  GOPROXY=off
  GOSUMDB=off
  "RUSTUP_HOME=$rustup_home"
  "PLAYWRIGHT_BROWSERS_PATH=$playwright_browsers_path"
  "BUN_INSTALL_CACHE_DIR=$bun_install_cache_dir"
  CHOKIDAR_USEPOLLING=true
  CHOKIDAR_INTERVAL=100
  "ONEDAY_CONFIG=$workspace/config.yaml"
  "ONEDAY_DB_PATH=$workspace/oneday.db"
)
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
  run "${matrix_environment[@]}" "$@"
}

matrix_env() {
  "${matrix_environment[@]}" "$@"
}

require_go_tests() {
  local package="$1"
  shift
  local listed
  listed="$(matrix_env go test -list . "$package")"
  local test_name
  for test_name in "$@"; do
    if ! grep -Fxq "$test_name" <<<"$listed"; then
      printf 'expected Go test %s in %s was not listed\n' "$test_name" "$package" >&2
      exit 1
    fi
  done
}

run_go_tests() {
  local package="$1"
  shift
  require_go_tests "$package" "$@"
  local expression
  expression="$(IFS='|'; printf '%s' "$*")"
  run_isolated go test -count=1 "$package" -run "^(${expression})$"
}

require_cargo_filter() {
  local manifest="$1"
  local filter="$2"
  local listed
  listed="$(matrix_env cargo test --manifest-path "$manifest" -- --list)"
  if ! grep -Fq "$filter" <<<"$listed"; then
    printf 'expected Cargo test filter %s in %s was not listed\n' "$filter" "$manifest" >&2
    exit 1
  fi
}

run_cargo_tests() {
  local manifest="$1"
  local filter="$2"
  require_cargo_filter "$manifest" "$filter"
  run_isolated cargo test --manifest-path "$manifest" "$filter"
}

copy_tracked_tree() {
  local source_directory="$1"
  local destination_directory="$2"
  local source_path relative_path destination_path
  while IFS= read -r -d '' source_path; do
    relative_path="${source_path#"$source_directory"/}"
    destination_path="$destination_directory/$relative_path"
    mkdir -p "$(dirname "$destination_path")"
    cp -- "$repo_root/$source_path" "$destination_path"
  done < <(git -C "$repo_root" ls-files -z -- "$source_directory")
}

prepare_web_workspace() {
  copy_tracked_tree gateway/web "$workspace/gateway-web"
  matrix_env bash -c 'cd "$1" && bun install --frozen-lockfile --offline' -- "$workspace/gateway-web"
}

prepare_desktop_workspace() {
  copy_tracked_tree desktop "$workspace/desktop"
  matrix_env bash -c 'cd "$1" && bun install --frozen-lockfile --offline' -- "$workspace/desktop"
}

run_web_tool() {
  run_isolated bash -c 'cd "$1"; shift; bun x --no-install "$@"' -- "$workspace/gateway-web" "$@"
}

require_playwright_matches() {
  local output
  output="$(matrix_env bash -c 'cd "$1" && bun x --no-install playwright test --list e2e/oneday.e2e.ts --grep "$2"' -- "$workspace/gateway-web" "$1")"
  if ! grep -Eq 'Total: [1-9][0-9]* test' <<<"$output"; then
    printf 'Playwright grep selected zero tests: %s\n' "$1" >&2
    exit 1
  fi
}

run_cli() {
  require_commands go timeout
  printf '== CLI: empty configuration, setup/doctor redaction, first canonical turn ==\n'
  run_go_tests ./cmd/oneday \
    TestSetupConfigForChoice \
    TestSetupNoInputRequiresExistingNarrativeConfiguration \
    TestDoctorJSONAndTextShareReadinessProbesAndRequiredExit \
    TestDoctorUsesExplicitDatabasePathWithoutExposingIt
  run_go_tests ./internal/game/service \
    TestInProcessTurnServiceSubmitActionProducesOrderedEvents \
    TestInProcessTurnServiceCreateAndLoadSave
}

run_web() {
  require_commands cargo bun git timeout
  printf '== Gateway/web: empty installation, auth bootstrap, onboarding, first action ==\n'
  run_cargo_tests gateway/Cargo.toml auth::tests
  prepare_web_workspace
  run_web_tool vitest run src/features/installation-onboarding/InstallationOnboarding.test.tsx src/features/portability/templateCode.test.ts
  require_playwright_matches 'submits once|keeps installation readiness|reviews a story preset'
  run_web_tool playwright test --output "$workspace/playwright-results" e2e/oneday.e2e.ts --grep 'submits once|keeps installation readiness|reviews a story preset'
}

run_desktop() {
  require_commands cargo bun git timeout
  printf '== Desktop: isolated standalone and remote profile contracts ==\n'
  run_cargo_tests desktop/src-tauri/Cargo.toml config::tests
  run_cargo_tests desktop/src-tauri/Cargo.toml standalone::tests
  run_cargo_tests desktop/src-tauri/Cargo.toml portability::tests
  prepare_desktop_workspace
  run_isolated bash -c 'cd "$1" && bun run test' -- "$workspace/desktop"
}

run_portability() {
  require_commands cargo bash timeout
  printf '== Portability: archive/template round-trip and recovery boundaries ==\n'
  run_cargo_tests gateway/Cargo.toml portability::tests
  run_isolated bash scripts/verify-sqlite-backup-restore-test.sh
  require_commands go
  run_go_tests ./internal/storage TestReleaseUpgradeFromV114
}

run_providers() {
  require_commands cargo go timeout
  printf '== Providers: text-only, local/remote bridge and direct adapter contracts ==\n'
  run_cargo_tests gateway/Cargo.toml imagegen::adapter_tests
  run_go_tests ./internal/ai/providers TestOpenAICompatComplete
  run_go_tests ./internal/aifactory TestSelectEmbeddingProviderAllowsLocalWithoutAPIKey
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
