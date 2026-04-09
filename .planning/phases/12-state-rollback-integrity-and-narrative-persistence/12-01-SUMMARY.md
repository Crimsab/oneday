# Summary 12.1: Full Rollback Snapshot Integrity

Expanded save snapshots from a character/world-only payload into a story rollback artifact that also captures canonical story state: story metadata, NPCs, achievements, chapters, sessions, chat history, RAG chunks, combat logs, and session JSONL files. `LoadGame` now restores that richer snapshot from disk and rewinds the canonical database plus session files instead of pretending a partial state restore is a true rollback.

Legacy saves are still loadable, but they are now explicitly marked as partial so the app can surface that the rollback is not fully coherent.
