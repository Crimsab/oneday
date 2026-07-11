# Phase 37 context

TTS is a projection of committed canon. Provisional streaming text, repaired
drafts, rejected turns, and sibling-branch messages never enter synthesis.

The phase preserves the current OneDay product UI and adds compact playback and
voice controls. Generation policy and autoplay are separate. Exact voice
collisions are blocked for major cast unless an explicit override is stored.
Aliases inherit entity assignments; form-specific voices require an authored
override.

Initial provider boundary: one OpenAI-compatible cloud adapter and one
persistent Piper-compatible local adapter. Both must fail closed and report
availability without making story turns fail.
