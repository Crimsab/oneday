# Summary 36.2 — Genre-neutral minigames across browser and TUI

Deduction, negotiation, pattern, and bidding now run as registered reducers in
the same replayable host. Each produces one of the authoritative graded outcome
degrees; negotiation randomness is seeded, deduction rewards evidence quality,
pattern matching is timing-free, and bidding distinguishes a fair offer,
overpayment, useful failed bids, and budget catastrophe.

Migration V34 adds mutable minigame instances whose story, branch, source
commit, kind, and protocol lineage cannot change. Checkout cannot read or
update an instance from another branch. The Go gateway owns authoritative
answers/state and returns a redacted player view through Rust routes for start,
active-instance read, pause/resume, and input.

The browser Challenge Host disables narrative input while an instance is
active, renders pause/resume and graded results, and offers all four timing-free
families. The TUI uses the same Go reducer registry and deterministic seeds.
Prompt guidance now prefers these genre-neutral families when they fit.

V34 was rehearsed on a fresh live V33 copy, then deployed to `homelab-main`.
Go reported 507 tests, Rust 45, browser 96, and Playwright 8 desktop/mobile
scenarios. Live schema integrity, branch-scoped empty active-instance API,
restart count zero, responsive launch controls, console safety, and horizontal
overflow checks passed.
