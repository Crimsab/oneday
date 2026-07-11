# Summary 15.1: Canonical Turn Commit Contract

Added an engine-owned canonical turn commit path through `GameSession.CommitTurn`, so the main narrative loop now persists character state, world state, and canonical chat history inside one DB transaction before the session turn advances.

The narrator no longer advances `session.turn` or `world.CurrentTurn` optimistically. Meta/history-only paths such as combat summaries and `/narrator` interactions stay outside turn advancement and continue to use `AppendHistoryEntry` without impersonating a committed narrative turn.
