# Verification 15

## Automated Checks

- `go test ./internal/engine ./internal/storage`
- `go test ./...`
- `go vet ./...`

## Verified Outcomes

- Main narrative turns now commit character state, world state, and canonical chat history together before `session.turn` advances.
- Resume/session recovery now reads the turn cursor from canonical DB state instead of trusting JSONL line counts.
- Meta-only history entries such as `/narrator` and combat summaries no longer distort the next playable turn number.
- JSONL mirror failures no longer renumber or erase canonical turns; they are surfaced as degraded mirror-sync warnings instead.
- Regression coverage now distinguishes canonical-write failure from mirror-write failure and verifies the correct turn-counter behavior for both.

## Residual Risks

- `ApplyStateChanges` still performs some NPC-side persistence outside the new canonical turn transaction. The audited turn-counter/history corruption findings are fixed, but deeper folding of all NPC side effects into the same transaction remains future hardening work.
