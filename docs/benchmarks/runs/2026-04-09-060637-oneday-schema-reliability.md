# OneDay Schema Reliability Benchmark

- Generated at: `2026-04-09T06:04:59+02:00`
- Brief: `Mondo steampunk in tono serio e tenebroso, città industriale decadente, culto del vapore, dialoghi tesi, prosa elegante ma non prolissa. Lingua italiana. Voglio una campagna lunga, politica, investigativa e pericolosa.`
- Cases: `4`

## Ranking

| Model | Success Rate | Avg Latency |
| --- | ---: | ---: |
| `grok-4.1-fast` | `100.0%` | `14.776s` |
| `gemini-2.5-flash-lite` | `75.0%` | `1.855s` |
| `main-fast` | `75.0%` | `3.993s` |
| `gemini-3.1-flash-lite-preview` | `25.0%` | `0.000s` |
| `qwen3.6-plus` | `25.0%` | `0.000s` |

## grok-4.1-fast

- Success rate: `100.0%`
- Average latency: `14.776s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `9.775s` | `x-ai/grok-4.1-fast` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `21.436s` | `x-ai/grok-4.1-fast` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `PASS` | `27.894s` | `x-ai/grok-4.1-fast` | story="Ombre di Vapore" world="Vaporgrad" |

## gemini-2.5-flash-lite

- Success rate: `75.0%`
- Average latency: `1.855s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `2.671s` | `google/gemini-2.5-flash-lite` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `2.895s` | `google/gemini-2.5-flash-lite` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `FAIL` | `10.290s` | `-` | invalid story definition returned by AI: missing stats_schema.currency.name |

## main-fast

- Success rate: `75.0%`
- Average latency: `3.993s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `PASS` | `3.526s` | `google/gemini-2.5-flash-lite` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `missing_world_rules_and_stats` | `PASS` | `8.455s` | `google/gemini-2.5-flash-lite` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `FAIL` | `11.538s` | `-` | invalid story definition returned by AI: missing stats_schema.currency.name |

## gemini-3.1-flash-lite-preview

- Success rate: `25.0%`
- Average latency: `0.000s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `FAIL` | `0.029s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: status 400: {"error":{"message":"litellm.BadRequestError: You passed in model=gemini-3.1-flash-lite-preview. There are no healthy deployments for this modelNo fallback model group found for original model_group=gemini-3.1-flash-lite-preview. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]. Received Model Group=gemini-3.1-flash-lite-preview\nAvailable Model Group Fallbacks=None\nError doing the fallback: litellm.BadRequestError: You passed in model=gemini-3.1-flash-lite-preview. There are no healthy deployments for this modelNo fallback model group found for original model_group=gemini-3.1-flash-lite-preview. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]","type":null,"param":null,"code":"400"}} |
| `missing_world_rules_and_stats` | `FAIL` | `0.020s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: status 400: {"error":{"message":"litellm.BadRequestError: You passed in model=gemini-3.1-flash-lite-preview. There are no healthy deployments for this modelNo fallback model group found for original model_group=gemini-3.1-flash-lite-preview. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]. Received Model Group=gemini-3.1-flash-lite-preview\nAvailable Model Group Fallbacks=None\nError doing the fallback: litellm.BadRequestError: You passed in model=gemini-3.1-flash-lite-preview. There are no healthy deployments for this modelNo fallback model group found for original model_group=gemini-3.1-flash-lite-preview. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]","type":null,"param":null,"code":"400"}} |
| `wrong_shapes` | `PASS` | `0.001s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `FAIL` | `0.018s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: status 400: {"error":{"message":"litellm.BadRequestError: You passed in model=gemini-3.1-flash-lite-preview. There are no healthy deployments for this modelNo fallback model group found for original model_group=gemini-3.1-flash-lite-preview. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]. Received Model Group=gemini-3.1-flash-lite-preview\nAvailable Model Group Fallbacks=None\nError doing the fallback: litellm.BadRequestError: You passed in model=gemini-3.1-flash-lite-preview. There are no healthy deployments for this modelNo fallback model group found for original model_group=gemini-3.1-flash-lite-preview. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]","type":null,"param":null,"code":"400"}} |

## qwen3.6-plus

- Success rate: `25.0%`
- Average latency: `0.000s`

| Case | Result | Time | Resolved Model | Notes |
| --- | --- | ---: | --- | --- |
| `missing_authoring_fields` | `FAIL` | `0.015s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: status 400: {"error":{"message":"litellm.BadRequestError: You passed in model=qwen3.6-plus. There are no healthy deployments for this modelNo fallback model group found for original model_group=qwen3.6-plus. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]. Received Model Group=qwen3.6-plus\nAvailable Model Group Fallbacks=None\nError doing the fallback: litellm.BadRequestError: You passed in model=qwen3.6-plus. There are no healthy deployments for this modelNo fallback model group found for original model_group=qwen3.6-plus. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]","type":null,"param":null,"code":"400"}} |
| `missing_world_rules_and_stats` | `FAIL` | `0.013s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: status 400: {"error":{"message":"litellm.BadRequestError: You passed in model=qwen3.6-plus. There are no healthy deployments for this modelNo fallback model group found for original model_group=qwen3.6-plus. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]. Received Model Group=qwen3.6-plus\nAvailable Model Group Fallbacks=None\nError doing the fallback: litellm.BadRequestError: You passed in model=qwen3.6-plus. There are no healthy deployments for this modelNo fallback model group found for original model_group=qwen3.6-plus. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]","type":null,"param":null,"code":"400"}} |
| `wrong_shapes` | `PASS` | `0.000s` | `seed-invalid` | story="Le Ciminiere di Nerofumo" world="Nerofumo" |
| `not_json` | `FAIL` | `0.014s` | `-` | all AI providers failed:   seeded-repair-provider: all AI providers failed:   litellm: HTTP request to litellm: status 400: {"error":{"message":"litellm.BadRequestError: You passed in model=qwen3.6-plus. There are no healthy deployments for this modelNo fallback model group found for original model_group=qwen3.6-plus. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]. Received Model Group=qwen3.6-plus\nAvailable Model Group Fallbacks=None\nError doing the fallback: litellm.BadRequestError: You passed in model=qwen3.6-plus. There are no healthy deployments for this modelNo fallback model group found for original model_group=qwen3.6-plus. Fallbacks=[{'grok-4.1-fast': ['trinity-free', 'step-3.5-flash-free', 'qwen3.5-397b']}, {'main-fast': ['vision-main', 'free-main']}, {'free-main': ['gemini-free']}, {'ascii-ambient': ['ascii-ambient-fallback-1', 'ascii-ambient-fallback-2', 'main-fast', 'free-main']}, {'app-classifier': ['free-main']}]","type":null,"param":null,"code":"400"}} |

## Cases

- `missing_authoring_fields`
- `missing_world_rules_and_stats`
- `wrong_shapes`
- `not_json`

