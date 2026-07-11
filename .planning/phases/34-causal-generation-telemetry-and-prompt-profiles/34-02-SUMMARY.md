# Summary 34.2 — Causal stage instrumentation

The Go AI router now records one run per operation and an ordered attempt for
every provider fallback, including requested/observed streaming, first token,
duration, resolved model, usage, cost, retry reason, and redacted error class.
Request bodies are fingerprinted but never persisted.

Narrator, scene judge, repair, reroll, chapter summary, combat, crafting,
narrator meta, story creation, story enhancement, and ASCII art receive stage,
trace, parent, story, branch, and source-commit context. The final assistant
message binds transactionally to its actual authoring run. Rust image jobs use
the same schema with causal retry parents and prompt fingerprints. The schema is
ready for the TTS stage added later in this milestone.
