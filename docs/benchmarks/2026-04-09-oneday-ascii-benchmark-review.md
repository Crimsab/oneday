# OneDay ASCII Benchmark Review

This document preserves the summarized results of the April 2026 benchmark.
Raw run artifacts are generated locally and intentionally excluded from Git.

## Setup

- Base URL: `https://openrouter.ai/api/v1`
- Mode tested: `json_schema`
- Candidate models:
  - `google/gemma-4-31b-it:free`
  - `google/gemma-4-26b-a4b-it:free`
  - `x-ai/grok-4.1-fast`
  - `google/gemini-2.5-flash-lite`
  - `z-ai/glm-4.5-air:free`
  - `minimax/minimax-m2.5:free`
  - `openai/gpt-oss-120b:free`
  - `nvidia/nemotron-3-super-120b-a12b:free`
- Cases:
  - major location reveal
  - neon signage
  - terminal screen
  - ritual circle / diagram
  - map fragment
  - iconic artifact reveal

## What This Benchmark Optimizes For

Ambient ASCII in OneDay is not pure art generation. It needs:

- terminal-safe output width/height
- repeatable structured JSON
- quick completion so it does not stall the same turn
- low or zero cost when possible
- visually readable scene flavor rather than maximal complexity

## Automated Leaderboards

### Overall

1. `openai/gpt-oss-120b:free`
2. `x-ai/grok-4.1-fast`
3. `google/gemma-4-26b-a4b-it:free`
4. `google/gemini-2.5-flash-lite`
5. `google/gemma-4-31b-it:free`
6. `nvidia/nemotron-3-super-120b-a12b:free`
7. `z-ai/glm-4.5-air:free`
8. `minimax/minimax-m2.5:free`

### Quality / Cost

1. `openai/gpt-oss-120b:free`
2. `google/gemma-4-26b-a4b-it:free`
3. `google/gemma-4-31b-it:free`
4. `nvidia/nemotron-3-super-120b-a12b:free`
5. `google/gemini-2.5-flash-lite`
6. `x-ai/grok-4.1-fast`

### Quality / Latency

1. `x-ai/grok-4.1-fast`
2. `openai/gpt-oss-120b:free`
3. `google/gemma-4-26b-a4b-it:free`
4. `google/gemma-4-31b-it:free`
5. `google/gemini-2.5-flash-lite`

### Practical Runtime Fit

1. `x-ai/grok-4.1-fast`
2. `openai/gpt-oss-120b:free`
3. `google/gemma-4-26b-a4b-it:free`
4. `google/gemma-4-31b-it:free`
5. `google/gemini-2.5-flash-lite`

## Subjective ASCII Taste Review

This part is human judgment based on the benchmark outputs, not an automated score.

| Model | Taste score | Notes |
| --- | ---: | --- |
| `openai/gpt-oss-120b:free` | 93 | Most consistent terminal-safe compositions. Excellent on signage, terminals, maps, and artifact silhouettes. |
| `google/gemma-4-26b-a4b-it:free` | 89 | Clean and readable. Slightly less expressive than GPT-OSS, but very usable and cheap. |
| `x-ai/grok-4.1-fast` | 88 | Fast and expressive with strong scene energy, but more likely to drift wider or looser. |
| `google/gemini-2.5-flash-lite` | 84 | Reliable and tidy, though visually plainer than the best free ASCII specialists. |
| `google/gemma-4-31b-it:free` | 83 | Good results overall, but weaker than the 26b A4B variant on this specific task mix. |

## Recommendations

### Best Default ASCII Model

Use `openai/gpt-oss-120b:free`.

- Best overall benchmark score
- Free
- Strongest balance of correctness, speed, and terminal-friendly composition

### Best Free Fallback

Use `google/gemma-4-26b-a4b-it:free`.

- Strong quality
- Free
- Better practical fallback than the slower or weaker alternatives

### Best Paid Speed/Quality Option

Use `x-ai/grok-4.1-fast`.

- Fastest good-looking model in the set
- Strong scene flavor
- Cost is much higher than the free winners, so it is better as an opt-in premium choice than a default ASCII worker

### Operational Recommendation For OneDay

For the live app:

- keep narrator primary: `grok-4.1-fast`
- keep narrator fallback: `google/gemini-2.5-flash-lite`
- keep embeddings: `text-embedding-3-small`
- use `ascii-ambient` with:
  1. `openai/gpt-oss-120b:free`
  2. `google/gemma-4-26b-a4b-it:free`
  3. `google/gemma-4-31b-it:free`
  4. fallback to `main-fast` only when the free ASCII pool is unhealthy

This gives OneDay a dedicated ambient-art path without slowing or destabilizing the main narrator.
