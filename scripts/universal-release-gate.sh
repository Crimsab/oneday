#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"
export CARGO_TARGET_DIR="${CARGO_TARGET_DIR:-${HOME}/.cache/oneday/cargo-target}"
mkdir -p "${CARGO_TARGET_DIR}"

step() { printf '\n== %s ==\n' "$1"; }
run() { printf '+ %s\n' "$*"; "$@"; }

step "Go, migrations, compatibility, and authoring"
run go test ./...
run go vet ./...
run go test -count=1 ./internal/storage -run 'TestMigration|TestLegacy|TestVisualCanon|TestWorldCanon|TestSnapshot'
run go test -count=1 ./internal/engine -run 'TestLoadStoryPack|TestSaveGame|TestLoadGame|TestLegacy|TestOutcome|TestOffscreen'
run go run ./cmd/oneday story-packs list
run go run ./cmd/oneday-minigame-eval -corpus internal/engine/testdata/minigame-evals.json

step "Rust gateway"
run cargo fmt --manifest-path gateway/Cargo.toml -- --check
run cargo test --manifest-path gateway/Cargo.toml
run cargo clippy --manifest-path gateway/Cargo.toml --all-targets -- -A clippy::too_many_arguments -D warnings
run cargo build --release --manifest-path gateway/Cargo.toml

step "React and browser"
(
  cd gateway/web
  run bun install --frozen-lockfile
  run bun run test
  run bun run build
  run bun run test:e2e
)

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  step "Gateway container"
  run docker build -f gateway/Dockerfile .
else
  printf '\nDocker daemon unavailable; container gate must run in CI or release host.\n'
fi

step "Universal release gate passed"
