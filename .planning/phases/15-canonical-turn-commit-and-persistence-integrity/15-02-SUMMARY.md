# Summary 15.2: Canonical DB Authority and JSONL Mirror Safety

SQLite-backed world state and canonical chat history now own turn recovery. `NewGameSession` restores its turn cursor from canonical DB state through `GetStoryTurnCursor` instead of counting JSONL lines, and meta-only history such as `/narrator` or combat summaries no longer pollutes that cursor.

JSONL is now written only after the canonical DB commit succeeds. Mirror failures are treated as degraded log-sync problems rather than as a reason to renumber or lose canonical turns.
