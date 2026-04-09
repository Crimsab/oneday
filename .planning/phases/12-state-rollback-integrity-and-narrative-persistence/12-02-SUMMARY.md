# Summary 12.2: Canonical Narrative Persistence

Added a chat-message schema migration that accepts `narrator` and `combat_summary` message types, then switched narrator meta turns and combat outcome summaries onto the canonical main-history append path. These auxiliary entries now persist in both JSONL and SQLite without consuming an extra story turn, which keeps resume, `/history`, and downstream long-memory features aligned.
