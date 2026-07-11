#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

automated_only="false"
if [[ "${1:-}" == "--automated-only" ]]; then
  automated_only="true"
fi

step() {
  printf "\n== %s ==\n" "$1"
}

run() {
  printf "+ %s\n" "$*"
  "$@"
}

step "Build Fresh Binary"
run make build
run ./oneday --version

step "Cross-System Engine Sweep"
run go test -count=1 ./internal/engine -run 'TestSaveGameAndLoadGameRestoreCanonicalStoryState|TestSaveGameWithMetadataPersistsRewindBranchContext|TestBuildStoryCodexShowsVisibleFrontsWithoutLeakingHiddenState|TestBuildStoryCodexSurfacesInvestigationCasesWithoutHiddenTruthLeaks|TestBuildStoryCodexSurfacesProjectsAndBacklinks|TestApplySocialDuelAftermathPersistsRelationshipAndRumor|TestApplySocialDuelAftermathPersistsFailForwardAndFrontPressure'

step "Narrative Runtime Sweep"
run go test -count=1 ./internal/tui/views -run 'TestNarrativeSlashCommandsOpenCodexBrowsers|TestNarrativeQuickSaveHotkeyCreatesSnapshot|TestApplyNarrativeResponseQueuesSocialDuelPrelude|TestApplyNarrativeResponseFallsBackToActiveSocialDuelCue|TestBeginPendingSocialDuelStartsViewAndState'

step "App Save/Resume Sweep"
run go test -count=1 ./internal/tui -run 'TestAppShowSaveListCancelReturnsToNarrative|TestAppLoadSaveAndResumeRestoresStoryState|TestAppLoadLegacySaveShowsPartialRollbackWarning'

if [[ "${automated_only}" == "true" ]]; then
  exit 0
fi

step "Manual Checklist"
cat <<'EOF'
- Launch `./oneday`, open a real story, and verify the running build matches `./oneday --version`.
- Create a manual save, quicksave, and load them back from the in-game picker without leaving stale choices behind.
- Load a rewind/branch save and confirm the save label still communicates rewind provenance.
- Open `/codex`, `/investigations`, and `/projects`; confirm entries exist, links drill down correctly, and hidden investigation truths are not surfaced.
- Play a social duel to completion and confirm the aftermath appears in NPC relationship fallout and world/front pressure surfaces.
- Advance a downtime project in a story that already has visible fronts; confirm both the project state and related pressure/front fallout remain coherent after save/load.
- If testing with an older snapshot, confirm the narrative view surfaces the legacy partial-rollback warning immediately after load.
EOF
