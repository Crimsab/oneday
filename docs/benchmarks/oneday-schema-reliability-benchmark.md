# OneDay Schema Reliability Benchmark

This benchmark measures how well candidate models recover or regenerate valid
`StoryDefinition` JSON for OneDay's story bootstrap flow.

## What It Tests

- Missing required top-level authoring fields
- Missing world/stats schema sections
- Wrong JSON shapes that should be normalized or repaired
- Completely invalid non-JSON output

## Runtime Path

The benchmark exercises the same stack used by the game:

- strict `json_schema` structured output
- OpenRouter/LiteLLM `response-healing` for non-streaming JSON calls
- deterministic validator inside `StoryCreator`
- repair pass using `ai.generation.repair_model`

## Command

```bash
cd /path/to/oneday
go run ./cmd/oneday-schema-benchmark
```

Optional flags:

```bash
go run ./cmd/oneday-schema-benchmark \
  --models "grok-4.1-fast,main-fast,gemini-2.5-flash-lite,qwen3.6-plus" \
  --brief "Mondo steampunk serio e tenebroso..." \
  --timeout 120s
```

## Output

Reports are written to:

- `docs/benchmarks/runs/*-oneday-schema-reliability.json`
- `docs/benchmarks/runs/*-oneday-schema-reliability.md`

## How To Read Results

- `Success rate`: how often the model ended with a valid, accepted story definition
- `Average latency`: average successful repair/generation time
- `Resolved model`: the actual model alias reported by the provider path

If a model shows `no healthy deployments`, treat the run as an availability
failure rather than a true quality signal.
