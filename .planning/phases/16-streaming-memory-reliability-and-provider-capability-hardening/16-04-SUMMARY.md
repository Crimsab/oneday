# Summary 16.4: Boot Path Consolidation and Reliability Regressions

Narrative startup now goes through shared app helpers for loading story state, opening sessions, mounting the narrative view, and choosing whether to start fresh or resume.

This removes the drift-prone duplication between new-story start, story resume, and load-save resume paths, so the reliability work from Phase 15-16 is consumed through one initialization path instead of three nearly identical copies.
