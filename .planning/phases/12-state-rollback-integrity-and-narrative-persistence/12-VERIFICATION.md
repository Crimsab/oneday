# Verification 12

## Automated Checks

- `go test ./...`
- `go build ./cmd/oneday`
- `go vet ./...`

## Verified Outcomes

- Save snapshots now carry enough canonical story state to support a real rollback instead of only restoring character/world numbers.
- Loading a fresh save restores story history, NPCs, achievements, chapters, RAG chunks, and session JSONL files coherently.
- Legacy saves remain loadable and are detectable as partial rollbacks.
- `/narrator` and combat summaries now persist into canonical history with supported message types.
- Auxiliary canonical entries no longer advance the story turn counter.
- `/craft` is recognized by the parser.
- Autosave replacement removes stale JSON snapshot files.
- Embedding selection now works with the first enabled embedding-capable provider in config, including OpenRouter when LiteLLM is disabled.
- Combat and crafting scaffolding now sends system prompts as `system` role messages.
