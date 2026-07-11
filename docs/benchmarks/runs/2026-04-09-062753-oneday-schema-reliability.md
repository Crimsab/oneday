# OneDay Schema Reliability Benchmark

- Generated at: `2026-04-09T06:26:24+02:00`
- Brief: `Mondo steampunk in tono serio e tenebroso, città industriale decadente, culto del vapore, dialoghi tesi, prosa elegante ma non prolissa. Lingua italiana. Voglio una campagna lunga, politica, investigativa e pericolosa.`
- Cases: `4`

## Ranking

| Model | Success Rate | Avg Latency |
| --- | ---: | ---: |
| `grok-4.1-fast` | `100.0%` | `10.746s` |
| `gemini-3.1-flash-lite-preview` | `100.0%` | `11.458s` |

## grok-4.1-fast

- Success rate: `100.0%`
- Average latency: `10.746s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `7.351s` | `x-ai/grok-4.1-fast` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `16.929s` | `x-ai/grok-4.1-fast` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `PASS` | `18.708s` | `x-ai/grok-4.1-fast` | story="Ombre di Nebbiaferro" world="Nebbiaferro" |

## gemini-3.1-flash-lite-preview

- Success rate: `100.0%`
- Average latency: `11.458s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `15.371s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `14.207s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `PASS` | `16.253s` | `google/gemini-3.1-flash-lite-preview-20260303` | story="Il Crepuscolo del Vapore" world="Aethelgard" |

## Cases

- `missing_authoring_fields`
- `missing_world_rules_and_stats`
- `wrong_shapes`
- `not_json`

