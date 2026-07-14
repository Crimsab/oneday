# Model benchmarks

OneDay includes three opt-in benchmark commands for comparing models against
real engine contracts. They make live API calls and may incur provider costs.
Generated reports are local artifacts and are ignored by Git.

## Benchmark commands

| Command | Purpose | Default output mode |
| --- | --- | --- |
| `go run ./cmd/oneday-benchmark` | Narrative quality and contract compatibility | `prompt` |
| `go run ./cmd/oneday-ascii-benchmark` | Terminal-safe ambient ASCII generation | `json_schema` |
| `go run ./cmd/oneday-schema-benchmark` | Story-definition repair reliability | Configured runtime path |

The narrative and ASCII benchmarks accept any OpenAI-compatible endpoint. The
schema benchmark loads OneDay's normal `config.yaml` so it exercises the same
router, repair model aliases, validation, and response-healing path as the app.

## Narrative benchmark

Set credentials in your shell and pass an explicit model list:

```bash
export ONEDAY_BENCH_BASE_URL="https://openrouter.ai/api/v1"
export ONEDAY_BENCH_API_KEY="your-key"

go run ./cmd/oneday-benchmark \
  -models "provider/model-a,provider/model-b" \
  -mode all
```

The cases cover story creation, narrative introductions, dialogue metadata,
challenge scenes, and chapter summaries. Available modes are `prompt`,
`json_object`, `json_schema`, and `all`.

Automated measurements include payload validity, Go contract compatibility,
required fields, latency, token usage, estimated throughput, and estimated
cost. Narrative voice and player agency still require human review.

## ASCII benchmark

```bash
export ONEDAY_ASCII_BENCH_BASE_URL="https://openrouter.ai/api/v1"
export ONEDAY_ASCII_BENCH_API_KEY="your-key"

go run ./cmd/oneday-ascii-benchmark \
  -models "provider/model-a,provider/model-b" \
  -mode json_schema
```

The suite measures structured-output compatibility, latency, cost, and TUI
limits such as maximum width and height across location, signage, terminal,
ritual, map, and artifact prompts.

## Schema reliability benchmark

Prepare a working `config.yaml`, then select one or more configured repair-model
aliases:

```bash
export ONEDAY_SCHEMA_BENCH_MODELS="repair-fast,repair-quality"

go run ./cmd/oneday-schema-benchmark \
  -config config.yaml \
  -brief "A political steampunk mystery in a decaying industrial city"
```

This suite feeds malformed story definitions through the production repair and
validation pipeline. It distinguishes model-quality failures from provider
availability failures such as missing deployments.

## Reports

All commands write JSON and Markdown reports to `docs/benchmarks/runs/` by
default. Change the destination with `-output-dir`. The directory is ignored so
raw responses, endpoint metadata, and experimental results are not published by
accident.

## Reproducible comparisons

For useful results:

1. Use the same endpoint, model revisions, modes, prompts, and timeout.
2. Run each candidate multiple times and compare medians, not a single run.
3. Record the date, provider, resolved model name, and relevant configuration.
4. Separate automated contract scores from subjective narrative or visual review.
5. Treat provider outages and rate limits as availability failures, not quality scores.
6. Never commit API keys, private endpoint URLs, raw reports, or user story data.

Published benchmark claims should include enough configuration and methodology
for another contributor to reproduce them. Model rankings age quickly and are
therefore not maintained as permanent project documentation.
