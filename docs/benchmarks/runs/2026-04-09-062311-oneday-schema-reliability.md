# OneDay Schema Reliability Benchmark

- Generated at: `2026-04-09T06:20:06+02:00`
- Brief: `Mondo steampunk in tono serio e tenebroso, città industriale decadente, culto del vapore, dialoghi tesi, prosa elegante ma non prolissa. Lingua italiana. Voglio una campagna lunga, politica, investigativa e pericolosa.`
- Cases: `4`

## Ranking

| Model | Success Rate | Avg Latency |
| --- | ---: | ---: |
| `gemini-3.1-flash-lite-preview` | `100.0%` | `4.982s` |
| `qwen3.6-plus` | `50.0%` | `22.445s` |

## gemini-3.1-flash-lite-preview

- Success rate: `100.0%`
- Average latency: `4.982s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `8.955s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `4.180s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `PASS` | `6.795s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Vapore e Cenere: L'Eclissi di Ferro" world="Ossidiana" |

## qwen3.6-plus

- Success rate: `50.0%`
- Average latency: `22.445s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `44.891s` | `qwen/qwen3.6-plus-04-02` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `FAIL` | `60.024s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: Post "http://lite.homelab.local/v1/chat/completions": context deadline exceeded (Client.Timeout exceeded while awaiting headers) |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `FAIL` | `60.016s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: Post "http://lite.homelab.local/v1/chat/completions": context deadline exceeded (Client.Timeout exceeded while awaiting headers) |

## Cases

- `missing_authoring_fields`
- `missing_world_rules_and_stats`
- `wrong_shapes`
- `not_json`

