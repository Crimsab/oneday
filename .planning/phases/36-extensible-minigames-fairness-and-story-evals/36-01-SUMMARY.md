# Summary 36.1 — Legacy repair and serializable host

Legacy active minigames now retain graded authoritative outcomes. RPS draws are
explicit success-with-cost results rather than hidden failures. Memory scores
full, near, partial, and weak recall separately. Quick-time resolution accepts
an elapsed duration, with an injected-clock TUI adapter. Riddles compare
normalized exact answers and author-provided semantic aliases; partial
substrings no longer pass.

`MiniGameHost` provides a versioned JSON state machine with registered reducers,
start, pause, resume, serialize, restore, resolve, and replay operations. Seed
plus input history reproduces the same result across host instances. TUI
component messages now carry the complete outcome envelope instead of
rebuilding a lossy `Passed bool` result.
