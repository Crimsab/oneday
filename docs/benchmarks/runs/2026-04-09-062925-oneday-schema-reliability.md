# OneDay Schema Reliability Benchmark

- Generated at: `2026-04-09T06:26:24+02:00`
- Brief: `Mondo steampunk in tono serio e tenebroso, città industriale decadente, culto del vapore, dialoghi tesi, prosa elegante ma non prolissa. Lingua italiana. Voglio una campagna lunga, politica, investigativa e pericolosa.`
- Cases: `4`

## Ranking

| Model | Success Rate | Avg Latency |
| --- | ---: | ---: |
| `qwen3.6-plus` | `25.0%` | `0.000s` |

## qwen3.6-plus

- Success rate: `25.0%`
- Average latency: `0.000s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `FAIL` | `60.406s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: Post "http://lite.homelab.local/v1/chat/completions": context deadline exceeded (Client.Timeout exceeded while awaiting headers) |
| `missing_world_rules_and_stats` | `FAIL` | `60.027s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: Post "http://lite.homelab.local/v1/chat/completions": context deadline exceeded (Client.Timeout exceeded while awaiting headers) |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `FAIL` | `60.028s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: Post "http://lite.homelab.local/v1/chat/completions": context deadline exceeded (Client.Timeout exceeded while awaiting headers) |

## Cases

- `missing_authoring_fields`
- `missing_world_rules_and_stats`
- `wrong_shapes`
- `not_json`

