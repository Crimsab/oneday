# QA Matrix

High-risk release and regression checklist for OneDay. The goal is to keep mixed-system flows visible, not just isolated helpers.

## Automated Sweep

Run the focused sweep with:

```bash
make verify
```

The full script also rebuilds `./oneday`, prints `./oneday --version`, and then prints the manual checklist:

```bash
./scripts/qa-matrix.sh
```

## Matrix

| Scenario | Automated coverage | Manual focus | 2026-04-09 sweep |
| --- | --- | --- | --- |
| Fresh binary identity | `internal/buildinfo/buildinfo_test.go`, `cmd/oneday/main_test.go`, `scripts/qa-matrix.sh` rebuild + `./oneday --version` | Confirm the running binary matches the version shown in the terminal before playtesting | Passed |
| Canonical save, load, and resume | `TestSaveGameAndLoadGameRestoreCanonicalStoryState`, `TestAppLoadSaveAndResumeRestoresStoryState`, `TestAppShowSaveListCancelReturnsToNarrative` | Create a save in the TUI, load it back, and confirm the story resumes without stale choices or wrong turn state | Passed |
| Legacy save load fallback | `TestAppLoadLegacySaveShowsPartialRollbackWarning` | Load an older snapshot and confirm the warning about partial rollback is visible immediately | Passed |
| Rewind branch provenance | `TestSaveGameWithMetadataPersistsRewindBranchContext` | Load a rewind branch and confirm the save picker still communicates rewind context clearly | Automated pass, manual follow-up still recommended |
| Fronts + investigations | `TestBuildStoryCodexShowsVisibleFrontsWithoutLeakingHiddenState`, `TestBuildStoryCodexSurfacesInvestigationCasesWithoutHiddenTruthLeaks`, `TestNarrativeSlashCommandsOpenCodexBrowsers` | Open `/codex` and `/investigations`; confirm hidden truths stay hidden while visible fronts and clues remain linked | Passed |
| Projects + fronts/pressure | `TestBuildStoryCodexSurfacesProjectsAndBacklinks`, `TestNarrativeSlashCommandsOpenCodexBrowsers`, `TestNarrativeQuickSaveHotkeyCreatesSnapshot`, `TestApplyStateChangesProjectUpdateAdvancesWithCostAndPressure`, `TestApplyStateChangesProjectUpdateCompleteAppliesDurableRewards` | Advance a project in a pressured region, save/load, and confirm both project state and pressure fallout still line up | Automated pass, manual long-run follow-up still recommended |
| Social-duel runtime handoff | `TestApplyNarrativeResponseQueuesSocialDuelPrelude`, `TestApplyNarrativeResponseFallsBackToActiveSocialDuelCue`, `TestBeginPendingSocialDuelStartsViewAndState` | Accept or continue a duel from the TUI and verify the transition is clean | Passed |
| Social-duel aftermath | `TestApplySocialDuelAftermathPersistsRelationshipAndRumor`, `TestApplySocialDuelAftermathPersistsFailForwardAndFrontPressure` | Resolve a duel and verify relationship/world fallout shows up in later surfaces | Passed |

## Local Sweep Notes

- `./scripts/qa-matrix.sh --automated-only` passed on `2026-04-09`.
- `go test ./...` and `go vet ./...` were also green during the same sweep.
- The main gap that was closed during this pass was explicit legacy-save load coverage in the TUI/app flow.
