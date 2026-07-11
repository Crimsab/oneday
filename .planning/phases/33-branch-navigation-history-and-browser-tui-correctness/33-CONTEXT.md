# Phase 33 Context — Branch Navigation, History, and Surface Correctness

The immutable branch DAG exists, but players cannot yet inspect or navigate it
from both play surfaces. Browser history is a bounded snapshot rather than a
paginated reader, dialogue metadata is underused, and several browser labels
still present turn-derived guesses as world facts.

This phase exposes stale-safe branch operations through the Go engine bridge,
adds branch-aware history/chapter/export APIs, and gives browser/TUI one command
registry and equivalent navigation semantics. Browser UX must clear submissions
optimistically without losing replacement text, render canonical time/weather,
separate operator diagnostics from play, and remain keyboard/screen-reader safe.
