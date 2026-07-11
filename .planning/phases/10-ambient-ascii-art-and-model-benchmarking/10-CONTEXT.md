# Phase 10 Context: Ambient ASCII Art and Model Benchmarking

## Why This Phase Exists

Phase 9 fixed the core narrative UX issues, but a few follow-ups remained:

- ambient ASCII art needs a real orchestration design, not ad-hoc inline blobs
- the repo-root `./oneday` binary can drift behind the latest code unless rebuilt explicitly
- the player still lacks an in-context way to understand choice-related stat badges
- OneDay needs a dedicated ASCII-model benchmark instead of guessing which model should generate text art
- the local LiteLLM/OpenRouter setup should reflect the models OneDay actually wants to use

## User Direction

- ASCII art should be **ambient**, not only cinematic
- ASCII should be scene-aware and not appear on every turn
- the likely good design is a structured cue from the narrator plus a second specialized call
- benchmark candidate ASCII models before locking the default
- keep `grok-4.1-fast` as main narrator and `gemini-2.5-flash-lite` as fallback
- ensure local testing via `cd /opt/lab/docker/oneday && ./oneday` runs the updated binary

## Scope

In scope:

- add structured ambient ASCII cue metadata to the narrative contract
- generate ASCII art asynchronously but attach it to the same scene/turn
- keep strict fallback behavior when the cue is absent or the ASCII call fails
- add keyboard-first stat inspect/help for choice stat badges
- provide a root-binary build path for local manual testing
- benchmark the requested ASCII candidate models
- align LiteLLM/OpenRouter config and local virtual-key expectations with OneDay needs

Out of scope:

- full image generation or pixel-art rendering
- replacing the main narrator with a dedicated ASCII model
- mouse hover UI for stat help
- changing the core combat/crafting loop
