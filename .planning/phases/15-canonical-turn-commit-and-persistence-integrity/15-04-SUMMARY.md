# Summary 15.4: Failure-Injection Regression Harness

Added focused regression coverage for the persistence scenarios that previously risked continuity corruption:

- stale JSONL mirrors no longer drive turn recovery
- meta-only history entries do not advance the canonical turn cursor
- canonical DB failure does not advance `session.turn`
- JSONL mirror failure still preserves canonical DB history and advances the committed turn safely
