# Plan 20.2 Summary

Added engine-side `investigation_update` normalization so the narrator can propose clue, suspect, claim, contradiction, lead, theory, and hidden-truth reveal changes without directly mutating the board schema.

The update path now merges duplicates, degrades malformed payloads safely, supports theory movement and hidden-truth reveals, and feeds open-investigation digest lines back into narrator context for continuity.

Code commit: `8e184b2 feat(investigation): normalize evidence and theory updates`
