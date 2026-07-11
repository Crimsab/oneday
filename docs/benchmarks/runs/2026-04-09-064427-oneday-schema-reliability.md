# OneDay Schema Reliability Benchmark

- Generated at: `2026-04-09T06:43:26+02:00`
- Brief: `Mondo steampunk in tono serio e tenebroso, città industriale decadente, culto del vapore, dialoghi tesi, prosa elegante ma non prolissa. Lingua italiana. Voglio una campagna lunga, politica, investigativa e pericolosa.`
- Cases: `4`

## Ranking

| Model | Success Rate | Avg Latency |
| --- | ---: | ---: |
| `gemini-3.1-flash-lite-preview` | `100.0%` | `4.060s` |
| `grok-4.1-fast` | `100.0%` | `11.396s` |

## gemini-3.1-flash-lite-preview

- Success rate: `100.0%`
- Average latency: `4.060s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `5.280s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `4.103s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `PASS` | `6.861s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Cronache di Fuliggine e Vapore" world="Ossidiana" |

## grok-4.1-fast

- Success rate: `100.0%`
- Average latency: `11.396s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `8.180s` | `x-ai/grok-4.1-fast` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `10.963s` | `x-ai/grok-4.1-fast` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `PASS` | `26.443s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Cronache di Fuliggine e Vapore" world="Ossidiana" |

## Cases

- `missing_authoring_fields`
- `missing_world_rules_and_stats`
- `wrong_shapes`
- `not_json`

