# OneDay Benchmark Review

This document preserves the summarized results of the April 2026 benchmark.
Raw run artifacts are generated locally and intentionally excluded from Git.

## Setup

- Base URL: `https://openrouter.ai/api/v1`
- Models tested:
  - `x-ai/grok-4.1-fast`
  - `qwen/qwen3.5-flash-02-23`
  - `google/gemini-2.5-flash-lite`
  - `google/gemini-3.1-flash-lite-preview`
- Cases:
  - `story-creation-final`
  - `narrative-intro`
  - `dialogue-metadata`
  - `challenge-scene`
  - `chapter-summary`

## Runtime Relevance

`oneday` now supports three practical output modes:

- `prompt`: prompt-only, parser accepts fenced JSON and raw JSON
- `json_object`: light structured output
- `json_schema`: strict schema on OpenAI-compatible providers, with runtime fallback only when the endpoint rejects `response_format`

The current runtime integration requests `json_schema` on compatible LiteLLM/OpenRouter providers.

## Contract Leaderboards

### Prompt

| Model | Compat | Avg s | Avg cost | Avg tok/s |
| --- | ---: | ---: | ---: | ---: |
| `google/gemini-3.1-flash-lite-preview` | 94.5 | 8.497 | $0.001711 | 74.9 |
| `x-ai/grok-4.1-fast` | 94.0 | 0.723 | $0.001399 | 2235.1 |
| `qwen/qwen3.5-flash-02-23` | 94.0 | 0.680 | $0.000803 | 4141.7 |
| `google/gemini-2.5-flash-lite` | 92.2 | 0.617 | $0.000592 | 1137.0 |

### JSON Object

| Model | Compat | Avg s | Avg cost | Avg tok/s |
| --- | ---: | ---: | ---: | ---: |
| `google/gemini-3.1-flash-lite-preview` | 92.2 | 5.371 | $0.001665 | 302.7 |
| `qwen/qwen3.5-flash-02-23` | 91.0 | 0.582 | $0.000849 | 4596.8 |
| `x-ai/grok-4.1-fast` | 90.2 | 0.550 | $0.001421 | 2939.0 |
| `google/gemini-2.5-flash-lite` | 80.2 | 0.760 | $0.000519 | 778.2 |

### JSON Schema

| Model | Compat | Avg s | Avg cost | Avg tok/s |
| --- | ---: | ---: | ---: | ---: |
| `qwen/qwen3.5-flash-02-23` | 91.5 | 0.627 | $0.000751 | 3834.8 |
| `google/gemini-3.1-flash-lite-preview` | 89.2 | 8.916 | $0.001652 | 73.2 |
| `x-ai/grok-4.1-fast` | 75.0 | 0.905 | $0.001462 | 1694.4 |
| `google/gemini-2.5-flash-lite` | 70.8 | 0.964 | $0.000521 | 539.5 |

## Prompt To Structured Delta

Compared to `prompt`, `json_schema` changed compatibility like this:

- `x-ai/grok-4.1-fast`: `-19.0` compat, `+0.181s`
- `qwen/qwen3.5-flash-02-23`: `-2.5` compat, `-0.053s`
- `google/gemini-2.5-flash-lite`: `-21.5` compat, `+0.347s`
- `google/gemini-3.1-flash-lite-preview`: `-5.2` compat, `+0.419s`

Takeaway:

- `json_schema` is not a universal upgrade.
- It is acceptable for `qwen`.
- It is materially worse for `grok` and `gemini-2.5-flash-lite` on this current prompt/schema set.

## Narrative Quality Review

This part is subjective and based on reading the `prompt` run outputs for the core story cases.

| Model | Narrative score | Notes |
| --- | ---: | --- |
| `x-ai/grok-4.1-fast` | 92 | Best roleplay voice. Strong atmosphere, fast, consistent scene momentum. |
| `google/gemini-3.1-flash-lite-preview` | 90 | Most literary and polished prose, but much slower. |
| `google/gemini-2.5-flash-lite` | 85 | Clean, readable, efficient. Slightly less vivid than Grok/3.1. |
| `qwen/qwen3.5-flash-02-23` | 79 | Contract-strong and very fast, but flatter voice and occasional awkward choice phrasing. |

## Rankings

### Narrative Quality

1. `x-ai/grok-4.1-fast`
2. `google/gemini-3.1-flash-lite-preview`
3. `google/gemini-2.5-flash-lite`
4. `qwen/qwen3.5-flash-02-23`

### Prompt Quality / Cost

1. `google/gemini-2.5-flash-lite`
2. `qwen/qwen3.5-flash-02-23`
3. `x-ai/grok-4.1-fast`
4. `google/gemini-3.1-flash-lite-preview`

### Prompt Quality / Speed

1. `google/gemini-2.5-flash-lite`
2. `x-ai/grok-4.1-fast`
3. `qwen/qwen3.5-flash-02-23`
4. `google/gemini-3.1-flash-lite-preview`

### Prompt Compatibility / Cost

1. `google/gemini-2.5-flash-lite`
2. `qwen/qwen3.5-flash-02-23`
3. `x-ai/grok-4.1-fast`
4. `google/gemini-3.1-flash-lite-preview`

### Prompt Compatibility / Speed

1. `google/gemini-2.5-flash-lite`
2. `qwen/qwen3.5-flash-02-23`
3. `x-ai/grok-4.1-fast`
4. `google/gemini-3.1-flash-lite-preview`

### JSON Schema Compatibility / Cost

1. `google/gemini-2.5-flash-lite`
2. `qwen/qwen3.5-flash-02-23`
3. `google/gemini-3.1-flash-lite-preview`
4. `x-ai/grok-4.1-fast`

### JSON Schema Compatibility / Speed

1. `qwen/qwen3.5-flash-02-23`
2. `x-ai/grok-4.1-fast`
3. `google/gemini-2.5-flash-lite`
4. `google/gemini-3.1-flash-lite-preview`

## Recommendations

### If You Want Best Roleplay Voice

Use `x-ai/grok-4.1-fast`.

- This is still the best narrative model in the set.
- It is also fast enough to feel snappy in play.
- Caveat: it dislikes strict `json_schema` more than the others in this benchmark.

### If You Want Best Value

Use `google/gemini-2.5-flash-lite`.

- Best quality/cost ratio.
- Best quality/speed ratio.
- Best compatibility/cost ratio in both practical and structured comparisons.

### If You Want Strongest Strict-Schema Contract

Use `qwen/qwen3.5-flash-02-23`.

- It handled `json_schema` best in this run.
- It is also the fastest/highest-throughput model tested.
- Caveat: narrative voice is clearly below Grok and slightly below Gemini 2.5.

### Project Recommendation

For the configuration requested in this session:

- primary model: `x-ai/grok-4.1-fast`
- fallback model: `google/gemini-2.5-flash-lite`

This is a valid product choice if narrative feel matters most.

But there is one important technical caveat:

- the current `json_schema` runtime path helps enforce structure
- the benchmark shows that this exact mode hurts `grok` compatibility more than prompt-only

If runtime issues appear in actual play, the next change should be:

1. make structured-output mode configurable per provider or per model
2. keep `json_schema` for models that benefit from it
3. let `grok` use prompt-only or `json_object`
