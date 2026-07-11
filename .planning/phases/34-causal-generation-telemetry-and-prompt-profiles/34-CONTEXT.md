# Phase 34 Context — Causal Generation Telemetry and Prompt Profiles

OneDay currently retains aggregate narrator model, latency, TTFT, and usage on
messages, but loses provider fallbacks, repair/reroll causality, prompt identity,
and stage-specific attempts. Image jobs have separate operational metadata and
future audio needs the same trace vocabulary.

This phase adds a redacted trace model shared by AI stages: one trace contains
causally linked runs, each run owns ordered provider attempts and immutable
events, and messages may point at their authoring run. Prompt profiles retain
only reproducibility metadata and cryptographic hashes—never API secrets,
private reasoning, or raw hidden prompts in diagnostics/export.
