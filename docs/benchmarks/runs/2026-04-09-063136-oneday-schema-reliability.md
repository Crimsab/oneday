# OneDay Schema Reliability Benchmark

- Generated at: `2026-04-09T06:27:59+02:00`
- Brief: `Mondo steampunk in tono serio e tenebroso, città industriale decadente, culto del vapore, dialoghi tesi, prosa elegante ma non prolissa. Lingua italiana. Voglio una campagna lunga, politica, investigativa e pericolosa.`
- Cases: `4`

## Ranking

| Model | Success Rate | Avg Latency |
| --- | ---: | ---: |
| `qwen3.6-plus` | `100.0%` | `54.309s` |

## qwen3.6-plus

- Success rate: `100.0%`
- Average latency: `54.309s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `63.924s` | `qwen/qwen3.6-plus-04-02` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `67.872s` | `qwen/qwen3.6-plus-04-02` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `PASS` | `85.442s` | `qwen/qwen3.6-plus-04-02` | story="Cronache di Caligine" world="Caligine" |

## Cases

- `missing_authoring_fields`
- `missing_world_rules_and_stats`
- `wrong_shapes`
- `not_json`

